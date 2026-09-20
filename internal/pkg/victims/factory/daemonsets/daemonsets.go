package daemonsets

import (
	"fmt"
	"time"

	"kube-monkey/internal/pkg/calendar"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/victims"

	appsv1 "k8s.io/api/apps/v1"
)

type DaemonSet struct {
	*victims.VictimBase
}

// New creates a new instance of DaemonSet
func New(dep *appsv1.DaemonSet) (*DaemonSet, error) {
	ident, err := identifier(dep)
	if err != nil {
		return nil, err
	}
	mtbf, err := meanTimeBetweenFailures(dep)
	if err != nil {
		return nil, err
	}
	kind := fmt.Sprintf("%T", *dep)
	podSelector, err := victims.NewPodSelector(kind, dep.Name, ident, dep.Spec.Template.Labels, dep.Spec.Selector)
	if err != nil {
		return nil, err
	}

	return &DaemonSet{VictimBase: victims.New(kind, dep.Name, dep.Namespace, ident, mtbf, podSelector)}, nil
}

// Returns the value of the label defined by config.IdentLabelKey
// from the DaemonSet metadata labels
// This label should be unique to a DaemonSet. It also picks out the pods that
// belong to this DaemonSet when the pod template passes the label down to them
func identifier(kubekind *appsv1.DaemonSet) (string, error) {
	identifier, ok := kubekind.Labels[config.IdentLabelKey]
	if !ok {
		return "", fmt.Errorf("%T %s does not have %s label", kubekind, kubekind.Name, config.IdentLabelKey)
	}
	return identifier, nil
}

// Read the mean-time-between-failures value defined by the DaemonSet
// in the label defined by config.MtbfLabelKey
func meanTimeBetweenFailures(kubekind *appsv1.DaemonSet) (time.Duration, error) {
	mtbf, ok := kubekind.Labels[config.MtbfLabelKey]
	if !ok {
		return 0, fmt.Errorf("%T %s does not have %s label", kubekind, kubekind.Name, config.MtbfLabelKey)
	}

	duration, err := calendar.ParseMtbf(mtbf)
	if err != nil {
		return 0, fmt.Errorf("%T %s has an %s", kubekind, kubekind.Name, err)
	}

	return duration, nil
}
