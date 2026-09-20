package factory

import (
	"testing"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/config/param"
	"kube-monkey/internal/pkg/victims"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMain(m *testing.M) {
	config.SetDefaults()
	m.Run()
}

func victimInNamespace(namespace string) *victims.Victim {
	return victims.New(victims.Spec{Kind: "v1.Deployment", Name: "app", Namespace: namespace})
}

func namespacesOf(allowed []*victims.Victim) []string {
	namespaces := make([]string, 0, len(allowed))
	for _, victim := range allowed {
		namespaces = append(namespaces, victim.Namespace())
	}
	return namespaces
}

func TestInAllowedNamespace(t *testing.T) {
	viper.Set(param.WhitelistedNamespaces, []string{"team-*"})
	viper.Set(param.BlacklistedNamespaces, []string{"*-prod"})
	t.Cleanup(func() {
		viper.Reset()
		config.SetDefaults()
	})

	allowed := inAllowedNamespace([]*victims.Victim{
		victimInNamespace("team-shop"),
		victimInNamespace("team-checkout"),
		victimInNamespace("team-shop-prod"),
		victimInNamespace("default"),
	})

	assert.Equal(t, []string{"team-shop", "team-checkout"}, namespacesOf(allowed))
}

// Where the two lists overlap the blacklist has to win, otherwise a namespace
// someone deliberately protected could be opted back in by a broad whitelist
func TestInAllowedNamespaceBlacklistBeatsWhitelist(t *testing.T) {
	viper.Set(param.WhitelistedNamespaces, []string{"*"})
	viper.Set(param.BlacklistedNamespaces, []string{"team-shop"})
	t.Cleanup(func() {
		viper.Reset()
		config.SetDefaults()
	})

	allowed := inAllowedNamespace([]*victims.Victim{
		victimInNamespace("team-shop"),
		victimInNamespace("team-checkout"),
	})

	assert.Equal(t, []string{"team-checkout"}, namespacesOf(allowed))
}

// kube-system is blocked out of the box, everything else is open
func TestInAllowedNamespaceWithDefaultConfig(t *testing.T) {
	allowed := inAllowedNamespace([]*victims.Victim{
		victimInNamespace(metav1.NamespaceDefault),
		victimInNamespace(metav1.NamespaceSystem),
	})

	assert.Equal(t, []string{metav1.NamespaceDefault}, namespacesOf(allowed))
}

func TestEnrollmentFilterOnlyAsksForWorkloadsThatOptedIn(t *testing.T) {
	filter, err := enrollmentFilter()

	require.NoError(t, err)
	assert.Equal(t, config.EnabledLabelKey+"="+config.EnabledLabelValue, filter.LabelSelector)
}
