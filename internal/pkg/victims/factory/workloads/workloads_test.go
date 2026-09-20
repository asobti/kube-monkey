package workloads

import (
	"testing"
	"time"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/victims"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	NAME       = "shop"
	NAMESPACE  = metav1.NamespaceDefault
	IDENTIFIER = "kube-monkey-id"
)

// The three kinds are meant to behave identically, so every test runs against
// all of them
var kinds = []struct {
	name      string
	newObject func(objectLabels, templateLabels map[string]string) runtime.Object
	newVictim func(runtime.Object) (*victims.Victim, error)
	eligible  func(kube.Interface, string, metav1.ListOptions) ([]*victims.Victim, error)
}{
	{
		name: "v1.Deployment",
		newObject: func(objectLabels, templateLabels map[string]string) runtime.Object {
			return &appsv1.Deployment{
				ObjectMeta: objectMeta(objectLabels),
				Spec: appsv1.DeploymentSpec{
					Selector: workloadSelector(),
					Template: podTemplate(templateLabels),
				},
			}
		},
		newVictim: func(obj runtime.Object) (*victims.Victim, error) { return NewDeployment(obj.(*appsv1.Deployment)) },
		eligible:  EligibleDeployments,
	},
	{
		name: "v1.StatefulSet",
		newObject: func(objectLabels, templateLabels map[string]string) runtime.Object {
			return &appsv1.StatefulSet{
				ObjectMeta: objectMeta(objectLabels),
				Spec: appsv1.StatefulSetSpec{
					Selector: workloadSelector(),
					Template: podTemplate(templateLabels),
				},
			}
		},
		newVictim: func(obj runtime.Object) (*victims.Victim, error) { return NewStatefulSet(obj.(*appsv1.StatefulSet)) },
		eligible:  EligibleStatefulSets,
	},
	{
		name: "v1.DaemonSet",
		newObject: func(objectLabels, templateLabels map[string]string) runtime.Object {
			return &appsv1.DaemonSet{
				ObjectMeta: objectMeta(objectLabels),
				Spec: appsv1.DaemonSetSpec{
					Selector: workloadSelector(),
					Template: podTemplate(templateLabels),
				},
			}
		},
		newVictim: func(obj runtime.Object) (*victims.Victim, error) { return NewDaemonSet(obj.(*appsv1.DaemonSet)) },
		eligible:  EligibleDaemonSets,
	},
}

func objectMeta(labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: NAME, Namespace: NAMESPACE, Labels: labels}
}

func workloadSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: map[string]string{"app": NAME}}
}

func podTemplate(labels map[string]string) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}}
}

// enrolledLabels is a workload set up the way the docs ask for
func enrolledLabels() map[string]string {
	return map[string]string{
		config.IdentLabelKey:     IDENTIFIER,
		config.MtbfLabelKey:      "1",
		config.EnabledLabelKey:   config.EnabledLabelValue,
		config.KillTypeLabelKey:  config.KillFixedLabelValue,
		config.KillValueLabelKey: "2",
	}
}

func appLabels() map[string]string {
	return map[string]string{"app": NAME}
}

func eachKind(t *testing.T, test func(t *testing.T, kind int)) {
	t.Helper()
	for i, kind := range kinds {
		t.Run(kind.name, func(t *testing.T) { test(t, i) })
	}
}

func TestNewReadsTheLabels(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]

		victim, err := kind.newVictim(kind.newObject(enrolledLabels(), appLabels()))

		require.NoError(t, err)
		assert.Equal(t, kind.name, victim.Kind())
		assert.Equal(t, NAME, victim.Name())
		assert.Equal(t, NAMESPACE, victim.Namespace())
		assert.Equal(t, IDENTIFIER, victim.Identifier())
		assert.Equal(t, 24*time.Hour, victim.Mtbf())
	})
}

func TestNewReadsMtbfUnits(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]
		labels := enrolledLabels()
		labels[config.MtbfLabelKey] = "90m"

		victim, err := kind.newVictim(kind.newObject(labels, appLabels()))

		require.NoError(t, err)
		assert.Equal(t, 90*time.Minute, victim.Mtbf())
	})
}

