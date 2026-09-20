package schedule

import (
	"strings"
	"testing"
	"time"

	"kube-monkey/internal/pkg/chaos"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/config/param"
	"kube-monkey/internal/pkg/victims"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	config.SetDefaults()
	m.Run()
}

// resetConfig puts the defaults back after a test has changed them, so the
// tests do not have to run in any particular order
func resetConfig(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		viper.Reset()
		config.SetDefaults()
	})
}

// wholeDayRange makes the kill range cover the rest of the day, so tests do not
// depend on the time of day they run at
func wholeDayRange() {
	viper.Set(param.StartHour, 0)
	viper.Set(param.EndHour, 24)
}

func newEntry(kind, namespace, name string, killAt time.Time) *chaos.Chaos {
	return chaos.New(killAt, victims.New(victims.Spec{
		Kind:      kind,
		Name:      name,
		Namespace: namespace,
		Mtbf:      24 * time.Hour,
	}))
}

func TestEntriesStartsEmpty(t *testing.T) {
	assert.Empty(t, (&Schedule{}).Entries())
}

func TestAddKeepsTheOrderEntriesCameIn(t *testing.T) {
	first := newEntry("v1.Deployment", "shop", "checkout", time.Now())
	second := newEntry("v1.StatefulSet", "shop", "basket", time.Now())

	s := &Schedule{}
	s.Add(first)
	s.Add(second)

	assert.Equal(t, []*chaos.Chaos{first, second}, s.Entries())
}

func TestStringWithNoEntries(t *testing.T) {
	t.Setenv("KUBE_MONKEY_ID", "")

	printed := (&Schedule{}).String()

	assert.Contains(t, printed, "Today's schedule")
	assert.Contains(t, printed, "No terminations scheduled")
	assert.Contains(t, printed, "End of schedule")
	assert.NotContains(t, printed, "KubeMonkey ID")
}

func TestStringNamesTheClusterWhenItHasAnID(t *testing.T) {
	t.Setenv("KUBE_MONKEY_ID", "cluster-a")

	assert.Contains(t, (&Schedule{}).String(), "KubeMonkey ID: cluster-a")
}

func TestStringListsEveryEntry(t *testing.T) {
	t.Setenv("KUBE_MONKEY_ID", "")

	killAt := time.Date(2024, 3, 12, 14, 30, 0, 0, time.UTC)
	s := &Schedule{}
	s.Add(newEntry("v1.Deployment", "team-shop", "checkout", killAt))
	s.Add(newEntry("v1.StatefulSet", "team-shop", "basket", killAt.Add(time.Hour)))

	lines := strings.Split(s.String(), "\n")

	require.Len(t, lines, 6, "a header, a separator, two entries and the two banners")
	assert.Contains(t, lines[3], "v1.Deployment")
	assert.Contains(t, lines[3], "team-shop")
	assert.Contains(t, lines[3], "checkout")
	assert.Contains(t, lines[3], "03/12/2024 14:30:00")
	assert.Contains(t, lines[4], "basket")
	assert.Contains(t, lines[4], "03/12/2024 15:30:00")
	assert.NotContains(t, s.String(), "No terminations scheduled")
}

func TestCalculateKillTimesStaysInTheConfiguredZone(t *testing.T) {
	resetConfig(t)
	wholeDayRange()
	viper.Set(param.Timezone, "UTC")

	// One kill a minute on average, so there is always more than one left today
	killtimes := CalculateKillTimes(time.Minute)

	assert.NotEmpty(t, killtimes)
	for _, killtime := range killtimes {
		assert.Equal(t, time.UTC, killtime.Location())
		assert.False(t, killtime.Before(time.Now().Add(-time.Second)), "kill times should not be in the past")
	}
}

// A victim that dies once every two centuries is not expected to die today
func TestCalculateKillTimesForAnMtbfLongerThanTheDay(t *testing.T) {
	resetConfig(t)
	wholeDayRange()

	assert.Empty(t, CalculateKillTimes(200*365*24*time.Hour))
}

func TestCalculateKillTimesInDebugImmediateMode(t *testing.T) {
	resetConfig(t)
	viper.Set(param.DebugEnabled, true)
	viper.Set(param.DebugScheduleImmediateKill, true)

	killtimes := CalculateKillTimes(24 * time.Hour)

	require.Len(t, killtimes, 1, "immediate mode always schedules exactly one kill")
	assert.Equal(t, config.Timezone(), killtimes[0].Location())
	assert.WithinDuration(t, time.Now(), killtimes[0], time.Minute)
}

// The mtbf asks for no kill today, but the debug flag forces one anyway
func TestCalculateKillTimesWhenDebugForcesAKill(t *testing.T) {
	resetConfig(t)
	wholeDayRange()
	viper.Set(param.DebugEnabled, true)
	viper.Set(param.DebugForceShouldKill, true)

	killtimes := CalculateKillTimes(200 * 365 * 24 * time.Hour)

	require.Len(t, killtimes, 1)
	assert.False(t, killtimes[0].Before(time.Now().Add(-time.Second)))
}

// Forcing a kill cannot conjure one up once the day's range has passed
func TestCalculateKillTimesWhenDebugForcesAKillTooLate(t *testing.T) {
	resetConfig(t)
	viper.Set(param.Timezone, "UTC")
	viper.Set(param.StartHour, 0)
	viper.Set(param.EndHour, 1)
	viper.Set(param.DebugEnabled, true)
	viper.Set(param.DebugForceShouldKill, true)

	if time.Now().UTC().Hour() < 1 {
		t.Skip("the range has not passed yet")
	}

	assert.Empty(t, CalculateKillTimes(200*365*24*time.Hour))
}
