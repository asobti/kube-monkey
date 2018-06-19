package schedule

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config/param"
	"github.com/asobti/kube-monkey/victims"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/asobti/kube-monkey/config"
)

func newSchedule() *Schedule {
	return &Schedule{}
}

func newChaos() *chaos.Chaos {
	return chaos.New(time.Now(), victims.NewVictimMock())
}

func TestEntries(t *testing.T) {
	s := newSchedule()
	assert.Equal(t, s.Entries(), s.entries)
	assert.Len(t, s.Entries(), 0)
}

func TestAdd(t *testing.T) {
	e := newChaos()
	s := newSchedule()

	s.Add(e)
	assert.Len(t, s.entries, 1)

}

func TestStringNoEntries(t *testing.T) {
	s := newSchedule()

	schedString := []string{}
	schedString = append(schedString, fmt.Sprint(TODAY))
	schedString = append(schedString, fmt.Sprint(NO_TERMINATION))
	schedString = append(schedString, fmt.Sprint(END))

	assert.Equal(t, strings.Join(schedString, "\n"), s.String())
}

func TestStringWithEntries(t *testing.T) {
	s := newSchedule()
	e1 := newChaos()
	e2 := newChaos()
	s.Add(e1)
	s.Add(e2)

	schedString := []string{}
	schedString = append(schedString, fmt.Sprint(TODAY))
	schedString = append(schedString, fmt.Sprint(HEADER_ROW))
	schedString = append(schedString, fmt.Sprint(SEP_ROW))
	for _, chaos := range s.entries {
		schedString = append(schedString, fmt.Sprintf(ROW_FORMAT, chaos.Victim().Kind(), chaos.Victim().Name(), chaos.KillAt().Format(DATE_FORMAT)))
	}
	schedString = append(schedString, fmt.Sprint(END))

	assert.Equal(t, strings.Join(schedString, "\n"), s.String())
}

func TestCalculateKillTimeRandom(t *testing.T) {
	config.SetDefaults()
	killtime := CalculateKillTime()

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
	killtime := CalculateKillTime()

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
