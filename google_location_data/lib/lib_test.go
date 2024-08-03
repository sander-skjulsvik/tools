package lib

import (
	"os"
	"path/filepath"
	"testing"
)

const TEST_DATA_FOLDER = "lib_test_data"

func TestImportSourceLocationData(t *testing.T) {
	// Create a test data file
	testDataFolder := filepath.Join(TEST_DATA_FOLDER, "TestImportSourceLocationData")
	testSourceData := filepath.Join(testDataFolder, "data.json")
	err := os.MkdirAll(testDataFolder, 0o755)
	if err != nil {
		t.Fatalf("Error creating test data folder: %v", err)
	}
	defer os.RemoveAll(TEST_DATA_FOLDER)

	data := `{
		"locations": [
			{"latitudeE7": 1234567, "longitudeE7": 2345678, "timestamp": "2021-01-01T12:00:00Z"},
			{"latitudeE7": 2345678, "longitudeE7": 3456789, "timestamp": "2021-01-02T12:00:00Z"},
			{"latitudeE7": 3456789, "longitudeE7": 4567890, "timestamp": "2021-01-03T12:00:00Z"}
		]
	}`

	err = os.WriteFile(testSourceData, []byte(data), 0o644)
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
