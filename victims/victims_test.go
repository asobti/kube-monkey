package victims

import (
	"strings"
	"testing"

	"github.com/asobti/kube-monkey/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	NAMESPACE  = metav1.NamespaceDefault
	IDENTIFIER = "kube-monkey-id"
)

func newPod(name string, status v1.PodPhase) v1.Pod {

	return v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels: map[string]string{
				"kube-monkey/identifier": IDENTIFIER,
			},
		},
		Status: v1.PodStatus{
			Phase: status,
		},
	}

}

func newVictimBase() *VictimBase {
	return New("Pod", "name", NAMESPACE, IDENTIFIER, 1)
}

func getPodList(client kube.Interface) *v1.PodList {
	podList, _ := client.CoreV1().Pods(NAMESPACE).List(metav1.ListOptions{})
	return podList
}

func TestVictimBaseTemplateGetters(t *testing.T) {

	v := newVictimBase()

	assert.Equal(t, "Pod", v.Kind())
	assert.Equal(t, "name", v.Name())
	assert.Equal(t, NAMESPACE, v.Namespace())
	assert.Equal(t, IDENTIFIER, v.Identifier())
	assert.Equal(t, 1, v.Mtbf())
}

func TestRunningPods(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)

	client := fake.NewSimpleClientset(&pod1, &pod2)

	podList, e := v.RunningPods(client)

	if e != nil {
		t.Errorf("Unexpected error %s while getting running pod", e)
	}

	if len(podList) != 1 {
		t.Errorf("Expected 1 item in podList, got %d", len(podList))
	}
}

func TestPods(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)

	client := fake.NewSimpleClientset(&pod1, &pod2)

	podList, _ := v.Pods(client)

	if len(podList) != 2 {
		t.Errorf("Expected 2 item in podList, got %d", len(podList))
	}
}

func TestDeletePod(t *testing.T) {

	v := newVictimBase()
	pod := newPod("app", v1.PodRunning)

	client := fake.NewSimpleClientset(&pod)

	if e := v.DeletePod(client, "app"); e != nil {
		t.Errorf("Unexpected error %s while deleting pod", e)
	}

	if podList := getPodList(client); len(podList.Items) != 0 {
		t.Errorf("Expected 0 items in podList, got %d", len(podList.Items))
	}
}

func TestDeleteRandomPods(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)
	pod3 := newPod("app3", v1.PodRunning)

	client := fake.NewSimpleClientset(&pod1, &pod2, &pod3)

	if e := v.DeleteRandomPods(client, 0); e == nil {
		t.Error("Expected an error if no termination was requested")
	}

	if e := v.DeleteRandomPods(client, -1); e == nil {
		t.Error("Expected an error if negative termination was requested")
	}

	_ = v.DeleteRandomPods(client, 1)

	if podList := getPodList(client); len(podList.Items) != 2 {
		t.Errorf("Expected 2 items in podList, got %d", len(podList.Items))
	}

	_ = v.DeleteRandomPods(client, 2)

	if podList := getPodList(client); len(podList.Items) != 1 {
		t.Errorf("Expected 1 items in podList, got %d", len(podList.Items))
	}

	if e := v.DeleteRandomPods(client, 2); e == nil {
		t.Error("Expected an error if no pods")
	}
}

func TestTerminateAllPods(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)
	pod3 := newPod("app3", v1.PodRunning)

	client := fake.NewSimpleClientset(&pod1, &pod2, &pod3)

	_ = v.TerminateAllPods(client)

	if podList := getPodList(client); len(podList.Items) != 0 {
		t.Errorf("Expected 0 items in podList, got %d", len(podList.Items))
	}

	if e := v.TerminateAllPods(client); e == nil {
		t.Error("Expected an error if no pods")
	}

}

func TestDeleteRandomPod(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)

	client := fake.NewSimpleClientset(&pod1, &pod2)

	_ = v.DeleteRandomPod(client)

	if podList := getPodList(client); len(podList.Items) != 1 {
		t.Errorf("Expected 1 items in podList, got %d", len(podList.Items))
	}

	if e := v.DeleteRandomPods(client, 2); e == nil {
		t.Error("Expected an error if no pods")
	}
}

func TestIsBlacklisted(t *testing.T) {

	v := newVictimBase()

	config.SetDefaults()

	if e := v.IsBlacklisted(); e != false {
		t.Errorf("%s namespace should not be blacklisted", NAMESPACE)
	}

	v = New("Pod", "name", metav1.NamespaceSystem, IDENTIFIER, 1)

	if e := v.IsBlacklisted(); e != true {
		t.Errorf("%s namespace should be blacklisted", metav1.NamespaceSystem)
	}

}

func TestIsWhitelisted(t *testing.T) {

	v := newVictimBase()

	config.SetDefaults()

	if e := v.IsWhitelisted(); e != true {
		t.Errorf("%s namespace should be whitelisted", NAMESPACE)
	}
}

func TestRandomPodName(t *testing.T) {

	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)

	if name := RandomPodName([]v1.Pod{pod1, pod2}); strings.HasPrefix(name, "pod") {
		t.Errorf("Pod name %s should start with 'app'", name)
	}

}
