package victims

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/config/param"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	NAMESPACE  = metav1.NamespaceDefault
	IDENTIFIER = "kube-monkey-id"
	KIND       = "Pod"
	NAME       = "name"
)

// Terminations only really happen outside dry run mode, which is not the
// default, so every test here has to ask for it
func TestMain(m *testing.M) {
	config.SetDefaults()
	viper.Set(param.DryRun, false)
	m.Run()
}

func newVictim() *Victim {
	return New(Spec{
		Kind:        KIND,
		Name:        NAME,
		Namespace:   NAMESPACE,
		Identifier:  IDENTIFIER,
		Mtbf:        24 * time.Hour,
		PodSelector: IdentifierSelector(IDENTIFIER),
	})
}

func newPod(name string, status corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    map[string]string{config.IdentLabelKey: IDENTIFIER},
		},
		Status: corev1.PodStatus{Phase: status},
	}
}

func runningPods(namePrefix string, n int) []runtime.Object {
	pods := make([]runtime.Object, 0, n)
	for i := range n {
		pods = append(pods, newPod(fmt.Sprintf("%s%d", namePrefix, i), corev1.PodRunning))
	}
	return pods
}

func remainingPodNames(t *testing.T, client kube.Interface) []string {
	t.Helper()

	podList, err := client.CoreV1().Pods(NAMESPACE).List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err)

	names := make([]string, 0, len(podList.Items))
	for _, pod := range podList.Items {
		names = append(names, pod.Name)
	}
	return names
}

// recordDeletes notes the name of every pod the client is asked to delete, in
// the order it was asked, and lets the deletion go ahead
func recordDeletes(client *fake.Clientset) *[]string {
	deleted := []string{}
	client.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		deleted = append(deleted, action.(k8stesting.DeleteActionImpl).GetName())
		return false, nil, nil
	})
	return &deleted
}

func TestNewKeepsTheSpecApart(t *testing.T) {
	v := New(Spec{
		Kind:       "v1.Deployment",
		Name:       "shop",
		Namespace:  "team-checkout",
		Identifier: "shop-id",
		Mtbf:       2 * time.Hour,
	})

	assert.Equal(t, "v1.Deployment", v.Kind())
	assert.Equal(t, "shop", v.Name())
	assert.Equal(t, "team-checkout", v.Namespace())
	assert.Equal(t, "shop-id", v.Identifier())
	assert.Equal(t, 2*time.Hour, v.Mtbf())
}

func TestPodsFindsOnlyTheVictimsPods(t *testing.T) {
	mine := newPod("mine", corev1.PodRunning)
	theirs := newPod("theirs", corev1.PodRunning)
	theirs.Labels = map[string]string{config.IdentLabelKey: "someone-else"}

	pods, err := newVictim().Pods(fake.NewSimpleClientset(mine, theirs))

	assert.NoError(t, err)
	require.Len(t, pods, 1)
	assert.Equal(t, "mine", pods[0].Name)
}

func TestPodsRefusesAnEmptySelector(t *testing.T) {
	// An empty selector matches every pod in the namespace, so it must never
	// reach the apiserver
	for name, selector := range map[string]labels.Selector{
		"no selector":        nil,
		"empty selector":     labels.NewSelector(),
		"matches everything": labels.Everything(),
	} {
		t.Run(name, func(t *testing.T) {
			v := New(Spec{Kind: KIND, Name: NAME, Namespace: NAMESPACE, PodSelector: selector})

			_, err := v.Pods(fake.NewSimpleClientset(runningPods("app", 3)...))

			assert.EqualError(t, err, KIND+" "+NAME+" has no selector to find its pods with")
		})
	}
}

func TestRunningPodsLeavesOutTheRest(t *testing.T) {
	client := fake.NewSimpleClientset(
		newPod("running", corev1.PodRunning),
		newPod("pending", corev1.PodPending),
		newPod("succeeded", corev1.PodSucceeded),
		newPod("failed", corev1.PodFailed),
	)

	pods, err := newVictim().RunningPods(client)

	assert.NoError(t, err)
	require.Len(t, pods, 1)
	assert.Equal(t, "running", pods[0].Name)
}

func TestDeletePod(t *testing.T) {
	client := fake.NewSimpleClientset(newPod("app", corev1.PodRunning))

	assert.NoError(t, newVictim().DeletePod(client, "app"))
	assert.Empty(t, remainingPodNames(t, client))
}

