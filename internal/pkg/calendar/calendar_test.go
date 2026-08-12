package calendar

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsWeekDay(t *testing.T) {
	monday := time.Date(2018, 4, 16, 0, 0, 0, 0, time.UTC)

	assert.True(t, isWeekday(monday))
	assert.True(t, isWeekday(monday.Add(time.Hour*24)))
	assert.True(t, isWeekday(monday.Add(time.Hour*24*2)))
	assert.True(t, isWeekday(monday.Add(time.Hour*24*3)))
	assert.True(t, isWeekday(monday.Add(time.Hour*24*4)))

	assert.False(t, isWeekday(monday.Add(time.Hour*24*5)))
	assert.False(t, isWeekday(monday.Add(time.Hour*24*6)))
}

func TestParseMtbf(t *testing.T) {
	for value, expected := range map[string]time.Duration{
		"1":   24 * time.Hour,
		"3":   72 * time.Hour,
		"1d":  24 * time.Hour,
		"10d": 240 * time.Hour,
		"1h":  time.Hour,
		"36h": 36 * time.Hour,
		"1m":  time.Minute,
		"90m": 90 * time.Minute,
	} {
		mtbf, err := ParseMtbf(value)

		assert.NoError(t, err, value)
		assert.Equal(t, expected, mtbf, value)
	}
}

func TestParseMtbfInvalid(t *testing.T) {
	// Seconds are deliberately not a unit: a minute is as often as kube-monkey
	// is willing to kill
	for _, value := range []string{"", " ", "0", "0h", "-1", "1.5h", "30s", "1h30m", "3D", "day", "1 d", "3dd", "999999999999999999999d"} {
		_, err := ParseMtbf(value)

		assert.Error(t, err, value)
	}
}

func TestKillTimesInRangeCount(t *testing.T) {
	loc := time.UTC
	// Before the range opens, so the whole 6 hour range is still available
	now := time.Date(2024, 3, 12, 8, 0, 0, 0, loc)

	// A victim dies 24h/mtbf times a day, whatever the length of the range
	for mtbf, expected := range map[time.Duration]int{
		24 * time.Hour:   1,
		12 * time.Hour:   2,
		2 * time.Hour:    12,
		30 * time.Minute: 48,
	} {
		killtimes := killTimesInRange(now, testRand(), mtbf, 10, 16, loc)

		assert.Len(t, killtimes, expected, mtbf.String())
	}
}

func TestKillTimesInRangeAreInsideTheRange(t *testing.T) {
	loc := time.UTC
	now := time.Date(2024, 3, 12, 8, 0, 0, 0, loc)
	rangeStart := time.Date(2024, 3, 12, 10, 0, 0, 0, loc)
	rangeEnd := time.Date(2024, 3, 12, 16, 0, 0, 0, loc)

	killtimes := killTimesInRange(now, testRand(), 10*time.Minute, 10, 16, loc)

	assert.NotEmpty(t, killtimes)
	for i, killtime := range killtimes {
		assert.False(t, killtime.Before(rangeStart))
		assert.True(t, killtime.Before(rangeEnd))
		if i > 0 {
			assert.False(t, killtime.Before(killtimes[i-1]), "kill times should be in order")
		}
	}
}

func TestKillTimesInRangeSharesWhatIsLeftOfTheRange(t *testing.T) {
	loc := time.UTC
	// Half way through a 6 hour range, so only half of the kills are still due
	now := time.Date(2024, 3, 12, 13, 0, 0, 0, loc)

	killtimes := killTimesInRange(now, testRand(), time.Hour, 10, 16, loc)

	assert.Len(t, killtimes, 12)
	for _, killtime := range killtimes {
		assert.False(t, killtime.Before(now))
	}
}

func TestKillTimesInRangeNothingLeftToSchedule(t *testing.T) {
	loc := time.UTC

	// The range has already passed
	now := time.Date(2024, 3, 12, 20, 0, 0, 0, loc)
	assert.Empty(t, killTimesInRange(now, testRand(), time.Hour, 10, 16, loc))

	// The range is not a range at all
	now = time.Date(2024, 3, 12, 8, 0, 0, 0, loc)
	assert.Empty(t, killTimesInRange(now, testRand(), time.Hour, 16, 16, loc))
	assert.Empty(t, killTimesInRange(now, testRand(), time.Hour, 16, 10, loc))

	// No mtbf to go by
	assert.Empty(t, killTimesInRange(now, testRand(), 0, 10, 16, loc))
}

// An mtbf longer than a day means a kill on some days only. The odds of a kill
// on any single day are 24h/mtbf, which is how the mtbf label behaved when days
// were the only unit it took.
func TestKillTimesInRangeOddsForMtbfLongerThanADay(t *testing.T) {
	loc := time.UTC
	now := time.Date(2024, 3, 12, 8, 0, 0, 0, loc)
	r := testRand()

	for mtbf, expected := range map[time.Duration]float64{
		48 * time.Hour:  0.5,
		72 * time.Hour:  1.0 / 3.0,
		240 * time.Hour: 0.1,
	} {
		const days = 20000
		kills := 0
		for day := 0; day < days; day++ {
			kills += len(killTimesInRange(now, r, mtbf, 10, 16, loc))
		}

		assert.InDelta(t, expected, float64(kills)/days, 0.02, mtbf.String())
	}
}

func testRand() *rand.Rand {
	return rand.New(rand.NewSource(1))
}
