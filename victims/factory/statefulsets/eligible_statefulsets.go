package statefulsets

//All these functions require api access specific to the version of the app

import (
	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims"

	kube "k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Get all eligible statefulsets that opted in (filtered by config.EnabledLabel)
func EligibleStatefulSets(clientset *kube.Clientset, filter *metav1.ListOptions) (eligVictims []victims.Victim, err error) {
	enabledVictims, err := clientset.AppsV1beta1().StatefulSets(metav1.NamespaceAll).List(*filter)
	if err != nil {
		return nil, err
	}

	for _, vic := range enabledVictims.Items {
		victim, err := New(&vic)
		if err != nil {
			glog.Warningf("Skipping eligible %T %s because of error: %s", vic, vic.Name, err.Error())
			continue
		}

		if victim.IsBlacklisted() {
			continue
		}

		eligVictims = append(eligVictims, victim)
	}

	return
}

/* Below methods are used to verify the victim's attributes have not changed at the scheduled time of termination */

// Checks if the statefulset is currently enrolled in kube-monkey
func (ss *StatefulSet) IsEnrolled(clientset *kube.Clientset) (bool, error) {
	statefulset, err := clientset.AppsV1beta1().StatefulSets(ss.Namespace()).Get(ss.Name(), metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return statefulset.Labels[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

// Checks if the statefulset is flagged for killall at this time
func (ss *StatefulSet) HasKillAll(clientset *kube.Clientset) (bool, error) {
	statefulset, err := clientset.AppsV1beta1().StatefulSets(ss.Namespace()).Get(ss.Name(), metav1.GetOptions{})
	if err != nil {
		// Ran into some error: return 'false' for killAll to be safe
		return false, nil
	}

	return statefulset.Labels[config.KillAllLabelKey] == config.KillAllLabelValue, nil
}
