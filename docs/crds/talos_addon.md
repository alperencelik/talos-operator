# TalosAddon - Helm Chart Support

The `TalosAddon` resource allows you to install and manage Helm charts on your Talos-managed Kubernetes clusters. This provides a declarative way to deploy applications and add-ons to your clusters.

## Overview

The TalosAddon feature enables you to:

- Install Helm charts from any Helm repository
- Manage chart versions and upgrades declaratively
- Pass custom values to charts
- Reference values from ConfigMaps or Secrets
- Automatically uninstall charts when the addon is deleted

## Prerequisites

- A running TalosCluster managed by the talos-operator
- The cluster must be in a ready state with a valid kubeconfig

## Basic Usage

Here's a simple example of installing the NGINX Ingress Controller:

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosAddon
metadata:
  name: nginx-ingress
  namespace: default
spec:
  clusterRef:
    name: my-talos-cluster
  helmRelease:
    chartName: ingress-nginx
    repoURL: https://kubernetes.github.io/ingress-nginx
    version: "4.8.0"
    targetNamespace: ingress-nginx
    values:
      controller.service.type: "LoadBalancer"
```

## Spec Fields

### clusterRef

References the TalosCluster where the addon should be installed.

- **Required**: Yes
- **Type**: `LocalObjectReference`

```yaml
clusterRef:
  name: my-talos-cluster
```

### helmRelease

Defines the Helm chart to install.

#### chartName

The name of the Helm chart to install.

- **Required**: Yes
- **Type**: string

#### repoURL

The URL of the Helm repository containing the chart.

- **Required**: Yes
- **Type**: string

#### version

The version of the chart to install. If not specified, the latest version is used.

- **Required**: No
- **Type**: string

#### releaseName

The name to use for the Helm release. If not specified, the addon's name is used.

- **Required**: No
- **Type**: string
- **Default**: Same as addon name

#### targetNamespace

The namespace where the chart will be installed.

- **Required**: No
- **Type**: string
- **Default**: `default`

#### values

A map of values to pass to the Helm chart. These override the default values in the chart.

- **Required**: No
- **Type**: `map[string]string`

```yaml
values:
  controller.replicaCount: "3"
  controller.service.type: "NodePort"
```

#### valuesFrom

References to ConfigMaps or Secrets containing additional values.

- **Required**: No
- **Type**: array of `ValueReference`

Each `ValueReference` contains:

- **kind**: Either `ConfigMap` or `Secret`
- **name**: Name of the ConfigMap or Secret
- **key**: (Optional) Specific key to use from the resource

```yaml
valuesFrom:
  - kind: ConfigMap
    name: my-chart-values
  - kind: Secret
    name: my-secret-values
    key: credentials
```

## Status Fields

The TalosAddon status provides information about the installation:

- **state**: Current state of the addon (Installing, Ready, Failed)
- **conditions**: Detailed conditions about the addon
- **lastAppliedRevision**: The revision number of the last successful installation
- **observedGeneration**: The generation of the addon spec that was last reconciled

## Examples

### Example 1: Basic Addon

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosAddon
metadata:
  name: metrics-server
  namespace: default
spec:
  clusterRef:
    name: my-cluster
  helmRelease:
    chartName: metrics-server
    repoURL: https://kubernetes-sigs.github.io/metrics-server
    targetNamespace: kube-system
```

### Example 2: Addon with Custom Values

```yaml
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosAddon
metadata:
  name: prometheus
  namespace: default
spec:
  clusterRef:
    name: my-cluster
  helmRelease:
    chartName: kube-prometheus-stack
    repoURL: https://prometheus-community.github.io/helm-charts
    version: "45.0.0"
    targetNamespace: monitoring
    values:
      prometheus.prometheusSpec.retention: "30d"
      prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName: "local-path"
      prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage: "50Gi"
```

### Example 3: Using ValuesFrom

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cert-manager-config
  namespace: default
data:
  installCRDs: "true"
  prometheus.enabled: "true"
---
apiVersion: talos.alperen.cloud/v1alpha1
kind: TalosAddon
metadata:
  name: cert-manager
  namespace: default
spec:
  clusterRef:
    name: my-cluster
  helmRelease:
    chartName: cert-manager
    repoURL: https://charts.jetstack.io
    version: "v1.13.0"
    targetNamespace: cert-manager
    valuesFrom:
      - kind: ConfigMap
        name: cert-manager-config
```

## Lifecycle Management

### Installation

When you create a TalosAddon resource, the operator will:

1. Verify that the referenced TalosCluster exists
2. Retrieve the kubeconfig for the cluster
3. Install the Helm chart with the specified configuration
4. Update the addon's status to reflect the installation

### Updates

When you modify a TalosAddon resource (e.g., changing the version or values), the operator will:

1. Detect the change through the generation field
2. Upgrade the Helm release with the new configuration
3. Update the status with the new revision

### Deletion

When you delete a TalosAddon resource:

1. The operator will uninstall the Helm release from the cluster
2. The addon resource will be removed from Kubernetes

## Monitoring

You can check the status of your addons using kubectl:

```bash
# List all addons
kubectl get talosaddons

# Get detailed information about an addon
kubectl describe talosaddon nginx-ingress

# Check addon status
kubectl get talosaddon nginx-ingress -o jsonpath='{.status.state}'
```

## Troubleshooting

### Addon stuck in Installing state

Check the addon status for error messages:

```bash
kubectl describe talosaddon <addon-name>
```

Common issues:
- The referenced cluster doesn't exist or isn't ready
- The kubeconfig secret is missing or invalid
- The Helm repository is unreachable
- Invalid chart name or version

### Addon installation failed

Check the conditions in the status:

```bash
kubectl get talosaddon <addon-name> -o yaml
```

Look for the condition with type "Failed" for details about the error.

## Best Practices

1. **Version Pinning**: Always specify a version for your charts to ensure reproducible deployments
2. **Namespace Isolation**: Use separate namespaces for different addons to avoid conflicts
3. **ConfigMaps for Values**: Use ConfigMaps or Secrets for complex value configurations
4. **Monitoring**: Regularly check addon status to ensure they remain in a healthy state
5. **Testing**: Test addon configurations in a development cluster before applying to production

## Limitations

- The addon feature requires the TalosCluster to have a valid kubeconfig secret
- Helm operations are performed synchronously, so large charts may take time to install
- Currently, only Helm charts are supported (not raw manifests, though those can be added via Talos cluster configuration's `.cluster.extraManifests`)
