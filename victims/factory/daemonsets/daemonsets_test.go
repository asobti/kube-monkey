package daemonsets

import (
	"testing"

	"github.com/asobti/kube-monkey/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDENTIFIER = "kube-monkey-id"
	NAME       = "daemonset_name"
	NAMESPACE  = metav1.NamespaceDefault
)

func newDaemonSet(name string, labels map[string]string) v1.DaemonSet {

	return v1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
	}
}

func newDaemonSetWithSelector(name string, labels map[string]string, selectorMatchLabels map[string]string) v1.DaemonSet {

	return v1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
		Spec: v1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorMatchLabels,
			},
		},
	}
}

func TestNew(t *testing.T) {

	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "1",
		},
	)
	ds, err := New(&v1ds)

	assert.NoError(t, err)
	assert.Equal(t, "v1.DaemonSet", ds.Kind())
	assert.Equal(t, NAME, ds.Name())
	assert.Equal(t, NAMESPACE, ds.Namespace())
	assert.Equal(t, IDENTIFIER, ds.Identifier())
	assert.Equal(t, 1, ds.Mtbf())
}

func TestInvalidIdentifier(t *testing.T) {
	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			config.MtbfLabelKey: "1",
		},
	)
	_, err := New(&v1ds)

	assert.Errorf(t, err, "Expected an error if "+config.IdentLabelKey+" label doesn't exist")
}

func TestInvalidMtbf(t *testing.T) {
	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
		},
	)
	_, err := New(&v1ds)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label doesn't exist")

	v1ds = newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "string",
		},
	)
	_, err = New(&v1ds)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label can't be converted a Int type")

	v1ds = newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "0",
		},
	)
	_, err = New(&v1ds)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label is lower than 1")
}
