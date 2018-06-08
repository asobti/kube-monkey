package daemonsets

import (
	"fmt"
	"strconv"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims"

	"k8s.io/api/apps/v1"
)

type DaemonSet struct {
	*victims.VictimBase
}

// New creates a new instance of DaemonSet
func New(dep *v1.DaemonSet) (*DaemonSet, error) {
	ident, err := identifier(dep)
	if err != nil {
		return nil, err
	}
	mtbf, err := meanTimeBetweenFailures(dep)
	if err != nil {
		return nil, err
	}
	kind := fmt.Sprintf("%T", *dep)

	return &DaemonSet{victims.New(kind, dep.Name, dep.Namespace, ident, mtbf)}, nil
}

// Returns the value of the label defined by config.IdentLabelKey
// from the DaemonSet labels
// This label should be unique to a DaemonSet, and is used to
// identify the pods that belong to this DaemonSet, as pods
// inherit labels from the DaemonSet
func identifier(kubekind *v1.DaemonSet) (string, error) {
	identifier, ok := kubekind.Labels[config.IdentLabelKey]
	if !ok {
		return "", fmt.Errorf("%T %s does not have %s label", kubekind, kubekind.Name, config.IdentLabelKey)
	}
	return identifier, nil
}

// Read the mean-time-between-failures value defined by the DaemonSet
// in the label defined by config.MtbfLabelKey
func meanTimeBetweenFailures(kubekind *v1.DaemonSet) (int, error) {
	mtbf, ok := kubekind.Labels[config.MtbfLabelKey]
	if !ok {
		return -1, fmt.Errorf("%T %s does not have %s label", kubekind, kubekind.Name, config.MtbfLabelKey)
	}

	mtbfInt, err := strconv.Atoi(mtbf)
	if err != nil {
		return -1, err
	}

	if !(mtbfInt > 0) {
		return -1, fmt.Errorf("Invalid value for label %s: %d", config.MtbfLabelKey, mtbfInt)
	}

	return mtbfInt, nil
}
