package notifications

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestPostsTheBodyAndHeaders(t *testing.T) {
	var got *http.Request
	var body string

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		read, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		got, body = r, string(read)
	}))
	defer server.Close()

	err := CreateClient("").Request(server.URL+"/hook", `{"text":"hello"}`, map[string]string{"api-key": "123456"})

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, http.MethodPost, got.Method)
	assert.Equal(t, "/hook", got.URL.Path)
	assert.Equal(t, "123456", got.Header.Get("api-key"))
	assert.Equal(t, `{"text":"hello"}`, body)
}

// The response body usually says why the endpoint refused it, which is the only
// clue whoever configured the endpoint gets
func TestRequestReportsWhatTheEndpointSaid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusForbidden)
		_, _ = rw.Write([]byte("Unauthorized\n"))
	}))
	defer server.Close()

	err := CreateClient("").Request(server.URL, "", nil)

	assert.EqualError(t, err, "POST returned 403 Unauthorized, expected 2xx")
}

// A secret can sit in the endpoint's path, and callers log these errors
func TestRequestKeepsTheEndpointOutOfItsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := CreateClient("").Request(server.URL+"/hook/secret-token", "", nil)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret-token")
}

func TestRequestWhenTheEndpointCannotBeReached(t *testing.T) {
	err := CreateClient("").Request("http://127.0.0.1:1/hook", "", nil)

	assert.ErrorContains(t, err, "http request")
}

func TestRequestWithAnEndpointThatIsNotAURL(t *testing.T) {
	err := CreateClient("").Request("://nonsense", "", nil)

	assert.ErrorContains(t, err, "new http request")
}

// Every notification goes through the proxy when one is configured
func TestCreateClientWithAProxy(t *testing.T) {
	var proxied string
	proxy := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		proxied = r.URL.String()
	}))
	defer proxy.Close()

	err := CreateClient(proxy.URL).Request("http://example.test/hook", "", nil)

	require.NoError(t, err)
	assert.Equal(t, "http://example.test/hook", proxied)
}

// A proxy nobody can parse should not stop the notification going out directly
func TestCreateClientWithAProxyThatIsNotAURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()

	assert.NoError(t, CreateClient("://nonsense").Request(server.URL, "", nil))
}
