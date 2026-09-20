/*
Package factory is responsible for generating eligible victim kinds

New types of kinds can be added easily
*/
package factory

import (
	"github.com/golang/glog"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/kubernetes"
	"kube-monkey/internal/pkg/victims"
	"kube-monkey/internal/pkg/victims/factory/daemonsets"
	"kube-monkey/internal/pkg/victims/factory/deployments"
	"kube-monkey/internal/pkg/victims/factory/statefulsets"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
)

// EligibleVictims gathers list of enabled/enrolled kinds for judgement by
// the scheduler
//
// The namespace lists hold patterns rather than names, so they cannot be turned
// into API paths. Each kind is fetched across the whole cluster in one call and
// the namespace lists are applied to the result. The enrollment filter already
// narrows the fetch to opted-in workloads, which keeps the response small.
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

	// Fetch deployments
	deployments, err := deployments.EligibleDeployments(clientset, metav1.NamespaceAll, filter)
	if err != nil {
		// Allow pass through to schedule other kinds. A failure here is worth
		// shouting about because it leaves the schedule empty for this kind
		// across the whole cluster
		glog.Errorf("Failed to fetch eligible deployments due to error: %s", err.Error())
	}
	eligibleVictims = append(eligibleVictims, deployments...)

	// Fetch statefulsets
	statefulsets, err := statefulsets.EligibleStatefulSets(clientset, metav1.NamespaceAll, filter)
	if err != nil {
		// Allow pass through to schedule other kinds. A failure here is worth
		// shouting about because it leaves the schedule empty for this kind
		// across the whole cluster
		glog.Errorf("Failed to fetch eligible statefulsets due to error: %s", err.Error())
	}
	eligibleVictims = append(eligibleVictims, statefulsets...)

	// Fetch daemonsets
	daemonsets, err := daemonsets.EligibleDaemonSets(clientset, metav1.NamespaceAll, filter)
	if err != nil {
		// Allow pass through to schedule other kinds. A failure here is worth
		// shouting about because it leaves the schedule empty for this kind
		// across the whole cluster
		glog.Errorf("Failed to fetch eligible daemonsets due to error: %s", err.Error())
	}
	eligibleVictims = append(eligibleVictims, daemonsets...)

	return InAllowedNamespace(eligibleVictims), nil
}

// InAllowedNamespace keeps the victims whose namespace is whitelisted and not
// blacklisted. The blacklist wins where the two overlap.
func InAllowedNamespace(candidates []victims.Victim) (allowed []victims.Victim) {
	for _, victim := range candidates {
		if victim.IsBlacklisted() {
			glog.V(6).Infof("Skipping %s %s because namespace %s is blacklisted", victim.Kind(), victim.Name(), victim.Namespace())
			continue
		}

		if !victim.IsWhitelisted() {
			glog.V(6).Infof("Skipping %s %s because namespace %s is not whitelisted", victim.Kind(), victim.Name(), victim.Namespace())
			continue
		}

		allowed = append(allowed, victim)
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
