package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const schemaVersion = 1

// droppedAssetFields are parts of Immich's response not worth keeping.
var droppedAssetFields = []string{"owner", "ownerId", "thumbhash", "resized", "stack"}

// Face is a face box in Immich's coordinates.
type Face struct {
	BoundingBoxX1 int `json:"boundingBoxX1"`
	BoundingBoxY1 int `json:"boundingBoxY1"`
	BoundingBoxX2 int `json:"boundingBoxX2"`
	BoundingBoxY2 int `json:"boundingBoxY2"`
	ImageWidth    int `json:"imageWidth"`
	ImageHeight   int `json:"imageHeight"`
}

// Asset is the part of Immich's response this tool reads. The full response is
// kept in Record.Immich.
type Asset struct {
	ID               string `json:"id"`
	Checksum         string `json:"checksum"`
	DuplicateID      string `json:"duplicateId"`
	OriginalFileName string `json:"originalFileName"`
	OriginalPath     string `json:"originalPath"`
	LocalDateTime    string `json:"localDateTime"`
	Visibility       string `json:"visibility"`
	IsTrashed        bool   `json:"isTrashed"`
	IsOffline        bool   `json:"isOffline"`
	ExifInfo         struct {
		Make string `json:"make"`
	} `json:"exifInfo"`
	People []struct {
		ID    string `json:"id"`
		Faces []Face `json:"faces"`
	} `json:"people"`
	UnassignedFaces []Face `json:"unassignedFaces"`
}

// Camera returns the camera's maker, which separates phones from dedicated cameras.
func (a Asset) Camera() string {
	if a.ExifInfo.Make == "" {
		return "unknown"
	}
	return a.ExifInfo.Make
}

// LocalMetrics are computed from the preview and Immich's face boxes.
type LocalMetrics struct {
	Version               int     `json:"version"`
	DHash                 string  `json:"dhash"`
	FaceFound             bool    `json:"face_found"`
	OtherFaceCount        int     `json:"other_face_count"`
	FaceArea              float64 `json:"face_area"`
	LargestOtherFaceRatio float64 `json:"largest_other_face_ratio"`
	FaceSharpness         float64 `json:"face_sharpness"`
	AnalyzedAt            string  `json:"analyzed_at"`
}

// Label is a judgement of a photo from an LLM or a person.
type Label struct {
	Model         string   `json:"model"`
	PromptVersion string   `json:"prompt_version"`
	Keep          bool     `json:"keep"`
	Category      string   `json:"category"`
	Description   string   `json:"description"`
	Issues        []string `json:"issues"`
	// EditNeeded marks a photo that is a good fit only after editing. EditNotes
	// says what the edit is, and is required when EditNeeded is set.
	EditNeeded bool     `json:"edit_needed"`
	EditNotes  string   `json:"edit_notes,omitempty"`
	Score      *float64 `json:"score,omitempty"`
	Notes      string   `json:"notes,omitempty"`
	LabelledAt string   `json:"labelled_at"`
}

// Record is everything known about one photo, stored as assets/<id>.json.
type Record struct {
	SchemaVersion int             `json:"schema_version"`
	AssetID       string          `json:"asset_id"`
	Checksum      string          `json:"checksum"`
	PreviewFile   string          `json:"preview_file"`
	Immich        json.RawMessage `json:"immich"`
	Queries       []string        `json:"queries"`
	Local         *LocalMetrics   `json:"local"`
	LLM           *Label          `json:"llm"`
}

// Asset decodes the fields of the stored Immich response that this tool uses.
func (r *Record) Asset() (Asset, error) {
	var asset Asset
	err := json.Unmarshal(r.Immich, &asset)
	return asset, err
}

// Run describes the last collect.
type Run struct {
	SchemaVersion int      `json:"schema_version"`
	CollectedAt   string   `json:"collected_at"`
	ImmichURL     string   `json:"immich_url"`
	PersonID      string   `json:"person_id"`
	PersonName    string   `json:"person_name"`
	Since         string   `json:"since"`
	Queries       []string `json:"queries"`
	AssetIDs      []string `json:"asset_ids"`
}

// Catalog is the data folder: one JSON file per photo, so a crash loses nothing
// and every result can be read directly.
type Catalog struct {
	root, assets, previews, shortlist, sheets string
}

// OpenCatalog creates the folder if needed, readable only by this user.
func OpenCatalog(root string) (*Catalog, error) {
	c := &Catalog{
		root:      root,
		assets:    filepath.Join(root, "assets"),
		previews:  filepath.Join(root, "previews"),
		shortlist: filepath.Join(root, "shortlist"),
		sheets:    filepath.Join(root, "sheets"),
	}
	for _, dir := range []string{c.root, c.assets, c.previews, c.shortlist, c.sheets} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
	}
	return c, os.Chmod(root, 0o700)
}

func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func writeJSON(path string, value any) error {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return err
	}
	return writeFileAtomic(path, buffer.Bytes())
}

