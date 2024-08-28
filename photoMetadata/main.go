package main

import (
	"errors"
	"fmt"
	"os"

	locationData "github.com/sander-skjulsvik/tools/google_location_data/lib"
	"github.com/sander-skjulsvik/tools/photoMetadata/lib"
)

func main() {
	method := os.Args[1]
	photoPath := os.Args[2]
	locationSource := os.Args[3]

	p := lib.Photo{Path: photoPath}

	switch method {
	case "list":
		List(photoPath)
	case "write":
	// 	Write(photoPath)
	// 	// List(photoPath)
	// 	pos := p.SearchExifData("GPSPosition")
	// 	fmt.Println(pos)
	case "search":
		pos := p.SearchExifData("GPSPosition")
		fmt.Println(pos)
	case "applyGPS":
		ApplyLocationData(photoPath, locationSource, true)
	default:
		fmt.Println("Invalid method")
	}
}

func ApplyLocationData(photoPath, locationPath string, dryRun bool) error {
	var (
		photos        *lib.PhotoCollection
		locationStore *locationData.LocationStore
		err           error
	)
	photos, err = lib.NewPhotoCollectionFromPath(photoPath)
	if err != nil {
		return fmt.Errorf("Error creating photo collection: %v", err)
	}
	locationStore, err = locationData.NewLocationStoreFromGoogleTimelinePath(locationPath)
	if err != nil {
		return fmt.Errorf("Error creating location store: %v", err)
	}

	for _, photo := range photos.Photos {
		// If photo already has gps location pass

		photoLocation, err := photo.GetLocationRecord()
		switch {
		case photoLocation == nil:
			fmt.Printf("Skipping: %s\n", photo.Path)
			continue
		case errors.Is(err, lib.ErrGetLocationRecordGetTime):
			// Empty time is error case
			fmt.Printf("Unable to get time for %s\n", photo.Path)
			continue
		case errors.Is(err, lib.ErrGetLocationRecordGPSempty):
			// Empty GPS is fine
		case errors.Is(err, lib.ErrGetLocationRecordGPSstring):
			// Failed to sting GPS is error
			fmt.Printf("%v\n", err)
			continue
		case errors.Is(err, lib.ErrGetLocationRecordParsingGPS):
			// Failed to parse gps is error
			fmt.Printf("%v, for %s\n", err, photo.Path)
			continue
		}

		photoTime, err := photo.GetDateTimeOriginal()
		if err != nil {
			fmt.Printf("Error getting photo time for %s, err: %v", photo.Path, err)
			continue
		}
		coordinates, err := locationStore.GetCoordinatesByTime(photoTime)
		switch {
		case err == nil || errors.Is(err, locationData.ErrTimeDiffMedium):
			fmt.Printf("Applying location data to %s: %v\n", photo.Path, coordinates.CoordDMS())
			if !dryRun {
				photo.WriteExifGPSLocation(coordinates)
			}
		default:
			fmt.Printf("Error applying location: %s: %v\n", photo.Path, err)
			continue
		}

	}
	return nil
}

func List(path string) {
	data, err := lib.GetAllExifData(path)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	err = lib.PrintAllExifData(data)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