// Dry run is what stands between a misconfigured kube-monkey and a real outage,
// so it has to hold even when everything else says to terminate
func TestDeletePodInDryRunMode(t *testing.T) {
	viper.Set(param.DryRun, true)
	t.Cleanup(func() { viper.Set(param.DryRun, false) })

	client := fake.NewSimpleClientset(newPod("app", corev1.PodRunning))

	assert.NoError(t, newVictim().DeletePod(client, "app"))
	assert.Equal(t, []string{"app"}, remainingPodNames(t, client), "dry run should not delete anything")
}

func TestDeleteRandomPodsInDryRunMode(t *testing.T) {
	viper.Set(param.DryRun, true)
	t.Cleanup(func() { viper.Set(param.DryRun, false) })

	client := fake.NewSimpleClientset(runningPods("app", 3)...)

	assert.NoError(t, newVictim().DeleteRandomPods(client, 3))
	assert.Len(t, remainingPodNames(t, client), 3, "dry run should not delete anything")
}

func TestDeleteRandomPodsKillsWhatWasAskedFor(t *testing.T) {
	client := fake.NewSimpleClientset(runningPods("app", 5)...)

	assert.NoError(t, newVictim().DeleteRandomPods(client, 2))
	assert.Len(t, remainingPodNames(t, client), 3)
}

// Deleting a pod that is already going away succeeds but kills nothing extra,
// so asking for n pods has to pick n different ones
func TestDeleteRandomPodsPicksADifferentPodEveryTime(t *testing.T) {
	client := fake.NewSimpleClientset(runningPods("app", 10)...)
	deleted := recordDeletes(client)

	assert.NoError(t, newVictim().DeleteRandomPods(client, 6))

	assert.Len(t, *deleted, 6)
	assert.Len(t, uniqueNames(*deleted), 6, "expected 6 different pods, got %v", *deleted)
	assert.Len(t, remainingPodNames(t, client), 4)
}

func TestDeleteRandomPodsLeavesPodsThatAreNotRunning(t *testing.T) {
	client := fake.NewSimpleClientset(
		newPod("running", corev1.PodRunning),
		newPod("pending", corev1.PodPending),
	)

	// More than are running, so every running pod goes and nothing else does
	assert.NoError(t, newVictim().DeleteRandomPods(client, 5))
	assert.Equal(t, []string{"pending"}, remainingPodNames(t, client))
}

func TestDeleteRandomPodsRejectsNonsense(t *testing.T) {
	for name, tc := range map[string]struct {
		killNum     int
		pods        []runtime.Object
		expectedErr string
	}{
		"no pods requested": {
			killNum:     0,
			pods:        runningPods("app", 3),
			expectedErr: "no terminations requested for " + KIND + " " + NAME,
		},
		"negative pods requested": {
			killNum:     -1,
			pods:        runningPods("app", 3),
			expectedErr: "cannot request negative terminations -1 for " + KIND + " " + NAME,
		},
		"nothing running": {
			killNum:     1,
			pods:        []runtime.Object{newPod("pending", corev1.PodPending)},
			expectedErr: KIND + " " + NAME + " has no running pods at the moment",
		},
	} {
		t.Run(name, func(t *testing.T) {
			client := fake.NewSimpleClientset(tc.pods...)
			deleted := recordDeletes(client)

			err := newVictim().DeleteRandomPods(client, tc.killNum)

			assert.EqualError(t, err, tc.expectedErr)
			assert.Empty(t, *deleted, "nothing should be deleted when the request makes no sense")
		})
	}
}

func TestDeletePodUsesTheConfiguredGracePeriod(t *testing.T) {
	viper.Set(param.GracePeriodSec, 42)
	t.Cleanup(func() { viper.Set(param.GracePeriodSec, 5) })

	client := fake.NewSimpleClientset(newPod("app", corev1.PodRunning))

	var gracePeriod *int64
	client.PrependReactor("delete", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		gracePeriod = action.(k8stesting.DeleteActionImpl).DeleteOptions.GracePeriodSeconds
		return false, nil, nil
	})

	require.NoError(t, newVictim().DeletePod(client, "app"))

	require.NotNil(t, gracePeriod)
	assert.Equal(t, int64(42), *gracePeriod)
}

func uniqueNames(names []string) map[string]bool {
	unique := make(map[string]bool, len(names))
	for _, name := range names {
		unique[name] = true
	}
	return unique
}

