/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ReconciliationTotal tracks the total number of reconciliations per controller
	ReconciliationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "talos_operator_reconciliation_total",
			Help: "Total number of reconciliations per controller",
		},
		[]string{"controller", "result"},
	)

	// ReconciliationDuration tracks the duration of reconciliation operations
	ReconciliationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "talos_operator_reconciliation_duration_seconds",
			Help:    "Duration of reconciliation operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller"},
	)

	// ResourceStatus tracks the current status of resources
	ResourceStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "talos_operator_resource_status",
			Help: "Current status of Talos resources (1 = ready, 0 = not ready)",
		},
		[]string{"resource_type", "namespace", "name", "status"},
	)

	// ResourceTotal tracks the total number of resources by type
	ResourceTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "talos_operator_resource_total",
			Help: "Total number of Talos resources by type",
		},
		[]string{"resource_type", "namespace"},
	)

	// TalosAPICallsTotal tracks the total number of Talos API calls
	TalosAPICallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "talos_operator_talos_api_calls_total",
			Help: "Total number of Talos API calls",
		},
		[]string{"operation", "result"},
	)

	// TalosAPICallDuration tracks the duration of Talos API calls
	TalosAPICallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "talos_operator_talos_api_call_duration_seconds",
			Help:    "Duration of Talos API calls in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// EtcdBackupTotal tracks the total number of etcd backup operations
	EtcdBackupTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "talos_operator_etcd_backup_total",
			Help: "Total number of etcd backup operations",
		},
		[]string{"namespace", "name", "status"},
	)

	// EtcdBackupDuration tracks the duration of etcd backup operations
	EtcdBackupDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "talos_operator_etcd_backup_duration_seconds",
			Help:    "Duration of etcd backup operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"namespace", "name"},
	)

	// EtcdBackupSize tracks the size of etcd backups
	EtcdBackupSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "talos_operator_etcd_backup_size_bytes",
			Help: "Size of etcd backups in bytes",
		},
		[]string{"namespace", "name"},
	)

	// MachineReadyGauge tracks the number of ready machines
	MachineReadyGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "talos_operator_machine_ready",
			Help: "Number of ready Talos machines",
		},
		[]string{"namespace", "cluster"},
	)

	// ClusterHealthGauge tracks cluster health status
	ClusterHealthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "talos_operator_cluster_health",
			Help: "Health status of Talos clusters (1 = healthy, 0 = unhealthy)",
		},
		[]string{"namespace", "name"},
	)
)

// init registers all custom metrics with the controller-runtime metrics registry
func init() {
	metrics.Registry.MustRegister(
		ReconciliationTotal,
		ReconciliationDuration,
		ResourceStatus,
		ResourceTotal,
		TalosAPICallsTotal,
		TalosAPICallDuration,
		EtcdBackupTotal,
		EtcdBackupDuration,
		EtcdBackupSize,
		MachineReadyGauge,
		ClusterHealthGauge,
	)
}
