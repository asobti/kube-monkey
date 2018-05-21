package victims

import (
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

type VictimMock struct {
	mock.Mock
	VictimBase
}

func (vm *VictimMock) IsEnrolled(clientset kube.Interface) (bool, error) {
	args := vm.Called(clientset)
	return args.Bool(0), args.Error(1)
}

func (vm *VictimMock) KillType(clientset kube.Interface) (string, error) {
	args := vm.Called(clientset)
	return args.String(0), args.Error(1)
}

func (vm *VictimMock) KillValue(clientset kube.Interface) (int, error) {
	args := vm.Called(clientset)
	return args.Int(0), args.Error(1)
}

func (vm *VictimMock) DeleteRandomPod(clientset kube.Interface) error {
	args := vm.Called(clientset)
	return args.Error(0)
}

func (vm *VictimMock) DeleteRandomPods(clientset kube.Interface, killValue int) error {
	args := vm.Called(clientset, killValue)
	return args.Error(0)
}

func (vm *VictimMock) TerminateAllPods(clientset kube.Interface) error {
	args := vm.Called(clientset)
	return args.Error(0)
}

func (vm *VictimMock) IsBlacklisted() bool {
	args := vm.Called()
	return args.Bool(0)
}

func (vm *VictimMock) IsWhitelisted() bool {
	args := vm.Called()
	return args.Bool(0)
}

func NewVictimMock() *VictimMock {
	v := New(KIND, NAME, NAMESPACE, IDENTIFIER, 1)
	return &VictimMock{
		VictimBase: *v,
	}
}
