package deployments

//All these functions require api access specific to the version of the app

import (
	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims"

	kube "k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Get all eligible deployments that opted in (filtered by config.EnabledLabel)
func EligibleDeployments(clientset *kube.Clientset, namespace string, filter *metav1.ListOptions) (eligVictims []victims.Victim, err error) {
	enabledVictims, err := clientset.ExtensionsV1beta1().Deployments(namespace).List(*filter)
	if err != nil {
		return nil, err
	}

	for _, vic := range enabledVictims.Items {
		victim, err := New(&vic)
		if err != nil {
			glog.Warningf("Skipping eligible %T %s because of error: %s", vic, vic.Name, err.Error())
			continue
		}

		// TODO: After generating whitelisting ns list, this will move to factory.
		// IsBlacklisted will change to something like IsAllowedNamespace
		// and will only be used to verify at time of scheduled execution
		if victim.IsBlacklisted() {
			continue
		}

		eligVictims = append(eligVictims, victim)
	}

	return
}

/* Below methods are used to verify the victim's attributes have not changed at the scheduled time of termination */

// Checks if the deployment is currently enrolled in kube-monkey
func (d *Deployment) IsEnrolled(clientset *kube.Clientset) (bool, error) {
	deployment, err := clientset.ExtensionsV1beta1().Deployments(d.Namespace()).Get(d.Name(), metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return deployment.Labels[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

// Checks if the deployment is flagged for killall at this time
func (d *Deployment) HasKillAll(clientset *kube.Clientset) (bool, error) {
	deployment, err := clientset.ExtensionsV1beta1().Deployments(d.Namespace()).Get(d.Name(), metav1.GetOptions{})
	if err != nil {
		// Ran into some error: return 'false' for killAll to be safe
		return false, nil
	}

	return deployment.Labels[config.KillAllLabelKey] == config.KillAllLabelValue, nil
}
