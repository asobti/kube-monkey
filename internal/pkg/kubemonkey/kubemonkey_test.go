package kubemonkey

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"kube-monkey/internal/pkg/chaos"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/config/param"
	"kube-monkey/internal/pkg/metrics"
	"kube-monkey/internal/pkg/notifications"
	"kube-monkey/internal/pkg/victims"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	config.SetDefaults()
	m.Run()
}

func newResult(name string, err error) *chaos.Result {
	victim := victims.New(victims.Spec{Kind: "v1.Deployment", Name: name, Namespace: "team-checkout"})
	return chaos.New(time.Now(), victim).NewResult(err)
}

// scrapeMetrics returns the metrics as a scraper would see them
func scrapeMetrics(t *testing.T) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return recorder.Body.String()
}

// metricValue reads one metric out of a scrape, by the name and labels it is
// exposed under. Counters carry on from wherever the last test left them, so
// tests compare a value before with the value after.
func metricValue(t *testing.T, scraped, metric string) float64 {
	t.Helper()

	for line := range strings.SplitSeq(scraped, "\n") {
		name, value, found := strings.Cut(line, " ")
		if !found || name != metric {
			continue
		}

		parsed, err := strconv.ParseFloat(value, 64)
		require.NoError(t, err)
		return parsed
	}

	return 0
}

func terminationsMetric(name, result string) string {
	return fmt.Sprintf(`kube_monkey_terminations_total{kind="v1.Deployment",name=%q,namespace="team-checkout",result=%q}`, name, result)
}

func scheduledMetric(name string) string {
	return fmt.Sprintf(`kube_monkey_scheduled_terminations_total{kind="v1.Deployment",name=%q,namespace="team-checkout"}`, name)
}

func TestReportResultsWaitsForEveryTermination(t *testing.T) {
	results := make(chan *chaos.Result, 3)
	for i := range 3 {
		results <- newResult(fmt.Sprintf("victim-%d", i), nil)
	}

	done := make(chan struct{})
	go func() {
		reportResults(results, 3, notifications.Client{})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reportResults did not return once every termination had reported back")
	}

	assert.Empty(t, results, "every result should have been taken off the channel")
}

// A termination that was skipped counts as a failure, so the numbers tell an
// operator whether chaos is really happening
func TestReportResultsRecordsTheOutcomeOfEachTermination(t *testing.T) {
	before := scrapeMetrics(t)

	results := make(chan *chaos.Result, 2)
	results <- newResult("outcome-victim", nil)
	results <- newResult("outcome-victim", errors.New("no longer enrolled"))

	reportResults(results, 2, notifications.Client{})

	after := scrapeMetrics(t)
	for _, result := range []string{"success", "failure"} {
		metric := terminationsMetric("outcome-victim", result)
		assert.Equal(t, 1.0, metricValue(t, after, metric)-metricValue(t, before, metric), result)
	}
}

func TestReportResultsNotifiesWhenNotificationsAreOn(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()

	viper.Set(param.NotificationsEnabled, true)
	viper.Set(param.NotificationsAttacks, map[string]any{"endpoint": server.URL, "message": `{"text":"{$name}"}`})
	t.Cleanup(func() {
		viper.Reset()
		config.SetDefaults()
	})

	results := make(chan *chaos.Result, 2)
	results <- newResult("notified-victim", nil)
	results <- newResult("notified-victim", errors.New("no running pods"))

	reportResults(results, 2, notifications.CreateClient(""))

	assert.Equal(t, 2, requests, "both the successful and the failed termination should be reported")
}

func TestReportResultsStaysQuietWhenNotificationsAreOff(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()

	viper.Set(param.NotificationsAttacks, map[string]any{"endpoint": server.URL})
	t.Cleanup(func() {
		viper.Reset()
		config.SetDefaults()
	})

	results := make(chan *chaos.Result, 1)
	results <- newResult("quiet-victim", nil)

	reportResults(results, 1, notifications.Client{})

	assert.Equal(t, 0, requests)
}

// Outside a cluster every termination fails to get a client, which is enough to
// show the schedule is run through to the end and recorded
func TestScheduleTerminationsRunsEveryEntry(t *testing.T) {
	entries := []*chaos.Chaos{
		chaos.New(time.Now(), victims.New(victims.Spec{Kind: "v1.Deployment", Name: "scheduled-a", Namespace: "team-checkout"})),
		chaos.New(time.Now(), victims.New(victims.Spec{Kind: "v1.Deployment", Name: "scheduled-b", Namespace: "team-checkout"})),
	}
	before := scrapeMetrics(t)

	done := make(chan struct{})
	go func() {
		ScheduleTerminations(entries, notifications.Client{})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("ScheduleTerminations did not return once every entry had been dealt with")
	}

	after := scrapeMetrics(t)
	assert.Equal(t, 2.0, metricValue(t, after, "kube_monkey_schedule_size"))
	for _, name := range []string{"scheduled-a", "scheduled-b"} {
		metric := scheduledMetric(name)
		assert.Equal(t, 1.0, metricValue(t, after, metric)-metricValue(t, before, metric), name)
		assert.Equal(t, 1.0, metricValue(t, after, terminationsMetric(name, "failure"))-metricValue(t, before, terminationsMetric(name, "failure")), name)
	}
}

func TestScheduleTerminationsWithAnEmptySchedule(t *testing.T) {
	ScheduleTerminations(nil, notifications.Client{})

	assert.Equal(t, 0.0, metricValue(t, scrapeMetrics(t), "kube_monkey_schedule_size"))
}

func TestDurationToNextRunInDebugMode(t *testing.T) {
	viper.Set(param.DebugEnabled, true)
	viper.Set(param.DebugScheduleDelay, 45)
	t.Cleanup(func() {
		viper.Reset()
		config.SetDefaults()
	})

	assert.Equal(t, 45*time.Second, durationToNextRun(8, time.UTC, []time.Weekday{time.Monday}))
}

// Every day is a run day, so the next run is at most a day away and never in
// the past
func TestDurationToNextRun(t *testing.T) {
	everyDay := []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday}

	duration := durationToNextRun(8, time.UTC, everyDay)

	assert.Positive(t, duration)
	assert.LessOrEqual(t, duration, 24*time.Hour)
}

func TestDurationToNextRunOnASingleRunDay(t *testing.T) {
	// Whichever day it is, the next Wednesday is never more than a week away
	duration := durationToNextRun(8, time.UTC, []time.Weekday{time.Wednesday})

	assert.Positive(t, duration)
	assert.LessOrEqual(t, duration, 7*24*time.Hour)
}

func TestReportResultsIsFineWithNothingToReport(t *testing.T) {
	require.NotPanics(t, func() {
		reportResults(make(chan *chaos.Result), 0, notifications.Client{})
	})
}
