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

type GoogleTimelineLocation struct {
	LatitudeE7  int    `json:"latitudeE7"`
	LongitudeE7 int    `json:"longitudeE7"`
	Timestamp   string `json:"timestamp"`
}

type GoogleTimelineTakeout struct {
	Locations []GoogleTimelineLocation `json:"locations"`
}

var (
	ErrUnableToOpenSourceDataFile      = errors.New("unable to open source data file")
	ErrUnableToReadSourceDataFile      = errors.New("unable to read source data file")
	ErrUnableToUnmarshalSourceDataFile = errors.New("unable to unmarshal source data file")
	ErrUnableToCreateCoordinates       = errors.New("unable to create coordinates")
)

func NewSourceLocationsFromGoogleTimeline(path string) (*SourceLocations, error) {
	// Read the file
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, errors.Join(
			ErrUnableToOpenSourceDataFile,
			fmt.Errorf("file: %s,", path),
			err,
		)
	}
	defer jsonFile.Close()
	// Unmarshal the data into a struct
	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		return nil, errors.Join(
			ErrUnableToReadSourceDataFile,
			fmt.Errorf("file: %s", path),
			err,
		)
	}
	takeout := &GoogleTimelineTakeout{}
	if err := json.Unmarshal(bytes, &takeout); err != nil {
		return nil, errors.Join(
			ErrUnableToUnmarshalSourceDataFile,
			fmt.Errorf("file: %s", path),
			err,
		)
	}
	locationStore := takeout.ToLocationRecords()
	// Convert the data into a SourceLocations struct
	return locationStore, nil
}

func (g *GoogleTimelineTakeout) ToLocationRecords() *SourceLocations {
	SourceLocations := SourceLocations{}

	locations := make([]LocationRecord, len(g.Locations))

	for i, loc := range g.Locations {
		c := NewCoordinatesE7(loc.LatitudeE7, loc.LongitudeE7)
		parsedTime, err := g.ParseTime(loc.Timestamp)
		if err != nil {
			fmt.Printf("Error parsing time for record: %v, err: %v\n", loc, err)
			continue
		}
		locations[i] = LocationRecord{
			Coordinates: c,
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
