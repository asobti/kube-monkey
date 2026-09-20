package customresources

//All these functions require api access specific to the custom resource

import (
	"context"
	"fmt"
	"strconv"

	"github.com/golang/glog"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/victims"

	"k8s.io/client-go/dynamic"
	kube "k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// EligibleCustomResources gets all custom resources of one configured kind that
// opted in (filtered by config.EnabledLabel)
func EligibleCustomResources(client dynamic.Interface, resource config.CustomResource, namespace string, filter *metav1.ListOptions) (eligVictims []victims.Victim, err error) {
	enabledVictims, err := client.Resource(GroupVersionResource(resource)).Namespace(namespace).List(context.TODO(), *filter)
	if err != nil {
		return nil, err
	}

	for i := range enabledVictims.Items {
		vic := &enabledVictims.Items[i]
		victim, err := New(client, resource, vic)
		if err != nil {
			glog.Warningf("Skipping eligible %s %s because of error: %s", resource.Name(), vic.GetName(), err.Error())
			continue
		}

		eligVictims = append(eligVictims, victim)
	}

	return
}

/* Below methods are used to verify the victim's attributes have not changed at the scheduled time of termination */

// IsEnrolled checks if the custom resource is currently enrolled in kube-monkey
func (cr *CustomResource) IsEnrolled(kube.Interface) (bool, error) {
	obj, err := cr.get()
	if err != nil {
		return false, err
	}
	return obj.GetLabels()[config.EnabledLabelKey] == config.EnabledLabelValue, nil
}

// KillType returns current killtype config label for update
func (cr *CustomResource) KillType(kube.Interface) (string, error) {
	obj, err := cr.get()
	if err != nil {
		return "", err
	}

	killType, ok := obj.GetLabels()[config.KillTypeLabelKey]
	if !ok {
		return "", fmt.Errorf("%s %s does not have %s label", cr.Kind(), cr.Name(), config.KillTypeLabelKey)
	}

	return killType, nil
}

// KillValue returns current killvalue config label for update
func (cr *CustomResource) KillValue(kube.Interface) (int, error) {
	obj, err := cr.get()
	if err != nil {
		return -1, err
	}

	killMode, ok := obj.GetLabels()[config.KillValueLabelKey]
	if !ok {
		return -1, fmt.Errorf("%s %s does not have %s label", cr.Kind(), cr.Name(), config.KillValueLabelKey)
	}

	killModeInt, err := strconv.Atoi(killMode)
	if err != nil || !(killModeInt > 0) {
		return -1, fmt.Errorf("Invalid value for label %s: %q", config.KillValueLabelKey, killMode)
	}

	return killModeInt, nil
}

func (cr *CustomResource) get() (*unstructured.Unstructured, error) {
	return cr.client.Resource(cr.gvr).Namespace(cr.Namespace()).Get(context.TODO(), cr.Name(), metav1.GetOptions{})
}
