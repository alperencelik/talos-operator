# TalosClusterAddonRelease

| Field | Value |
|-------|-------|
| **API Group** | `talos.alperen.cloud` |
| **API Version** | `v1alpha1` |
| **Kind** | `TalosClusterAddonRelease` |
| **Short Names** | `tcar` |
| **Scope** | Namespaced |
| **Subresources** | `status` |

`TalosClusterAddonRelease` represents a specific Helm chart installation on a single Talos cluster. It is created automatically by the `TalosClusterAddon` controller for each matched cluster, but can also be created manually.

!!!warning
    This resource is not intended to be created manually. It is managed automatically by the `TalosClusterAddon` controller. Use `TalosClusterAddon` for addon management.

## Print Columns

| Name | JSON Path |
|------|-----------|
| Cluster | `.spec.clusterRef.name` |
| Chart | `.spec.helmSpec.chartName` |
| Age | `.metadata.creationTimestamp` |

---

## Example

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosClusterAddonRelease
metadata:
  name: example-cluster-ingress-nginx
spec:
  clusterRef:
    name: example-cluster
    namespace: default
  helmSpec:
    chartName: ingress-nginx
    repoURL: https://kubernetes.github.io/ingress-nginx
    releaseName: ingress-nginx
    namespace: ingress-nginx
    version: 4.12.1
```

---

## Spec Fields

### `spec` (TalosClusterAddonReleaseSpec)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `clusterRef` | [ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectreference-v1-core) | Yes | - | Reference to the `TalosControlPlane` (or `TalosCluster`) where the addon will be installed. |
| `helmSpec` | [HelmSpec](./talosclusteraddon.md#helmspec) | No | - | Helm chart configuration. Uses the same `HelmSpec` type as `TalosClusterAddon`. |

### `clusterRef` Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Name of the target cluster resource. |
| `namespace` | string | Namespace of the target cluster resource. |
| `kind` | string | Kind of the target resource (e.g. `TalosControlPlane`). |
| `apiVersion` | string | API version of the target resource. |

---

## Status Fields

### `status` (TalosClusterAddonReleaseStatus)

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) | List of conditions. Map-list keyed by `type`. |

#### Condition Types

| Type | Status | Reason | Description |
|------|--------|--------|-------------|
| `Ready` | `True` | `Installed` | Helm chart successfully installed or upgraded. |
| `Ready` | `False` | `KubeconfigFailed` | Failed to retrieve kubeconfig for the target cluster. |
| `Ready` | `False` | `HelmClientFailed` | Failed to create the Helm client. |
| `Ready` | `False` | `HelmInstallFailed` | Failed to install or upgrade the Helm chart. |
