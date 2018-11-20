package victims

import (
	"fmt"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/asobti/kube-monkey/config"
	"github.com/bouk/monkey"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	NAMESPACE  = metav1.NamespaceDefault
	IDENTIFIER = "kube-monkey-id"
	KIND       = "Pod"
	NAME       = "name"
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
	return New(KIND, NAME, NAMESPACE, IDENTIFIER, 1)
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

	podList, err := v.RunningPods(client)

	assert.NoError(t, err)
	assert.Lenf(t, podList, 1, "Expected 1 item in podList, got %d", len(podList))

	name := podList[0].GetName()
	assert.Equal(t, name, "app1", "Unexpected pod name, got %s", name)
}

func TestPods(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)

	client := fake.NewSimpleClientset(&pod1, &pod2)

	podList, _ := v.Pods(client)

	assert.Lenf(t, podList, 2, "Expected 2 items in podList, got %d", len(podList))
}

func TestDeletePod(t *testing.T) {

	v := newVictimBase()
	pod := newPod("app", v1.PodRunning)

	client := fake.NewSimpleClientset(&pod)

	err := v.DeletePod(client, "app")
	assert.NoError(t, err)

	podList := getPodList(client).Items
	assert.Lenf(t, podList, 0, "Expected 0 items in podList, got %d", len(podList))
}

func TestDeletePodDryRun(t *testing.T) {
	// mock DryRun
	monkey.Patch(config.DryRun, func() bool { return true })
	defer monkey.Unpatch(config.DryRun)

	v := newVictimBase()
	pod := newPod("app", v1.PodRunning)

	client := fake.NewSimpleClientset(&pod)

	err := v.DeletePod(client, "app")
	assert.NoError(t, err)

	podList := getPodList(client).Items
	assert.Lenf(t, podList, 1, "Expected 1 item in podList, got %d", len(podList))
}

func TestDeleteRandomPods(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)
	pod3 := newPod("app3", v1.PodRunning)

	client := fake.NewSimpleClientset(&pod1, &pod2, &pod3)
	podList := getPodList(client).Items
	assert.Lenf(t, podList, 3, "Expected 3 items in podList, got %d", len(podList))

	err := v.DeleteRandomPods(client, 0)
	assert.EqualError(t, err, "No terminations requested for Pod name")

	err = v.DeleteRandomPods(client, -1)
	assert.EqualError(t, err, "Cannot request negative terminations 2 for Pod name")

	_ = v.DeleteRandomPods(client, 1)
	podList = getPodList(client).Items
	assert.Lenf(t, podList, 2, "Expected 2 items in podList, got %d", len(podList))

	_ = v.DeleteRandomPods(client, 2)
	podList = getPodList(client).Items
	assert.Lenf(t, podList, 1, "Expected 1 item in podList, got %d", len(podList))
	name := podList[0].GetName()
	assert.Equalf(t, name, "app2", "Expected not running pods not be deleted")

	err = v.DeleteRandomPods(client, 2)
	assert.EqualError(t, err, KIND+" "+NAME+" has no running pods at the moment")
}

func TestInvalidInputsForDeletePodsRandomMaxPercentage(t *testing.T) {

	v := newVictimBase()

	var pods []runtime.Object
	for i := 0; i < 100; i++ {
		pod := newPod(fmt.Sprintf("app%d", i), v1.PodRunning)
		pods = append(pods, &pod)
	}

	client := fake.NewSimpleClientset(pods...)

	killNum := v.KillNumberForMaxPercentage(client, -1)
	assert.Equalf(t, 0, killNum, "Should default to 0 percent when percentage has a negative value, got %d", killNum)

	killNum = v.KillNumberForMaxPercentage(client, 101)
	assert.Truef(t, killNum > 0 && killNum <= 100, "Should default to 100 percent pods when percentage is greater than 100, got %d", killNum)
}

func TestKillNumberForMaxPercentage(t *testing.T) {

	v := newVictimBase()

	var pods []runtime.Object
	for i := 0; i < 100; i++ {
		pod := newPod(fmt.Sprintf("app%d", i), v1.PodRunning)
		pods = append(pods, &pod)
	}

	client := fake.NewSimpleClientset(pods...)

	killNum := v.KillNumberForMaxPercentage(client, 0) // 0% means we don't kill any pods
	assert.Equal(t, killNum, 0, "Expected 0 pods to be killed, got %d", killNum)

	killNum = v.KillNumberForMaxPercentage(client, 50) // 50% means we kill between at most 50 pods of the 100 that are running
	assert.Truef(t, killNum >= 0 && killNum <= 50, "Expected kill number between 0 and 50 pods, got %d", killNum)
}

