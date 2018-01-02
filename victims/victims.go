package victims

import (
	"fmt"
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
	VictimSpecificApiCalls
}

type VictimBaseTemplate interface {
	// Get value methods
	Kind() string
	Name() string
	Namespace() string
	Identifier() string
	Mtbf() int

	VictimApiCalls
}

type VictimSpecificApiCalls interface {
	// Depends on which version i.e. apps/v1 or extensions/v1beta2
	IsEnrolled(*kube.Clientset) (bool, error)
	HasKillAll(*kube.Clientset) (bool, error)
}

type VictimApiCalls interface {
	// Exposed Api Calls
	RunningPods(*kube.Clientset) ([]v1.Pod, error)
	Pods(*kube.Clientset) ([]v1.Pod, error)
	DeletePod(*kube.Clientset, string) error
	DeleteRandomPod(*kube.Clientset) error
	IsBlacklisted() bool
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
func (v *VictimBase) RunningPods(clientset *kube.Clientset) (runningPods []v1.Pod, err error) {
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
func (v *VictimBase) Pods(clientset *kube.Clientset) ([]v1.Pod, error) {
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
func (v *VictimBase) DeletePod(clientset *kube.Clientset, podName string) error {
	deleteopts := &metav1.DeleteOptions{
		GracePeriodSeconds: config.GracePeriodSeconds(),
	}

	return clientset.CoreV1().Pods(v.namespace).Delete(podName, deleteopts)
}

// Remove a random pod for the victim
func (v *VictimBase) DeleteRandomPod(clientset *kube.Clientset) error {
	// Pick a target pod to delete
	pods, err := v.RunningPods(clientset)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("%s %s has no running pods at the moment", v.kind, v.name)
	}

	targetPod := RandomPodName(pods)

	glog.Errorf("Terminating pod %s for %s %s\n", targetPod, v.kind, v.name)
	return v.DeletePod(clientset, targetPod)
}

// Check if this victim is blacklisted
func (v *VictimBase) IsBlacklisted() bool {
	blacklist := config.BlacklistedNamespaces()
	return blacklist.Has(v.namespace)
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

// Pick a random pod name from a list of Pods
func RandomPodName(pods []v1.Pod) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	randIndex := r.Intn(len(pods))
	return pods[randIndex].Name
}
