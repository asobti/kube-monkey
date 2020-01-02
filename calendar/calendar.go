package calendar

import (
	"math/rand"
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

// RandomTimeInRange returns a random time within the range specified by startHour and endHour
func RandomTimeInRange(startHour int, endHour int, loc *time.Location) time.Time {
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
	return rangeStart.Add(offsetDuration)
}

func CustzRandomTimeInRange(mtbf string, startHour, endHour int, loc *time.Location) time.Time {
	tmptimeDuration, err := time.ParseDuration(mtbf)
	if err != nil {
		glog.Errorf("error parsing customized mtbf %s: %v", mtbf, err)
		return time.Now().Add(time.Duration(24*365*10) * time.Hour)
	}
	//time range should be twice of the input mean time between failure value
	timeDuration := tmptimeDuration * 2
	//compute random offset time
	now := time.Now().In(loc)
	mtbfEndTime := now.Add(timeDuration)
	subSecond := int64(mtbfEndTime.Sub(now) / time.Second)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randSecondOffset := r.Int63n(subSecond)
	randCalTime := now.Add(time.Duration(randSecondOffset) * time.Second)

	// compute randSecondOffset between start and end hour
	year, month, date := now.Date()
	todayEndTime := time.Date(year, month, date, endHour, 0, 0, 0, loc)
	todayStartTime := time.Date(year, month, date, startHour, 0, 0, 0, loc)
	if randCalTime.Before(todayEndTime) { // time offset before today's endHour
		glog.V(1).Infof("CustzRandomTimeInRange calculate time %s", randCalTime)
		return randCalTime
	} else {
		leftOffset := randSecondOffset - int64(todayEndTime.Sub(now)/time.Second)
		offsetDay := leftOffset/(int64(endHour-startHour)*60*60) + 1
		modOffsetSecond := leftOffset % (int64(endHour-startHour) * 60 * 60)
		glog.V(1).Infof("CustzRandomTimeInRange calculate time %s", todayStartTime.Add(time.Duration(offsetDay*24)*time.Hour).Add(time.Duration(modOffsetSecond)*time.Second))
		return todayStartTime.Add(time.Duration(offsetDay*24) * time.Hour).Add(time.Duration(modOffsetSecond) * time.Second)
	}
}