func TestInvalidInputsForDeletePodsFixedMaxPercentage(t *testing.T) {

	v := newVictimBase()

	var pods []runtime.Object
	for i := 0; i < 100; i++ {
		pod := newPod(fmt.Sprintf("app%d", i), v1.PodRunning)
		pods = append(pods, &pod)
	}

	client := fake.NewSimpleClientset(pods...)

	killNum := v.KillNumberForFixedPercentage(client, -1)
	assert.Equalf(t, 0, killNum, "Should default to 0 percent when percentage has a negative value, got %d", killNum)

	killNum = v.KillNumberForFixedPercentage(client, 101)
	assert.Equalf(t, 100, killNum, "Should default to 100 percent when percentage is greater than 100, got %d", killNum)
}

func TestDeletePodsFixedPercentage(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending) // not running
	pod3 := newPod("app3", v1.PodRunning)
	pod4 := newPod("app4", v1.PodRunning)
	pod5 := newPod("app5", v1.PodRunning)
	pod6 := newPod("app6", v1.PodRunning)

	client := fake.NewSimpleClientset(&pod1, &pod2, &pod3, &pod4, &pod5, &pod6)

	killNum := v.KillNumberForFixedPercentage(client, 0) // 0% means we don't kill any pods
	assert.Equalf(t, killNum, 0, "Expected 0 pods to be killed, got %d", killNum)

	killNum = v.KillNumberForFixedPercentage(client, 50) // 50% means we kill 2 (rounded down from 2.5) out of 5 running pods
	assert.Equalf(t, killNum, 2, "Expected 2 pods to be killed, got %d", killNum)

	killNum = v.KillNumberForFixedPercentage(client, 100) // 100% means we kill all 6 running pods
	assert.Equalf(t, killNum, 5, "Expected 5 pods to be killed, got %d", killNum)
}

func TestDeleteRandomPod(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)

	client := fake.NewSimpleClientset(&pod1, &pod2)

	_ = v.DeleteRandomPod(client)
	podList := getPodList(client).Items
	assert.Len(t, podList, 1)

	err := v.DeleteRandomPods(client, 2)
	assert.EqualError(t, err, KIND+" "+NAME+" has no running pods at the moment")
}

func TestIsBlacklisted(t *testing.T) {

	v := newVictimBase()

	config.SetDefaults()

	b := v.IsBlacklisted()
	assert.False(t, b, "%s namespace should not be blacklisted", NAMESPACE)

	v = New("Pod", "name", metav1.NamespaceSystem, IDENTIFIER, 1)
	b = v.IsBlacklisted()
	assert.True(t, b, "%s namespace should be blacklisted", metav1.NamespaceSystem)

}

func TestIsWhitelisted(t *testing.T) {

	v := newVictimBase()

	config.SetDefaults()

	b := v.IsWhitelisted()
	assert.True(t, b, "%s namespace should be whitelisted", NAMESPACE)
}

func TestRandomPodName(t *testing.T) {

	pod1 := newPod("app1", v1.PodRunning)
	pod2 := newPod("app2", v1.PodPending)

	name := RandomPodName([]v1.Pod{pod1, pod2})
	assert.Truef(t, strings.HasPrefix(name, "app"), "Pod name %s should start with 'app'", name)
}

func TestGetDeleteOptsForPod(t *testing.T) {
	type TestCase struct {
		name                   string
		terminationGracePeriod *int64
		expectedGracePeriod    *int64
	}

	// helper method to create *int64 from int64 since Go does not allow
	// use of address operator (&) on numeric constants
	newInt64Pointer := func(val int64) *int64 {
		return &val
	}

	defaultGracePeriod := config.GracePeriodSeconds()

	tcs := []TestCase{
		{
			name:                   "nil pod TerminationGracePeriod",
			terminationGracePeriod: nil,
			expectedGracePeriod:    defaultGracePeriod,
		},
		{
			name:                   "pod TerminateGracePeriod lower than configured grace period",
			terminationGracePeriod: newInt64Pointer(*defaultGracePeriod - int64(1)),
			expectedGracePeriod:    defaultGracePeriod,
		},
		{
			name:                   "pod TerminationGracePeriod higher than configured grace period",
			terminationGracePeriod: newInt64Pointer(*defaultGracePeriod + int64(1)),
			expectedGracePeriod:    newInt64Pointer(*defaultGracePeriod + int64(1)),
		},
	}

	for _, tc := range tcs {
		pod := newPod("app", v1.PodRunning)
		pod.Spec.TerminationGracePeriodSeconds = tc.terminationGracePeriod

		v := newVictimBase()
		deleteOpts := v.GetDeleteOptsForPod(&pod)

		assert.Equal(t, deleteOpts.GracePeriodSeconds, tc.expectedGracePeriod, tc.name)
	}
}
