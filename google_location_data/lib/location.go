package lib

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"time"
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
	SourceLocations SourceLocations

	// Time difference thresholds
	// Low time difference threshold
	LowTimeDiffThreshold time.Duration
	// Medium time difference threshold
	MediumTimeDiffThreshold time.Duration
	// High time difference threshold
	HighTimeDiffThreshold time.Duration
}

func NewLocationStoreFromGoogleTimelinePath(path string) (*LocationStore, error) {
	sourceLocations, err := NewSourceLocationsFromGoogleTimeline(path)
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
func (locStore *LocationStore) GetCoordinatesByTime(time time.Time) (Coordinates, error) {
	// Find the closest location to the given time
	closestLocationInd, err := locStore.SourceLocations.FindClosestLocation(time)
	if err != nil {
		return Coordinates{}, err
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
			fmt.Errorf("Diff: %s", timeDiff),
		)
	}

	// Return the location
	return Coordinates{}, ErrNoLocation
}

type SourceLocations struct {
	Locations []LocationRecord `json:"locations"`
}

// This function assumes Locations is sorted by time
func (sourceData *SourceLocations) FindClosestLocation(time time.Time) (ind int, err error) {
	var (
		locationBeforeInd int
		locationAfterInd  int
	)

	locationAfterInd = sort.Search(len(sourceData.Locations), func(i int) bool {
		return sourceData.Locations[i].Time.After(time)
	})
	log.Default().Printf("locationAfterInd: %d", locationAfterInd)
	// Handling edge cases, max and min
	if locationAfterInd == len(sourceData.Locations) {
		return locationAfterInd - 1, nil
	}
	if locationAfterInd == 0 {
		return locationAfterInd, nil
	}

	locationBeforeInd = locationAfterInd - 1
	afterTime := sourceData.Locations[locationAfterInd].Time
	beforeTime := sourceData.Locations[locationBeforeInd].Time

	if time.Sub(afterTime).Abs() <= time.Sub(beforeTime).Abs() {
		return locationAfterInd, nil
	} else {
		return locationBeforeInd, nil
	}
}

func (sourceData *SourceLocations) SortByTime() {
	// Sort the data by time
	sort.Sort(ByTime(sourceData.Locations))
	// return sourceData.Locations
}

type ByTime []LocationRecord

func (bt ByTime) Less(i, j int) bool {
	return bt[i].Time.Before(bt[j].Time)
}

func (bt ByTime) Swap(i, j int) {
	bt[i], bt[j] = bt[j], bt[i]
}

func (bt ByTime) Len() int {
	return len(bt)
}

type LocationRecord struct {
	Coordinates Coordinates `json:"coordinates"`
	Time        time.Time   `json:"timestamp"`
}
