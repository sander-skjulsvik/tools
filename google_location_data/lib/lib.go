package lib

import (
	"errors"
	"time"

	geo "github.com/kellydunn/golang-geo"
)

/*
This function will have to do some assumtions when the time between location stamps is too large.

It is implemented with 3 types of assumptions:

- If the time between the photo and the location is low we will assume it is accurate.
- If the time between the photo and the location is medium we will assume we will attempt a linear interpolation between the two locations.
- If the time is large we will return an error, and assume the user will have to provide the data themselves.
*/

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

	return NewCoordinatesFromGeopoint(*p3)
}

func timeRatio(time1, time2, time time.Time) float64 {
	return time.Sub(time1).Seconds() / time2.Sub(time1).Seconds()
}
