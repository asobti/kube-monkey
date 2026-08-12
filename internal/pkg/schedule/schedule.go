package schedule

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"

	"kube-monkey/internal/pkg/calendar"
	"kube-monkey/internal/pkg/chaos"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/victims/factory"
)

const (
	Today         = "\t********** Today's schedule **********"
	KubeMonkeyID  = "\tKubeMonkey ID: %s"
	NoTermination = "\tNo terminations scheduled"
	HeaderRow     = "\tk8 Api Kind\tKind Namespace\tKind Name\t\tTermination Time"
	SepRow        = "\t-----------\t--------------\t---------\t\t----------------"
	RowFormat     = "\t%s\t%s\t%s\t\t%s"
	DateFormat    = "01/02/2006 15:04:05 -0700 MST"
	End           = "\t********** End of schedule **********"
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

	schedString = append(schedString, fmt.Sprint(Today))

	kubeMonkeyID := os.Getenv("KUBE_MONKEY_ID")
	if kubeMonkeyID != "" {
		schedString = append(schedString, fmt.Sprintf(KubeMonkeyID, kubeMonkeyID))
	}

	if len(s.entries) == 0 {
		schedString = append(schedString, fmt.Sprint(NoTermination))
	} else {
		schedString = append(schedString, fmt.Sprint(HeaderRow))
		schedString = append(schedString, fmt.Sprint(SepRow))
		for _, chaos := range s.entries {
			schedString = append(schedString, fmt.Sprintf(RowFormat, chaos.Victim().Kind(), chaos.Victim().Namespace(), chaos.Victim().Name(), chaos.KillAt().Format(DateFormat)))
		}
	}
	schedString = append(schedString, fmt.Sprint(End))

	return strings.Join(schedString, "\n")
}

func (s Schedule) Print() {
	glog.V(4).Infof("Status Update: %v terminations scheduled today", len(s.entries))
	for _, chaos := range s.entries {
		glog.V(4).Infof("%s %s scheduled for termination at %s", chaos.Victim().Kind(), chaos.Victim().Name(), chaos.KillAt().Format(DateFormat))
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
		for _, killtime := range CalculateKillTimes(victim.Mtbf()) {
			schedule.Add(chaos.New(killtime, victim))
		}
	}

	return schedule, nil
}

// CalculateKillTimes returns the times of today's terminations for a victim
// with the given mtbf. An mtbf shorter than a day gives more than one
// termination, a longer one gives none on most days.
func CalculateKillTimes(mtbf time.Duration) []time.Time {
	loc := config.Timezone()
	if config.DebugEnabled() && config.DebugScheduleImmediateKill() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		// calculate a second-offset in the next minute
		secOffset := r.Intn(60)
		return []time.Time{time.Now().In(loc).Add(time.Duration(secOffset) * time.Second)}
	}

	killtimes := calendar.KillTimesInRange(mtbf, config.StartHour(), config.EndHour(), loc)

	if len(killtimes) == 0 && config.DebugEnabled() && config.DebugForceShouldKill() {
		if killtime, ok := calendar.RandomTimeInRange(config.StartHour(), config.EndHour(), loc); ok {
			killtimes = append(killtimes, killtime)
		}
	}

	return killtimes
}
