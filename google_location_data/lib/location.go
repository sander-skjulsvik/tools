package lib

import (
	"fmt"
	"sort"
	"time"
)

type SourceLocations struct {
	Locations []LocationRecord `json:"locations"`
}

// This function assumes Locations is sorted by time
// other is the ind of the location the search time is between
func (sourceData *SourceLocations) FindClosestLocations(qTime time.Time) (closest, other int) {
	var (
		locationBeforeInd int
		locationAfterInd  int
	)
	// Handling edge cases, max and min
	n :=  len(sourceData.Locations)
	if qTime.Before(sourceData.Locations[0].Time) || qTime.Equal(sourceData.Locations[0].Time) {
		return 0, 0
	}
	if qTime.After(sourceData.Locations[n-1].Time)  || qTime.Equal(sourceData.Locations[n-1].Time) {
		return n-1, n-1
	}

	locationAfterInd = sort.Search(len(sourceData.Locations), func(i int) bool {
		return sourceData.Locations[i].Time.After(qTime)
	})
	// log.Default().Printf("locationAfterInd: %d", locationAfterInd)
	locationBeforeInd = locationAfterInd - 1

	afterTime := sourceData.Locations[locationAfterInd].Time
	beforeTime := sourceData.Locations[locationBeforeInd].Time

	afterTimeDelta := qTime.Sub(afterTime).Abs()
	beforeTimeDelta := qTime.Sub(beforeTime).Abs()

	if afterTimeDelta < beforeTimeDelta {
		return locationAfterInd, locationBeforeInd
	} else {
		return locationBeforeInd, locationAfterInd
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

func (lr *LocationRecord) String() string {
	return fmt.Sprintf(
		"Coordinates: lat: %f, lng: %f, Time: %s",
		lr.Coordinates.Lat(),
		lr.Coordinates.Lng(),
		lr.Time.String(),
	)
}

func (lr *LocationRecord) Equal(other *LocationRecord) bool {
	if lr.Coordinates.Point.Lat() != other.Coordinates.Point.Lat() {
		return false
	}
	if lr.Coordinates.Point.Lng() != other.Coordinates.Point.Lng() {
		return false
	}
	if lr.Time.Equal(other.Time) {
		return false
	}
	return true
}
