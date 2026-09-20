/*
Package victims turns a kubernetes object that opted in to kube-monkey into
something whose pods can be counted and terminated.

Every kind of victim works the same way once it has been found: kube-monkey
reads its own labels off the object to learn how many pods to kill, and uses a
label selector to find the pods. Only the API call that reaches the object
differs between kinds, which is what CurrentLabels stands in for.
*/
package victims

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"time"

	"kube-monkey/internal/pkg/calendar"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/metrics"

	"github.com/golang/glog"

	kube "k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// Victim is a workload kube-monkey can terminate pods for
type Victim struct {
	kind          string
	name          string
	namespace     string
	identifier    string
	mtbf          time.Duration
	podSelector   labels.Selector
	currentLabels CurrentLabels
}

// CurrentLabels reads the labels the victim carries right now. Terminations are
// scheduled hours in advance, so the labels are read again at kill time and a
// victim that opted out or changed its settings in the meantime is respected.
type CurrentLabels func(kube.Interface) (map[string]string, error)

// Spec is everything kube-monkey needs to know about a victim
type Spec struct {
	Kind          string
	Name          string
	Namespace     string
	Identifier    string
	Mtbf          time.Duration
	PodSelector   labels.Selector
	CurrentLabels CurrentLabels
}

func New(spec Spec) *Victim {
	if spec.CurrentLabels == nil {
		// Reporting a termination as failed is the safe way to be short of a
		// label reader, because nothing gets killed either way
		spec.CurrentLabels = func(kube.Interface) (map[string]string, error) {
			return nil, fmt.Errorf("%s %s has no way to read its labels back", spec.Kind, spec.Name)
		}
	}

	return &Victim{
		kind:          spec.Kind,
		name:          spec.Name,
		namespace:     spec.Namespace,
		identifier:    spec.Identifier,
		mtbf:          spec.Mtbf,
		podSelector:   spec.PodSelector,
		currentLabels: spec.CurrentLabels,
	}
}

// SpecFromLabels reads the kube-monkey configuration an object carries in its
// labels. The caller fills in the pod selector and the label reader, which are
// the two things that differ between kinds of victim.
func SpecFromLabels(kind, name, namespace string, objectLabels map[string]string) (Spec, error) {
	identifier, ok := objectLabels[config.IdentLabelKey]
	if !ok {
		return Spec{}, fmt.Errorf("%s %s does not have %s label", kind, name, config.IdentLabelKey)
	}

	mtbfLabel, ok := objectLabels[config.MtbfLabelKey]
	if !ok {
		return Spec{}, fmt.Errorf("%s %s does not have %s label", kind, name, config.MtbfLabelKey)
	}

	mtbf, err := calendar.ParseMtbf(mtbfLabel)
	if err != nil {
		return Spec{}, fmt.Errorf("%s %s has an %w", kind, name, err)
	}

	return Spec{
		Kind:       kind,
		Name:       name,
		Namespace:  namespace,
		Identifier: identifier,
		Mtbf:       mtbf,
	}, nil
}

func (v *Victim) Kind() string {
	return v.kind
}

func (v *Victim) Name() string {
	return v.name
}

func (v *Victim) Namespace() string {
	return v.namespace
}

func (v *Victim) Identifier() string {
	return v.identifier
}

func (v *Victim) Mtbf() time.Duration {
	return v.mtbf
}

// PodSelector returns the selector used to find the pods belonging to this victim
func (v *Victim) PodSelector() labels.Selector {
	return v.podSelector
}

