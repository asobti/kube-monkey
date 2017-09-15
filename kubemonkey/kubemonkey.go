package kubemonkey

import (
	"fmt"
	"github.com/andreic92/kube-monkey/calendar"
	"github.com/andreic92/kube-monkey/chaos"
	"github.com/andreic92/kube-monkey/config"
	"github.com/andreic92/kube-monkey/kubernetes"
	"github.com/andreic92/kube-monkey/schedule"
	"time"
)

func verifyKubeClient() error {
	client, err := kubernetes.NewInClusterClient()
	if err != nil {
		return err
	}
	if !kubernetes.VerifyClient(client) {
		fmt.Println(err)
		return fmt.Errorf("Unable to verify client connectivity to Kubernetes server")
	}
	return nil
}

func durationToNextRun(runhour int, location *time.Location) time.Duration {
	if config.DebugEnabled() {
		debugDelayDuration := config.DebugScheduleDelay()
		fmt.Printf("Debug mode detected. Next run scheduled in %.0f sec\n", debugDelayDuration.Seconds())
		return debugDelayDuration
	} else {
		nextRun := calendar.NextRuntime(location, runhour)
		fmt.Printf("Next run scheduled at %s\n", nextRun)
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
			panic(err.Error())
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

	fmt.Println("Waiting for terminations to run")

	// Gather results
	for completedCount < len(entries) {
		result = <-resultchan
		if result.Error() != nil {
			fmt.Printf("Failed to execute termination for deployment %s. Error:\n", result.Deployment().Name())
			fmt.Println(result.Error().Error())
		} else {
			fmt.Printf("Termination successfully executed for deployment %s\n", result.Deployment().Name())
		}
		completedCount++
	}

	fmt.Println("All terminations done")
}
