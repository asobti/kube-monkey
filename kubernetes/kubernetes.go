package kubernetes

import (
	"fmt"
	
	cfg "github.com/asobti/kube-monkey/config"
	
	kube "k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/rest"
)

func NewInClusterClient() (*kube.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if apiserverHost, override := cfg.ClusterAPIServerHost(); override {
		fmt.Printf("API server host overriden to: %s\n", apiserverHost)
		config.Host = apiserverHost
	}

	clientset, err := kube.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return clientset, nil
}

func VerifyClient(client *kube.Clientset) bool {
	_, err := client.ServerVersion()
	return err == nil
}
