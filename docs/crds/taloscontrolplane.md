# TalosControlPlane

| Field | Value |
|-------|-------|
| **API Group** | `talos.alperen.cloud` |
| **API Version** | `v1alpha1` |
| **Kind** | `TalosControlPlane` |
| **Short Names** | `tcp` |
| **Scope** | Namespaced |
| **Subresources** | `status` |

`TalosControlPlane` defines the control plane of a Talos Linux cluster. It manages the lifecycle of control plane nodes including provisioning, configuration, upgrades, and secrets. It can be owned by a `TalosCluster` or managed independently.

## Print Columns

| Name | JSON Path |
|------|-----------|
| State | `.status.state` |
| Version | `.spec.version` |
| KubeVersion | `.spec.kubeVersion` |
| Mode | `.spec.mode` |
| Age | `.metadata.creationTimestamp` |

---

## Modes

| Mode | Description |
|------|-------------|
| `container` | Runs Talos control plane as containers within Kubernetes pods. Requires `replicas`. |
| `metal` | Runs Talos on bare metal or virtual machines. Requires `metalSpec.machines`. |
| `cloud` | Reserved for future cloud provider integration. |

---

## Example

### Container Mode

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosControlPlane
metadata:
  name: my-controlplane
spec:
  version: v1.13.0
  mode: container
  replicas: 3
  kubeVersion: v1.35.0
  clusterDomain: cluster.local
  podCIDR:
    - 10.244.0.0/16
  serviceCIDR:
    - 10.96.0.0/12
  cni:
    name: flannel
    flannel:
      kubeNetworkPoliciesEnabled: true
  storageClassName: standard
  deletionPolicy: reset
```

### Metal Mode

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosControlPlane
metadata:
  name: my-controlplane
spec:
  version: v1.13.0
  mode: metal
  kubeVersion: v1.35.0
  endpoint: https://192.168.1.100:6443
  metalSpec:
    machines:
      - address: "192.168.1.101"
        pxeClientSpec:
          macAddress: "00:11:22:33:44:55"
          cpuArchitecture: amd64
      - address: "192.168.1.102"
        pxeClientSpec:
          macAddress: "00:11:22:33:44:66"
          cpuArchitecture: amd64
    machineSpec:
      installDisk: /dev/sda
      wipe: false
  rolloutStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
```

---

## Spec Fields

