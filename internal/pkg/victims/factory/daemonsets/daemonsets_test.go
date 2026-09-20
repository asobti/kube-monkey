package daemonsets

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
	NAME       = "daemonset_name"
	NAMESPACE  = metav1.NamespaceDefault
)

func newDaemonSet(name string, labels map[string]string) appsv1.DaemonSet {
	return newDaemonSetWithTemplateLabels(name, labels, map[string]string{"app": name})
}

func newDaemonSetWithTemplateLabels(name string, labels, templateLabels map[string]string) appsv1.DaemonSet {

	return appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
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
	assert.Equal(t, 24*time.Hour, ds.Mtbf())
}

func TestNewWithMtbfInHours(t *testing.T) {
	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "2h",
		},
	)
	ds, err := New(&v1ds)

	assert.NoError(t, err)
	assert.Equal(t, 2*time.Hour, ds.Mtbf())
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

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label is not a valid mtbf")

	v1ds = newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "0",
		},
	)
	_, err = New(&v1ds)

	assert.Errorf(t, err, "Expected an error if "+config.MtbfLabelKey+" label is not greater than zero")
}

func TestNewFallsBackToTheDaemonSetSelector(t *testing.T) {

	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "1",
		},
	)
	victim, err := New(&v1ds)

	assert.NoError(t, err)
	assert.Equal(t, "app="+NAME, victim.PodSelector().String())
}

func TestNewUsesTheIdentifierOnThePodTemplate(t *testing.T) {

	v1ds := newDaemonSetWithTemplateLabels(
		NAME,
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
			config.MtbfLabelKey:  "1",
		},
		map[string]string{
			config.IdentLabelKey: IDENTIFIER,
		},
	)
	victim, err := New(&v1ds)

	assert.NoError(t, err)
	assert.Equal(t, config.IdentLabelKey+"="+IDENTIFIER, victim.PodSelector().String())
}
