package main

import (
	"fmt"
	"strings"

	"github.com/jessevdk/go-flags"
	photoMetadata "github.com/sander-skjulsvik/tools/photoMetadata/lib"
)

var opts struct {
	FilePath     string `short:"p" long:"path" description:"pathToPhoto" required:"true"`
	SearchString string `short:"s" long:"search" description:"string to search for" required:"true"`
}

func main() {

	_, err := flags.Parse(&opts)
	if err != nil {
		panic(fmt.Errorf("failed to parse flags: %v", err))
	}

	exifData, err := photoMetadata.GetAllExifData(opts.FilePath)
	if err != nil {
		panic(fmt.Errorf("failed to get exif data: %v", err))
	}
	ds := make(map[string]string)
	for _, f := range exifData {
		for k, v := range f.Fields {
			// fmt.Printf("key: %s\n", k)
			if strings.Contains(strings.ToLower(k), strings.ToLower(opts.SearchString)) {
				s, ok := v.(string)
				if !ok {
					continue
				}
				ds[k] = s
			}
		}
	}

	for k, v := range ds {
		fmt.Printf("%s:%s\n", k, v)
	}
}
