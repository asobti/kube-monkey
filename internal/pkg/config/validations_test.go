package config

import (
	"testing"

	"kube-monkey/internal/pkg/config/param"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// freshConfig gives a test the defaults and nothing else. Values set with
// viper.Set outrank defaults and survive SetDefaults, so the reset matters.
func freshConfig(t *testing.T) {
	t.Helper()

	viper.Reset()
	SetDefaults()
	t.Cleanup(func() {
		viper.Reset()
		SetDefaults()
	})
}

func TestValidateConfigsAcceptsTheDefaults(t *testing.T) {
	freshConfig(t)

	assert.NoError(t, ValidateConfigs())
}

func TestValidateHours(t *testing.T) {
	for name, tc := range map[string]struct {
		runHour     int
		startHour   int
		endHour     int
		expectedErr string
	}{
		"the usual working day":      {8, 10, 16, ""},
		"run hour out of range":      {24, 10, 16, "RunHour: " + param.RunHour + " is outside valid range of [0,23]"},
		"negative run hour":          {-1, 10, 16, "RunHour: " + param.RunHour + " is outside valid range of [0,23]"},
		"start hour out of range":    {8, 24, 16, "StartHour: " + param.StartHour + " is outside valid range of [0,23]"},
		"end hour out of range":      {8, 10, 24, "EndHour: " + param.EndHour + " is outside valid range of [0,23]"},
		"start hour after end hour":  {8, 16, 10, "StartHour: " + param.StartHour + " must be less than " + param.EndHour},
		"start hour on the end hour": {8, 16, 16, "StartHour: " + param.StartHour + " must be less than " + param.EndHour},
		"run hour after start hour":  {11, 10, 16, "RunHour: " + param.RunHour + " should be less than " + param.StartHour},
		"run hour on the start hour": {10, 10, 16, "RunHour: " + param.RunHour + " should be less than " + param.StartHour},
	} {
		t.Run(name, func(t *testing.T) {
			freshConfig(t)
			viper.Set(param.RunHour, tc.runHour)
			viper.Set(param.StartHour, tc.startHour)
			viper.Set(param.EndHour, tc.endHour)

			err := ValidateConfigs()

			if tc.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr)
			}
		})
	}
}

func TestValidateRunDays(t *testing.T) {
	freshConfig(t)

	viper.Set(param.RunDays, []string{"Saturday", "SUN"})
	assert.NoError(t, ValidateConfigs())

	viper.Set(param.RunDays, []string{})
	assert.EqualError(t, ValidateConfigs(), "RunDays: "+param.RunDays+" must list at least one day")

	viper.Set(param.RunDays, []string{"mon", "funday"})
	assert.ErrorContains(t, ValidateConfigs(), "RunDays: "+param.RunDays+` contains an invalid day "funday"`)
}

// An unknown zone would otherwise only surface when the first kill time is
// worked out, long after kube-monkey reports itself as started
func TestValidateTimezone(t *testing.T) {
	freshConfig(t)

	viper.Set(param.Timezone, "Europe/Lisbon")
	assert.NoError(t, ValidateConfigs())

	viper.Set(param.Timezone, "Middle/Earth")
	assert.ErrorContains(t, ValidateConfigs(), "Timezone: "+param.Timezone+" is not a known time zone")
}

func TestValidateMetricsAddress(t *testing.T) {
	freshConfig(t)
	viper.Set(param.MetricsEnabled, true)

	assert.NoError(t, ValidateConfigs(), "the default address should be usable")

	viper.Set(param.MetricsAddress, "8080")
	assert.ErrorContains(t, ValidateConfigs(), "MetricsAddress: "+param.MetricsAddress+" is not a valid host:port address")

	viper.Set(param.MetricsAddress, "127.0.0.1:8080")
	assert.NoError(t, ValidateConfigs())
}

// An address is only worth checking when something is going to listen on it
func TestValidateMetricsAddressIgnoredWhenMetricsAreOff(t *testing.T) {
	freshConfig(t)
	viper.Set(param.MetricsAddress, "not-a-host-port")

	assert.NoError(t, ValidateConfigs())
}

// A typo like "[foo" would silently stop matching the namespaces it was meant
// to cover, so it is caught when the config loads rather than at kill time
func TestValidateNamespacePatterns(t *testing.T) {
	freshConfig(t)

	viper.Set(param.BlacklistedNamespaces, []string{"kube-*", "team-?"})
	viper.Set(param.WhitelistedNamespaces, []string{"*-staging"})
	assert.NoError(t, ValidateConfigs())

	viper.Set(param.BlacklistedNamespaces, []string{"[kube-system"})
	assert.ErrorContains(t, ValidateConfigs(), "BlacklistedNamespaces: "+param.BlacklistedNamespaces+" contains an invalid pattern")

	viper.Set(param.BlacklistedNamespaces, []string{"kube-system"})
	viper.Set(param.WhitelistedNamespaces, []string{"[default"})
	assert.ErrorContains(t, ValidateConfigs(), "WhitelistedNamespaces: "+param.WhitelistedNamespaces+" contains an invalid pattern")
}

