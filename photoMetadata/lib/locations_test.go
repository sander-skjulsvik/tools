package lib

import (
	"testing"
	"time"

	locationData "github.com/sander-skjulsvik/tools/google_location_data/lib"
)

func TestGetCoordiantesByTime(t *testing.T) {
	localEpoch := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	lr0 := locationData.LocationRecord{
		Coordinates: locationData.NewCoordinatesE2(0, 0),
		Time:        localEpoch,
	}
	lrLowTimeThresh := locationData.LocationRecord{
		Coordinates: locationData.NewCoordinatesE2(1, 1),
	}
	lrLowTimeThresh.Time = localEpoch.Add(LOW_TIME_DIFF_THRESHOLD - 1*time.Minute)

	lrMedThresh := locationData.LocationRecord{
		Coordinates: locationData.NewCoordinatesE2(1, 1),
	}
	lrMedThresh.Time = localEpoch.Add(MEDIUM_TIME_DIFF_THRESHOLD - 1*time.Minute)

	lrHighThresh := locationData.LocationRecord{
		Coordinates: locationData.NewCoordinatesE2(1, 1),
	}
	lrHighThresh.Time = localEpoch.Add(HIGH_TIME_DIFF_THRESHOLD - 1*time.Minute)

	lrSuperHighThresh := locationData.LocationRecord{
		Coordinates: locationData.NewCoordinatesE2(1, 1),
	}
	lrSuperHighThresh.Time = localEpoch.Add(HIGH_TIME_DIFF_THRESHOLD + 1*time.Minute)

	ls := LocationStore{
		LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
		MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
		HighTimeDiffThreshold:   HIGH_TIME_DIFF_THRESHOLD,
		SourceLocations: locationData.SourceLocations{
			Locations: []locationData.LocationRecord{
				lr0, lrLowTimeThresh, lrMedThresh, lrHighThresh, lrSuperHighThresh,
			},
		},
	}

	checkResult := func(expectedCoordinates locationData.Coordinates, qTime time.Time, expectedDiff time.Duration) {
		coord, calcTimeDiff, err := ls.GetCoordinatesByTime(qTime)
		if err != nil {
			t.Errorf("err not nil: %v", err)
		}
		if calcTimeDiff != expectedDiff {
			t.Errorf("time diff not as expected:\n\texpected: %s\n\tcalculated: %s", expectedDiff, calcTimeDiff)
		}

		if !coord.Equal(expectedCoordinates) {
			t.Errorf(
				"low time diff did not give coordinates equal to expected:\n\texpected: %s\n\tcalculated: %s",
				expectedCoordinates.String(), coord.String(),
			)
		}

	}

	// Before 0 time

	// Low low time
	checkResult(lr0.Coordinates, lr0.Time.Add(1 * time.Minute),  1 * time.Minute)

	// high low time
	checkResult(lrLowTimeThresh.Coordinates, lrLowTimeThresh.Time.Add(-time.Minute),  -1 * time.Minute)

	// Med low

	// Med high

	// High low

	// High high

	// Super high ?

}
