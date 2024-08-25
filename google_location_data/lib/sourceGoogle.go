package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	toolsTime "github.com/sander-skjulsvik/tools/libs/time"
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

var ErrUnableToCreateCoordinates = fmt.Errorf("Unable to create coordinates")

func (g *GoogleTimelineTakeout) ToLocationRecords() *SourceLocations {
	SourceLocations := SourceLocations{}

	locations := make([]LocationRecord, len(g.Locations))

	for i, loc := range g.Locations {
		c := NewCorrdinatesE7(loc.LatitudeE7, loc.LongitudeE7)
		parsedTime, err := g.ParseTime(loc.Timestamp)
		if err != nil {
			fmt.Printf("Error parsing time for record: %v, err: %v\n", loc, err)
			continue
		}
		locations[i] = LocationRecord{
			Corrdinates: c,
			Time:        *parsedTime,
		}
	}

	SourceLocations.Locations = locations
	return &SourceLocations
}

func (g GoogleTimelineTakeout) ParseTime(timeStr string) (*time.Time, error) {
	googleTimelineTimeLayout := toolsTime.RFC3339
	t, err := time.Parse(googleTimelineTimeLayout, timeStr)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
