package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"math/bits"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	nearDuplicatePrefix   = "near-duplicate of "
	immichDuplicatePrefix = "Immich duplicate of "
	burstPrefix           = "same burst as "
	overOutingCap         = "over the limit for its outing"
	// excludeCategory is the label category for photos that must never be
	// shortlisted again, such as identity documents.
	excludeCategory = "exclude"
)

// ShortlistOptions are the filters. They only use saved results, so they can be
// changed and re-run without any network access.
type ShortlistOptions struct {
	Limit       int
	MinFaceArea float64
	// FaceAreaEnough is the face size that earns full credit. Larger faces earn no
	// more, so that selfies don't win just by being close to the camera.
	FaceAreaEnough       float64
	MaxOtherFace         float64
	DropBlurriestPercent float64
	// DuplicateBits is the most differing hash bits to still count as a
	// duplicate. Negative turns duplicate removal off.
	DuplicateBits int
	// BurstWindow keeps one photo from any moment this short. Zero turns it off.
	BurstWindow time.Duration
	// MaxPerOuting caps the photos kept from one outing, so that a single
	// well-photographed day can't dominate. Zero turns the cap off.
	MaxPerOuting int
	// OutingGap is the time between photos that starts a new outing.
	OutingGap time.Duration
}

// ShortlistEntry is one chosen photo.
type ShortlistEntry struct {
	Rank         int      `json:"rank"`
	AssetID      string   `json:"asset_id"`
	Score        float64  `json:"score"`
	File         string   `json:"file"`
	OriginalPath string   `json:"original_path"`
	Taken        string   `json:"taken"`
	Camera       string   `json:"camera"`
	Outing       int      `json:"outing"`
	Queries      []string `json:"queries"`
	PreviewFile  string   `json:"preview_file"`
	Labelled     bool     `json:"labelled"`
}

// Shortlist is the content of shortlist.json.
type Shortlist struct {
	CreatedAt string           `json:"created_at"`
	Photos    []ShortlistEntry `json:"photos"`
}

func rejectReason(record *Record, asset Asset, options ShortlistOptions) string {
	switch {
	case record.LLM != nil && record.LLM.Category == excludeCategory:
		return "excluded by its label"
	case asset.IsTrashed || asset.IsOffline:
		return "trashed or offline"
	case asset.Visibility != "" && asset.Visibility != "timeline":
		return "visibility is " + asset.Visibility
	case record.Local == nil:
		return "not analyzed"
	case !record.Local.FaceFound:
		return "no face box for this person"
	case record.Local.FaceArea < options.MinFaceArea:
		return "face too small"
	case record.Local.LargestOtherFaceRatio > options.MaxOtherFace:
		return "another prominent face"
	}
	return ""
}

// percentileRanks maps each ID to the share of values below it, from 0 to 1. A
// lone photo has nothing to lose against, so it ranks 1.
func percentileRanks(ids []string, value func(string) float64) map[string]float64 {
	ordered := slices.Clone(ids)
	slices.SortStableFunc(ordered, func(a, b string) int { return cmp.Compare(value(a), value(b)) })
	ranks := make(map[string]float64, len(ordered))
	for index, id := range ordered {
		ranks[id] = 1
		if len(ordered) > 1 {
			ranks[id] = float64(index) / float64(len(ordered)-1)
		}
	}
	return ranks
}

// themeOrder orders photos by taking turns between their search themes, then
// adds the rest, keeping the given order within each theme.
func themeOrder(ids []string, queries func(string) []string) []string {
	var themes []string
	groups := map[string][]string{}
	for _, id := range ids {
		for _, query := range queries(id) {
			if _, known := groups[query]; !known {
				themes = append(themes, query)
			}
			groups[query] = append(groups[query], id)
		}
	}
	ordered := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for progress := true; progress; {
		progress = false
		for _, theme := range themes {
			for len(groups[theme]) > 0 && seen[groups[theme][0]] {
				groups[theme] = groups[theme][1:]
			}
			if len(groups[theme]) > 0 {
				ordered = append(ordered, groups[theme][0])
				seen[groups[theme][0]] = true
				groups[theme] = groups[theme][1:]
				progress = true
			}
		}
	}
	for _, id := range ids {
		if !seen[id] {
			ordered = append(ordered, id)
		}
	}
	return ordered
}

