package notifications

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestSuccess(t *testing.T) {
	path := "/path"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if want, have := path, r.URL.Path; want != have {
			t.Errorf("unexpected endpoint, want: %q, have %q", want, have)
		}
	}))
	defer server.Close()

	c := CreateClient(nil)
	body := "message"
	err := c.Request(server.URL+path, body, map[string]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRequestFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(403)
		_, _ = rw.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	c := CreateClient(nil)
	body := ""
	err := c.Request(server.URL, body, map[string]string{})
	expectedErr := fmt.Sprintf("POST %s returned 403 Unauthorized, expected 2xx", server.URL)
	if want, have := expectedErr, err.Error(); want != have {
		t.Errorf("unexpected error, want %q, have %q", want, have)
	}
}
