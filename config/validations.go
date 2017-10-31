package config

import (
	"fmt"
	
	"github.com/asobti/kube-monkey/config/param"
)

func ValidateConfigs() error {
	// RunHour should be [0, 23]
	runHour := RunHour()

	if !IsValidHour(runHour) {
		return fmt.Errorf("%s is outside valid range of [0,23]", param.RunHour)
	}

	// StartHour should be [0, 23]
	startHour := StartHour()
	if !IsValidHour(startHour) {
		return fmt.Errorf("%s is outside valid range of [0,23]", param.StartHour)
	}

	// EndHour should be [0, 23]
	endHour := EndHour()
	if !IsValidHour(endHour) {
		return fmt.Errorf("%s is outside valid range of [0,23]", param.EndHour)
	}

	// StartHour should be < EndHour
	if !(startHour < endHour) {
		return fmt.Errorf("%s must be less than %s", param.StartHour, param.EndHour)
	}

	// RunHour should be < StartHour
	if !(runHour < startHour) {
		return fmt.Errorf("%s should be less than %s", param.RunHour, param.StartHour)
	}

	return nil
}

func IsValidHour(hour int) bool {
	return hour >= 0 && hour < 24
}
