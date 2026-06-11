# TalosWorker

| Field | Value |
|-------|-------|
| **API Group** | `talos.alperen.cloud` |
| **API Version** | `v1alpha1` |
| **Kind** | `TalosWorker` |
| **Short Names** | `tw` |
| **Scope** | Namespaced |
| **Subresources** | `status` |

`TalosWorker` defines the worker nodes of a Talos Linux cluster. Unlike `TalosControlPlane`, it cannot be used standalone — it must reference a `TalosControlPlane` via `controlPlaneRef` so it can join the correct cluster.

## Print Columns

| Name | JSON Path |
|------|-----------|
| State | `.status.state` |
| Version | `.spec.version` |
| Mode | `.spec.mode` |
| Age | `.metadata.creationTimestamp` |

---

## Modes

| Mode | Description |
|------|-------------|
| `container` | Runs Talos workers as containers within Kubernetes pods. Requires `replicas`. |
| `metal` | Runs Talos on bare metal or virtual machines. Requires `metalSpec`. |
| `cloud` | Reserved for future cloud provider integration. |

---

## Example

### Container Mode

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosWorker
metadata:
  name: my-worker
spec:
  version: v1.13.0
  mode: container
  replicas: 3
  kubeVersion: v1.35.0
  controlPlaneRef:
    name: my-controlplane
  storageClassName: standard
  deletionPolicy: reset
```

### Metal Mode

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosWorker
metadata:
  name: my-worker
spec:
  version: v1.13.0
  mode: metal
  kubeVersion: v1.35.0
  controlPlaneRef:
    name: my-controlplane
  metalSpec:
    machines:
      - address: "192.168.1.201"
        pxeClientSpec:
          macAddress: "00:11:22:33:44:77"
          cpuArchitecture: amd64
      - address: "192.168.1.202"
        pxeClientSpec:
          macAddress: "00:11:22:33:44:88"
          cpuArchitecture: amd64
    machineSpec:
      installDisk: /dev/sda
  rolloutStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
```

---

## Spec Fields

### `spec` (TalosWorkerSpec)

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `version` | string | Yes | `v1.13.0` | Pattern: `^v\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$` | Talos version for worker nodes. e.g. `v1.13.0` |
| `mode` | string | Yes | - | Enum: `container`, `metal`, `cloud` | Deployment mode. **Immutable** after creation. |
| `replicas` | int32 | No | `1` | Must be >= 1 when mode is `container` | Number of worker machines. Only applies when mode is `container`. |
| `metalSpec` | [MetalSpec](./taloscontrolplane.md#metalspec) | Yes (when mode=metal) | - | - | Metal-specific configuration. Uses the same `MetalSpec` type as `TalosControlPlane`. |
| `kubeVersion` | string | Yes | `v1.35.0` | Pattern: `^v\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$` | Kubernetes version for worker nodes. |
| `storageClassName` | *string | No | - | Pattern: `^[a-zA-Z0-9][-a-zA-Z0-9_.]*[a-zA-Z0-9]$` | StorageClass for persistent volumes. **Immutable** after creation. |
| `controlPlaneRef` | [LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#localobjectreference-v1-core) | Yes | - | - | Reference to the `TalosControlPlane` this worker belongs to (by name). |
| `configRef` | [ConfigMapKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#configmapkeyselector-v1-core) | No | - | - | Reference to a ConfigMap key containing the Talos worker configuration. |
| `deletionPolicy` | string | No | `reset` | Enum: `reset`, `preserve` | What to do to machines when this resource is deleted. `reset` wipes Talos; `preserve` leaves machines as-is. |
| `rolloutStrategy` | [RolloutStrategy](./taloscontrolplane.md#rolloutstrategy) | No | `{type: "RollingUpdate", rollingUpdate: {maxUnavailable: 1}}` | - | Controls how Talos version upgrades roll out. Only applies when mode is `metal`. |

### Cross-Field Validations

| Rule | Message |
|------|---------|
| `mode` is immutable | Mode is immutable |
| `mode == 'metal'` requires `metalSpec` | MetalSpec is required when mode 'metal' |
| `mode == 'container'` requires `replicas >= 1` | replicas must be at least 1 when mode is 'container' |

---

## Status Fields

### `status` (TalosWorkerStatus)

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) | List of conditions. Map-list keyed by `type`. |
| `config` | string | Serialized Talos worker configuration. |
| `imported` | *bool | Whether this worker has been imported (only relevant for import reconciliation mode). |
| `state` | string | Current state (e.g. `Ready`, `Provisioning`, `Failed`). |
