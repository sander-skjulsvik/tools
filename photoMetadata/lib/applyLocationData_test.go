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
	testing2 "github.com/sander-skjulsvik/tools/libs/testing"
	timelib "github.com/sander-skjulsvik/tools/libs/time"
)

type TestVars struct {
	PhotoCollection *PhotoCollection
	LocationStore   LocationStore
	testDirectory   string
}

func (tv *TestVars) GetNoGpsPhotos() []Photo {
	var noGPSPhotos []Photo
	for _, photo := range tv.PhotoCollection.Photos {
		if filepath.Base(photo.Path) == "fuji_no_gps.RAF" {
			noGPSPhotos = append(noGPSPhotos, photo)
		}
	}
	return noGPSPhotos
}

func (tv *TestVars) Clean() {
	os.RemoveAll(tv.testDirectory)
}

var (
	cest           = timelib.GetCEST()
	minorTimeDelta = 1 * time.Second
	t0             = time.Date(2024, 05, 19, 17, 27, 48, 0, cest)
	c0             = lib.NewCoordinatesE7(0, 0)
	t1             = t0.Add(MEDIUM_TIME_DIFF_THRESHOLD - minorTimeDelta)
	c1             = lib.NewCoordinatesE7(1, 1)
)

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

	ls := LocationStore{
		LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
		MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
		SourceLocations: lib.SourceLocations{
			Locations: []lib.LocationRecord{
				{
					Coordinates: c0,
					Time:        t0,
				},
				{
					Coordinates: c1,
					Time:        t1,
				},
			},
		},
	}
	ls.SourceLocations.SortByTime()

	return TestVars{
		PhotoCollection: pc,
		LocationStore:   ls,
		testDirectory:   path,
	}
}

func TestApplyLocationData(t *testing.T) {
	dir := "TestApplyLocationDataDir"
	testVars := TestingSetup(dir)
	defer testVars.Clean()

	// Test no alter photo with gps location
	noGPSPhoto := testVars.GetNoGpsPhotos()[0]

	// Sanity check
	location, err := noGPSPhoto.GetLocationRecord()
	if location != nil {
		t.Errorf("Expected %s to not have location data, got: %s", noGPSPhoto.Path, location)
	}
	if errors.Is(err, ErrGetLocationRecordParsingGPS) {
		testing2.ErrorfStackTrace(t,
			"Expected %s to not have location data, got: %v",
			noGPSPhoto.Path, err,
		)
	}
	origTime, err := noGPSPhoto.GetDateTimeOriginal()
	if err != nil {
		panic(err)
	}
	fmt.Printf("photo time Orig: %s\n", origTime)
	// Apply midpoint time to photo
	timeMidpoint := t0.Add(t1.Sub(t0) / 2)
	fmt.Printf("Midpoint: %s\n", timeMidpoint)
	if err := noGPSPhoto.WriteDateTime(timeMidpoint); err != nil {
		panic(err)
	}
	alteredTime, err := noGPSPhoto.GetDateTimeOriginal()
	if err != nil {
		panic("here1")
	}
	if !alteredTime.Equal(timeMidpoint) {
		testing2.ErrorfStackTrace(t, "Failed to write midtime to file")
	}
	alteredTimeString := alteredTime.String()
	fmt.Printf("photo time altered: %s\n", alteredTimeString)

	// Actually testing
	if ok := applyLocationData(noGPSPhoto, testVars.LocationStore, false); !ok {
		testing2.ErrorfStackTrace(t, "failed to apply locaction data")
	}
	readLocation, err := noGPSPhoto.GetLocationRecord()
	if err != nil {
		testing2.ErrorfStackTrace(t,
			"Failed to get location data after applying location data: path: %s, err: %s",
			noGPSPhoto.Path, err,
		)
	}

	midpoint := lib.NewCoordinatesE2(0.500019, 0.500019)
	if !readLocation.Coordinates.Equal(midpoint) {
		testing2.ErrorfStackTrace(t,
			"written location record is not equal to midpoint coordinate:\n\t midpoint: %s, read: %s",
			midpoint.String(), readLocation.Coordinates.String(),
		)
	}
}
