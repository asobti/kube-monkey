package deployments

import (
	"fmt"
	"strconv"
	
	"github.com/asobti/kube-monkey/config"
	
	kube "k8s.io/client-go/kubernetes"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Deployment struct {
	name       string
	namespace  string
	identifier string
	mtbf       int
}

// Create a new instance of Deployment
func New(dep *v1beta1.Deployment) (*Deployment, error) {
	ident, err := identifier(dep)
	if err != nil {
		return nil, err
	}
	mtbf, err := meanTimeBetweenFailures(dep)
	if err != nil {
		return nil, err
	}

	return &Deployment{
		name:       dep.Name,
		namespace:  dep.Namespace,
		identifier: ident,
		mtbf:       mtbf,
	}, nil
}

// Returns the value of the label defined by config.IdentLabelKey
// from the deployment labels
// This label should be unique to a deployment, and is used to
// identify the pods that belong to this deployment, as pods
// inherit labels from the Deployment
func identifier(kubedep *v1beta1.Deployment) (string, error) {
	identifier, ok := kubedep.Labels[config.IdentLabelKey]
	if !ok {
		return "", fmt.Errorf("Deployment %s does not have %s label", kubedep.Name, config.IdentLabelKey)
	}
	return identifier, nil
}

// Read the mean-time-between-failures value defined by the Deployment
// in the label defined by config.MtbfLabelKey
func meanTimeBetweenFailures(kubedep *v1beta1.Deployment) (int, error) {
	mtbf, ok := kubedep.Labels[config.MtbfLabelKey]
	if !ok {
		return -1, fmt.Errorf("Deployment %s does not have %s label", kubedep.Name, config.MtbfLabelKey)
	}

	mtbfInt, err := strconv.Atoi(mtbf)
	if err != nil {
		return -1, err
	}

	if !(mtbfInt > 0) {
		return -1, fmt.Errorf("Invalid value for label %s: %d", config.MtbfLabelKey, mtbfInt)
	}

	return mtbfInt, nil
}

func (d *Deployment) Name() string {
	return d.name
}

func (d *Deployment) Namespace() string {
	return d.namespace
}

func (d *Deployment) Mtbf() int {
	return d.mtbf
}

// Returns a list of running pods for the deployment
func (d *Deployment) RunningPods(clientset *kube.Clientset) ([]v1.Pod, error) {
	runningPods := []v1.Pod{}

	pods, err := d.Pods(clientset)
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

// Returns a list of pods under the Deployment
func (d *Deployment) Pods(clientset *kube.Clientset) ([]v1.Pod, error) {
	labelSelector, err := d.LabelFilterForPods()
	if err != nil {
		return nil, err
	}

	podlist, err := clientset.Core().Pods(d.namespace).List(*labelSelector)
	if err != nil {
		return nil, err
	}
	return podlist.Items, nil
}

func (d *Deployment) DeletePod(clientset *kube.Clientset, podName string) error {
	deleteopts := &meta_v1.DeleteOptions{
		GracePeriodSeconds: config.GracePeriodSeconds(),
	}

	return clientset.Core().Pods(d.namespace).Delete(podName, deleteopts)
}

// Create a label filter to filter only for pods that belong to the this
// deployment. This is done using the identifier label
func (d *Deployment) LabelFilterForPods() (*meta_v1.ListOptions, error) {
	req, err := d.LabelRequirementForPods()
	if err != nil {
		return nil, err
	}
	labelFilter := &meta_v1.ListOptions{
		LabelSelector: labels.NewSelector().Add(*req).String(),
	}
	return labelFilter, nil
}

// Create a labels.Requirement that can be used to build a filter
func (d *Deployment) LabelRequirementForPods() (*labels.Requirement, error) {
	return labels.NewRequirement(config.IdentLabelKey, selection.Equals, sets.NewString(d.identifier).UnsortedList())
}

// Checks if the deployment is enrolled in kube-monkey
func (d *Deployment) IsEnrolled(clientset *kube.Clientset) (bool, error) {
	deployment, err := clientset.ExtensionsV1beta1().Deployments(d.namespace).Get(d.name, meta_v1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return deployment.Labels[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

func (d * Deployment) HasKillAll(clientset *kube.Clientset) (bool, error) {
	deployment, err := clientset.ExtensionsV1beta1().Deployments(d.namespace).Get(d.name, meta_v1.GetOptions{})
	if err != nil {
		// Ran into some error: return 'false' for killAll to be safe
		return false, nil
	}

	return deployment.Labels[config.KillAllLabelKey] == config.KillAllLabelValue, nil
}

// Checks if this deployment is blacklisted
func (d *Deployment) IsBlacklisted(blacklist sets.String) bool {
	return blacklist.Has(d.namespace)
}
