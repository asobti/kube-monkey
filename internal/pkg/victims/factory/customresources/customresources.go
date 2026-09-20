/*
Package customresources makes any custom resource listed in the config a victim.

A custom resource is owned by an operator, which creates the pods and keeps them
running. So unlike a deployment there is no pod template to read a pod selector
off, and the pods have to be found through a label the operator sets.
*/
package customresources

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/victims"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/dynamic"
	kube "k8s.io/client-go/kubernetes"
)

// EligibleCustomResources returns the custom resources of one configured kind
// that opted in, as victims
func EligibleCustomResources(client dynamic.Interface, resource config.CustomResource, namespace string, filter metav1.ListOptions) ([]*victims.Victim, error) {
	list, err := client.Resource(groupVersionResource(resource)).Namespace(namespace).List(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	eligible := make([]*victims.Victim, 0, len(list.Items))
	for i := range list.Items {
		victim, err := New(client, resource, &list.Items[i])
		if err != nil {
			// One resource missing a label does not cost the rest their schedule
			glog.Warningf("Skipping eligible victim: %s", err)
			continue
		}
		eligible = append(eligible, victim)
	}

	return eligible, nil
}

// New turns a custom resource into a victim
func New(client dynamic.Interface, resource config.CustomResource, obj *unstructured.Unstructured) (*victims.Victim, error) {
	kind := resource.Name()
	name, namespace := obj.GetName(), obj.GetNamespace()

	spec, err := victims.SpecFromLabels(kind, name, namespace, obj.GetLabels())
	if err != nil {
		return nil, err
	}

	spec.PodSelector, err = newPodSelector(kind, resource, name, spec.Identifier)
	if err != nil {
		return nil, err
	}

	// The typed clientset the other victim kinds are handed has no Go type for a
	// custom resource, so the dynamic client that can read it is captured here
	gvr := groupVersionResource(resource)
	spec.CurrentLabels = func(kube.Interface) (map[string]string, error) {
		current, err := client.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return current.GetLabels(), nil
	}

	return victims.New(spec), nil
}

// newPodSelector works out how to find the pods the operator created for this
// resource.
//
// The configured label is preferred, because an operator names the pods it owns
// after the resource they belong to. Without one the pods have to carry the
// kube-monkey identifier label themselves, which usually means the operator was
// asked to pass the resource's labels down.
func newPodSelector(kind string, resource config.CustomResource, name, identifier string) (labels.Selector, error) {
	if resource.PodLabel == "" {
		return victims.IdentifierSelector(identifier), nil
	}

	req, err := labels.NewRequirement(resource.PodLabel, selection.Equals, []string{name})
	if err != nil {
		return nil, fmt.Errorf("%s %s cannot be matched on label %s: %w", kind, name, resource.PodLabel, err)
	}

	return labels.NewSelector().Add(*req), nil
}

// groupVersionResource turns a config entry into the coordinates the dynamic
// client asks the API server for
func groupVersionResource(resource config.CustomResource) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    resource.Group,
		Version:  resource.Version,
		Resource: resource.Resource,
	}
}
