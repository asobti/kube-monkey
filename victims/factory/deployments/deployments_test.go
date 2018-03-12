package deployments

import (
	"testing"

	"github.com/asobti/kube-monkey/config"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDENTIFIER = "kube-monkey-id"
	NAME       = "deployment_name"
	NAMESPACE  = metav1.NamespaceDefault
)

func newDeployment(name string, labels map[string]string) v1beta1.Deployment {

	return v1beta1.Deployment{
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
			config.MtbfLabelKey:  "1",
		},
	)
	depl, err := New(&v1depl)

	if err != nil {
		t.Error(err)
	}

	if ns := depl.Namespace(); ns != NAMESPACE {
		t.Errorf("Unexpected deployment Namespace, got %s", ns)
	}

	if k := depl.Kind(); k != "v1beta1.Deployment" {
		t.Errorf("Unexpected deployment Kindepl, got %s", k)
	}

	if n := depl.Name(); n != NAME {
		t.Errorf("Unexpected deployment Name, got %s", n)
	}

	if i := depl.Identifier(); i != IDENTIFIER {
		t.Errorf("Unexpected deployment Identifier, got %s", i)
	}

	if m := depl.Mtbf(); m != 1 {
		t.Errorf("Unexpected deployment Mtbf, got %d", m)
	}

}

func TestInvalidIdentifier(t *testing.T) {
	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.MtbfLabelKey: "1",
		},
	)
	_, err := New(&v1depl)

	if err == nil {
		t.Error("Expected an error if config.IdentLabelKey label doesn't exist")
	}
}

func TestInvalidMtbf(t *testing.T) {
	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
		},
	)
	_, err := New(&v1depl)

	if err == nil {
		t.Error("Expected an error if config.MtbfLabelKey label doesn't exist")
	}

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "string",
		},
	)
	_, err = New(&v1depl)

	if err == nil {
		t.Error("Expected an error if config.MtbfLabelKey label can't be converted a Int type")
	}
	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "0",
		},
	)
	_, err = New(&v1depl)

	if err == nil {
		t.Error("Expected an error if config.MtbfLabelKey label is lower than 1")
	}
}
