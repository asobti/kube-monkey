package chaos

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

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
		return errors.Wrapf(err, "Failed to check KillType label for %s %s", c.Victim().Kind(), c.Victim().Name())
	}

	killValue, err := c.getKillValue(clientset)

	// KillAll and KillPodDisruptionBudget are the only kill types that do not require a kill-value
	if killType != config.KillAllLabelValue && killType != config.KillPodDisruptionBudgetLabelValue && err != nil {
		return err
	}

	// Validate killtype
	switch killType {
	case config.KillFixedLabelValue:
		return c.Victim().DeleteRandomPods(clientset, killValue)
	case config.KillAllLabelValue:
		killNum, err := c.Victim().KillNumberForKillingAll(clientset)
		if err != nil {
			return err
		}
		return c.Victim().DeleteRandomPods(clientset, killNum)
	case config.KillPodDisruptionBudgetLabelValue:
		selector, err := c.Victim().Selector(clientset)
		if err != nil {
			return err
		}

		desiredNumberOfPods, numberOfHealthyPods, err := c.Victim().PodDisruptionBudget(clientset, selector)

		if err != nil {
			return err
		}

		killNum := c.Victim().KillNumberForKillingPodDisruptionBudget(clientset, desiredNumberOfPods, numberOfHealthyPods)

		return c.Victim().DeleteRandomPods(clientset, killNum)
	case config.KillRandomMaxLabelValue:
		killNum, err := c.Victim().KillNumberForMaxPercentage(clientset, killValue)
		if err != nil {
			return err
		}
		return c.Victim().DeleteRandomPods(clientset, killNum)
	case config.KillFixedPercentageLabelValue:
		killNum, err := c.Victim().KillNumberForFixedPercentage(clientset, killValue)
		if err != nil {
			return err
		}
		return c.Victim().DeleteRandomPods(clientset, killNum)
	default:
		return fmt.Errorf("failed to recognize KillType label for %s %s", c.Victim().Kind(), c.Victim().Name())
	}
}

func (c *Chaos) getKillValue(clientset kube.Interface) (int, error) {
	killValue, err := c.Victim().KillValue(clientset)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to check KillValue label for %s %s", c.Victim().Kind(), c.Victim().Name())
	}

	return killValue, nil
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
