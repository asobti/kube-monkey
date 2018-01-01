package deployments

import (
	"github.com/golang/glog"
	
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/kubernetes"
	
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			glog.V(1).Infof("Skipping eligible deployment %s because of error:\n%s\n", dep.Name, err.Error())
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
	clientset, err := kubernetes.CreateClient()
	if err != nil {
		return nil, err
	}

	filter, err := EnrollmentFilter()
	if err != nil {
		return nil, err
	}

	deployments, err := clientset.ExtensionsV1beta1().Deployments(meta_v1.NamespaceAll).List(*filter)
	if err != nil {
		return nil, err
	}
	return deployments.Items, nil
}

func EnrollmentFilter() (*meta_v1.ListOptions, error) {
	req, err := EnrollmentRequirement()
	if err != nil {
		return nil, err
	}
	return &meta_v1.ListOptions{
		LabelSelector: labels.NewSelector().Add(*req).String(),
	}, nil
}

func EnrollmentRequirement() (*labels.Requirement, error) {
	return labels.NewRequirement(config.EnabledLabelKey, selection.Equals, sets.NewString(config.EnabledLabelValue).UnsortedList())
}
