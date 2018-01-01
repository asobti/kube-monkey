package victims

import (
	"github.com/asobti/kube-monkey/config"

	kube "k8s.io/client-go/kubernetes"

	"k8s.io/api/core/v1"
        "k8s.io/apimachinery/pkg/labels"
        "k8s.io/apimachinery/pkg/selection"
        "k8s.io/apimachinery/pkg/util/sets"
        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Victim interface {
	VictimBaseTemplate
	VictimApiCalls
}

type VictimBaseTemplate interface {
        // Get value methods
        Kind()          string
        Name()          string
        Namespace()     string
        Identifier()    string
	Mtbf()          int
}

type VictimApiCalls interface {
        // Exposed Api Calls
        RunningPods(*kube.Clientset) ([]v1.Pod, error)
        Pods(*kube.Clientset) ([]v1.Pod, error)
        DeletePod(*kube.Clientset, string) error
	IsEnrolled(*kube.Clientset) (bool, error)
	HasKillAll(*kube.Clientset) (bool, error)
	IsBlacklisted(sets.String) bool
}

type VictimBase struct {
	kind            string
        name            string
        namespace       string
        identifier      string
        mtbf            int
	
	VictimBaseTemplate
}

func New(kind, name, namespace, identifier string, mtbf int) *VictimBase {
	return &VictimBase{kind: kind, name: name, namespace: namespace, identifier: identifier, mtbf: mtbf}
}

func (v *VictimBase) Kind() string {
        return v.kind
}

func (v *VictimBase) Name() string {
        return v.name
}

func (v *VictimBase) Namespace() string {
        return v.namespace
}

func (v *VictimBase) Identifier() string {
        return v.identifier
}

func (v *VictimBase) Mtbf() int {
        return v.mtbf
}
      
// Create a label filter to filter only for pods that belong to the this
// victim. This is done using the identifier label
func LabelFilterForPods(identifier string) (*metav1.ListOptions, error) {
        req, err := labelRequirementForPods(identifier)
        if err != nil {
                return nil, err
        }
        labelFilter := &metav1.ListOptions{
                LabelSelector: labels.NewSelector().Add(*req).String(),
        }
        return labelFilter, nil
}

// Create a labels.Requirement that can be used to build a filter
func labelRequirementForPods(identifier string) (*labels.Requirement, error) {
        return labels.NewRequirement(config.IdentLabelKey, selection.Equals, sets.NewString(identifier).UnsortedList())
}

