package lib

import (
	"errors"
	"fmt"
	"strings"
	"time"

	locationData "github.com/sander-skjulsvik/tools/google_location_data/lib"
)

type Photo struct {
	Path string
}

const (
	DateTimeOriginal   = "DateTimeOriginal"
	GPSPosition        = "GPSPosition"
	GPSDateTime        = "GPSDateTime"
	ExifDateTimeLatout = "2006:01:02 15:04:05-07:00"
)

// New photo funcs

// Photo methods

func (photo *Photo) SearchExifData(search string) interface{} {
	fmt.Println("Searching for data")

	fileInfos, err := GetAllExifData(photo.Path)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return nil
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.Err != nil {
			fmt.Printf("Error concerning %v: %v\n", fileInfo.File, fileInfo.Err)
			continue
		}

		for k, v := range fileInfo.Fields {
			if k == search {
				return v
			}
		}
	}
	return nil
}

/*
fuji date time format:  2023:11:07 11:46:28+01:00
go layout: 2006:01:02 15:04:05-07:00
*/
func (photo *Photo) GetDateTimeOriginal() (time.Time, error) {
	dateTimeOriginal, ok := photo.SearchExifData(DateTimeOriginal).(string)
	if !ok {
		return time.Time{}, errors.New("dateTimeOriginal not found")
	}
	if dateTimeOriginal == "" {
		return time.Time{}, errors.New("dateTimeOriginal empty")
	}
	parsedTime, err := time.Parse(ExifDateTimeLatout, dateTimeOriginal)
	if err != nil {
		return time.Time{}, fmt.Errorf("Error parsing dateTimeOriginal: %v", err)
	}

	return parsedTime, nil
}

func (photo *Photo) GetLocationRecord() (*locationData.LocationRecord, error) {
	// Location
	gpsPosition, ok := photo.SearchExifData("GPSPosition").(string)
	if !ok {
		return nil, errors.New("GPSPosition unable to string assert")
	}
	if gpsPosition == "" {
		return nil, errors.New("GPSPosition empty")
	}
	latLong := strings.Split(gpsPosition, ",")
	coords, err := locationData.NewCoordinatesFromDMS(latLong[0], latLong[1])
	if err != nil {
		return nil, fmt.Errorf("Error parsing GPSPosition: %v", err)
	}

	// Time
	dateTimeOriginal, err := photo.GetDateTimeOriginal()
	if err != nil {
		return nil, fmt.Errorf("Error getting dateTimeOriginal: %v", err)
	}

	return &locationData.LocationRecord{
		Corrdinates: coords,
		Time:        dateTimeOriginal,
	}, nil
}
