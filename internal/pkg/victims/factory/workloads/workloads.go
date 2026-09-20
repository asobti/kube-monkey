/*
Package workloads makes the built in kubernetes workload kinds - deployments,
statefulsets and daemonsets - victims.

The three differ only in the API call that reaches them. Everything else, from
the labels kube-monkey reads to the way the pods are found, is the same.
*/
package workloads

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	"kube-monkey/internal/pkg/victims"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
)

// EligibleDeployments returns the deployments that opted in, as victims.
// The filter narrows the list to the workloads carrying the enabled label.
func EligibleDeployments(clientset kube.Interface, namespace string, filter metav1.ListOptions) ([]*victims.Victim, error) {
	list, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	return eligible(list.Items, NewDeployment), nil
}

// EligibleStatefulSets returns the statefulsets that opted in, as victims
func EligibleStatefulSets(clientset kube.Interface, namespace string, filter metav1.ListOptions) ([]*victims.Victim, error) {
	list, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	return eligible(list.Items, NewStatefulSet), nil
}

// EligibleDaemonSets returns the daemonsets that opted in, as victims
func EligibleDaemonSets(clientset kube.Interface, namespace string, filter metav1.ListOptions) ([]*victims.Victim, error) {
	list, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	return eligible(list.Items, NewDaemonSet), nil
}

// NewDeployment turns a deployment into a victim
func NewDeployment(deployment *appsv1.Deployment) (*victims.Victim, error) {
	return newVictim(
		kindOf(*deployment),
		deployment.ObjectMeta,
		deployment.Spec.Template.Labels,
		deployment.Spec.Selector,
		func(clientset kube.Interface, namespace, name string) (map[string]string, error) {
			current, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return current.Labels, nil
		},
	)
}

// NewStatefulSet turns a statefulset into a victim
func NewStatefulSet(statefulset *appsv1.StatefulSet) (*victims.Victim, error) {
	return newVictim(
		kindOf(*statefulset),
		statefulset.ObjectMeta,
		statefulset.Spec.Template.Labels,
		statefulset.Spec.Selector,
		func(clientset kube.Interface, namespace, name string) (map[string]string, error) {
			current, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return current.Labels, nil
		},
	)
}

// NewDaemonSet turns a daemonset into a victim
func NewDaemonSet(daemonset *appsv1.DaemonSet) (*victims.Victim, error) {
	return newVictim(
		kindOf(*daemonset),
		daemonset.ObjectMeta,
		daemonset.Spec.Template.Labels,
		daemonset.Spec.Selector,
		func(clientset kube.Interface, namespace, name string) (map[string]string, error) {
			current, err := clientset.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return current.Labels, nil
		},
	)
}

// newVictim reads the kube-monkey labels off a workload and works out how to
// find its pods. readLabels is how the workload is fetched again at kill time.
func newVictim(
	kind string,
	meta metav1.ObjectMeta,
	podTemplateLabels map[string]string,
	workloadSelector *metav1.LabelSelector,
	readLabels func(clientset kube.Interface, namespace, name string) (map[string]string, error),
) (*victims.Victim, error) {
	spec, err := victims.SpecFromLabels(kind, meta.Name, meta.Namespace, meta.Labels)
	if err != nil {
		return nil, err
	}

	spec.PodSelector, err = victims.NewPodSelector(kind, meta.Name, spec.Identifier, podTemplateLabels, workloadSelector)
	if err != nil {
		return nil, err
	}

	spec.CurrentLabels = func(clientset kube.Interface) (map[string]string, error) {
		return readLabels(clientset, meta.Namespace, meta.Name)
	}

	return victims.New(spec), nil
}

// eligible turns the workloads that opted in into victims. A workload that is
// not set up correctly is skipped on its own, so one bad workload does not cost
// the rest of the cluster its schedule.
func eligible[T any](items []T, newVictim func(*T) (*victims.Victim, error)) []*victims.Victim {
	eligible := make([]*victims.Victim, 0, len(items))

	for i := range items {
		victim, err := newVictim(&items[i])
		if err != nil {
			glog.Warningf("Skipping eligible victim: %s", err)
			continue
		}
		eligible = append(eligible, victim)
	}

	return eligible
}

// kindOf names the kind the way kube-monkey reports it, e.g. "v1.Deployment"
func kindOf(workload any) string {
	return fmt.Sprintf("%T", workload)
}
