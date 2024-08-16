package lib

import (
	"fmt"
	"log"
	"sort"
	"time"

	geo "github.com/kellydunn/golang-geo"
)

var RFC3339_LAYOUT string = "2006-01-02T15:04:05Z07:00"

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

const (
	LOW_TIME_DIFF_THRESHOLD    = 30 * time.Minute
	MEDIUM_TIME_DIFF_THRESHOLD = 2 * time.Hour
	HIGH_TIME_DIFF_THRESHOLD   = 12 * time.Hour
)

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
	Corrdinates Corrdinates `json:"coordinates"`
	Time        time.Time   `json:"timestamp"`
}

func ParseTime(timeStr string) time.Time {
	t, err := time.Parse(RFC3339_LAYOUT, timeStr)
	if err != nil {
		log.Fatalf("Unable to parse timestamp: %v", err)
	}
	return t
}

type Corrdinates struct {
	LatitudeE7  int
	LongitudeE7 int
}

var (
	ErrLatitudeOutOfRange  = fmt.Errorf("Latitude out of range")
	ErrLongitudeOutOfRange = fmt.Errorf("Longitude out of range")
)

func NewCorrdinatesE7(lat, long int) (Corrdinates, error) {
	if lat < -90e7 || lat > 90e7 {
		return Corrdinates{}, fmt.Errorf("%w: %d", ErrLatitudeOutOfRange, lat)
	}
	if long < -180e7 || long > 180e7 {
		return Corrdinates{}, fmt.Errorf("%w: %d", ErrLongitudeOutOfRange, long)
	}
	return Corrdinates{
		LatitudeE7:  lat,
		LongitudeE7: long,
	}, nil
}

func NewCorrdinatesE2(lat, long float64) (Corrdinates, error) {
	if lat < -90 || lat > 90 {
		return Corrdinates{}, fmt.Errorf("%w: %f", ErrLatitudeOutOfRange, lat)
	}
	if long < -180 || long > 180 {
		return Corrdinates{}, fmt.Errorf("%w: %f", ErrLongitudeOutOfRange, long)
	}
	return Corrdinates{
		LatitudeE7:  int(lat * 1e7),
		LongitudeE7: int(long * 1e7),
	}, nil
}

func (coordinate *Corrdinates) GetE2Coord() (lat float64, long float64) {
	return float64(coordinate.LatitudeE7) / 1e7, float64(coordinate.LongitudeE7) / 1e7
}

/*
This function will have to do some assumtions when the time between location stamps is too large.

It is implemented with 3 types of assumptions:

- If the time between the photo and the location is low we will assume it is accurate.
- If the time between the photo and the location is medium we will assume we will attempt a linear interpolation between the two locations.
- If the time is large we will return an error, and assume the user will have to provide the data themselves.
*/

var (
	ErrTimeDiffTooHigh = fmt.Errorf("Time difference too high")
	ErrTimeDiffMedium  = fmt.Errorf("Time difference medium")
	ErrNoLocation      = fmt.Errorf("No location found")
)

func (locStore *LocationStore) GetLocationByTime(time time.Time) (Corrdinates, error) {
	// Find the closest location to the given time
	closestLocationInd, err := locStore.SourceLocations.FindClosestLocation(time)
	if err != nil {
		return Corrdinates{}, err
	}
	closestLocation := locStore.SourceLocations.Locations[closestLocationInd]

	// Check the time difference
	timeDiff := time.Sub(closestLocation.Time)
	switch {
	case timeDiff <= locStore.LowTimeDiffThreshold:
		// If the time difference is low, return the location
		return closestLocation.Corrdinates, nil
	case timeDiff <= locStore.MediumTimeDiffThreshold:
		// If the time difference is medium, attempt linear interpolation
		// Find the previous location
		return closestLocation.Corrdinates, ErrTimeDiffMedium
	case timeDiff <= locStore.HighTimeDiffThreshold:
		// If the time difference is high, return an error
		return closestLocation.Corrdinates, fmt.Errorf("Time difference too high: %v", timeDiff)
	}

	// Return the location
	return Corrdinates{}, ErrNoLocation
}

func interpolation(loc1, loc2 LocationRecord, time time.Time) Corrdinates {
	// Calculate the ratio of the time difference
	timeRatio := timeRatio(loc1.Time, loc2.Time, time)
	// Normalized ratio
	loc1LatitudeE2, loc1LongitudeE2 := loc1.Corrdinates.GetE2Coord()
	loc2LatitudeE2, loc2LongitudeE2 := loc2.Corrdinates.GetE2Coord()

	p1 := geo.NewPoint(loc1LatitudeE2, loc1LongitudeE2)
	p2 := geo.NewPoint(loc2LatitudeE2, loc2LongitudeE2)

	bearing := p1.BearingTo(p2)
	distance := p1.GreatCircleDistance(p2)

	p3 := p1.PointAtDistanceAndBearing(distance*timeRatio, bearing)

	c, err := NewCorrdinatesE2(p3.Lat(), p3.Lng())
	if err != nil {
		log.Fatalf("Unable to create coordinates: %v", err)
	}

	return c
}

func timeRatio(time1, time2, time time.Time) float64 {
	return time.Sub(time1).Seconds() / time2.Sub(time1).Seconds()
}
