package victims

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/asobti/kube-monkey/config"
	"github.com/golang/glog"
	"github.com/pkg/errors"

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
	VictimKillNumberGenerator
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
	IsEnrolled(kube.Interface) (bool, error)                // Get updated enroll status
	Selector(kube.Interface) (*metav1.LabelSelector, error) // Get labels for this controller FIXME: rename to selector
	KillType(kube.Interface) (string, error)                // Get updated kill config type
	KillValue(kube.Interface) (int, error)                  // Get updated kill config value
}

type VictimAPICalls interface {
	// Exposed Api Calls
	RunningPods(kube.Interface) ([]v1.Pod, error)
	Pods(kube.Interface) ([]v1.Pod, error)
	DeletePod(kube.Interface, string) error
	DeleteRandomPod(kube.Interface) error // Deprecated, but faster than DeleteRandomPods for single pod termination
	DeleteRandomPods(kube.Interface, int) error
	IsBlacklisted() bool
	IsWhitelisted() bool
}

type VictimKillNumberGenerator interface {
	KillNumberForMaxPercentage(kube.Interface, int) (int, error)
	KillNumberForKillingAll(kube.Interface) (int, error)
	KillNumberForKillingPodDisruptionBudget(kube.Interface, int, *metav1.LabelSelector) (int, error)
	KillNumberForFixedPercentage(kube.Interface, int) (int, error)
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

// Returns the pod disruption budget for this controller
func (v *VictimBase) PodDisruptionBudget(clientset kube.Interface, controllerSelector *metav1.LabelSelector) (*intstr.IntOrString, *intstr.IntOrString, error) {
	glog.Warningf("### TEST ### kind=%s, name=%d, namespace=%s, identifier=%s", v.kind, v.name, v.namespace, v.identifier)

	labelFilter := &metav1.ListOptions{}

	pdbs, err := clientset.PolicyV1beta1().PodDisruptionBudgets(v.namespace).List(*labelFilter)
	if err != nil {
		return nil, nil, err
	}

	// A PDB is not directly associated with a controller.
	// We need to iterate them and find one with a common selector to our controller.
	items := &pdbs.Items

	var min *intstr.IntOrString
	var max *intstr.IntOrString

	var foundMatchingSelector bool

	// index and value
	for _, element := range *items {
		var pdbMatchLabels map[string]string = element.Spec.Selector.MatchLabels
		var pdbMatchExpressions []metav1.LabelSelectorRequirement = element.Spec.Selector.MatchExpressions
		var controllerMatchLabels map[string]string = controllerSelector.MatchLabels
		var controllerMatchExpressions []metav1.LabelSelectorRequirement = controllerSelector.MatchExpressions

		glog.Warningf("### TEST (expressions) ### pdbSelector=%s, controllerSelector=%s", &pdbMatchExpressions, &controllerMatchExpressions)
		glog.Warningf("### TEST (labels) ### pdbSelector=%s, controllerSelector=%s", &pdbMatchLabels, &controllerMatchLabels)

		// assume we'll find one and set this to false when a label doesnt match
		foundMatchingSelector = true
		// check if labels match
		for k, v := range pdbMatchLabels {
			labelValue := controllerMatchLabels[k]

			if v != labelValue {
				foundMatchingSelector = false
				continue
			}
		}

		if foundMatchingSelector {
			min = element.Spec.MinAvailable
			max = element.Spec.MaxUnavailable
		}
	}

	if foundMatchingSelector {
		return min, max, nil
	} else {
		return nil, nil, errors.Wrapf(err, "unable to find a matching pdb for %s/%s", v.namespace, v.name)
	}
}

// Removes specified pod for victim
func (v *VictimBase) DeletePod(clientset kube.Interface, podName string) error {
	if config.DryRun() {
		glog.Infof("[DryRun Mode] Terminated pod %s for %s/%s", podName, v.namespace, v.name)
		return nil
	}

	pod, err := clientset.CoreV1().Pods(v.namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to get pod %s for %s/%s", podName, v.namespace, v.name)
	}

	deleteOpts := v.GetDeleteOptsForPod(pod)
	return clientset.CoreV1().Pods(v.namespace).Delete(podName, deleteOpts)
}

// Creates the DeleteOptions object for the pod. Grace period is calculated as the higher
// of configured grace period and termination grace period set on the pod
func (v *VictimBase) GetDeleteOptsForPod(pod *v1.Pod) *metav1.DeleteOptions {
	gracePeriodSec := config.GracePeriodSeconds()

	if pod.Spec.TerminationGracePeriodSeconds != nil && *pod.Spec.TerminationGracePeriodSeconds > *gracePeriodSec {
		gracePeriodSec = pod.Spec.TerminationGracePeriodSeconds
	}

	return &metav1.DeleteOptions{
		GracePeriodSeconds: gracePeriodSec,
	}
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

// Returns the number of pods to kill based on the number of all running pods
func (v *VictimBase) KillNumberForKillingAll(clientset kube.Interface) (int, error) {
	killNum, err := v.numberOfRunningPods(clientset)
	if err != nil {
		return 0, err
	}

	return killNum, nil
}

// Returns the number of pods to kill based on the pod disruption budget
func (v *VictimBase) KillNumberForKillingPodDisruptionBudget(clientset kube.Interface, killPercentage int, selector *metav1.LabelSelector) (int, error) {
	runningPods, err := v.numberOfRunningPods(clientset)
	if err != nil {
		return 0, err
	}

	pods, err := v.Pods(clientset) // all pods (dead and alive) FIXME: replace with a call to get the number of desired pods
	if err != nil {
		return 0, err
	}
	requiredPods := len(pods)

	min, max, err := v.PodDisruptionBudget(clientset, selector)
	if err != nil {
		return 0, err
	}

	glog.Warningf("### TEST ### requiredPods=%d, minPDB=%, maxPDB=%d", requiredPods, &min, &max)

	var killNum int
	if min != nil {
		switch min.Type {
		case intstr.Int:
			minAvailablePods := int(min.IntVal)
			// requiredPods: 10
			// runningPods = 9
			// minAvailable = 2
			// killPercentage = 50%
			// pdb needs 2 available pods and we can kill 50% of the remaining, which is 4 pods (out of 8), but since one is already dead then we can only kill 3

			targetKillNum := killPercentage * (requiredPods - minAvailablePods) / 100 // potentially kill 50% of (10 - 2) = 4
			killNum = targetKillNum - (requiredPods - runningPods)

			glog.Warningf("### TEST ### case min int, minAvailablePods=%d, targetKillNum=%d, killNum=%d", minAvailablePods, targetKillNum, killNum)

			if killNum <= 0 {
				return 0, fmt.Errorf("%s %s is already at or below min pdb availability (runningPods=%d, minAvailable=%d)", v.kind, v.name, runningPods, minAvailablePods)
			}
		case intstr.String:
			sanitizedMinAvailablePodsPercent := strings.Replace(min.StrVal, "%", "", -1)
			minAvailablePodsPercent, err := strconv.Atoi(sanitizedMinAvailablePodsPercent)
			if err != nil {
				return 0, err
			}

			// requiredPods: 10
			// runningPods = 9
			// minValue = 50%
			// killPercentage = 50%
			// this means we can kill 50% of 50% of the deployment, which is 2.5 pods, but since one is already dead then we can only kill 1.5

			targetKillPercentage := killPercentage * minAvailablePodsPercent / 100 // 50% of 50% = 25%
			targetKillNum := targetKillPercentage * requiredPods                   // potentially kill 25% of 10 which is 2.5
			killNum = targetKillNum - (requiredPods - runningPods)                 // in reality kill 2.5 - (1) = 1.5

			glog.Warningf("### TEST ### case min string, targetKillPercentage=%d, targetKillNum=%d, killNum=%d", targetKillPercentage, targetKillNum, killNum)

			if killNum <= 0 {
				return 0, fmt.Errorf("%s %s is already at or below min pdb availability (runningPods=%d, minAvailablePercent=%d)", v.kind, v.name, runningPods, minAvailablePodsPercent)
			}
		}
	} else if max != nil {
		switch max.Type {
		case intstr.Int:
			maxUnavailablePods := int(max.IntVal)
			// requiredPods: 10
			// runningPods = 9
			// maxUnavailable = 2
			// killPercentage = 50%
			// pdb can have at most 2 unavailable pods and we can kill 50% of those, which is 1 pods, but since one is already dead then we can only kill 3

			targetKillNum := killPercentage * maxUnavailablePods / 100 // potentially kill 50% of (2) = 4
			killNum = targetKillNum - (requiredPods - runningPods)

			glog.Warningf("### TEST ### case max int, maxUnavailablePods=%d, targetKillNum=%d, killNum=%d", maxUnavailablePods, targetKillNum, killNum)

			if killNum <= 0 {
				return 0, fmt.Errorf("%s %s is already at or above max pdb unavailability (runningPods=%d, maxUnavailable=%d)", v.kind, v.name, runningPods, maxUnavailablePods)
			}
		case intstr.String:
			sanitizedMaxUnavailablePodsPercent := strings.Replace(max.StrVal, "%", "", -1)
			maxUnavailablePodsPercent, err := strconv.Atoi(sanitizedMaxUnavailablePodsPercent)
			if err != nil {
				return 0, err
			}

			// requiredPods: 10
			// runningPods = 9
			// maxUnavailable = 50%
			// killPercentage = 50%
			// pdb can have at most 50% of pods unavailable and we want to kill 50% those, which is 25% of 10 or 2.5 pods, but since one is already dead then we can only kill 1.5

			targetKillPercentage := killPercentage * maxUnavailablePodsPercent / 100 // 50% of 50% = 25%
			targetKillNum := targetKillPercentage * requiredPods                     // potentially kill 25% of 10 which is 2.5
			killNum = targetKillNum - (requiredPods - runningPods)                   // in reality kill 2.5 - (1) = 1.5

			glog.Warningf("### TEST ### case max string, targetKillPercentage=%d, targetKillNum=%d, killNum=%d", targetKillPercentage, targetKillNum, killNum)

			if killNum <= 0 {
				return 0, fmt.Errorf("%s %s is already at or above max pdb unavailability (runningPods=%d, maxUnavailablePercent=%d)", v.kind, v.name, runningPods, maxUnavailablePodsPercent)
			}
		}
	}

	// todo: check if runningPods < killNum and adjust

	return killNum, nil
}

// Returns the number of pods to kill based on a kill percentage and the number of running pods
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

// Returns a number of pods to kill based on a a random kill percentage (between 0 and maxPercentage) and the number of running pods
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
