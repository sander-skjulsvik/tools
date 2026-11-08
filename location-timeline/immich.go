package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/sander-skjulsvik/tools/libs/immich"
)

// allowedRequests is every request this tool can make to Immich: it only reads
// asset metadata. Search is a POST because Immich implements search as POST.
var allowedRequests = []immich.Rule{
	immich.Allow(http.MethodPost, `^/api/search/metadata$`),
}

const immichFile = "immich.json"

// Photo is what this tool keeps of one geotagged Immich asset.
type Photo struct {
	ID string `json:"id"`
	// Taken is when the photo was taken, in UTC.
	Taken time.Time `json:"taken"`
	// LocalTime is the wall clock where it was taken, as 2006-01-02T15:04:05.
	LocalTime string      `json:"local_time"`
	TimeZone  string      `json:"time_zone,omitempty"`
	Latitude  float64     `json:"latitude"`
	Longitude float64     `json:"longitude"`
	City      string      `json:"city,omitempty"`
	State     string      `json:"state,omitempty"`
	Country   string      `json:"country,omitempty"`
	Make      string      `json:"make,omitempty"`
	Model     string      `json:"model,omitempty"`
	People    []PersonRef `json:"people,omitempty"`
}

// PersonRef is a person Immich recognised in a photo.
type PersonRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ImmichData is the immich.json file in the data folder.
type ImmichData struct {
	SchemaVersion int    `json:"schema_version"`
	SyncedAt      string `json:"synced_at"`
	Server        string `json:"server"`
	// Assets counts every asset seen, with or without a location.
	Assets int     `json:"assets"`
	Photos []Photo `json:"photos"`
}

type immichAsset struct {
	ID            string `json:"id"`
	FileCreatedAt string `json:"fileCreatedAt"`
	LocalDateTime string `json:"localDateTime"`
	ExifInfo      struct {
		DateTimeOriginal string   `json:"dateTimeOriginal"`
		TimeZone         string   `json:"timeZone"`
		Latitude         *float64 `json:"latitude"`
		Longitude        *float64 `json:"longitude"`
		City             string   `json:"city"`
		State            string   `json:"state"`
		Country          string   `json:"country"`
		Make             string   `json:"make"`
		Model            string   `json:"model"`
	} `json:"exifInfo"`
	People []PersonRef `json:"people"`
}

// photoFromAsset returns false for assets without a usable location or time.
func photoFromAsset(raw json.RawMessage) (Photo, bool, error) {
	var asset immichAsset
	if err := json.Unmarshal(raw, &asset); err != nil {
		return Photo{}, false, fmt.Errorf("decoding asset: %w", err)
	}
	exif := asset.ExifInfo
	if exif.Latitude == nil || exif.Longitude == nil || (*exif.Latitude == 0 && *exif.Longitude == 0) {
		return Photo{}, false, nil
	}
	taken := exif.DateTimeOriginal
	if taken == "" {
		taken = asset.FileCreatedAt
	}
	when, err := time.Parse(time.RFC3339Nano, taken)
	// Immich stores the local wall clock as if it were UTC.
	if err != nil || len(asset.LocalDateTime) < len("2006-01-02T15:04:05") {
		return Photo{}, false, nil
	}
	people := []PersonRef{}
	for _, person := range asset.People {
		people = append(people, PersonRef{ID: person.ID, Name: person.Name})
	}
	return Photo{
		ID:        asset.ID,
		Taken:     when.UTC(),
		LocalTime: asset.LocalDateTime[:len("2006-01-02T15:04:05")],
		TimeZone:  exif.TimeZone,
		Latitude:  *exif.Latitude,
		Longitude: *exif.Longitude,
		City:      exif.City,
		State:     exif.State,
		Country:   exif.Country,
		Make:      exif.Make,
		Model:     exif.Model,
		People:    people,
	}, true, nil
}

// SyncImmich replaces immich.json with every geotagged asset in the library.
// It keeps all of them, so changing the config never needs another sync.
func SyncImmich(client *immich.Client, dataDir string) error {
	data := ImmichData{SchemaVersion: 1, Server: client.BaseURL(), Photos: []Photo{}}
	err := client.SearchMetadata(map[string]any{
		"withExif":   true,
		"withPeople": true,
		"order":      "asc",
	}, func(items []json.RawMessage) error {
		for _, item := range items {
			data.Assets++
			photo, ok, err := photoFromAsset(item)
			if err != nil {
				return err
			}
			if ok {
				data.Photos = append(data.Photos, photo)
			}
		}
		fmt.Fprintf(os.Stderr, "\r  %d assets, %d with a location", data.Assets, len(data.Photos))
		return nil
	})
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return err
	}
	slices.SortStableFunc(data.Photos, func(a, b Photo) int { return a.Taken.Compare(b.Taken) })
	data.SyncedAt = time.Now().UTC().Format(time.RFC3339)
	if err := writeJSON(filepath.Join(dataDir, immichFile), data); err != nil {
		return err
	}
	fmt.Printf("Saved %d geotagged of %d assets to %s\n", len(data.Photos), data.Assets, filepath.Join(dataDir, immichFile))
	return nil
}

// LoadImmich reads immich.json.
func LoadImmich(dataDir string) (*ImmichData, error) {
	path := filepath.Join(dataDir, immichFile)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no %s: run sync first", path)
	}
	if err != nil {
		return nil, err
	}
	var data ImmichData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	return &data, nil
}
