package statefulsets

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

// Get all eligible statefulsets that opted in (filtered by config.EnabledLabel)
func EligibleStatefulSets(clientset kube.Interface, namespace string, filter *metav1.ListOptions) (eligVictims []victims.Victim, err error) {
	enabledVictims, err := clientset.AppsV1beta1().StatefulSets(namespace).List(*filter)
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

// Checks if the statefulset is currently enrolled in kube-monkey
func (ss *StatefulSet) IsEnrolled(clientset kube.Interface) (bool, error) {
	statefulset, err := clientset.AppsV1beta1().StatefulSets(ss.Namespace()).Get(ss.Name(), metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return statefulset.Labels[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

// Returns current killtype config label for update
func (ss *StatefulSet) KillType(clientset kube.Interface) (string, error) {
	statefulset, err := clientset.AppsV1beta1().StatefulSets(ss.Namespace()).Get(ss.Name(), metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	killType, ok := statefulset.Labels[config.KillTypeLabelKey]
	if !ok {
		return "", fmt.Errorf("%s %s does not have %s label", ss.Kind(), ss.Name(), config.KillTypeLabelKey)
	}

	return killType, nil
}

// Returns current killvalue config label for update
func (ss *StatefulSet) KillValue(clientset kube.Interface) (int, error) {
	statefulset, err := clientset.AppsV1beta1().StatefulSets(ss.Namespace()).Get(ss.Name(), metav1.GetOptions{})
	if err != nil {
		return -1, err
	}

	killMode, ok := statefulset.Labels[config.KillValueLabelKey]
	if !ok {
		return -1, fmt.Errorf("%s %s does not have %s label", ss.Kind(), ss.Name(), config.KillValueLabelKey)
	}

	killModeInt, err := strconv.Atoi(killMode)
	if !(killModeInt > 0) {
		return -1, fmt.Errorf("Invalid value for label %s: %d", config.KillValueLabelKey, killModeInt)
	}

	return killModeInt, nil
}
