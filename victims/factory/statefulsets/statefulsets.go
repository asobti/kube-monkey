package statefulsets

import (
	"fmt"
	"time"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims"

	"k8s.io/api/apps/v1"
)

type StatefulSet struct {
	*victims.VictimBase
}

// New creates a new instance of StatefulSet
func New(ss *v1.StatefulSet) (*StatefulSet, error) {
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
func identifier(kubekind *v1.StatefulSet) (string, error) {
	identifier, ok := kubekind.Labels[config.IdentLabelKey]
	if !ok {
		return "", fmt.Errorf("%T %s does not have %s label", kubekind, kubekind.Name, config.IdentLabelKey)
	}
	return identifier, nil
}

// Read the mean-time-between-failures value defined by the StatefulSet
// in the label defined by config.MtbfLabelKey
func meanTimeBetweenFailures(kubekind *v1.StatefulSet) (string, error) {
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
