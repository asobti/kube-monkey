/*
Package factory is responsible for generating eligible victim kinds

New types of kinds can be added easily
*/
package factory

import (
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/kubernetes"
	"github.com/asobti/kube-monkey/victims"
	"github.com/asobti/kube-monkey/victims/factory/deployments"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Gathers list of enabled/enrolled kinds for judgement by the scheduler
func EligibleVictims() (eligibleVictims []victims.Victim, err error) {
	clientset, err := kubernetes.CreateClient()
	if err != nil {
		return nil, err
	}

	filter, err := enrollmentFilter()
	if err != nil {
		return nil, err
	}

	blacklist := config.BlacklistedNamespaces()

	// Fetch deployments
	deployments, err := deployments.EligibleDeployments(clientset, filter, blacklist)
	if err != nil {
		// Should probably be a warning, allow pass through to schedule other kinds
		return nil, err
	}
	eligibleVictims = append(eligibleVictims, deployments...)

	return
}

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
