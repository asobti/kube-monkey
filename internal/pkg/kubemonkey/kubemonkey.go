/*
Package kubemonkey is kube-monkey's main loop: draw up a schedule once a day,
then carry out the terminations on it.
*/
package kubemonkey

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"kube-monkey/internal/pkg/calendar"
	"kube-monkey/internal/pkg/chaos"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/kubernetes"
	"kube-monkey/internal/pkg/metrics"
	"kube-monkey/internal/pkg/notifications"
	"kube-monkey/internal/pkg/schedule"
)

func Run() error {
	// Verify kubernetes client can be created and works before
	// we enter execution loop
	if _, err := kubernetes.CreateClient(); err != nil {
		return err
	}

	if config.MetricsEnabled() {
		if err := metrics.Serve(config.MetricsAddress()); err != nil {
			return err
		}
	}

	var notificationsClient notifications.Client
	if config.NotificationsEnabled() {
		glog.V(1).Infof("Notifications enabled!")
		proxy := config.NotificationsProxy()
		if proxy != "" {
			glog.V(1).Infof("Notifications proxy set: %s!", proxy)
		}
		notificationsClient = notifications.CreateClient(proxy)
	}

	for {
		time.Sleep(durationToNextRun(config.RunHour(), config.Timezone(), config.RunDays()))

		schedule, err := schedule.New()
		if err != nil {
			glog.Fatal(err.Error())
		}

		if config.NotificationsEnabled() && config.NotificationsReportSchedule() {
			if err := notifications.ReportSchedule(notificationsClient, schedule); err != nil {
				glog.Errorf("Failed to report the schedule. Error: %v", err)
			}
		}
		fmt.Println(schedule)

		ScheduleTerminations(schedule.Entries(), notificationsClient)
	}
}

func durationToNextRun(runhour int, loc *time.Location, runDays []time.Weekday) time.Duration {
	if config.DebugEnabled() {
		debugDelayDuration := config.DebugScheduleDelay()
		glog.V(1).Infof("Debug mode detected!")
		glog.V(1).Infof("Status Update: Generating next schedule in %.0f sec\n", debugDelayDuration.Seconds())
		return debugDelayDuration
	}

	nextRun := calendar.NextRuntime(loc, runhour, runDays)
	glog.V(1).Infof("Status Update: Generating next schedule at %s\n", nextRun)
	return time.Until(nextRun)
}

// ScheduleTerminations carries out every termination on the schedule and
// returns once the last one is done
func ScheduleTerminations(entries []*chaos.Chaos, notificationsClient notifications.Client) {
	// Buffered so a termination that finishes while another result is being
	// reported does not hold up its goroutine
	results := make(chan *chaos.Result, len(entries))

	metrics.RecordSchedule(len(entries))

	for _, entry := range entries {
		victim := entry.Victim()
		metrics.RecordScheduledTermination(victim.Kind(), victim.Namespace(), victim.Name())
		go entry.Schedule(results)
	}

	glog.V(3).Infof("Status Update: Waiting to run scheduled terminations.")
	reportResults(results, len(entries), notificationsClient)
	glog.V(3).Info("Status Update: All terminations done.")
}

// reportResults logs, records and notifies the outcome of each termination as
// it comes in, until all of them have been accounted for
func reportResults(results <-chan *chaos.Result, expected int, notificationsClient notifications.Client) {
	for completed := 0; completed < expected; completed++ {
		result := <-results
		victim := result.Victim()

		if result.Error() != nil {
			glog.Errorf("Failed to execute termination for %s %s. Error: %v", victim.Kind(), victim.Name(), result.Error())
		} else {
			glog.V(2).Infof("Termination successfully executed for %s %s\n", victim.Kind(), victim.Name())
		}

		metrics.RecordTermination(victim.Kind(), victim.Namespace(), victim.Name(), result.Error())

		if config.NotificationsEnabled() {
			if err := notifications.ReportAttack(notificationsClient, result, time.Now()); err != nil {
				glog.Errorf("Failed to report the attack on %s %s. Error: %v", victim.Kind(), victim.Name(), err)
			}
		}

		glog.V(4).Info("Status Update: ", expected-completed-1, " scheduled terminations left.")
	}
}
