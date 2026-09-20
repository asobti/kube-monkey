// Package metrics records what kube-monkey does and exposes it in the
// Prometheus text format. Recording is always on, so the numbers are correct
// from the first schedule even if the endpoint is only enabled later.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const metricNamespace = "kube_monkey"

const (
	resultSuccess = "success"
	resultFailure = "failure"
)

// Naming a victim in the labels is cheap because apps have to opt in to
// kube-monkey, so the number of series stays small.
var (
	scheduledTerminations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Name:      "scheduled_terminations_total",
		Help:      "Number of terminations kube-monkey has put on a schedule.",
	}, []string{"kind", "namespace", "name"})

	terminations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Name:      "terminations_total",
		Help:      "Number of terminations kube-monkey has carried out, by outcome. A failure means the whole termination was skipped or errored, for example because the victim opted out after it was scheduled.",
	}, []string{"kind", "namespace", "name", "result"})

	podsTerminated = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricNamespace,
		Name:      "pods_terminated_total",
		Help:      "Number of pods kube-monkey has deleted. Stays at zero in dry run mode because no pod is really deleted.",
	}, []string{"kind", "namespace", "name"})

	scheduleSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "schedule_size",
		Help:      "Number of terminations on the latest schedule.",
	})

	lastScheduleTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Name:      "last_schedule_timestamp_seconds",
		Help:      "Time the latest schedule was generated, in seconds since the Unix epoch.",
	})
)

// A private registry keeps the endpoint to kube-monkey's own metrics plus the
// standard Go and process ones, whatever a dependency registers by default.
var registry = prometheus.NewRegistry()

func init() {
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		scheduledTerminations,
		terminations,
		podsTerminated,
		scheduleSize,
		lastScheduleTime,
	)
}

// RecordSchedule records that a schedule holding size terminations was generated.
func RecordSchedule(size int) {
	scheduleSize.Set(float64(size))
	lastScheduleTime.Set(float64(time.Now().Unix()))
}

// RecordScheduledTermination records a termination added to a schedule.
func RecordScheduledTermination(kind, namespace, name string) {
	scheduledTerminations.WithLabelValues(kind, namespace, name).Inc()
}

// RecordTermination records the outcome of a termination that was due.
func RecordTermination(kind, namespace, name string, err error) {
	result := resultSuccess
	if err != nil {
		result = resultFailure
	}
	terminations.WithLabelValues(kind, namespace, name, result).Inc()
}

// RecordPodTermination records a single deleted pod.
func RecordPodTermination(kind, namespace, name string) {
	podsTerminated.WithLabelValues(kind, namespace, name).Inc()
}