func TestIsValidHour(t *testing.T) {
	for hour := 0; hour <= 23; hour++ {
		assert.True(t, IsValidHour(hour), hour)
	}

	assert.False(t, IsValidHour(24))
	assert.False(t, IsValidHour(-1))
}

func TestIsValidHeader(t *testing.T) {
	for _, header := range []string{
		"header1Key:header1Value",
		"header1/Key:header1/Value",
		"header1:{$env:VARIABLE_NAME}",
		"Accept:application/json;charset=utf-8",
	} {
		assert.True(t, isValidHeader(header), header)
	}

	for _, header := range []string{"header1Key", "header1Key:", ":value", "", ":"} {
		assert.False(t, isValidHeader(header), header)
	}
}

func TestValidateNotificationHeaders(t *testing.T) {
	freshConfig(t)

	viper.Set(param.NotificationsAttacks, map[string]any{
		"endpoint": "https://example.test/hook",
		"headers":  []string{"Content-Type:application/json"},
	})
	assert.NoError(t, ValidateConfigs())

	viper.Set(param.NotificationsAttacks, map[string]any{"headers": []string{"no-separator"}})
	assert.EqualError(t, ValidateConfigs(), "Header: no-separator is not in valid format")
}

func TestValidateCustomResources(t *testing.T) {
	cnpg := map[string]string{
		"group":     "postgresql.cnpg.io",
		"version":   "v1",
		"resource":  "clusters",
		"pod_label": "cnpg.io/cluster",
	}

	t.Run("a resource the operator docs would describe", func(t *testing.T) {
		freshConfig(t)
		viper.Set(param.CustomResources, []map[string]string{cnpg})

		assert.NoError(t, ValidateConfigs())
	})

	// Pods can carry the identifier label instead, so the pod label is optional
	t.Run("without a pod label", func(t *testing.T) {
		freshConfig(t)
		viper.Set(param.CustomResources, []map[string]string{{
			"group": "postgresql.cnpg.io", "version": "v1", "resource": "clusters",
		}})

		assert.NoError(t, ValidateConfigs())
	})

	// A group is optional too, core resources do not have one
	t.Run("without a group", func(t *testing.T) {
		freshConfig(t)
		viper.Set(param.CustomResources, []map[string]string{{"version": "v1", "resource": "clusters"}})

		assert.NoError(t, ValidateConfigs())
	})

	for name, tc := range map[string]struct {
		resources   []map[string]string
		expectedErr string
	}{
		"no version": {
			resources:   []map[string]string{{"group": "postgresql.cnpg.io", "resource": "clusters"}},
			expectedErr: "CustomResources: " + param.CustomResources + " has an entry missing its version or its resource",
		},
		"no resource": {
			resources:   []map[string]string{{"group": "postgresql.cnpg.io", "version": "v1"}},
			expectedErr: "CustomResources: " + param.CustomResources + " has an entry missing its version or its resource",
		},
		// The kind rather than the plural resource name is the easy mistake,
		// and it would 404 against the apiserver
		"the kind instead of the resource name": {
			resources:   []map[string]string{{"group": "postgresql.cnpg.io", "version": "v1", "resource": "Cluster"}},
			expectedErr: `CustomResources: "Cluster" should be the lowercase plural resource name, e.g. "cluster" rather than the kind`,
		},
		// Two entries for one resource would schedule every victim twice
		"the same resource twice": {
			resources:   []map[string]string{cnpg, cnpg},
			expectedErr: "CustomResources: clusters.postgresql.cnpg.io is listed more than once",
		},
	} {
		t.Run(name, func(t *testing.T) {
			freshConfig(t)
			viper.Set(param.CustomResources, tc.resources)

			assert.EqualError(t, ValidateConfigs(), tc.expectedErr)
		})
	}

	t.Run("a pod label kubernetes would not accept", func(t *testing.T) {
		freshConfig(t)
		viper.Set(param.CustomResources, []map[string]string{{
			"version": "v1", "resource": "clusters", "pod_label": "not a label",
		}})

		err := ValidateConfigs()

		require.Error(t, err)
		assert.ErrorContains(t, err, "is not a valid label key")
	})
}
