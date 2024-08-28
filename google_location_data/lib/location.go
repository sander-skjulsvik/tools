package lib

import (
	"sort"
	"time"
)

type SourceLocations struct {
	Locations []LocationRecord `json:"locations"`
}

// This function assumes Locations is sorted by time
func (sourceData *SourceLocations) FindClosestLocation(time time.Time) (ind int, err error) {
	var (
		locationBeforeInd int
		locationAfterInd  int
	)

	locationAfterInd = sort.Search(len(sourceData.Locations), func(i int) bool {
		return sourceData.Locations[i].Time.After(time)
	})
	// log.Default().Printf("locationAfterInd: %d", locationAfterInd)
	// Handling edge cases, max and min
	if locationAfterInd == len(sourceData.Locations) {
		return locationAfterInd - 1, nil
	}
	if locationAfterInd == 0 {
		return locationAfterInd, nil
	}

	locationBeforeInd = locationAfterInd - 1
	afterTime := sourceData.Locations[locationAfterInd].Time
	beforeTime := sourceData.Locations[locationBeforeInd].Time

	if time.Sub(afterTime).Abs() <= time.Sub(beforeTime).Abs() {
		return locationAfterInd, nil
	} else {
		return locationBeforeInd, nil
	}
}

func (sourceData *SourceLocations) SortByTime() {
	// Sort the data by time
	sort.Sort(ByTime(sourceData.Locations))
	// return sourceData.Locations
}

type ByTime []LocationRecord

func (bt ByTime) Less(i, j int) bool {
	return bt[i].Time.Before(bt[j].Time)
}

func (bt ByTime) Swap(i, j int) {
	bt[i], bt[j] = bt[j], bt[i]
}

func (bt ByTime) Len() int {
	return len(bt)
}

type LocationRecord struct {
	Coordinates Coordinates `json:"coordinates"`
	Time        time.Time   `json:"timestamp"`
}
