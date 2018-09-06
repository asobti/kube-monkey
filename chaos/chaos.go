package chaos

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/kubernetes"
	"github.com/asobti/kube-monkey/victims"

	kube "k8s.io/client-go/kubernetes"
)

type Chaos struct {
	killAt time.Time
	victim victims.Victim
}

// New creates a new Chaos instance
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
func (c *Chaos) Schedule(resultchan chan<- *Result) {
	time.Sleep(c.DurationToKillTime())
	c.Execute(resultchan)
}

// Calculates the duration from now until Chaos.killAt
func (c *Chaos) DurationToKillTime() time.Duration {
	return time.Until(c.killAt)
}

// Exposed function that calls the actual execution of the chaos, i.e. termination of pods
// The result is sent back over the channel provided
func (c *Chaos) Execute(resultchan chan<- *Result) {
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
		return
	}

	// Send a success msg
	resultchan <- c.NewResult(nil)
}

// Verify if the victim has opted out since scheduling
func (c *Chaos) verifyExecution(clientset kube.Interface) error {
	// Is victim still enrolled in kube-monkey
	enrolled, err := c.Victim().IsEnrolled(clientset)
	if err != nil {
		return err
	}

	if !enrolled {
		return fmt.Errorf("%s %s is no longer enrolled in kube-monkey. Skipping", c.Victim().Kind(), c.Victim().Name())
	}

	// Has the victim been blacklisted since scheduling?
	if c.Victim().IsBlacklisted() {
		return fmt.Errorf("%s %s is blacklisted. Skipping", c.Victim().Kind(), c.Victim().Name())
	}

	// Has the victim been removed from the whitelist since scheduling?
	if !c.Victim().IsWhitelisted() {
		return fmt.Errorf("%s %s is not whitelisted. Skipping", c.Victim().Kind(), c.Victim().Name())
	}

	// Send back valid for termination
	return nil
}

// The termination type and value is processed here
func (c *Chaos) terminate(clientset kube.Interface) error {
	killType, err := c.Victim().KillType(clientset)
	if err != nil {
		glog.Errorf("Failed to check KillType label for %s %s. Proceeding with termination of a single pod. Error: %v", c.Victim().Kind(), c.Victim().Name(), err.Error())
		return c.terminatePod(clientset)
	}

	// Validate killtype
	switch killType {
	case config.KillFixedLabelValue:
		return c.Victim().DeleteRandomPods(clientset, c.getKillValue(clientset))
	case config.KillAllLabelValue:
		killNum := c.Victim().KillNumberForKillingAll(clientset)
		return c.Victim().DeleteRandomPods(clientset, killNum)
	case config.KillRandomMaxLabelValue:
		killNum := c.Victim().KillNumberForMaxPercentage(clientset, c.getKillValue(clientset))
		return c.Victim().DeleteRandomPods(clientset, killNum)
	case config.KillFixedPercentageLabelValue:
		killNum := c.Victim().KillNumberForFixedPercentage(clientset, c.getKillValue(clientset))
		return c.Victim().DeleteRandomPods(clientset, killNum)
	default:
		return fmt.Errorf("Failed to recognize KillType label for %s %s", c.Victim().Kind(), c.Victim().Name())
	}
}

func (c *Chaos) getKillValue(clientset kube.Interface) int {
	killValue, err := c.Victim().KillValue(clientset)
	if err != nil {
		glog.Errorf("Failed to check KillValue label for %s %s. Defaulting to 1. Error: %v", c.Victim().Kind(), c.Victim().Name(), err.Error())
		killValue = 1
	}

	return killValue
}

// Redundant for DeleteRandomPods(clientset,1) but DeleteRandomPod is faster
// Terminates one random pod
func (c *Chaos) terminatePod(clientset kube.Interface) error {
	return c.Victim().DeleteRandomPod(clientset)
}

// Create a ChaosResult instance
func (c *Chaos) NewResult(e error) *Result {
	return &Result{
		chaos: c,
		err:   e,
	}
}