### `spec` (TalosControlPlaneSpec)

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `version` | string | Yes | `v1.13.0` | Pattern: `^v\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$` | Talos version for control plane components (controller-manager, scheduler, kube-apiserver, etcd). e.g. `v1.13.0` |
| `mode` | string | Yes | - | Enum: `container`, `metal`, `cloud` | Deployment mode. **Immutable** after creation. |
| `replicas` | int32 | No | - | Must be >= 1 when mode is `container` | Number of control-plane machines. Only applies when mode is `container`. |
| `metalSpec` | [MetalSpec](#metalspec) | No | - | Required when mode is `metal` | Metal-specific configuration. |
| `endpoint` | string | No | - | Pattern: `^https?://[a-zA-Z0-9.-]+(:\d+)?$` | Kubernetes API Server endpoint URL. |
| `kubeVersion` | string | Yes | `v1.35.0` | Pattern: `^v\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$` | Kubernetes version for the control plane. |
| `clusterDomain` | string | No | `cluster.local` | Pattern: `^([a-zA-Z0-9]([-a-zA-Z0-9]*[a-zA-Z0-9])?\.)+[a-z]{2,}$` | Cluster DNS domain. **Immutable** after creation. |
| `storageClassName` | string | No | - | Pattern: `^[a-zA-Z0-9][-a-zA-Z0-9_.]*[a-zA-Z0-9]$` | StorageClass name for persistent volumes (used by etcd data, etc.). |
| `podCIDR` | []string | No | - | Max 4 items. Each must match `^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$` | CIDR ranges for pod IPs. |
| `serviceCIDR` | []string | No | - | Max 4 items. Each must match `^(\d{1,3}\.){3}\d{1,3}/\d{1,2}$` | CIDR ranges for service VIPs. |
| `configRef` | [ConfigMapKeySelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#configmapkeyselector-v1-core) | No | - | - | Reference to a ConfigMap key containing the Talos controlplane configuration. |
| `cni` | [CNIConfig](#cniconfig) | No | - | - | CNI plugin configuration. |
| `deletionPolicy` | string | No | `reset` | Enum: `reset`, `preserve` | What to do to machines when this resource is deleted. `reset` wipes the Talos installation; `preserve` leaves machines as-is. |
| `rolloutStrategy` | [RolloutStrategy](#rolloutstrategy) | No | `{type: "RollingUpdate", rollingUpdate: {maxUnavailable: 1}}` | - | Controls how Talos version upgrades roll out. Only applies when mode is `metal`. |

### Cross-Field Validations

| Rule | Message |
|------|---------|
| `clusterDomain` is immutable | ClusterDomain is immutable |
| `mode` is immutable | Mode is immutable |
| `mode == 'metal'` requires `metalSpec.machines` | Machines is required when mode is 'metal' |
| `mode == 'container'` requires `replicas >= 1` | replicas must be at least 1 when mode is 'container' |

---

## Nested Types

### MetalSpec

Configuration for bare-metal / VM deployments.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `machines` | [][Machine](#machine) | Yes (when mode=metal) | - | List of machine specifications. Atomic list type (replaced as a whole). |
| `machineSpec` | *[MachineSpec](./talosmachine.md#machinespec) | No | - | Shared machine spec applied to all machines in this set. Individual machines can override via their own fields. |

### Machine

Defines a single Talos machine. Either `address` or `machineRef` must be set, but not both.

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `address` | *string | No | - | Pattern: `^(\d{1,3}\.){3}\d{1,3}$` | IP address of the machine. Mutually exclusive with `machineRef`. |
| `version` | string | No | - | Pattern: `^v\d+\.\d+\.\d+(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$` | Per-machine Talos version override. |
| `image` | *string | No | - | - | Talos installer image override for this machine. |
| `pxeClientSpec` | *[PxeClientSpec](#pxeclientspec) | No | - | - | PXE boot configuration for this machine. |
| `machineRef` | [ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectreference-v1-core) | No | - | - | Reference to a Kubernetes object whose status contains the machine IP. Mutually exclusive with `address`. |
| `configPatches` | []RawExtension | No | - | - | Machine-specific strategic merge config patches. Applied after `machineSpec.configPatches`. |
| `additionalConfig` | []RawExtension | No | - | - | Machine-specific additional Talos config documents. Appended after `machineSpec.additionalConfig`. |

#### Cross-Field Validation

| Rule | Message |
|------|---------|
| `has(address) != has(machineRef)` | address and machineRef are mutually exclusive |

### PxeClientSpec

PXE boot configuration for a machine.

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `macAddress` | *string | Yes | - | - | MAC address of the NIC used by the PXE firmware. e.g. `00:11:22:33:44:55` |
| `cpuArchitecture` | *string | Yes | - | Enum: `amd64`, `arm64` | CPU architecture of the machine. |
| `kernelCmdlineArgs` | *string | No | - | - | Additional kernel command line arguments injected during PXE boot. These are **not** preserved after installation. |

### META

Network metadata written to the Talos META partition.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `hostname` | string | No | - | Hostname for the machine. |
| `interface` | string | No | - | Network interface name. e.g. `eth0` |
| `subnet` | int | No | - | Subnet prefix length. e.g. `24` |
| `gateway` | string | No | - | Default gateway IP address. |
| `dnsServers` | []string | No | - | List of DNS server IP addresses. |

### CNIConfig

CNI plugin configuration.

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `name` | string | No | - | Enum: `flannel`, `custom`, `none` | CNI plugin to use. |
| `urls` | []string | No | - | - | URLs of manifest YAMLs to apply. Required when name is `custom`; must be empty for `flannel` and `none`. |
| `flannel` | *[FlannelCNIConfig](#flannelcniconfig) | No | - | - | Flannel-specific options. |

### FlannelCNIConfig

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `extraArgs` | []string | No | - | Extra arguments passed to `flanneld`. |
| `kubeNetworkPoliciesEnabled` | *bool | No | - | Deploy `kube-network-policies` to enable Kubernetes NetworkPolicy support. |

### RolloutStrategy

Controls how Talos version upgrades are rolled out across machines.

| Field | Type | Required | Default | Validation | Description |
|-------|------|----------|---------|------------|-------------|
| `type` | [RolloutStrategyType](#rolloutstrategytype) | No | `RollingUpdate` | Enum: `RollingUpdate` | Strategy type. Currently only `RollingUpdate` is supported. |
| `rollingUpdate` | *[RollingUpdateRolloutStrategy](#rollingupdaterolloutstrategy) | No | - | - | Rolling update parameters. Only used when `type` is `RollingUpdate`. |

### RollingUpdateRolloutStrategy

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `maxUnavailable` | *[IntOrString](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#intorstring-intstr-util) | No | `1` | Maximum number of machines upgrading simultaneously. Can be an absolute number (e.g. `1`) or a percentage of total machines (e.g. `"25%"`). |

### RolloutStrategyType

| Value | Description |
|-------|-------------|
| `RollingUpdate` | Upgrades machines one cohort at a time, gated by `maxUnavailable` and per-machine health checks. |

---

## Status Fields

### `status` (TalosControlPlaneStatus)

| Field | Type | Description |
|-------|------|-------------|
| `state` | string | Current reconciliation state (e.g. `Ready`, `Provisioning`, `Failed`). |
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) | List of conditions. Map-list keyed by `type`. |
| `config` | string | Reference to the Talos configuration resource. |
| `secretBundle` | string | Reference to the secrets bundle. |
| `bundleConfig` | string | Reference to the bundle configuration. |
| `imported` | *bool | Indicates whether the control plane has been imported (only relevant for import reconciliation mode). |
| `observedKubeVersion` | string | The last observed Kubernetes version on the control plane. |
