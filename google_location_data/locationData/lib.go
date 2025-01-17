package locationData

import (
	"time"

	geo "github.com/kellydunn/golang-geo"
	toolsTime "github.com/sander-skjulsvik/tools/libs/time"
)

func Interpolation(loc1, loc2 LocationRecord, searchTime time.Time) Coordinates {
	// Early exit
	if loc1.Time == searchTime {
		return loc1.Coordinates
	}
	if loc2.Time == searchTime {
		return loc1.Coordinates
	}
	// Calculate the ratio of the time difference
	timeRatio := toolsTime.GetTimeRatio(loc1.Time, loc2.Time, searchTime)
	// Normalized ratio
	loc1LatitudeE2, loc1LongitudeE2 := loc1.Coordinates.CoordE2()
	loc2LatitudeE2, loc2LongitudeE2 := loc2.Coordinates.CoordE2()

	p1 := geo.NewPoint(loc1LatitudeE2, loc1LongitudeE2)
	p2 := geo.NewPoint(loc2LatitudeE2, loc2LongitudeE2)

	bearing := p1.BearingTo(p2)
	distance := p1.GreatCircleDistance(p2)

	scaledDistance := distance*timeRatio

	p3 := p1.PointAtDistanceAndBearing(scaledDistance, bearing)

	return NewCoordinatesFromGeoPoint(*p3)
}
