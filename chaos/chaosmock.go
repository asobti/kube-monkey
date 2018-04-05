package chaos

import (
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

func (vm *victimMock) TerminateAllPods(clientset kube.Interface) error {
	args := vm.Called(clientset)
	return args.Error(0)
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
