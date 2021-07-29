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

func Send(client Client, endpoint string, msg string, headers map[string]string) error {
	if err := client.Request(endpoint, msg, headers); err != nil {
		return fmt.Errorf("send request: %v", err)
	}
	return nil
}

func ReportSchedule(client Client, schedule *schedule.Schedule) bool {
	success := true
	receiver := config.NotificationsAttacks()

	msg := fmt.Sprintf("{\"text\": \"\n%s\n\"}", schedule)

	glog.V(1).Infof("reporting next schedule")
	if err := Send(client, receiver.Endpoint, msg, toHeaders(receiver.Headers)); err != nil {
		glog.Errorf("error reporting next schedule")
		success = false
	}

	return success
}

func ReportAttack(client Client, result *chaos.Result, time time.Time) bool {
	success := true

	receiver := config.NotificationsAttacks()
	errorString := ""
	if result.Error() != nil {
		errorString = result.Error().Error()
	}
	msg := ReplacePlaceholders(receiver.Message, result.Victim().Name(), result.Victim().Kind(), result.Victim().Namespace(), errorString, time, os.Getenv("KUBE_MONKEY_ID"))
	glog.V(1).Infof("reporting attack for %s %s to %s with message %s\n", result.Victim().Kind(), result.Victim().Name(), receiver.Endpoint, msg)
	if err := Send(client, receiver.Endpoint, msg, toHeaders(receiver.Headers)); err != nil {
		glog.Errorf("error reporting attack for %s %s to %s with message %s, error: %v\n", result.Victim().Kind(), result.Victim().Name(), receiver.Endpoint, msg, err)
		success = false
	}

	return success
}
