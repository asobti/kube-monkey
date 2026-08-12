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

// wholeDayRange makes the kill range cover the rest of the day, so tests do not
// depend on the time of day they run at
func wholeDayRange() {
	viper.SetDefault(param.StartHour, 0)
	viper.SetDefault(param.EndHour, 24)
}

func TestCalculateKillTimesRandom(t *testing.T) {
	config.SetDefaults()
	wholeDayRange()
	// One kill a minute on average, so there is always more than one left today
	killtimes := CalculateKillTimes(time.Minute)

	assert.NotEmpty(t, killtimes)
	for _, killtime := range killtimes {
		assert.Equal(t, config.Timezone(), killtime.Location())
		assert.True(t, killtime.After(time.Now().Add(-time.Second)))
	}
	config.SetDefaults()
}

func TestCalculateKillTimesForLongMtbf(t *testing.T) {
	config.SetDefaults()
	wholeDayRange()
	// A victim that dies once every two centuries is not expected to die today
	assert.Empty(t, CalculateKillTimes(time.Duration(200*365)*24*time.Hour))
	config.SetDefaults()
}

func TestCalculateKillTimesNow(t *testing.T) {
	config.SetDefaults()
	viper.SetDefault(param.DebugEnabled, true)
	viper.SetDefault(param.DebugScheduleImmediateKill, true)
	killtimes := CalculateKillTimes(24 * time.Hour)

	assert.Len(t, killtimes, 1)
	assert.Equal(t, config.Timezone(), killtimes[0].Location())
	assert.WithinDuration(t, killtimes[0], time.Now(), time.Second*time.Duration(60))
	config.SetDefaults()
}

func TestCalculateKillTimesForced(t *testing.T) {
	config.SetDefaults()
	wholeDayRange()
	viper.SetDefault(param.DebugEnabled, true)
	viper.SetDefault(param.DebugForceShouldKill, true)
	// The mtbf asks for no kill today, but the debug flag forces one anyway
	killtimes := CalculateKillTimes(time.Duration(200*365) * 24 * time.Hour)

	assert.Len(t, killtimes, 1)
	assert.True(t, killtimes[0].After(time.Now().Add(-time.Second)))
	config.SetDefaults()
}
