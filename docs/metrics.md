# Talos Operator Metrics

The Talos Operator exposes custom Prometheus metrics to provide observability into its operations and the state of Talos resources.

## Metrics Overview

### Reconciliation Metrics

These metrics track controller reconciliation operations:

- **`talos_operator_reconciliation_total`** (Counter)
  - Labels: `controller`, `result`
  - Total number of reconciliation operations per controller
  - Result values: `success`, `error`, `requeue`, `not_found`

- **`talos_operator_reconciliation_duration_seconds`** (Histogram)
  - Labels: `controller`
  - Duration of reconciliation operations in seconds
  - Controllers: `taloscluster`, `taloscontrolplane`, `talosworker`, `talosmachine`, `talosetcdbackup`, `talosetcdbackupschedule`

### Resource Status Metrics

These metrics track the state of Talos resources:

- **`talos_operator_resource_status`** (Gauge)
  - Labels: `resource_type`, `namespace`, `name`, `status`
  - Current status of resources (1 = ready, 0 = not ready)
  - Resource types: `taloscluster`, `taloscontrolplane`, `talosworker`, `talosmachine`, `talosetcdbackupschedule`

- **`talos_operator_resource_total`** (Gauge)
  - Labels: `resource_type`, `namespace`
  - Total number of resources by type

### Cluster Health Metrics

- **`talos_operator_cluster_health`** (Gauge)
  - Labels: `namespace`, `name`
  - Health status of Talos clusters (1 = healthy, 0 = unhealthy)

- **`talos_operator_machine_ready`** (Gauge)
  - Labels: `namespace`, `cluster`
  - Number of ready Talos machines

### Talos API Metrics

These metrics track interactions with the Talos API:

- **`talos_operator_talos_api_calls_total`** (Counter)
  - Labels: `operation`, `result`
  - Total number of Talos API calls
  - Result values: `success`, `error`

- **`talos_operator_talos_api_call_duration_seconds`** (Histogram)
  - Labels: `operation`
  - Duration of Talos API calls in seconds

### Etcd Backup Metrics

These metrics track etcd backup operations:

- **`talos_operator_etcd_backup_total`** (Counter)
  - Labels: `namespace`, `name`, `status`
  - Total number of etcd backup operations
  - Status values: `success`, `failed`

- **`talos_operator_etcd_backup_duration_seconds`** (Histogram)
  - Labels: `namespace`, `name`
  - Duration of etcd backup operations in seconds

- **`talos_operator_etcd_backup_size_bytes`** (Gauge)
  - Labels: `namespace`, `name`
  - Size of etcd backups in bytes

## Grafana Dashboards

The operator includes pre-built Grafana dashboards in the `grafana/` directory:

1. **controller-runtime-metrics.json**: Default controller-runtime metrics dashboard
2. **controller-resources-metrics.json**: Resource-specific metrics dashboard
3. **custom-metrics/custom-metrics-dashboard.json**: Custom Talos operator metrics dashboard

### Importing Dashboards

To import these dashboards into Grafana:

1. Navigate to Grafana UI → Dashboards → Import
2. Upload the JSON file or paste its contents
3. Select your Prometheus datasource
4. Click "Import"

### Configuring Custom Metrics

Custom metrics can be configured in `grafana/custom-metrics/config.yaml`. After modifying this file, regenerate the dashboards:

```bash
make custom-dashboard
```

## Prometheus Configuration

Ensure your Prometheus instance is configured to scrape the operator's metrics endpoint. The operator exposes metrics on port 8443 (HTTPS) by default, or port 8080 (HTTP) if `--metrics-secure=false` is set.

### ServiceMonitor

A ServiceMonitor resource is available in `config/prometheus/monitor.yaml` for automatic discovery when using the Prometheus Operator.

## Querying Metrics

### Example PromQL Queries

**Reconciliation success rate:**
```promql
rate(talos_operator_reconciliation_total{result="success"}[5m]) / 
rate(talos_operator_reconciliation_total[5m])
```

**95th percentile reconciliation duration:**
```promql
histogram_quantile(0.95, 
  sum by(controller, le) (
    rate(talos_operator_reconciliation_duration_seconds_bucket[5m])
  )
)
```

**Total number of ready clusters:**
```promql
sum(talos_operator_cluster_health == 1)
```

**Etcd backup failure rate:**
```promql
rate(talos_operator_etcd_backup_total{status="failed"}[5m])
```

**Talos API call latency:**
```promql
histogram_quantile(0.95,
  sum by(operation, le) (
    rate(talos_operator_talos_api_call_duration_seconds_bucket[5m])
  )
)
```

## Alerting Rules

Here are some recommended Prometheus alerting rules:

```yaml
groups:
  - name: talos-operator
    rules:
      - alert: TalosOperatorHighReconciliationFailures
        expr: |
          rate(talos_operator_reconciliation_total{result="error"}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High reconciliation failure rate for {{ $labels.controller }}"
          description: "Controller {{ $labels.controller }} is experiencing a high failure rate"

      - alert: TalosOperatorEtcdBackupFailed
        expr: |
          increase(talos_operator_etcd_backup_total{status="failed"}[5m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Etcd backup failed for {{ $labels.namespace }}/{{ $labels.name }}"
          description: "Etcd backup operation has failed"

      - alert: TalosOperatorClusterUnhealthy
        expr: |
          talos_operator_cluster_health == 0
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Talos cluster {{ $labels.namespace }}/{{ $labels.name }} is unhealthy"
          description: "Cluster has been unhealthy for more than 10 minutes"
```

## Development

To add new custom metrics:

1. Define the metric in `internal/metrics/metrics.go`
2. Register it in the `init()` function
3. Add helper functions in `internal/metrics/helpers.go` (optional)
4. Use the metric in the appropriate controller
5. Update `grafana/custom-metrics/config.yaml` with the new metric
6. Regenerate dashboards with `make custom-dashboard`

## References

- [Kubebuilder Metrics Documentation](https://book.kubebuilder.io/reference/metrics.html)
- [Prometheus Client Go](https://github.com/prometheus/client_golang)
- [Grafana Plugin Documentation](https://book.kubebuilder.io/plugins/available/grafana-v1-alpha)