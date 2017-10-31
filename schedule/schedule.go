package schedule

import (
	"time"
	"math/rand"
	
	"github.com/golang/glog"
	
	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/calendar"
	"github.com/asobti/kube-monkey/deployments"
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
	glog.V(3).Info("********** Today's schedule **********")
	if len(s.entries) == 0 {
		glog.V(3).Info("No terminations scheduled")
	} else {
		glog.V(3).Info("\tDeployment\t\tTermination time\n")
		glog.V(3).Info("\t----------\t\t----------------\n")
		for _, chaos := range s.entries {
			glog.V(3).Info("\t%s\t\t%s\n", chaos.Deployment().Name(), chaos.KillAt())
		}
	}

	glog.V(3).Info("********** End of schedule **********")
}

func New() (*Schedule, error) {
	glog.V(2).Info("Generating schedule for terminations")
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
