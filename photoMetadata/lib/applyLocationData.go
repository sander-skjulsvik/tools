package lib

import (
	"errors"
	"fmt"
	"sync"
)

type ApplyLocationOpts struct {
	DryRun       bool
	LocationPath string
	PhotoPath    string
	Threads      int
	Verbose      bool
}

func ApplyLocationsData(opts ApplyLocationOpts) error {
	var (
		photos        *PhotoCollection
		locationStore *LocationStore
		err           error
	)

	fmt.Println("Loading photo collection")
	photos, err = NewPhotoCollectionFromPath(opts.PhotoPath)
	if err != nil {
		return fmt.Errorf("error creating photo collection: %v", err)
	}
	if opts.Verbose {
		fmt.Printf("\tFound %d photos\n", len(photos.Photos))
	}

	fmt.Println("Loading location store")
	locationStore, err = NewLocationStoreFromGoogleTimelinePath(opts.LocationPath)
	if err != nil {
		return fmt.Errorf("error creating location store: %v", err)
	}
	if opts.Verbose {
		fmt.Printf("\tFound %d location record\n", len(locationStore.SourceLocations.Locations))
	}
	sem := make(chan int, opts.Threads)
	var wg sync.WaitGroup
	for _, photo := range photos.Photos {
		wg.Add(1)
		sem <- 1
		go func() {
			defer func() {
				wg.Done()
				<-sem
			}()
			applyLocationData(photo, *locationStore, opts.DryRun)
		}()
	}
	wg.Wait()
	return nil
}

func applyLocationData(photo Photo, locationStore LocationStore, dryRun bool) {
	// If photo already has gps location pass
	photoLocation, err := photo.GetLocationRecord()
	switch {
	case photoLocation == nil:
		// fmt.Printf("Found no location: %s\n", photo.Path)
	case errors.Is(err, ErrGetLocationRecordGetTime):
		// Empty time is error case
		fmt.Printf("Unable to get time for %s\n", photo.Path)
		return
	case errors.Is(err, ErrGetLocationRecordGPSempty):
		// Empty GPS is fine
	case errors.Is(err, ErrGetLocationRecordGPSstring):
		// Failed to sting GPS is error
		fmt.Printf("%v\n", err)
		return
	case errors.Is(err, ErrGetLocationRecordParsingGPS):
		// Failed to parse gps is error
		fmt.Printf("%v, for %s\n", err, photo.Path)
		return
	}

	photoTime, err := photo.GetDateTimeOriginal()
	if err != nil {
		fmt.Printf("Error getting photo time for %s, err: %v", photo.Path, err)
		return
	}

	coordinates, timediff, err := locationStore.GetCoordinatesByTime(photoTime)
	switch {
	case err == nil || errors.Is(err, ErrTimeDiffMedium):
		if !dryRun {
			fmt.Printf("%s\t%v,\ttime diff: %s\n", photo.Path, coordinates.CoordFuji(), timediff)
			photo.WriteExifGPSLocation(coordinates)
		} else {
			fmt.Printf("%s,\t%v,\ttime diff: %s\n", photo.Path, coordinates.CoordFuji(), timediff)
		}
	default:
		fmt.Printf("Error applying location: %s: %v\n", photo.Path, err)
		return
	}
}
