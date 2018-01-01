package deployments

import (
	"fmt"
	"strconv"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims"

	kube "k8s.io/client-go/kubernetes"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Deployment struct {
	*victims.VictimBase
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

	return &Deployment{victims.New("Deployment", dep.Name, dep.Namespace, ident, mtbf)}, nil
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
	labelSelector, err := victims.LabelFilterForPods(d.Identifier())
	if err != nil {
		return nil, err
	}

	podlist, err := clientset.CoreV1().Pods(d.Namespace()).List(*labelSelector)
	if err != nil {
		return nil, err
	}
	return podlist.Items, nil
}

func (d *Deployment) DeletePod(clientset *kube.Clientset, podName string) error {
	deleteopts := &metav1.DeleteOptions{
		GracePeriodSeconds: config.GracePeriodSeconds(),
	}

	return clientset.CoreV1().Pods(d.Namespace()).Delete(podName, deleteopts)
}

// Checks if the deployment is enrolled in kube-monkey
func (d *Deployment) IsEnrolled(clientset *kube.Clientset) (bool, error) {
	deployment, err := clientset.ExtensionsV1beta1().Deployments(d.Namespace()).Get(d.Name(), metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return deployment.Labels[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

func (d *Deployment) HasKillAll(clientset *kube.Clientset) (bool, error) {
	deployment, err := clientset.ExtensionsV1beta1().Deployments(d.Namespace()).Get(d.Name(), metav1.GetOptions{})
	if err != nil {
		// Ran into some error: return 'false' for killAll to be safe
		return false, nil
	}

	return deployment.Labels[config.KillAllLabelKey] == config.KillAllLabelValue, nil
}

// Checks if this deployment is blacklisted
func (d *Deployment) IsBlacklisted(blacklist sets.String) bool {
	return blacklist.Has(d.Namespace())
}
