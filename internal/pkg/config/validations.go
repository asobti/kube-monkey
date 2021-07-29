package config

import (
	"fmt"
	"regexp"

	"kube-monkey/internal/pkg/config/param"
)

func ValidateConfigs() error {
	// RunHour should be [0, 23]
	runHour := RunHour()
	if !IsValidHour(runHour) {
		return fmt.Errorf("RunHour: %s is outside valid range of [0,23]", param.RunHour)
	}

	// StartHour should be [0, 23]
	startHour := StartHour()
	if !IsValidHour(startHour) {
		return fmt.Errorf("StartHour: %s is outside valid range of [0,23]", param.StartHour)
	}

	// EndHour should be [0, 23]
	endHour := EndHour()
	if !IsValidHour(endHour) {
		return fmt.Errorf("EndHour: %s is outside valid range of [0,23]", param.EndHour)
	}

	// StartHour should be < EndHour
	if !(startHour < endHour) {
		return fmt.Errorf("StartHour: %s must be less than %s", param.StartHour, param.EndHour)
	}

	// RunHour should be < StartHour
	if !(runHour < startHour) {
		return fmt.Errorf("RunHour: %s should be less than %s", param.RunHour, param.StartHour)
	}

	notificationsReceiver := NotificationsAttacks()

	// Notification headers should be in a valid format
	for _, header := range notificationsReceiver.Headers {
		if !isValidHeader(header) {
			return fmt.Errorf("Header: %s is not in valid format", header)
		}
	}

	return nil
}

func IsValidHour(hour int) bool {
	return hour >= 0 && hour < 24
}

func isValidHeader(header string) bool {
	re := regexp.MustCompile("^(.+:.+)$")

	return re.MatchString(header)
}
