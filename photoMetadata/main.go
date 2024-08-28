package main

import (
	"errors"
	"flag"
	"fmt"
	"sync"

	"github.com/sander-skjulsvik/tools/photoMetadata/lib"
)

func main() {

	var (
		photoPath      string
		locationSource string
	)
	flag.StringVar(&photoPath, "photoPath", ".", "Path to photos directory")
	flag.StringVar(&locationSource, "locationSource", ".", "Path to location source")
	flag.Parse()

	fmt.Printf("photoPath: %s, locationSource: %s\n", photoPath, locationSource)

	err := ApplyLocationData(photoPath, locationSource, true)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	}
}

func ApplyLocationData(photoPath, locationPath string, dryRun bool) error {
	var (
		photos        *lib.PhotoCollection
		locationStore *lib.LocationStore
		err           error
	)
	fmt.Println("Loading photo collection")
	photos, err = lib.NewPhotoCollectionFromPath(photoPath)
	if err != nil {
		return fmt.Errorf("error creating photo collection: %v", err)
	}
	fmt.Printf("\tFound %d photos\n", len(photos.Photos))
	fmt.Println("Loading location store")
	locationStore, err = lib.NewLocationStoreFromGoogleTimelinePath(locationPath)
	if err != nil {
		return fmt.Errorf("error creating location store: %v", err)
	}

	fmt.Printf("\tFound %d location record\n", len(locationStore.SourceLocations.Locations))
	sem := make(chan int, 10)
	var wg sync.WaitGroup
	for _, photo := range photos.Photos {
		wg.Add(1)
		sem <- 1
		go applyLocationData(photo, *locationStore, dryRun, sem, &wg)

	}
	wg.Wait()
	return nil
}

func applyLocationData(photo lib.Photo, locationStore lib.LocationStore, dryRun bool, sem chan int, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		<-sem
	}()
	// If photo already has gps location pass
	photoLocation, err := photo.GetLocationRecord()
	switch {
	case photoLocation == nil:
		// fmt.Printf("Found no location: %s\n", photo.Path)
	case errors.Is(err, lib.ErrGetLocationRecordGetTime):
		// Empty time is error case
		fmt.Printf("Unable to get time for %s\n", photo.Path)
		return
	case errors.Is(err, lib.ErrGetLocationRecordGPSempty):
		// Empty GPS is fine
	case errors.Is(err, lib.ErrGetLocationRecordGPSstring):
		// Failed to sting GPS is error
		fmt.Printf("%v\n", err)
		return
	case errors.Is(err, lib.ErrGetLocationRecordParsingGPS):
		// Failed to parse gps is error
		fmt.Printf("%v, for %s\n", err, photo.Path)
		return
	}

	photoTime, err := photo.GetDateTimeOriginal()
	if err != nil {
		fmt.Printf("Error getting photo time for %s, err: %v", photo.Path, err)
		return
	}

	coordinates, err := locationStore.GetCoordinatesByTime(photoTime)
	switch {
	case err == nil || errors.Is(err, lib.ErrTimeDiffMedium):
		if !dryRun {
			fmt.Printf("%s: %v\n", photo.Path, coordinates.CoordDMS())

			photo.WriteExifGPSLocation(coordinates)
		} else {
			fmt.Printf("%s: %v\n", photo.Path, coordinates.CoordDMS())

		}
	default:
		fmt.Printf("Error applying location: %s: %v\n", photo.Path, err)
		return
	}
}
