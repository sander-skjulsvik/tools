package time

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestParseTimeNoErrorRFC3339(t *testing.T) {
	s := "2006-01-02T15:04:05Z07:00"
	ParseTimeNoErrorRFC3339(s)
}

func TestGetMidpointByWeights(t *testing.T) {

	t1 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	t2 := time.Date(2000, 1, 2, 0, 0, 0, 0, time.Local)
	assert.Equal(
		t,
		GetMidpointByWeights(t1, t2, 0.5),
		t1.Add(12*time.Hour),
	)
	assert.Equal(
		t,
		GetMidpointByWeights(t1, t2, 1),
		t2,
	)
	assert.Equal(
		t,
		GetMidpointByWeights(t1, t2, 0),
		t1,
	)
	assert.Equal(
		t,
		GetMidpointByWeights(t1, t2, 0.7),
		time.Date(2000, 1, 1, 16, 48, 0, 0, time.Local),
	)

}
