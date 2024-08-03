package main

import (
	"image"
	"image/png"
	"os"

	"github.com/dustin/go-heatmap"
	"github.com/dustin/go-heatmap/schemes"
	"github.com/sander-skjulsvik/tools/google_location_data/lib"
)

func main() {
	googleLocationPath := "data/takeout-20240803T162513Z-001/Takeout/Location History (Timeline)/Records.json"
	locationRecords := lib.ImportSourceLocationData(googleLocationPath)

	points := []heatmap.DataPoint{}
	for _, record := range locationRecords.Locations {
		points = append(points,
			heatmap.P(record.LatitudeE7/1e7, record.LongitudeE7/1e7))
	}

	// scheme, _ := schemes.FromImage("../schemes/fire.png")
	scheme := schemes.AlphaFire

	img := heatmap.Heatmap(image.Rect(0, 0, 1024, 1024),
		points, 150, 128, scheme)
	png.Encode(os.Stdout, img)
}
