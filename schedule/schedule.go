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

const (
	TODAY          = "\t********** Today's schedule **********"
	NO_TERMINATION = "No terminations scheduled"
	HEADER_ROW     = "\tk8 Api Kind\tKind Name\t\tTermination Time"
	SEP_ROW        = "\t-----------\t---------\t\t----------------"
	ROW_FORMAT     = "\t%s\t%s\t\t%s"
	DATE_FORMAT    = "01/02/2006 15:04:05 -0700 MST"
	END            = "\t********** End of schedule **********"
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
	schedString = append(schedString, fmt.Sprint(TODAY))
	if len(s.entries) == 0 {
		schedString = append(schedString, fmt.Sprint(NO_TERMINATION))
	} else {
		schedString = append(schedString, fmt.Sprint(HEADER_ROW))
		schedString = append(schedString, fmt.Sprint(SEP_ROW))
		for _, chaos := range s.entries {
			schedString = append(schedString, fmt.Sprintf(ROW_FORMAT, chaos.Victim().Kind(), chaos.Victim().Name(), chaos.KillAt().Format(DATE_FORMAT)))
		}
	}
	schedString = append(schedString, fmt.Sprint(END))

	return strings.Join(schedString, "\n")
}

func (s Schedule) Print() {
	glog.V(4).Infof("Status Update: %v terminations scheduled today", len(s.entries))
	for _, chaos := range s.entries {
		glog.V(4).Infof("%s %s scheduled for termination at %s", chaos.Victim().Kind(), chaos.Victim().Name(), chaos.KillAt().Format(DATE_FORMAT))
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
