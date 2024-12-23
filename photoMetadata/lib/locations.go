package lib

import (
	"errors"
	"fmt"
	"time"

	locationData "github.com/sander-skjulsvik/tools/google_location_data/locationData"
)

const (
	LOW_TIME_DIFF_THRESHOLD    = 30 * time.Minute
	MEDIUM_TIME_DIFF_THRESHOLD = 2 * time.Hour
)

var (
	ErrTimeDiffTooHigh           = errors.New("time difference too high")
	ErrTimeDiffMedium            = errors.New("time difference medium")
	ErrNoLocation                = errors.New("no location found")
	ErrQueryTimeIsOutsideOfRange = errors.New("query time is outside of range")
)

type LocationStore struct {
	SourceLocations locationData.SourceLocations

	// Time difference thresholds
	// Low time difference threshold
	LowTimeDiffThreshold time.Duration
	// Medium time difference threshold
	MediumTimeDiffThreshold time.Duration
}

func NewLocationStoreFromGoogleTimelinePath(path string) (*LocationStore, error) {
	sourceLocations, err := locationData.NewSourceLocationsFromGoogleTimeline(path)
	if err != nil {
		return nil, fmt.Errorf("error creating source locations: %v", err)
	}
	l := LocationStore{
		LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
		MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
		SourceLocations:         *sourceLocations,
	}
	return &l, nil
}

/*
This function will have to do some assumptions when the time between location stamps is too large.

It is implemented with 3 types of assumptions:

- If the time between the photo and the location is low we will assume it is accurate.
- If the time between the photo and the location is medium we will assume we will attempt a linear interpolation between the two locations.
- If the time is large we will return an error, and assume the user will have to provide the data themselves.
*/
// TODO: Add test for this function
func (locStore *LocationStore) GetCoordinatesByTime(qTime time.Time) (locationData.Coordinates, time.Duration, error) {
	// Find the closest location to the given time
	closestLocationInd, otherLocationInd := locStore.SourceLocations.FindClosestLocations(qTime)
	closestLocation := locStore.SourceLocations.Locations[closestLocationInd]

	// Check the time difference
	timeDiff := qTime.Sub(closestLocation.Time).Abs()
	switch {
	// If qTime is significantly before the earlies location record, return error
	case timeDiff > locStore.LowTimeDiffThreshold && qTime.Before(locStore.SourceLocations.Locations[0].Time):
		return locationData.Coordinates{}, 0, errors.Join(
			ErrQueryTimeIsOutsideOfRange,
			fmt.Errorf("before"),
		)
	// If qTime is significantly after the latest location record, return error
	case timeDiff > locStore.LowTimeDiffThreshold && qTime.After(locStore.SourceLocations.Locations[len(locStore.SourceLocations.Locations)-1].Time):
		return locationData.Coordinates{}, 0, errors.Join(
			ErrQueryTimeIsOutsideOfRange,
			fmt.Errorf("after"),
		)
	case timeDiff <= locStore.LowTimeDiffThreshold:
		// If the time difference is low, return the location
		fmt.Println("Chosing the closest")
		return closestLocation.Coordinates, timeDiff, nil
	case timeDiff <= locStore.MediumTimeDiffThreshold:
		// If the time difference is medium, attempt linear interpolation
		// Find the previous location
		fmt.Println("Interpolating")
		interCoord := locationData.Interpolation(
			locStore.SourceLocations.Locations[closestLocationInd],
			locStore.SourceLocations.Locations[otherLocationInd],
			qTime,
		)
		return interCoord, timeDiff, ErrTimeDiffMedium
	case timeDiff > locStore.MediumTimeDiffThreshold:
		// If the time difference is high, return an error
		return closestLocation.Coordinates, timeDiff, errors.Join(
			ErrTimeDiffTooHigh,
			fmt.Errorf("diff: %s", timeDiff),
		)
	default:
		return locationData.Coordinates{}, timeDiff, ErrNoLocation
	}
}
