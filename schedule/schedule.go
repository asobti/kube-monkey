package schedule

import (
	"fmt"
	"github.com/asobti/kube-monkey/calendar"
	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/deployments"
	"math/rand"
	"time"
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

func (s *Schedule) Print() {
	fmt.Println("********** Today's schedule **********")
	if len(s.entries) == 0 {
		fmt.Println("No terminations scheduled")
	} else {
		fmt.Printf("\tDeployment\t\tTermination time\n")
		fmt.Printf("\t----------\t\t----------------\n")
		for _, chaos := range s.entries {
			fmt.Printf("\t%s\t\t%s\n", chaos.Deployment().Name(), chaos.KillAt())
		}
	}

	fmt.Println("********** End of schedule **********")
}

func New() (*Schedule, error) {
	fmt.Println("Generating schedule for terminations")
	deployments, err := deployments.EligibleDeployments()
	if err != nil {
		return nil, err
	}

	schedule := &Schedule{
		entries: []*chaos.Chaos{},
	}

	for _, dep := range deployments {
		killtime := CalculateKillTime()

		if ShouldScheduleChaos(dep.Mtbf()) {
			schedule.Add(chaos.New(killtime, dep))
		}
	}

	return schedule, nil
}

func CalculateKillTime() time.Time {
	if config.DebugEnabled() && config.DebugScheduleImmediateKill() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		// calculate a second-offset in the next minute
		secOffset := r.Intn(60)
		return time.Now().Add(time.Duration(secOffset) * time.Second)
	} else {
		return calendar.RandomTimeInRange(config.StartHour(), config.EndHour(), config.Timezone())
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
