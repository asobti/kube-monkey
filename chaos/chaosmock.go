package chaos

import (
	"k8s.io/apimachinery/pkg/util/intstr"
	"time"

	"github.com/asobti/kube-monkey/victims"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
)

const (
	NAMESPACE  = metav1.NamespaceDefault
	IDENTIFIER = "kube-monkey-id"
	KIND       = "Pod"
	NAME       = "name"
)

type victimMock struct {
	mock.Mock
	victims.VictimBase
}

// Returns the selector associated with this statefulset
func (vm *victimMock) Selector(clientset kube.Interface) (*metav1.LabelSelector, error) {
	args := vm.Called(clientset)
	selector, _ := args.Get(0).(*metav1.LabelSelector)
	return selector, args.Error(1)
}

// Returns the selector associated with this statefulset
func (vm *victimMock) DesiredNumberOfPods(clientset kube.Interface) (int, error) {
	args := vm.Called(clientset)
	return args.Int(0), args.Error(1)
}

func (vm *victimMock) IsEnrolled(clientset kube.Interface) (bool, error) {
	args := vm.Called(clientset)
	return args.Bool(0), args.Error(1)
}

func (vm *victimMock) KillType(clientset kube.Interface) (string, error) {
	args := vm.Called(clientset)
	return args.String(0), args.Error(1)
}

func (vm *victimMock) KillValue(clientset kube.Interface) (int, error) {
	args := vm.Called(clientset)
	return args.Int(0), args.Error(1)
}

func (vm *victimMock) DeleteRandomPod(clientset kube.Interface) error {
	args := vm.Called(clientset)
	return args.Error(0)
}

func (vm *victimMock) DeleteRandomPods(clientset kube.Interface, killValue int) error {
	args := vm.Called(clientset, killValue)
	return args.Error(0)
}

func (vm *victimMock) KillNumberForKillingAll(clientset kube.Interface) (int, error) {
	args := vm.Called(clientset)
	return args.Int(0), args.Error(1)
}

func (vm *victimMock) KillNumberForMaxPercentage(clientset kube.Interface, killValue int) (int, error) {
	args := vm.Called(clientset, killValue)
	return args.Int(0), args.Error(1)
}

func (vm *victimMock) KillNumberForKillingPodDisruptionBudget(clientset kube.Interface, killValue int, minAvailable *intstr.IntOrString, maxUnavailable *intstr.IntOrString, desiredPodsForPDB int, healthyNumberOfPods int) (int, error) {
	args := vm.Called(clientset, killValue, minAvailable, maxUnavailable)
	return args.Int(0), args.Error(1)
}

func (vm *victimMock) KillNumberForFixedPercentage(clientset kube.Interface, killValue int) (int, error) {
	args := vm.Called(clientset, killValue)
	return args.Int(0), args.Error(1)
}

func (vm *victimMock) IsBlacklisted() bool {
	args := vm.Called()
	return args.Bool(0)
}

func (vm *victimMock) IsWhitelisted() bool {
	args := vm.Called()
	return args.Bool(0)
}

func newVictimMock() *victimMock {
	v := victims.New(KIND, NAME, NAMESPACE, IDENTIFIER, 1)
	return &victimMock{
		VictimBase: *v,
	}
}

func NewMock() *Chaos {
	return &Chaos{
		killAt: time.Now(),
		victim: newVictimMock(),
	}
}
