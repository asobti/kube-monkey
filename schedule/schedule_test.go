package schedule

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config/param"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/asobti/kube-monkey/config"
)

func newSchedule() *Schedule {
	return &Schedule{}
}

func TestEntries(t *testing.T) {
	s := newSchedule()
	assert.Equal(t, s.Entries(), s.entries)
	assert.Len(t, s.Entries(), 0)
}

func TestAdd(t *testing.T) {
	e := chaos.NewMock()
	s := newSchedule()

	s.Add(e)
	assert.Len(t, s.entries, 1)

}

func TestStringNoEntries(t *testing.T) {
	s := newSchedule()

	schedString := []string{}
	schedString = append(schedString, fmt.Sprint(Today))
	schedString = append(schedString, fmt.Sprint(NoTermination))
	schedString = append(schedString, fmt.Sprint(End))

	assert.Equal(t, strings.Join(schedString, "\n"), s.String())
}

func TestStringWithEntries(t *testing.T) {
	s := newSchedule()
	e1 := chaos.NewMock()
	e2 := chaos.NewMock()
	s.Add(e1)
	s.Add(e2)

	schedString := []string{}
	schedString = append(schedString, fmt.Sprint(Today))
	schedString = append(schedString, fmt.Sprint(HeaderRow))
	schedString = append(schedString, fmt.Sprint(SepRow))
	for _, chaos := range s.entries {
		schedString = append(schedString, fmt.Sprintf(RowFormat, chaos.Victim().Kind(), chaos.Victim().Namespace(), chaos.Victim().Name(), chaos.KillAt().Format(DateFormat)))
	}
	schedString = append(schedString, fmt.Sprint(End))

	assert.Equal(t, strings.Join(schedString, "\n"), s.String())
}

func TestCalculateKillTimeRandom(t *testing.T) {
	config.SetDefaults()
	killtime := CalculateKillTime("1h")

	scheduledTime := func() (success bool) {
		if killtime.Hour() >= config.StartHour() && killtime.Hour() <= config.EndHour() {
			success = true
		}
		return
	}

	assert.Equal(t, killtime.Location(), config.Timezone())
	assert.Condition(t, scheduledTime)

}

func TestCalculateKillTimeNow(t *testing.T) {
	config.SetDefaults()
	viper.SetDefault(param.DebugEnabled, true)
	viper.SetDefault(param.DebugScheduleImmediateKill, true)
	killtime := CalculateKillTime("1h")

	assert.Equal(t, killtime.Location(), config.Timezone())
	assert.WithinDuration(t, killtime, time.Now(), time.Second*time.Duration(60))
	config.SetDefaults()
}

func TestShouldScheduleChaosNow(t *testing.T) {
	config.SetDefaults()
	viper.SetDefault(param.DebugEnabled, true)
	viper.SetDefault(param.DebugForceShouldKill, true)
	assert.True(t, ShouldScheduleChaos(100000000000))
	config.SetDefaults()
}

func TestShouldScheduleChaosMtbf(t *testing.T) {
	assert.False(t, ShouldScheduleChaos(100000000000))
	assert.True(t, ShouldScheduleChaos(1))
}
