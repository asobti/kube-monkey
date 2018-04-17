package calendar

import (
	"testing"
	"time"

	"github.com/bouk/monkey"
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

func TestNextWeekDay(t *testing.T) {
	var today, next time.Time
	defer monkey.Unpatch(time.Now)

	for i := 16; i < 23; i++ {
		today = time.Date(2018, 4, i, 0, 0, 0, 0, time.UTC)

		switch today.Weekday() {
		case time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Sunday:
			next = today.AddDate(0, 0, 1)
		case time.Saturday:
			next = today.AddDate(0, 0, 2)
		case time.Friday:
			next = today.AddDate(0, 0, 3)
		}

		monkey.Patch(time.Now, func() time.Time {
			return today
		})

		assert.Equal(t, nextWeekday(time.UTC), next)
	}
}

func TestNextRuntime(t *testing.T) {

	monkey.Patch(time.Now, func() time.Time {
		return time.Date(2018, 4, 16, 12, 0, 0, 0, time.UTC)
	})
	defer monkey.Unpatch(time.Now)

	r := 13
	monday := time.Date(2018, 4, 16, r, 0, 0, 0, time.UTC)
	assert.Equalf(t, NextRuntime(time.UTC, r), monday, "Expected to be run today if today is a weekday and there is time for it")

	r = 1
	assert.Equalf(t, NextRuntime(time.UTC, r), monday.Add(time.Hour*12), "Expected to be run next weekday if today is a weekday and there is not time for it")

	sunday := time.Date(2018, 4, 15, 0, 0, 0, 0, time.UTC)
	monkey.Patch(time.Now, func() time.Time {
		return sunday
	})

	assert.Equalf(t, NextRuntime(time.UTC, 0), sunday.Add(time.Hour*24), "Expected to be run next weekday if today is a weekend day")
}

func TestRandomTimeInRange(t *testing.T) {
	loc := time.UTC

	monkey.Patch(time.Now, func() time.Time {
		return time.Date(2018, 4, 16, 12, 0, 0, 0, time.UTC)
	})
	defer monkey.Unpatch(time.Now)

	randomTime := RandomTimeInRange(10, 12, loc)

	scheduledTime := func() (success bool) {
		if randomTime.Hour() >= 10 && randomTime.Hour() <= 12 {
			success = true
		}
		return
	}

	assert.Condition(t, scheduledTime)
}
