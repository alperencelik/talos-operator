# TalosClusterAddonRelease

`TalosClusterAddonRelease` represents a specific installation of a Helm chart on a single Talos cluster. It is typically created and managed automatically by the `TalosClusterAddon` controller, but can also be created manually.


!!!warning
    This resource is not intended to be created manually. It is created automatically by the `TalosClusterAddon` controller but not enforced. Please consider managing the `TalosClusterAddon` instead.

## Example

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosClusterAddonRelease
metadata:
  name: example-cluster-example-addon-addonrelease
spec:
  clusterRef:
    name: example-cluster
    namespace: default
  helmSpec:
    chartName: ingress-nginx
    repoURL: https://kubernetes.github.io/ingress-nginx
    releaseName: ingress-nginx
    namespace: ingress-nginx
```

## Spec

### `clusterRef`

Reference to the `TalosControlPlane` where the addon is installed.

| Field | Type | Description |
|---|---|---|
| `name` | string | Name of the cluster. |
| `namespace` | string | Namespace of the cluster. |

### `helmSpec`

Configuration for the Helm chart (same as `TalosClusterAddon`).

## Status

The `status` section reflects the installation state of the Helm chart.

### Conditions

| Type | Status | Reason | Message |
|---|---|---|---|
| `Ready` | `True` | `Installed` | The Helm chart has been successfully installed or upgraded. |
| `Ready` | `False` | `KubeconfigFailed` | Failed to retrieve kubeconfig for the target cluster. |
| `Ready` | `False` | `HelmClientFailed` | Failed to create the Helm client. |
| `Ready` | `False` | `HelmInstallFailed` | Failed to install or upgrade the Helm chart. |
