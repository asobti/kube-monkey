package notifications

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/asobti/kube-monkey/chaos"
	"github.com/asobti/kube-monkey/config"
	"github.com/asobti/kube-monkey/schedule"
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
	s.client = CreateClient("")
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

func (s *NotificationsTestSuite) TestReportSchedule() {
	//mock Receiver
	endpoint := s.server.URL + "/path"
	receiver := config.NewReceiver(endpoint, "message", []string{"header1Key:header1Value", "header2Key:header2Value"})
	monkey.Patch(config.NotificationsAttacks, func() config.Receiver { return receiver })
	defer monkey.Unpatch(config.NotificationsAttacks)

	sch := &schedule.Schedule{}
	e1 := chaos.NewMock()
	e2 := chaos.NewMock()
	sch.Add(e1)
	sch.Add(e2)

	success := ReportSchedule(s.client, sch)
	s.Assert().True(success)
}

func (s *NotificationsTestSuite) TestReportSuccessfulAttack() {
	//mock Receiver
	endpoint := s.server.URL + "/path"
	receiver := config.NewReceiver(endpoint, "message", []string{"header1Key:header1Value", "header2Key:header2Value"})
	monkey.Patch(config.NotificationsAttacks, func() config.Receiver { return receiver })
	defer monkey.Unpatch(config.NotificationsAttacks)

	success := ReportAttack(s.client, s.result, s.currentTime)
	s.Assert().True(success)
}

func (s *NotificationsTestSuite) TestReportUnsuccessfulAttack() {
	//mock Receiver
	endpoint := s.server.URL + "/path"
	receiver := config.NewReceiver(endpoint, "message", []string{"header1Key:header1Value", "header2Key:header2Value"})
	monkey.Patch(config.NotificationsAttacks, func() config.Receiver { return receiver })
	defer monkey.Unpatch(config.NotificationsAttacks)

	success := ReportAttack(s.client, s.result, s.currentTime)
	s.Assert().True(success)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(NotificationsTestSuite))
}
