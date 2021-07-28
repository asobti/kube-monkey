package kubernetes

import (
	"errors"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"

	"k8s.io/client-go/discovery"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestCreateClient(t *testing.T) {

	defer monkey.UnpatchAll()

	_, err := CreateClient()
	assert.Errorf(t, err, "Expected an error if NewInClusterClient fails")

	monkey.Patch(rest.InClusterConfig, func() (*rest.Config, error) {
		return &rest.Config{}, nil
	})
	_, err = CreateClient()
	assert.Errorf(t, err, "Expected an error if VerifyClient fails")

	monkey.Patch(VerifyClient, func(client discovery.DiscoveryInterface) bool {
		return true
	})
	client, err := CreateClient()
	assert.Nil(t, err)
	assert.Implements(t, (*discovery.DiscoveryInterface)(nil), client)
}

func TestNewInClusterClient(t *testing.T) {
	defer monkey.UnpatchAll()

	_, err := NewInClusterClient()
	assert.Errorf(t, err, "Expected an error if client is not running inside a kubernetes environment")

	monkey.Patch(rest.InClusterConfig, func() (*rest.Config, error) {
		return &rest.Config{}, nil
	})
	client, err := NewInClusterClient()
	assert.Nil(t, err)
	assert.Implements(t, (*discovery.DiscoveryInterface)(nil), client)
}

func TestVerifyClient(t *testing.T) {
	defer monkey.UnpatchAll()

	client := fakeclientset.NewSimpleClientset()
	fakeDiscovery, _ := client.Discovery().(*fakediscovery.FakeDiscovery)
	assert.True(t, VerifyClient(fakeDiscovery))

	monkey.PatchInstanceMethod(reflect.TypeOf(fakeDiscovery), "ServerVersion", func(_ *fakediscovery.FakeDiscovery) (*version.Info, error) {
		return nil, errors.New("")
	})
	assert.False(t, VerifyClient(fakeDiscovery))
}
