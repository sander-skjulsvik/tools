package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/sander-skjulsvik/tools/libs/immich"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// metricsVersion must be bumped when computeMetrics changes, so that analyze
// redoes only that work and reuses the downloaded previews.
const metricsVersion = 2

const (
	sharpnessCropPixels = 128
	dhashWidth          = 9
	dhashHeight         = 8
)

var previewExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
	"image/png":  ".png",
}

// boxArea returns a face's share of the image, from 0 to 1.
func boxArea(face Face) float64 {
	if face.ImageWidth <= 0 || face.ImageHeight <= 0 {
		return 0
	}
	width := float64(face.BoundingBoxX2-face.BoundingBoxX1) / float64(face.ImageWidth)
	height := float64(face.BoundingBoxY2-face.BoundingBoxY1) / float64(face.ImageHeight)
	return max(width, 0) * max(height, 0)
}

// splitFaces returns this person's largest face, and every other face in the photo.
func splitFaces(asset Asset, personID string) (*Face, []Face) {
	var own []Face
	others := slices.Clone(asset.UnassignedFaces)
	for _, person := range asset.People {
		if person.ID == personID {
			own = append(own, person.Faces...)
		} else {
			others = append(others, person.Faces...)
		}
	}
	if len(own) == 0 {
		return nil, others
	}
	slices.SortStableFunc(own, func(a, b Face) int {
		switch {
		case boxArea(a) > boxArea(b):
			return -1
		case boxArea(a) < boxArea(b):
			return 1
		}
		return 0
	})
	return &own[0], append(others, own[1:]...)
}

func toGray(img image.Image) *image.Gray {
	gray := image.NewGray(img.Bounds())
	draw.Draw(gray, gray.Bounds(), img, img.Bounds().Min, draw.Src)
	return gray
}

// laplacianVariance measures fine detail: higher means more in focus.
func laplacianVariance(gray *image.Gray) float64 {
	bounds := gray.Bounds()
	var sum, squares, count float64
	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			value := float64(gray.GrayAt(x-1, y).Y) + float64(gray.GrayAt(x+1, y).Y) +
				float64(gray.GrayAt(x, y-1).Y) + float64(gray.GrayAt(x, y+1).Y) -
				4*float64(gray.GrayAt(x, y).Y)
			sum += value
			squares += value * value
			count++
		}
	}
	if count == 0 {
		return 0
	}
	mean := sum / count
	return squares/count - mean*mean
}

// dhash is a 64-bit difference hash; similar photos differ in few bits.
func dhash(gray *image.Gray) uint64 {
	small := image.NewGray(image.Rect(0, 0, dhashWidth, dhashHeight))
	draw.CatmullRom.Scale(small, small.Bounds(), gray, gray.Bounds(), draw.Src, nil)
	var value uint64
	for y := range dhashHeight {
		for x := range dhashWidth - 1 {
			value <<= 1
			if small.GrayAt(x+1, y).Y > small.GrayAt(x, y).Y {
				value |= 1
			}
		}
	}
	return value
}

func round(value float64, decimals int) float64 {
	scale := math.Pow10(decimals)
	return math.Round(value*scale) / scale
}

