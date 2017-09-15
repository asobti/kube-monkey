package deployments

import (
	"fmt"

	"github.com/andreic92/kube-monkey/config"
	"github.com/andreic92/kube-monkey/kubernetes"
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
		if blacklist.Has(dep.Namespace) {
			continue
		}

		deployment, err := New(&dep)
		if err != nil {
			fmt.Printf("Skipping eligible deployment %s because of error:\n%s\n", dep.Name, err.Error())
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
	if config.SafeMode() {
		req, err = SafeEnrollmentRequirement()
	}
	if err != nil {
		return nil, err
	}

	return &api.ListOptions{
		LabelSelector: labels.NewSelector().Add(*req),
	}, nil
}

func EnrollmentRequirement() (*labels.Requirement, error) {
	return labels.NewRequirement(config.DisabledLabelKey, selection.DoesNotExist, nil)
}

func SafeEnrollmentRequirement() (*labels.Requirement, error) {
	return labels.NewRequirement(config.EnabledLabelKey, selection.Equals, sets.NewString(config.EnabledLabelValue))
}
