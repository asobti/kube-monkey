package notifications

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kube-monkey/internal/pkg/chaos"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/config/param"
	"kube-monkey/internal/pkg/schedule"
	"kube-monkey/internal/pkg/victims"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingServer answers every request and keeps the last one
type recordingServer struct {
	*httptest.Server
	requests int
	path     string
	body     string
	headers  http.Header
}

func newRecordingServer(t *testing.T) *recordingServer {
	t.Helper()

	recorder := &recordingServer{}
	recorder.Server = httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		recorder.requests++
		recorder.path, recorder.body, recorder.headers = r.URL.Path, string(body), r.Header
	}))
	t.Cleanup(recorder.Close)

	return recorder
}

func configureReceiver(t *testing.T, receiver map[string]any) {
	t.Helper()

	viper.Set(param.NotificationsAttacks, receiver)
	t.Cleanup(func() {
		viper.Reset()
		config.SetDefaults()
	})
}

func newResult(err error) *chaos.Result {
	victim := victims.New(victims.Spec{Kind: "v1.Deployment", Name: "shop", Namespace: "team-checkout"})
	return chaos.New(time.Now(), victim).NewResult(err)
}

func TestReportSchedule(t *testing.T) {
	server := newRecordingServer(t)
	configureReceiver(t, map[string]any{"endpoint": server.URL + "/hook"})

	err := ReportSchedule(CreateClient(""), &schedule.Schedule{})

	require.NoError(t, err)
	assert.Equal(t, 1, server.requests)
	assert.Equal(t, "/hook", server.path)
	assert.Contains(t, server.body, "No terminations scheduled")
}

// The endpoint can hold a secret, so it can be kept in the environment instead
// of the config file
func TestReportScheduleResolvesAnEnvironmentEndpoint(t *testing.T) {
	server := newRecordingServer(t)
	t.Setenv("TEST_NOTIFICATION_ENDPOINT", server.URL)
	configureReceiver(t, map[string]any{"endpoint": "{$env:TEST_NOTIFICATION_ENDPOINT}"})

	require.NoError(t, ReportSchedule(CreateClient(""), &schedule.Schedule{}))
	assert.Equal(t, 1, server.requests)
}

func TestReportScheduleWhenTheEndpointRefuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	configureReceiver(t, map[string]any{"endpoint": server.URL})

	assert.ErrorContains(t, ReportSchedule(CreateClient(""), &schedule.Schedule{}), "expected 2xx")
}

func TestReportAttackFillsInTheConfiguredMessage(t *testing.T) {
	server := newRecordingServer(t)
	configureReceiver(t, map[string]any{
		"endpoint": server.URL,
		"message":  `{"text":"{$kind} {$name} in {$namespace} on {$date}","error":"{$error}"}`,
		"headers":  []string{"Content-Type:application/json"},
	})
	t.Setenv("KUBE_MONKEY_ID", "cluster-a")

	attackTime := time.Date(2024, 3, 12, 14, 30, 5, 0, time.UTC)
	err := ReportAttack(CreateClient(""), newResult(nil), attackTime)

	require.NoError(t, err)
	assert.Equal(t, 1, server.requests)
	assert.JSONEq(t, `{"text":"v1.Deployment shop in team-checkout on 2024-03-12","error":""}`, server.body)
	assert.Equal(t, "application/json", server.headers.Get("Content-Type"))
}

// A termination that did not happen is worth reporting too, with the reason
func TestReportAttackReportsAFailedTermination(t *testing.T) {
	server := newRecordingServer(t)
	configureReceiver(t, map[string]any{"endpoint": server.URL, "message": `{"error":"{$error}"}`})

	err := ReportAttack(CreateClient(""), newResult(errors.New("no running pods")), time.Now())

	require.NoError(t, err)
	assert.JSONEq(t, `{"error":"no running pods"}`, server.body)
}
