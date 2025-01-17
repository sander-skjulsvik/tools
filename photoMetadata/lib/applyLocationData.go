package lib

import (
	"errors"
	"fmt"
	"sync"
)

type ApplyLocationOpts struct {
	DryRun       bool
	OverWrite    bool
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
			applyLocationData(photo, *locationStore, opts.DryRun, opts.OverWrite)
		}()
	}
	wg.Wait()
	return nil
}

var ErrApplyLocationDataReadGpsError = fmt.Errorf("parsing existing gps reocrd")

func applyLocationData(photo Photo, locationStore LocationStore, dryRun bool, overWrite bool) (bool, error) {
	if !overWrite {
		photoLocation, err := photo.GetLocationRecord()
		switch {
		case errors.Is(err, ErrGetLocationRecordGetTime):
			// Empty time is error case
			fmt.Printf("Unable to get time for %s\n", photo.Path)
			return false, fmt.Errorf("%w: %w", ErrApplyLocationDataReadGpsError, err)
		case errors.Is(err, ErrGetLocationRecordParsingGPS):
			// Failed to parse gps is error
			return false, fmt.Errorf("%w: %w", ErrApplyLocationDataReadGpsError, err)
		}
		if photoLocation != nil {
			return false, nil
		}
	}

	photoTime, err := photo.GetDateTimeOriginal()
	if err != nil {
		return false, fmt.Errorf("getting photo time for %s, err: %w", photo.Path, err)
	}

	photoTimeStr := photoTime
	fmt.Printf("photo time: \n\t%s\n\tt0: %s\n\tt1: %s\n",
		photoTimeStr,
		locationStore.SourceLocations.Locations[0].Time,
		locationStore.SourceLocations.Locations[1].Time,
	)

	coordinates, timeDiff, err := locationStore.GetCoordinatesByTime(photoTime)
	switch {
	case err == nil || errors.Is(err, ErrTimeDiffMedium):
		if dryRun {
			fmt.Printf("%s,\t%v,\ttime diff: %s\n", photo.Path, coordinates.CoordFuji(), timeDiff)
		} else {
			fmt.Printf("%s\t%v,\ttime diff: %s, writing\n", photo.Path, coordinates.CoordFuji(), timeDiff)
			photo.WriteExifGPSLocation(coordinates)
		}
		return true, nil
	default:
		
		return false, fmt.Errorf("applying location: %s: %w", photo.Path, err)
	}
}
