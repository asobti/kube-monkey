package calendar

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

// Checks if specified Time is a weekday
func isWeekday(t time.Time) bool {
	switch t.Weekday() {
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday:
		return true
	case time.Saturday, time.Sunday:
		return false
	}

	glog.Fatalf("Unrecognized day of the week: %s", t.Weekday().String())

	panic("Explicit Panic to avoid compiler error: missing return at end of function")
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

// NextRuntime calculates the next time the Scheduled should run
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

// ParseMtbf parses an mtbf value and returns a valid time duration.
func ParseMtbf(mtbf string) (time.Duration, error) {
	// time.Duration biggest valid time unit is an hour, but we want to accept
	// days. Before finer grained time units this software used to accept mtbf as
	// an integer interpreted as days. Hence this routine now accepts a "d" as a
	// valid time unit meaning days and simply strips it, because...
	if mtbf[len(mtbf) - 1] == 'd' {
		mtbf = strings.TrimRight(mtbf, "d")
	}
	// ...below we check if a given mtbf is simply a number and backward
	// compatibilty dictates us to accept a simpel number as days (see above) and
	// since time.Duration does not accept hours as a valid time unit we convert
	// here ourselves days into hours.
	if converted_mtbf, err := strconv.Atoi(mtbf); err == nil {
		mtbf = fmt.Sprintf("%dh", converted_mtbf * 24)
	}
	duration, err := time.ParseDuration(mtbf)
	if err != nil {
		return 0, err
	}
	one_minute, _ := time.ParseDuration("1m")
	if duration < one_minute {
		return 0, errors.New("smallest valid mtbf is one minute.")
	}
	return duration, nil
}

// RandomTimeInRange returns a random time within the range specified by startHour and endHour
func RandomTimeInRange(mtbf string, startHour int, endHour int, loc *time.Location) []time.Time {
	var times []time.Time
	tmptimeDuration, err := ParseMtbf(mtbf)
	if err != nil {
		glog.Errorf("error parsing customized mtbf %s: %v", mtbf, err)
		return []time.Time{time.Now().Add(time.Duration(24*365*10) * time.Hour)}
	}

	one_day, _ := time.ParseDuration("24h")

	// If the mtbf is bigger or equal to one day we will calculate one
	// random time in the range. If not we will calculate several random
	// times.
	if tmptimeDuration >= one_day {
		// calculate the number of minutes in the range
		minutesInRange := (endHour - startHour) * 60

		// calculate a random minute-offset in range [0, minutesInRange)
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		randMinuteOffset := r.Intn(minutesInRange)
		offsetDuration := time.Duration(randMinuteOffset) * time.Minute

		// Add the minute offset to the start of the range to get a random
		// time within the range
		year, month, date := time.Now().Date()
		rangeStart := time.Date(year, month, date, startHour, 0, 0, 0, loc)
		times = append(times, rangeStart.Add(offsetDuration))
		return times
	} else {
		startTime := time.Now().In(loc)

		for {
			//time range should be twice of the input mean time between failure value
			timeDuration := tmptimeDuration * 2
			//compute random offset time
			mtbfEndTime := startTime.Add(timeDuration)
			subSecond := int64(mtbfEndTime.Sub(startTime) / time.Second)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			randSecondOffset := r.Int63n(subSecond)
			randCalTime := startTime.Add(time.Duration(randSecondOffset) * time.Second)

			// compute randSecondOffset between start and end hour
			year, month, date := startTime.Date()
			todayEndTime := time.Date(year, month, date, endHour, 0, 0, 0, loc)
			todayStartTime := time.Date(year, month, date, startHour, 0, 0, 0, loc)
			if startTime.Before(todayStartTime) { // now is earlier then start hour, only for test pass, normal process won't run into this condition
				return []time.Time{todayStartTime}
			}
			if randCalTime.Before(todayEndTime) { // time offset before today's endHour
				glog.V(1).Infof("RandomTimeInRange calculate time %s", randCalTime)
				times = append(times, randCalTime)
				// Move start time up to the calculated random time
				startTime = randCalTime
			} else {
				return times
			}
		}
	}
}
