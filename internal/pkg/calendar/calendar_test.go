package calendar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsWeekDay(t *testing.T) {
	monday := time.Date(2018, 4, 16, 0, 0, 0, 0, time.UTC)

	assert.True(t, isWeekday(monday))
	assert.True(t, isWeekday(monday.Add(time.Hour*24)))
	assert.True(t, isWeekday(monday.Add(time.Hour*24*2)))
	assert.True(t, isWeekday(monday.Add(time.Hour*24*3)))
	assert.True(t, isWeekday(monday.Add(time.Hour*24*4)))

	assert.False(t, isWeekday(monday.Add(time.Hour*24*5)))
	assert.False(t, isWeekday(monday.Add(time.Hour*24*6)))
}

// FIXME:  add more tests
