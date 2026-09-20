package customresources

import (
	"testing"

	"kube-monkey/internal/pkg/config"

	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEligibleCustomResources(t *testing.T) {
	cluster := newCluster(NAME, enrolledLabels())
	client := newClient(cluster)

	victims, err := EligibleCustomResources(client, cnpg, NAMESPACE, &metav1.ListOptions{})

	assert.NoError(t, err)
	assert.Len(t, victims, 1)
	assert.Equal(t, NAME, victims[0].Name())
}

// A resource missing a label kube-monkey needs is skipped on its own, so one bad
// resource does not cost the rest of the namespace its schedule
func TestEligibleCustomResourcesSkipsUnreadableOnes(t *testing.T) {
	good := newCluster(NAME, enrolledLabels())
	bad := newCluster("no-identifier", map[string]string{config.MtbfLabelKey: "1"})

	victims, err := EligibleCustomResources(newClient(good, bad), cnpg, NAMESPACE, &metav1.ListOptions{})

	assert.NoError(t, err)
	assert.Len(t, victims, 1)
	assert.Equal(t, NAME, victims[0].Name())
}

func TestIsEnrolled(t *testing.T) {
	cluster := newCluster(NAME, enrolledLabels())
	victim, err := New(newClient(cluster), cnpg, cluster)
	assert.NoError(t, err)

	enrolled, err := victim.IsEnrolled(nil)

	assert.NoError(t, err)
	assert.True(t, enrolled)
}

func TestIsNotEnrolled(t *testing.T) {
	labels := enrolledLabels()
	labels[config.EnabledLabelKey] = "x"
	cluster := newCluster(NAME, labels)

	victim, err := New(newClient(cluster), cnpg, cluster)
	assert.NoError(t, err)

	enrolled, err := victim.IsEnrolled(nil)

	assert.NoError(t, err)
	assert.False(t, enrolled)
}

// The labels are read again at kill time, so a resource that opted out after it
// was scheduled is left alone
func TestIsEnrolledReadsTheLiveResource(t *testing.T) {
	cluster := newCluster(NAME, enrolledLabels())
	victim, err := New(newClient(cluster), cnpg, cluster)
	assert.NoError(t, err)

	optedOut := newCluster(NAME, map[string]string{
		config.IdentLabelKey:   "1",
		config.MtbfLabelKey:    "1",
		config.EnabledLabelKey: "x",
	})
	victim.client = newClient(optedOut)

	enrolled, err := victim.IsEnrolled(nil)

	assert.NoError(t, err)
	assert.False(t, enrolled)
}

func TestKillType(t *testing.T) {
	cluster := newCluster(NAME, enrolledLabels())
	victim, err := New(newClient(cluster), cnpg, cluster)
	assert.NoError(t, err)

	killType, err := victim.KillType(nil)

	assert.NoError(t, err)
	assert.Equal(t, config.KillFixedLabelValue, killType)
}

func TestKillTypeWithoutTheLabel(t *testing.T) {
	labels := enrolledLabels()
	delete(labels, config.KillTypeLabelKey)
	cluster := newCluster(NAME, labels)

	victim, err := New(newClient(cluster), cnpg, cluster)
	assert.NoError(t, err)

	_, err = victim.KillType(nil)

	assert.ErrorContains(t, err, config.KillTypeLabelKey)
}

func TestKillValue(t *testing.T) {
	labels := enrolledLabels()
	labels[config.KillValueLabelKey] = "2"
	cluster := newCluster(NAME, labels)

	victim, err := New(newClient(cluster), cnpg, cluster)
	assert.NoError(t, err)

	killValue, err := victim.KillValue(nil)

	assert.NoError(t, err)
	assert.Equal(t, 2, killValue)
}

func TestKillValueRejectsNonsense(t *testing.T) {
	labels := enrolledLabels()
	labels[config.KillValueLabelKey] = "lots"
	cluster := newCluster(NAME, labels)

	victim, err := New(newClient(cluster), cnpg, cluster)
	assert.NoError(t, err)

	_, err = victim.KillValue(nil)

	assert.ErrorContains(t, err, "lots")
}
