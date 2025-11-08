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
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// RecordReconciliation records a reconciliation operation
func RecordReconciliation(controller string, result string, duration time.Duration) {
	ReconciliationTotal.WithLabelValues(controller, result).Inc()
	ReconciliationDuration.WithLabelValues(controller).Observe(duration.Seconds())
}

// RecordTalosAPICall records a Talos API call
func RecordTalosAPICall(operation string, result string, duration time.Duration) {
	TalosAPICallsTotal.WithLabelValues(operation, result).Inc()
	TalosAPICallDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// SetResourceStatus sets the status of a resource
func SetResourceStatus(resourceType, namespace, name, status string, value float64) {
	ResourceStatus.WithLabelValues(resourceType, namespace, name, status).Set(value)
}

// SetResourceTotal sets the total count of resources
func SetResourceTotal(resourceType, namespace string, count float64) {
	ResourceTotal.WithLabelValues(resourceType, namespace).Set(count)
}

// RecordEtcdBackup records an etcd backup operation
func RecordEtcdBackup(namespace, name, status string, duration time.Duration, size float64) {
	EtcdBackupTotal.WithLabelValues(namespace, name, status).Inc()
	EtcdBackupDuration.WithLabelValues(namespace, name).Observe(duration.Seconds())
	if size > 0 {
		EtcdBackupSize.WithLabelValues(namespace, name).Set(size)
	}
}

// SetMachineReady sets the number of ready machines
func SetMachineReady(namespace, cluster string, count float64) {
	MachineReadyGauge.WithLabelValues(namespace, cluster).Set(count)
}

// SetClusterHealth sets the health status of a cluster
func SetClusterHealth(namespace, name string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	ClusterHealthGauge.WithLabelValues(namespace, name).Set(value)
}

// Timer is a helper for timing operations
type Timer struct {
	start time.Time
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// ObserveDuration observes the duration since timer creation
func (t *Timer) ObserveDuration() time.Duration {
	return time.Since(t.start)
}

// ObserveReconciliation observes and records a reconciliation duration
func (t *Timer) ObserveReconciliation(controller, result string) {
	RecordReconciliation(controller, result, t.ObserveDuration())
}

// ObserveTalosAPICall observes and records a Talos API call duration
func (t *Timer) ObserveTalosAPICall(operation, result string) {
	RecordTalosAPICall(operation, result, t.ObserveDuration())
}

// DeleteResourceMetrics deletes metrics for a specific resource
func DeleteResourceMetrics(resourceType, namespace, name string) {
	// Delete all status label combinations for this resource
	ResourceStatus.DeletePartialMatch(prometheus.Labels{
		"resource_type": resourceType,
		"namespace":     namespace,
		"name":          name,
	})
}
