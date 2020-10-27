package statefulsets

import (
	"fmt"
	"strconv"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims"

	corev1 "k8s.io/api/apps/v1"
)

type StatefulSet struct {
	*victims.VictimBase
}

// New creates a new instance of StatefulSet
func New(ss *corev1.StatefulSet) (*StatefulSet, error) {
	ident, err := identifier(ss)
	if err != nil {
		return nil, err
	}
	mtbf, err := meanTimeBetweenFailures(ss)
	if err != nil {
		return nil, err
	}
	kind := fmt.Sprintf("%T", *ss)

	return &StatefulSet{VictimBase: victims.New(kind, ss.Name, ss.Namespace, ident, mtbf)}, nil
}

// Returns the value of the label defined by config.IdentLabelKey
// from the statefulset labels
// This label should be unique to a statefulset, and is used to
// identify the pods that belong to this statefulset, as pods
// inherit labels from the StatefulSet
func identifier(kubekind *corev1.StatefulSet) (string, error) {
	identifier, ok := kubekind.Labels[config.IdentLabelKey]
	if !ok {
		return "", fmt.Errorf("%T %s does not have %s label", kubekind, kubekind.Name, config.IdentLabelKey)
	}
	return identifier, nil
}

// Read the mean-time-between-failures value defined by the StatefulSet
// in the label defined by config.MtbfLabelKey
func meanTimeBetweenFailures(kubekind *corev1.StatefulSet) (int, error) {
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