// IsEnrolled reports whether the victim is still opted in to kube-monkey
func (v *Victim) IsEnrolled(clientset kube.Interface) (bool, error) {
	current, err := v.currentLabels(clientset)
	if err != nil {
		return false, err
	}

	return current[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

// KillType returns how the victim wants its pods picked, which is one of the
// kill mode label values
func (v *Victim) KillType(clientset kube.Interface) (string, error) {
	current, err := v.currentLabels(clientset)
	if err != nil {
		return "", err
	}

	killType, ok := current[config.KillTypeLabelKey]
	if !ok {
		return "", fmt.Errorf("%s %s does not have %s label", v.kind, v.name, config.KillTypeLabelKey)
	}

	return killType, nil
}

// KillValue returns the number the kill mode works off, which is a count of
// pods or a percentage depending on the mode
func (v *Victim) KillValue(clientset kube.Interface) (int, error) {
	current, err := v.currentLabels(clientset)
	if err != nil {
		return 0, err
	}

	value, ok := current[config.KillValueLabelKey]
	if !ok {
		return 0, fmt.Errorf("%s %s does not have %s label", v.kind, v.name, config.KillValueLabelKey)
	}

	killValue, err := strconv.Atoi(value)
	if err != nil || killValue <= 0 {
		return 0, fmt.Errorf("%s %s has an invalid %s label %q: expected a whole number greater than zero", v.kind, v.name, config.KillValueLabelKey, value)
	}

	return killValue, nil
}

// Pods returns the pods belonging to the victim
func (v *Victim) Pods(clientset kube.Interface) ([]corev1.Pod, error) {
	// An empty selector matches every pod in the namespace, so refuse to send one
	if v.podSelector == nil || v.podSelector.Empty() {
		return nil, fmt.Errorf("%s %s has no selector to find its pods with", v.kind, v.name)
	}

	listOpts := metav1.ListOptions{LabelSelector: v.podSelector.String()}
	podlist, err := clientset.CoreV1().Pods(v.namespace).List(context.TODO(), listOpts)
	if err != nil {
		return nil, err
	}

	return podlist.Items, nil
}

// RunningPods returns the pods belonging to the victim that are running
func (v *Victim) RunningPods(clientset kube.Interface) ([]corev1.Pod, error) {
	pods, err := v.Pods(clientset)
	if err != nil {
		return nil, err
	}

	var running []corev1.Pod
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodRunning {
			running = append(running, pod)
		}
	}

	return running, nil
}

// DeletePod removes the named pod
func (v *Victim) DeletePod(clientset kube.Interface, podName string) error {
	if config.DryRun() {
		glog.Infof("[DryRun Mode] Terminated pod %s for %s/%s", podName, v.namespace, v.name)
		return nil
	}

	gracePeriod := config.GracePeriodSeconds()
	deleteOpts := metav1.DeleteOptions{GracePeriodSeconds: &gracePeriod}
	if err := clientset.CoreV1().Pods(v.namespace).Delete(context.TODO(), podName, deleteOpts); err != nil {
		return err
	}

	metrics.RecordPodTermination(v.kind, v.namespace, v.name)
	return nil
}

// DeleteRandomPods removes killNum of the victim's running pods, picked at
// random. Asking for more pods than are running kills all of them.
func (v *Victim) DeleteRandomPods(clientset kube.Interface, killNum int) error {
	switch {
	case killNum < 0:
		return fmt.Errorf("cannot request negative terminations %d for %s %s", killNum, v.kind, v.name)
	case killNum == 0:
		return fmt.Errorf("no terminations requested for %s %s", v.kind, v.name)
	}

	pods, err := v.RunningPods(clientset)
	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("%s %s has no running pods at the moment", v.kind, v.name)
	}

	if killNum > len(pods) {
		glog.Warningf("%s %s has only %d currently running pods, but %d terminations requested", v.kind, v.name, len(pods), killNum)
		killNum = len(pods)
	}
	glog.V(6).Infof("Killing %d of the %d running pods for %s %s", killNum, len(pods), v.kind, v.name)

	// Deleting a pod that is already going away succeeds but kills nothing extra,
	// so take the victims off a shuffled list to keep every pick a different pod
	rand.Shuffle(len(pods), func(i, j int) { pods[i], pods[j] = pods[j], pods[i] })

	for _, pod := range pods[:killNum] {
		glog.V(6).Infof("Terminating pod %s for %s %s/%s\n", pod.Name, v.kind, v.namespace, v.name)

		if err := v.DeletePod(clientset, pod.Name); err != nil {
			return err
		}
	}

	return nil
}