// victimWithLabels builds a victim whose labels are read back from the map it
// is given, standing in for the object still sitting in the cluster
func victimWithLabels(current map[string]string) *Victim {
	return New(Spec{
		Kind:      KIND,
		Name:      NAME,
		Namespace: NAMESPACE,
		CurrentLabels: func(kube.Interface) (map[string]string, error) {
			return current, nil
		},
	})
}

func victimWithUnreadableLabels(err error) *Victim {
	return New(Spec{
		Kind:      KIND,
		Name:      NAME,
		Namespace: NAMESPACE,
		CurrentLabels: func(kube.Interface) (map[string]string, error) {
			return nil, err
		},
	})
}

func TestIsEnrolled(t *testing.T) {
	for name, tc := range map[string]struct {
		labels   map[string]string
		enrolled bool
	}{
		"opted in":         {map[string]string{config.EnabledLabelKey: config.EnabledLabelValue}, true},
		"opted out":        {map[string]string{config.EnabledLabelKey: "disabled"}, false},
		"label gone":       {map[string]string{}, false},
		"no labels at all": {nil, false},
	} {
		t.Run(name, func(t *testing.T) {
			enrolled, err := victimWithLabels(tc.labels).IsEnrolled(nil)

			assert.NoError(t, err)
			assert.Equal(t, tc.enrolled, enrolled)
		})
	}
}

func TestIsEnrolledWhenTheVictimCannotBeRead(t *testing.T) {
	_, err := victimWithUnreadableLabels(errors.New("not found")).IsEnrolled(nil)

	assert.EqualError(t, err, "not found")
}

func TestKillType(t *testing.T) {
	killType, err := victimWithLabels(map[string]string{config.KillTypeLabelKey: config.KillAllLabelValue}).KillType(nil)

	assert.NoError(t, err)
	assert.Equal(t, config.KillAllLabelValue, killType)
}

func TestKillTypeWithoutTheLabel(t *testing.T) {
	_, err := victimWithLabels(map[string]string{}).KillType(nil)

	assert.EqualError(t, err, KIND+" "+NAME+" does not have "+config.KillTypeLabelKey+" label")
}

func TestKillTypeWhenTheVictimCannotBeRead(t *testing.T) {
	_, err := victimWithUnreadableLabels(errors.New("not found")).KillType(nil)

	assert.EqualError(t, err, "not found")
}

func TestKillValue(t *testing.T) {
	killValue, err := victimWithLabels(map[string]string{config.KillValueLabelKey: "3"}).KillValue(nil)

	assert.NoError(t, err)
	assert.Equal(t, 3, killValue)
}

func TestKillValueRejectsWhatItCannotUse(t *testing.T) {
	// The message has to name the value, because that is what the owner of the
	// workload has to go and fix
	for _, value := range []string{"lots", "-1", "1.5", "", " 2"} {
		t.Run(value, func(t *testing.T) {
			_, err := victimWithLabels(map[string]string{config.KillValueLabelKey: value}).KillValue(nil)

			assert.ErrorContains(t, err, config.KillValueLabelKey)
			assert.ErrorContains(t, err, `"`+value+`"`)
		})
	}
}

// Zero means "kill none of them" to the percentage modes, so it is the kill
// mode that decides whether it makes sense, not this parser
func TestKillValueAcceptsZero(t *testing.T) {
	killValue, err := victimWithLabels(map[string]string{config.KillValueLabelKey: "0"}).KillValue(nil)

	assert.NoError(t, err)
	assert.Equal(t, 0, killValue)
}

func TestKillValueWithoutTheLabel(t *testing.T) {
	_, err := victimWithLabels(map[string]string{}).KillValue(nil)

	assert.EqualError(t, err, KIND+" "+NAME+" does not have "+config.KillValueLabelKey+" label")
}

func TestKillNumberForKillingAll(t *testing.T) {
	client := fake.NewSimpleClientset(append(runningPods("app", 4), newPod("pending", corev1.PodPending))...)

	killNum, err := newVictim().KillNumberForKillingAll(client)

	assert.NoError(t, err)
	assert.Equal(t, 4, killNum, "only the running pods can be killed")
}

func TestKillNumberForFixedPercentage(t *testing.T) {
	for name, tc := range map[string]struct {
		percentage int
		running    int
		pending    int
		expected   int
	}{
		"half of them":              {50, 10, 0, 5},
		"rounds to the nearest pod": {33, 10, 0, 3},
		"rounds a half up":          {35, 10, 0, 4},
		"all of them":               {100, 7, 0, 7},
		"never less than the whole": {1, 10, 0, 0},
		"ignores pods not running":  {80, 1, 1, 1},
		"no pods at all":            {50, 0, 0, 0},
	} {
		t.Run(name, func(t *testing.T) {
			objects := runningPods("running", tc.running)
			for i := range tc.pending {
				objects = append(objects, newPod(fmt.Sprintf("pending%d", i), corev1.PodPending))
			}

			killNum, err := newVictim().KillNumberForFixedPercentage(fake.NewSimpleClientset(objects...), tc.percentage)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, killNum)
		})
	}
}

