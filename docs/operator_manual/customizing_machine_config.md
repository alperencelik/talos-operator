# Customizing the Talos Machine Config

The operator generates the Talos machine configuration for every node from the bundle config stored in the cluster's state secret. In most cases the defaults are enough to get a cluster running, but real deployments almost always need to tweak something — kubelet flags, network bonds, image registry mirrors, additional Talos config documents, and so on.

This page explains the customization points exposed on the CRDs, how they compose, and how changes get rolled out to running machines.

## The two mechanisms

The operator offers two complementary ways to customize a machine config. The distinction matters because they apply at different layers of the rendered output.

### `configPatches`

`configPatches` is a list of **strategic merge patches** applied directly to the generated `MachineConfig` document. Each entry can override or extend any field of the main machine config — for example `machine.kubelet`, `machine.network`, or `cluster.apiServer.extraArgs`.

```yaml
configPatches:
  - machine:
      kubelet:
        extraArgs:
          max-pods: "150"
```

Use `configPatches` when you want to change values inside the main machine config without having to manage the whole document yourself.

### `additionalConfig`

`additionalConfig` is a list of **standalone Talos configuration documents** appended after the main machine config, each separated by `---`. Use it for Talos config kinds that are independent documents rather than part of `MachineConfig` — for example `VolumeConfig`, `NetworkDefaultActionConfig`, or `Layer2VIPConfig`.

```yaml
additionalConfig:
  - apiVersion: v1alpha1
    kind: NetworkDefaultActionConfig
    ingress: accept
```

The rule of thumb: if the Talos docs describe the thing you want to configure as a separate `kind:` document, use `additionalConfig`. If it's a field inside `MachineConfig`, use `configPatches`.

## Where you can set them

Both fields can be set at two levels on `TalosControlPlane` and `TalosWorker`:

| Level | Path | Scope |
| --- | --- | --- |
| Global | `spec.metalSpec.machineSpec.configPatches` / `additionalConfig` | Applied to every machine managed by this CR |
| Per-machine | `spec.metalSpec.machines[].configPatches` / `additionalConfig` | Applied only to the listed machine |

If you create `TalosMachine` resources directly (instead of through `TalosControlPlane` / `TalosWorker`), the same fields exist under `spec.machineSpec` on the `TalosMachine`.

### Merge order

When both levels are set, the operator composes them deterministically:

- **`configPatches`** — global patches are applied first, then per-machine patches are appended after. Because Talos applies patches in order, later entries win on conflicting fields.
- **`additionalConfig`** — global documents are emitted first, then per-machine documents, each separated by `---`. They are independent documents, so there is no override semantics — every entry ends up in the final config.

## A worked example

The example below configures kubelet args and a default network ingress action globally, then bonds two NICs on one specific machine and pins a Layer 2 VIP to it:

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosControlPlane
metadata:
  name: taloscontrolplane-sample
spec:
  version: v1.12.2
  mode: metal
  kubeVersion: v1.34.0
  endpoint: "https://192.168.0.153:6443"
  metalSpec:
    machineSpec:
      configPatches:
        - machine:
            kubelet:
              extraArgs:
                max-pods: "150"
      additionalConfig:
        - apiVersion: v1alpha1
          kind: NetworkDefaultActionConfig
          ingress: accept
    machines:
      - address: "192.168.0.153"
        configPatches:
          - machine:
              network:
                interfaces:
                  - interface: bond0
                    bond:
                      mode: 802.3ad
                      interfaces: [net0, net1]
                    vlans:
                      - vlanId: 100
                        addresses: ["10.0.0.1/24"]
        additionalConfig:
          - apiVersion: v1alpha1
            kind: Layer2VIPConfig
            name: 10.0.0.99
            link: enp0s2
```

A runnable version of this example lives at [`examples/talos-controlplane-metal-config-patches.yaml`](https://github.com/alperencelik/talos-operator/blob/main/examples/talos-controlplane-metal-config-patches.yaml).

## How changes get applied

In `metal` mode each `TalosMachine` reconciler regenerates the full machine config every time it reconciles, compares it against `TalosMachine.Status.Config`, and — if they differ — sends an apply through the Talos API. The actual reboot/no-reboot behavior is then decided by Talos itself, based on which fields changed: some fields can be applied live, others require a reboot, and a few require a staged apply. See the upstream [Talos configuration documentation](https://www.talos.dev/latest/talos-guides/configuration/) for the field-level rules.

!!!tip
    If you want to inspect the config that will actually be applied before it lands on a machine, the operator writes generated configs to the cluster's state secret and `TalosMachine.Status.Config`. Reading those is the fastest way to verify a patch produced the YAML you expected.

## CNI configuration

CNI selection is *not* a `configPatches` field — `TalosControlPlane` exposes a dedicated `spec.cni` block with three modes: `flannel` (the Talos default), `custom`, and `none`. Reach for this before reaching for a patch:

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosControlPlane
metadata:
  name: taloscontrolplane-sample
spec:
  cni:
    name: flannel
    flannel:
      kubeNetworkPoliciesEnabled: true
      extraArgs:
        - "--iface-regex=eth.*"
```

For a self-managed CNI (Cilium, Calico, etc.) set `name: custom` and provide the manifest URLs the operator should apply:

```yaml
spec:
  cni:
    name: custom
    urls:
      - https://raw.githubusercontent.com/.../cilium.yaml
```

Use `name: none` if you intend to install the CNI out of band (e.g. via a `TalosClusterAddon`). The `urls` field must be empty for both `flannel` and `none`.

The `cni` field flows directly into the generated control-plane config, so it takes effect the same way as any other config change (see *How changes get applied* above). It's set once at cluster bootstrap; switching CNIs on a running cluster is a Talos-level operation that this field alone won't orchestrate.

## Common patterns

- **CNI**: use the dedicated `spec.cni` field on `TalosControlPlane` (see *CNI configuration* above) rather than patching `cluster.network.cni`.
- **Kubelet args**: patch `machine.kubelet.extraArgs` via `configPatches`.
- **Network bonds and VLANs**: patch `machine.network.interfaces` via `configPatches` at the per-machine level.
- **Container registry mirrors**: prefer the dedicated `spec.metalSpec.machineSpec.registries` field on the CRD when possible; fall back to a `configPatches` entry on `machine.registries` for cases the field doesn't cover.
- **Standalone documents** (e.g. `VolumeConfig`, `NetworkDefaultActionConfig`, `Layer2VIPConfig`): use `additionalConfig`.
