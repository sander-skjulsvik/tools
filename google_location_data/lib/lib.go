package lib

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

type SourceLocationData struct {
	Locations []SourceLocationRecord `json:"locations"`
}

type SourceLocationRecord struct {
	LatitudeE7  float64 `json:"latitudeE7"`
	LongitudeE7 float64 `json:"longitudeE7"`
	TimeStamp   string  `json:"timestamp"`
}

func ImportSourceLocationData(path string) SourceLocationData {
	// Read the data from the file
	jsonFile, err := os.Open(path)
	if err != nil {
		log.Fatalf("Unable to open source data file: %v", err)
	}
	defer jsonFile.Close()
	// Unmarshal the data into a struct
	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Unable to read source data file: %v", err)
	}

	sourceData := SourceLocationData{}
	err = json.Unmarshal(bytes, &sourceData)
	if err != nil {
		log.Fatalf("Unable to unmarshal source data: %v", err)
	}

	// Return the list of records
	return sourceData2
}
