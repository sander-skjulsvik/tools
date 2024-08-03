package main

import (
	"fmt"
	"os"

	"github.com/sander-skjulsvik/tools/photoMetadata/lib"
)

func main() {
	method := os.Args[1]
	photoPath := os.Args[2]

	switch method {
	case "list":
		List(photoPath)
	case "write":
		Write(photoPath)
		// List(photoPath)
		pos := Search(photoPath, "GPSPosition")
		fmt.Println(pos)
	case "search":
		pos := Search(photoPath, "GPSPosition")
		fmt.Println(pos)
	default:
		fmt.Println("Invalid method")
	}
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

func Search(path string, search string) interface{} {
	fmt.Println("Searching for data")

	fileInfos, err := lib.GetAllExifData(path)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return nil
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.Err != nil {
			fmt.Printf("Error concerning %v: %v\n", fileInfo.File, fileInfo.Err)
			continue
		}

		for k, v := range fileInfo.Fields {
			if k == search {
				return v
			}
		}
	}
	return nil
}
