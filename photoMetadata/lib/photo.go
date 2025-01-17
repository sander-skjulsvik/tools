package lib

import (
	"errors"
	"fmt"
	"time"

	locationData "github.com/sander-skjulsvik/tools/google_location_data/locationData"
	"github.com/sander-skjulsvik/tools/libs/files"
	"github.com/sander-skjulsvik/tools/photoMetadata/lib/exif"
)

// should all be lowercase
var SUPPORTED_FILE_TYPES = []string{
	".raf",
}

type PhotoCollection struct {
	Photos []Photo
}

func NewPhotoCollectionFromPath(path string) (*PhotoCollection, error) {
	collection := PhotoCollection{}
	paths, err := files.GetAllFilesOfTypes(path, SUPPORTED_FILE_TYPES)
	if err != nil {
		return nil, fmt.Errorf("error getting files %v", err)
	}
	for _, path := range paths {
		collection.Photos = append(collection.Photos, *NewPhotoFromPath(path))
	}
	return &collection, nil
}

type Photo struct {
	Path string
}

// New photo funcs
func NewPhotoFromPath(path string) *Photo {
	return &Photo{Path: path}
}

const (
	FUJI_RAW_TIME_EXIF_NAME = "SubSecDateTimeOriginal"
	FUJI_RAW_TIME_LAYOUT    = "2006:01:02 15:04:05-07:00"
	FUJI_RAW_GPS_LATITUDE   = "GPSLatitude"
	FUJI_RAW_GPS_LONGITUDE  = "GPSLongitude"
	GPSDateTime             = "GPSDateTime"
	GPSPosition             = "GPSPosition"
)

// Photo methods

func (photo *Photo) SearchExifData(search string) interface{} {
	fileInfos, err := exif.GetAllExifData(photo.Path)
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

func (p *Photo) WriteExifData(key, value string) error {
	return exif.WriteExifDataToFile(key, value, p.Path)
}

/*
fuji date time format:  2023:11:07 11:46:28+01:00
go layout: 2006:01:02 15:04:05-07:00
*/
func (photo *Photo) GetDateTimeOriginal() (time.Time, error) {
	dateTimeOriginal, ok := photo.SearchExifData(FUJI_RAW_TIME_EXIF_NAME).(string)
	if !ok {
		return time.Time{}, fmt.Errorf("%s not found", FUJI_RAW_TIME_EXIF_NAME)
	}
	if dateTimeOriginal == "" {
		return time.Time{}, fmt.Errorf("%s empty", FUJI_RAW_TIME_EXIF_NAME)
	}
	parsedTime, err := time.Parse("2006:01:02 15:04:05-07:00", dateTimeOriginal)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing dateTimeOriginal: %v", err)
	}

	return parsedTime, nil
}

func (photo *Photo) WriteDateTime(t time.Time) error {
	err := photo.WriteExifData(FUJI_RAW_TIME_EXIF_NAME, t.Format(FUJI_RAW_TIME_LAYOUT))
	if err != nil {
		return errors.Join(
			err, fmt.Errorf("failed to write time to photo: %s", photo.Path),
		)
	}
	return nil
}

var (
	ErrGetLocationRecordGetTime    = errors.New("error getting dateTimeOriginal")
	ErrGetLocationRecordGPSstring  = errors.New("GPS Position unable to string assert")
	ErrGetLocationRecordParsingGPS = errors.New("error parsing GPSPosition")
)

func (photo *Photo) GetLocationRecord() (*locationData.LocationRecord, error) {
	// Location
	lat, ok := photo.SearchExifData(FUJI_RAW_GPS_LATITUDE).(string)
	if !ok {
		return nil, fmt.Errorf("failed to get latitude: %w", ErrGetLocationRecordGPSstring)
	}
	if lat == "" {
		return nil, nil
	}

	lng, ok := photo.SearchExifData(FUJI_RAW_GPS_LONGITUDE).(string)
	if !ok {
		return nil, fmt.Errorf("failed to get longitude: %w",  ErrGetLocationRecordGPSstring)
	}
	if lng == "" {
		return nil, nil
	}

	coords, err := locationData.NewCoordinatesFromDMS(lat, lng)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetLocationRecordParsingGPS,err)
	}

	// Time
	dateTimeOriginal, err := photo.GetDateTimeOriginal()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetLocationRecordGetTime, err)
	}

	return &locationData.LocationRecord{
		Coordinates: coords,
		Time:        dateTimeOriginal,
	}, nil
}

func (p *Photo) WriteExifGPSLocation(coordinates locationData.Coordinates) {
	p.WriteExifData(
		"GPSPosition",
		coordinates.CoordFuji(),
	)
	p.WriteExifData(
		FUJI_RAW_GPS_LONGITUDE,
		coordinates.LngFuji(),
	)
	p.WriteExifData(
		FUJI_RAW_GPS_LATITUDE,
		coordinates.LatFuji(),
	)
}
