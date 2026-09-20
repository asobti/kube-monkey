/*
Package schedule draws up the day's terminations: which victims die today and
at what time.
*/
package schedule

import (
	"fmt"
	"math/rand/v2"
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

// New draws up today's schedule from the victims that have opted in
func New() (*Schedule, error) {
	glog.V(3).Info("Status Update: Generating schedule for terminations")
	victims, err := factory.EligibleVictims()
	if err != nil {
		return nil, err
	}

	schedule := &Schedule{}
	for _, victim := range victims {
		for _, killtime := range CalculateKillTimes(victim.Mtbf()) {
			schedule.Add(chaos.New(killtime, victim))
		}
	}

	return schedule, nil
}

func (s *Schedule) Entries() []*chaos.Chaos {
	return s.entries
}

func (s *Schedule) Add(entry *chaos.Chaos) {
	s.entries = append(s.entries, entry)
}

func (s *Schedule) String() string {
	rows := []string{Today}

	if kubeMonkeyID := os.Getenv("KUBE_MONKEY_ID"); kubeMonkeyID != "" {
		rows = append(rows, fmt.Sprintf(KubeMonkeyID, kubeMonkeyID))
	}

	if len(s.entries) == 0 {
		rows = append(rows, NoTermination)
	} else {
		rows = append(rows, HeaderRow, SepRow)
		for _, entry := range s.entries {
			victim := entry.Victim()
			rows = append(rows, fmt.Sprintf(RowFormat, victim.Kind(), victim.Namespace(), victim.Name(), entry.KillAt().Format(DateFormat)))
		}
	}

	return strings.Join(append(rows, End), "\n")
}

// CalculateKillTimes returns the times of today's terminations for a victim
// with the given mtbf. An mtbf shorter than a day gives more than one
// termination, a longer one gives none on most days.
func CalculateKillTimes(mtbf time.Duration) []time.Time {
	loc := config.Timezone()

	if config.DebugEnabled() && config.DebugScheduleImmediateKill() {
		// Somewhere in the next minute, so a debugging run does not have to wait
		// for the configured hours to come round
		return []time.Time{time.Now().In(loc).Add(time.Duration(rand.IntN(60)) * time.Second)}
	}

	killtimes := calendar.KillTimesInRange(mtbf, config.StartHour(), config.EndHour(), loc)

	if len(killtimes) == 0 && config.DebugEnabled() && config.DebugForceShouldKill() {
		if killtime, ok := calendar.RandomTimeInRange(config.StartHour(), config.EndHour(), loc); ok {
			killtimes = append(killtimes, killtime)
		}
	}

	return killtimes
}
