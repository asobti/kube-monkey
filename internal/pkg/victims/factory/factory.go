/*
Package factory finds the victims that have opted in to kube-monkey, so the
scheduler has something to draw a schedule from.
*/
package factory

import (
	"github.com/golang/glog"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/kubernetes"
	"kube-monkey/internal/pkg/victims"
	"kube-monkey/internal/pkg/victims/factory/customresources"
	"kube-monkey/internal/pkg/victims/factory/workloads"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	kube "k8s.io/client-go/kubernetes"
)

// EligibleVictims gathers the victims that opted in, for the scheduler to judge.
//
// The namespace lists hold patterns rather than names, so they cannot be turned
// into API paths. Each kind is fetched across the whole cluster in one call and
// the namespace lists are applied to the result. The enrollment filter already
// narrows the fetch to opted-in workloads, which keeps the response small.
func EligibleVictims() ([]*victims.Victim, error) {
	clientset, err := kubernetes.CreateClient()
	if err != nil {
		return nil, err
	}

	// Verify opt-in at scheduling time
	filter, err := enrollmentFilter()
	if err != nil {
		return nil, err
	}

	// A slice rather than a map so the schedule always lists the kinds in the
	// same order
	builtIn := []struct {
		kind string
		list func(kube.Interface, string, metav1.ListOptions) ([]*victims.Victim, error)
	}{
		{"deployments", workloads.EligibleDeployments},
		{"statefulsets", workloads.EligibleStatefulSets},
		{"daemonsets", workloads.EligibleDaemonSets},
	}

	var eligible []*victims.Victim
	for _, kind := range builtIn {
		found, err := kind.list(clientset, metav1.NamespaceAll, filter)
		if err != nil {
			// Carry on with the other kinds. A failure here is worth shouting
			// about because it leaves the schedule empty for this kind across
			// the whole cluster
			glog.Errorf("Failed to fetch eligible %s due to error: %s", kind.kind, err)
			continue
		}
		eligible = append(eligible, found...)
	}

	eligible = append(eligible, eligibleCustomResources(filter)...)

	return inAllowedNamespace(eligible), nil
}

// eligibleCustomResources fetches every custom resource kind the config names.
//
// The dynamic client is only built when there is something to use it for, so a
// config without custom resources needs no extra permissions.
func eligibleCustomResources(filter metav1.ListOptions) []*victims.Victim {
	resources := config.CustomResources()
	if len(resources) == 0 {
		return nil
	}

	client, err := kubernetes.NewDynamicClient()
	if err != nil {
		glog.Errorf("Failed to create a client for custom resources due to error: %s", err)
		return nil
	}

	var eligible []*victims.Victim
	for _, resource := range resources {
		found, err := customresources.EligibleCustomResources(client, resource, metav1.NamespaceAll, filter)
		switch {
		case apierrors.IsNotFound(err):
			// A resource nobody has installed the CRD for is worth a word but
			// not a shout, because a single config can cover a fleet of
			// clusters that do not all run the same operators
			glog.V(4).Infof("Skipping %s because the cluster does not serve it", resource.Name())
		case err != nil:
			// Anything else, a missing RBAC rule most likely, leaves the
			// schedule empty for this kind and is worth shouting about
			glog.Errorf("Failed to fetch eligible %s due to error: %s", resource.Name(), err)
		default:
			eligible = append(eligible, found...)
		}
	}

	return eligible
}

// inAllowedNamespace keeps the victims whose namespace is whitelisted and not
// blacklisted. The blacklist wins where the two overlap.
func inAllowedNamespace(candidates []*victims.Victim) []*victims.Victim {
	allowed := make([]*victims.Victim, 0, len(candidates))

	for _, victim := range candidates {
		switch {
		case victim.IsBlacklisted():
			glog.V(6).Infof("Skipping %s %s because namespace %s is blacklisted", victim.Kind(), victim.Name(), victim.Namespace())
		case !victim.IsWhitelisted():
			glog.V(6).Infof("Skipping %s %s because namespace %s is not whitelisted", victim.Kind(), victim.Name(), victim.Namespace())
		default:
			allowed = append(allowed, victim)
		}
	}

	return allowed
}

// enrollmentFilter narrows a list call to the objects carrying the enabled label
func enrollmentFilter() (metav1.ListOptions, error) {
	enrolled, err := labels.NewRequirement(config.EnabledLabelKey, selection.Equals, []string{config.EnabledLabelValue})
	if err != nil {
		return metav1.ListOptions{}, err
	}

	return metav1.ListOptions{LabelSelector: labels.NewSelector().Add(*enrolled).String()}, nil
}
