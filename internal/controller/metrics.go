package controller

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// operation label values for operationDuration.
const (
	operationEtcdBackup   = "etcd_backup"
	operationTalosUpgrade = "talos_upgrade"
	operationBootstrap    = "bootstrap"
)

// operationDuration records the long-running calls the operator makes against Talos nodes.
// "name" is the TalosControlPlane for etcd_backup and bootstrap, and the TalosMachine for talos_upgrade.
var operationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "talos_operator_operation_duration_seconds",
	Help:    "Duration of long-running operations against Talos nodes, by operation and result.",
	Buckets: prometheus.ExponentialBuckets(1, 2, 12), // 1s .. ~34m
}, []string{"operation", "namespace", "name", "result"})

func init() {
	metrics.Registry.MustRegister(operationDuration)
}

// observeOperation records one attempt of an operation; err decides the result label.
func observeOperation(operation, namespace, name string, start time.Time, err error) {
	result := "success"
	if err != nil {
		result = "error"
	}
	operationDuration.WithLabelValues(operation, namespace, name, result).Observe(time.Since(start).Seconds())
}
