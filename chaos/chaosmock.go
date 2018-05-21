package chaos

import (
	"time"

	"github.com/asobti/kube-monkey/victims"
	"github.com/stretchr/testify/mock"
)

type ChaosMock struct {
	mock.Mock
	Chaos
}

func NewMock() *ChaosMock {
	c := New(time.Now(), victims.NewVictimMock())
	return &ChaosMock{
		Chaos: *c,
	}
}

func (c *ChaosMock) Schedule(resultChan chan<- *ChaosResult) {
	_ = c.Called(resultChan)
	resultChan <- c.NewResult(nil)
}
