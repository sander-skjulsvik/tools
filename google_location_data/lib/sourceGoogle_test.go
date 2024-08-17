package lib

import (
	"errors"
	"testing"
	"time"
)

func TestToLocationRecords(t *testing.T) {
	// Create a GoogleTimelineTakeout struct with some test data
	googleTimelineTakeout := GoogleTimelineTakeout{
		Locations: []GoogleTimelineLocations{
			{
				LatitudeE7:  633954185,
				LongitudeE7: 103719669,
				Timestamp:   "2014-04-22T12:15:05.138Z",
			},
			{
				LatitudeE7:  633954162,
				LongitudeE7: 103720388,
				Timestamp:   "2014-04-22T12:16:05.138Z",
			},
		},
	}

	// Call the ToLocationRecords method
	sourceLocations, err := googleTimelineTakeout.ToLocationRecords()
	if err != nil {
		t.Errorf("Error calling ToLocationRecords: %v", err)
	}

	// Check that the sourceLocations struct has the correct number of locations
	if len(sourceLocations.Locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(sourceLocations.Locations))
	}

	// Check that the sourceLocations struct has the correct locations
	if sourceLocations.Locations[0].Corrdinates.LatitudeE7 != 633954185 {
		t.Errorf("Expected first location latitude 633954185, got %d", sourceLocations.Locations[0].Corrdinates.LatitudeE7)
	}
	if sourceLocations.Locations[0].Corrdinates.LongitudeE7 != 103719669 {
		t.Errorf("Expected first location longitude 103719669, got %d", sourceLocations.Locations[0].Corrdinates.LongitudeE7)
	}
	if sourceLocations.Locations[1].Corrdinates.LatitudeE7 != 633954162 {
		t.Errorf("Expected second location latitude 633954162, got %d", sourceLocations.Locations[1].Corrdinates.LatitudeE7)
	}
	if sourceLocations.Locations[1].Corrdinates.LongitudeE7 != 103720388 {
		t.Errorf("Expected second location longitude 103720388, got %d", sourceLocations.Locations[1].Corrdinates.LongitudeE7)
	}

	// Check that the sourceLocations struct has the correct timestamps
	expectedTime1 := "2014-04-22 12:15:05.138 +0000 UTC"
	expectedTime2 := "2014-04-22 12:16:05.138 +0000 UTC"

	if sourceLocations.Locations[0].Time.String() != expectedTime1 {
		t.Errorf("Expected first location timestamp %s, got %s", expectedTime1, sourceLocations.Locations[0].Time)
	}
	if sourceLocations.Locations[1].Time.String() != expectedTime2 {
		t.Errorf("Expected second location timestamp %s, got %s", expectedTime2, sourceLocations.Locations[1].Time)
	}
	sourceLocations.SortByTime()
	if sourceLocations.Locations[1].Time.Before(sourceLocations.Locations[0].Time) {
		t.Errorf("Expected first location timestamp to be before second location timestamp")
	}
	if sourceLocations.Locations[1].Time.Sub(sourceLocations.Locations[0].Time) == time.Hour*1 {
		t.Errorf("Expected first location timestamp to be before second location timestamp")
	}

	// Tesing with strange coordinates
	latitudeOutOfRange := GoogleTimelineTakeout{
		Locations: []GoogleTimelineLocations{
			// Latitude out of range
			{
				LatitudeE7:  1000000000,
				LongitudeE7: 103719669,
				Timestamp:   "2014-04-22T12:15:05.138Z",
			},
		},
	}
	longitudeOutOfRange := GoogleTimelineTakeout{
		Locations: []GoogleTimelineLocations{
			// Longitude out of range
			{
				LatitudeE7:  633954185,
				LongitudeE7: -100000000,
				Timestamp:   "2014-04-22T12:15:05.138Z",
			},
		},
	}

	_, err = latitudeOutOfRange.ToLocationRecords()
	if errors.Is(err, ErrLatitudeOutOfRange) {
		t.Errorf("Expected ErrLatitudeOutOfRange error, got: %v", err)
	}
	_, err = longitudeOutOfRange.ToLocationRecords()
	if errors.Is(err, ErrLongitudeOutOfRange) {
		t.Errorf("Expected ErrLongitudeOutOfRange error, got: %v", err)
	}
}
