package chaos

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/config/param"
	"kube-monkey/internal/pkg/victims"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	NAMESPACE  = metav1.NamespaceDefault
	IDENTIFIER = "kube-monkey-id"
	KIND       = "v1.Deployment"
	NAME       = "shop"
)

// Terminations only really happen outside dry run mode, which is not the default
func TestMain(m *testing.M) {
	config.SetDefaults()
	viper.Set(param.DryRun, false)
	m.Run()
}

// newChaos builds a chaos due now, for a victim whose labels read back as given
func newChaos(labels map[string]string) *Chaos {
	return New(time.Now(), victims.New(victims.Spec{
		Kind:        KIND,
		Name:        NAME,
		Namespace:   NAMESPACE,
		Identifier:  IDENTIFIER,
		Mtbf:        24 * time.Hour,
		PodSelector: victims.IdentifierSelector(IDENTIFIER),
		CurrentLabels: func(kube.Interface) (map[string]string, error) {
			return labels, nil
		},
	}))
}

func chaosWithUnreadableVictim(err error) *Chaos {
	return New(time.Now(), victims.New(victims.Spec{
		Kind:      KIND,
		Name:      NAME,
		Namespace: NAMESPACE,
		CurrentLabels: func(kube.Interface) (map[string]string, error) {
			return nil, err
		},
	}))
}

func killLabels(killType string, killValue string) map[string]string {
	labels := map[string]string{config.KillTypeLabelKey: killType}
	if killValue != "" {
		labels[config.KillValueLabelKey] = killValue
	}
	return labels
}

func clientWithRunningPods(n int) *fake.Clientset {
	pods := make([]runtime.Object, 0, n)
	for i := range n {
		pods = append(pods, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod-%d", i),
				Namespace: NAMESPACE,
				Labels:    map[string]string{config.IdentLabelKey: IDENTIFIER},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		})
	}
	return fake.NewSimpleClientset(pods...)
}

func podsLeft(t *testing.T, client kube.Interface) int {
	t.Helper()

	podList, err := client.CoreV1().Pods(NAMESPACE).List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err)
	return len(podList.Items)
}

func TestVerifyExecution(t *testing.T) {
	enrolled := map[string]string{config.EnabledLabelKey: config.EnabledLabelValue}

	t.Run("enrolled and allowed", func(t *testing.T) {
		assert.NoError(t, newChaos(enrolled).verifyExecution(nil))
	})

	t.Run("opted out since it was scheduled", func(t *testing.T) {
		err := newChaos(map[string]string{config.EnabledLabelKey: "disabled"}).verifyExecution(nil)

		assert.EqualError(t, err, KIND+" "+NAME+" is no longer enrolled in kube-monkey. Skipping")
	})

	t.Run("blacklisted since it was scheduled", func(t *testing.T) {
		viper.Set(param.BlacklistedNamespaces, []string{NAMESPACE})
		t.Cleanup(func() { viper.Set(param.BlacklistedNamespaces, []string{metav1.NamespaceSystem}) })

		err := newChaos(enrolled).verifyExecution(nil)

		assert.EqualError(t, err, KIND+" "+NAME+" is blacklisted. Skipping")
	})

	t.Run("dropped off the whitelist since it was scheduled", func(t *testing.T) {
		viper.Set(param.WhitelistedNamespaces, []string{"team-*"})
		t.Cleanup(func() { viper.Set(param.WhitelistedNamespaces, []string{metav1.NamespaceAll}) })

		err := newChaos(enrolled).verifyExecution(nil)

		assert.EqualError(t, err, KIND+" "+NAME+" is not whitelisted. Skipping")
	})

	t.Run("victim cannot be read", func(t *testing.T) {
		err := chaosWithUnreadableVictim(errors.New("deployments.apps \"shop\" not found")).verifyExecution(nil)

		assert.EqualError(t, err, `deployments.apps "shop" not found`)
	})
}

// The kill mode and its value decide how many pods go, which is the whole point
// of a termination, so these check the pods that are actually left afterwards
func TestTerminateKillsTheRightNumberOfPods(t *testing.T) {
	for name, tc := range map[string]struct {
		killType     string
		killValue    string
		running      int
		expectedLeft int
	}{
		"a fixed number":                  {config.KillFixedLabelValue, "2", 5, 3},
		"a fixed number above the count":  {config.KillFixedLabelValue, "9", 5, 0},
		"all of them":                     {config.KillAllLabelValue, "", 5, 0},
		"all of them, ignoring the value": {config.KillAllLabelValue, "1", 5, 0},
		"a fixed percentage":              {config.KillFixedPercentageLabelValue, "40", 10, 6},
		"a fixed percentage, rounded":     {config.KillFixedPercentageLabelValue, "33", 10, 7},
		"a hundred percent":               {config.KillFixedPercentageLabelValue, "100", 4, 0},
	} {
		t.Run(name, func(t *testing.T) {
			client := clientWithRunningPods(tc.running)

			err := newChaos(killLabels(tc.killType, tc.killValue)).terminate(client)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedLeft, podsLeft(t, client))
		})
	}
}

