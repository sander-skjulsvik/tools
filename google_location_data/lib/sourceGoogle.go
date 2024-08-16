package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type GoogleTimelineTakeout struct {
	Locations []GoogleTimelineLocations `json:"locations"`
}

type GoogleTimelineLocations struct {
	LatitudeE7  int    `json:"latitudeE7"`
	LongitudeE7 int    `json:"longitudeE7"`
	Timestamp   string `json:"timestampMs"`
}

var (
	ErrUnableToOpenSourceDataFile      = errors.New("Unable to open source data file")
	ErrUnableToReadSourceDataFile      = errors.New("Unable to read source data file")
	ErrUnableToUnmarshalSourceDataFile = errors.New("Unable to unmarshal source data file")
)

func NewGoogleTimelineLocationsFromFile(path string) (*GoogleTimelineLocations, error) {
	// Read the file
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnableToOpenSourceDataFile, err)
	}
	defer jsonFile.Close()
	// Unmarshal the data into a struct
	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnableToReadSourceDataFile, err)
	}
	gogleTimeLineLocations := &GoogleTimelineLocations{}
	if err := json.Unmarshal(bytes, &GoogleTimelineLocations{}); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnableToUnmarshalSourceDataFile, err)
	}
	// Convert the data into a SourceLocations struct
	return gogleTimeLineLocations, nil
}

func (g *GoogleTimelineTakeout) ToLocationRecords() (*SourceLocations, error) {
	SourceLocations := SourceLocations{}

	locations := make([]LocationRecord, len(g.Locations))

	for i, loc := range g.Locations {
		c, err := NewCorrdinatesE7(loc.LatitudeE7, loc.LongitudeE7)
		if err != nil {
			return nil, fmt.Errorf("Unable to create coordinates: %v", err)
		}

		locations[i] = LocationRecord{
			Corrdinates: c,
			Time:        ParseTime(loc.Timestamp),
		}
	}

	SourceLocations.Locations = locations
	return &SourceLocations, nil
}
