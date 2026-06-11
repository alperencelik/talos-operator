# TalosMachine

| Field | Value |
|-------|-------|
| **API Group** | `talos.alperen.cloud` |
| **API Version** | `v1alpha1` |
| **Kind** | `TalosMachine` |
| **Short Names** | `tm` |
| **Scope** | Namespaced |
| **Subresources** | `status` |

`TalosMachine` represents a single Talos machine. It is created and managed by the operator when `TalosControlPlane` or `TalosWorker` are in `metal` mode. It tracks the per-machine lifecycle including configuration, version, and health.

!!!warning
    Users should not create `TalosMachine` resources directly. They are automatically created and managed by the operator based on `TalosControlPlane` and `TalosWorker` specifications.

## Print Columns

| Name | JSON Path |
|------|-----------|
| State | `.status.state` |
| Version | `.spec.version` |
| Endpoint | `.spec.endpoint` |
| Age | `.metadata.creationTimestamp` |

---

## Example

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosMachine
metadata:
  name: my-controlplane-machine-0
spec:
  endpoint: 192.168.1.101
  version: v1.13.0
  machineSpec:
    installDisk: /dev/sda
    wipe: false
    airGap: false
    imageCache: false
  controlPlaneRef:
    name: my-controlplane
  deletionPolicy: reset
  pxeClientSpec:
    macAddress: "00:11:22:33:44:55"
    cpuArchitecture: amd64
```

---

## Spec Fields

### `spec` (TalosMachineSpec)

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `endpoint` | string | Yes | - | - | Talos API endpoint (IP or hostname) for this machine. |
| `version` | string | Yes | - | - | Desired Talos version to run on this machine. |
| `machineSpec` | *[MachineSpec](#machinespec) | No | - | - | Machine-level configuration (disk, network, config patches, etc.). |
| `controlPlaneRef` | [ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectreference-v1-core) | No | - | - | Reference to the `TalosControlPlane` this machine belongs to. |
| `workerRef` | [ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectreference-v1-core) | No | - | - | Reference to the `TalosWorker` this machine belongs to. |
| `configRef` | [ConfigMapKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#configmapkeyselector-v1-core) | No | - | - | Reference to a ConfigMap key containing the Talos machine configuration. |
| `deletionPolicy` | string | No | `reset` | Enum: `reset`, `preserve` | What to do when this resource is deleted. `reset` wipes Talos; `preserve` leaves the machine as-is. |
| `pxeClientSpec` | [PxeClientSpec](./taloscontrolplane.md#pxeclientspec) | No | - | - | PXE boot configuration for this machine. |

---

## Nested Types

### MachineSpec

Shared machine configuration used by both `TalosControlPlane` (via `MetalSpec.machineSpec`) and `TalosMachine`.

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `installDisk` | *string | No | - | Pattern: `^/dev/(sd[a-z][0-9]*\|vd[a-z][0-9]*\|nvme[0-9]+n[0-9]+(p[0-9]+)?)$` | Disk device for Talos installation. e.g. `/dev/sda`, `/dev/nvme0n1` |
| `wipe` | bool | No | `false` | - | Wipe the installation disk before installing Talos. |
| `image` | *string | No | - | - | Custom Talos installer image. |
| `meta` | [META](./taloscontrolplane.md#meta) | No | - | - | Network metadata written to the Talos META partition. |
| `airGap` | bool | No | `false` | - | Indicates the machine is in an air-gapped environment with no internet access. |
| `imageCache` | bool | No | `false` | - | Enable local image caching on the machine. |
| `allowSchedulingOnControlPlanes` | bool | No | `false` | - | Allow scheduling regular workloads on control plane nodes (removes the NoSchedule taint). |
| `registries` | [RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg) | No | - | - | Custom container registry configuration (Talos registries YAML document). |
| `additionalConfig` | [][RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg) | No | - | - | Additional Talos configuration documents to append. Each entry is a separate YAML document joined with `---`. Applied in order: global first, then machine-specific. |
| `configPatches` | [][RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg) | No | - | - | Strategic merge patches applied to the generated Talos machine config. Unlike `additionalConfig`, each patch is merged into the main config to override or extend fields (e.g. `machine.network`). |

---

## Status Fields

### `status` (TalosMachineStatus)

| Field | Type | Description |
|-------|------|-------------|
| `observedVersion` | string | The version of Talos currently running on this machine. |
| `config` | string | Base64-encoded Talos machine configuration. |
| `imported` | *bool | Whether this machine has been imported (only relevant for import reconciliation mode). |
| `state` | string | Current state (e.g. `Ready`, `Provisioning`, `Failed`). |
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) | List of conditions. Map-list keyed by `type`. |
