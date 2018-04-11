package config

import (
	"os"
	"testing"
	"time"

	"github.com/asobti/kube-monkey/config/param"
	"github.com/bouk/monkey"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigTestSuite struct {
	suite.Suite
}

func (s *ConfigTestSuite) SetupTest() {
	viper.Reset()
	SetDefaults()
}

func (s *ConfigTestSuite) TestSetDefaults() {

	s.True(viper.GetBool(param.DryRun))
	s.Equal("America/Los_Angeles", viper.GetString(param.Timezone))
	s.Equal(8, viper.GetInt(param.RunHour))
	s.Equal(10, viper.GetInt(param.StartHour))
	s.Equal(16, viper.GetInt(param.EndHour))
	s.Equal(int64(5), viper.GetInt64(param.GracePeriodSec))
	s.Equal([]string{metav1.NamespaceSystem}, viper.GetStringSlice(param.BlacklistedNamespaces))
	s.Equal([]string{metav1.NamespaceDefault}, viper.GetStringSlice(param.WhitelistedNamespaces))
	s.False(viper.GetBool(param.DebugEnabled))
	s.Equal(viper.GetInt(param.DebugScheduleDelay), 30)
	s.False(viper.GetBool(param.DebugForceShouldKill))
	s.False(viper.GetBool(param.DebugScheduleImmediateKill))

}

func (s *ConfigTestSuite) TestDryRun() {
	viper.Set(param.DryRun, false)
	s.False(DryRun())
	viper.Set(param.DryRun, true)
	s.True(DryRun())
}

func (s *ConfigTestSuite) TestTimezone() {
	viper.Set(param.Timezone, "nolnexistent")

	// avoid Exit(255) on glog.Fatal
	monkey.Patch(glog.Fatal, func(a ...interface{}) {
		s.Equal(a[0], "cannot find nolnexistent in zip file "+os.Getenv("GOROOT")+"/lib/time/zoneinfo.zip")
	})
	defer func() { monkey.Unpatch(glog.Fatal) }()
	s.Equal((*time.Location)(nil), Timezone())
	viper.Set(param.Timezone, "UTC")
	s.Equal(Timezone().String(), "UTC")
}

func (s *ConfigTestSuite) TestRunHour() {
	viper.Set(param.RunHour, 11)
	s.Equal(11, RunHour())
}

func (s *ConfigTestSuite) TestStartHour() {
	viper.Set(param.StartHour, 10)
	s.Equal(10, StartHour())
}

func (s *ConfigTestSuite) TestEndHour() {
	viper.Set(param.EndHour, 9)
	s.Equal(9, EndHour())
}

func (s *ConfigTestSuite) TestGracePeriodSeconds() {
	g := int64(100)
	viper.Set(param.GracePeriodSec, 100)
	s.Equal(&g, GracePeriodSeconds())
}

func (s *ConfigTestSuite) TestBlacklistedNamespaces() {
	blns := []string{"namespace1", "namespace2"}
	viper.Set(param.BlacklistedNamespaces, blns)
	ns := BlacklistedNamespaces()
	s.Len(ns, len(blns))
	for _, v := range blns {
		s.Contains(ns, v)
	}
}

func (s *ConfigTestSuite) TestWhitelistedNamespaces() {
	wlns := []string{"namespace1", "namespace2"}
	viper.Set(param.WhitelistedNamespaces, wlns)
	ns := WhitelistedNamespaces()
	s.Len(ns, len(wlns))
	for _, v := range wlns {
		s.Contains(ns, v)
	}
}

func (s *ConfigTestSuite) TestBlacklistEnabled() {
	s.True(BlacklistEnabled())
	viper.Set(param.BlacklistedNamespaces, []string{metav1.NamespaceNone})
	s.False(BlacklistEnabled())
}

func (s *ConfigTestSuite) TestWhitelistEnabled() {
	s.True(WhitelistEnabled())
	viper.Set(param.WhitelistedNamespaces, []string{metav1.NamespaceAll})
	s.False(WhitelistEnabled())
}

func (s *ConfigTestSuite) TestClusterrAPIServerHost() {
	host, enabled := ClusterAPIServerHost()
	s.False(enabled)
	s.Empty(host)
	viper.Set(param.ClusterAPIServerHost, "Host")
	host, enabled = ClusterAPIServerHost()
	s.True(enabled)
	s.Equal("Host", host)
}

func (s *ConfigTestSuite) TestDebugEnabled() {
	viper.Set(param.DebugEnabled, true)
	s.True(DebugEnabled())
}

func (s *ConfigTestSuite) TestDebugScheduleDelay() {
	viper.Set(param.DebugScheduleDelay, 10)
	s.Equal(10*time.Second, DebugScheduleDelay())
}
func (s *ConfigTestSuite) TestDebugForceShouldKill() {
	viper.Set(param.DebugForceShouldKill, true)
	s.True(DebugForceShouldKill())
}

func (s *ConfigTestSuite) TestDebugInmediateKill() {
	viper.Set(param.DebugScheduleImmediateKill, true)
	s.True(DebugScheduleImmediateKill())
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
