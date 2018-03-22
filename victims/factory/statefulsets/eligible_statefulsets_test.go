package statefulsets

import (
	"testing"

	"github.com/asobti/kube-monkey/config"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEligibleStatefulSets(t *testing.T) {
	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			"kube-monkey/identifier": "1",
			"kube-monkey/mtbf":       "1",
		},
	)

	client := fake.NewSimpleClientset(&v1stfs)
	victims, _ := EligibleStatefulSets(client, NAMESPACE, &metav1.ListOptions{})

	assert.Len(t, victims, 1)
}

func TestIsEnrolled(t *testing.T) {
	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:   "1",
			config.MtbfLabelKey:    "1",
			config.EnabledLabelKey: config.EnabledLabelValue,
		},
	)

	stfs, _ := New(&v1stfs)

	client := fake.NewSimpleClientset(&v1stfs)

	b, _ := stfs.IsEnrolled(client)

	assert.Equal(t, b, true, "Expected stfsoyment to be enrolled")
}

func TestIsNotEnrolled(t *testing.T) {
	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:   "1",
			config.MtbfLabelKey:    "1",
			config.EnabledLabelKey: "x",
		},
	)

	stfs, _ := New(&v1stfs)

	client := fake.NewSimpleClientset(&v1stfs)

	b, _ := stfs.IsEnrolled(client)

	assert.Equal(t, b, false, "Expected stfsoyment to not be enrolled")
}

func TestKillType(t *testing.T) {

	ident := "1"
	mtbf := "1"
	killMode := "kill-mode"

	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: ident,
			config.MtbfLabelKey:  mtbf,
		},
	)

	stfs, _ := New(&v1stfs)

	client := fake.NewSimpleClientset(&v1stfs)

	_, err := stfs.KillType(client)

	assert.EqualError(t, err, stfs.Kind()+" "+stfs.Name()+" does not have "+config.KillTypeLabelKey+" label")

	v1stfs = newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:    ident,
			config.MtbfLabelKey:     mtbf,
			config.KillTypeLabelKey: killMode,
		},
	)

	client = fake.NewSimpleClientset(&v1stfs)

	kill, _ := stfs.KillType(client)

	assert.Equal(t, kill, killMode, "Unexpected kill value, got %d", kill)
}

func TestKillValue(t *testing.T) {

	ident := "1"
	mtbf := "1"
	killValue := "0"

	v1stfs := newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey: ident,
			config.MtbfLabelKey:  mtbf,
		},
	)

	stfs, _ := New(&v1stfs)

	client := fake.NewSimpleClientset(&v1stfs)

	_, err := stfs.KillValue(client)

	assert.EqualError(t, err, stfs.Kind()+" "+stfs.Name()+" does not have "+config.KillValueLabelKey+" label")

	v1stfs = newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:     ident,
			config.MtbfLabelKey:      mtbf,
			config.KillValueLabelKey: killValue,
		},
	)

	client = fake.NewSimpleClientset(&v1stfs)

	_, err = stfs.KillValue(client)

	assert.EqualError(t, err, "Invalid value for label "+config.KillValueLabelKey+": "+killValue)

	killValue = "1"

	v1stfs = newStatefulSet(
		NAME,
		map[string]string{
			config.IdentLabelKey:     ident,
			config.MtbfLabelKey:      mtbf,
			config.KillValueLabelKey: killValue,
		},
	)

	client = fake.NewSimpleClientset(&v1stfs)

	kill, _ := stfs.KillValue(client)

	assert.Equalf(t, kill, 1, "Unexpected a kill value, got %d", kill)
}
