package victims

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/config"

	kube "k8s.io/client-go/kubernetes"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Victim interface {
	VictimBaseTemplate
	VictimSpecificAPICalls
}

type VictimBaseTemplate interface {
	// Get value methods
	Kind() string
	Name() string
	Namespace() string
	Identifier() string
	Mtbf() int

	VictimAPICalls
}

type VictimSpecificAPICalls interface {
	// Depends on which version i.e. apps/v1 or extensions/v1beta2
	IsEnrolled(kube.Interface) (bool, error) // Get updated enroll status
	KillType(kube.Interface) (string, error) // Get updated kill config type
	KillValue(kube.Interface) (int, error)   // Get updated kill config value
}

type VictimAPICalls interface {
	// Exposed Api Calls
	RunningPods(kube.Interface) ([]v1.Pod, error)
	Pods(kube.Interface) ([]v1.Pod, error)
	DeletePod(kube.Interface, string) error
	DeleteRandomPod(kube.Interface) error // Deprecated, but faster than DeleteRandomPods for single pod termination
	DeleteRandomPods(kube.Interface, int) error
	DeleteRandomPodsMaxPercentage(kube.Interface, int) error
	DeletePodsFixedPercentage(kube.Interface, int) error
	TerminateAllPods(kube.Interface) error
	IsBlacklisted() bool
	IsWhitelisted() bool
}

type VictimBase struct {
	kind       string
	name       string
	namespace  string
	identifier string
	mtbf       int

	VictimBaseTemplate
}

func New(kind, name, namespace, identifier string, mtbf int) *VictimBase {
	return &VictimBase{kind: kind, name: name, namespace: namespace, identifier: identifier, mtbf: mtbf}
}

func (v *VictimBase) Kind() string {
	return v.kind
}

func (v *VictimBase) Name() string {
	return v.name
}

func (v *VictimBase) Namespace() string {
	return v.namespace
}

func (v *VictimBase) Identifier() string {
	return v.identifier
}

func (v *VictimBase) Mtbf() int {
	return v.mtbf
}

// Returns a list of running pods for the victim
func (v *VictimBase) RunningPods(clientset kube.Interface) (runningPods []v1.Pod, err error) {
	pods, err := v.Pods(clientset)
	if err != nil {
		return nil, err
	}

	for _, pod := range pods {
		if pod.Status.Phase == v1.PodRunning {
			runningPods = append(runningPods, pod)
		}
	}

	return runningPods, nil
}

// Returns a list of pods under the victim
func (v *VictimBase) Pods(clientset kube.Interface) ([]v1.Pod, error) {
	labelSelector, err := labelFilterForPods(v.identifier)
	if err != nil {
		return nil, err
	}

	podlist, err := clientset.CoreV1().Pods(v.namespace).List(*labelSelector)
	if err != nil {
		return nil, err
	}
	return podlist.Items, nil
}

// Removes specified pod for victim
func (v *VictimBase) DeletePod(clientset kube.Interface, podName string) error {
	deleteopts := &metav1.DeleteOptions{
		GracePeriodSeconds: config.GracePeriodSeconds(),
	}

	return clientset.CoreV1().Pods(v.namespace).Delete(podName, deleteopts)
}

// Removes a fixed percentage of pods for the victim
func (v *VictimBase) DeletePodsFixedPercentage(clientset kube.Interface, killPercentage int) error {
	if killPercentage < 0 || killPercentage > 100 {
		return fmt.Errorf("The kill percentage needs to be between 0 and 100. It was %d for %s %s", killPercentage, v.kind, v.name)
	}
	if killPercentage == 0 {
		glog.V(6).Infof("Not terminating any pods for %s %s as kill percentage is 0\n", v.kind, v.name)
		// Report success
		return nil
	}

	pods, err := v.RunningPods(clientset)
	if err != nil {
		return err
	}

	numPods := len(pods)

	numberOfPodsToKill := float64(numPods) * float64(killPercentage) / 100
	killNum := int(math.Floor(numberOfPodsToKill))

	glog.V(6).Infof("Killing %d percent of running pods for %s %s", killPercentage, v.kind, v.name)

	return v.DeleteRandomPods(clientset, killNum)
}