// KillNumberForKillingAll returns the number of pods to kill when every running
// pod should go
func (v *Victim) KillNumberForKillingAll(clientset kube.Interface) (int, error) {
	return v.numberOfRunningPods(clientset)
}

// KillNumberForFixedPercentage returns the number of pods making up the given
// percentage of the running pods
func (v *Victim) KillNumberForFixedPercentage(clientset kube.Interface, killPercentage int) (int, error) {
	return v.killNumberForPercentage(clientset, killPercentage)
}

// KillNumberForMaxPercentage returns the number of pods making up a percentage
// of the running pods drawn at random between 0 and maxPercentage
func (v *Victim) KillNumberForMaxPercentage(clientset kube.Interface, maxPercentage int) (int, error) {
	if err := validPercentage(maxPercentage); err != nil {
		return 0, err
	}

	// +1 because IntN draws from [0,n) and the range is meant to include maxPercentage
	return v.killNumberForPercentage(clientset, rand.IntN(maxPercentage+1))
}

func (v *Victim) killNumberForPercentage(clientset kube.Interface, killPercentage int) (int, error) {
	if err := validPercentage(killPercentage); err != nil {
		return 0, err
	}
	if killPercentage == 0 {
		glog.V(6).Infof("Not terminating any pods for %s %s as kill percentage is 0", v.kind, v.name)
		return 0, nil
	}

	numRunningPods, err := v.numberOfRunningPods(clientset)
	if err != nil {
		return 0, err
	}

	return int(math.Round(float64(numRunningPods) * float64(killPercentage) / 100)), nil
}

func validPercentage(percentage int) error {
	if percentage < 0 || percentage > 100 {
		return fmt.Errorf("percentage value of %d is invalid. Must be [0-100]", percentage)
	}
	return nil
}

func (v *Victim) numberOfRunningPods(clientset kube.Interface) (int, error) {
	pods, err := v.RunningPods(clientset)
	if err != nil {
		return 0, fmt.Errorf("failed to get running pods for victim %s %s: %w", v.kind, v.name, err)
	}

	return len(pods), nil
}

// IsBlacklisted checks if this victim is blacklisted
func (v *Victim) IsBlacklisted() bool {
	return config.IsBlacklistedNamespace(v.namespace)
}

// IsWhitelisted checks if this victim is whitelisted
func (v *Victim) IsWhitelisted() bool {
	return config.IsWhitelistedNamespace(v.namespace)
}

// IdentifierSelector matches the pods carrying the given identifier label
func IdentifierSelector(identifier string) labels.Selector {
	return labels.SelectorFromSet(labels.Set{config.IdentLabelKey: identifier})
}

// NewPodSelector works out how to find the pods belonging to a victim.
//
// The identifier label is used whenever the pod template carries it, so several
// workloads can share one identifier and be treated as a single pool of pods.
// A pod template without the label falls back to the workload's own pod
// selector, which lets an app opt in to chaos using only the labels on its
// metadata.
func NewPodSelector(kind, name, identifier string, podTemplateLabels map[string]string, workloadSelector *metav1.LabelSelector) (labels.Selector, error) {
	if templateIdentifier, ok := podTemplateLabels[config.IdentLabelKey]; ok {
		if templateIdentifier != identifier {
			glog.Warningf("%s %s has conflicting %s labels: %q on the metadata and %q on the pod template. Pods are matched on the metadata value, so this will most likely find no pods", kind, name, config.IdentLabelKey, identifier, templateIdentifier)
		}
		return IdentifierSelector(identifier), nil
	}

	// An absent or empty selector converts to one that matches everything, which
	// would put every pod in the namespace at risk
	if workloadSelector == nil || len(workloadSelector.MatchLabels)+len(workloadSelector.MatchExpressions) == 0 {
		return nil, fmt.Errorf("%s %s has no %s label on its pod template and no pod selector to fall back on", kind, name, config.IdentLabelKey)
	}

	selector, err := metav1.LabelSelectorAsSelector(workloadSelector)
	if err != nil {
		return nil, fmt.Errorf("%s %s has an unusable pod selector: %w", kind, name, err)
	}

	return selector, nil
}