// A percentage of zero is a victim asking to be left alone, not a mistake
func TestKillNumberForZeroPercentage(t *testing.T) {
	client := fake.NewSimpleClientset(runningPods("app", 10)...)

	fixed, err := newVictim().KillNumberForFixedPercentage(client, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, fixed)

	max, err := newVictim().KillNumberForMaxPercentage(client, 0)
	assert.NoError(t, err)
	assert.Equal(t, 0, max)
}

func TestKillNumberRejectsPercentagesOutsideTheRange(t *testing.T) {
	client := fake.NewSimpleClientset(runningPods("app", 10)...)

	for _, percentage := range []int{-1, 101, 1000} {
		t.Run(fmt.Sprint(percentage), func(t *testing.T) {
			_, err := newVictim().KillNumberForFixedPercentage(client, percentage)
			assert.EqualError(t, err, fmt.Sprintf("percentage value of %d is invalid. Must be [0-100]", percentage))

			_, err = newVictim().KillNumberForMaxPercentage(client, percentage)
			assert.EqualError(t, err, fmt.Sprintf("percentage value of %d is invalid. Must be [0-100]", percentage))
		})
	}
}

// drawExactly pins the random percentage down for one test
func drawExactly(t *testing.T, percentage int) {
	t.Helper()

	original := randomPercentage
	randomPercentage = func(int) int { return percentage }
	t.Cleanup(func() { randomPercentage = original })
}

func TestKillNumberForMaxPercentageUsesTheDraw(t *testing.T) {
	// Both ends of the range, so the maximum is shown to be included and a
	// draw of nothing is shown to kill nothing
	for _, tc := range []struct{ draw, expected int }{{0, 0}, {13, 13}, {50, 50}} {
		t.Run(fmt.Sprint(tc.draw), func(t *testing.T) {
			drawExactly(t, tc.draw)
			client := fake.NewSimpleClientset(runningPods("app", 100)...)

			killNum, err := newVictim().KillNumberForMaxPercentage(client, 50)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, killNum)
		})
	}
}

// The draw has to cover both ends of the range. With a maximum of 1 a run of
// 200 draws that misses either end is not chance, it is a bug.
func TestRandomPercentageCoversBothEnds(t *testing.T) {
	seen := map[int]bool{}
	for range 200 {
		draw := randomPercentage(1)
		require.GreaterOrEqual(t, draw, 0)
		require.LessOrEqual(t, draw, 1)
		seen[draw] = true
	}

	assert.True(t, seen[0], "a draw of 0 should be possible")
	assert.True(t, seen[1], "a draw of the maximum should be possible")
}

func TestRandomPercentageOfZeroDrawsZero(t *testing.T) {
	for range 20 {
		assert.Equal(t, 0, randomPercentage(0))
	}
}

func TestKillNumberWhenThePodsCannotBeListed(t *testing.T) {
	// No selector, so listing the pods fails before the apiserver is reached
	v := New(Spec{Kind: KIND, Name: NAME, Namespace: NAMESPACE})
	client := fake.NewSimpleClientset()

	_, err := v.KillNumberForKillingAll(client)
	assert.ErrorContains(t, err, "failed to get running pods for victim")

	_, err = v.KillNumberForFixedPercentage(client, 50)
	assert.ErrorContains(t, err, "failed to get running pods for victim")
}

func TestIsBlacklisted(t *testing.T) {
	viper.Set(param.BlacklistedNamespaces, []string{metav1.NamespaceSystem})
	t.Cleanup(func() { viper.Set(param.BlacklistedNamespaces, []string{metav1.NamespaceSystem}) })

	assert.False(t, newVictim().IsBlacklisted(), NAMESPACE+" should not be blacklisted")

	inKubeSystem := New(Spec{Kind: KIND, Name: NAME, Namespace: metav1.NamespaceSystem})
	assert.True(t, inKubeSystem.IsBlacklisted(), metav1.NamespaceSystem+" should be blacklisted")
}

