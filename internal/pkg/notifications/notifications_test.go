package notifications

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"kube-monkey/internal/pkg/config/param"
	"kube-monkey/internal/pkg/schedule"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func Test_ReportScheduleEndpoint(t *testing.T) {
	server, requests := recordingServer()
	defer server.Close()

	viper.Set(param.NotificationsAttacks, map[string]interface{}{"endpoint": server.URL})
	defer viper.Reset()

	assert.True(t, ReportSchedule(CreateClient(nil), &schedule.Schedule{}))
	assert.Equal(t, 1, *requests)
}

func Test_ReportScheduleEnvVariableEndpoint(t *testing.T) {
	server, requests := recordingServer()
	defer server.Close()

	os.Setenv("TEST_NOTIFICATION_ENDPOINT", server.URL)
	defer os.Unsetenv("TEST_NOTIFICATION_ENDPOINT")

	viper.Set(param.NotificationsAttacks, map[string]interface{}{"endpoint": "{$env:TEST_NOTIFICATION_ENDPOINT}"})
	defer viper.Reset()

	assert.True(t, ReportSchedule(CreateClient(nil), &schedule.Schedule{}))
	assert.Equal(t, 1, *requests)
}

func recordingServer() (*httptest.Server, *int) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		requests++
	}))
	return server, &requests
}
