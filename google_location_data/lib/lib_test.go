package lib

import (
	"math"
	"testing"

	toolsTime "github.com/sander-skjulsvik/tools/libs/time"
)

const (
	TEST_DATA_FOLDER        = "lib_test_data"
	SIMPLE_TEST_DATA_STRING = `{
		"locations": [
			{"latitudeE7": 1234567, "longitudeE7": 2345678, "timestamp": "2021-01-01T12:00:00Z"},
			{"latitudeE7": 2345678, "longitudeE7": 3456789, "timestamp": "2021-01-02T12:00:00Z"},
			{"latitudeE7": 3456789, "longitudeE7": 4567890, "timestamp": "2021-01-03T12:00:00Z"}
		]
	}`
)

var LOCATIONS = []LocationRecord{
	{NewCoordinatesE7(1234567, 2345678), *toolsTime.ParseTimeNoErrorRFC3339("2021-01-01T12:00:00Z")},
	{NewCoordinatesE7(2345678, 3456789), *toolsTime.ParseTimeNoErrorRFC3339("2021-01-02T12:00:00Z")},
	{NewCoordinatesE7(3456789, 4567890), *toolsTime.ParseTimeNoErrorRFC3339("2021-01-03T12:00:00Z")},
}

var (
	LOCATIONS2                       = []LocationRecord{}
	SIMPLE_TEST_DATA_SOURCE_LOCATION = SourceLocations{Locations: LOCATIONS}
)

func TestSortByTime(t *testing.T) {
	sourceData := SIMPLE_TEST_DATA_SOURCE_LOCATION
	// Swap
	sourceData.Locations[0], sourceData.Locations[1] = sourceData.Locations[1], sourceData.Locations[0]
	sourceData.SortByTime()
	locations := sourceData.Locations

	// Check the order of the records
	if locations[0].Time != *toolsTime.ParseTimeNoErrorRFC3339("2021-01-01T12:00:00Z") {
		t.Errorf("Expected first record timestamp 2021-01-01T12:00:00Z, got %s", locations[0].Time)
	}
	if locations[1].Time != *toolsTime.ParseTimeNoErrorRFC3339("2021-01-02T12:00:00Z") {
		t.Errorf("Expected second record timestamp 2021-01-02T12:00:00Z, got %s", locations[1].Time)
	}
	if locations[2].Time != *toolsTime.ParseTimeNoErrorRFC3339("2021-01-03T12:00:00Z") {
		t.Errorf("Expected third record timestamp 2021-01-03T12:00:00Z, got %s", locations[2].Time)
	}
}

func TestGetLocation(t *testing.T) {
	// Setup
	sourceData := SIMPLE_TEST_DATA_SOURCE_LOCATION
	sourceData.Locations = append(sourceData.Locations, LocationRecord{
		NewCoordinatesE7(4567890, 5678901), *toolsTime.ParseTimeNoErrorRFC3339("2021-01-04T12:00:00Z"),
	})
	sourceData.SortByTime()

	// Test the GetLocation function with exact match
	{
		timeStamp := *toolsTime.ParseTimeNoErrorRFC3339("2021-01-02T12:00:00Z")
		locationInd, err := sourceData.FindClosestLocation(timeStamp)
		if err != nil {
			t.Errorf("Error getting location: %v", err)
		}
		location := sourceData.Locations[locationInd]

		// Check the values of the records
		if location.Time != timeStamp {
			t.Errorf("Expected location before timestamp %s, got %s", timeStamp, location.Time)
		}
	}
	// Test the GetLocation function with in-between time
	{
		timeStamp := "2021-01-02T18:00:00Z"
		locationInd, err := sourceData.FindClosestLocation(*toolsTime.ParseTimeNoErrorRFC3339(timeStamp))
		if err != nil {
			t.Errorf("Error getting location: %v", err)
		}
		locationBefore := sourceData.Locations[locationInd]
		if locationBefore.Time != *toolsTime.ParseTimeNoErrorRFC3339("2021-01-02T12:00:00Z") {
			t.Errorf("Expected location before timestamp 2021-01-02T12:00:00Z, got %s", locationBefore.Time)
		}
	}

	// Testing limits
	{
		timestampFarAfter := toolsTime.ParseTimeNoErrorRFC3339("2022-01-01T12:00:00Z")
		beforeInd, err := sourceData.FindClosestLocation(*timestampFarAfter)
		before := sourceData.Locations[beforeInd]
		if err != nil {
			t.Errorf("Error getting location: %v", err)
		}
		if before.Time != *toolsTime.ParseTimeNoErrorRFC3339("2021-01-04T12:00:00Z") {
			t.Errorf("Expected location before timestamp 2021-01-04T12:00:00Z, got %s", before.Time)
		}

		timestampFarBefore := *toolsTime.ParseTimeNoErrorRFC3339("2020-01-01T12:00:00Z")
		afterInd, err := sourceData.FindClosestLocation(timestampFarBefore)
		after := sourceData.Locations[afterInd]
		if err != nil {
			t.Errorf("Error getting location: %v", err)
		}
		if after.Time != sourceData.Locations[0].Time {
			t.Errorf("Expected location after timestamp %s, got %s", sourceData.Locations[0].Time, after.Time)
		}
	}
}

