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

	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Create, verify and return an instance of k8 clientset
func CreateClient() (*kube.Clientset, error) {
	client, err := NewInClusterClient()
	if err != nil {
		return nil, fmt.Errorf("Failed to generate NewInClusterClient: %v", err)
	}

	if VerifyClient(client) {
		return client, nil
	} else {
		return nil, fmt.Errorf("Unable to verify client connectivity to Kubernetes apiserver")
	}
}

// Only creates an initialized instance of k8 clientset
func NewInClusterClient() (*kube.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("failed to obtain config from InClusterConfig: %v", err)
		return nil, err
	}

	if apiserverHost, override := cfg.ClusterAPIServerHost(); override {
		glog.V(5).Infof("API server host overriden to: %s\n", apiserverHost)
		config.Host = apiserverHost
	}

	clientset, err := kube.NewForConfig(config)
	if err != nil {
		glog.Errorf("failed to create clientset in NewForConfig: %v", err)
		return nil, err
	}
	return clientset, nil
}

func VerifyClient(client *kube.Clientset) bool {
	_, err := client.ServerVersion()
	return err == nil
}