// outingNumbers splits photos into outings by the time between them. A photo with
// no usable time gets an outing of its own.
func outingNumbers(assets []Asset, gap time.Duration) map[string]int {
	type stamped struct {
		id string
		at time.Time
	}
	numbers := make(map[string]int, len(assets))
	next := 0
	var dated []stamped
	for _, asset := range assets {
		at, err := time.Parse(time.RFC3339, asset.LocalDateTime)
		if err != nil {
			next++
			numbers[asset.ID] = next
			continue
		}
		dated = append(dated, stamped{asset.ID, at})
	}
	slices.SortStableFunc(dated, func(a, b stamped) int { return a.at.Compare(b.at) })
	for index, photo := range dated {
		if index == 0 || photo.at.Sub(dated[index-1].at) > gap {
			next++
		}
		numbers[photo.id] = next
	}
	return numbers
}

// BuildShortlist filters and ranks records, and returns why the others were left out.
func BuildShortlist(records []*Record, options ShortlistOptions) ([]ShortlistEntry, map[string]string, error) {
	rejected := map[string]string{}
	candidates := map[string]*Record{}
	assets := map[string]Asset{}
	allAssets := make([]Asset, 0, len(records))
	var ids []string
	for _, record := range records {
		asset, err := record.Asset()
		if err != nil {
			return nil, nil, fmt.Errorf("decoding %s: %w", record.AssetID, err)
		}
		allAssets = append(allAssets, asset)
		if reason := rejectReason(record, asset, options); reason != "" {
			rejected[record.AssetID] = reason
			continue
		}
		candidates[record.AssetID] = record
		assets[record.AssetID] = asset
		ids = append(ids, record.AssetID)
	}

	// Rank sharpness against photos from the same camera. Sharpness has no universal
	// scale, and across cameras it mostly measures how close a phone selfie is.
	cameraIDs := map[string][]string{}
	for _, id := range ids {
		camera := assets[id].Camera()
		cameraIDs[camera] = append(cameraIDs[camera], id)
	}
	sharpness := map[string]float64{}
	for _, group := range cameraIDs {
		ranks := percentileRanks(group, func(id string) float64 { return candidates[id].Local.FaceSharpness })
		for id, rank := range ranks {
			sharpness[id] = rank
		}
	}
	score := map[string]float64{}
	var ordered []string
	for _, id := range ids {
		if sharpness[id]*100 < options.DropBlurriestPercent {
			rejected[id] = "blurriest faces"
			continue
		}
		score[id] = 0.7*sharpness[id] + 0.3*min(candidates[id].Local.FaceArea/options.FaceAreaEnough, 1)
		ordered = append(ordered, id)
	}
	slices.SortStableFunc(ordered, func(a, b string) int { return cmp.Compare(score[b], score[a]) })

	// Keep the best of each group of duplicates, and of each outing. Outings are
	// worked out from every photo, so a filter can't merge or split them.
	outings := outingNumbers(allAssets, options.OutingGap)
	duplicateGroups := map[string]string{}
	outingCounts := map[int]int{}
	type hashed struct {
		id    string
		hash  uint64
		taken time.Time
	}
	var kept []hashed
	for _, id := range ordered {
		group := assets[id].DuplicateID
		if keeper, seen := duplicateGroups[group]; group != "" && seen {
			rejected[id] = immichDuplicatePrefix + keeper
			continue
		}
		hash, err := strconv.ParseUint(candidates[id].Local.DHash, 16, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("decoding hash of %s: %w", id, err)
		}
		twin := slices.IndexFunc(kept, func(other hashed) bool {
			return bits.OnesCount64(hash^other.hash) <= options.DuplicateBits
		})
		if twin >= 0 {
			rejected[id] = nearDuplicatePrefix + kept[twin].id
			continue
		}
		taken, timeErr := time.Parse(time.RFC3339, assets[id].LocalDateTime)
		if options.BurstWindow > 0 && timeErr == nil {
			burst := slices.IndexFunc(kept, func(other hashed) bool {
				return !other.taken.IsZero() && taken.Sub(other.taken).Abs() <= options.BurstWindow
			})
			if burst >= 0 {
				rejected[id] = burstPrefix + kept[burst].id
				continue
			}
		}
		if options.MaxPerOuting > 0 && outingCounts[outings[id]] >= options.MaxPerOuting {
			rejected[id] = overOutingCap
			continue
		}
		kept = append(kept, hashed{id, hash, taken})
		outingCounts[outings[id]]++
		if group != "" {
			duplicateGroups[group] = id
		}
	}

	// Take turns between cameras, so phone selfies can't crowd out the rest, and
	// within each camera between themes, so one theme can't fill the list.
	var cameras []string
	keptByCamera := map[string][]string{}
	for _, item := range kept {
		camera := assets[item.id].Camera()
		if _, known := keptByCamera[camera]; !known {
			cameras = append(cameras, camera)
		}
		keptByCamera[camera] = append(keptByCamera[camera], item.id)
	}
	queues := map[string][]string{}
	for camera, group := range keptByCamera {
		queues[camera] = themeOrder(group, func(id string) []string { return candidates[id].Queries })
	}
	var chosen []string
	seen := map[string]bool{}
	for progress := true; progress && len(chosen) < options.Limit; {
		progress = false
		for _, camera := range cameras {
			if len(queues[camera]) > 0 && len(chosen) < options.Limit {
				chosen = append(chosen, queues[camera][0])
				seen[queues[camera][0]] = true
				queues[camera] = queues[camera][1:]
				progress = true
			}
		}
	}
	for _, item := range kept {
		if !seen[item.id] {
			rejected[item.id] = "over the limit"
		}
	}

	entries := make([]ShortlistEntry, 0, len(chosen))
	for index, id := range chosen {
		record, asset := candidates[id], assets[id]
		entries = append(entries, ShortlistEntry{
			Rank:         index + 1,
			AssetID:      id,
			Score:        round(score[id], 3),
			File:         asset.OriginalFileName,
			OriginalPath: asset.OriginalPath,
			Taken:        asset.LocalDateTime,
			Camera:       asset.Camera(),
			Outing:       outings[id],
			Queries:      record.Queries,
			PreviewFile:  record.PreviewFile,
			Labelled:     record.LLM != nil,
		})
	}
	return entries, rejected, nil
}

