package kubemonkey

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"kube-monkey/internal/pkg/calendar"
	"kube-monkey/internal/pkg/chaos"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/kubernetes"
	"kube-monkey/internal/pkg/notifications"
	"kube-monkey/internal/pkg/schedule"
)

func durationToNextRun(runhour int, loc *time.Location) time.Duration {
	if config.DebugEnabled() {
		debugDelayDuration := config.DebugScheduleDelay()
		glog.V(1).Infof("Debug mode detected!")
		glog.V(1).Infof("Status Update: Generating next schedule in %.0f sec\n", debugDelayDuration.Seconds())
		return debugDelayDuration
	}
	nextRun := calendar.NextRuntime(loc, runhour)
	glog.V(1).Infof("Status Update: Generating next schedule at %s\n", nextRun)
	return time.Until(nextRun)
}

func Run() error {
	// Verify kubernetes client can be created and works before
	// we enter execution loop
	if _, err := kubernetes.CreateClient(); err != nil {
		return err
	}

	var notificationsClient notifications.Client
	if config.NotificationsEnabled() {
		glog.V(1).Infof("Notifications enabled!")
		proxy := config.NotificationsProxy()
		if proxy != "" {
			glog.V(1).Infof("Notifications proxy set: %s!", proxy)
		}
		notificationsClient = notifications.CreateClient(&proxy)
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
		if config.NotificationsEnabled() && config.NotificationsReportSchedule() {
			notifications.ReportSchedule(notificationsClient, schedule)
		}
		fmt.Println(schedule)
		ScheduleTerminations(schedule.Entries(), notificationsClient)
	}
}

func ScheduleTerminations(entries []*chaos.Chaos, notificationsClient notifications.Client) {
	resultchan := make(chan *chaos.Result)
	defer close(resultchan)

	// Spin off all terminations
	for _, chaos := range entries {
		go chaos.Schedule(resultchan)
	}

	completedCount := 0
	var result *chaos.Result

	glog.V(3).Infof("Status Update: Waiting to run scheduled terminations.")

	// Gather results
	for completedCount < len(entries) {
		result = <-resultchan
		if result.Error() != nil {
			glog.Errorf("Failed to execute termination for %s %s. Error: %v", result.Victim().Kind(), result.Victim().Name(), result.Error().Error())
		} else {
			glog.V(2).Infof("Termination successfully executed for %s %s\n", result.Victim().Kind(), result.Victim().Name())
		}
		if config.NotificationsEnabled() {
			currentTime := time.Now()
			notifications.ReportAttack(notificationsClient, result, currentTime)
		}
		completedCount++
		glog.V(4).Info("Status Update: ", len(entries)-completedCount, " scheduled terminations left.")
	}

	glog.V(3).Info("Status Update: All terminations done.")
}
