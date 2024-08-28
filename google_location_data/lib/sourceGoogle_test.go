package lib

import (
	"testing"
	"time"
)

func TestNewGoogleTimelineLocationsFromFile(t *testing.T) {
	// Open test file
	path := "test_data/test_google_location_data.json"
	sourceLocations, err := NewSourceLocationsFromGoogleTimeline(path)
	if err != nil {
		t.Errorf("Expected no error opening data, got %v", err)
	}

	// Check that the googleTimelineLocations struct has the correct number of locations
	if len(sourceLocations.Locations) != 3 {
		t.Errorf("Expected 3 locations, got %d, takeout: %v", len(sourceLocations.Locations), sourceLocations)
	}

	// Check that the googleTimelineLocations struct has the correct locations
	if sourceLocations.Locations[0].Corrdinates.LatE7() != 1 {
		t.Errorf("Expected Lat 1, got %d from location 0", sourceLocations.Locations[0].Corrdinates.LatE7())
	}
	if sourceLocations.Locations[0].Corrdinates.LngE7() != 2 {
		t.Errorf("Expected Long 2, got %d from location 0", sourceLocations.Locations[0].Corrdinates.LngE7())
	}
	if sourceLocations.Locations[1].Corrdinates.LatE7() != 3 {
		t.Errorf("Expected Lat 3, got %d from location 1", sourceLocations.Locations[1].Corrdinates.LatE7())
	}
	if sourceLocations.Locations[1].Corrdinates.LngE7() != 4 {
		t.Errorf("Expected Long 4, got %d from location 1", sourceLocations.Locations[1].Corrdinates.LngE7())
	}
	if sourceLocations.Locations[2].Corrdinates.LatE7() != 5 {
		t.Errorf("Expected Lat 5, got %d from location 2", sourceLocations.Locations[2].Corrdinates.LatE7())
	}
	if sourceLocations.Locations[2].Corrdinates.LngE7() != 6 {
		t.Errorf("Expected Long 6, got %d from location 2", sourceLocations.Locations[2].Corrdinates.LngE7())
	}
}

func TestToLocationRecords(t *testing.T) {
	// Create a GoogleTimelineTakeout struct with some test data
	googleTimelineTakeout := GoogleTimelineTakeout{
		Locations: []GoogleTimelineLocation{
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
	sourceLocations := googleTimelineTakeout.ToLocationRecords()

	// Check that the sourceLocations struct has the correct number of locations
	if len(sourceLocations.Locations) != 2 {
		t.Errorf("Expected 2 locations, got %d", len(sourceLocations.Locations))
	}

	// Check that the sourceLocations struct has the correct locations
	if sourceLocations.Locations[0].Corrdinates.LatE7() != 633954185 {
		t.Errorf("Expected first location latitude 633954185, got %d", sourceLocations.Locations[0].Corrdinates.LatE7())
	}
	if sourceLocations.Locations[0].Corrdinates.LngE7() != 103719669 {
		t.Errorf("Expected first location longitude 103719669, got %d", sourceLocations.Locations[0].Corrdinates.LngE7())
	}
	if sourceLocations.Locations[1].Corrdinates.LatE7() != 633954162 {
		t.Errorf("Expected second location latitude 633954162, got %d", sourceLocations.Locations[1].Corrdinates.LatE7())
	}
	if sourceLocations.Locations[1].Corrdinates.LngE7() != 103720388 {
		t.Errorf("Expected second location longitude 103720388, got %d", sourceLocations.Locations[1].Corrdinates.LngE7())
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
}
