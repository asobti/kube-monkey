package deployments

import (
	"testing"

	"github.com/asobti/kube-monkey/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDENTIFIER = "kube-monkey-id"
	NAME       = "deployment_name"
	NAMESPACE  = metav1.NamespaceDefault
	REPLICAS   = 1
)

func newDeployment(name string, labels map[string]string, replicas int32) v1.Deployment {

	return v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
}

func newDeploymentWithSelector(name string, labels map[string]string, selectorMatchLabels map[string]string) v1.Deployment {

	return v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorMatchLabels,
			},
		},
	}
}

func TestNew(t *testing.T) {

	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "1",
		},
		REPLICAS,
	)
	depl, err := New(&v1depl)

	assert.NoError(t, err)
	assert.Equal(t, "v1.Deployment", depl.Kind())
	assert.Equal(t, NAME, depl.Name())
	assert.Equal(t, NAMESPACE, depl.Namespace())
	assert.Equal(t, IDENTIFIER, depl.Identifier())
	assert.Equal(t, 1, depl.Mtbf())
}

func TestInvalidIdentifier(t *testing.T) {
	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.MtbfLabelKey: "1",
		},
		REPLICAS,
	)
	_, err := New(&v1depl)

	assert.Errorf(t, err, "Expected an error if "+config.IdentLabelKey+" label doesn't exist")
}

func TestInvalidMtbf(t *testing.T) {
	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
		},
		REPLICAS,
	)
	_, err := New(&v1depl)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label doesn't exist")

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "string",
		},
		REPLICAS,
	)
	_, err = New(&v1depl)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label can't be converted a Int type")

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "0",
		},
		REPLICAS,
	)
	_, err = New(&v1depl)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label is lower than 1")
}
