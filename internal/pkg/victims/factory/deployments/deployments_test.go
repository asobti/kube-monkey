package deployments

import (
	"testing"

	"kube-monkey/internal/pkg/config"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDENTIFIER = "kube-monkey-id"
	NAME       = "deployment_name"
	NAMESPACE  = metav1.NamespaceDefault
)

func newDeployment(name string, labels map[string]string) appsv1.Deployment {

	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
	}
}

func TestNew(t *testing.T) {

	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "1h",
		},
	)
	depl, err := New(&v1depl)

	assert.NoError(t, err)
	assert.Equal(t, "v1.Deployment", depl.Kind())
	assert.Equal(t, NAME, depl.Name())
	assert.Equal(t, NAMESPACE, depl.Namespace())
	assert.Equal(t, IDENTIFIER, depl.Identifier())
	assert.Equal(t, "1h", depl.Mtbf())
}

func TestInvalidIdentifier(t *testing.T) {
	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.MtbfLabelKey: "1h",
		},
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
	)
	_, err := New(&v1depl)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label doesn't exist")

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "string",
		},
	)
	_, err = New(&v1depl)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label can't be converted a time.Duration type")
}
