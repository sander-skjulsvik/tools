package lib

import (
	"errors"
	"runtime/debug"
	"testing"
	"time"

	locationData "github.com/sander-skjulsvik/tools/google_location_data/lib"
)

func TestGetCoordinatesByTime(t *testing.T) {
	var (
		minorTimeDelta = time.Millisecond
		majorTimeDelta = time.Second
		t0             = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		c0             = locationData.NewCoordinatesE2(0, 0)
		t1             = t0.Add(5*MEDIUM_TIME_DIFF_THRESHOLD + majorTimeDelta)
		c1             = locationData.NewCoordinatesE2(1, 1)
		t2             = t0.Add(MEDIUM_TIME_DIFF_THRESHOLD * 10)
		c2             = locationData.NewCoordinatesE2(2, 2)
	)

	ls := LocationStore{
		LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
		MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
		SourceLocations: locationData.SourceLocations{
			Locations: []locationData.LocationRecord{
				{
					Coordinates: c0,
					Time:        t0,
				},
				{
					Coordinates: c1,
					Time:        t1,
				},
				{
					Coordinates: c2,
					Time:        t2,
				},
			},
		},
	}

	checkResult := func(expectedCoordinates locationData.Coordinates, qTime time.Time, expectedDiff time.Duration, allowedCoordinateDelta float64, expectedErr error) {
		coord, calcTimeDiff, err := ls.GetCoordinatesByTime(qTime)
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error '%v' got '%v', for qTime: %s\n\n%s", expectedErr, err, qTime, debug.Stack())
			return
		}
		if calcTimeDiff != expectedDiff {
			t.Errorf("time diff not as expected:\n\texpected: %s\n\tcalculated: %s\n\n%s", expectedDiff, calcTimeDiff, debug.Stack())
		}

		if !coord.EqualDelta(expectedCoordinates, allowedCoordinateDelta) {
			t.Errorf(
				"low time diff did not give coordinates equal to expected:\n\texpected: %s\n\tcalculated: %s\n\n%s",
				expectedCoordinates.String(), coord.String(), debug.Stack(),
			)
		}
	}

	// No diff
	checkResult(c0, t0, 0, 1e-7, nil)
	checkResult(c1, t1, 0, 1e-7, nil)

	// Under low threshold
	checkResult(c0, t0.Add(minorTimeDelta), minorTimeDelta, 1e-7, nil)
	checkResult(c0, t0.Add(LOW_TIME_DIFF_THRESHOLD-minorTimeDelta), LOW_TIME_DIFF_THRESHOLD-minorTimeDelta, 1e-7, nil)

	checkResult(c0, t0.Add(-minorTimeDelta), minorTimeDelta, 1e-7, nil)
	checkResult(c0, t0.Add(-(LOW_TIME_DIFF_THRESHOLD - minorTimeDelta)), LOW_TIME_DIFF_THRESHOLD-minorTimeDelta, 1e-7, nil)

	checkResult(c1, t1.Add(minorTimeDelta), minorTimeDelta, 1e-7, nil)
	checkResult(c1, t1.Add(LOW_TIME_DIFF_THRESHOLD-minorTimeDelta), LOW_TIME_DIFF_THRESHOLD-minorTimeDelta, 1e-7, nil)

	// In the medium range
	lowerDelta := LOW_TIME_DIFF_THRESHOLD + minorTimeDelta
	upperDelta := MEDIUM_TIME_DIFF_THRESHOLD - minorTimeDelta
	checkResult(c0, t0.Add(lowerDelta), lowerDelta, 0.051, ErrTimeDiffMedium)
	checkResult(c1, t1.Add(-lowerDelta), lowerDelta, 0.2, ErrTimeDiffMedium)

	checkResult(c0, t0.Add(upperDelta), upperDelta, 0.21, ErrTimeDiffMedium)

	// Over low threshold and outside of data
	checkResult(locationData.Coordinates{}, t0.Add(-(LOW_TIME_DIFF_THRESHOLD + majorTimeDelta)), 0, 1e-7, ErrQueryTimeIsOutsideOfRange)
	checkResult(locationData.Coordinates{}, t2.Add(LOW_TIME_DIFF_THRESHOLD+majorTimeDelta), 0, 1e-7, ErrQueryTimeIsOutsideOfRange)

}