// WriteShortlist builds the shortlist and saves it with the reasons and previews.
func WriteShortlist(catalog *Catalog, options ShortlistOptions) error {
	records, err := catalog.RunRecords()
	if err != nil {
		return err
	}
	entries, rejected, err := BuildShortlist(records, options)
	if err != nil {
		return err
	}
	shortlist := Shortlist{CreatedAt: nowUTC(), Photos: entries}
	if err := writeJSON(filepath.Join(catalog.root, "shortlist.json"), shortlist); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(catalog.root, "rejections.json"), rejected); err != nil {
		return err
	}

	if err := clearFiles(catalog.shortlist); err != nil {
		return err
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(catalog.previews, entry.PreviewFile))
		if err != nil {
			return err
		}
		name := fmt.Sprintf("%03d_%s", entry.Rank, entry.PreviewFile)
		if err := writeFileAtomic(filepath.Join(catalog.shortlist, name), data); err != nil {
			return err
		}
	}

	counts := map[string]int{}
	for _, reason := range rejected {
		switch {
		case strings.HasPrefix(reason, nearDuplicatePrefix):
			reason = "near-duplicate"
		case strings.HasPrefix(reason, immichDuplicatePrefix):
			reason = "Immich duplicate"
		case strings.HasPrefix(reason, burstPrefix):
			reason = "same burst"
		}
		counts[reason]++
	}
	reasons := make([]string, 0, len(counts))
	for reason := range counts {
		reasons = append(reasons, reason)
	}
	slices.SortFunc(reasons, func(a, b string) int {
		return cmp.Or(cmp.Compare(counts[b], counts[a]), cmp.Compare(a, b))
	})
	fmt.Printf("Shortlisted %d photos in %s\n", len(entries), catalog.shortlist)
	for _, reason := range reasons {
		fmt.Printf("  rejected %d: %s\n", counts[reason], reason)
	}
	return nil
}

