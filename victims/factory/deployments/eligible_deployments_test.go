package deployments

import (
	"testing"

	"github.com/asobti/kube-monkey/config"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEligibleDeployments(t *testing.T) {
	v1depl := newDeployment(
		NAME,
		map[string]string{
			"kube-monkey/identifier": "1",
			"kube-monkey/mtbf":       "1h",
		},
	)

	client := fake.NewSimpleClientset(&v1depl)
	victims, _ := EligibleDeployments(client, NAMESPACE, &metav1.ListOptions{})

	assert.Len(t, victims, 1)
}

func TestIsEnrolled(t *testing.T) {
	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:   "1",
			config.MtbfLabelKey:    "1h",
			config.EnabledLabelKey: config.EnabledLabelValue,
		},
	)

	depl, _ := New(&v1depl)

	client := fake.NewSimpleClientset(&v1depl)

	b, _ := depl.IsEnrolled(client)

	assert.Equal(t, b, true, "Expected deployment to be enrolled")
}

func TestIsNotEnrolled(t *testing.T) {
	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:   "1",
			config.MtbfLabelKey:    "1h",
			config.EnabledLabelKey: "x",
		},
	)

	depl, _ := New(&v1depl)

	client := fake.NewSimpleClientset(&v1depl)

	b, _ := depl.IsEnrolled(client)

	assert.Equal(t, b, false, "Expected deployment to not be enrolled")
}

func TestKillType(t *testing.T) {

	ident := "1"
	mtbf := "1h"
	killMode := "kill-mode"

	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: ident,
			config.MtbfLabelKey:  mtbf,
		},
	)

	depl, _ := New(&v1depl)

	client := fake.NewSimpleClientset(&v1depl)

	_, err := depl.KillType(client)

	assert.EqualError(t, err, depl.Kind()+" "+depl.Name()+" does not have "+config.KillTypeLabelKey+" label")

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:    ident,
			config.MtbfLabelKey:     mtbf,
			config.KillTypeLabelKey: killMode,
		},
	)

	client = fake.NewSimpleClientset(&v1depl)

	kill, _ := depl.KillType(client)

	assert.Equal(t, kill, killMode, "Unexpected kill value, got %d", kill)
}

func TestKillValue(t *testing.T) {

	ident := "1"
	mtbf := "1h"
	killValue := "0"

	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: ident,
			config.MtbfLabelKey:  mtbf,
		},
	)

	depl, _ := New(&v1depl)

	client := fake.NewSimpleClientset(&v1depl)

	_, err := depl.KillValue(client)

	assert.EqualError(t, err, depl.Kind()+" "+depl.Name()+" does not have "+config.KillValueLabelKey+" label")

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:     ident,
			config.MtbfLabelKey:      mtbf,
			config.KillValueLabelKey: killValue,
		},
	)

	client = fake.NewSimpleClientset(&v1depl)

	_, err = depl.KillValue(client)

	assert.EqualError(t, err, "Invalid value for label "+config.KillValueLabelKey+": "+killValue)

	killValue = "1"

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:     ident,
			config.MtbfLabelKey:      mtbf,
			config.KillValueLabelKey: killValue,
		},
	)

	client = fake.NewSimpleClientset(&v1depl)

	kill, _ := depl.KillValue(client)

	assert.Equalf(t, kill, 1, "Unexpected a kill value, got %d", kill)
}
