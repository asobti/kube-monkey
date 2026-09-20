package metrics

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeRejectsInvalidAddress(t *testing.T) {
	assert.Error(t, Serve("not-a-host-port"))
}

func TestServeAnswersScrapes(t *testing.T) {
	reset()
	RecordSchedule(4)

	// Port 0 lets the kernel pick a free port, so the test never clashes with
	// whatever else is listening on the machine
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	serve(listener)

	response, err := http.Get("http://" + listener.Addr().String() + endpointPath)
	require.NoError(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Contains(t, string(body), "kube_monkey_schedule_size 4")
}

func TestHandlerServesPrometheusText(t *testing.T) {
	reset()
	RecordSchedule(2)

	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, endpointPath, nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "text/plain")
	assert.Contains(t, recorder.Body.String(), "kube_monkey_schedule_size 2")
}

// scrape returns the metrics as a scraper would see them
func scrape(t *testing.T) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, endpointPath, nil))

	body, err := io.ReadAll(recorder.Body)
	assert.NoError(t, err)
	return string(body)
}
