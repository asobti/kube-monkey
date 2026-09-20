package customresources

import (
	"context"
	"testing"
	"time"

	"kube-monkey/internal/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	NAME      = "cluster"
	NAMESPACE = "app"
	KIND      = "clusters.postgresql.cnpg.io"
)

var cnpg = config.CustomResource{
	Group:    "postgresql.cnpg.io",
	Version:  "v1",
	Resource: "clusters",
	PodLabel: "cnpg.io/cluster",
}

func newCluster(name string, labels map[string]string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("postgresql.cnpg.io/v1")
	obj.SetKind("Cluster")
	obj.SetName(name)
	obj.SetNamespace(NAMESPACE)
	obj.SetLabels(labels)
	return obj
}

func enrolledLabels() map[string]string {
	return map[string]string{
		config.IdentLabelKey:     "1",
		config.MtbfLabelKey:      "1",
		config.EnabledLabelKey:   config.EnabledLabelValue,
		config.KillTypeLabelKey:  config.KillFixedLabelValue,
		config.KillValueLabelKey: "1",
	}
}

func newClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	listKinds := map[schema.GroupVersionResource]string{
		groupVersionResource(cnpg): "ClusterList",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, objects...)
}

func updateCluster(t *testing.T, client dynamic.Interface, obj *unstructured.Unstructured) {
	t.Helper()

	_, err := client.Resource(groupVersionResource(cnpg)).Namespace(NAMESPACE).Update(context.TODO(), obj, metav1.UpdateOptions{})
	require.NoError(t, err)
}

func newOperatorPod(name string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NAMESPACE,
			Labels:    labels,
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
}

func TestNewReadsTheLabels(t *testing.T) {
	cluster, err := New(newClient(), cnpg, newCluster(NAME, enrolledLabels()))

	require.NoError(t, err)
	assert.Equal(t, KIND, cluster.Kind())
	assert.Equal(t, NAME, cluster.Name())
	assert.Equal(t, NAMESPACE, cluster.Namespace())
	assert.Equal(t, "1", cluster.Identifier())
	assert.Equal(t, 24*time.Hour, cluster.Mtbf())
}

func TestNewRejectsAResourceItCannotRead(t *testing.T) {
	for name, labels := range map[string]map[string]string{
		"no identifier":   {config.MtbfLabelKey: "1"},
		"no mtbf":         {config.IdentLabelKey: "1"},
		"unreadable mtbf": {config.IdentLabelKey: "1", config.MtbfLabelKey: "soon"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := New(newClient(), cnpg, newCluster(NAME, labels))

			assert.ErrorContains(t, err, KIND+" "+NAME)
		})
	}
}

// The pods an operator creates are named after the resource they belong to, not
// after the labels sitting on the resource itself
func TestPodSelectorUsesTheConfiguredLabel(t *testing.T) {
	cluster, err := New(newClient(), cnpg, newCluster(NAME, enrolledLabels()))

	require.NoError(t, err)
	assert.Equal(t, "cnpg.io/cluster=cluster", cluster.PodSelector().String())
}

func TestPodSelectorFallsBackToTheIdentifier(t *testing.T) {
	withoutPodLabel := cnpg
	withoutPodLabel.PodLabel = ""

	cluster, err := New(newClient(), withoutPodLabel, newCluster(NAME, enrolledLabels()))

	require.NoError(t, err)
	assert.Equal(t, config.IdentLabelKey+"=1", cluster.PodSelector().String())
}

// The pods an operator creates carry the operator's own labels, not the labels
// sitting on the custom resource, so this is what makes a custom resource
// victim find anything to kill at all
func TestPodsFindsTheOperatorsPods(t *testing.T) {
	cluster, err := New(newClient(), cnpg, newCluster(NAME, enrolledLabels()))
	require.NoError(t, err)

	pods := fake.NewSimpleClientset(
		newOperatorPod("cluster-1", map[string]string{"cnpg.io/cluster": NAME}),
		newOperatorPod("cluster-2", map[string]string{"cnpg.io/cluster": NAME}),
		newOperatorPod("other-1", map[string]string{"cnpg.io/cluster": "other"}),
		newOperatorPod("unrelated", map[string]string{"app": "web"}),
	)

	found, err := cluster.Pods(pods)

	require.NoError(t, err)
	require.Len(t, found, 2)
	assert.Equal(t, "cluster-1", found[0].Name)
	assert.Equal(t, "cluster-2", found[1].Name)
}

// Labels on the custom resource are not passed down to the pods unless the
// operator is asked to, so the identifier fallback finds nothing on its own
func TestPodsFindsNothingWhenTheLabelsAreNotPassedDown(t *testing.T) {
	withoutPodLabel := cnpg
	withoutPodLabel.PodLabel = ""

	cluster, err := New(newClient(), withoutPodLabel, newCluster(NAME, enrolledLabels()))
	require.NoError(t, err)

	pods := fake.NewSimpleClientset(newOperatorPod("cluster-1", map[string]string{"cnpg.io/cluster": NAME}))

	found, err := cluster.Pods(pods)

	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestEligibleCustomResources(t *testing.T) {
	client := newClient(newCluster(NAME, enrolledLabels()))

	found, err := EligibleCustomResources(client, cnpg, NAMESPACE, metav1.ListOptions{})

	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, NAME, found[0].Name())
	assert.Equal(t, KIND, found[0].Kind())
}

// A resource missing a label kube-monkey needs is skipped on its own, so one bad
// resource does not cost the rest of the namespace its schedule
func TestEligibleCustomResourcesSkipsUnreadableOnes(t *testing.T) {
	good := newCluster(NAME, enrolledLabels())
	bad := newCluster("no-identifier", map[string]string{config.MtbfLabelKey: "1"})

	found, err := EligibleCustomResources(newClient(good, bad), cnpg, NAMESPACE, metav1.ListOptions{})

	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, NAME, found[0].Name())
}

// The labels are read again at kill time, so a resource that opted out or moved
// the goalposts after it was scheduled is respected
func TestVictimReadsItsLabelsAgainAtKillTime(t *testing.T) {
	cluster := newCluster(NAME, enrolledLabels())
	client := newClient(cluster)

	victim, err := New(client, cnpg, cluster)
	require.NoError(t, err)

	changed := enrolledLabels()
	changed[config.EnabledLabelKey] = "disabled"
	changed[config.KillTypeLabelKey] = config.KillAllLabelValue
	changed[config.KillValueLabelKey] = "4"
	updateCluster(t, client, newCluster(NAME, changed))

	enrolled, err := victim.IsEnrolled(nil)
	require.NoError(t, err)
	assert.False(t, enrolled)

	killType, err := victim.KillType(nil)
	require.NoError(t, err)
	assert.Equal(t, config.KillAllLabelValue, killType)

	killValue, err := victim.KillValue(nil)
	require.NoError(t, err)
	assert.Equal(t, 4, killValue)
}

func TestVictimThatIsGoneAtKillTime(t *testing.T) {
	victim, err := New(newClient(), cnpg, newCluster(NAME, enrolledLabels()))
	require.NoError(t, err)

	_, err = victim.IsEnrolled(nil)

	assert.ErrorContains(t, err, "not found")
}
