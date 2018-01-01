package chaos

import (
	"fmt"
	"time"
	"math/rand"
	
	"github.com/golang/glog"
	
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/kubernetes"
	"github.com/asobti/kube-monkey/victims"
	
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/api/core/v1"
)

type Chaos struct {
	killAt time.Time
	victim victims.Victim
}

// Create a new Chaos instance
func New(killtime time.Time, victim victims.Victim) *Chaos {
	// TargetPodName will be populated at time of termination
	return &Chaos{
		killAt: killtime,
		victim: victim,
	}
}

func (c *Chaos) Victim() victims.Victim {
	return c.victim
}

func (c *Chaos) KillAt() time.Time {
	return c.killAt
}

// Schedule the execution of Chaos
func (c *Chaos) Schedule(resultchan chan<- *ChaosResult) {
	time.Sleep(c.DurationToKillTime())
	c.Execute(resultchan)
}

// Calculates the duration from now until Chaos.killAt
func (c *Chaos) DurationToKillTime() time.Duration {
	return c.killAt.Sub(time.Now())
}

// Does the actual execution of the chaos, i.e.
// termination of pods
// The result is sent back over the channel provided
func (c *Chaos) Execute(resultchan chan<- *ChaosResult) {
	// Create kubernetes client
	client, err := kubernetes.CreateClient()
	if err != nil {
		resultchan <- c.NewResult(err)
		return
	}

	// Is victim still enrolled in kube-monkey
	enrolled, err := c.Victim().IsEnrolled(client)
	if err != nil {
		resultchan <- c.NewResult(err)
		return
	}
	if !enrolled {
		resultchan <- c.NewResult(fmt.Errorf("%s %s is no longer enrolled in kube-monkey. Skipping\n", c.Victim().Kind(), c.Victim().Name()))
		return
	}

	// Has victim been blacklisted since scheduling?
	if c.Victim().IsBlacklisted(config.BlacklistedNamespaces()) {
		resultchan <- c.NewResult(fmt.Errorf("%s %s is blacklisted. Skipping\n", c.Victim().Kind(), c.Victim().Name()))
		return
	}

	// Do the termination
	killAll, err := c.Victim().HasKillAll(client)
	if err != nil {
		glog.Errorf("Failed to check KillAll label for %s %s. Proceeding with termination of a single pod. Error: %v", c.Victim().Kind(), c.Victim().Name(), err.Error())
	}

	if killAll {
		err = c.TerminateAll(client)
	} else {
		err = c.Terminate(client)
	}

	if err != nil {
		resultchan <- c.NewResult(err)
	} else {
		// Send a success msg
		resultchan <- c.NewResult(nil)
	}
}

// Runs the actual pod-termination logic
func (c *Chaos) Terminate(client *kube.Clientset) error {
	// Pick a target pod to delete
	pods, err := c.Victim().RunningPods(client)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("%s %s has no running pods at the moment", c.Victim().Kind(), c.Victim().Name())
	}

	targetPod := RandomPodName(pods)

	glog.V(2).Infof("Terminating pod %s for %s %s\n", targetPod, c.Victim().Kind(), c.Victim().Name())
	return c.Victim().DeletePod(client, targetPod)
}

// Terminates ALL pods for the victim
// Not the default, or recommended, behavior
func (c *Chaos) TerminateAll(client *kube.Clientset) error {
	glog.V(1).Infof("Terminating ALL pods for %s %s\n", c.Victim().Kind(), c.Victim().Name())

	pods, err := c.Victim().Pods(client)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("%s %s has no pods at the moment", c.Victim().Kind(), c.Victim().Name())
	}

	for _, pod := range pods {
		// In case of error, log it and move on to next pod
		if err = c.DeletePod(client, pod.Name); err != nil {
			glog.Errorf("Failed to delete pod %s for %s %s", pod.Name, c.Victim().Kind(), c.Victim().Name())
		}
	}

	return nil
}

// Deletes a pod for a victim
func (c *Chaos) DeletePod(client *kube.Clientset, podName string) error {
	if config.DryRun() {
		glog.V(1).Infof("[DryRun Mode] Terminated pod %s for %s %s\n", podName, c.Victim().Kind(), c.Victim().Name())
		return nil
	} else {
		return c.Victim().DeletePod(client, podName)
	}
}

// Create a ChaosResult instance
func (c *Chaos) NewResult(e error) *ChaosResult {
	return &ChaosResult{
		chaos: c,
		err:   e,
	}
}

// Pick a random pod name from a list of Pods
func RandomPodName(pods []v1.Pod) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randIndex := r.Intn(len(pods))
	return pods[randIndex].Name
}
