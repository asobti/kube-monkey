package chaos

import (
	"errors"
	"testing"
	"time"

	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/victims"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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

func newChaos() *Chaos {
	return &Chaos{
		killAt: time.Now(),
		victim: newVictimMock(),
	}
}

type ChaosTestSuite struct {
	suite.Suite
	chaos  *Chaos
	client kube.Interface
}

func (s *ChaosTestSuite) SetupTest() {
	s.chaos = newChaos()
	s.client = fake.NewSimpleClientset()
}

func (s *ChaosTestSuite) TestVerifyExecutionNotEnrolled() {
	v := s.chaos.victim.(*victimMock)
	v.On("IsEnrolled", s.client).Return(false, nil)
	err := s.chaos.verifyExecution(s.client)
	v.AssertExpectations(s.T())
	s.EqualError(err, v.Kind()+" "+v.Name()+" is no longer enrolled in kube-monkey. Skipping\n")
}

func (s *ChaosTestSuite) TestVerifyExecutionBlacklisted() {
	v := s.chaos.victim.(*victimMock)
	v.On("IsEnrolled", s.client).Return(true, nil)
	v.On("IsBlacklisted").Return(true)
	err := s.chaos.verifyExecution(s.client)
	v.AssertExpectations(s.T())
	s.EqualError(err, v.Kind()+" "+v.Name()+" is blacklisted. Skipping\n")
}

func (s *ChaosTestSuite) TestVerifyExecutionNotWhitelisted() {
	v := s.chaos.victim.(*victimMock)
	v.On("IsEnrolled", s.client).Return(true, nil)
	v.On("IsBlacklisted").Return(false)
	v.On("IsWhitelisted").Return(false)
	err := s.chaos.verifyExecution(s.client)
	v.AssertExpectations(s.T())
	s.EqualError(err, v.Kind()+" "+v.Name()+" is not whitelisted. Skipping\n")
}

func (s *ChaosTestSuite) TestVerifyExecutionWhitelisted() {
	v := s.chaos.victim.(*victimMock)
	v.On("IsEnrolled", s.client).Return(true, nil)
	v.On("IsBlacklisted").Return(false)
	v.On("IsWhitelisted").Return(true)
	err := s.chaos.verifyExecution(s.client)
	v.AssertExpectations(s.T())
	s.NoError(err)
}

func (s *ChaosTestSuite) TestTerminateKillTypeError() {
	v := s.chaos.victim.(*victimMock)
	errMsg := "KillType Error"
	v.On("KillType", s.client).Return("", errors.New(errMsg))
	v.On("DeleteRandomPod", s.client).Return(nil)
	_ = s.chaos.terminate(s.client)
	v.AssertExpectations(s.T())
}

func (s *ChaosTestSuite) TestTerminateAll() {
	v := s.chaos.victim.(*victimMock)
	v.On("KillType", s.client).Return(config.KillAllLabelValue, nil)
	v.On("TerminateAllPods", s.client).Return(nil)
	_ = s.chaos.terminate(s.client)
	v.AssertExpectations(s.T())
}

func (s *ChaosTestSuite) TestTerminateKillValueError() {
	v := s.chaos.victim.(*victimMock)
	errMsg := "KillValue Error"
	v.On("KillType", s.client).Return("", nil)
	v.On("KillValue", s.client).Return(0, errors.New(errMsg))
	v.On("DeleteRandomPod", s.client).Return(nil)
	_ = s.chaos.terminate(s.client)
	v.AssertExpectations(s.T())
}

func (s *ChaosTestSuite) TestTerminateKillFixed() {
	v := s.chaos.victim.(*victimMock)
	killValue := 1
	v.On("KillType", s.client).Return(config.KillFixedLabelValue, nil)
	v.On("KillValue", s.client).Return(killValue, nil)
	v.On("DeleteRandomPods", s.client, killValue).Return(nil)
	_ = s.chaos.terminate(s.client)
	v.AssertExpectations(s.T())
}

func (s *ChaosTestSuite) TestTerminateKillRandom() {
	v := s.chaos.victim.(*victimMock)
	killValue := 1
	v.On("KillType", s.client).Return(config.KillRandomLabelValue, nil)
	v.On("KillValue", s.client).Return(killValue, nil)
	v.On("DeleteRandomPods", s.client, mock.AnythingOfType("int")).Return(nil)
	_ = s.chaos.terminate(s.client)
	v.AssertExpectations(s.T())
}

func (s *ChaosTestSuite) TestDurationToKillTime() {
	t := s.chaos.DurationToKillTime()
	s.WithinDuration(s.chaos.KillAt(), time.Now(), t+time.Millisecond)
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(ChaosTestSuite))
}
