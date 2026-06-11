# TalosCluster

| Field | Value |
|-------|-------|
| **API Group** | `talos.alperen.cloud` |
| **API Version** | `v1alpha1` |
| **Kind** | `TalosCluster` |
| **Short Names** | `tc` |
| **Scope** | Namespaced |
| **Subresources** | `status` |

`TalosCluster` is the top-level CRD that defines a complete Talos Linux cluster. It composes a `TalosControlPlane` and `TalosWorker` into a single resource. You can either define them **inline** (embedded directly) or **by reference** (pointing to separately managed resources).

!!!warning
    Mixing modes (e.g. `container` control plane with `metal` workers) is highly discouraged and may lead to unexpected behavior.

---

## Example

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosCluster
metadata:
  name: taloscluster-sample
spec:
  controlPlane:
    version: v1.13.0
    mode: container
    replicas: 3
    kubeVersion: v1.35.0
  worker:
    version: v1.13.0
    mode: container
    replicas: 2
    kubeVersion: v1.35.0
```

---

## Spec Fields

### `spec` (TalosClusterSpec)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `controlPlane` | [TalosControlPlaneSpec](./taloscontrolplane.md#spec-fields) | No | - | Inline control plane configuration. Mutually exclusive with `controlPlaneRef`. |
| `controlPlaneRef` | [LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#localobjectreference-v1-core) | No | - | Reference to an existing `TalosControlPlane` resource by name. Mutually exclusive with `controlPlane`. |
| `worker` | [TalosWorkerSpec](./talosworker.md#spec-fields) | No | - | Inline worker configuration. Mutually exclusive with `workerRef`. |
| `workerRef` | [LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#localobjectreference-v1-core) | No | - | Reference to an existing `TalosWorker` resource by name. Mutually exclusive with `worker`. |
| `pxeServerSpec` | [PxeServerSpec](#pxeserverspec) | No | - | PXE server configuration for network booting Talos machines. |

### Cross-Field Validations

| Rule | Message |
|------|---------|
| `!(has(controlPlane) && has(controlPlaneRef))` | Specify either controlPlane or controlPlaneRef, but not both |
| `!(has(worker) && has(workerRef))` | Specify either worker or workerRef, but not both |

---

## Nested Types

### PxeServerSpec

Defines the PXE server used for booting Talos over the network.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `address` | string | Yes | - | IP address of the PXE server. Must match pattern `^(\d{1,3}\.){3}\d{1,3}$`. |
| `interface` | string | Yes | - | Network interface on the PXE server connected to the boot network (Linux interface name, e.g. `eth0`). |

---

## Status Fields

### `status` (TalosClusterStatus)

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) | List of conditions representing the current state of the TalosCluster. |

#### Condition Types

| Type | Description |
|------|-------------|
| `Ready` | The cluster and all its components are fully reconciled and healthy. |
| `Progressing` | The cluster is being created, updated, or upgraded. |
