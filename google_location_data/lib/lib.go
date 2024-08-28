package lib

import (
	"time"

	geo "github.com/kellydunn/golang-geo"
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
