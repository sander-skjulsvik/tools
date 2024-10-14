package main

import (
	"errors"
	"fmt"

	flags "github.com/jessevdk/go-flags"
	"github.com/sander-skjulsvik/tools/photoMetadata/lib"
)

type opts struct {
	// is encountered (can be set multiple times, like -vvv)
	DryRun           bool   `short:"d" long:"dryrun" description:"Dry run will not modify the photos" required:"false"`
	LocationDataPath string `short:"l" long:"location" description:"Path to location data" required:"true"`
	PhotosPath       string `short:"p" long:"photos" description:"Path to photos" required:"true"`
	Threads          int    `short:"t" long:"threads" description:"number of simultaneously threads to use" required:"false" default:"10"`
	Verbose          bool   `short:"v" long:"verbose" description:"Show verbose debug information"`
}

func main() {
	args := opts{}
	_, err := flags.Parse(&args)
	if err != nil {
		panic(errors.Join(fmt.Errorf("could not parse args"), err))
	}

	fmt.Printf("photoPath: %s, locationSource: %s\n", args.PhotosPath, args.LocationDataPath)

	err = lib.ApplyLocationsData(lib.ApplyLocationOpts{
		LocationPath: args.LocationDataPath,
		PhotoPath:    args.PhotosPath,
		Threads:      args.Threads,
		Verbose:      args.Verbose,
	})
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	}
}
