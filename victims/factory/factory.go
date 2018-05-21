/*
Package factory is responsible for generating eligible victim kinds

New types of kinds can be added easily
*/
package factory

import (
	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/kubernetes"
	"github.com/asobti/kube-monkey/victims"
	"github.com/asobti/kube-monkey/victims/factory/daemonsets"
	"github.com/asobti/kube-monkey/victims/factory/deployments"
	"github.com/asobti/kube-monkey/victims/factory/statefulsets"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Gathers list of enabled/enrolled kinds for judgement by the scheduler
// This checks against config.WhitelistedNamespaces but
// each victim checks themselves against the ns blacklist
// TODO: fetch all namespaces from k8 apiserver to check blacklist here
func EligibleVictims() (eligibleVictims []victims.Victim, err error) {
	clientset, err := kubernetes.CreateClient()
	if err != nil {
		return nil, err
	}

	// Verify opt-in at scheduling time
	filter, err := enrollmentFilter()
	if err != nil {
		return nil, err
	}

	for _, namespace := range config.WhitelistedNamespaces().UnsortedList() {
		// Fetch deployments
		deployments, err := deployments.EligibleDeployments(clientset, namespace, filter)
		if err != nil {
			//allow pass through to schedule other kinds and namespaces
			glog.Warningf("Failed to fetch eligible deployments for namespace %s due to error: %s", namespace, err.Error())
			continue
		}
		eligibleVictims = append(eligibleVictims, deployments...)

		// Fetch statefulsets
		statefulsets, err := statefulsets.EligibleStatefulSets(clientset, namespace, filter)
		if err != nil {
			//allow pass through to schedule other kinds and namespaces
			glog.Warningf("Failed to fetch eligible statefulsets for namespace %s due to error: %s", namespace, err.Error())
			continue
		}
		eligibleVictims = append(eligibleVictims, statefulsets...)

		// Fetch daemonsets
		daemonsets, err := daemonsets.EligibleDaemonSets(clientset, namespace, filter)
		if err != nil {
			//allow pass through to schedule other kinds and namespaces
			glog.Warningf("Failed to fetch eligible daemonsets for namespace %s due to error: %s", namespace, err.Error())
			continue
		}
		eligibleVictims = append(eligibleVictims, daemonsets...)
	}

	return
}

// Verifies opt-in of victims
func enrollmentFilter() (*metav1.ListOptions, error) {
	req, err := enrollmentRequirement()
	if err != nil {
		return nil, err
	}
	return &metav1.ListOptions{
		LabelSelector: labels.NewSelector().Add(*req).String(),
	}, nil
}

func enrollmentRequirement() (*labels.Requirement, error) {
	return labels.NewRequirement(config.EnabledLabelKey, selection.Equals, sets.NewString(config.EnabledLabelValue).UnsortedList())
}
