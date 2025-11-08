# Talos Operator Metrics

The Talos Operator exposes custom Prometheus metrics to provide observability into its operations and the state of Talos resources.

## Metrics Overview

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
  - Labels: `operation`
  - Total number of Talos API calls

### Etcd Backup Metrics

These metrics track etcd backup operations:

- **`talos_operator_etcd_backup_total`** (Counter)
  - Labels: `namespace`, `name`
  - Total number of etcd backup operations

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

**Total number of ready clusters:**
```promql
sum(talos_operator_cluster_health == 1)
```

**Total number of resources by type:**
```promql
sum by(resource_type) (talos_operator_resource_total)
```

**Total Talos API calls:**
```promql
rate(talos_operator_talos_api_calls_total[5m])
```

**Etcd backup operations:**
```promql
rate(talos_operator_etcd_backup_total[5m])
```

**Etcd backup size:**
```promql
talos_operator_etcd_backup_size_bytes
```

## Alerting Rules

Here are some recommended Prometheus alerting rules:

```yaml
groups:
  - name: talos-operator
    rules:
      - alert: TalosOperatorClusterUnhealthy
        expr: |
          talos_operator_cluster_health == 0
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Talos cluster {{ $labels.namespace }}/{{ $labels.name }} is unhealthy"
          description: "Cluster has been unhealthy for more than 10 minutes"
      
      - alert: TalosOperatorNoReadyMachines
        expr: |
          talos_operator_machine_ready == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "No ready machines in cluster {{ $labels.namespace }}/{{ $labels.cluster }}"
          description: "Cluster has no ready machines"
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