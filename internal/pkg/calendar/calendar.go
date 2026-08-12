package calendar

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
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

// mtbfPattern matches a whole number with an optional unit, e.g. 3, 3d, 4h or 20m
var mtbfPattern = regexp.MustCompile(`^([0-9]+)([dhm]?)$`)

// ParseMtbf turns the value of the kube-monkey/mtbf label into a duration.
// The unit is one of d (days), h (hours) or m (minutes). A value without a unit
// is read as days, which is what the label meant before smaller units existed.
func ParseMtbf(value string) (time.Duration, error) {
	match := mtbfPattern.FindStringSubmatch(value)
	if match == nil {
		return 0, fmt.Errorf("invalid mtbf %q: expected a whole number optionally followed by d, h or m", value)
	}

	amount, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("invalid mtbf %q: %v", value, err)
	}
	if amount == 0 {
		return 0, fmt.Errorf("invalid mtbf %q: must be greater than zero", value)
	}

	var unit time.Duration
	switch match[2] {
	case "m":
		unit = time.Minute
	case "h":
		unit = time.Hour
	default:
		unit = 24 * time.Hour
	}

	mtbf := time.Duration(amount) * unit
	if mtbf/unit != time.Duration(amount) {
		return 0, fmt.Errorf("invalid mtbf %q: too large", value)
	}

	return mtbf, nil
}

// KillTimesInRange returns the times at which a victim with the given mtbf
// should be killed today, at random within the range specified by startHour and
// endHour. A victim dies 24h/mtbf times a day on average, so an mtbf shorter
// than a day gives several kills and a longer one gives a kill only on some
// days. When part of the range has already passed, only the share of the kills
// belonging to the remaining part is scheduled.
func KillTimesInRange(mtbf time.Duration, startHour int, endHour int, loc *time.Location) []time.Time {
	return killTimesInRange(time.Now(), newRand(), mtbf, startHour, endHour, loc)
}

// RandomTimeInRange returns a single random time within the range specified by
// startHour and endHour. It reports false when the range has already passed.
func RandomTimeInRange(startHour int, endHour int, loc *time.Location) (time.Time, bool) {
	return randomTimeInRange(time.Now(), newRand(), startHour, endHour, loc)
}

func killTimesInRange(now time.Time, r *rand.Rand, mtbf time.Duration, startHour int, endHour int, loc *time.Location) []time.Time {
	start, remaining, full := rangeToday(now, startHour, endHour, loc)
	if remaining == 0 || mtbf <= 0 {
		return nil
	}

	// Kills are spread over the day but only ever happen inside the range, so
	// the share of the range still ahead of us is the share of the day's kills
	// left to schedule.
	expected := (float64(24*time.Hour) / float64(mtbf)) * (float64(remaining) / float64(full))

	// The whole part is a kill we owe for sure, the fraction left over is the
	// chance of one more.
	count := int(expected)
	if r.Float64() < expected-float64(count) {
		count++
	}

	killtimes := make([]time.Time, 0, count)
	for i := 0; i < count; i++ {
		killtimes = append(killtimes, start.Add(time.Duration(r.Int63n(int64(remaining)))))
	}
	sort.Slice(killtimes, func(i, j int) bool { return killtimes[i].Before(killtimes[j]) })

	return killtimes
}

func randomTimeInRange(now time.Time, r *rand.Rand, startHour int, endHour int, loc *time.Location) (time.Time, bool) {
	start, remaining, _ := rangeToday(now, startHour, endHour, loc)
	if remaining == 0 {
		return time.Time{}, false
	}

	return start.Add(time.Duration(r.Int63n(int64(remaining)))), true
}

// rangeToday returns the part of today's range that is still ahead of now, plus
// the length of the whole range. The remaining length is zero when the range
// has passed or when startHour and endHour do not describe a real range.
func rangeToday(now time.Time, startHour int, endHour int, loc *time.Location) (start time.Time, remaining time.Duration, full time.Duration) {
	now = now.In(loc)
	year, month, date := now.Date()
	start = time.Date(year, month, date, startHour, 0, 0, 0, loc)
	end := time.Date(year, month, date, endHour, 0, 0, 0, loc)

	full = end.Sub(start)
	if start.Before(now) {
		start = now
	}
	remaining = end.Sub(start)
	if full <= 0 || remaining <= 0 {
		return start, 0, 0
	}

	return start, remaining, full
}

func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}
