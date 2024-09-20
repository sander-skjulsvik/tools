package time

import (
	"math"
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
		GetMidpointByRatio(t1, t2, 0.5),
		t1.Add(12*time.Hour),
	)
	assert.Equal(
		t,
		GetMidpointByRatio(t1, t2, 1),
		t2,
	)
	assert.Equal(
		t,
		GetMidpointByRatio(t1, t2, 0),
		t1,
	)
	assert.Equal(
		t,
		GetMidpointByRatio(t1, t2, 0.7),
		time.Date(2000, 1, 1, 16, 48, 0, 0, time.Local),
	)

}

func TestTimeRatio(t *testing.T) {
	// Setup
	var (
		time1 = *toolsTime.ParseTimeNoErrorRFC3339("2021-01-01T12:00:00Z")
		time2 = *toolsTime.ParseTimeNoErrorRFC3339("2021-01-02T12:00:00Z")
		time3 = *toolsTime.ParseTimeNoErrorRFC3339("2021-01-03T12:00:00Z")
	)

	// In the middle
	ratio_middle_1_2 := GetTimeRatio(time1, time2, time1.Add(time2.Sub(time1)/2))
	ratio_middle_1_3 := GetTimeRatio(time1, time3, time1.Add(time3.Sub(time1)/2))

	// Check the values of the records
	if ratio_middle_1_2 != 0.5 {
		t.Errorf("Expected ratio 0.5, got %f", ratio_middle_1_2)
	}
	if ratio_middle_1_3 != 0.5 {
		t.Errorf("Expected ratio 0.5, got %f", ratio_middle_1_3)
	}
	// 3rd
	ratio_3rd_1_2 := GetTimeRatio(time1, time2, time1.Add(time2.Sub(time1)/3))
	ratio_3rd_1_3 := GetTimeRatio(time1, time3, time1.Add(time3.Sub(time1)/3))

	// Check the values of the records
	if math.Abs(ratio_3rd_1_2-0.3) < 1e-14 {
		t.Errorf("Expected ratio 0.3, got %f", ratio_3rd_1_2)
	}
	if math.Abs(ratio_3rd_1_3-0.3) < 1e-14 {
		t.Errorf("Expected ratio 0.3, got %f", ratio_3rd_1_3)
	}

	// At the start
	ratio_start_1_2 := GetTimeRatio(time1, time2, time1)
	ratio_start_1_3 := GetTimeRatio(time1, time3, time1)

	// Check the values of the records
	if ratio_start_1_2 != 0 {
		t.Errorf("Expected ratio 0, got %f", ratio_start_1_2)
	}
	if ratio_start_1_3 != 0 {
		t.Errorf("Expected ratio 0, got %f", ratio_start_1_3)
	}

	// At the end
	ratio_end_1_2 := GetTimeRatio(time1, time2, time2)
	ratio_end_1_3 := GetTimeRatio(time1, time3, time3)

	// Check the values of the records
	if ratio_end_1_2 != 1 {
		t.Errorf("Expected ratio 1, got %f", ratio_end_1_2)
	}
	if ratio_end_1_3 != 1 {
		t.Errorf("Expected ratio 1, got %f", ratio_end_1_3)
	}
}
