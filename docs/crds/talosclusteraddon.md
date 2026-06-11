# TalosClusterAddon

| Field | Value |
|-------|-------|
| **API Group** | `talos.alperen.cloud` |
| **API Version** | `v1alpha1` |
| **Kind** | `TalosClusterAddon` |
| **Short Names** | `tca` |
| **Scope** | Namespaced |
| **Subresources** | `status` |

`TalosClusterAddon` defines a Helm chart addon that should be installed on a set of Talos clusters matched by a label selector. The controller automatically creates a `TalosClusterAddonRelease` for each matching cluster.

## Print Columns

| Name | JSON Path |
|------|-----------|
| Chart | `.spec.helmSpec.chartName` |
| Version | `.spec.helmSpec.version` |
| Age | `.metadata.creationTimestamp` |

---

## Example

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosClusterAddon
metadata:
  name: ingress-nginx-addon
spec:
  clusterSelector:
    matchLabels:
      env: production
  helmSpec:
    chartName: ingress-nginx
    repoURL: https://kubernetes.github.io/ingress-nginx
    releaseName: ingress-nginx
    namespace: ingress-nginx
    version: 4.12.1
    valuesTemplate: |
      controller:
        replicaCount: 2
        service:
          type: LoadBalancer
    options:
      wait: true
      timeout: 300
      skipCRDs: false
    credentials:
      secret:
        name: helm-credentials
    tlsConfig:
      secret:
        name: helm-tls
```

---

## Spec Fields

### `spec` (TalosClusterAddonSpec)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `clusterSelector` | [LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#labelselector-v1-meta) | No | - | Label selector matching `TalosCluster` resources where this addon should be installed. |
| `helmSpec` | [HelmSpec](#helmspec) | No | - | Helm chart configuration. |

### HelmSpec

Configuration for the Helm chart release.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `chartName` | string | Yes | - | Name of the Helm chart in the repository. e.g. for `oci://repo-url/chart-name`, this is `chart-name`. |
| `repoURL` | string | Yes | - | URL of the Helm chart repository. e.g. for `oci://repo-url/chart-name`, this is `oci://repo-url`. Supports both `https://` and `oci://` schemes. |
| `releaseName` | string | No | Auto-generated | Helm release name. If omitted, a name is generated. |
| `namespace` | string | No | `default` | Kubernetes namespace where the Helm release will be installed on each matched cluster. |
| `version` | string | No | Latest | Helm chart version. If omitted, the latest available version is used and kept up to date. |
| `valuesTemplate` | string | No | - | Inline YAML with Helm values. Supports Go templating to reference fields from each matched workload Cluster. |
| `options` | *[HelmOptions](https://pkg.go.dev/sigs.k8s.io/cluster-api-addon-provider-helm/api/v1alpha1#HelmOptions) | No | - | CLI flags for Helm operations (install, upgrade, delete). Includes `wait`, `skipCRDs`, `timeout`, `waitForJobs`, etc. Inherited from Cluster API Addons. |
| `credentials` | *[Credentials](https://pkg.go.dev/sigs.k8s.io/cluster-api-addon-provider-helm/api/v1alpha1#Credentials) | No | - | OCI registry credentials reference. Inherited from Cluster API Addons. |
| `tlsConfig` | *[TLSConfig](https://pkg.go.dev/sigs.k8s.io/cluster-api-addon-provider-helm/api/v1alpha1#TLSConfig) | No | - | TLS configuration for the Helm repository. Inherited from Cluster API Addons. |

---

## Status Fields

### `status` (TalosClusterAddonStatus)

| Field | Type | Description |
|-------|------|-------------|
| `conditions` | [][Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) | List of conditions. Map-list keyed by `type`. |

#### Condition Types

| Type | Status | Reason | Description |
|------|--------|--------|-------------|
| `Ready` | `True` | `Reconciled` | Addon processed and `TalosClusterAddonRelease` resources created for all matching clusters. |
| `Ready` | `False` | `ListClustersFailed` | Failed to list matching clusters. |
| `Ready` | `False` | `CreateOrUpdateReleaseFailed` | Failed to create or update a release for one or more clusters. |
