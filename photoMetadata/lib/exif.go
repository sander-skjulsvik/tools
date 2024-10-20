package lib

import (
	"fmt"

	"github.com/barasher/go-exiftool"
)

func GetAllExifData(filePath string) ([]exiftool.FileMetadata, error) {
	et, err := exiftool.NewExiftool()
	if err != nil {
		return nil, fmt.Errorf("error when initializing: %v", err)
	}
	defer et.Close()
	return et.ExtractMetadata(filePath), nil
}

func PrintAllExifData(fileInfos []exiftool.FileMetadata) error {
	for _, fileInfo := range fileInfos {
		if fileInfo.Err != nil {
			fmt.Printf("Error concerning %v: %v\n", fileInfo.File, fileInfo.Err)
			continue
		}

		for k, v := range fileInfo.Fields {
			fmt.Printf("[%v] %v\n", k, v)
		}
	}
	return nil
}

func WriteExifDataToFile(key, value, filePath string) error {
	et, err := exiftool.NewExiftool()
	if err != nil {
		return fmt.Errorf("error when initializing: %v", err)
	}
	defer et.Close()
	currentData := et.ExtractMetadata(filePath)

	currentData[0].SetString(key, value)

	et.WriteMetadata(currentData)
	for _, d := range currentData {
		if d.Err != nil {
			return fmt.Errorf("error concerning %v: %v", d.File, d.Err)
		}
	}

	return nil
}
