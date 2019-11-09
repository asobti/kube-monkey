package deployments

//All these functions require api access specific to the version of the app

import (
	"fmt"
	"strconv"

	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims"

	kube "k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EligibleDeployments gets all eligible deployments that opted in (filtered by config.EnabledLabel)
func EligibleDeployments(clientset kube.Interface, namespace string, filter *metav1.ListOptions) (eligVictims []victims.Victim, err error) {
	enabledVictims, err := clientset.AppsV1().Deployments(namespace).List(*filter)
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

// IsEnrolled checks if the deployment is currently enrolled in kube-monkey
func (d *Deployment) IsEnrolled(clientset kube.Interface) (bool, error) {
	deployment, err := clientset.AppsV1().Deployments(d.Namespace()).Get(d.Name(), metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return deployment.Labels[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

// KillType returns current killtype config label for update
func (d *Deployment) KillType(clientset kube.Interface) (string, error) {
	deployment, err := clientset.AppsV1().Deployments(d.Namespace()).Get(d.Name(), metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	killType, ok := deployment.Labels[config.KillTypeLabelKey]
	if !ok {
		return "", fmt.Errorf("%s %s does not have %s label", d.Kind(), d.Name(), config.KillTypeLabelKey)
	}

	return killType, nil
}

// KillValue returns current killvalue config label for update
func (d *Deployment) KillValue(clientset kube.Interface) (int, error) {
	deployment, err := clientset.AppsV1().Deployments(d.Namespace()).Get(d.Name(), metav1.GetOptions{})
	if err != nil {
		return -1, err
	}

	killMode, ok := deployment.Labels[config.KillValueLabelKey]
	if !ok {
		return -1, fmt.Errorf("%s %s does not have %s label", d.Kind(), d.Name(), config.KillValueLabelKey)
	}

	killModeInt, err := strconv.Atoi(killMode)
	if err != nil || !(killModeInt > 0) {
		return -1, fmt.Errorf("Invalid value for label %s: %d", config.KillValueLabelKey, killModeInt)
	}

	return killModeInt, nil
}
