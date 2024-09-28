package lib

import (
	"errors"
	"testing"
	"time"

	locationData "github.com/sander-skjulsvik/tools/google_location_data/lib"
)

func TestGetCoordiantesByTime(t *testing.T) {
	t0 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	majorTimeDelta := time.Second
	t1 := t0.Add(HIGH_TIME_DIFF_THRESHOLD + majorTimeDelta)
	minorTimeDelta := time.Millisecond

	ls := LocationStore{
		LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
		MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
		HighTimeDiffThreshold:   HIGH_TIME_DIFF_THRESHOLD,
		SourceLocations: locationData.SourceLocations{
			Locations: []locationData.LocationRecord{
				{
					Coordinates: locationData.NewCoordinatesE2(0, 0),
					Time:        t0,
				},
				{
					Coordinates: locationData.NewCoordinatesE2(1, 1),
					Time:        t1,
				},
			},
		},
	}

	checkResult := func(expectedCoordinates locationData.Coordinates, qTime time.Time, expectedDiff time.Duration, allowedCoordinateDelta float64, expectedErr error) {
		coord, calcTimeDiff, err := ls.GetCoordinatesByTime(qTime)
		if !errors.Is(err, expectedErr) {
			t.Errorf("err not nil: %v", err)
		}
		if calcTimeDiff != expectedDiff {
			t.Errorf("time diff not as expected:\n\texpected: %s\n\tcalculated: %s", expectedDiff, calcTimeDiff)
		}

		if !coord.EqualDelta(expectedCoordinates, allowedCoordinateDelta) {
			t.Errorf(
				"low time diff did not give coordinates equal to expected:\n\texpected: %s\n\tcalculated: %s",
				expectedCoordinates.String(), coord.String(),
			)
		}
	}

	// No diff
	checkResult(locationData.NewCoordinatesE2(0, 0), t0, 0, 1e-7, nil)
	checkResult(locationData.NewCoordinatesE2(1, 1), t1, 0, 1e-7, nil)

	// Under low threshold
	checkResult(locationData.NewCoordinatesE2(0, 0), t0.Add(minorTimeDelta), minorTimeDelta, 1e-7, nil)
	checkResult(locationData.NewCoordinatesE2(0, 0), t0.Add(LOW_TIME_DIFF_THRESHOLD-minorTimeDelta), LOW_TIME_DIFF_THRESHOLD-minorTimeDelta, 1e-7, nil)

	checkResult(locationData.NewCoordinatesE2(0, 0), t0.Add(-minorTimeDelta), minorTimeDelta, 1e-7, nil)
	checkResult(locationData.NewCoordinatesE2(0, 0), t0.Add(-(LOW_TIME_DIFF_THRESHOLD - minorTimeDelta)), LOW_TIME_DIFF_THRESHOLD-minorTimeDelta, 1e-7, nil)

	checkResult(locationData.NewCoordinatesE2(1, 1), t1.Add(minorTimeDelta), minorTimeDelta, 1e-7, nil)
	checkResult(locationData.NewCoordinatesE2(1, 1), t1.Add(LOW_TIME_DIFF_THRESHOLD-minorTimeDelta), LOW_TIME_DIFF_THRESHOLD-minorTimeDelta, 1e-7, nil)

	// Over low threshold but under high threshold
	checkResult(locationData.NewCoordinatesE2(0, 0), t0.Add(LOW_TIME_DIFF_THRESHOLD+minorTimeDelta), LOW_TIME_DIFF_THRESHOLD+minorTimeDelta, 0.05, ErrTimeDiffMedium)
	// ...
	// over high treshold

	// lrsLow := []locationData.LocationRecord{}
	// lrsMed := []locationData.LocationRecord{}
	// lrsHigh := []locationData.LocationRecord{}
	// n := 3
	// for i:= 0; i < n; i++ {
	// 	lrsLow = append(lrsLow, locationData.LocationRecord{
	// 		Coordinates: locationData.NewCoordinatesE2(float64(i), float64(i)),
	// 		Time:        t0.Add((time.Duration(i)*LOW_TIME_DIFF_THRESHOLD) - majorTimeDelta),
	// 	})
	// 	lrsMed = append(lrsMed, locationData.LocationRecord{
	// 		Coordinates: locationData.NewCoordinatesE2(float64(i), float64(i)),
	// 		Time:        t0.Add((time.Duration(i)*MEDIUM_TIME_DIFF_THRESHOLD) - majorTimeDelta),
	// 	})
	// 	lrsHigh = append(lrsHigh, locationData.LocationRecord{
	// 		Coordinates: locationData.NewCoordinatesE2(float64(i), float64(i)),
	// 		Time:        t0.Add((time.Duration(i)*HIGH_TIME_DIFF_THRESHOLD) - majorTimeDelta),
	// 	})
	// }

	// ls := LocationStore{
	// 	LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
	// 	MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
	// 	HighTimeDiffThreshold:   HIGH_TIME_DIFF_THRESHOLD,
	// 	SourceLocations: locationData.SourceLocations{},
	// }

	// for i:= 0; i < n; i++ {
	// 	// Low thresh
	// 	ls0.SourceLocations.Locations = lrsLow
	// 	checkResult()

	// 	// Med thresh
	// 	ls0.SourceLocations.Locations = lrsMed

	// 	// High tresh
	// 	ls0.SourceLocations.Locations = lrsHigh

	// }

	// lr0 := locationData.LocationRecord{
	// 	Coordinates: locationData.NewCoordinatesE2(0, 0),
	// 	Time:        t0,
	// }
	// lrLowTimeThresh := locationData.LocationRecord{
	// 	Coordinates: locationData.NewCoordinatesE2(1, 1),
	// }
	// lrLowTimeThresh.Time = t0.Add(LOW_TIME_DIFF_THRESHOLD - 1*time.Minute)

	// lrMedThresh := locationData.LocationRecord{
	// 	Coordinates: locationData.NewCoordinatesE2(1, 1),
	// }
	// lrMedThresh.Time = t0.Add(MEDIUM_TIME_DIFF_THRESHOLD - 1*time.Minute)

	// lrHighThresh := locationData.LocationRecord{
	// 	Coordinates: locationData.NewCoordinatesE2(1, 1),
	// }
	// lrHighThresh.Time = t0.Add(HIGH_TIME_DIFF_THRESHOLD - 1*time.Minute)

	// lrSuperHighThresh := locationData.LocationRecord{
	// 	Coordinates: locationData.NewCoordinatesE2(1, 1),
	// }
	// lrSuperHighThresh.Time = t0.Add(HIGH_TIME_DIFF_THRESHOLD + 1*time.Minute)

	// ls := LocationStore{
	// 	LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
	// 	MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
	// 	HighTimeDiffThreshold:   HIGH_TIME_DIFF_THRESHOLD,
	// 	SourceLocations: locationData.SourceLocations{
	// 		Locations: []locationData.LocationRecord{
	// 			lr0, lrLowTimeThresh, lrMedThresh, lrHighThresh, lrSuperHighThresh,
	// 		},
	// 	},
	// }

	// // Before 0 time
	// checkResult(lr0.Coordinates, lr0.Time.Add(-1 * time.Minute),  1 * time.Minute)

	// // Low low time
	// checkResult(lr0.Coordinates, lr0.Time.Add(1 * time.Minute),  1 * time.Minute)
	// // high low time
	// checkResult(lrLowTimeThresh.Coordinates, lrLowTimeThresh.Time.Add(-time.Minute),  -1 * time.Minute)

	// // Med low
	// checkResult(lrLowTimeThresh.Coordinates, lrLowTimeThresh.Time.Add(1 * time.Minute),  1 * time.Minute)
	// // Med high
	// checkResult(lrHighThresh.Coordinates, lrHighThresh.Time.Add(-time.Minute),  -1 * time.Minute)

	// // High low

	// // High high

	// // Super high ?

}
