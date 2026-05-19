# talos-operator

Helm chart for the Talos Operator — declaratively manage Talos Linux clusters from Kubernetes.

![Version: 0.5.3](https://img.shields.io/badge/Version-0.5.3-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.5.3](https://img.shields.io/badge/AppVersion-0.5.3-informational?style=flat-square)

**Homepage:** <https://github.com/alperencelik/talos-operator>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| alperencelik |  | <https://github.com/alperencelik> |

## Source Code

* <https://github.com/alperencelik/talos-operator>

## TL;DR

```bash
helm repo add talos-operator https://alperencelik.github.io/helm-charts
helm install talos-operator talos-operator/talos-operator
```

## Prerequisites

- Kubernetes 1.28+
- Helm 3.10+
- Cluster-admin permission for the install (the chart creates CRDs and RBAC by default)

## Installing the chart

```bash
helm install my-release talos-operator/talos-operator
```

To install into a dedicated namespace:

```bash
helm install my-release talos-operator/talos-operator \
  --namespace talos-operator --create-namespace
```

To deploy without the bundled UI:

```bash
helm install my-release talos-operator/talos-operator --set ui.enabled=false
```

## Uninstalling the chart

```bash
helm uninstall my-release
```

CRDs are intentionally not removed on uninstall to avoid losing existing
`TalosCluster`/`TalosControlPlane`/`TalosWorker`/`TalosMachine` resources. Delete
them manually if you want a clean tear-down:

```bash
kubectl delete crd \
  talosclusters.talos.alperen.cloud \
  taloscontrolplanes.talos.alperen.cloud \
  talosworkers.talos.alperen.cloud \
  talosmachines.talos.alperen.cloud \
  talosclusteraddons.talos.alperen.cloud \
  talosclusteraddonreleases.talos.alperen.cloud \
  talosetcdbackups.talos.alperen.cloud \
  talosetcdbackupschedules.talos.alperen.cloud
```

## Compatibility

See the [Compatibility matrix](https://github.com/alperencelik/talos-operator#compatibility)
in the repo README for the operator ↔ Talos version mapping.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules for scheduling the operator pod. |
| autoscaling.enabled | bool | `false` | Enable a `HorizontalPodAutoscaler` for the operator deployment. |
| autoscaling.maxReplicas | int | `100` | Maximum replicas. |
| autoscaling.minReplicas | int | `1` | Minimum replicas. |
| autoscaling.targetCPUUtilizationPercentage | int | `80` | Target average CPU utilization across pods. |
| env | list | `[]` | Extra environment variables to add to the operator container. |
| extraObjects | list | `[]` | Arbitrary additional manifests to ship alongside the chart (useful for `ExternalSecret`, `IngressRoute`, etc.). |
| featureFlags.enableMetaKey | bool | `true` | Handle the Talos `meta` key in reconciliation. Requires a custom Talos ISO with the necessary changes — see [the upstream discussion](https://github.com/siderolabs/talos/discussions/11648#discussioncomment-14253477). |
| featureFlags.enablePxeBootStack | bool | `false` | Install a DHCP+TFTP + Matchbox stack so the operator can PXE-boot Talos machines. |
| fullnameOverride | string | `""` | Override the full release name (chart name + release name). |
| image.pullPolicy | string | `"Always"` | Image pull policy. |
| image.repository | string | `"alperencelik/talos-operator"` | Operator container image repository. |
| image.tag | string | `"latest"` | Image tag. Falls back to the chart `appVersion` when empty. |
| imagePullSecrets | list | `[]` | `imagePullSecrets` to attach to the operator pod for pulling from private registries. |
| ingress.annotations | object | `{}` | Annotations to add to the `Ingress`. |
| ingress.className | string | `""` | `IngressClass` name. |
| ingress.enabled | bool | `false` | Create an `Ingress` for the operator metrics service. |
| ingress.hosts | list | `[{"host":"chart-example.local","paths":[{"path":"/","pathType":"ImplementationSpecific"}]}]` | Hosts/paths to expose. |
| ingress.tls | list | `[]` | TLS configuration. |
| installCRDs | bool | `true` | Install the Talos Operator CRDs as part of this chart release. |
| metrics.serviceMonitor.enabled | bool | `false` | Create a `ServiceMonitor` targeting the operator metrics endpoint. Requires the Prometheus Operator CRDs. |
| metrics.serviceMonitor.interval | string | `"30s"` | Scrape interval. |
| metrics.serviceMonitor.labels | object | `{}` | Extra labels to add to the `ServiceMonitor` so it gets picked up by a specific Prometheus selector. |
| metrics.serviceMonitor.scrapeTimeout | string | `"10s"` | Per-scrape timeout. |
| nameOverride | string | `""` | Override the chart name. |
| nodeSelector | object | `{}` | Node selectors for scheduling the operator pod. |
| podAnnotations | object | `{}` | Annotations to add to the operator pod. |
| podLabels | object | `{}` | Labels to add to the operator pod. |
| podSecurityContext | object | `{}` | Pod-level security context. |
| pxeBootStack.dnsmasq.image | object | `{"pullPolicy":"Always","repository":"dockurr/dnsmasq","tag":"latest"}` | `dnsmasq` image (DHCP+TFTP server used for PXE boot). |
| pxeBootStack.dnsmasq.volumeMounts | list | `[{"mountPath":"/etc/dnsmasq.d","name":"dnsmasq-vol"},{"mountPath":"/var/lib/tftp","name":"tftp-vol"}]` | `dnsmasq` container volume mounts. |
| pxeBootStack.ipxeBaseUrl | string | `"https://boot.ipxe.org"` | Base URL used to download iPXE binaries. |
| pxeBootStack.matchbox.args | list | `["-address=0.0.0.0:8000"]` | Matchbox process arguments. |
| pxeBootStack.matchbox.image | object | `{"pullPolicy":"Always","repository":"quay.io/poseidon/matchbox","tag":"latest"}` | Matchbox image (HTTP server delivering Talos images per MAC). |
| pxeBootStack.matchbox.port | int | `8000` | Matchbox listening port. |
| pxeBootStack.matchbox.volumeMounts | list | `[{"mountPath":"/var/lib/matchbox","name":"matchbox-vol"}]` | Matchbox container volume mounts. |
| pxeBootStack.talosBootImagesBaseUrl | string | `"https://github.com/siderolabs/talos/releases/download"` | Base URL used to download Talos boot images. |
| pxeBootStack.volumes | list | `[{"emptyDir":{"sizeLimit":"1Mi"},"name":"dnsmasq-vol"},{"emptyDir":{"sizeLimit":"2Mi"},"name":"tftp-vol"},{"emptyDir":{"sizeLimit":"10Gi"},"name":"matchbox-vol"}]` | Volumes added to the operator pod when `featureFlags.enablePxeBootStack` is true. Backs the `dnsmasq.volumeMounts` and `matchbox.volumeMounts` below. |
| replicaCount | int | `1` | Number of operator replicas. See [the Kubernetes docs](https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/). |
| resources | object | `{}` | Resource requests/limits for the operator container. Leaving this empty (`{}`) defers the choice to the cluster admin. |
| securityContext | object | `{}` | Container-level security context. |
| service.port | int | `8080` | Operator metrics service port. |
| service.type | string | `"ClusterIP"` | Operator metrics service type. |
| serviceAccount.annotations | object | `{}` | Annotations to add to the `ServiceAccount`. |
| serviceAccount.automount | bool | `true` | Auto-mount the `ServiceAccount`'s API credentials. |
| serviceAccount.create | bool | `true` | Create a `ServiceAccount` for the operator. |
| serviceAccount.name | string | `""` | Name of the `ServiceAccount` to use. When empty and `create` is true, a name is generated from the fullname template. |
| tolerations | list | `[]` | Tolerations for scheduling the operator pod. |
| tracing.enabled | bool | `false` | Enable distributed tracing via OpenTelemetry + `operatortrace`. |
| tracing.otlpEndpoint | string | `"otel-collector.tracing.svc.cluster.local:4318"` | OTLP HTTP endpoint the operator exports spans to. |
| ui.containerPort | int | `8082` | Port the UI container listens on. |
| ui.enabled | bool | `true` | Deploy the Talos Operator UI alongside the controller. |
| ui.image.pullPolicy | string | `"Always"` | Image pull policy for the UI container. |
| ui.image.repository | string | `"alperencelik/talos-operator-ui"` | UI container image repository. |
| ui.image.tag | string | `"latest"` | UI image tag. |
| ui.ingress.annotations | object | `{}` | Annotations to add to the UI `Ingress`. |
| ui.ingress.className | string | `""` | `IngressClass` name. |
| ui.ingress.enabled | bool | `false` | Create an `Ingress` for the UI service. |
| ui.ingress.hosts | list | `[{"host":"ui.chart-example.local","paths":[{"path":"/","pathType":"ImplementationSpecific"}]}]` | Hosts/paths to expose the UI at. |
| ui.ingress.tls | list | `[]` | TLS configuration for the UI `Ingress`. |
| ui.resources | object | `{}` | Resource requests/limits for the UI container. |
| ui.service.port | int | `8082` | UI service port. |
| ui.service.type | string | `"ClusterIP"` | UI service type. |
| volumeMounts | list | `[]` | Additional `volumeMounts` for the operator container. |
| volumes | list | `[]` | Additional volumes to attach to the operator pod. The PXE stack's own volumes live under `pxeBootStack.volumes` and are added automatically when `featureFlags.enablePxeBootStack` is true. |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs](https://github.com/norwoodj/helm-docs).
