package kubernetes

import (
	kube "k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/rest"
)

func NewInClusterClient() (*kube.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kube.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func VerifyClient(client *kube.Clientset) bool {
	_, err := client.ServerVersion()
	return err == nil
}
