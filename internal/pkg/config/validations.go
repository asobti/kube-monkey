package config

import (
	"fmt"
	"net"
	"path"
	"regexp"
	"strings"

	"github.com/spf13/viper"

	"kube-monkey/internal/pkg/config/param"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
)

func ValidateConfigs() error {
	// RunDays should name at least one day of the week
	if _, err := parseRunDays(); err != nil {
		return err
	}

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

	// A custom resource entry that does not describe a real resource would
	// silently schedule nothing, so reject it here rather than at kill time
	if err := validateCustomResources(); err != nil {
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

func validateCustomResources() error {
	var resources []CustomResource
	if err := viper.UnmarshalKey(param.CustomResources, &resources); err != nil {
		return fmt.Errorf("CustomResources: %s could not be read: %v", param.CustomResources, err)
	}

	seen := sets.NewString()
	for _, resource := range resources {
		if resource.Version == "" || resource.Resource == "" {
			return fmt.Errorf("CustomResources: %s has an entry without a version and a resource", param.CustomResources)
		}

		// The plural resource name, not the kind, because that is what the
		// dynamic client asks the API server for. "Cluster" would 404
		if strings.ToLower(resource.Resource) != resource.Resource {
			return fmt.Errorf("CustomResources: %q should be the lowercase plural resource name, e.g. %q rather than the kind", resource.Resource, strings.ToLower(resource.Resource))
		}

		if resource.PodLabel != "" {
			if errs := validation.IsQualifiedName(resource.PodLabel); len(errs) > 0 {
				return fmt.Errorf("CustomResources: %q is not a valid label key: %s", resource.PodLabel, strings.Join(errs, ", "))
			}
		}

		// Two entries for one resource would schedule every victim twice
		if seen.Has(resource.Name()) {
			return fmt.Errorf("CustomResources: %s is listed more than once", resource.Name())
		}
		seen.Insert(resource.Name())
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
