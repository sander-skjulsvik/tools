package exif

import "time"

type ExifSpec interface {
	WriteDateTime(time.Time) error
	GetCrationDateTime() (time.Time, error)
	GetLocationRecord() 
	WriteLocation() error
}


