package lib

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"
	"time"
)

var RFC3339_LAYOUT string = "2006-01-02T15:04:05Z07:00"

type GoogleTimelineLocationStore struct {
	SourceLocations SourceLocationData

	// Time difference thresholds
	// Low time difference threshold
	LowTimeDiffThreshold time.Duration
	// Medium time difference threshold
	MediumTimeDiffThreshold time.Duration
	// High time difference threshold
	HighTimeDiffThreshold time.Duration
}

const (
	LOW_TIME_DIFF_THRESHOLD    = 30 * time.Minute
	MEDIUM_TIME_DIFF_THRESHOLD = 2 * time.Hour
	HIGH_TIME_DIFF_THRESHOLD   = 12 * time.Hour
)

func NewGoogleTimelineLocationStoreFromFile(path string) *GoogleTimelineLocationStore {
	return &GoogleTimelineLocationStore{
		SourceLocations:         ImportSourceLocationData(path),
		LowTimeDiffThreshold:    LOW_TIME_DIFF_THRESHOLD,
		MediumTimeDiffThreshold: MEDIUM_TIME_DIFF_THRESHOLD,
		HighTimeDiffThreshold:   HIGH_TIME_DIFF_THRESHOLD,
	}
}

type SourceLocationData struct {
	Locations []SourceLocationRecord `json:"locations"`
}

// This function assumes Locations is sorted by time
func (sourceData *SourceLocationData) GetLocation(time time.Time) (before SourceLocationRecord, after SourceLocationRecord, err error) {
	locationBeforeInd := sort.Search(len(sourceData.Locations), func(i int) bool {
		return sourceData.Locations[i].GetTime().After(time)
	})

	locationAfterInd := locationBeforeInd + 1

	return sourceData.Locations[locationBeforeInd], sourceData.Locations[locationAfterInd], nil
}

func (sourceData *SourceLocationData) SortByTime() {
	// Sort the data by time
	sort.Sort(ByTime(sourceData.Locations))
	// return sourceData.Locations
}

type ByTime []SourceLocationRecord

func (bt ByTime) Less(i, j int) bool {
	return bt[i].GetTime().Before(bt[j].GetTime())
}

func (bt ByTime) Swap(i, j int) {
	bt[i], bt[j] = bt[j], bt[i]
}

func (bt ByTime) Len() int {
	return len(bt)
}

func (bt ByTime) Comp(i, j int) int {
	return bt[i].GetTime().Compare(bt[j].GetTime())
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
	return sourceData
}

func (locStore *SourceLocationRecord) GetTime() time.Time {
	return ParseTime(locStore.TimeStamp)
}

func ParseTime(timeStr string) time.Time {
	t, err := time.Parse(RFC3339_LAYOUT, timeStr)
	if err != nil {
		log.Fatalf("Unable to parse timestamp: %v", err)
	}
	return t
}

type Location struct {
	LatitudeE7  float64
	LongitudeE7 float64
}

/*
This function will have to do some assumtions when the time between location stamps is too large.

It is implemented with 3 types of assumptions:

- If the time between the photo and the location is low we will assume it is accurate.
- If the time between the photo and the location is medium we will assume we will attempt a linear interpolation between the two locations.
- If the time is large we will return an error, and assume the user will have to provide the data themselves.
*/
// func (locStore *GoogleTimelineLocationStore) GetLocationByTime(time time.Time) (Location, error) {
// 	// Find the closest location to the given time

// 	// Return the location
// 	// return Location{
// 	// 	LatitudeE7:  closestLocation.LatitudeE7,
// 	// 	LongitudeE7: closestLocation.LongitudeE7,
// 	// }
// }

// func (locStore *GoogleTimelineLocationStore) GetLocationByTimeBefore(time time.Time) (Location, error) {
// 	for _, record := range locStore.SourceLocations.Locations {
// 		time.Parse(RFC3339_LAYOUT, record.TimeStamp)
// 	}
// }

// func (locStore *GoogleTimelineLocationStore) GetLocationByTimeAfter(time time.Time) (Location, error) {
// }
