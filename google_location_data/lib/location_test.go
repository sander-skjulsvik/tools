package lib

import (
	"testing"
	"time"
)

func TestFindClosestLocations(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	majorTimeDelta := time.Hour
	minorTimeDelta := time.Second
	n := 10
	lrs := []LocationRecord{}
	for i := 0; i < n; i++ {
		lrs = append(lrs, LocationRecord{
			Coordinates: NewCoordinatesE2(float64(i),float64(i)),
			Time: t0.Add(majorTimeDelta*time.Duration(i)),
		})
	}
	sourceData := SourceLocations{
		Locations: lrs,
	}

	check := func(qTime time.Time, expectedClosest, expectedOther int) {
		closest, other := sourceData.FindClosestLocations(qTime)
		if closest != expectedClosest {
			t.Errorf("findClosestLocation, did not return expected closest: qTime: %s, expected: %d, got: %d", qTime, expectedClosest, closest)
		}
		if other != expectedOther {
			t.Errorf("findOtherLocation, did not return expected other: qTime: %s, expected: %d, got: %d", qTime, expectedOther, other)
		}
	}

	check(t0, 0, 0)
	check(lrs[n-1].Time, n-1, n-1)

	// checking +- 1 second around the
	for ind, lr := range sourceData.Locations {
		if ind != 0 {
			check(lr.Time.Add(-1*minorTimeDelta), ind, ind-1)
		}
		if ind != len(sourceData.Locations)-1 {
			check(lr.Time.Add(1*minorTimeDelta), ind, ind+1)
		}
	}

}
