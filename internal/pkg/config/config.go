package config

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/spf13/viper"

	"kube-monkey/internal/pkg/calendar"
	"kube-monkey/internal/pkg/config/param"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	configpath = "/etc/kube-monkey"
	configtype = "toml"
	configname = "config"

	// Currently, there does not appear to be
	// any value in making these configurable
	// so defining them as consts

	IdentLabelKey                 = "kube-monkey/identifier"
	EnabledLabelKey               = "kube-monkey/enabled"
	EnabledLabelValue             = "enabled"
	MtbfLabelKey                  = "kube-monkey/mtbf"
	KillTypeLabelKey              = "kube-monkey/kill-mode"
	KillValueLabelKey             = "kube-monkey/kill-value"
	KillRandomMaxLabelValue       = "random-max-percent"
	KillFixedPercentageLabelValue = "fixed-percent"
	KillFixedLabelValue           = "fixed"
	KillAllLabelValue             = "kill-all"
)

type Receiver struct {
	Endpoint string   `mapstructure:"endpoint"`
	Message  string   `mapstructure:"message"`
	Headers  []string `mapstructure:"headers"`
}

// NewReceiver creates a new Receiver instance
func NewReceiver(endpoint string, message string, headers []string) Receiver {
	return Receiver{
		Endpoint: endpoint,
		Message:  message,
		Headers:  headers,
	}
}

func SetDefaults() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault(param.DryRun, true)
	viper.SetDefault(param.Timezone, "America/Los_Angeles")
	viper.SetDefault(param.RunDays, []string{"mon", "tue", "wed", "thu", "fri"})
	viper.SetDefault(param.RunHour, 8)
	viper.SetDefault(param.StartHour, 10)
	viper.SetDefault(param.EndHour, 16)
	viper.SetDefault(param.GracePeriodSec, 5)
	viper.SetDefault(param.BlacklistedNamespaces, []string{metav1.NamespaceSystem})
	viper.SetDefault(param.WhitelistedNamespaces, []string{metav1.NamespaceAll})

	viper.SetDefault(param.DebugEnabled, false)
	viper.SetDefault(param.DebugScheduleDelay, 30)
	viper.SetDefault(param.DebugForceShouldKill, false)
	viper.SetDefault(param.DebugScheduleImmediateKill, false)

	viper.SetDefault(param.NotificationsEnabled, false)
	viper.SetDefault(param.NotificationsProxy, nil)
	viper.SetDefault(param.NotificationsReportSchedule, false)
	viper.SetDefault(param.NotificationsAttacks, Receiver{})

	viper.SetDefault(param.MetricsEnabled, false)
	viper.SetDefault(param.MetricsAddress, ":8080")
}

func setupWatch() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		glog.V(4).Info("Config change detected")
		if err := ValidateConfigs(); err != nil {
			panic(err)
		}
		glog.V(4).Info("Successfully reloaded configs")
	})
}

func Init() error {
	SetDefaults()
	viper.AddConfigPath(configpath)
	viper.SetConfigType(configtype)
	viper.SetConfigName(configname)

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	if err := ValidateConfigs(); err != nil {
		glog.Errorf("Failed to validate %v", err)
		return err
	}
	glog.V(4).Info("Successfully validated configs")
	setupWatch()
	return nil
}

func DryRun() bool {
	return viper.GetBool(param.DryRun)
}

func Timezone() *time.Location {
	tz := viper.GetString(param.Timezone)
	location, err := time.LoadLocation(tz)
	if err != nil {
		glog.Fatal(err.Error())
	}
	return location
}

// RunDays lists the days of the week kube-monkey builds a schedule on
func RunDays() []time.Weekday {
	days, err := parseRunDays()
	if err != nil {
		// Unreachable: ValidateConfigs rejects a bad list at startup and on reload
		glog.Fatal(err.Error())
	}
	return days
}

func parseRunDays() ([]time.Weekday, error) {
	values := viper.GetStringSlice(param.RunDays)
	if len(values) == 0 {
		return nil, fmt.Errorf("RunDays: %s must list at least one day", param.RunDays)
	}

	days := make([]time.Weekday, 0, len(values))
	for _, value := range values {
		day, err := calendar.ParseWeekday(value)
		if err != nil {
			return nil, fmt.Errorf("RunDays: %s contains an %v", param.RunDays, err)
		}
		days = append(days, day)
	}
	return days, nil
}

func RunHour() int {
	return viper.GetInt(param.RunHour)
}

