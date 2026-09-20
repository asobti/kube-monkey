package factory

import (
	"testing"

	"kube-monkey/internal/pkg/config"
	"kube-monkey/internal/pkg/config/param"
	"kube-monkey/internal/pkg/victims"
	"kube-monkey/internal/pkg/victims/factory/deployments"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func victimInNamespace(t *testing.T, namespace string) victims.Victim {
	t.Helper()

	victim, err := deployments.New(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: namespace,
			Labels: map[string]string{
				config.IdentLabelKey: "app",
				config.MtbfLabelKey:  "1",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "app"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "app"}},
			},
		},
	})
	assert.NoError(t, err)

	return victim
}

func namespacesOf(allowed []victims.Victim) []string {
	namespaces := make([]string, 0, len(allowed))
	for _, victim := range allowed {
		namespaces = append(namespaces, victim.Namespace())
	}
	return namespaces
}

func TestInAllowedNamespace(t *testing.T) {
	viper.Reset()
	config.SetDefaults()
	viper.Set(param.WhitelistedNamespaces, []string{"team-*"})
	viper.Set(param.BlacklistedNamespaces, []string{"*-prod"})

	candidates := []victims.Victim{
		victimInNamespace(t, "team-shop"),
		victimInNamespace(t, "team-checkout"),
		victimInNamespace(t, "team-shop-prod"),
		victimInNamespace(t, "default"),
	}

	allowed := InAllowedNamespace(candidates)

	assert.ElementsMatch(t, []string{"team-shop", "team-checkout"}, namespacesOf(allowed))
}

func TestInAllowedNamespaceWithDefaultConfig(t *testing.T) {
	viper.Reset()
	config.SetDefaults()

	candidates := []victims.Victim{
		victimInNamespace(t, metav1.NamespaceDefault),
		victimInNamespace(t, metav1.NamespaceSystem),
	}

	allowed := InAllowedNamespace(candidates)

	assert.Equal(t, []string{metav1.NamespaceDefault}, namespacesOf(allowed))
}
