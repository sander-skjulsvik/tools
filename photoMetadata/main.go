package main

import (
	"fmt"
	"os"

	"github.com/sander-skjulsvik/tools/photoMetadata/lib"
)

func main() {
	method := os.Args[1]
	photoPath := os.Args[2]

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
	default:
		fmt.Println("Invalid method")
	}
}

// func ApplyLocationData(locationRecord locationData.LocationRecord, photo )

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
