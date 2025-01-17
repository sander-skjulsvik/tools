package locationData

import (
	"testing"
	"time"

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
func TestInterpolation(t *testing.T) {
	// Setup
	var (
		locRecord1 = LocationRecord{
			NewCoordinatesE2(0, 0),
			time.Date(2021, 01, 01, 12, 0, 0, 0, time.UTC),
		}
		locRecord2 = LocationRecord {
			NewCoordinatesE2(1, 1),
			time.Date(2021, 01, 02, 12, 0, 0, 0, time.UTC),

		}
	)
	// In the middle
	{
		tt := toolsTime.GetMidpointByRatio(locRecord1.Time, locRecord2.Time, 0.5)
		calc := Interpolation(locRecord1, locRecord2, tt)
		expected := NewCoordinatesE2(0.500019, 0.499962)
		if !calc.EqualDeltaE7(expected) {
			t.Errorf("calc middle not equal enough to calculated: \n\tcalculated: %s,\n\texpected: %s", calc.String(), expected.String())
		}
	}
	// 0.7: expected coord, but 0.3 time, falsification test
	{
		tt := toolsTime.GetMidpointByRatio(locRecord1.Time, locRecord2.Time, 0.3)
		calc := Interpolation(locRecord1, locRecord2, tt)
		expected := NewCoordinatesE2(0.588259, 0.588201)
		if calc.EqualDeltaE7(expected) {
			t.Errorf("calc middle equal to calculated: \n\tcalculated: %s,\n\texpected: %s", calc.String(), expected.String())
		}
	}
	// 0.7: expected coord and 0.7 time
	{
		tt := toolsTime.GetMidpointByRatio(locRecord1.Time, locRecord2.Time, 0.7)
		calc := Interpolation(locRecord1, locRecord2, tt)
		expected := NewCoordinatesE2(0.700018, 0.699964)
		if !calc.EqualDelta(expected, 1E-6) {
			t.Errorf("calc middle not equal enough to calculated: \n\tcalculated: %s,\n\texpected: %s", calc.String(), expected.String())
		}
	}
	// 0.6: should be close enough to 0.6, 0.6 coords
	{
		tt := toolsTime.GetMidpointByRatio(locRecord1.Time, locRecord2.Time, 0.6)
		calc := Interpolation(locRecord1, locRecord2, tt)
		expected := NewCoordinatesE2(0.6, 0.6)
		if !calc.EqualDelta(expected, 1e-2) {
			t.Errorf("calc middle not equal enough to calculated: \n\tcalculated: %s,\n\texpected: %s", calc.String(), expected.String())
		}
	}

}
