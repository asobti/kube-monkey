package deployments

import (
	"github.com/golang/glog"

	"github.com/asobti/kube-monkey/victims"

	kube "k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Get all eligible deployments that opted in and are not in blacklisted nm
func EligibleDeployments(clientset *kube.Clientset, filter *metav1.ListOptions, blacklist sets.String) (deployVictims []victims.Victim, err error) {
	// Get all enrolled deployments, filtered by config EnabledLabel
	enabledDeployments, err := clientset.ExtensionsV1beta1().Deployments(metav1.NamespaceAll).List(*filter)
	if err != nil {
		return nil, err
	}

	for _, dep := range enabledDeployments.Items {
		deployment, err := New(&dep)
		if err != nil {
			glog.Warningf("Skipping eligible deployment %s because of error:\n%s\n", dep.Name, err.Error())
			continue
		}

		if deployment.IsBlacklisted(blacklist) {
			continue
		}

		deployVictims = append(deployVictims, deployment)
	}

	return
}
