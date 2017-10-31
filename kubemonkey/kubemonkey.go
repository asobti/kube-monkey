package kubemonkey

import (
	"time"
	
	"github.com/golang/glog"
	
	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/calendar"
	"github.com/asobti/kube-monkey/schedule"
)

func verifyKubeClient() error {
	_, err := chaos.CreateClient()
	return err
}

func durationToNextRun(runhour int, location *time.Location) time.Duration {
	if config.DebugEnabled() {
		debugDelayDuration := config.DebugScheduleDelay()
		glog.V(2).Infof("Debug mode detected! Generating next schedule in %.0f sec\n", debugDelayDuration.Seconds())
		return debugDelayDuration
	} else {
		nextRun := calendar.NextRuntime(location, runhour)
		glog.V(2).Infof("Generating next schedule at %s\n", nextRun)
		return nextRun.Sub(time.Now())
	}
}

func Run() error {
	// Verify kubernetes client can be created and works before
	// we enter execution loop
	if err := verifyKubeClient(); err != nil {
		return err
	}

	for {
		// Calculate duration to sleep before next run
		sleepDuration := durationToNextRun(config.RunHour(), config.Timezone())
		time.Sleep(sleepDuration)

		schedule, err := schedule.New()
		if err != nil {
			glog.Fatal(err.Error())
		}
		schedule.Print()
		ScheduleTerminations(schedule.Entries())
	}
}

func ScheduleTerminations(entries []*chaos.Chaos) {
	resultchan := make(chan *chaos.ChaosResult)
	defer close(resultchan)

	// Spin off all terminations
	for _, chaos := range entries {
		go chaos.Schedule(resultchan)
	}

	completedCount := 0
	var result *chaos.ChaosResult

	glog.V(3).Infof("Waiting for terminations to run")

	// Gather results
	for completedCount < len(entries) {
		result = <-resultchan
		if result.Error() != nil {
			glog.Errorf("Failed to execute termination for deployment %s. Error: %v", result.Deployment().Name(), result.Error().Error())
		} else {
			glog.V(2).Infof("Termination successfully executed for deployment %s\n", result.Deployment().Name())
		}
		completedCount++
	}

	glog.V(3).Info("All terminations done")
}
