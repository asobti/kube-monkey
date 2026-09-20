/*
Package chaos holds one scheduled termination: a victim and the time its pods
should be terminated at.
*/
package chaos

import (
	"fmt"
	"time"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/kubernetes"
	"kube-monkey/internal/pkg/victims"

	kube "k8s.io/client-go/kubernetes"
)

type Chaos struct {
	killAt time.Time
	victim *victims.Victim
}

// New creates a new Chaos instance
func New(killtime time.Time, victim *victims.Victim) *Chaos {
	return &Chaos{
		killAt: killtime,
		victim: victim,
	}
}

func (c *Chaos) Victim() *victims.Victim {
	return c.victim
}

func (c *Chaos) KillAt() time.Time {
	return c.killAt
}

// Schedule waits until the kill time and then executes the chaos
func (c *Chaos) Schedule(resultchan chan<- *Result) {
	time.Sleep(c.DurationToKillTime())
	c.Execute(resultchan)
}

// DurationToKillTime calculates the duration from now until Chaos.killAt
func (c *Chaos) DurationToKillTime() time.Duration {
	return time.Until(c.killAt)
}

// Execute terminates the victim's pods and sends the outcome back over the
// channel provided
func (c *Chaos) Execute(resultchan chan<- *Result) {
	// The client is created here rather than at scheduling time because a
	// termination can be hours away, and a connection does not keep that long
	clientset, err := kubernetes.CreateClient()
	if err != nil {
		resultchan <- c.NewResult(err)
		return
	}

	if err := c.verifyExecution(clientset); err != nil {
		resultchan <- c.NewResult(err)
		return
	}

	resultchan <- c.NewResult(c.terminate(clientset))
}

// verifyExecution checks the victim has not opted out since it was scheduled
func (c *Chaos) verifyExecution(clientset kube.Interface) error {
	enrolled, err := c.victim.IsEnrolled(clientset)
	if err != nil {
		return err
	}
	if !enrolled {
		return fmt.Errorf("%s %s is no longer enrolled in kube-monkey. Skipping", c.victim.Kind(), c.victim.Name())
	}

	if c.victim.IsBlacklisted() {
		return fmt.Errorf("%s %s is blacklisted. Skipping", c.victim.Kind(), c.victim.Name())
	}

	if !c.victim.IsWhitelisted() {
		return fmt.Errorf("%s %s is not whitelisted. Skipping", c.victim.Kind(), c.victim.Name())
	}

	return nil
}

// terminate kills the number of pods the victim's kill mode asks for
func (c *Chaos) terminate(clientset kube.Interface) error {
	killType, err := c.victim.KillType(clientset)
	if err != nil {
		return fmt.Errorf("failed to check %s label for %s %s: %w", config.KillTypeLabelKey, c.victim.Kind(), c.victim.Name(), err)
	}

	killNum, err := c.killNumber(clientset, killType)
	if err != nil {
		return err
	}

	return c.victim.DeleteRandomPods(clientset, killNum)
}

// killNumber works out how many pods the kill mode asks for
func (c *Chaos) killNumber(clientset kube.Interface, killType string) (int, error) {
	// Killing all of them is the only mode that does not read a kill value
	if killType == config.KillAllLabelValue {
		return c.victim.KillNumberForKillingAll(clientset)
	}

	killValue, err := c.victim.KillValue(clientset)
	if err != nil {
		return 0, fmt.Errorf("failed to check %s label for %s %s: %w", config.KillValueLabelKey, c.victim.Kind(), c.victim.Name(), err)
	}

	switch killType {
	case config.KillFixedLabelValue:
		return killValue, nil
	case config.KillRandomMaxLabelValue:
		return c.victim.KillNumberForMaxPercentage(clientset, killValue)
	case config.KillFixedPercentageLabelValue:
		return c.victim.KillNumberForFixedPercentage(clientset, killValue)
	default:
		return 0, fmt.Errorf("failed to recognize %s label %q for %s %s", config.KillTypeLabelKey, killType, c.victim.Kind(), c.victim.Name())
	}
}

// NewResult creates a Result for this chaos
func (c *Chaos) NewResult(err error) *Result {
	return &Result{
		chaos: c,
		err:   err,
	}
}
