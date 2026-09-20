package customresources

import (
	"testing"
	"time"

	"kube-monkey/internal/pkg/config"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	NAME      = "cluster"
	NAMESPACE = "app"
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
	scheme := runtime.NewScheme()
	listKinds := map[schema.GroupVersionResource]string{
		GroupVersionResource(cnpg): "ClusterList",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, objects...)
}

func TestNewReadsTheLabels(t *testing.T) {
	cluster, err := New(newClient(), cnpg, newCluster(NAME, enrolledLabels()))

	assert.NoError(t, err)
	assert.Equal(t, "clusters.postgresql.cnpg.io", cluster.Kind())
	assert.Equal(t, NAME, cluster.Name())
	assert.Equal(t, NAMESPACE, cluster.Namespace())
	assert.Equal(t, "1", cluster.Identifier())
	assert.Equal(t, 24*time.Hour, cluster.Mtbf())
}

// The pods an operator creates are named after the resource they belong to, not
// after the labels sitting on the resource itself
func TestPodSelectorUsesTheConfiguredLabel(t *testing.T) {
	cluster, err := New(newClient(), cnpg, newCluster(NAME, enrolledLabels()))

	assert.NoError(t, err)
	assert.Equal(t, "cnpg.io/cluster=cluster", cluster.PodSelector().String())
}

func TestPodSelectorFallsBackToTheIdentifier(t *testing.T) {
	withoutPodLabel := cnpg
	withoutPodLabel.PodLabel = ""

	cluster, err := New(newClient(), withoutPodLabel, newCluster(NAME, enrolledLabels()))

	assert.NoError(t, err)
	assert.Equal(t, "kube-monkey/identifier=1", cluster.PodSelector().String())
}

func TestNewRequiresTheIdentifierLabel(t *testing.T) {
	labels := enrolledLabels()
	delete(labels, config.IdentLabelKey)

	_, err := New(newClient(), cnpg, newCluster(NAME, labels))

	assert.ErrorContains(t, err, config.IdentLabelKey)
}

func TestNewRejectsAnUnreadableMtbf(t *testing.T) {
	labels := enrolledLabels()
	labels[config.MtbfLabelKey] = "soon"

	_, err := New(newClient(), cnpg, newCluster(NAME, labels))

	assert.ErrorContains(t, err, "mtbf")
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

// The pods an operator creates carry the operator's own labels, not the labels
// sitting on the custom resource, so this is what makes a custom resource
// victim find anything to kill at all
func TestPodsFindsTheOperatorsPods(t *testing.T) {
	cluster, err := New(newClient(), cnpg, newCluster(NAME, enrolledLabels()))
	assert.NoError(t, err)

	pods := fake.NewSimpleClientset(
		newOperatorPod("cluster-1", map[string]string{"cnpg.io/cluster": NAME}),
		newOperatorPod("cluster-2", map[string]string{"cnpg.io/cluster": NAME}),
		newOperatorPod("other-1", map[string]string{"cnpg.io/cluster": "other"}),
		newOperatorPod("unrelated", map[string]string{"app": "web"}),
	)

	found, err := cluster.Pods(pods)

	assert.NoError(t, err)
	assert.Len(t, found, 2)
	assert.Equal(t, "cluster-1", found[0].Name)
	assert.Equal(t, "cluster-2", found[1].Name)
}

// Labels on the custom resource are not passed down to the pods unless the
// operator is asked to, so the identifier fallback finds nothing on its own
func TestPodsFindsNothingWhenTheLabelsAreNotPassedDown(t *testing.T) {
	withoutPodLabel := cnpg
	withoutPodLabel.PodLabel = ""

	cluster, err := New(newClient(), withoutPodLabel, newCluster(NAME, enrolledLabels()))
	assert.NoError(t, err)

	pods := fake.NewSimpleClientset(newOperatorPod("cluster-1", map[string]string{"cnpg.io/cluster": NAME}))

	found, err := cluster.Pods(pods)

	assert.NoError(t, err)
	assert.Empty(t, found)
}
