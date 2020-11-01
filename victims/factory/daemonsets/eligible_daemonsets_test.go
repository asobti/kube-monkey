package daemonsets

import (
	"testing"

	"github.com/asobti/kube-monkey/config"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEligibleDaemonSets(t *testing.T) {
	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			"kube-monkey/identifier": "1",
			"kube-monkey/mtbf":       "1h",
		},
	)

	client := fake.NewSimpleClientset(&v1ds)
	victims, _ := EligibleDaemonSets(client, NAMESPACE, &metav1.ListOptions{})

	assert.Len(t, victims, 1)
}

func TestIsEnrolled(t *testing.T) {
	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:   "1",
			config.MtbfLabelKey:    "1h",
			config.EnabledLabelKey: config.EnabledLabelValue,
		},
	)

	depl, _ := New(&v1ds)

	client := fake.NewSimpleClientset(&v1ds)

	b, _ := depl.IsEnrolled(client)

	assert.Equal(t, b, true, "Expected daemonset to be enrolled")
}

func TestIsNotEnrolled(t *testing.T) {
	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:   "1",
			config.MtbfLabelKey:    "1h",
			config.EnabledLabelKey: "x",
		},
	)

	ds, _ := New(&v1ds)

	client := fake.NewSimpleClientset(&v1ds)

	b, _ := ds.IsEnrolled(client)

	assert.Equal(t, b, false, "Expected daemonset to not be enrolled")
}

func TestKillType(t *testing.T) {

	ident := "1"
	mtbf := "1h"
	killMode := "kill-mode"

	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: ident,
			config.MtbfLabelKey:  mtbf,
		},
	)

	depl, _ := New(&v1ds)

	client := fake.NewSimpleClientset(&v1ds)

	_, err := depl.KillType(client)

	assert.EqualError(t, err, depl.Kind()+" "+depl.Name()+" does not have "+config.KillTypeLabelKey+" label")

	v1ds = newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:    ident,
			config.MtbfLabelKey:     mtbf,
			config.KillTypeLabelKey: killMode,
		},
	)

	client = fake.NewSimpleClientset(&v1ds)

	kill, _ := depl.KillType(client)

	assert.Equal(t, kill, killMode, "Unexpected kill value, got %d", kill)
}

func TestKillValue(t *testing.T) {

	ident := "1"
	mtbf := "1h"
	killValue := "0"

	v1ds := newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: ident,
			config.MtbfLabelKey:  mtbf,
		},
	)

	depl, _ := New(&v1ds)

	client := fake.NewSimpleClientset(&v1ds)

	_, err := depl.KillValue(client)

	assert.EqualError(t, err, depl.Kind()+" "+depl.Name()+" does not have "+config.KillValueLabelKey+" label")

	v1ds = newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:     ident,
			config.MtbfLabelKey:      mtbf,
			config.KillValueLabelKey: killValue,
		},
	)

	client = fake.NewSimpleClientset(&v1ds)

	_, err = depl.KillValue(client)

	assert.EqualError(t, err, "Invalid value for label "+config.KillValueLabelKey+": "+killValue)

	killValue = "1"

	v1ds = newDaemonSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:     ident,
			config.MtbfLabelKey:      mtbf,
			config.KillValueLabelKey: killValue,
		},
	)

	client = fake.NewSimpleClientset(&v1ds)

	kill, _ := depl.KillValue(client)

	assert.Equalf(t, kill, 1, "Unexpected a kill value, got %d", kill)
}
