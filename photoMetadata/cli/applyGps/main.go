package main

import (
	"errors"
	"fmt"

	flags "github.com/jessevdk/go-flags"
	"github.com/sander-skjulsvik/tools/photoMetadata/lib"
)

var opts struct {
	// is encountered (can be set multiple times, like -vvv)
	Verbose          []bool `short:"v" long:"verbose" description:"Show verbose debug information"`
	PhotosPaths      string `short:"p" long:"photos" description:"Path to photos" required:"true"`
	LocationDataPath string `short:"l" long:"location" description:"Path to location data" required:"true"`
	DryRun           bool   `short:"d" long:"dryrun" description:"Dry run will not modify the photos" required:"false"`
	Threads          int    `short:"t" long:"threads" description:"number of simultaneously threads to use" required:"false" default:"10"`
}

func main() {

	_, err := flags.Parse(&opts)
	if err != nil {
		panic(errors.Join(fmt.Errorf("could not parse args"), err))
	}

	fmt.Printf("photoPath: %s, locationSource: %s\n", opts.PhotosPaths, opts.LocationDataPath)

	err = lib.ApplyLocationData(opts.PhotosPaths, opts.LocationDataPath, opts.Threads, false)
	if err != nil {
		fmt.Printf("Failed: %v\n", err)
	}
}
