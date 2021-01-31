package notifications

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

const (
	// header
	EnvVariableRegex = "^{\\$env:\\w+\\}$"

	// body (message)
	Name      = "{$name}"
	Kind      = "{$kind}"
	Namespace = "{$namespace}"
	Timestamp = "{$timestamp}"
	Time      = "{$time}"
	Date      = "{$date}"
	Error     = "{$error}"
)

func toHeaders(headersArray []string) map[string]string {
	headersMap := make(map[string]string)

	for _, h := range headersArray {
		kv := strings.SplitN(h, ":", 2)
		if len(kv) == 1 {
			glog.Errorf("Cannot find ':' separator in supplied header %s", h)
			headersMap[strings.TrimSpace(kv[0])] = ""
			continue
		}
		headersMap[strings.TrimSpace(kv[0])] = replaceEnvVariablePlaceholder(strings.TrimSpace(kv[1]))
	}
	return headersMap
}

func replaceEnvVariablePlaceholder(value string) string {
	envVariableRegex := regexp.MustCompile(EnvVariableRegex)
	if envVariableRegex.MatchString(value) {
		prefix, _ := envVariableRegex.LiteralPrefix()
		envVariableName := value[len(prefix) : len(value)-1]
		envVariableValue := os.Getenv(envVariableName)
		if len(envVariableValue) == 0 {
			glog.Errorf("Cannot find environment variable %s", envVariableName)
		}
		value = envVariableRegex.ReplaceAllString(value, envVariableValue)
	}
	return value
}

func ReplacePlaceholders(msg string, name string, kind string, namespace string, err string, attackTime time.Time) string {
	msg = strings.Replace(msg, Name, name, -1)
	msg = strings.Replace(msg, Kind, kind, -1)
	msg = strings.Replace(msg, Namespace, namespace, -1)
	msg = strings.Replace(msg, Timestamp, timeToEpoch(attackTime), -1)
	msg = strings.Replace(msg, Time, timeToTime(attackTime), -1)
	msg = strings.Replace(msg, Date, timeToDate(attackTime), -1)
	msg = strings.Replace(msg, Error, err, -1)

	return msg
}

func timeToEpoch(time time.Time) string {
	epoch := time.UnixNano() / 1000000

	return strconv.FormatInt(epoch, 10)
}

func timeToDate(time time.Time) string {
	return time.Format("2006-01-02")
}

func timeToTime(time time.Time) string {
	return time.Format("15:04:05 MST")
}
