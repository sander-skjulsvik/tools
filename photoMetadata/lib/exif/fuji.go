package exif

import (
	"fmt"
	"time"

	"github.com/sander-skjulsvik/tools/google_location_data/locationData"
)

const (
	FUJI_CREATION_TIME_EXIF_NAME = "SubSecDateTimeOriginal"
	FUJI_LATITUDE_EXIF_NAME      = "GPSLatitude"
	FUJI_LONGITUDE_EXIF_NAME     = "GPSLongitude"
	FUJI_DATE_TIME_FORMAT        = "2006:01:02 15:04:05-07:00"
)

type FujiRAF struct {
	CreationTimeExifName string
	LatitudeExifName     string
	LongitudeExifName    string
	DateTimeFormat       string
}

func NewFujiRaf() FujiRAF {
	return FujiRAF{
		CreationTimeExifName: FUJI_CREATION_TIME_EXIF_NAME,
		LatitudeExifName:     FUJI_LATITUDE_EXIF_NAME,
		LongitudeExifName:    FUJI_LONGITUDE_EXIF_NAME,
		DateTimeFormat:       FUJI_DATE_TIME_FORMAT,
	}
}

func (fujiRaf *FujiRAF) WriteDateTime(filePath string, t time.Time) error {
	if err := WriteExifDataToFile(
		fujiRaf.CreationTimeExifName,
		t.Format(fujiRaf.DateTimeFormat),
		filePath,
	); err != nil {
		return fmt.Errorf("fuji write date time: %w", err)
	}
	return nil
}

func (fujiRaf *FujiRAF) GetCrationDateTime(filepath string) (*time.Time, error) {
	str, err := GetExifValue(filepath, fujiRaf.CreationTimeExifName)
	if err != nil {
		return nil, fmt.Errorf("fuji raf failed to get crationtime: %w", err)
	}
	creation, err := time.Parse(fujiRaf.DateTimeFormat, str)
	if err != nil {
		return nil, fmt.Errorf("fuji raf failed to parse creation time: %w", err)
	}

	return &creation, err
}

func (fuijiRaf *FujiRAF) GetLocationRecord(filePath string) (*locationData.Coordinates, error) {
	latString, err := GetExifValue(filePath, fuijiRaf.LatitudeExifName)
	if err != nil {
		return nil, fmt.Errorf("failed to get latitude")
	}
	longString, err := GetExifValue(filePath, fuijiRaf.LongitudeExifName)
	if err != nil {
		return nil, fmt.Errorf("failed to get ongitude")
	}

	coord, err := locationData.NewCoordinatesFromDMS(latString, longString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fuji raf coordinates: %w", err)
	}

	return &coord, nil
}

func (fujiRaf *FujiRAF) WriteLocation(filepath string, coordinates *locationData.Coordinates ) error {
	fujiLat := coordinates.LatFuji()
	fujiLng := coordinates.LngFuji()

	if err := WriteExifDataToFile(fujiRaf.LatitudeExifName, fujiLat, filepath); err != nil {
		return fmt.Errorf("could not write fuji latitude to file: %w", err)
	}
	if err := WriteExifDataToFile(fujiRaf.LongitudeExifName, fujiLng, filepath); err != nil {
		return fmt.Errorf("could not write fuji longitude to file: %w", err)
	}
	return nil
}