// The percentage is drawn per termination, so all this can promise is that it
// never kills more than the maximum asks for, and never fails for trying
func TestTerminateWithARandomMaxPercentage(t *testing.T) {
	for range 50 {
		client := clientWithRunningPods(10)

		err := newChaos(killLabels(config.KillRandomMaxLabelValue, "50")).terminate(client)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, podsLeft(t, client), 5, "should never kill more than half of them")
	}
}

// A percentage that works out to no pods is the victim getting away with it,
// not a termination that went wrong. Reporting it as an error would count it
// in the failure metric and send a failure notification.
func TestTerminateWhenThePercentageWorksOutToNoPods(t *testing.T) {
	for name, labels := range map[string]map[string]string{
		"a fixed percentage of zero":          killLabels(config.KillFixedPercentageLabelValue, "0"),
		"a maximum percentage of zero":        killLabels(config.KillRandomMaxLabelValue, "0"),
		"a percentage too small to reach one": killLabels(config.KillFixedPercentageLabelValue, "1"),
	} {
		t.Run(name, func(t *testing.T) {
			client := clientWithRunningPods(10)

			err := newChaos(labels).terminate(client)

			assert.NoError(t, err)
			assert.Equal(t, 10, podsLeft(t, client), "no pods should be killed")
		})
	}
}

// Zero is a no-op for the percentage modes, but asking to kill a fixed zero
// pods is a misconfiguration worth reporting
func TestTerminateWithAFixedCountOfZero(t *testing.T) {
	client := clientWithRunningPods(3)

	err := newChaos(killLabels(config.KillFixedLabelValue, "0")).terminate(client)

	assert.EqualError(t, err, "no terminations requested for "+KIND+" "+NAME)
	assert.Equal(t, 3, podsLeft(t, client))
}

func TestTerminateRejectsAVictimItCannotUnderstand(t *testing.T) {
	for name, tc := range map[string]struct {
		labels      map[string]string
		expectedErr string
	}{
		"no kill mode": {
			labels:      map[string]string{},
			expectedErr: "failed to check " + config.KillTypeLabelKey + " label for " + KIND + " " + NAME + ": " + KIND + " " + NAME + " does not have " + config.KillTypeLabelKey + " label",
		},
		"a kill mode nobody recognises": {
			labels:      killLabels("gently", "1"),
			expectedErr: `failed to recognize ` + config.KillTypeLabelKey + ` label "gently" for ` + KIND + " " + NAME,
		},
		"no kill value": {
			labels:      killLabels(config.KillFixedLabelValue, ""),
			expectedErr: "failed to check " + config.KillValueLabelKey + " label for " + KIND + " " + NAME + ": " + KIND + " " + NAME + " does not have " + config.KillValueLabelKey + " label",
		},
		"a kill value that makes no sense": {
			labels:      killLabels(config.KillFixedLabelValue, "some"),
			expectedErr: "failed to check " + config.KillValueLabelKey + " label for " + KIND + " " + NAME + ": " + KIND + " " + NAME + ` has an invalid ` + config.KillValueLabelKey + ` label "some": expected a whole number that is not negative`,
		},
		"a percentage beyond 100": {
			labels:      killLabels(config.KillFixedPercentageLabelValue, "150"),
			expectedErr: "percentage value of 150 is invalid. Must be [0-100]",
		},
	} {
		t.Run(name, func(t *testing.T) {
			client := clientWithRunningPods(3)

			err := newChaos(tc.labels).terminate(client)

			assert.EqualError(t, err, tc.expectedErr)
			assert.Equal(t, 3, podsLeft(t, client), "nothing should be killed when the victim cannot be understood")
		})
	}
}

// Killing everything is the one mode that works without a kill value, so a
// victim using it must not be made to carry the label
func TestTerminateKillAllNeedsNoKillValue(t *testing.T) {
	client := clientWithRunningPods(3)

	err := newChaos(killLabels(config.KillAllLabelValue, "")).terminate(client)

	assert.NoError(t, err)
	assert.Equal(t, 0, podsLeft(t, client))
}

func TestExecuteReportsTheVictimItWasAskedAbout(t *testing.T) {
	chaos := newChaos(killLabels(config.KillAllLabelValue, ""))

	result := chaos.NewResult(errors.New("did not work"))

	assert.Equal(t, chaos.Victim(), result.Victim())
	assert.EqualError(t, result.Error(), "did not work")
}

func TestDurationToKillTime(t *testing.T) {
	killAt := time.Now().Add(time.Hour)
	c := New(killAt, nil)

	// The clock read inside DurationToKillTime happens between these two samples,
	// so the duration is bracketed exactly and no timing tolerance is needed
	before := time.Now()
	d := c.DurationToKillTime()
	after := time.Now()

	assert.LessOrEqual(t, d, killAt.Sub(before))
	assert.GreaterOrEqual(t, d, killAt.Sub(after))
}