func TestInterpolation(t *testing.T) {
	// Setup
	var (
		diff       = 1
		locRecord1 = LocationRecord{
			NewCoordinatesE7(1234567, 2345678),
			*toolsTime.ParseTimeNoErrorRFC3339("2021-01-01T12:00:00Z"),
		}
		locRecord2 = LocationRecord{
			NewCoordinatesE7(
				locRecord1.Coordinates.LatE7()+diff,
				locRecord1.Coordinates.LngE7()+diff,
			),
			*toolsTime.ParseTimeNoErrorRFC3339("2021-01-02T12:00:00Z"),
		}
	)

	// In the middle
	calc_middle_1_2 := Interpolation(
		locRecord1, locRecord2, locRecord1.Time.Add(locRecord2.Time.Sub(locRecord1.Time)/2))

	// Check the values of the records
	expectedLatitude := locRecord1.Coordinates.LatE7() + diff/2
	if calc_middle_1_2.LatE7() != expectedLatitude {
		t.Errorf("Expected latitude %d, got %d", expectedLatitude, calc_middle_1_2.LatE7())
	}
	expectedLongitude := locRecord1.Coordinates.LngE7() + diff/2
	if calc_middle_1_2.LngE7() != expectedLongitude {
		t.Errorf("Expected longitude %d, got %d", expectedLongitude, calc_middle_1_2.LngE7())
	}
}

func TestTimeRatio(t *testing.T) {
	// Setup
	var (
		time1 = *toolsTime.ParseTimeNoErrorRFC3339("2021-01-01T12:00:00Z")
		time2 = *toolsTime.ParseTimeNoErrorRFC3339("2021-01-02T12:00:00Z")
		time3 = *toolsTime.ParseTimeNoErrorRFC3339("2021-01-03T12:00:00Z")
	)

	// In the middle
	ratio_middle_1_2 := timeRatio(time1, time2, time1.Add(time2.Sub(time1)/2))
	ratio_middle_1_3 := timeRatio(time1, time3, time1.Add(time3.Sub(time1)/2))

	// Check the values of the records
	if ratio_middle_1_2 != 0.5 {
		t.Errorf("Expected ratio 0.5, got %f", ratio_middle_1_2)
	}
	if ratio_middle_1_3 != 0.5 {
		t.Errorf("Expected ratio 0.5, got %f", ratio_middle_1_3)
	}
	// 3rd
	ratio_3rd_1_2 := timeRatio(time1, time2, time1.Add(time2.Sub(time1)/3))
	ratio_3rd_1_3 := timeRatio(time1, time3, time1.Add(time3.Sub(time1)/3))

	// Check the values of the records
	if math.Abs(ratio_3rd_1_2-0.3) < 1e-14 {
		t.Errorf("Expected ratio 0.3, got %f", ratio_3rd_1_2)
	}
	if math.Abs(ratio_3rd_1_3-0.3) < 1e-14 {
		t.Errorf("Expected ratio 0.3, got %f", ratio_3rd_1_3)
	}

	// At the start
	ratio_start_1_2 := timeRatio(time1, time2, time1)
	ratio_start_1_3 := timeRatio(time1, time3, time1)

	// Check the values of the records
	if ratio_start_1_2 != 0 {
		t.Errorf("Expected ratio 0, got %f", ratio_start_1_2)
	}
	if ratio_start_1_3 != 0 {
		t.Errorf("Expected ratio 0, got %f", ratio_start_1_3)
	}

	// At the end
	ratio_end_1_2 := timeRatio(time1, time2, time2)
	ratio_end_1_3 := timeRatio(time1, time3, time3)

	// Check the values of the records
	if ratio_end_1_2 != 1 {
		t.Errorf("Expected ratio 1, got %f", ratio_end_1_2)
	}
	if ratio_end_1_3 != 1 {
		t.Errorf("Expected ratio 1, got %f", ratio_end_1_3)
	}
}
