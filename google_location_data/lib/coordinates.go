package lib

import (
	"errors"
	"fmt"
	"strings"

	nmea "github.com/adrianmo/go-nmea"
	geo "github.com/kellydunn/golang-geo"
)

type Coordinates struct {
	geo.Point
}

func NewCoordinatesE2(lat, long float64) Coordinates {
	return Coordinates{
		Point: *geo.NewPoint(lat, long),
	}
}

func NewCoordinatesE7(lat, long int) Coordinates {
	return Coordinates{
		Point: *geo.NewPoint(float64(lat)/1e7, float64(long)/1e7),
	}
}

func NewCoordinatesFromGeoPoint(point geo.Point) Coordinates {
	return Coordinates{
		Point: point,
	}
}

var ErrInvalidDMS = errors.New("invalid DMS")

func NewCoordinatesFromDMS(latitude, longitude string) (Coordinates, error) {
	lat, err := DMSCoordinateToE2(latitude)
	if err != nil {
		return Coordinates{}, errors.Join(
			fmt.Errorf("latitude"),
			err,
		)
	}
	lng, err := DMSCoordinateToE2(longitude)
	if err != nil {
		return Coordinates{}, errors.Join(
			fmt.Errorf("longitude"),
			err,
		)
	}

	return NewCoordinatesE2(lat, lng), nil
}

func DMSCoordinateToE2(c string) (float64, error) {
	cleanC := strings.ReplaceAll(c, " deg", string(nmea.Degrees))
	cleanC = strings.ReplaceAll(cleanC, "N", "")
	cleanC = strings.ReplaceAll(cleanC, "E", "")

	parsedC, err := nmea.ParseDMS(cleanC)

	if err != nil {
		return -1, errors.Join(
			ErrInvalidDMS,
			err,
		)
	}
	return parsedC, nil
}

func (c *Coordinates) CoordE2() (lat float64, long float64) {
	return c.Lat(), c.Lng()
}

func (c *Coordinates) String() string {
	return fmt.Sprintf("Lat: %f, Lng: %f", c.Lat(), c.Lng())
}

func (c *Coordinates) CoordE7() (lat int, long int) {
	return int(c.Lat() * 1e7), int(c.Lng() * 1e7)
}

func (c *Coordinates) LatE7() int {
	return int(c.Point.Lat() * 1e7)
}

func (c *Coordinates) LngE7() int {
	return int(c.Point.Lng() * 1e7)
}

func (c *Coordinates) LngFuji() string {
	return CoordToFuji(c.Lng()) + " E"

}

func (c *Coordinates) LatFuji() string {
	return CoordToFuji(c.Lat()) + " N"
}

func CoordToFuji(coordE2 float64) string {
	formatted := nmea.FormatDMS(coordE2)
	formatted = strings.ReplaceAll(formatted, string(nmea.Degrees), " deg")
	return formatted
}

func (c *Coordinates) CoordFuji() string {
	lat := c.LatFuji()
	lng := c.LngFuji()
	r := strings.Join(
		[]string{lat, lng},
		",",
	)
	if r == "" || r == "," {
		panic(
			fmt.Sprintf("coordDMS: returned empty string from: lat: %f, lng: %f", c.Lat(), c.Lng()),
		)
	}
	return r

}

func (c *Coordinates) Equal(other Coordinates) bool {
	if c.Lat() != other.Lat() {
		return false
	}
	if c.Lng() != other.Lng() {
		return false
	}
	return true
}
