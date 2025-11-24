# TalosClusterAddon

`TalosClusterAddon` is a Custom Resource Definition (CRD) that allows you to define addons (Helm charts) that should be installed on a set of Talos clusters. It uses a label selector to match target clusters.

## Example

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosClusterAddon
metadata:
  name: example-addon
spec:
  clusterSelector:
    matchLabels:
      env: production
  helmSpec:
    chartName: ingress-nginx
    repoURL: https://kubernetes.github.io/ingress-nginx
    releaseName: ingress-nginx
    namespace: ingress-nginx
    version: 4.0.13
    valuesTemplate: |
      controller:
        replicaCount: 2
```

## Spec

### `clusterSelector`

A standard Kubernetes LabelSelector that matches the `TalosControlPlane` resources where this addon should be installed.

### `helmSpec`

Configuration for the Helm chart.

| Field | Type | Description |
|Data | Type | Description |
|---|---|---|
| `chartName` | string | Name of the Helm chart. |
| `repoURL` | string | URL of the Helm chart repository. |
| `releaseName` | string | (Optional) Name of the release. Generated if not provided. |
| `namespace` | string | (Optional) Namespace to install the release into. Defaults to `default`. |
| `version` | string | (Optional) Version of the chart. Defaults to latest. |
| `valuesTemplate` | string | (Optional) Inline YAML values for the chart. |

## Status

The `status` section reflects the current state of the addon.

### Conditions

| Type | Status | Reason | Message |
|---|---|---|---|
| `Ready` | `True` | `Reconciled` | The addon has been successfully processed and `TalosClusterAddonRelease` resources have been created for all matching clusters. |
| `Ready` | `False` | `ListClustersFailed` | Failed to list matching clusters. |
| `Ready` | `False` | `CreateOrUpdateReleaseFailed` | Failed to create or update a release for one or more clusters. |
