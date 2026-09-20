package notifications

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToHeaders(t *testing.T) {
	headers := toHeaders([]string{
		"Content-Type:application/json",
		"Host:localhost",
		// A value can hold colons of its own, only the first one separates
		"Authorization:Bearer abc:def",
		// Spacing around the parts is for whoever reads the config, not the server
		"  X-Source : kube-monkey  ",
	})

	assert.Equal(t, map[string]string{
		"Content-Type":  "application/json",
		"Host":          "localhost",
		"Authorization": "Bearer abc:def",
		"X-Source":      "kube-monkey",
	}, headers)
}

func TestToHeadersWithoutASeparator(t *testing.T) {
	// Nothing useful can be sent, but the rest of the headers still go
	headers := toHeaders([]string{"no-separator", "Host:localhost"})

	assert.Equal(t, map[string]string{"no-separator": "", "Host": "localhost"}, headers)
}

// A token belongs in the environment rather than in the config file
func TestToHeadersResolvesAnEnvironmentVariable(t *testing.T) {
	t.Setenv("API_KEY", "123456")

	headers := toHeaders([]string{"Content-Type:application/json", "api-key:{$env:API_KEY}"})

	assert.Equal(t, "123456", headers["api-key"])
	assert.Equal(t, "application/json", headers["Content-Type"])
}

func TestToHeadersWithAnEnvironmentVariableThatIsNotSet(t *testing.T) {
	headers := toHeaders([]string{"api-key:{$env:VARIABLE_NOT_SET}"})

	assert.Equal(t, "", headers["api-key"])
}

func TestResolveEnvPlaceholder(t *testing.T) {
	t.Setenv("ENDPOINT", "https://example.test/hook")

	assert.Equal(t, "https://example.test/hook", resolveEnvPlaceholder("{$env:ENDPOINT}"))

	// Only a value that is nothing but a placeholder is resolved, so a real
	// value that happens to look similar is left alone
	for _, value := range []string{
		"https://example.test/hook",
		"https://example.test/{$env:ENDPOINT}",
		"{$env:ENDPOINT} ",
		"{$env:not a name}",
		"",
	} {
		assert.Equal(t, value, resolveEnvPlaceholder(value), value)
	}
}

func TestReplacePlaceholders(t *testing.T) {
	attackTime := time.Date(2024, 3, 12, 14, 30, 5, 0, time.UTC)

	msg := ReplacePlaceholders(
		`{"name":"{$name}","kind":"{$kind}","namespace":"{$namespace}","error":"{$error}","id":"{$kubemonkeyid}","timestamp":{$timestamp},"time":"{$time}","date":"{$date}"}`,
		"shop", "v1.Deployment", "team-checkout", "no running pods", attackTime, "cluster-a",
	)

	assert.JSONEq(t, `{
		"name": "shop",
		"kind": "v1.Deployment",
		"namespace": "team-checkout",
		"error": "no running pods",
		"id": "cluster-a",
		"timestamp": 1710253805000,
		"time": "14:30:05 UTC",
		"date": "2024-03-12"
	}`, msg)
}

// A placeholder can be used as often as the message needs it
func TestReplacePlaceholdersRepeated(t *testing.T) {
	attackTime := time.Date(2024, 3, 12, 14, 30, 5, 0, time.UTC)

	msg := ReplacePlaceholders(`{$name} on {$date}, {$name} again on {$date}`, "shop", "", "", "", attackTime, "")

	assert.Equal(t, "shop on 2024-03-12, shop again on 2024-03-12", msg)
}

// A message with nothing to fill in comes back as it went in
func TestReplacePlaceholdersLeavesPlainTextAlone(t *testing.T) {
	msg := ReplacePlaceholders(`{"text":"kube-monkey struck again"}`, "shop", "v1.Deployment", "team", "", time.Now(), "id")

	assert.Equal(t, `{"text":"kube-monkey struck again"}`, msg)
}
