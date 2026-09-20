/*
Package kubernetes is the km k8 package that sets up the configured k8 clientset used to communicate with the apiserver

Use CreateClient to create and verify connectivity.
It's recommended to create a new clientset after a period of inactivity
*/
package kubernetes

import (
	"errors"
	"fmt"

	"github.com/golang/glog"

	cfg "kube-monkey/internal/pkg/config"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// CreateClient creates, verifies and returns an instance of k8 clientset
func CreateClient() (*kube.Clientset, error) {
	client, err := newInClusterClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create a client for the apiserver: %v", err)
	}

	if verifyClient(client) {
		return client, nil
	}
	return nil, errors.New("unable to verify client connectivity to the kubernetes apiserver")
}

// newInClusterClient only creates an initialized instance of k8 clientset
func newInClusterClient() (*kube.Clientset, error) {
	config, err := inClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kube.NewForConfig(config)
	if err != nil {
		glog.Errorf("failed to create clientset in NewForConfig: %v", err)
		return nil, err
	}
	return clientset, nil
}

// NewDynamicClient creates a client that reads resources the typed clientset
// has no Go type for, which is how custom resources are reached
func NewDynamicClient() (dynamic.Interface, error) {
	config, err := inClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		glog.Errorf("failed to create dynamic client in NewForConfig: %v", err)
		return nil, err
	}
	return client, nil
}

func inClusterConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("failed to obtain config from InClusterConfig: %v", err)
		return nil, err
	}

	if apiserverHost, override := cfg.ClusterAPIServerHost(); override {
		glog.V(5).Infof("API server host overridden to: %s\n", apiserverHost)
		config.Host = apiserverHost
	}

	return config, nil
}

// verifyClient reports whether the apiserver answers
func verifyClient(client discovery.DiscoveryInterface) bool {
	_, err := client.ServerVersion()
	return err == nil
}
