/*
Package kubernetes is the km k8 package that sets up the configured k8 clientset used to communicate with the apiserver

Use CreateClient to create and verify connectivity.
It's recommended to create a new clientset after a period of inactivity
*/
package kubernetes

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"io"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
	"strings"

	cfg "github.com/asobti/kube-monkey/config"

	"k8s.io/client-go/discovery"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ExecOptions passed to ExecWithOptions
type ExecOptions struct {
	Command []string

	Namespace     string
	PodName       string
	ContainerName string

	Stdin         io.Reader
	CaptureStdout bool
	CaptureStderr bool
	// If false, whitespace in std{err,out} will be removed.
	PreserveWhitespace bool
}

// CreateClient creates, verifes and returns an instance of k8 clientset
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

func GetRestConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("failed to obtain config from InClusterConfig: %v", err)
		return nil, err
	}

	if apiserverHost, override := cfg.ClusterAPIServerHost(); override {
		glog.V(5).Infof("API server host overriden to: %s\n", apiserverHost)
		config.Host = apiserverHost
	}
	return config, err
}

func VerifyClient(client discovery.DiscoveryInterface) bool {
	_, err := client.ServerVersion()
	return err == nil
}

// ExecCommandInContainerWithFullOutput executes a command in the
// specified container and return stdout, stderr and error
func ExecCommandInContainerWithFullOutput(clientset kube.Interface, podName, containerName, namespace string, cmd ...string) (string, string, error) {
	return ExecWithOptions(ExecOptions{
		Command:       cmd,
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,

		Stdin:              nil,
		CaptureStdout:      true,
		CaptureStderr:      true,
		PreserveWhitespace: false,
	}, clientset)
}

// ExecWithOptions executes a command in the specified container,
// returning stdout, stderr and error. `options` allowed for
// additional parameters to be passed.
func ExecWithOptions(options ExecOptions, clientset kube.Interface) (string, string, error) {
	glog.Infof("ExecWithOptions: %v", options)

	restconfig, err := GetRestConfig()
	if err != nil {
		panic(err)
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(options.PodName).
		Namespace(options.Namespace).
		SubResource("exec").
		Param("container", options.ContainerName)

	req.VersionedParams(&v1.PodExecOptions{
		Container: options.ContainerName,
		Command:   options.Command,
		Stdin:     options.Stdin != nil,
		Stdout:    options.CaptureStdout,
		Stderr:    options.CaptureStderr,
		TTY:       false,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	err = execute("POST", req.URL(), restconfig, options.Stdin, &stdout, &stderr, false)

	if options.PreserveWhitespace {
		return stdout.String(), stderr.String(), err
	}
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    tty,
	})
}
