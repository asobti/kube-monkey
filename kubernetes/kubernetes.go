/*
Package kubernetes is the km k8 package that sets up the configured k8 clientset used to communicate with the apiserver

Use CreateClient to create and verify connectivity.
It's recommended to create a new clientset after a period of inactivity
*/
package kubernetes

import (
	"fmt"

	"github.com/golang/glog"

	cfg "github.com/asobti/kube-monkey/config"

	"k8s.io/client-go/discovery"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// CreateClient creates, verifies and returns an instance of k8 clientset
func CreateClient() (*kube.Clientset, error) {
	client, err := NewInClusterClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to generate NewInClusterClient: %v", err)
	}

	if VerifyClient(client) {
		return client, nil
	}
	return nil, fmt.Errorf("Unable to verify client connectivity to Kubernetes apiserver")
}

// NewInClusterClient only creates an initialized instance of k8 clientset
func NewInClusterClient() (*kube.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("failed to obtain config from InClusterConfig: %v", err)
		return nil, err
	}

	if apiserverHost, override := cfg.ClusterAPIServerHost(); override {
		glog.V(5).Infof("API server host overridden to: %s\n", apiserverHost)
		config.Host = apiserverHost
	}

	clientset, err := kube.NewForConfig(config)
	if err != nil {
		glog.Errorf("failed to create clientset in NewForConfig: %v", err)
		return nil, err
	}
	return clientset, nil
}

func VerifyClient(client discovery.DiscoveryInterface) bool {
	_, err := client.ServerVersion()
	return err == nil
}
