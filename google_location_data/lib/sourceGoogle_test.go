package lib

import "testing"

func TestToLocationRecords(t *testing.T) {
	// Create a GoogleTimelineTakeout struct with some test data
	googleTimelineTakeout := GoogleTimelineTakeout{
		Locations: []GoogleTimelineLocations{
			{
				LatitudeE7:  1234567,
				LongitudeE7: 1234567,
				Timestamp:   "123456789",
			},
			{
				LatitudeE7:  1234567,
				LongitudeE7: 1234567,
				Timestamp:   "123456789",
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
}
