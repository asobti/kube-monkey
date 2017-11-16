package deployments

import (
	"github.com/golang/glog"
	
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/kubernetes"
	
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/pkg/selection"
	"k8s.io/client-go/1.5/pkg/util/sets"
)

func EligibleDeployments() ([]*Deployment, error) {
	blacklist := config.BlacklistedNamespaces()
	eligibleDeployments := []*Deployment{}

	enabledDeployments, err := EnrolledDeployments()
	if err != nil {
		return nil, err
	}

	for _, dep := range enabledDeployments {
		deployment, err := New(&dep)
		if err != nil {
			glog.V(3).Infof("Skipping eligible deployment %s because of error:\n%s\n", dep.Name, err.Error())
			continue
		}

		if deployment.IsBlacklisted(blacklist) {
			continue
		}

		eligibleDeployments = append(eligibleDeployments, deployment)
	}

	return eligibleDeployments, nil
}

func EnrolledDeployments() ([]v1beta1.Deployment, error) {
	client, err := kubernetes.NewInClusterClient()
	if err != nil {
		return nil, err
	}

	filter, err := EnrollmentFilter()
	if err != nil {
		return nil, err
	}

	deployments, err := client.Extensions().Deployments(api.NamespaceAll).List(*filter)
	if err != nil {
		return nil, err
	}
	return deployments.Items, nil
}

func EnrollmentFilter() (*api.ListOptions, error) {
	req, err := EnrollmentRequirement()
	if err != nil {
		return nil, err
	}
	return &api.ListOptions{
		LabelSelector: labels.NewSelector().Add(*req),
	}, nil
}

func EnrollmentRequirement() (*labels.Requirement, error) {
	return labels.NewRequirement(config.EnabledLabelKey, selection.Equals, sets.NewString(config.EnabledLabelValue))
}
