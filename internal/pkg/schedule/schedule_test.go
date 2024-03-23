package schedule

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"kube-monkey/internal/pkg/chaos"
	"kube-monkey/internal/pkg/config/param"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"kube-monkey/internal/pkg/config"
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

func TestStringNoEntriesWithID(t *testing.T) {

	id := "TestingID"
	os.Setenv("KUBE_MONKEY_ID", id)

	s := newSchedule()

	schedString := []string{}
	schedString = append(schedString, fmt.Sprint(Today))
	schedString = append(schedString, fmt.Sprintf(KubeMonkeyID, id))

	schedString = append(schedString, fmt.Sprint(NoTermination))
	schedString = append(schedString, fmt.Sprint(End))

	assert.Equal(t, strings.Join(schedString, "\n"), s.String())

	os.Unsetenv("KUBE_MONKEY_ID")
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
	killtimes := CalculateKillTimes("1h")

	scheduledTime := func() (success bool) {
		if killtimes[0].Hour() >= config.StartHour() && killtimes[0].Hour() <= config.EndHour() {
			success = true
		}
		return
	}

	assert.Equal(t, killtimes[0].Location(), config.Timezone())
	assert.Condition(t, scheduledTime)

}

func TestCalculateKillTimeNow(t *testing.T) {
	config.SetDefaults()
	viper.SetDefault(param.DebugEnabled, true)
	viper.SetDefault(param.DebugScheduleImmediateKill, true)
	killtimes := CalculateKillTimes("1h")

	assert.Equal(t, killtimes[0].Location(), config.Timezone())
	assert.WithinDuration(t, killtimes[0], time.Now(), time.Second*time.Duration(60))
	config.SetDefaults()
}