// --- contact sheets ---

const (
	sheetCellPixels = 512
	sheetLabelScale = 3
)

// drawNumber draws an enlarged number on a black box in a cell's corner.
func drawNumber(sheet *image.RGBA, origin image.Point, number int) {
	text := strconv.Itoa(number)
	face := basicfont.Face7x13
	width, height := font.MeasureString(face, text).Ceil()+2, face.Height+2
	small := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(small, small.Bounds(), image.NewUniform(color.Black), image.Point{}, draw.Src)
	drawer := font.Drawer{
		Dst:  small,
		Src:  image.NewUniform(color.White),
		Face: face,
		Dot:  fixed.P(1, face.Ascent+1),
	}
	drawer.DrawString(text)
	box := image.Rectangle{Min: origin, Max: origin.Add(image.Pt(width*sheetLabelScale, height*sheetLabelScale))}
	draw.NearestNeighbor.Scale(sheet, box, small, small.Bounds(), draw.Src, nil)
}

// WriteSheets draws the shortlist into numbered grids, one number per rank in
// shortlist.json, so many photos can be reviewed as a few images.
func WriteSheets(catalog *Catalog, perSheet int) error {
	if perSheet < 1 {
		return fmt.Errorf("per-sheet must be at least 1")
	}
	data, err := os.ReadFile(filepath.Join(catalog.root, "shortlist.json"))
	if err != nil {
		return fmt.Errorf("reading shortlist: %w; run shortlist first", err)
	}
	var shortlist Shortlist
	if err := json.Unmarshal(data, &shortlist); err != nil {
		return fmt.Errorf("decoding shortlist: %w", err)
	}
	if err := clearFiles(catalog.sheets); err != nil {
		return err
	}

	columns := int(math.Ceil(math.Sqrt(float64(perSheet))))
	rows := int(math.Ceil(float64(perSheet) / float64(columns)))
	count := 0
	for photos := range slices.Chunk(shortlist.Photos, perSheet) {
		sheet := image.NewRGBA(image.Rect(0, 0, columns*sheetCellPixels, rows*sheetCellPixels))
		draw.Draw(sheet, sheet.Bounds(), image.NewUniform(color.RGBA{24, 24, 24, 255}), image.Point{}, draw.Src)
		for index, photo := range photos {
			preview, err := os.ReadFile(filepath.Join(catalog.previews, photo.PreviewFile))
			if err != nil {
				return err
			}
			img, _, err := image.Decode(bytes.NewReader(preview))
			if err != nil {
				return fmt.Errorf("decoding preview of %s: %w", photo.AssetID, err)
			}
			origin := image.Pt(index%columns*sheetCellPixels, index/columns*sheetCellPixels)
			bounds := img.Bounds()
			fit := min(float64(sheetCellPixels)/float64(bounds.Dx()), float64(sheetCellPixels)/float64(bounds.Dy()))
			size := image.Pt(int(float64(bounds.Dx())*fit), int(float64(bounds.Dy())*fit))
			offset := origin.Add(image.Pt((sheetCellPixels-size.X)/2, (sheetCellPixels-size.Y)/2))
			draw.CatmullRom.Scale(sheet, image.Rectangle{Min: offset, Max: offset.Add(size)}, img, bounds, draw.Src, nil)
			drawNumber(sheet, origin, photo.Rank)
		}
		count++
		var buffer bytes.Buffer
		if err := jpeg.Encode(&buffer, sheet, &jpeg.Options{Quality: 85}); err != nil {
			return err
		}
		if err := writeFileAtomic(filepath.Join(catalog.sheets, fmt.Sprintf("sheet-%03d.jpg", count)), buffer.Bytes()); err != nil {
			return err
		}
	}
	fmt.Printf("Wrote %d contact sheets of %d photos to %s\n", count, len(shortlist.Photos), catalog.sheets)
	return nil
}