func TestIsWhitelisted(t *testing.T) {
	assert.True(t, newVictim().IsWhitelisted(), "every namespace is whitelisted by default")

	viper.Set(param.WhitelistedNamespaces, []string{"team-*"})
	t.Cleanup(func() { viper.Set(param.WhitelistedNamespaces, []string{metav1.NamespaceAll}) })

	assert.False(t, newVictim().IsWhitelisted(), NAMESPACE+" should not be whitelisted")
	assert.True(t, New(Spec{Namespace: "team-shop"}).IsWhitelisted())
}

func TestSpecFromLabels(t *testing.T) {
	spec, err := SpecFromLabels(KIND, NAME, NAMESPACE, map[string]string{
		config.IdentLabelKey: IDENTIFIER,
		config.MtbfLabelKey:  "6h",
	})

	assert.NoError(t, err)
	assert.Equal(t, Spec{
		Kind:       KIND,
		Name:       NAME,
		Namespace:  NAMESPACE,
		Identifier: IDENTIFIER,
		Mtbf:       6 * time.Hour,
	}, spec)
}

func TestSpecFromLabelsNeedsTheLabelsItReadsFrom(t *testing.T) {
	for name, tc := range map[string]struct {
		labels      map[string]string
		expectedErr string
	}{
		"no identifier": {
			labels:      map[string]string{config.MtbfLabelKey: "1"},
			expectedErr: KIND + " " + NAME + " does not have " + config.IdentLabelKey + " label",
		},
		"no mtbf": {
			labels:      map[string]string{config.IdentLabelKey: IDENTIFIER},
			expectedErr: KIND + " " + NAME + " does not have " + config.MtbfLabelKey + " label",
		},
		"mtbf that makes no sense": {
			labels:      map[string]string{config.IdentLabelKey: IDENTIFIER, config.MtbfLabelKey: "soon"},
			expectedErr: KIND + " " + NAME + ` has an invalid mtbf "soon": expected a whole number optionally followed by d, h or m`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := SpecFromLabels(KIND, NAME, NAMESPACE, tc.labels)

			assert.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestNewPodSelectorUsesIdentifierFromPodTemplate(t *testing.T) {
	selector, err := NewPodSelector(KIND, NAME, IDENTIFIER,
		map[string]string{config.IdentLabelKey: IDENTIFIER, "app": "victim"},
		&metav1.LabelSelector{MatchLabels: map[string]string{"app": "victim"}},
	)

	assert.NoError(t, err)
	assert.Equal(t, config.IdentLabelKey+"="+IDENTIFIER, selector.String())
}

// Several workloads can share one identifier, so the value on the metadata is
// what decides which pool of pods the victim belongs to
func TestNewPodSelectorPrefersMetadataIdentifierWhenTemplateConflicts(t *testing.T) {
	selector, err := NewPodSelector(KIND, NAME, IDENTIFIER,
		map[string]string{config.IdentLabelKey: "a-different-id"},
		&metav1.LabelSelector{MatchLabels: map[string]string{"app": "victim"}},
	)

	assert.NoError(t, err)
	assert.Equal(t, config.IdentLabelKey+"="+IDENTIFIER, selector.String())
}

func TestNewPodSelectorFallsBackToTheWorkloadSelector(t *testing.T) {
	for name, tc := range map[string]struct {
		workloadSelector *metav1.LabelSelector
		expected         string
	}{
		"match labels": {
			workloadSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "victim"}},
			expected:         "app=victim",
		},
		"match expressions": {
			workloadSelector: &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
				{Key: "app", Operator: metav1.LabelSelectorOpIn, Values: []string{"victim"}},
			}},
			expected: "app in (victim)",
		},
	} {
		t.Run(name, func(t *testing.T) {
			selector, err := NewPodSelector(KIND, NAME, IDENTIFIER, map[string]string{"app": "victim"}, tc.workloadSelector)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, selector.String())
		})
	}
}

// Nothing to select on means every pod in the namespace, so it has to be an
// error rather than a selector
func TestNewPodSelectorRejectsHavingNothingToSelectOn(t *testing.T) {
	expectedErr := KIND + " " + NAME + " has no " + config.IdentLabelKey + " label on its pod template and no pod selector to fall back on"

	_, err := NewPodSelector(KIND, NAME, IDENTIFIER, map[string]string{"app": "victim"}, nil)
	assert.EqualError(t, err, expectedErr)

	_, err = NewPodSelector(KIND, NAME, IDENTIFIER, nil, &metav1.LabelSelector{})
	assert.EqualError(t, err, expectedErr)
}
