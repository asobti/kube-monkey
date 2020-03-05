package notifications

import (
	"fmt"
	"time"

	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config"
	"github.com/golang/glog"
)

func Send(client Client, endpoint string, msg string, headers map[string]string) error {
	if err := client.Request(endpoint, msg, headers); err != nil {
		return fmt.Errorf("send request: %v", err)
	}
	return nil
}

func ReportAttack(client Client, result *chaos.Result, time time.Time) bool {
	success := true

	for _, receivers := range config.NotificationsAttacks() {
		errorString := ""
		if result.Error() != nil {
			errorString = result.Error().Error()
		}
		msg := ReplacePlaceholders(receivers.Message, result.Victim().Name(), result.Victim().Kind(), result.Victim().Namespace(), errorString, time)
		glog.V(1).Infof("reporting attack for %s %s to %s with message %s\n", result.Victim().Kind(), result.Victim().Name(), receivers.Endpoint, msg)
		if err := Send(client, receivers.Endpoint, msg, toHeaders(receivers.Headers)); err != nil {
			glog.Errorf("error reporting attack for %s %s to %s with message %s, error: %v\n", result.Victim().Kind(), result.Victim().Name(), receivers.Endpoint, msg, err)
			success = false
		}
	}
	return success
}
