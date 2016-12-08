package deployments

import (
	"fmt"
	"github.com/asobti/kube-monkey/config"
	kube "k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/pkg/selection"
	"k8s.io/client-go/1.5/pkg/util/sets"
	"strconv"
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
func (d *Deployment) RunningPods(client *kube.Clientset) ([]v1.Pod, error) {
	runningPods := []v1.Pod{}

	pods, err := d.Pods(client)
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
func (d *Deployment) Pods(client *kube.Clientset) ([]v1.Pod, error) {
	labelSelector, err := d.LabelFilterForPods()
	if err != nil {
		return nil, err
	}

	podlist, err := client.Core().Pods(d.namespace).List(*labelSelector)
	if err != nil {
		return nil, err
	}
	return podlist.Items, nil
}

// Create a label filter to filter only for pods that belong to the this
// deployment. This is done using the identifier label
func (d *Deployment) LabelFilterForPods() (*api.ListOptions, error) {
	req, err := d.LabelRequirementForPods()
	if err != nil {
		return nil, err
	}
	labelFilter := &api.ListOptions{
		LabelSelector: labels.NewSelector().Add(*req),
	}
	return labelFilter, nil
}

// Create a labels.Requirement that can be used to build a filter
func (d *Deployment) LabelRequirementForPods() (*labels.Requirement, error) {
	return labels.NewRequirement(config.IdentLabelKey, selection.Equals, sets.NewString(d.identifier))
}

// Checks if the deployment is enrolled in kube-monkey
func (d *Deployment) IsEnrolled(client *kube.Clientset) (bool, error) {
	deployment, err := client.Extensions().Deployments(d.namespace).Get(d.name)
	if err != nil {
		return false, nil
	}
	return deployment.Labels[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

// Checks if this deployment is blacklisted
func (d *Deployment) IsBlacklisted(blacklist sets.String) bool {
	return blacklist.Has(d.namespace)
}
