package kubemonkey

import (
	"testing"
	"time"

	"github.com/asobti/kube-monkey/calendar"
	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/config/param"
	"github.com/bouk/monkey"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDurationToNextRun(t *testing.T) {
	debugScheduleDelay := 5
	config.SetDefaults()
	viper.Set(param.DebugEnabled, true)
	viper.Set(param.DebugScheduleDelay, debugScheduleDelay)
	loc := time.UTC
	now := time.Date(2018, 4, 16, 12, 0, 0, 0, time.UTC)
	runHour := 3

	assert.Equalf(t, time.Duration(debugScheduleDelay)*time.Second, durationToNextRun(1, loc), "Expected to get the next schedule when debug mode is enabled")

	viper.Set(param.DebugEnabled, false)

	monkey.Patch(time.Now, func() time.Time {
		return now
	})
	defer monkey.Unpatch(time.Now)

	nextRun := calendar.NextRuntime(loc, runHour)
	assert.Equal(t, nextRun.Sub(time.Now()), durationToNextRun(runHour, loc))

}

func TestScheduleTerminations(t *testing.T) {
	entries := make([]chaos.ChaosIntf, 1)

	chaos := chaos.NewMock()
	chaos.On("Schedule", mock.Anything)

	entries[0] = chaos
	ScheduleTerminations(entries)
	chaos.AssertExpectations(t)
}