// Load returns a photo's record, or nil if there is none.
func (c *Catalog) Load(assetID string) (*Record, error) {
	data, err := os.ReadFile(filepath.Join(c.assets, assetID+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("decoding record %s: %w", assetID, err)
	}
	return &record, nil
}

// Save writes a photo's record.
func (c *Catalog) Save(record *Record) error {
	return writeJSON(filepath.Join(c.assets, record.AssetID+".json"), record)
}

// LoadRun returns the last collect.
func (c *Catalog) LoadRun() (*Run, error) {
	data, err := os.ReadFile(filepath.Join(c.root, "run.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no catalog in %s: run collect first", c.root)
	}
	if err != nil {
		return nil, err
	}
	var run Run
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("decoding run.json: %w", err)
	}
	return &run, nil
}

// RunRecords returns the records of the photos in the last collect.
func (c *Catalog) RunRecords() ([]*Record, error) {
	run, err := c.LoadRun()
	if err != nil {
		return nil, err
	}
	records := make([]*Record, 0, len(run.AssetIDs))
	for _, id := range run.AssetIDs {
		record, err := c.Load(id)
		if err != nil {
			return nil, err
		}
		if record != nil {
			records = append(records, record)
		}
	}
	return records, nil
}

// PreviewPath returns the path of a record's saved preview, or "" if there is none.
func (c *Catalog) PreviewPath(record *Record) string {
	if record.PreviewFile == "" {
		return ""
	}
	path := filepath.Join(c.previews, record.PreviewFile)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// clearFiles removes the files directly inside a folder this tool generates.
func clearFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// trimAsset removes droppedAssetFields from a raw Immich asset.
func trimAsset(raw json.RawMessage) (json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	for _, field := range droppedAssetFields {
		delete(fields, field)
	}
	return json.Marshal(fields)
}

// mergeRecord refreshes Immich's data for a photo, keeping earlier results
// unless the image itself changed.
func (c *Catalog) mergeRecord(raw json.RawMessage) (*Record, error) {
	var asset Asset
	if err := json.Unmarshal(raw, &asset); err != nil {
		return nil, fmt.Errorf("decoding asset: %w", err)
	}
	if asset.ID == "" || strings.ContainsAny(asset.ID, `/\`) {
		return nil, fmt.Errorf("unexpected asset ID %q", asset.ID)
	}
	trimmed, err := trimAsset(raw)
	if err != nil {
		return nil, err
	}
	record, err := c.Load(asset.ID)
	if err != nil {
		return nil, err
	}
	if record != nil && record.Checksum != asset.Checksum {
		if path := c.PreviewPath(record); path != "" {
			if err := os.Remove(path); err != nil {
				return nil, err
			}
		}
		record = nil
	}
	if record == nil {
		record = &Record{SchemaVersion: schemaVersion, AssetID: asset.ID, Checksum: asset.Checksum}
	}
	record.Immich = trimmed
	record.Queries = []string{}
	return record, nil
}

// CollectOptions are the inputs to Collect.
type CollectOptions struct {
	Person     string
	Since      time.Time
	Queries    []string
	QueryLimit int
}

// Collect saves metadata for the person's photos and which queries they match.
func Collect(client *Client, catalog *Catalog, options CollectOptions) error {
	personID, personName, err := client.ResolvePerson(options.Person)
	if err != nil {
		return err
	}
	assets, err := client.SearchPersonAssets(personID, options.Since)
	if err != nil {
		return err
	}
	records := make(map[string]*Record, len(assets))
	ids := make([]string, 0, len(assets))
	for _, raw := range assets {
		record, err := catalog.mergeRecord(raw)
		if err != nil {
			return err
		}
		if _, seen := records[record.AssetID]; !seen {
			ids = append(ids, record.AssetID)
		}
		records[record.AssetID] = record
	}
	slices.Sort(ids)

	for _, query := range options.Queries {
		matches, err := client.SearchQuery(personID, options.Since, query, options.QueryLimit)
		if err != nil {
			return err
		}
		for _, id := range matches {
			if record, ok := records[id]; ok {
				record.Queries = append(record.Queries, query)
			}
		}
		fmt.Printf("  query %q: %d matches\n", query, len(matches))
	}

	for _, id := range ids {
		if err := catalog.Save(records[id]); err != nil {
			return err
		}
	}
	queries := options.Queries
	if queries == nil {
		queries = []string{}
	}
	err = writeJSON(filepath.Join(catalog.root, "run.json"), Run{
		SchemaVersion: schemaVersion,
		CollectedAt:   nowUTC(),
		ImmichURL:     client.BaseURL(),
		PersonID:      personID,
		PersonName:    personName,
		Since:         options.Since.Format("2006-01-02"),
		Queries:       queries,
		AssetIDs:      ids,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Collected %d photos of %s since %s into %s\n",
		len(ids), personName, options.Since.Format("2006-01-02"), catalog.root)
	return nil
}
