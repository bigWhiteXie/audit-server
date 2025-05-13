package pipeline

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	QueueSize      *prometheus.GaugeVec
	SuccessCounter *prometheus.CounterVec
	ErrorCounter   *prometheus.CounterVec
	ExportCounter  *prometheus.CounterVec
	DiskUsage      *prometheus.GaugeVec
	ExportLatency  *prometheus.HistogramVec

	// 新增状态相关指标
	StateTransitions *prometheus.CounterVec
	RecoveryAttempts *prometheus.CounterVec
	RecoveryErrors   *prometheus.CounterVec
	RecoveredItems   *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		QueueSize: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "queue_size",
			Help:      "Current number of items in the pipeline queue",
		}, []string{"queue"}),
		SuccessCounter: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "export_success_total",
			Help:      "Total number of successful exports",
		}, []string{"exporter"}),
		ErrorCounter: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "export_errors_total",
			Help:      "Total number of export errors",
		}, []string{"exporter"}),
		ExportCounter: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "export_attempts_total",
			Help:      "Total number of export attempts",
		}, []string{"exporter"}),
		DiskUsage: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "local_storage_bytes",
			Help:      "Current local storage usage in bytes",
		}, []string{"exporter"}),
		ExportLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "export_latency_seconds",
			Help:      "Export operation latency in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"exporter"}),

		// 新增状态相关指标
		StateTransitions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "state_transitions_total",
				Help:      "Number of pipeline state transitions",
			},
			[]string{"pipeline", "transition"},
		),
		RecoveryAttempts: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "recovery_attempts_total",
			Help:      "Number of recovery attempts from local storage",
		}, []string{"pipeline"}),
		RecoveryErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "recovery_errors_total",
			Help:      "Number of errors during recovery from local storage",
		}, []string{"pipeline"}),
		RecoveredItems: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "recovered_items_total",
			Help:      "Number of items successfully recovered from local storage",
		}, []string{"pipeline"}),
	}

	return m
}
