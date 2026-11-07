// immich-picker builds a local, reusable catalog of the photos of one person in
// Immich and shortlists the good ones. It never writes to Immich, XMP files or
// the originals: see allowedRequests in immich.go.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const usage = `usage: immich-picker [-data-dir DIR] [-url URL] <command> [flags]

Commands:
  collect        fetch metadata for a person's photos from Immich
  analyze        download previews and compute local metrics
  shortlist      filter and rank; no network, safe to re-run
  sheets         draw the shortlist into numbered contact sheets
  import-labels  merge a JSON file of labels into the catalog
  status         show catalog progress

The server comes from -url or IMMICH_URL. The API key comes from IMMICH_API_KEY,
or the file api-key in the data folder. Run "immich-picker <command> -h" for flags.
`

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "immich-picker-data"
	}
	return filepath.Join(home, ".local", "share", "immich-picker")
}

// readAPIKey keeps the key next to the photo data rather than in a config folder,
// because config folders are often kept in git.
func readAPIKey(dataDir string) (string, error) {
	if key := strings.TrimSpace(os.Getenv("IMMICH_API_KEY")); key != "" {
		return key, nil
	}
	path := filepath.Join(dataDir, "api-key")
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("no API key: set IMMICH_API_KEY or create %s", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "warning: other users can read %s; run chmod 600 on it\n", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("%s is empty", path)
	}
	return key, nil
}

func connect(serverURL, dataDir string) (*Client, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("no server: pass -url or set IMMICH_URL")
	}
	key, err := readAPIKey(dataDir)
	if err != nil {
		return nil, err
	}
	return NewClient(serverURL, key)
}

func run(args []string) error {
	global := flag.NewFlagSet("immich-picker", flag.ExitOnError)
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

	catalog, err := OpenCatalog(*dataDir)
	if err != nil {
		return err
	}

	switch command {
	case "collect":
		person := flags.String("person", "", "name, ID or Immich link of the person (required)")
		since := flags.String("since", time.Now().AddDate(-3, 0, 0).Format("2006-01-02"), "only photos taken after this date")
		limit := flags.Int("query-limit", 50, "closest matches to keep per query")
		var queries []string
		flags.Func("query", "smart search theme; repeatable", func(value string) error {
			queries = append(queries, value)
			return nil
		})
		flags.Parse(rest)
		if *person == "" {
			return fmt.Errorf("collect needs -person")
		}
		sinceDate, err := time.Parse("2006-01-02", *since)
		if err != nil {
			return fmt.Errorf("-since must be YYYY-MM-DD: %w", err)
		}
		client, err := connect(*serverURL, *dataDir)
		if err != nil {
			return err
		}
		return Collect(client, catalog, CollectOptions{Person: *person, Since: sinceDate, Queries: queries, QueryLimit: *limit})

	case "analyze":
		workers := flags.Int("workers", 4, "parallel downloads")
		flags.Parse(rest)
		client, err := connect(*serverURL, *dataDir)
		if err != nil {
			return err
		}
		return Analyze(client, catalog, *workers)

	case "shortlist":
		var options ShortlistOptions
		flags.IntVar(&options.Limit, "limit", 150, "photos to keep")
		flags.Float64Var(&options.MinFaceArea, "min-face-area", 0.001, "smallest face as a share of the image")
		flags.Float64Var(&options.FaceAreaEnough, "face-area-enough", 0.02, "face size that earns full credit; larger earns no more")
		flags.DurationVar(&options.BurstWindow, "burst-window", 30*time.Second, "keep one photo from any moment this short; zero turns it off")
		flags.Float64Var(&options.MaxOtherFace, "max-other-face", 0.25, "largest other face relative to this person's")
		flags.Float64Var(&options.DropBlurriestPercent, "drop-blurriest-percent", 25, "share of the blurriest faces to drop")
		flags.IntVar(&options.DuplicateBits, "duplicate-bits", 8, "most differing hash bits to count as a duplicate; negative turns it off")
		flags.IntVar(&options.MaxPerOuting, "max-per-outing", 15, "photos to keep per outing; zero turns the cap off")
		flags.DurationVar(&options.OutingGap, "outing-gap", 6*time.Hour, "time between photos that starts a new outing")
		flags.Parse(rest)
		return WriteShortlist(catalog, options)

	case "sheets":
		perSheet := flags.Int("per-sheet", 9, "photos per contact sheet")
		flags.Parse(rest)
		return WriteSheets(catalog, *perSheet)

	case "import-labels":
		flags.Parse(rest)
		if flags.NArg() != 1 {
			return fmt.Errorf("usage: immich-picker import-labels <file>")
		}
		return ImportLabels(catalog, flags.Arg(0))

	case "status":
		flags.Parse(rest)
		return Status(catalog)
	}
	global.Usage()
	return fmt.Errorf("unknown command %q", command)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "immich-picker:", err)
		os.Exit(1)
	}
}
