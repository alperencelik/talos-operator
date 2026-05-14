# Customizing the Talos Machine Config

The operator generates the Talos machine configuration for every node from the bundle config stored in the cluster's state secret. In most cases the defaults are enough to get a cluster running, but real deployments almost always need to tweak something — kubelet flags, network bonds, image registry mirrors, additional Talos config documents, and so on.

This page explains the customization points exposed on the CRDs, how they compose, and how changes get rolled out to running machines.

## First-class config fields

Reach for these before reaching for a patch. They are typed fields on the CRDs, so they get CRD validation, are easy to introspect, and the operator knows how to set them in the right shape.

### Cluster-level (`TalosControlPlane` / `TalosWorker`)

| Field | Effect on the rendered config |
| --- | --- |
| `spec.version` | Talos version contract and the default installer image tag |
| `spec.kubeVersion` | Kubernetes control-plane version |
| `spec.endpoint` | Kubernetes API endpoint |
| `spec.podCIDR` | `cluster.network.podSubnets` |
| `spec.serviceCIDR` | `cluster.network.serviceSubnets` |
| `spec.cni` | Cluster CNI selection (see below) |

#### CNI (`spec.cni`)

`TalosControlPlane` exposes a dedicated `spec.cni` block with three modes: `flannel` (the Talos default), `custom`, and `none`.

```yaml
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

Use `name: none` if you intend to install the CNI out of band (e.g. via a `TalosClusterAddon`). The `urls` field must be empty for both `flannel` and `none`. It's set once at cluster bootstrap; switching CNIs on a running cluster is a Talos-level operation that this field alone won't orchestrate.

### Machine-level (`MachineSpec`)

The following fields live on `spec.metalSpec.machineSpec` (or directly on `TalosMachine.spec.machineSpec`) and the operator turns each one into the corresponding patch for you:

| Field | Effect on the rendered config |
| --- | --- |
| `installDisk` | `machine.install.disk` (auto-resolved via the Talos API if left unset) |
| `wipe` | `machine.install.wipe` |
| `image` | `machine.install.image` — the installer image; version suffixes are handled for you |
| `airGap` | Sets `machine.time.disabled: true` and `cluster.discovery.enabled: false` |
| `imageCache` | Enables `machine.features.imageCache.localEnabled` and appends a `VolumeConfig` document for the cache disk |
| `allowSchedulingOnControlPlanes` | `cluster.allowSchedulingOnControlPlanes: true` |
| `registries` | Inserted as the `machine.registries` block — accepts the same shape as the Talos `registries` config |

!!!warning
    Per-machine overrides under `spec.metalSpec.machines[]` only honour `image`, `configPatches`, and `additionalConfig`. The other machine-level knobs above are only respected at the global `machineSpec` level. If you need a per-machine override for one of them, use a `configPatches` entry on the machine.

## Patches and additional config

For anything not covered by a dedicated field, the operator offers two complementary mechanisms. They apply at different layers of the rendered output, so the distinction matters.

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

In `container` mode the rendered config is written to a `ConfigMap` consumed by the StatefulSet, so updating a patch will roll the pods.

!!!tip
    If you want to inspect the config that will actually be applied before it lands on a machine, the operator writes generated configs to the cluster's state secret and `TalosMachine.Status.Config`. Reading those is the fastest way to verify a patch produced the YAML you expected.

## Common patterns

- **CNI**: use the dedicated `spec.cni` field on `TalosControlPlane` (see *First-class config fields* above) rather than patching `cluster.network.cni`.
- **Install disk / installer image / wipe**: use `machineSpec.installDisk`, `machineSpec.image`, `machineSpec.wipe`.
- **Air-gapped clusters**: set `machineSpec.airGap: true` instead of patching `machine.time.disabled` and `cluster.discovery.enabled` by hand.
- **Image cache on disk**: set `machineSpec.imageCache: true` — the operator also emits the matching `VolumeConfig` document for you.
- **Scheduling workloads on control-plane nodes**: set `machineSpec.allowSchedulingOnControlPlanes: true`.
- **Container registry mirrors**: prefer the dedicated `machineSpec.registries` field; fall back to a `configPatches` entry on `machine.registries` only for cases the field doesn't cover.
- **Kubelet args**: patch `machine.kubelet.extraArgs` via `configPatches`.
- **Network bonds and VLANs**: patch `machine.network.interfaces` via `configPatches` at the per-machine level.
- **Standalone documents** (e.g. `VolumeConfig`, `NetworkDefaultActionConfig`, `Layer2VIPConfig`): use `additionalConfig`.

!!!info
    This page is an initial pass. If a pattern you need isn't covered here, please open an issue — the goal is for this page to grow with real-world examples.
