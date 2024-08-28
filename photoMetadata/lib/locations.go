package lib

import (
	"errors"
	"fmt"
	"time"

	locationData "github.com/sander-skjulsvik/tools/google_location_data/lib"
)

const (
	LOW_TIME_DIFF_THRESHOLD    = 30 * time.Minute
	MEDIUM_TIME_DIFF_THRESHOLD = 2 * time.Hour
	HIGH_TIME_DIFF_THRESHOLD   = 12 * time.Hour
)

var (
	ErrTimeDiffTooHigh = errors.New("Time difference too high")
	ErrTimeDiffMedium  = errors.New("Time difference medium")
	ErrNoLocation      = errors.New("No location found")
)

type LocationStore struct {
	SourceLocations locationData.SourceLocations

	// Time difference thresholds
	// Low time difference threshold
	LowTimeDiffThreshold time.Duration
	// Medium time difference threshold
	MediumTimeDiffThreshold time.Duration
	// High time difference threshold
	HighTimeDiffThreshold time.Duration
}

func NewLocationStoreFromGoogleTimelinePath(path string) (*LocationStore, error) {
	sourceLocations, err := locationData.NewSourceLocationsFromGoogleTimeline(path)
	if err != nil {
		return nil, fmt.Errorf("Error creating source locations: %v", err)
	}
	l := LocationStore{
		LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
		MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
		HighTimeDiffThreshold:   HIGH_TIME_DIFF_THRESHOLD,
		SourceLocations:         *sourceLocations,
	}
	return &l, nil
}

/*
This function will have to do some assumtions when the time between location stamps is too large.

It is implemented with 3 types of assumptions:

- If the time between the photo and the location is low we will assume it is accurate.
- If the time between the photo and the location is medium we will assume we will attempt a linear interpolation between the two locations.
- If the time is large we will return an error, and assume the user will have to provide the data themselves.
*/
func (locStore *LocationStore) GetCoordinatesByTime(time time.Time) (locationData.Coordinates, error) {
	// Find the closest location to the given time
	closestLocationInd, err := locStore.SourceLocations.FindClosestLocation(time)
	if err != nil {
		return locationData.Coordinates{}, err
	}
	closestLocation := locStore.SourceLocations.Locations[closestLocationInd]

	// Check the time difference
	timeDiff := time.Sub(closestLocation.Time)
	switch {
	case timeDiff <= locStore.LowTimeDiffThreshold:
		// If the time difference is low, return the location
		return closestLocation.Coordinates, nil
	case timeDiff <= locStore.MediumTimeDiffThreshold:
		// If the time difference is medium, attempt linear interpolation
		// Find the previous location
		return closestLocation.Coordinates, ErrTimeDiffMedium
	case timeDiff <= locStore.HighTimeDiffThreshold:
		// If the time difference is high, return an error
		return closestLocation.Coordinates, errors.Join(
			ErrTimeDiffTooHigh,
			fmt.Errorf("diff: %s", timeDiff),
		)
	}

	// Return the location
	return locationData.Coordinates{}, ErrNoLocation
}
