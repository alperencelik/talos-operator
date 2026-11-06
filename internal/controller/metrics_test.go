package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestObserveOperation(t *testing.T) {
	observeOperation(operationEtcdBackup, "metrics-test", "cp", time.Now(), nil)
	observeOperation(operationEtcdBackup, "metrics-test", "cp", time.Now(), errors.New("boom"))

	for _, result := range []string{"success", "error"} {
		var m dto.Metric
		h := operationDuration.WithLabelValues(operationEtcdBackup, "metrics-test", "cp", result).(prometheus.Histogram)
		if err := h.Write(&m); err != nil {
			t.Fatal(err)
		}
		if got := m.GetHistogram().GetSampleCount(); got != 1 {
			t.Errorf("result=%s: want 1 sample, got %d", result, got)
		}
	}
}
