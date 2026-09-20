package metrics

import (
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func reset() {
	scheduledTerminations.Reset()
	terminations.Reset()
	podsTerminated.Reset()
	scheduleSize.Set(0)
	lastScheduleTime.Set(0)
}

func TestRecordSchedule(t *testing.T) {
	reset()
	before := time.Now().Unix()

	RecordSchedule(3)

	assert.Equal(t, float64(3), testutil.ToFloat64(scheduleSize))
	assert.GreaterOrEqual(t, testutil.ToFloat64(lastScheduleTime), float64(before))
}

func TestRecordScheduledTermination(t *testing.T) {
	reset()

	RecordScheduledTermination("Deployment", "app-namespace", "monkey-victim")
	RecordScheduledTermination("Deployment", "app-namespace", "monkey-victim")
	RecordScheduledTermination("StatefulSet", "app-namespace", "monkey-victim")

	assert.Equal(t, float64(2), testutil.ToFloat64(scheduledTerminations.WithLabelValues("Deployment", "app-namespace", "monkey-victim")))
	assert.Equal(t, float64(1), testutil.ToFloat64(scheduledTerminations.WithLabelValues("StatefulSet", "app-namespace", "monkey-victim")))
}

func TestRecordTermination(t *testing.T) {
	reset()

	RecordTermination("Deployment", "app-namespace", "monkey-victim", nil)
	RecordTermination("Deployment", "app-namespace", "monkey-victim", errors.New("no longer enrolled"))

	assert.Equal(t, float64(1), testutil.ToFloat64(terminations.WithLabelValues("Deployment", "app-namespace", "monkey-victim", resultSuccess)))
	assert.Equal(t, float64(1), testutil.ToFloat64(terminations.WithLabelValues("Deployment", "app-namespace", "monkey-victim", resultFailure)))
}

func TestRecordPodTermination(t *testing.T) {
	reset()

	RecordPodTermination("Deployment", "app-namespace", "monkey-victim")

	assert.Equal(t, float64(1), testutil.ToFloat64(podsTerminated.WithLabelValues("Deployment", "app-namespace", "monkey-victim")))
}

func TestGatherExposesRecordedMetrics(t *testing.T) {
	reset()
	RecordSchedule(1)
	RecordScheduledTermination("Deployment", "app-namespace", "monkey-victim")
	RecordTermination("Deployment", "app-namespace", "monkey-victim", nil)
	RecordPodTermination("Deployment", "app-namespace", "monkey-victim")

	exposed := scrape(t)

	assert.Contains(t, exposed, `kube_monkey_schedule_size 1`)
	assert.Contains(t, exposed, `kube_monkey_scheduled_terminations_total{kind="Deployment",name="monkey-victim",namespace="app-namespace"} 1`)
	assert.Contains(t, exposed, `kube_monkey_terminations_total{kind="Deployment",name="monkey-victim",namespace="app-namespace",result="success"} 1`)
	assert.Contains(t, exposed, `kube_monkey_pods_terminated_total{kind="Deployment",name="monkey-victim",namespace="app-namespace"} 1`)
	assert.Contains(t, exposed, "kube_monkey_last_schedule_timestamp_seconds")
}

func TestGatherExposesProcessMetrics(t *testing.T) {
	assert.Contains(t, scrape(t), "go_goroutines")
}
