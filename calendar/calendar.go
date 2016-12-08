package calendar

import (
	"fmt"
	"math/rand"
	"time"
)

// Checks if specified Time is a weekday
func isWeekday(t time.Time) bool {
	switch t.Weekday() {
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday:
		return true
	case time.Saturday, time.Sunday:
		return false
	}

	panic(fmt.Sprintf("Unrecognized day of the week: %s", t.Weekday().String()))
}

// Returns the next weekday in Location
func nextWeekday(loc *time.Location) time.Time {
	check := time.Now().In(loc)
	for {
		check = check.AddDate(0, 0, 1)
		if isWeekday(check) {
			return check
		}
	}
}

// Calculates the next time the Scheduled should run
func NextRuntime(loc *time.Location, r int) time.Time {
	now := time.Now().In(loc)

	// Is today a weekday and are we still in time for it?
	if isWeekday(now) {
		runtimeToday := time.Date(now.Year(), now.Month(), now.Day(), r, 0, 0, 0, loc)
		if runtimeToday.After(now) {
			return runtimeToday
		}
	}

	// Missed the train for today. Schedule on next weekday
	year, month, day := nextWeekday(loc).Date()
	return time.Date(year, month, day, r, 0, 0, 0, loc)
}

// Returns a random time within the range specified by startHour and endHour
func RandomTimeInRange(startHour int, endHour int, location *time.Location) time.Time {
	// calculate the number of minutes in the range
	minutesInRange := (endHour - startHour) * 60

	// calculate a random minute-offset in range [0, minutesInRange)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randMinuteOffset := r.Intn(minutesInRange)
	offsetDuration := time.Duration(randMinuteOffset) * time.Minute

	// Add the minute offset to the start of the range to get a random
	// time within the range
	year, month, date := time.Now().Date()
	rangeStart := time.Date(year, month, date, startHour, 0, 0, 0, location)
	return rangeStart.Add(offsetDuration)
}
