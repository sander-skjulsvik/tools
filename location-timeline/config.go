package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const configFile = "config.json"

const exampleConfig = `{
  "cameras": [
    {"make": "Apple", "model": "iPhone 15 Pro"},
    {"make": "FUJIFILM"}
  ],
  "people": ["Your Name"]
}`

// Camera is one of your own cameras. An empty model matches every model of the make.
type Camera struct {
	Make  string `json:"make"`
	Model string `json:"model,omitempty"`
}

// Config decides which photos show where you were: those taken with one of your
// cameras, and those you are in. Other photos, such as forwarded ones, are ignored.
type Config struct {
	Cameras []Camera `json:"cameras"`
	// People are names or Immich IDs of you; one person usually.
	People []string `json:"people"`
}

// LoadConfig reads config.json from the data folder.
func LoadConfig(dataDir string) (Config, error) {
	path := filepath.Join(dataDir, configFile)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("no %s; create it like this, using the cameras command to list yours:\n%s", path, exampleConfig)
	}
	if err != nil {
		return Config{}, err
	}
	var config Config
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decoding %s: %w", path, err)
	}
	if len(config.Cameras) == 0 && len(config.People) == 0 {
		return Config{}, fmt.Errorf("%s lists no cameras and no people, so no photo counts", path)
	}
	for _, camera := range config.Cameras {
		if strings.TrimSpace(camera.Make) == "" {
			return Config{}, fmt.Errorf("%s: every camera needs a make", path)
		}
	}
	return config, nil
}

func same(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// OwnCamera reports whether the photo was taken with one of the configured cameras.
func (c Config) OwnCamera(make, model string) bool {
	return slices.ContainsFunc(c.Cameras, func(camera Camera) bool {
		return same(camera.Make, make) && (camera.Model == "" || same(camera.Model, model))
	})
}

// Includes reports whether one of the people is a configured person.
func (c Config) Includes(people []PersonRef) bool {
	for _, person := range people {
		for _, wanted := range c.People {
			if same(wanted, person.ID) || (person.Name != "" && same(wanted, person.Name)) {
				return true
			}
		}
	}
	return false
}

// checkPeople refuses a configured name shared by several people in Immich,
// since a namesake's photos would count as yours.
func (c Config) checkPeople(photos []Photo) error {
	for _, wanted := range c.People {
		ids := map[string]bool{}
		for _, photo := range photos {
			for _, person := range photo.People {
				if person.Name != "" && same(wanted, person.Name) {
					ids[person.ID] = true
				}
			}
		}
		if len(ids) > 1 {
			return fmt.Errorf("%d people in Immich are named %q; put the right ID in %s instead: %s",
				len(ids), wanted, configFile, strings.Join(slices.Sorted(maps.Keys(ids)), ", "))
		}
	}
	return nil
}