// computeMetrics measures a preview using Immich's face boxes.
func computeMetrics(img image.Image, asset Asset, personID string) LocalMetrics {
	gray := toGray(img)
	own, others := splitFaces(asset, personID)
	metrics := LocalMetrics{
		Version:        metricsVersion,
		DHash:          fmt.Sprintf("%016x", dhash(gray)),
		FaceFound:      own != nil,
		OtherFaceCount: len(others),
	}
	if own == nil {
		return metrics
	}

	ownArea := boxArea(*own)
	metrics.FaceArea = round(ownArea, 5)
	largestOther := 0.0
	for _, face := range others {
		largestOther = max(largestOther, boxArea(face))
	}
	if ownArea > 0 {
		metrics.LargestOtherFaceRatio = round(largestOther/ownArea, 3)
	}

	bounds := gray.Bounds()
	if own.ImageWidth <= 0 || own.ImageHeight <= 0 {
		return metrics
	}
	scaleX := float64(bounds.Dx()) / float64(own.ImageWidth)
	scaleY := float64(bounds.Dy()) / float64(own.ImageHeight)
	crop := image.Rect(
		bounds.Min.X+int(float64(own.BoundingBoxX1)*scaleX),
		bounds.Min.Y+int(float64(own.BoundingBoxY1)*scaleY),
		bounds.Min.X+int(float64(own.BoundingBoxX2)*scaleX),
		bounds.Min.Y+int(float64(own.BoundingBoxY2)*scaleY),
	).Intersect(bounds)
	if crop.Empty() {
		return metrics
	}
	// Large faces are scaled down so they stay comparable, but small faces are
	// never scaled up: that would blur them and penalize full-body shots.
	longest := max(crop.Dx(), crop.Dy())
	scale := min(float64(sharpnessCropPixels)/float64(longest), 1)
	width := max(int(float64(crop.Dx())*scale), 1)
	height := max(int(float64(crop.Dy())*scale), 1)
	face := image.NewGray(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(face, face.Bounds(), gray, crop, draw.Src, nil)
	metrics.FaceSharpness = round(laplacianVariance(face), 2)
	return metrics
}

// analyzeOne does only the work still missing for a photo, and reports whether
// there was any.
func analyzeOne(client *Client, catalog *Catalog, record *Record, personID string) (cached bool, err error) {
	// An excluded photo's preview may have been deleted on purpose, so it must
	// never be downloaded again.
	if record.LLM != nil && record.LLM.Category == excludeCategory {
		return true, nil
	}
	previewPath := catalog.PreviewPath(record)
	if previewPath != "" && record.Local != nil && record.Local.Version == metricsVersion {
		return true, nil
	}

	asset, err := record.Asset()
	if err != nil {
		return false, err
	}
	if own, _ := splitFaces(asset, personID); own == nil {
		raw, err := client.GetAsset(record.AssetID)
		if err != nil {
			return false, err
		}
		if record.Immich, err = trimAsset(raw); err != nil {
			return false, err
		}
		if asset, err = record.Asset(); err != nil {
			return false, err
		}
	}

	var data []byte
	if previewPath != "" {
		if data, err = os.ReadFile(previewPath); err != nil {
			return false, err
		}
	} else {
		var contentType string
		if data, contentType, err = client.Preview(record.AssetID); err != nil {
			return false, err
		}
		contentType, _, _ = strings.Cut(contentType, ";")
		extension, ok := previewExtensions[strings.TrimSpace(contentType)]
		if !ok {
			return false, fmt.Errorf("unexpected preview type %q", contentType)
		}
		name := record.AssetID + extension
		if err := writeFileAtomic(filepath.Join(catalog.previews, name), data); err != nil {
			return false, err
		}
		record.PreviewFile = name
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return false, fmt.Errorf("decoding preview: %w", err)
	}
	metrics := computeMetrics(img, asset, personID)
	metrics.AnalyzedAt = nowUTC()
	record.Local = &metrics
	return false, catalog.Save(record)
}

// Analyze downloads previews and computes metrics for the last collect. One bad
// photo doesn't stop the rest; a re-run retries only what's missing.
func Analyze(client *Client, catalog *Catalog, workers int) error {
	run, err := catalog.LoadRun()
	if err != nil {
		return err
	}
	records, err := catalog.RunRecords()
	if err != nil {
		return err
	}

	var (
		mutex                          sync.Mutex
		wait                           sync.WaitGroup
		done, analyzed, cached, failed int
		forbidden                      error
	)
	queue := make(chan *Record)
	for range max(workers, 1) {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for record := range queue {
				wasCached, err := analyzeOne(client, catalog, record, run.PersonID)
				mutex.Lock()
				done++
				switch {
				case errors.Is(err, immich.ErrForbiddenRequest):
					forbidden = err
					failed++
				case err != nil:
					fmt.Fprintf(os.Stderr, "  failed %s: %v\n", record.AssetID, err)
					failed++
				case wasCached:
					cached++
				default:
					analyzed++
				}
				if done%100 == 0 {
					fmt.Printf("  %d/%d\n", done, len(records))
				}
				mutex.Unlock()
			}
		}()
	}
	for _, record := range records {
		queue <- record
	}
	close(queue)
	wait.Wait()

	fmt.Printf("Analyzed %d, already cached %d, failed %d\n", analyzed, cached, failed)
	if forbidden != nil {
		return forbidden
	}
	if failed > 0 {
		return fmt.Errorf("%d photos failed; re-run analyze to retry them", failed)
	}
	return nil
}
