package chaos

import (
	"errors"
	"testing"
	"time"

	"github.com/asobti/kube-monkey/config"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type ChaosTestSuite struct {
	suite.Suite
	chaos  *Chaos
	client kube.Interface
}

func (s *ChaosTestSuite) SetupTest() {
	s.chaos = NewMock()
	s.client = fake.NewSimpleClientset()
}

func (s *ChaosTestSuite) TestVerifyExecutionNotEnrolled() {
	v := s.chaos.victim.(*victimMock)
	v.On("IsEnrolled", s.client).Return(false, nil)
	err := s.chaos.verifyExecution(s.client)
	v.AssertExpectations(s.T())
	s.EqualError(err, v.Kind()+" "+v.Name()+" is no longer enrolled in kube-monkey. Skipping")
}

func (s *ChaosTestSuite) TestVerifyExecutionBlacklisted() {
	v := s.chaos.victim.(*victimMock)
	v.On("IsEnrolled", s.client).Return(true, nil)
	v.On("IsBlacklisted").Return(true)
	err := s.chaos.verifyExecution(s.client)
	v.AssertExpectations(s.T())
	s.EqualError(err, v.Kind()+" "+v.Name()+" is blacklisted. Skipping")
}

func (s *ChaosTestSuite) TestVerifyExecutionNotWhitelisted() {
	v := s.chaos.victim.(*victimMock)
	v.On("IsEnrolled", s.client).Return(true, nil)
	v.On("IsBlacklisted").Return(false)
	v.On("IsWhitelisted").Return(false)
	err := s.chaos.verifyExecution(s.client)
	v.AssertExpectations(s.T())
	s.EqualError(err, v.Kind()+" "+v.Name()+" is not whitelisted. Skipping")
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
	err := errors.New("KillType Error")
	v.On("KillType", s.client).Return("", err)

	s.NotNil(s.chaos.terminate(s.client))
	v.AssertExpectations(s.T())
}

func (s *ChaosTestSuite) TestTerminateKillValueError() {
	v := s.chaos.victim.(*victimMock)
	errMsg := "KillValue Error"
	v.On("KillType", s.client).Return(config.KillFixedLabelValue, nil)
	v.On("KillValue", s.client).Return(0, errors.New(errMsg))
	s.NotNil(s.chaos.terminate(s.client))
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

func (s *ChaosTestSuite) TestTerminateAllPods() {
	v := s.chaos.victim.(*victimMock)
	v.On("KillType", s.client).Return(config.KillAllLabelValue, nil)
	v.On("KillValue", s.client).Return(0, nil)
	v.On("KillNumberForKillingAll", s.client).Return(0, nil)
	v.On("DeleteRandomPods", s.client, 0).Return(nil)
	_ = s.chaos.terminate(s.client)
	v.AssertExpectations(s.T())
}

func (s *ChaosTestSuite) TestTerminateKillRandomMaxPercentage() {
	v := s.chaos.victim.(*victimMock)
	killValue := 1
	v.On("KillType", s.client).Return(config.KillRandomMaxLabelValue, nil)
	v.On("KillValue", s.client).Return(killValue, nil)
	v.On("KillNumberForMaxPercentage", s.client, mock.AnythingOfType("int")).Return(0, nil)
	v.On("DeleteRandomPods", s.client, 0).Return(nil)
	_ = s.chaos.terminate(s.client)
	v.AssertExpectations(s.T())
}

//func (s *ChaosTestSuite) TestTerminateKillPodDisruptionBudget() {
//	v := s.chaos.victim.(*victimMock)
//	killValue := 1
//
//	v.On("Selector", s.client).Return(&v1.LabelSelector{}, nil)
//	v.On("KillType", s.client).Return(config.KillPodDisruptionBudgetLabelValue, nil)
//	v.On("KillValue", s.client).Return(killValue, nil)
//	v.On("KillNumberForKillingPodDisruptionBudget", s.client, mock.AnythingOfType("int"), mock.Anything).Return(0, nil)
//	v.On("DeleteRandomPods", s.client, 0).Return(nil)
//	_ = s.chaos.terminate(s.client)
//	v.AssertExpectations(s.T())
//}

//func (s *ChaosTestSuite) TestTerminateKillPodDisruptionBudgetError() {
//	v := s.chaos.victim.(*victimMock)
//	killValue := 1
//	desiredNumberOfPods := 2
//
//	errMsg := "Error Getting Selector"
//	v.On("Selector", s.client).Return(&v1.LabelSelector{}, errors.New(errMsg))
//	v.On("DesiredNumberOfPods", s.client).Return(desiredNumberOfPods, errors.New(errMsg))
//	v.On("PodDisruptionBudget", s.client).Return(1, 2, errors.New(errMsg))
//	v.On("PodsBySelector", s.client).Return(1, 2, errors.New(errMsg))
//	v.On("KillType", s.client).Return(config.KillPodDisruptionBudgetLabelValue, nil)
//	v.On("KillValue", s.client).Return(killValue, nil)
//	v.On("KillNumberForKillingPodDisruptionBudget", s.client, mock.AnythingOfType("int"), mock.Anything).Return(0, nil)
//	v.On("DeleteRandomPods", s.client, 0).Return(nil)
//
//	s.NotNil(s.chaos.terminate(s.client))
//	v.AssertExpectations(s.T())
//}

func (s *ChaosTestSuite) TestTerminateKillFixedPercentage() {
	v := s.chaos.victim.(*victimMock)
	killValue := 1
	v.On("KillType", s.client).Return(config.KillFixedPercentageLabelValue, nil)
	v.On("KillValue", s.client).Return(killValue, nil)
	v.On("KillNumberForFixedPercentage", s.client, mock.AnythingOfType("int")).Return(0, nil)
	v.On("DeleteRandomPods", s.client, 0).Return(nil)
	_ = s.chaos.terminate(s.client)
	v.AssertExpectations(s.T())
}

func (s *ChaosTestSuite) TestInvalidKillType() {
	v := s.chaos.victim.(*victimMock)
	v.On("KillType", s.client).Return("InvalidKillTypeHere", nil)
	v.On("KillValue", s.client).Return(0, nil)
	err := s.chaos.terminate(s.client)
	v.AssertExpectations(s.T())
	s.NotNil(err)
}

func (s *ChaosTestSuite) TestGetKillValue() {
	v := s.chaos.victim.(*victimMock)
	killValue := 5
	v.On("KillValue", s.client).Return(killValue, nil)
	result, err := s.chaos.getKillValue(s.client)
	s.Nil(err)
	s.Equal(killValue, result)
}

func (s *ChaosTestSuite) TestGetKillValueReturnsError() {
	v := s.chaos.victim.(*victimMock)
	v.On("KillValue", s.client).Return(0, errors.New("InvalidKillValue"))
	_, err := s.chaos.getKillValue(s.client)
	s.NotNil(err)
}

func (s *ChaosTestSuite) TestDurationToKillTime() {
	t := s.chaos.DurationToKillTime()
	s.WithinDuration(s.chaos.KillAt(), time.Now(), t+time.Millisecond)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(ChaosTestSuite))
}
