/*
Package customresources makes any custom resource listed in the config a victim.

A custom resource is owned by an operator, which creates the pods and keeps
them running. So unlike a deployment there is no pod template to read a pod
selector off, and the pods have to be found through a label the operator sets.
*/
package customresources

import (
	"fmt"
	"time"

	"kube-monkey/internal/pkg/calendar"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/victims"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/dynamic"
)

type CustomResource struct {
	*victims.VictimBase

	// The typed clientset the other victim kinds are handed has no Go type for
	// a custom resource, so this victim carries the client that can read it
	client dynamic.Interface
	gvr    schema.GroupVersionResource
}

// New creates a new instance of CustomResource
func New(client dynamic.Interface, resource config.CustomResource, obj *unstructured.Unstructured) (*CustomResource, error) {
	kind := resource.Name()

	ident, err := identifier(kind, obj)
	if err != nil {
		return nil, err
	}
	mtbf, err := meanTimeBetweenFailures(kind, obj)
	if err != nil {
		return nil, err
	}
	podSelector, err := newPodSelector(kind, resource, obj, ident)
	if err != nil {
		return nil, err
	}

	return &CustomResource{
		VictimBase: victims.New(kind, obj.GetName(), obj.GetNamespace(), ident, mtbf, podSelector),
		client:     client,
		gvr:        GroupVersionResource(resource),
	}, nil
}

// GroupVersionResource turns a config entry into the coordinates the dynamic
// client asks the API server for
func GroupVersionResource(resource config.CustomResource) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    resource.Group,
		Version:  resource.Version,
		Resource: resource.Resource,
	}
}

// newPodSelector works out how to find the pods the operator created for this
// resource.
//
// The configured label is preferred, because an operator names the pods it owns
// after the resource they belong to. Without one the pods have to carry the
// kube-monkey identifier label themselves, which usually means the operator was
// asked to pass the resource's labels down.
func newPodSelector(kind string, resource config.CustomResource, obj *unstructured.Unstructured, identifier string) (labels.Selector, error) {
	if resource.PodLabel == "" {
		return victims.IdentifierSelector(identifier), nil
	}

	req, err := labels.NewRequirement(resource.PodLabel, selection.Equals, []string{obj.GetName()})
	if err != nil {
		return nil, fmt.Errorf("%s %s cannot be matched on label %s: %w", kind, obj.GetName(), resource.PodLabel, err)
	}

	return labels.NewSelector().Add(*req), nil
}

// Returns the value of the label defined by config.IdentLabelKey
// from the custom resource metadata labels
// This label should be unique to a custom resource. It names the victim in
// metrics and notifications, and picks out its pods when no pod label is set
func identifier(kind string, obj *unstructured.Unstructured) (string, error) {
	identifier, ok := obj.GetLabels()[config.IdentLabelKey]
	if !ok {
		return "", fmt.Errorf("%s %s does not have %s label", kind, obj.GetName(), config.IdentLabelKey)
	}
	return identifier, nil
}

// Read the mean-time-between-failures value defined by the custom resource
// in the label defined by config.MtbfLabelKey
func meanTimeBetweenFailures(kind string, obj *unstructured.Unstructured) (time.Duration, error) {
	mtbf, ok := obj.GetLabels()[config.MtbfLabelKey]
	if !ok {
		return 0, fmt.Errorf("%s %s does not have %s label", kind, obj.GetName(), config.MtbfLabelKey)
	}

	duration, err := calendar.ParseMtbf(mtbf)
	if err != nil {
		return 0, fmt.Errorf("%s %s has an %s", kind, obj.GetName(), err)
	}

	return duration, nil
}
