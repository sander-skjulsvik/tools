// location-timeline works out where you have been from the location metadata of
// your photos in Immich. It never writes to Immich: see allowedRequests in immich.go.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sander-skjulsvik/tools/libs/immich"
)

const usage = `usage: location-timeline [-data-dir DIR] [-url URL] <command> [flags]

Commands:
  sync immich  fetch the location of every geotagged photo from Immich
  cameras      list the cameras in your Immich photos, to choose yours in config.json
  stays        where you stayed, and the gaps with no evidence
  days         the places seen on each day
  countries    the countries seen in a period, such as the last 6 months
  points       every counted point as JSON, for other tools

Only sync uses the network. The server comes from -url or IMMICH_URL. The API key
comes from IMMICH_API_KEY, or the file api-key in the data folder. config.json in
the data folder lists the sources to use, and which of their data counts as yours.
Run "location-timeline <command> -h" for flags.
`

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "location-timeline-data"
	}
	return filepath.Join(home, ".local", "share", "location-timeline")
}

func openDataDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}

func writeJSON(path string, value any) error {
	var buffer bytes.Buffer
	if err := encodeJSON(&buffer, value); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buffer.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func encodeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// rangeFlags adds -since, -until and -on to a command.
func rangeFlags(flags *flag.FlagSet) func() (Range, error) {
	since := flags.String("since", "", "first date, as YYYY-MM-DD or a time ago such as 30d, 6w, 6m or 1y")
	until := flags.String("until", "", "last date, as YYYY-MM-DD or a time ago")
	on := flags.String("on", "", "a single date; overrides -since and -until")
	return func() (Range, error) {
		today := time.Now()
		if *on != "" {
			*since, *until = *on, *on
		}
		var r Range
		var err error
		if r.Since, err = ParseDate(*since, today); err != nil {
			return r, err
		}
		if r.Until, err = ParseDate(*until, today); err != nil {
			return r, err
		}
		if r.Since != "" && r.Until != "" && r.Since > r.Until {
			return r, fmt.Errorf("-since %s is after -until %s", r.Since, r.Until)
		}
		return r, nil
	}
}

func run(args []string, stdout io.Writer) error {
	global := flag.NewFlagSet("location-timeline", flag.ExitOnError)
	global.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	dataDir := global.String("data-dir", defaultDataDir(), "data folder")
	serverURL := global.String("url", os.Getenv("IMMICH_URL"), "Immich server root URL")
	global.Parse(args)
	if global.NArg() < 1 {
		global.Usage()
		os.Exit(2)
	}
	command, rest := global.Arg(0), global.Args()[1:]
	flags := flag.NewFlagSet(command, flag.ExitOnError)
	if err := openDataDir(*dataDir); err != nil {
		return err
	}

	switch command {
	case "sync":
		flags.Parse(rest)
		if flags.NArg() != 1 || flags.Arg(0) != "immich" {
			return fmt.Errorf("usage: location-timeline sync immich; it is the only source with something to sync")
		}
		if *serverURL == "" {
			return fmt.Errorf("no server: pass -url or set IMMICH_URL")
		}
		key, err := immich.ReadAPIKey(*dataDir)
		if err != nil {
			return err
		}
		client, err := immich.NewClient(*serverURL, key, allowedRequests)
		if err != nil {
			return err
		}
		return SyncImmich(client, *dataDir)

	case "cameras":
		asJSON := flags.Bool("json", false, "print JSON")
		flags.Parse(rest)
		data, err := LoadImmich(*dataDir)
		if err != nil {
			return err
		}
		// Listing cameras is how config.json gets written, so it may be missing.
		config, _ := LoadConfig(*dataDir)
		cameras := Cameras(data.Photos, config.Immich)
		if *asJSON {
			return encodeJSON(stdout, cameras)
		}
		return printCameras(stdout, cameras)

	case "stays":
		var options StayOptions
		flags.StringVar(&options.Level, "level", "city", `"city" groups photos within -radius, "country" by country`)
		flags.Float64Var(&options.RadiusKm, "radius", 25, "kilometres from a stay's centre that still count as the same place")
		flags.DurationVar(&options.MaxGap, "max-gap", 72*time.Hour, "longest time without photos inside a stay; longer is shown as a gap")
		asJSON := flags.Bool("json", false, "print JSON")
		parseRange := rangeFlags(flags)
		flags.Parse(rest)
		if options.Level != "city" && options.Level != "country" {
			return fmt.Errorf(`-level must be "city" or "country", got %q`, options.Level)
		}
		r, points, err := load(*dataDir, parseRange)
		if err != nil {
			return err
		}
		entries := InRange(Stays(points, options), r)
		if *asJSON {
			return encodeJSON(stdout, entries)
		}
		return printStays(stdout, entries)

	case "days":
		asJSON := flags.Bool("json", false, "print JSON, including days without photos")
		parseRange := rangeFlags(flags)
		flags.Parse(rest)
		r, points, err := load(*dataDir, parseRange)
		if err != nil {
			return err
		}
		days := Days(points, r)
		if *asJSON {
			return encodeJSON(stdout, days)
		}
		return printDays(stdout, days)

	case "countries":
		asJSON := flags.Bool("json", false, "print JSON")
		parseRange := rangeFlags(flags)
		flags.Parse(rest)
		r, points, err := load(*dataDir, parseRange)
		if err != nil {
			return err
		}
		report := Countries(points, r)
		if *asJSON {
			return encodeJSON(stdout, report)
		}
		return printCountries(stdout, report)

	case "points":
		parseRange := rangeFlags(flags)
		flags.Parse(rest)
		r, points, err := load(*dataDir, parseRange)
		if err != nil {
			return err
		}
		return encodeJSON(stdout, PointsInRange(points, r))
	}
	global.Usage()
	return fmt.Errorf("unknown command %q", command)
}

func load(dataDir string, parseRange func() (Range, error)) (Range, []Point, error) {
	r, err := parseRange()
	if err != nil {
		return r, nil, err
	}
	config, err := LoadConfig(dataDir)
	if err != nil {
		return r, nil, err
	}
	points, err := LoadPoints(dataDir, config)
	return r, points, err
}

func clock(local string) string {
	return strings.Replace(local[:len("2006-01-02T15:04")], "T", " ", 1)
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

func humanDuration(d time.Duration) string {
	if d < 48*time.Hour {
		return plural(int(math.Round(d.Hours())), "hour")
	}
	return plural(int(math.Round(d.Hours()/24)), "day")
}

func printStays(w io.Writer, entries []Entry) error {
	if len(entries) == 0 {
		fmt.Fprintln(w, "No photos in this period.")
		return nil
	}
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, entry := range entries {
		span := clock(entry.StartLocal) + " → " + clock(entry.EndLocal)
		if entry.Kind == "gap" {
			fmt.Fprintf(table, "%s\t(no photos for %s)\t\t\n", span, humanDuration(entry.End.Sub(entry.Start)))
			continue
		}
		place := entry.Place
		if others := len(entry.Places) - 1; others > 0 {
			place += fmt.Sprintf(" (+%d)", others)
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", span, place, plural(entry.Points, "photo"), plural(entry.Days, "day"))
	}
	return table.Flush()
}

func printDays(w io.Writer, days []Day) error {
	for i := 0; i < len(days); i++ {
		day := days[i]
		if len(day.Places) > 0 {
			var places []string
			for _, place := range day.Places {
				places = append(places, fmt.Sprintf("%s (%s–%s)", place.Place, place.First, place.Last))
			}
			fmt.Fprintf(w, "%s  %s\n", day.Date, strings.Join(places, " → "))
			continue
		}
		// A run of days without photos takes one line.
		last := i
		for last+1 < len(days) && len(days[last+1].Places) == 0 {
			last++
		}
		if last == i {
			fmt.Fprintf(w, "%s  no photos\n", day.Date)
		} else {
			fmt.Fprintf(w, "%s  no photos until %s (%s)\n", day.Date, days[last].Date, plural(last-i+1, "day"))
		}
		i = last
	}
	return nil
}

func printCountries(w io.Writer, report CountriesReport) error {
	coverage := report.Coverage
	if coverage.Days == 0 {
		fmt.Fprintln(w, "No photos in this period.")
		return nil
	}
	fmt.Fprintf(w, "%s to %s: %d of %d days have photos", coverage.Since, coverage.Until, coverage.DaysWithPoints, coverage.Days)
	if coverage.LongestGapDays > 0 {
		fmt.Fprintf(w, "; the longest gap is %s from %s", plural(coverage.LongestGapDays, "day"), coverage.LongestGapSince)
	}
	fmt.Fprintln(w, ".")
	fmt.Fprintln(w, "Countries without photos are missing here, so check the gaps.")
	fmt.Fprintln(w)
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, visit := range report.Countries {
		fmt.Fprintf(table, "%s\t%s → %s\t%s\n", visit.Country, visit.First, visit.Last, plural(visit.Days, "day"))
	}
	return table.Flush()
}

func printCameras(w io.Writer, cameras []CameraCount) error {
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "YOURS\tMAKE\tMODEL\tPHOTOS\tFIRST\tLAST")
	for _, camera := range cameras {
		yours := ""
		if camera.Yours {
			yours = "yes"
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%d\t%s\t%s\n", yours, camera.Make, camera.Model, camera.Photos, camera.First, camera.Last)
	}
	return table.Flush()
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "location-timeline:", err)
		os.Exit(1)
	}
}
