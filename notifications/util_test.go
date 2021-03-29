package notifications

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ToHeadersSingle(t *testing.T) {
	headersArray := []string{"Content-Type:application/json"}

	headers := toHeaders(headersArray)

	assert.Equal(t, 1, len(headers))
	assert.Equal(t, "application/json", headers["Content-Type"])
}

func Test_ToHeadersMultiple(t *testing.T) {
	headersArray := []string{"Content-Type:application/json", "Host:localhost"}

	headers := toHeaders(headersArray)

	assert.Equal(t, 2, len(headers))
	assert.Equal(t, "application/json", headers["Content-Type"])
	assert.Equal(t, "localhost", headers["Host"])
}

func Test_ToHeadersEnvVariablePlaceholder(t *testing.T) {
	headersArray := []string{"Content-Type:application/json", "api-key:{$env:API_KEY}"}
	os.Setenv("API_KEY", "123456")

	headers := toHeaders(headersArray)

	assert.Equal(t, 2, len(headers))
	assert.Equal(t, "application/json", headers["Content-Type"])
	assert.Equal(t, "123456", headers["api-key"])
}

func Test_ToHeadersEnvVariablePlaceholderNotExisting(t *testing.T) {
	headersArray := []string{"Content-Type:application/json", "api-key:{$env:VARIABLE_NOT_SET}"}

	headers := toHeaders(headersArray)

	assert.Equal(t, 2, len(headers))
	assert.Equal(t, "application/json", headers["Content-Type"])
	assert.Equal(t, "", headers["api-key"])
}

func Test_NamePlaceholder(t *testing.T) {
	msg := `{"name":"{$name}"}`
	currentTime := time.Now()
	actual := ReplacePlaceholders(msg, "testName", "", "", "", currentTime, "CLUSTER_A")
	assert.Equal(t, `{"name":"testName"}`, actual)
}

func Test_KindPlaceholder(t *testing.T) {
	msg := `{"kind":"{$kind}"}`
	currentTime := time.Now()
	actual := ReplacePlaceholders(msg, "", "testKind", "", "", currentTime, "CLUSTER_A")
	assert.Equal(t, `{"kind":"testKind"}`, actual)
}

func Test_NamespacePlaceholder(t *testing.T) {
	msg := `{"namespace":"{$namespace}"}`
	currentTime := time.Now()
	actual := ReplacePlaceholders(msg, "", "", "testNamespace", "", currentTime, "CLUSTER_A")
	assert.Equal(t, `{"namespace":"testNamespace"}`, actual)
}

func Test_ErrorPlaceholder(t *testing.T) {
	msg := `{"error":"{$error}"}`
	currentTime := time.Now()
	actual := ReplacePlaceholders(msg, "", "", "", "testError", currentTime, "CLUSTER_A")
	assert.Equal(t, `{"error":"testError"}`, actual)
}

func Test_IDPlaceholder(t *testing.T) {
	msg := `{"kubemonkeyid":"{$kubemonkeyid}"}`
	currentTime := time.Now()
	actual := ReplacePlaceholders(msg, "", "", "", "testError", currentTime, "CLUSTER_A")
	assert.Equal(t, `{"kubemonkeyid":"CLUSTER_A"}`, actual)
}

func Test_TimestampPlaceholder(t *testing.T) {
	msg := `{"timestamp":"{$timestamp}"}`
	currentTime := time.Now()
	actual := ReplacePlaceholders(msg, "", "", "", "", currentTime, "CLUSTER_A")
	assert.Equal(t, `{"timestamp":"`+timeToEpoch(currentTime)+`"}`, actual)
}

func Test_TimePlaceholder(t *testing.T) {
	msg := `{"time":"{$time}"}`
	currentTime := time.Now()
	actual := ReplacePlaceholders(msg, "", "", "", "", currentTime, "CLUSTER_A")
	assert.Equal(t, `{"time":"`+timeToTime(currentTime)+`"}`, actual)
}

func Test_DatePlaceholder(t *testing.T) {
	msg := `{"date":"{$date}"}`
	currentTime := time.Now()
	actual := ReplacePlaceholders(msg, "", "", "", "", currentTime, "CLUSTER_A")
	assert.Equal(t, `{"date":"`+timeToDate(currentTime)+`"}`, actual)
}

func Test_MultiplePlaceholders(t *testing.T) {
	msg := `{"date1":"{$date}","date2":"{$date}","name":"{$name}"}`
	currentTime := time.Now()
	actual := ReplacePlaceholders(msg, "testName", "", "", "", currentTime, "CLUSTER_A")
	assert.Equal(t, `{"date1":"`+timeToDate(currentTime)+`","date2":"`+timeToDate(currentTime)+`","name":"testName"}`, actual)
}
