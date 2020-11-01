package daemonsets

import (
	"fmt"
	"time"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims"

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

	return &DaemonSet{VictimBase: victims.New(kind, dep.Name, dep.Namespace, ident, mtbf)}, nil
}

// Returns the value of the label defined by config.IdentLabelKey
// from the DaemonSet labels
// This label should be unique to a DaemonSet, and is used to
// identify the pods that belong to this DaemonSet, as pods
// inherit labels from the DaemonSet
func identifier(kubekind *appsv1.DaemonSet) (string, error) {
	identifier, ok := kubekind.Labels[config.IdentLabelKey]
	if !ok {
		return "", fmt.Errorf("%T %s does not have %s label", kubekind, kubekind.Name, config.IdentLabelKey)
	}
	return identifier, nil
}

// Read the mean-time-between-failures value defined by the DaemonSet
// in the label defined by config.MtbfLabelKey
func meanTimeBetweenFailures(kubekind *appsv1.DaemonSet) (string, error) {
	mtbf, ok := kubekind.Labels[config.MtbfLabelKey]
	if !ok {
		return "", fmt.Errorf("%T %s does not have %s label", kubekind, kubekind.Name, config.MtbfLabelKey)
	}

	_, err := time.ParseDuration(mtbf)
	if err != nil {
		return "", fmt.Errorf("error parsing mtbf %s: %v", mtbf, err)
	}

	return mtbf, nil
}
