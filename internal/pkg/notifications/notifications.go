/*
Package notifications reports what kube-monkey is about to do, and what it did,
to an HTTP endpoint.
*/
package notifications

import (
	"fmt"
	"os"
	"time"

	"kube-monkey/internal/pkg/chaos"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/schedule"

	"github.com/golang/glog"
)

// ReportSchedule posts today's schedule to the configured endpoint
func ReportSchedule(client Client, schedule *schedule.Schedule) error {
	receiver := config.NotificationsAttacks()
	msg := fmt.Sprintf("{\"text\": \"\n%s\n\"}", schedule)

	glog.V(1).Infof("reporting next schedule")
	return client.Request(resolveEnvPlaceholder(receiver.Endpoint), msg, toHeaders(receiver.Headers))
}

// ReportAttack posts the outcome of a single termination to the configured
// endpoint
func ReportAttack(client Client, result *chaos.Result, attackTime time.Time) error {
	receiver := config.NotificationsAttacks()
	victim := result.Victim()

	errorString := ""
	if result.Error() != nil {
		errorString = result.Error().Error()
	}
	msg := ReplacePlaceholders(receiver.Message, victim.Name(), victim.Kind(), victim.Namespace(), errorString, attackTime, os.Getenv("KUBE_MONKEY_ID"))

	// Logs show the configured endpoint, not the resolved one, because a
	// resolved endpoint can carry a secret token in its path
	glog.V(1).Infof("reporting attack for %s %s to %s with message %s\n", victim.Kind(), victim.Name(), receiver.Endpoint, msg)

	return client.Request(resolveEnvPlaceholder(receiver.Endpoint), msg, toHeaders(receiver.Headers))
}
