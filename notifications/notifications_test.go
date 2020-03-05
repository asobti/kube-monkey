package notifications

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config"
	"github.com/bouk/monkey"
	"github.com/stretchr/testify/suite"
)

type NotificationsTestSuite struct {
	suite.Suite
	client      Client
	server      *httptest.Server
	currentTime time.Time
	result      *chaos.Result
}

func (s *NotificationsTestSuite) SetupTest() {
	//create HTTP client
	s.client = CreateClient()
	//start server
	s.server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {}))
	//create Result
	s.currentTime = time.Now()
	v := chaos.NewVictimMock()
	c := chaos.New(s.currentTime, v)
	s.result = chaos.NewResult(c, errors.New("Result Error"))
}

func (s *NotificationsTestSuite) TearDownTest() {
	defer s.server.Close()
}

func (s *NotificationsTestSuite) TestReportSuccessfulAttack() {
	//mock Receiver
	endpoint := s.server.URL + "/path"
	receiver := config.NewReceiver(endpoint, "message", []string{"header1Key:header1Value", "header2Key:header2Value"})
	monkey.Patch(config.NotificationsAttacks, func() []config.Receiver { return []config.Receiver{receiver} })
	defer monkey.Unpatch(config.NotificationsAttacks)

	success := ReportAttack(s.client, s.result, s.currentTime)
	s.Assert().True(success)
}

func (s *NotificationsTestSuite) TestReportUnsuccessfulAttack() {
	//mock Receiver
	endpoint := s.server.URL + "/path"
	receiver := config.NewReceiver(endpoint, "message", []string{"header1Key:header1Value", "header2Key:header2Value"})
	monkey.Patch(config.NotificationsAttacks, func() []config.Receiver { return []config.Receiver{receiver} })
	defer monkey.Unpatch(config.NotificationsAttacks)

	success := ReportAttack(s.client, s.result, s.currentTime)
	s.Assert().True(success)
}

func (s *NotificationsTestSuite) TestReportMultipleReceiversSuccess() {
	//mock Receivers
	endpoint1 := s.server.URL + "/path1"
	receiver1 := config.NewReceiver(endpoint1, "message", []string{"header1Key:header1Value", "header2Key:header2Value"})
	endpoint2 := s.server.URL + "/path2"
	receiver2 := config.NewReceiver(endpoint2, "message", []string{"header1Key:header1Value", "header2Key:header2Value"})
	monkey.Patch(config.NotificationsAttacks, func() []config.Receiver { return []config.Receiver{receiver1, receiver2} })
	defer monkey.Unpatch(config.NotificationsAttacks)

	success := ReportAttack(s.client, s.result, s.currentTime)
	s.Assert().True(success)
}

func (s *NotificationsTestSuite) TestReportMultipleReceiversFailure() {
	//mock Receivers
	validEndpoint := s.server.URL + "/path1"
	receiver1 := config.NewReceiver(validEndpoint, "message", []string{"header1Key:header1Value", "header2Key:header2Value"})
	invalidEndpoint := "/path2"
	receiver2 := config.NewReceiver(invalidEndpoint, "message", []string{"header1Key:header1Value", "header2Key:header2Value"})
	monkey.Patch(config.NotificationsAttacks, func() []config.Receiver { return []config.Receiver{receiver1, receiver2} })
	defer monkey.Unpatch(config.NotificationsAttacks)

	success := ReportAttack(s.client, s.result, s.currentTime)
	s.Assert().False(success)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(NotificationsTestSuite))
}
