package lib

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/otiai10/copy"
	"github.com/sander-skjulsvik/tools/google_location_data/lib"
	"github.com/sander-skjulsvik/tools/libs/files"
)

type TestVars struct {
	PhotoCollection *PhotoCollection
	LocationStore   LocationStore
}

func TestingSetup(path string) TestVars {
	sourceData := filepath.Clean("./testData/")
	testDir := filepath.Clean(path)

	filepaths, err := files.GetAllFilesOfTypes(sourceData, []string{".raf"})
	if err != nil {
		panic(fmt.Errorf("NewTestVars failed to get all file of relevant types: %v", err))
	}
	os.MkdirAll(testDir, 0o755)
	for _, fp := range filepaths {
		copy.Copy(fp, filepath.Join(testDir, filepath.Base(fp)))
	}

	pc, err := NewPhotoCollectionFromPath(testDir)
	if err != nil {
		panic(fmt.Errorf("NewTestVars failed to create photoCollection: %v", err))
	}

	cest, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(fmt.Errorf("failed to get cest time location: %v", err))
	}

	ls := LocationStore{
		LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
		MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
		HighTimeDiffThreshold:   HIGH_TIME_DIFF_THRESHOLD,
		SourceLocations: lib.SourceLocations{
			Locations: []lib.LocationRecord{
				{
					Coordinates: lib.NewCoordinatesE7(0, 0),
					Time:        time.Date(2024, 05, 19, 17, 27, 48, 0, cest).Add(1 * time.Hour),
				},
				{
					Coordinates: lib.NewCoordinatesE7(1, 1),
					Time:        time.Date(2024, 05, 19, 17, 27, 48, 0, cest).Add(-1 * time.Hour),
				},
			},
		},
	}
	ls.SourceLocations.SortByTime()

	return TestVars{
		PhotoCollection: pc,
		LocationStore:   ls,
	}
}

func TestApplyLocationData(t *testing.T) {
	dir := "TestApplyLocationData"
	defer os.RemoveAll(dir)
	testVars := TestingSetup(dir)

	// Test no alter photo with gps location
	var noGPSPhoto Photo
	for _, photo := range testVars.PhotoCollection.Photos {
		if filepath.Base(photo.Path) == "fuji_no_gps.RAF" {
			noGPSPhoto = photo
		}
	}
	// Sanity check
	location, err := noGPSPhoto.GetLocationRecord()
	if location != nil {
		t.Errorf("Expected %s to not have location data, got: %s", noGPSPhoto.Path, location)
	}
	if errors.Is(err, ErrGetLocationRecordParsingGPS) {
		t.Errorf("Expected %s to not have location data, got: %v", noGPSPhoto.Path, err)
	}
	// Actually testing
	applyLocationData(noGPSPhoto, testVars.LocationStore, false)
	readLocation, err := noGPSPhoto.GetLocationRecord()
	if err != nil {
		t.Errorf(
			"Failed to get location data after applying location data: path: %s, err: %s",
			noGPSPhoto.Path, err,
		)
	}
	midpoint := lib.NewCoordinatesE2(0.500019, 0.500019)

	if !readLocation.Equal(&testVars.LocationStore.SourceLocations.Locations[0]) {
		t.Errorf("written location record is not equal to read location record:\n\t written: %s, read: %s", readLocation, &testVars.LocationStore.SourceLocations.Locations[0])
	}

}
