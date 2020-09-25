package config

import (
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/golang/glog"
	"github.com/spf13/viper"

	"github.com/asobti/kube-monkey/config/param"

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
	viper.SetDefault(param.NotificationsAttacks, Receiver{})
}

func setupWatch() {
	// TODO: This does not appear to be working
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		glog.V(4).Info("Config change detected")
		if err := ValidateConfigs(); err != nil {
			panic(err)
		}
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

func NotificationsAttacks() Receiver {
	var receiver Receiver
	err := viper.UnmarshalKey(param.NotificationsAttacks, &receiver)
	if err != nil {
		glog.Errorf("Failed to parse notifications.attacks %v", err)
	}
	return receiver
}
