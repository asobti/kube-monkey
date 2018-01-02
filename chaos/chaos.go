package chaos

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/kubernetes"
	"github.com/asobti/kube-monkey/victims"

	kube "k8s.io/client-go/kubernetes"
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

// Exposed function that calls the actual execution of the chaos, i.e. termination of pods
// The result is sent back over the channel provided
func (c *Chaos) Execute(resultchan chan<- *ChaosResult) {
	// Create kubernetes clientset
	clientset, err := kubernetes.CreateClient()
	if err != nil {
		resultchan <- c.NewResult(err)
		return
	}

	err = c.verifyExecution(clientset)
	if err != nil {
		resultchan <- c.NewResult(err)
		return
	}

	err = c.terminate(clientset)
	if err != nil {
		resultchan <- c.NewResult(err)
	}

	// Send a success msg
	resultchan <- c.NewResult(nil)
}

// Verify if the victim has opted out since scheduling
func (c *Chaos) verifyExecution(clientset *kube.Clientset) error {
	// Is victim still enrolled in kube-monkey
	enrolled, err := c.Victim().IsEnrolled(clientset)
	if err != nil {
		return err
	}

	if !enrolled {
		return fmt.Errorf("%s %s is no longer enrolled in kube-monkey. Skipping\n", c.Victim().Kind(), c.Victim().Name())
	}

	// Has the victim been blacklisted since scheduling?
	if c.Victim().IsBlacklisted() {
		return fmt.Errorf("%s %s is blacklisted. Skipping\n", c.Victim().Kind(), c.Victim().Name())
	}

	// Send back valid for termination
	return nil
}

// The termination type and termination of pods happens here
func (c *Chaos) terminate(clientset *kube.Clientset) error {
	// Do the termination
	killAll, err := c.Victim().HasKillAll(clientset)
	if err != nil {
		glog.Errorf("Failed to check KillAll label for %s %s. Proceeding with termination of a single pod. Error: %v", c.Victim().Kind(), c.Victim().Name(), err.Error())
	}

	if killAll {
		err = c.terminateAll(clientset)
	} else {
		err = c.terminatePod(clientset)
	}

	// Send back termination success
	return nil
}

// Terminates one random pod
func (c *Chaos) terminatePod(clientset *kube.Clientset) error {
	return c.Victim().DeleteRandomPod(clientset)
}

// Terminates ALL pods for the victim
// Not the default, or recommended, behavior
func (c *Chaos) terminateAll(clientset *kube.Clientset) error {
	glog.V(1).Infof("Terminating ALL pods for %s %s\n", c.Victim().Kind(), c.Victim().Name())

	pods, err := c.Victim().Pods(clientset)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("%s %s has no pods at the moment", c.Victim().Kind(), c.Victim().Name())
	}

	for _, pod := range pods {
		// In case of error, log it and move on to next pod
		if err = c.Victim().DeletePod(clientset, pod.Name); err != nil {
			glog.Errorf("Failed to delete pod %s for %s %s", pod.Name, c.Victim().Kind(), c.Victim().Name())
		}
	}

	return nil
}

// Create a ChaosResult instance
func (c *Chaos) NewResult(e error) *ChaosResult {
	return &ChaosResult{
		chaos: c,
		err:   e,
	}
}
