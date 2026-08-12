package deployments

import (
	"fmt"
	"time"

	"kube-monkey/internal/pkg/calendar"
	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/victims"

	appsv1 "k8s.io/api/apps/v1"
)

type Deployment struct {
	*victims.VictimBase
}

// New creates a new instance of Deployment
func New(dep *appsv1.Deployment) (*Deployment, error) {
	ident, err := identifier(dep)
	if err != nil {
		return nil, err
	}
	mtbf, err := meanTimeBetweenFailures(dep)
	if err != nil {
		return nil, err
	}
	kind := fmt.Sprintf("%T", *dep)

	return &Deployment{VictimBase: victims.New(kind, dep.Name, dep.Namespace, ident, mtbf)}, nil
}

// Returns the value of the label defined by config.IdentLabelKey
// from the deployment labels
// This label should be unique to a deployment, and is used to
// identify the pods that belong to this deployment, as pods
// inherit labels from the Deployment
func identifier(kubekind *appsv1.Deployment) (string, error) {
	identifier, ok := kubekind.Labels[config.IdentLabelKey]
	if !ok {
		return "", fmt.Errorf("%T %s does not have %s label", kubekind, kubekind.Name, config.IdentLabelKey)
	}
	return identifier, nil
}

// Read the mean-time-between-failures value defined by the Deployment
// in the label defined by config.MtbfLabelKey
func meanTimeBetweenFailures(kubekind *appsv1.Deployment) (time.Duration, error) {
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
