package lib

import (
	geo "github.com/kellydunn/golang-geo"
)

type Corrdinates struct {
	geo.Point
}

func NewCorrdinatesE2(lat, long float64) Corrdinates {
	return Corrdinates{
		Point: *geo.NewPoint(lat, long),
	}
}

func NewCorrdinatesE7(lat, long int) Corrdinates {
	return Corrdinates{
		Point: *geo.NewPoint(float64(lat)/1e7, float64(long)/1e7),
	}
}

func NewCoordinatesFromGeopoint(point geo.Point) Corrdinates {
	return Corrdinates{
		Point: point,
	}
}

func (coordinate *Corrdinates) GetE2Coord() (lat float64, long float64) {
	return coordinate.Lat(), coordinate.Lng()
}

func (coordinate *Corrdinates) GetE7Coord() (lat int, long int) {
	return int(coordinate.Lat() * 1e7), int(coordinate.Lng() * 1e7)
}

func (coordinate *Corrdinates) LatE7() int {
	return int(coordinate.Point.Lat() * 1e7)
}

func (coordinate *Corrdinates) LngE7() int {
	return int(coordinate.Point.Lng() * 1e7)
}
