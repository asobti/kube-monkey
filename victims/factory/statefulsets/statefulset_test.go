package statefulsets

import (
	"testing"

	"github.com/asobti/kube-monkey/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDENTIFIER = "kube-monkey-id"
	NAME       = "statefulset_name"
	NAMESPACE  = metav1.NamespaceDefault
	REPLICAS   = 1
)

func newStatefulSet(name string, labels map[string]string, replicas int32) v1.StatefulSet {

	return v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
		Spec: v1.StatefulSetSpec{
			Replicas: &replicas,
		},
	}
}

func newStatefulSetWithSelector(name string, labels map[string]string, selectorMatchLabels map[string]string) v1.StatefulSet {

	return v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
		Spec: v1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorMatchLabels,
			},
		},
	}
}

func TestNew(t *testing.T) {

	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "1",
		},
		REPLICAS,
	)
	stfs, err := New(&v1stfs)

	assert.NoError(t, err)
	assert.Equal(t, "v1.StatefulSet", stfs.Kind())
	assert.Equal(t, NAME, stfs.Name())
	assert.Equal(t, NAMESPACE, stfs.Namespace())
	assert.Equal(t, IDENTIFIER, stfs.Identifier())
	assert.Equal(t, 1, stfs.Mtbf())
}

func TestInvalidIdentifier(t *testing.T) {
	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.MtbfLabelKey: "1",
		},
		REPLICAS,
	)
	_, err := New(&v1stfs)

	assert.Errorf(t, err, "Expected an error if "+config.IdentLabelKey+" label doesn't exist")
}

func TestInvalidMtbf(t *testing.T) {
	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
		},
		REPLICAS,
	)
	_, err := New(&v1stfs)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label doesn't exist")

	v1stfs = newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "string",
		},
		REPLICAS,
	)
	_, err = New(&v1stfs)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label can't be converted a Int type")

	v1stfs = newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "0",
		},
		REPLICAS,
	)
	_, err = New(&v1stfs)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label is lower than 1")
}
