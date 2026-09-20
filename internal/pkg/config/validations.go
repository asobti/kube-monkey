package config

import (
	"fmt"
	"net"
	"path"
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

	// Namespace entries are patterns, so a typo like "[foo" would silently
	// stop matching the namespaces it was meant to cover
	if err := validateNamespacePatterns("BlacklistedNamespaces", param.BlacklistedNamespaces, BlacklistedNamespaces().UnsortedList()); err != nil {
		return err
	}

	if err := validateNamespacePatterns("WhitelistedNamespaces", param.WhitelistedNamespaces, WhitelistedNamespaces().UnsortedList()); err != nil {
		return err
	}

	notificationsReceiver := NotificationsAttacks()

	// Notification headers should be in a valid format
	for _, header := range notificationsReceiver.Headers {
		if !isValidHeader(header) {
			return fmt.Errorf("Header: %s is not in valid format", header)
		}
	}

	// Metrics need somewhere to listen
	if MetricsEnabled() {
		if _, _, err := net.SplitHostPort(MetricsAddress()); err != nil {
			return fmt.Errorf("MetricsAddress: %s is not a valid host:port address: %v", param.MetricsAddress, err)
		}
	}

	return nil
}

func validateNamespacePatterns(name, key string, patterns []string) error {
	for _, pattern := range patterns {
		if _, err := path.Match(pattern, ""); err != nil {
			return fmt.Errorf("%s: %s contains an invalid pattern %q: %v", name, key, pattern, err)
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
