package main

import (
	"cmp"
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

// immichSource reads the photos saved by SyncImmich.
type immichSource struct {
	dataDir string
	config  *ImmichConfig
}

func (s immichSource) Name() string { return "immich" }

func (s immichSource) Points() ([]Point, error) {
	data, err := LoadImmich(s.dataDir)
	if err != nil {
		return nil, err
	}
	if err := s.config.checkPeople(data.Photos); err != nil {
		return nil, err
	}
	return immichPoints(data.Photos, s.config), nil
}

func immichPoints(photos []Photo, config *ImmichConfig) []Point {
	points := []Point{}
	for _, photo := range photos {
		var evidence []string
		if config.OwnCamera(photo.Make, photo.Model) {
			evidence = append(evidence, "camera")
		}
		if config.Includes(photo.People) {
			evidence = append(evidence, "person")
		}
		if len(evidence) == 0 {
			continue
		}
		points = append(points, Point{
			Source: "immich", ID: photo.ID, Time: photo.Taken, LocalTime: photo.LocalTime,
			Latitude: photo.Latitude, Longitude: photo.Longitude,
			City: photo.City, State: photo.State, Country: photo.Country,
			Evidence: evidence,
		})
	}
	slices.SortStableFunc(points, func(a, b Point) int { return a.Time.Compare(b.Time) })
	return points
}

// CameraCount is how many geotagged photos one camera took.
type CameraCount struct {
	Make   string `json:"make"`
	Model  string `json:"model"`
	Photos int    `json:"photos"`
	First  string `json:"first"`
	Last   string `json:"last"`
	Yours  bool   `json:"yours"`
}

// Cameras lists the cameras of geotagged photos, most photos first. The config
// may be nil, since listing cameras is how it gets written.
func Cameras(photos []Photo, config *ImmichConfig) []CameraCount {
	byCamera := map[[2]string]*CameraCount{}
	for _, photo := range photos {
		key := [2]string{photo.Make, photo.Model}
		date := photo.LocalTime[:len(dateLayout)]
		count, ok := byCamera[key]
		if !ok {
			count = &CameraCount{Make: photo.Make, Model: photo.Model, First: date, Last: date,
				Yours: config.OwnCamera(photo.Make, photo.Model)}
			byCamera[key] = count
		}
		count.Photos++
		count.First, count.Last = min(count.First, date), max(count.Last, date)
	}
	cameras := []CameraCount{}
	for _, count := range byCamera {
		cameras = append(cameras, *count)
	}
	slices.SortFunc(cameras, func(a, b CameraCount) int {
		return cmp.Or(cmp.Compare(b.Photos, a.Photos), cmp.Compare(a.Make, b.Make), cmp.Compare(a.Model, b.Model))
	})
	return cameras
}
