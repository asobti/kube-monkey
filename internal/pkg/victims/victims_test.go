package victims

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"kube-monkey/internal/pkg/config"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
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

func newPod(name string, status corev1.PodPhase) corev1.Pod {

	return corev1.Pod{
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
		Status: corev1.PodStatus{
			Phase: status,
		},
	}
}

func generateNPods(namePrefix string, n int, status corev1.PodPhase) []runtime.Object {
	var pods []runtime.Object
	for i := 0; i < n; i++ {
		pod := newPod(fmt.Sprintf("%s%d", namePrefix, i), status)
		pods = append(pods, &pod)
	}

	return pods
}

func generateNRunningPods(namePrefix string, n int) []runtime.Object {
	return generateNPods(namePrefix, n, corev1.PodRunning)
}

func newVictimBase() *VictimBase {
	return New(KIND, NAME, NAMESPACE, IDENTIFIER, 1)
}

func getPodList(client kube.Interface) *corev1.PodList {
	podList, _ := client.CoreV1().Pods(NAMESPACE).List(context.TODO(), metav1.ListOptions{})
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
	pod1 := newPod("app1", corev1.PodRunning)
	pod2 := newPod("app2", corev1.PodPending)

	client := fake.NewSimpleClientset(&pod1, &pod2)

	podList, err := v.RunningPods(client)

	assert.NoError(t, err)
	assert.Lenf(t, podList, 1, "Expected 1 item in podList, got %d", len(podList))

	name := podList[0].GetName()
	assert.Equal(t, name, "app1", "Unexpected pod name, got %s", name)
}

func TestPods(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", corev1.PodRunning)
	pod2 := newPod("app2", corev1.PodPending)

	client := fake.NewSimpleClientset(&pod1, &pod2)

	podList, _ := v.Pods(client)

	assert.Lenf(t, podList, 2, "Expected 2 items in podList, got %d", len(podList))
}

func TestDeletePod(t *testing.T) {

	v := newVictimBase()
	pod := newPod("app", corev1.PodRunning)

	client := fake.NewSimpleClientset(&pod)

	err := v.DeletePod(client, "app")
	assert.NoError(t, err)

	podList := getPodList(client).Items
	assert.Lenf(t, podList, 0, "Expected 0 items in podList, got %d", len(podList))
}

func TestDeleteRandomPods(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", corev1.PodRunning)
	pod2 := newPod("app2", corev1.PodPending)
	pod3 := newPod("app3", corev1.PodRunning)

	client := fake.NewSimpleClientset(&pod1, &pod2, &pod3)
	podList := getPodList(client).Items
	assert.Lenf(t, podList, 3, "Expected 3 items in podList, got %d", len(podList))

	err := v.DeleteRandomPods(client, 0)
	assert.NotNil(t, err, "expected err for killNum=0 but got nil")

	err = v.DeleteRandomPods(client, -1)
	assert.NotNil(t, err, "expected err for negative terminations but got nil")

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

func TestKillNumberForMaxPercentage(t *testing.T) {

	v := newVictimBase()

	pods := generateNRunningPods("app", 100)

	client := fake.NewSimpleClientset(pods...)

	killNum, err := v.KillNumberForMaxPercentage(client, 50) // 50% means we kill between at most 50 pods of the 100 that are running
	assert.Nil(t, err, "Expected err to be nil but got %v", err)
	assert.Truef(t, killNum >= 0 && killNum <= 50, "Expected kill number between 0 and 50 pods, got %d", killNum)
}

func TestKillNumberForMaxPercentageInvalidValues(t *testing.T) {
	type TestCase struct {
		name          string
		maxPercentage int
		expectedNum   int
		expectedErr   bool
	}

	tcs := []TestCase{
		{
			name:          "Negative value for maxPercentage",
			maxPercentage: -1,
			expectedNum:   0,
			expectedErr:   true,
		},
		{
			name:          "0 value for maxPercentage",
			maxPercentage: 0,
			expectedNum:   0,
			expectedErr:   false,
		},
		{
			name:          "maxPercentage greater than 100",
			maxPercentage: 110,
			expectedNum:   0,
			expectedErr:   true,
		},
	}

	for _, tc := range tcs {
		v := newVictimBase()
		client := fake.NewSimpleClientset()

		result, err := v.KillNumberForMaxPercentage(client, tc.maxPercentage)

		if tc.expectedErr {
			assert.NotNil(t, err, tc.name)
		} else {
			assert.Nil(t, err, tc.name)
			assert.Equal(t, result, tc.expectedNum, tc.name)
		}
	}
}

func TestDeletePodsFixedPercentage(t *testing.T) {
	type TestCase struct {
		name           string
		killPercentage int
		pods           []runtime.Object
		expectedNum    int
		expectedErr    bool
	}

	tcs := []TestCase{
		{
			name:           "negative value for killPercentage",
			killPercentage: -1,
			expectedNum:    0,
			expectedErr:    true,
		},
		{
			name:           "0 value for killPercentage",
			killPercentage: 0,
			expectedNum:    0,
			expectedErr:    false,
		},
		{
			name:           "killPercentage greater than 100",
			killPercentage: 110,
			expectedNum:    0,
			expectedErr:    true,
		},
		{
			name:           "correctly calculates pods to kill based on killPercentage",
			killPercentage: 50,
			pods:           generateNRunningPods("app", 10),
			expectedNum:    5,
			expectedErr:    false,
		},
		{
			name:           "correctly floors fractional values for the number of pods to kill",
			killPercentage: 33,
			pods:           generateNRunningPods("app", 10),
			expectedNum:    3,
			expectedErr:    false,
		},
		{
			name:           "does not count pending pods when calculating num of pods to kill",
			killPercentage: 80,
			pods: append(
				generateNPods("running", 1, corev1.PodRunning),
				generateNPods("pending", 1, corev1.PodPending)...),
			expectedNum: 1,
			expectedErr: false,
		},
	}

	for _, tc := range tcs {
		client := fake.NewSimpleClientset(tc.pods...)
		v := newVictimBase()

		result, err := v.KillNumberForFixedPercentage(client, tc.killPercentage)

		if tc.expectedErr {
			assert.NotNil(t, err, tc.name)
		} else {
			assert.Nil(t, err, tc.name)
			assert.Equal(t, tc.expectedNum, result, tc.name)
		}
	}

}

func TestDeleteRandomPod(t *testing.T) {

	v := newVictimBase()
	pod1 := newPod("app1", corev1.PodRunning)
	pod2 := newPod("app2", corev1.PodPending)

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

	pod1 := newPod("app1", corev1.PodRunning)
	pod2 := newPod("app2", corev1.PodPending)

	name := RandomPodName([]corev1.Pod{pod1, pod2})
	assert.Truef(t, strings.HasPrefix(name, "app"), "Pod name %s should start with 'app'", name)
}

func TestGetDeleteOptsForPod(t *testing.T) {
	configuredGracePeriod := config.GracePeriodSeconds()

	v := newVictimBase()
	deleteOpts := v.GetDeleteOptsForPod()

	assert.Equal(t, deleteOpts.GracePeriodSeconds, configuredGracePeriod)

}
