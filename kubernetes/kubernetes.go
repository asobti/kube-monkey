package kubernetes

import (
	"fmt"
	"os"
	
	cfg "github.com/asobti/kube-monkey/config"
	kube "k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/rest"
	"k8s.io/client-go/1.5/tools/clientcmd"
)

// Check if running in a cluster with service accounts
// Use case: firewall based/only cluster 
func RunningInCluster() (bool)  {
        if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
                return true;
        } else {
                return false;
        }
}

func GetConfig() (*rest.Config, error) {
        var kubeConfig rest.Config

        // Set the Kubernetes configuration based on the environment
        if RunningInCluster() {
                config, err := rest.InClusterConfig()

                if err != nil {
                        return nil, fmt.Errorf("Failed to create in-cluster config: %v.", err)
                }

                kubeConfig = *config
        } else {
                loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
                configOverrides := &clientcmd.ConfigOverrides{}
                config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
                tmpKubeConfig, err := config.ClientConfig()
                if err != nil {
                        return nil, fmt.Errorf("Failed to load local kube config: %v", err)
                }
                kubeConfig = *tmpKubeConfig;
        }

	if apiserverHost, override := cfg.ClusterAPIServerHost(); override {
		fmt.Printf("API server host overriden to: %s\n", apiserverHost)
		config.Host = apiserverHost
	}

        return &kubeConfig, nil
}

func NewInClusterClient() (*kube.Clientset, error) {
	config, err := GetConfig()
	if err != nil {
		fmt.Println(err)
		return nil, err
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
