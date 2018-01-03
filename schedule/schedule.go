package schedule

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/calendar"
	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims/factory"
)

type Schedule struct {
	entries []*chaos.Chaos
}

func (s *Schedule) Entries() []*chaos.Chaos {
	return s.entries
}

func (s *Schedule) Add(entry *chaos.Chaos) {
	s.entries = append(s.entries, entry)
}

func (s *Schedule) String() string {
	schedString := []string{}
	schedString = append(schedString, fmt.Sprint("\t********** Today's schedule **********"))
	if len(s.entries) == 0 {
		schedString = append(schedString, fmt.Sprint("No terminations scheduled"))
	} else {
		schedString = append(schedString, fmt.Sprint("\tk8 Api Kind\tKind Name\t\tTermination Time"))
		schedString = append(schedString, fmt.Sprint("\t-----------\t---------\t\t----------------"))
		for _, chaos := range s.entries {
			schedString = append(schedString, fmt.Sprintf("\t%s\t%s\t\t%s", chaos.Victim().Kind(), chaos.Victim().Name(), chaos.KillAt().Format("01/02/2006 15:04:05 -0700 MST")))
		}
	}
	schedString = append(schedString, fmt.Sprint("\t********** End of schedule **********"))

	return strings.Join(schedString, "\n")
}

func (s Schedule) Print() {
	glog.V(4).Infof("Status Update: %v terminations scheduled today", len(s.entries))
	for _, chaos := range s.entries {
		glog.V(4).Infof("%s %s scheduled for termination at %s", chaos.Victim().Kind(), chaos.Victim().Name(), chaos.KillAt().Format("01/02/2006 15:04:05 -0700 MST"))
	}
}

func New() (*Schedule, error) {
	glog.V(3).Info("Status Update: Generating schedule for terminations")
	victims, err := factory.EligibleVictims()
	if err != nil {
		return nil, err
	}

	schedule := &Schedule{
		entries: []*chaos.Chaos{},
	}

	for _, victim := range victims {
		killtime := CalculateKillTime()

		if ShouldScheduleChaos(victim.Mtbf()) {
			schedule.Add(chaos.New(killtime, victim))
		}
	}

	return schedule, nil
}

func CalculateKillTime() time.Time {
	loc := config.Timezone()
	if config.DebugEnabled() && config.DebugScheduleImmediateKill() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		// calculate a second-offset in the next minute
		secOffset := r.Intn(60)
		return time.Now().In(loc).Add(time.Duration(secOffset) * time.Second)
	} else {
		return calendar.RandomTimeInRange(config.StartHour(), config.EndHour(), loc)
	}
}

func ShouldScheduleChaos(mtbf int) bool {
	if config.DebugEnabled() && config.DebugForceShouldKill() {
		return true
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var probability float64 = 1 / float64(mtbf)
	return probability > r.Float64()
}
