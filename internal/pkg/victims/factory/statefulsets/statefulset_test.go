package statefulsets

import (
	"testing"
	"time"

	"kube-monkey/internal/pkg/config"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IDENTIFIER = "kube-monkey-id"
	NAME       = "statefulset_name"
	NAMESPACE  = metav1.NamespaceDefault
)

func newStatefulSet(name string, labels map[string]string) appsv1.StatefulSet {
	return newStatefulSetWithTemplateLabels(name, labels, map[string]string{"app": name})
}

func newStatefulSetWithTemplateLabels(name string, labels, templateLabels map[string]string) appsv1.StatefulSet {

	return appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: templateLabels,
				},
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
	)
	stfs, err := New(&v1stfs)

	assert.NoError(t, err)
	assert.Equal(t, "v1.StatefulSet", stfs.Kind())
	assert.Equal(t, NAME, stfs.Name())
	assert.Equal(t, NAMESPACE, stfs.Namespace())
	assert.Equal(t, IDENTIFIER, stfs.Identifier())
	assert.Equal(t, 24*time.Hour, stfs.Mtbf())
}

func TestNewWithMtbfInHours(t *testing.T) {
	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "2h",
		},
	)
	stfs, err := New(&v1stfs)

	assert.NoError(t, err)
	assert.Equal(t, 2*time.Hour, stfs.Mtbf())
}

func TestInvalidIdentifier(t *testing.T) {
	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.MtbfLabelKey: "1",
		},
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
	)
	_, err := New(&v1stfs)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label doesn't exist")

	v1stfs = newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "string",
		},
	)
	_, err = New(&v1stfs)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label is not a valid mtbf")

	v1stfs = newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "0",
		},
	)
	_, err = New(&v1stfs)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label is not greater than zero")
}

func TestNewFallsBackToTheStatefulSetSelector(t *testing.T) {

	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "1",
		},
	)
	victim, err := New(&v1stfs)

	assert.NoError(t, err)
	assert.Equal(t, "app="+NAME, victim.PodSelector().String())
}

func TestNewUsesTheIdentifierOnThePodTemplate(t *testing.T) {

	v1stfs := newStatefulSetWithTemplateLabels(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "1",
		},
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
		},
	)
	victim, err := New(&v1stfs)

	assert.NoError(t, err)
	assert.Equal(t, config.IdentLabelKey+"="+IDENTIFIER, victim.PodSelector().String())
}
