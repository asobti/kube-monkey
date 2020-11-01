package victims

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/asobti/kube-monkey/config"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	kube "k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Victim interface {
	VictimBaseTemplate
	VictimSpecificAPICalls
	VictimKillNumberGenerator
}

type VictimBaseTemplate interface {
	// Get value methods
	Kind() string
	Name() string
	Namespace() string
	Identifier() string
	Mtbf() string

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
	RunningPods(kube.Interface) ([]corev1.Pod, error)
	Pods(kube.Interface) ([]corev1.Pod, error)
	DeletePod(kube.Interface, string) error
	DeleteRandomPod(kube.Interface) error // Deprecated, but faster than DeleteRandomPods for single pod termination
	DeleteRandomPods(kube.Interface, int) error
	IsBlacklisted() bool
	IsWhitelisted() bool
}

type VictimKillNumberGenerator interface {
	KillNumberForMaxPercentage(kube.Interface, int) (int, error)
	KillNumberForKillingAll(kube.Interface) (int, error)
	KillNumberForFixedPercentage(kube.Interface, int) (int, error)
}

type VictimBase struct {
	kind       string
	name       string
	namespace  string
	identifier string
	mtbf       string

	VictimBaseTemplate
}

func New(kind, name, namespace, identifier string, mtbf string) *VictimBase {
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

func (v *VictimBase) Mtbf() string {
	return v.mtbf
}

// RunningPods returns a list of running pods for the victim
func (v *VictimBase) RunningPods(clientset kube.Interface) (runningPods []corev1.Pod, err error) {
	pods, err := v.Pods(clientset)
	if err != nil {
		return nil, err
	}

	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods = append(runningPods, pod)
		}
	}

	return runningPods, nil
}

// Pods returns a list of pods under the victim
func (v *VictimBase) Pods(clientset kube.Interface) ([]corev1.Pod, error) {
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

// DeletePod removes specified pod for victim
func (v *VictimBase) DeletePod(clientset kube.Interface, podName string) error {
	if config.DryRun() {
		glog.Infof("[DryRun Mode] Terminated pod %s for %s/%s", podName, v.namespace, v.name)
		return nil
	}

	deleteOpts := v.GetDeleteOptsForPod()
	return clientset.CoreV1().Pods(v.namespace).Delete(podName, deleteOpts)
}

// Creates the DeleteOptions object
// Grace period is derived from config
func (v *VictimBase) GetDeleteOptsForPod() *metav1.DeleteOptions {
	gracePeriodSec := config.GracePeriodSeconds()

	return &metav1.DeleteOptions{
		GracePeriodSeconds: gracePeriodSec,
	}
}

// DeleteRandomPods removes specified number of random pods for the victim
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
	case killNum == 0:
		return fmt.Errorf("no terminations requested for %s %s", v.kind, v.name)
	case numPods < killNum:
		glog.Warningf("%s %s has only %d currently running pods, but %d terminations requested", v.kind, v.name, numPods, killNum)
		fallthrough
	case numPods == killNum:
		glog.V(6).Infof("Killing ALL %d running pods for %s %s", numPods, v.kind, v.name)
	case killNum < 0:
		return fmt.Errorf("cannot request negative terminations %d for %s %s", killNum, v.kind, v.name)
	case numPods > killNum:
		glog.V(6).Infof("Killing %d running pods for %s %s", killNum, v.kind, v.name)
	default:
		return fmt.Errorf("unexpected behavior for terminating %s %s", v.kind, v.name)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < killNum; i++ {
		victimIndex := r.Intn(numPods)
		targetPod := pods[victimIndex].Name

		glog.V(6).Infof("Terminating pod %s for %s %s/%s\n", targetPod, v.kind, v.namespace, v.name)

		err = v.DeletePod(clientset, targetPod)
		if err != nil {
			return err
		}
	}

	// Successful termination
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

// IsBlacklisted checks if this victim is blacklisted
func (v *VictimBase) IsBlacklisted() bool {
	if config.BlacklistEnabled() {
		blacklist := config.BlacklistedNamespaces()
		return blacklist.Has(v.namespace)
	}
	return false
}

// IsWhitelisted checks if this victim is whitelisted
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
func RandomPodName(pods []corev1.Pod) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randIndex := r.Intn(len(pods))
	return pods[randIndex].Name
}

// KillNumberForKillingAll returns the number of pods to kill based on the number of all running pods
func (v *VictimBase) KillNumberForKillingAll(clientset kube.Interface) (int, error) {
	killNum, err := v.numberOfRunningPods(clientset)
	if err != nil {
		return 0, err
	}

	return killNum, nil
}

// KillNumberForFixedPercentage returns the number of pods to kill based on a kill percentage and the number of running pods
func (v *VictimBase) KillNumberForFixedPercentage(clientset kube.Interface, killPercentage int) (int, error) {
	if killPercentage == 0 {
		glog.V(6).Infof("Not terminating any pods for %s %s as kill percentage is 0\n", v.kind, v.name)
		// Report success
		return 0, nil
	}
	if killPercentage < 0 || killPercentage > 100 {
		return 0, fmt.Errorf("percentage value of %d is invalid. Must be [0-100]", killPercentage)
	}

	numRunningPods, err := v.numberOfRunningPods(clientset)
	if err != nil {
		return 0, err
	}

	numberOfPodsToKill := float64(numRunningPods) * float64(killPercentage) / 100
	killNum := int(math.Floor(numberOfPodsToKill))

	return killNum, nil
}

// KillNumberForMaxPercentage returns a number of pods to kill based on a a random kill percentage (between 0 and maxPercentage) and the number of running pods
func (v *VictimBase) KillNumberForMaxPercentage(clientset kube.Interface, maxPercentage int) (int, error) {
	if maxPercentage == 0 {
		glog.V(6).Infof("Not terminating any pods for %s %s as kill percentage is 0", v.kind, v.name)
		// Report success
		return 0, nil
	}
	if maxPercentage < 0 || maxPercentage > 100 {
		return 0, fmt.Errorf("percentage value of %d is invalid. Must be [0-100]", maxPercentage)
	}

	numRunningPods, err := v.numberOfRunningPods(clientset)
	if err != nil {
		return 0, err
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	killPercentage := r.Intn(maxPercentage + 1) // + 1 because Intn works with half open interval [0,n) and we want [0,n]
	numberOfPodsToKill := float64(numRunningPods) * float64(killPercentage) / 100
	killNum := int(math.Floor(numberOfPodsToKill))

	return killNum, nil
}

// Returns the number of running pods or 0 if the operation fails
func (v *VictimBase) numberOfRunningPods(clientset kube.Interface) (int, error) {
	pods, err := v.RunningPods(clientset)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to get running pods for victim %s %s", v.kind, v.name)
	}

	return len(pods), nil
}