func StartHour() int {
	return viper.GetInt(param.StartHour)
}

func EndHour() int {
	return viper.GetInt(param.EndHour)
}

func GracePeriodSeconds() *int64 {
	gpInt64 := viper.GetInt64(param.GracePeriodSec)
	return &gpInt64
}

func BlacklistedNamespaces() sets.String {
	// Return as set for O(1) membership checks
	namespaces := viper.GetStringSlice(param.BlacklistedNamespaces)
	return sets.NewString(namespaces...)
}

// IsBlacklistedNamespace reports whether a namespace is covered by the blacklist.
//
// Entries are shell-style patterns, so "team-*" covers every namespace with
// that prefix. A namespace name can only hold lowercase letters, digits and
// "-", so it never contains a wildcard character, and a plain entry still
// matches nothing but itself.
func IsBlacklistedNamespace(namespace string) bool {
	if !BlacklistEnabled() {
		return false
	}

	matched, invalid := matchesNamespace(BlacklistedNamespaces().UnsortedList(), namespace)

	// A pattern that does not parse cannot be shown to leave the namespace
	// alone, so block it rather than guess
	return matched || invalid
}

// IsWhitelistedNamespace reports whether a namespace is covered by the
// whitelist. Entries are shell-style patterns, in the same form the blacklist
// takes.
//
// The whitelist is off when it holds nothing but an empty entry, which is the
// default and allows every namespace. An empty entry sitting alongside real
// entries matches nothing, because a namespace name is never empty.
func IsWhitelistedNamespace(namespace string) bool {
	if !WhitelistEnabled() {
		return true
	}

	matched, invalid := matchesNamespace(WhitelistedNamespaces().UnsortedList(), namespace)

	// A list holding a pattern that does not parse no longer describes what the
	// operator meant, so grant nothing rather than act on the half of it that
	// still reads
	return matched && !invalid
}

// matchesNamespace reports whether the namespace matches any of the patterns,
// and separately whether any pattern was malformed. Patterns are checked when
// the config loads, so a malformed one means validation was skipped, and each
// caller picks the side it is safe to fail on.
//
// Every pattern is read even once a match is found, so invalid always covers
// the whole list.
func matchesNamespace(patterns []string, namespace string) (matched bool, invalid bool) {
	for _, pattern := range patterns {
		ok, err := path.Match(pattern, namespace)
		if err != nil {
			glog.Warningf("Ignoring namespace pattern %q because it is invalid: %v", pattern, err)
			invalid = true
			continue
		}
		matched = matched || ok
	}

	return matched, invalid
}

func WhitelistedNamespaces() sets.String {
	// Return as set for O(1) membership checks
	namespaces := viper.GetStringSlice(param.WhitelistedNamespaces)
	return sets.NewString(namespaces...)
}

func BlacklistEnabled() bool {
	return !BlacklistedNamespaces().Equal(sets.NewString(metav1.NamespaceNone))
}

func WhitelistEnabled() bool {
	return !WhitelistedNamespaces().Equal(sets.NewString(metav1.NamespaceAll))
}

func ClusterAPIServerHost() (string, bool) {
	if viper.IsSet(param.ClusterAPIServerHost) {
		return viper.GetString(param.ClusterAPIServerHost), true
	}
	return "", false
}

func DebugEnabled() bool {
	return viper.GetBool(param.DebugEnabled)
}

func DebugScheduleDelay() time.Duration {
	delaySec := viper.GetInt(param.DebugScheduleDelay)
	return time.Duration(delaySec) * time.Second
}

func DebugForceShouldKill() bool {
	return viper.GetBool(param.DebugForceShouldKill)
}

func DebugScheduleImmediateKill() bool {
	return viper.GetBool(param.DebugScheduleImmediateKill)
}

func NotificationsEnabled() bool {
	return viper.GetBool(param.NotificationsEnabled)
}

func NotificationsProxy() string {
	return viper.GetString(param.NotificationsProxy)
}

func NotificationsReportSchedule() bool {
	return viper.GetBool(param.NotificationsReportSchedule)
}

func NotificationsAttacks() Receiver {
	var receiver Receiver
	err := viper.UnmarshalKey(param.NotificationsAttacks, &receiver)
	if err != nil {
		glog.Errorf("Failed to parse notifications.attacks %v", err)
	}
	return receiver
}

func MetricsEnabled() bool {
	return viper.GetBool(param.MetricsEnabled)
}

func MetricsAddress() string {
	return viper.GetString(param.MetricsAddress)
}
