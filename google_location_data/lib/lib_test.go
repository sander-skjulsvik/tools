package lib

import (
	"os"
	"path/filepath"
	"testing"
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

var SIMPLE_TEST_DATA_SOURCE_LOCATION = SourceLocationData{
	Locations: []SourceLocationRecord{
		{LatitudeE7: 1234567, LongitudeE7: 2345678, TimeStamp: "2021-01-01T12:00:00Z"},
		{LatitudeE7: 2345678, LongitudeE7: 3456789, TimeStamp: "2021-01-02T12:00:00Z"},
		{LatitudeE7: 3456789, LongitudeE7: 4567890, TimeStamp: "2021-01-03T12:00:00Z"},
	},
}

func TestImportSourceLocationData(t *testing.T) {
	// Create a test data file
	testDataFolder := filepath.Join(TEST_DATA_FOLDER, "TestImportSourceLocationData")
	testSourceData := filepath.Join(testDataFolder, "data.json")
	err := os.MkdirAll(testDataFolder, 0o755)
	if err != nil {
		t.Fatalf("Error creating test data folder: %v", err)
	}
	defer os.RemoveAll(TEST_DATA_FOLDER)

	err = os.WriteFile(testSourceData, []byte(SIMPLE_TEST_DATA_STRING), 0o644)
	if err != nil {
		t.Fatalf("Error creating test data file: %v", err)
	}

	// Test the ImportSourceLocationData function
	sourceData := ImportSourceLocationData(testSourceData)
	locations := sourceData.Locations
	if len(locations) != 3 {
		t.Errorf("Expected 3 records, got %d", len(locations))
	}

	// Check the values of the records
	if locations[0].LatitudeE7 != 1234567 {
		t.Errorf("Expected latitude 1234567, got %f", locations[0].LatitudeE7)
	}
	if locations[1].LongitudeE7 != 3456789 {
		t.Errorf("Expected longitude 3456789, got %f", locations[0].LongitudeE7)
	}
	if locations[2].TimeStamp != "2021-01-03T12:00:00Z" {
		t.Errorf("Expected timestamp 2021-01-03T12:00:00Z, got %s", locations[0].TimeStamp)
	}
}

func TestSortByTime(t *testing.T) {
	sourceData := SIMPLE_TEST_DATA_SOURCE_LOCATION
	// Swap
	sourceData.Locations[0], sourceData.Locations[1] = sourceData.Locations[1], sourceData.Locations[0]
	sourceData.SortByTime()
	locations := sourceData.Locations

	// Check the order of the records
	if locations[0].TimeStamp != "2021-01-01T12:00:00Z" {
		t.Errorf("Expected first record timestamp 2021-01-01T12:00:00Z, got %s", locations[0].TimeStamp)
	}
	if locations[1].TimeStamp != "2021-01-02T12:00:00Z" {
		t.Errorf("Expected second record timestamp 2021-01-02T12:00:00Z, got %s", locations[1].TimeStamp)
	}
	if locations[2].TimeStamp != "2021-01-03T12:00:00Z" {
		t.Errorf("Expected third record timestamp 2021-01-03T12:00:00Z, got %s", locations[2].TimeStamp)
	}
}

func TestByTimeComp(t *testing.T) {
	var byTime ByTime
	byTime = SIMPLE_TEST_DATA_SOURCE_LOCATION.Locations

	if byTime.Comp(0, 1) != -1 {
		t.Errorf("Expected -1, got %d", byTime.Comp(0, 1))
	}
	if byTime.Comp(1, 0) != 1 {
		t.Errorf("Expected 1, got %d", byTime.Comp(1, 0))
	}
	if byTime.Comp(0, 0) != 0 {
		t.Errorf("Expected 0, got %d", byTime.Comp(0, 0))
	}
}

func TestGetLocation(t *testing.T) {
	// Setup
	sourceData := SIMPLE_TEST_DATA_SOURCE_LOCATION
	sourceData.Locations = append(sourceData.Locations, SourceLocationRecord{
		LatitudeE7: 4567890, LongitudeE7: 5678901, TimeStamp: "2021-01-04T12:00:00Z",
	})
	sourceData.SortByTime()

	// Test the GetLocation function with exact match
	{
		timeStamp := "2021-01-02T12:00:00Z"
		locationInd, err := sourceData.FindClosestLocation(ParseTime(timeStamp))
		if err != nil {
			t.Errorf("Error getting location: %v", err)
		}
		location := sourceData.Locations[locationInd]

		// Check the values of the records
		if location.TimeStamp != timeStamp {
			t.Errorf("Expected location before timestamp %s, got %s", timeStamp, location.TimeStamp)
		}
	}
	// Test the GetLocation function with in-between time
	{
		timeStamp := "2021-01-02T18:00:00Z"
		locationInd, err := sourceData.FindClosestLocation(ParseTime(timeStamp))
		if err != nil {
			t.Errorf("Error getting location: %v", err)
		}
		locationBefore := sourceData.Locations[locationInd]
		if locationBefore.TimeStamp != "2021-01-02T12:00:00Z" {
			t.Errorf("Expected location before timestamp 2021-01-02T12:00:00Z, got %s", locationBefore.TimeStamp)
		}
	}

	// Testing limits
	{
		timestampFarAfter := "2022-01-01T12:00:00Z"
		beforeInd, err := sourceData.FindClosestLocation(ParseTime(timestampFarAfter))
		before := sourceData.Locations[beforeInd]
		if err != nil {
			t.Errorf("Error getting location: %v", err)
		}
		if before.TimeStamp != "2021-01-04T12:00:00Z" {
			t.Errorf("Expected location before timestamp 2021-01-04T12:00:00Z, got %s", before.TimeStamp)
		}

		timestampFarBefore := "2020-01-01T12:00:00Z"
		afterInd, err := sourceData.FindClosestLocation(ParseTime(timestampFarBefore))
		after := sourceData.Locations[afterInd]
		if err != nil {
			t.Errorf("Error getting location: %v", err)
		}
		if after.TimeStamp != sourceData.Locations[0].TimeStamp {
			t.Errorf("Expected location after timestamp %s, got %s", sourceData.Locations[0].TimeStamp, after.TimeStamp)
		}
	}
}
