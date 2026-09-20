package notifications

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

// Placeholders the configured message is written with
const (
	Name         = "{$name}"
	Kind         = "{$kind}"
	Namespace    = "{$namespace}"
	Timestamp    = "{$timestamp}"
	Time         = "{$time}"
	Date         = "{$date}"
	Error        = "{$error}"
	KubeMonkeyID = "{$kubemonkeyid}"
)

// A header value or the endpoint can be given as {$env:NAME} to keep a secret
// out of the config file
var envPlaceholder = regexp.MustCompile(`^\{\$env:(\w+)\}$`)

// toHeaders turns the configured "key:value" lines into request headers
func toHeaders(headerLines []string) map[string]string {
	headers := make(map[string]string, len(headerLines))

	for _, line := range headerLines {
		key, value, found := strings.Cut(line, ":")
		if !found {
			glog.Errorf("Cannot find ':' separator in supplied header %s", line)
		}
		headers[strings.TrimSpace(key)] = resolveEnvPlaceholder(strings.TrimSpace(value))
	}

	return headers
}

// resolveEnvPlaceholder swaps a whole {$env:NAME} value for what NAME holds,
// and leaves anything else alone
func resolveEnvPlaceholder(value string) string {
	match := envPlaceholder.FindStringSubmatch(value)
	if match == nil {
		return value
	}

	name := match[1]
	resolved := os.Getenv(name)
	if resolved == "" {
		glog.Errorf("Cannot find environment variable %s", name)
	}

	return resolved
}

// ReplacePlaceholders fills the placeholders in the configured message with the
// details of one attack
func ReplacePlaceholders(msg string, name string, kind string, namespace string, err string, attackTime time.Time, kubeMonkeyID string) string {
	return strings.NewReplacer(
		Name, name,
		Kind, kind,
		Namespace, namespace,
		Timestamp, strconv.FormatInt(attackTime.UnixMilli(), 10),
		Time, attackTime.Format("15:04:05 MST"),
		Date, attackTime.Format("2006-01-02"),
		Error, err,
		KubeMonkeyID, kubeMonkeyID,
	).Replace(msg)
}