// Removes a random percentage of pods for the victim (up to the max percentage value specified)
func (v *VictimBase) DeleteRandomPodsMaxPercentage(clientset kube.Interface, maxPercentage int) error {
	if maxPercentage < 0 || maxPercentage > 100 {
		return fmt.Errorf("The max percentage needs to be between 0 and 100. It was %d for %s %s", maxPercentage, v.kind, v.name)
	}
	if maxPercentage == 0 {
		glog.V(6).Infof("Not terminating any pods for %s %s as kill percentage is 0\n", v.kind, v.name)
		// Report success
		return nil
	}

	pods, err := v.RunningPods(clientset)
	if err != nil {
		return err
	}

	numPods := len(pods)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	killPercentage := (r.Intn(maxPercentage + 1)) // + 1 because Intn works with half open interval [0,n) and we want [0,n]
	numberOfPodsToKill := float64(numPods) * float64(killPercentage) / 100
	killNum := int(math.Floor(numberOfPodsToKill))

	glog.V(6).Infof("Killing %d percent of running pods for %s %s", killPercentage, v.kind, v.name)

	return v.DeleteRandomPods(clientset, killNum)
}

// Removes specified number of random pods for the victim
func (v *VictimBase) DeleteRandomPods(clientset kube.Interface, killNum int) error {
	// Pick a target pod to delete
	pods, err := v.RunningPods(clientset)
	if err != nil {
		return err
	}

	numPods := len(pods)
	switch {
	case numPods == 0:
		return fmt.Errorf("%s %s has no running pods at the moment", v.kind, v.name)
	case numPods < killNum:
		glog.Warningf("%s %s has only %d currently running pods, but %d terminations requested", v.kind, v.name, numPods, killNum)
		fallthrough
	case numPods == killNum:
		glog.V(6).Infof("Killing ALL %d running pods for %s %s", numPods, v.kind, v.name)
	case killNum == 0:
		return fmt.Errorf("No terminations requested for %s %s", v.kind, v.name)
	case killNum < 0:
		return fmt.Errorf("Cannot request negative terminations %d for %s %s", numPods, v.kind, v.name)
	case numPods > killNum:
		glog.V(6).Infof("Killing %d running pods for %s %s", numPods, v.kind, v.name)
	default:
		return fmt.Errorf("unexpected behavior for terminating %s %s", v.kind, v.name)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	killCount := 0
	for _, i := range r.Perm(numPods) {
		if killCount == killNum {
			// Report success
			return nil
		}
		targetPod := pods[i].Name
		glog.V(6).Infof("Terminating pod %s for %s %s\n", targetPod, v.kind, v.name)
		err = v.DeletePod(clientset, targetPod)
		if err != nil {
			return err
		}
		killCount++
	}

	// Successful termination
	return nil
}

// Terminate all pods for the victim, regardless of status
func (v *VictimBase) TerminateAllPods(clientset kube.Interface) error {
	glog.V(2).Infof("Terminating ALL pods for %s %s\n", v.kind, v.name)

	pods, err := v.Pods(clientset)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("%s %s has no pods at the moment", v.kind, v.name)
	}

	for _, pod := range pods {
		// In case of error, log it and move on to next pod
		if err = v.DeletePod(clientset, pod.Name); err != nil {
			glog.Errorf("Failed to delete pod %s for %s %s", pod.Name, v.kind, v.name)
		}
	}

	return nil
}

// Deprecated for DeleteRandomPods(clientset, 1)
// Remove a random pod for the victim
func (v *VictimBase) DeleteRandomPod(clientset kube.Interface) error {
	// Pick a target pod to delete
	pods, err := v.RunningPods(clientset)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("%s %s has no running pods at the moment", v.kind, v.name)
	}

	targetPod := RandomPodName(pods)

	glog.V(6).Infof("Terminating pod %s for %s %s\n", targetPod, v.kind, v.name)
	return v.DeletePod(clientset, targetPod)
}

// Check if this victim is blacklisted
func (v *VictimBase) IsBlacklisted() bool {
	if config.BlacklistEnabled() {
		blacklist := config.BlacklistedNamespaces()
		return blacklist.Has(v.namespace)
	}
	return false
}

// Check if this victim is whitelisted
func (v *VictimBase) IsWhitelisted() bool {
	if config.WhitelistEnabled() {
		whitelist := config.WhitelistedNamespaces()
		return whitelist.Has(v.namespace)
	}
	return true
}

// Create a label filter to filter only for pods that belong to the this
// victim. This is done using the identifier label
func labelFilterForPods(identifier string) (*metav1.ListOptions, error) {
	req, err := labelRequirementForPods(identifier)
	if err != nil {
		return nil, err
	}
	labelFilter := &metav1.ListOptions{
		LabelSelector: labels.NewSelector().Add(*req).String(),
	}
	return labelFilter, nil
}

// Create a labels.Requirement that can be used to build a filter
func labelRequirementForPods(identifier string) (*labels.Requirement, error) {
	return labels.NewRequirement(config.IdentLabelKey, selection.Equals, sets.NewString(identifier).UnsortedList())
}

// RandomPodName picks a random pod name from a list of Pods
func RandomPodName(pods []v1.Pod) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randIndex := r.Intn(len(pods))
	return pods[randIndex].Name
}