// Without the identifier on the pod template there is no way to tell the pods
// apart from the labels, so the workload's own selector has to do the job
func TestNewFallsBackToTheWorkloadSelector(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]

		victim, err := kind.newVictim(kind.newObject(enrolledLabels(), appLabels()))

		require.NoError(t, err)
		assert.Equal(t, "app="+NAME, victim.PodSelector().String())
	})
}

func TestNewUsesTheIdentifierOnThePodTemplate(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]

		victim, err := kind.newVictim(kind.newObject(enrolledLabels(), map[string]string{config.IdentLabelKey: IDENTIFIER}))

		require.NoError(t, err)
		assert.Equal(t, config.IdentLabelKey+"="+IDENTIFIER, victim.PodSelector().String())
	})
}

func TestNewRejectsAWorkloadItCannotRead(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]

		for name, labels := range map[string]map[string]string{
			"no identifier": {config.MtbfLabelKey: "1"},
			"no mtbf":       {config.IdentLabelKey: IDENTIFIER},
			"unreadable mtbf": {
				config.IdentLabelKey: IDENTIFIER,
				config.MtbfLabelKey:  "whenever",
			},
		} {
			t.Run(name, func(t *testing.T) {
				_, err := kind.newVictim(kind.newObject(labels, appLabels()))

				assert.ErrorContains(t, err, kind.name+" "+NAME)
			})
		}
	})
}

func TestEligibleReturnsTheWorkloadsThatOptedIn(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]
		client := fake.NewSimpleClientset(kind.newObject(enrolledLabels(), appLabels()))

		found, err := kind.eligible(client, NAMESPACE, metav1.ListOptions{})

		require.NoError(t, err)
		require.Len(t, found, 1)
		assert.Equal(t, NAME, found[0].Name())
		assert.Equal(t, kind.name, found[0].Kind())
	})
}

// One workload missing a label must not cost the rest of the cluster its
// schedule, so it is skipped rather than failing the whole fetch
func TestEligibleSkipsWorkloadsItCannotRead(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]

		good := kind.newObject(enrolledLabels(), appLabels())
		bad := kind.newObject(map[string]string{config.MtbfLabelKey: "1"}, appLabels())
		bad.(interface{ SetName(string) }).SetName("no-identifier")

		found, err := kind.eligible(fake.NewSimpleClientset(good, bad), NAMESPACE, metav1.ListOptions{})

		require.NoError(t, err)
		require.Len(t, found, 1)
		assert.Equal(t, NAME, found[0].Name())
	})
}

func TestEligibleAppliesTheFilter(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]
		client := fake.NewSimpleClientset(kind.newObject(enrolledLabels(), appLabels()))

		found, err := kind.eligible(client, NAMESPACE, metav1.ListOptions{LabelSelector: "kube-monkey/enabled=nobody"})

		require.NoError(t, err)
		assert.Empty(t, found)
	})
}

// Terminations are scheduled hours ahead, so the labels are read again at kill
// time rather than trusted from when the schedule was drawn up
func TestVictimReadsItsLabelsAgainAtKillTime(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]

		victim, err := kind.newVictim(kind.newObject(enrolledLabels(), appLabels()))
		require.NoError(t, err)

		optedOut := enrolledLabels()
		optedOut[config.EnabledLabelKey] = "disabled"
		optedOut[config.KillTypeLabelKey] = config.KillAllLabelValue
		optedOut[config.KillValueLabelKey] = "9"
		client := fake.NewSimpleClientset(kind.newObject(optedOut, appLabels()))

		enrolled, err := victim.IsEnrolled(client)
		require.NoError(t, err)
		assert.False(t, enrolled, "the victim opted out after it was scheduled")

		killType, err := victim.KillType(client)
		require.NoError(t, err)
		assert.Equal(t, config.KillAllLabelValue, killType)

		killValue, err := victim.KillValue(client)
		require.NoError(t, err)
		assert.Equal(t, 9, killValue)
	})
}

func TestVictimThatIsGoneAtKillTime(t *testing.T) {
	eachKind(t, func(t *testing.T, i int) {
		kind := kinds[i]

		victim, err := kind.newVictim(kind.newObject(enrolledLabels(), appLabels()))
		require.NoError(t, err)

		empty := fake.NewSimpleClientset()

		_, err = victim.IsEnrolled(empty)
		assert.ErrorContains(t, err, "not found")

		_, err = victim.KillType(empty)
		assert.ErrorContains(t, err, "not found")
	})
}
