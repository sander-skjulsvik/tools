package main

import (
	"fmt"
	"slices"
)

// Source turns one kind of location data into points. Points works offline,
// from what the source saved in the data folder, and returns only points that
// pass the source's config.
//
// Getting the data into the data folder is up to each source, since they differ:
// Immich has an API to sync from, while Google Timeline is an exported file.
type Source interface {
	Name() string
	Points() ([]Point, error)
}

// Sources returns the sources the config lists.
func Sources(dataDir string, config Config) []Source {
	var sources []Source
	if config.Immich != nil {
		sources = append(sources, immichSource{dataDir: dataDir, config: config.Immich})
	}
	return sources
}

// LoadPoints returns the points of every configured source, in time order.
func LoadPoints(dataDir string, config Config) ([]Point, error) {
	return mergePoints(Sources(dataDir, config))
}

func mergePoints(sources []Source) ([]Point, error) {
	points := []Point{}
	for _, source := range sources {
		found, err := source.Points()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source.Name(), err)
		}
		points = append(points, found...)
	}
	slices.SortStableFunc(points, func(a, b Point) int { return a.Time.Compare(b.Time) })
	return points, nil
}
