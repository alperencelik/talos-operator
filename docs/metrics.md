# Metrics

The operator serves Prometheus metrics on `/metrics`. The Helm chart exposes them on port `8080`.

## Enable scraping

Set this Helm value. It needs the Prometheus Operator CRDs.

```yaml
metrics:
  serviceMonitor:
    enabled: true
```

## Three metric groups

### 1. Controller-runtime

Standard metrics for reconcile counts, errors, durations, and work queues. See the [kubebuilder metrics reference](https://book.kubebuilder.io/reference/metrics-reference).

### 2. External watcher

The operator polls Talos nodes for drift with [kube-external-watcher](https://github.com/alperencelik/kube-external-watcher). The `controller` label is always `TalosMachine`.

| Metric | Type | Labels | Meaning |
| --- | --- | --- | --- |
| `kube_external_watcher_poll_total` | counter | controller, namespace, name, result | Poll cycles per machine. |
| `kube_external_watcher_fetch_external_duration_seconds` | histogram | controller, namespace, name | Time to read the node state. |
| `kube_external_watcher_fetch_external_errors_total` | counter | controller, namespace, name | Failed node reads. |
| `kube_external_watcher_drift_detected_total` | counter | controller, namespace, name | Drifts that caused a reconcile. |
| `kube_external_watcher_registered_resources` | gauge | controller | Machines the watcher polls now. |

### 3. Operator operations

One histogram, `talos_operator_operation_duration_seconds`, covers the long calls the operator makes to Talos nodes.

| Label | Values |
| --- | --- |
| `operation` | `etcd_backup`, `talos_upgrade`, `bootstrap` |
| `namespace` | Namespace of the resource. |
| `name` | `TalosControlPlane` name for `etcd_backup` and `bootstrap`. `TalosMachine` name for `talos_upgrade`. |
| `result` | `success` or `error` |

Rules:

- Each attempt is one sample. A retry after an error is a new sample.
- `talos_upgrade` covers the upgrade API call: image pull, install, and reboot request. It stops when the node starts to reboot.
- Series for a deleted resource stay until the operator restarts. They do not change after the delete.
- Prometheus renames the `namespace` label of groups 2 and 3 to `exported_namespace`. The scrape target keeps `namespace`.

## Resource state and counts

Counts and states are in the `.status` of the CRDs. The operator does not copy them into its own metrics. Export them with the [custom resource state](https://github.com/kubernetes/kube-state-metrics/blob/main/docs/metrics/extend/customresourcestate-metrics.md) feature of kube-state-metrics. These metrics keep working when the operator is down.

!!! note
    kube-state-metrics has frozen this feature. It works, but new development goes to [resource-state-metrics](https://github.com/kubernetes-sigs/resource-state-metrics). Switch when that project is stable.

With the kube-prometheus-stack chart, add these values. Paste the config from the block below under `config`.

```yaml
kube-state-metrics:
  rbac:
    extraRules:
      - apiGroups: ["talos.alperen.cloud"]
        resources: ["talosmachines", "taloscontrolplanes", "talosworkers", "talosetcdbackups", "talosetcdbackupschedules"]
        verbs: ["list", "watch"]
  customResourceState:
    enabled: true
    config:
      kind: CustomResourceStateMetrics
      spec:
        resources: [] # see below
```

??? example "Custom resource state config"

    ```yaml
    kind: CustomResourceStateMetrics
    spec:
      resources:
        - groupVersionKind: { group: talos.alperen.cloud, version: v1alpha1, kind: TalosMachine }
          metricNamePrefix: kube_talos
          labelsFromPath:
            name: [metadata, name]
            namespace: [metadata, namespace]
          metrics:
            - name: machine_state
              help: Current state of the TalosMachine. 1 for the active state.
              each:
                type: StateSet
                stateSet:
                  labelName: state
                  path: [status, state]
                  list: [Pending, Booting, Installing, Available, Ready, Upgrading, Drifted, Orphaned, Failed]
            - name: machine_status_condition
              help: Status conditions of the TalosMachine. 1 is True, otherwise 0.
              each:
                type: Gauge
                gauge:
                  path: [status, conditions]
                  labelsFromPath:
                    type: [type]
                  valueFrom: [status]
        - groupVersionKind: { group: talos.alperen.cloud, version: v1alpha1, kind: TalosControlPlane }
          metricNamePrefix: kube_talos
          labelsFromPath:
            name: [metadata, name]
            namespace: [metadata, namespace]
          metrics:
            - name: controlplane_state
              help: Current state of the TalosControlPlane. 1 for the active state.
              each:
                type: StateSet
                stateSet:
                  labelName: state
                  path: [status, state]
                  list: [Pending, Installing, Available, Bootstrapped, Ready, Failed, UpgradingKubernetes, KubernetesUpgradeFailed]
            - name: controlplane_spec_replicas
              help: Desired control plane replicas.
              each:
                type: Gauge
                gauge:
                  path: [spec, replicas]
                  nilIsZero: true
            - name: controlplane_status_condition
              help: Status conditions of the TalosControlPlane. 1 is True, otherwise 0.
              each:
                type: Gauge
                gauge:
                  path: [status, conditions]
                  labelsFromPath:
                    type: [type]
                  valueFrom: [status]
        - groupVersionKind: { group: talos.alperen.cloud, version: v1alpha1, kind: TalosWorker }
          metricNamePrefix: kube_talos
          labelsFromPath:
            name: [metadata, name]
            namespace: [metadata, namespace]
          metrics:
            - name: worker_state
              help: Current state of the TalosWorker. 1 for the active state.
              each:
                type: StateSet
                stateSet:
                  labelName: state
                  path: [status, state]
                  list: [Pending, Installing, Available, Ready, Failed]
            - name: worker_spec_replicas
              help: Desired worker replicas.
              each:
                type: Gauge
                gauge:
                  path: [spec, replicas]
                  nilIsZero: true
        - groupVersionKind: { group: talos.alperen.cloud, version: v1alpha1, kind: TalosEtcdBackup }
          metricNamePrefix: kube_talos
          labelsFromPath:
            name: [metadata, name]
            namespace: [metadata, namespace]
          metrics:
            - name: etcdbackup_status_condition
              help: Status conditions of the TalosEtcdBackup. 1 is True, otherwise 0.
              each:
                type: Gauge
                gauge:
                  path: [status, conditions]
                  labelsFromPath:
                    type: [type]
                  valueFrom: [status]
        - groupVersionKind: { group: talos.alperen.cloud, version: v1alpha1, kind: TalosEtcdBackupSchedule }
          metricNamePrefix: kube_talos
          labelsFromPath:
            name: [metadata, name]
            namespace: [metadata, namespace]
          metrics:
            - name: etcdbackupschedule_last_successful_backup_time
              help: Unix time of the last successful backup.
              each:
                type: Gauge
                gauge:
                  path: [status, lastSuccessfulBackupTime]
                  nilIsZero: true
    ```

## Kubernetes upgrades

The operator runs a Kubernetes upgrade as a Job named after the cluster. The standard kube-state-metrics Job metrics cover it: `kube_job_status_start_time`, `kube_job_status_completion_time`, and `kube_job_failed`. The control plane state is `UpgradingKubernetes` during the run and `KubernetesUpgradeFailed` after a failure.

## Dashboards

| File | Content |
| --- | --- |
| `grafana/controller-runtime-metrics.json` | Reconcile and work queue panels. |
| `grafana/controller-resources-metrics.json` | CPU and memory of the operator. |
| `grafana/custom-metrics/config.yaml` | Panel definitions for the operator and watcher metrics. |
| `grafana/custom-metrics/custom-metrics-dashboard.json` | Rendered from the panel definitions. |

After you edit the panel definitions, run `make custom-dashboard` to render the dashboard again. It needs `kubebuilder` on your `PATH`.

In the dev environment, `make dev-grafana-dashboard` loads the controller-runtime dashboard into Grafana.

## Example alerts

```yaml
groups:
  - name: talos-operator
    rules:
      - alert: TalosEtcdBackupFailed
        expr: increase(talos_operator_operation_duration_seconds_count{operation="etcd_backup", result="error"}[1h]) > 0
      - alert: TalosEtcdBackupStale
        expr: time() - kube_talos_etcdbackupschedule_last_successful_backup_time > 86400
      - alert: TalosControlPlaneFailed
        expr: kube_talos_controlplane_state{state=~"Failed|KubernetesUpgradeFailed"} == 1
        for: 10m
```
