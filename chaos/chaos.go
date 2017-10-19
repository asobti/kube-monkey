package chaos

import (
	"fmt"
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/deployments"
	"github.com/asobti/kube-monkey/kubernetes"
	kube "k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"math/rand"
	"time"
)

type Chaos struct {
	killAt     time.Time
	deployment *deployments.Deployment
}

// Create a new Chaos instance
func New(killtime time.Time, dep *deployments.Deployment) *Chaos {
	// TargetPodName will be populated at time of termination
	return &Chaos{
		killAt:     killtime,
		deployment: dep,
	}
}

func (c *Chaos) Deployment() *deployments.Deployment {
	return c.deployment
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
	client, err := CreateClient()
	if err != nil {
		resultchan <- c.NewResult(err)
		return
	}

	// Is deployment still enrolled in kube-monkey
	enrolled, err := c.deployment.IsEnrolled(client)
	if err != nil {
		resultchan <- c.NewResult(err)
		return
	}
	if !enrolled {
		resultchan <- c.NewResult(fmt.Errorf("Deployment %s is no longer enrolled in kube-monkey. Skipping\n", c.deployment.Name()))
		return
	}

	// Has deployment been blacklisted since scheduling?
	if c.deployment.IsBlacklisted(config.BlacklistedNamespaces()) {
		resultchan <- c.NewResult(fmt.Errorf("Deployment %s is blacklisted. Skipping\n", c.deployment.Name()))
		return
	}

	// Do the termination
	killAll, err := c.deployment.HasKillAll(client)
	if err != nil {
		fmt.Printf("Failed to check KillAll label for deployment %s. Proceeding with termination of a single pod.\n", c.deployment.Name())
		fmt.Printf(err.Error())
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
	pods, err := c.deployment.RunningPods(client)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("Deployment %s has no running pods at the moment", c.deployment.Name())
	}

	targetPod := RandomPodName(pods)

	fmt.Printf("Terminating pod %s for deployment %s\n", targetPod, c.deployment.Name())
	return c.DeletePod(client, targetPod)
}

// Terminates ALL pods for the deployment
// Not the default, or recommended, behavior
func (c *Chaos) TerminateAll(client *kube.Clientset) error {
	fmt.Printf("Terminating ALL pods for deployment %s\n", c.deployment.Name())

	pods, err := c.deployment.Pods(client)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("Deployment %s has no pods at the moment", c.deployment.Name())
	}

	for _, pod := range pods {
		// In case of error, log it and move on to next pod
		if err = c.DeletePod(client, pod.Name); err != nil {
			fmt.Printf("Failed to delete pod %s for deployment %s", pod.Name, c.deployment.Name())
		}
	}

	return nil
}

// Deletes a pod for a deployment
func (c *Chaos) DeletePod(client *kube.Clientset, podName string) error {
	if config.DryRun() {
		fmt.Printf("[DryRun Mode] Terminated pod %s for deployment %s\n", podName, c.deployment.Name())
		return nil
	} else {
		return c.deployment.DeletePod(client, podName)
	}
}

// Create a ChaosResult instance
func (c *Chaos) NewResult(e error) *ChaosResult {
	return &ChaosResult{
		chaos: c,
		err:   e,
	}
}

// Create, verify and return an instance of kubernetes.Clientset
func CreateClient() (*kube.Clientset, error) {
	client, err := kubernetes.NewInClusterClient()
	if err != nil {
		return nil, err
	}

	if kubernetes.VerifyClient(client) {
		return client, nil
	} else {
		return nil, fmt.Errorf("Unable to verify Kubernetes client")
	}
}

// Pick a random pod name from a list of Pods
func RandomPodName(pods []v1.Pod) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randIndex := r.Intn(len(pods))
	return pods[randIndex].Name
}
