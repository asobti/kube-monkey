package deployments

import (
	"testing"

	"github.com/asobti/kube-monkey/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEligibleDeployments(t *testing.T) {
	v1deplepl := newDeployment(
		NAME,
		map[string]string{
			"kube-monkey/identifier": "1",
			"kube-monkey/mtbf":       "1",
		},
	)

	client := fake.NewSimpleClientset(&v1deplepl)
	victims, _ := EligibleDeployments(client, NAMESPACE, &metav1.ListOptions{})

	if l := len(victims); l != 1 {
		t.Errorf("Expected 1 items in victins, got %d", l)
	}
}

func TestIsEnrolled(t *testing.T) {
	v1deplepl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:   "1",
			config.MtbfLabelKey:    "1",
			config.EnabledLabelKey: config.EnabledLabelValue,
		},
	)

	depl, _ := New(&v1deplepl)

	client := fake.NewSimpleClientset(&v1deplepl)

	b, _ := depl.IsEnrolled(client)

	if b != true {
		t.Error("Expected deployment to be enrolled")
	}
}

func TestIsNotEnrolled(t *testing.T) {
	v1deplepl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:   "1",
			config.MtbfLabelKey:    "1",
			config.EnabledLabelKey: "x",
		},
	)

	depl, _ := New(&v1deplepl)

	client := fake.NewSimpleClientset(&v1deplepl)

	b, _ := depl.IsEnrolled(client)

	if b != false {
		t.Error("Expected deployment to not be enrolled")
	}
}

func TestKillType(t *testing.T) {

	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: "1",
			config.MtbfLabelKey:  "1",
		},
	)

	depl, _ := New(&v1depl)

	client := fake.NewSimpleClientset(&v1depl)

	_, err := depl.KillType(client)

	if err == nil {
		t.Error("Expected an error if kill mode label is not present")
	}

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:    "1",
			config.MtbfLabelKey:     "1",
			config.KillTypeLabelKey: "kill-mode",
		},
	)

	client = fake.NewSimpleClientset(&v1depl)

	kill, _ := depl.KillType(client)

	if kill != "kill-mode" {
		t.Errorf("Unexpected a kill mode, got %s", kill)
	}
}

func TestKillValue(t *testing.T) {

	v1depl := newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey: "1",
			config.MtbfLabelKey:  "1",
		},
	)

	depl, _ := New(&v1depl)

	client := fake.NewSimpleClientset(&v1depl)

	_, err := depl.KillValue(client)

	if err == nil {
		t.Error("Expected an error if kill value label is not present")
	}

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:     "1",
			config.MtbfLabelKey:      "1",
			config.KillValueLabelKey: "0",
		},
	)

	client = fake.NewSimpleClientset(&v1depl)

	_, err = depl.KillValue(client)

	if err == nil {
		t.Error("Expected an error if kill value label is less than 1")
	}

	v1depl = newDeployment(
		NAME,
		map[string]string{
			config.IdentLabelKey:     "1",
			config.MtbfLabelKey:      "1",
			config.KillValueLabelKey: "1",
		},
	)

	client = fake.NewSimpleClientset(&v1depl)

	kill, _ := depl.KillValue(client)

	if kill != 1 {
		t.Errorf("Unexpected a kill value, got %d", kill)
	}
}
