package main

import (
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
		Write(photoPath)
		// List(photoPath)
		pos := p.SearchExifData("GPSPosition")
		fmt.Println(pos)
	case "search":
		pos := p.SearchExifData("GPSPosition")
		fmt.Println(pos)
	case "applyGPS":
		ApplyLocationData(photoPath, locationSource)
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
		photoTime, err := photo.GetTime()
		if err != nil {
			fmt.Printf("Error getting photo time for %s, err: %v", photo.Path, err)
			continue
		}
		location, err := locationStore.GetLocationByTime(photoTime)
		switch err {
		case nil:
			fmt.Printf("Applying location data to %s: %v\n", photo.Path, location)
			if !dryRun {
				err = photo.WriteExifData("GPSPosition", location.String())
				if err != nil {
					fmt.Printf("Error writing location data to %s: %v", photo.Path, err)
				}
			}
		case locationData.ErrTimeDiffMedium:
			fmt.Printf("Time difference is medium, interpolating for %s\n", photo.Path)
			if !dryRun {
				err = photo.WriteExifData("GPSPosition", location.String())
				if err != nil {
					fmt.Printf("Error writing location data to %s: %v", photo.Path, err)
				}
			}

		case locationData.ErrTimeDiffTooHigh:
			fmt.Printf("Time difference is too high for %s\n", photo.Path)
		case locationData.ErrNoLocation:
			fmt.Printf("No location found for %s\n", photo.Path)
		default:
			fmt.Printf("Error getting location for %s: %v", photo.Path, err)
		}

	}
	return nil
}

func List(path string) {
	data, err := lib.GetAllExifData(path)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return
	}
	err = lib.PrintAllExifData(data)
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
}

func Write(path string) error {
	err := lib.WriteExifDataToFile("GPSPosition", "61 deg 39' 50.71\" N, 9 deg 39' 57.94\" E", path)
	if err != nil {
		return fmt.Errorf("Error writing to file: %v", err)
	}
	return nil
}
