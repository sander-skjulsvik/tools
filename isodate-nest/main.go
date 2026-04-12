package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "print moves without executing them")
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("usage: isodate-nest [-dry-run] <folder>")
	}
	folder := flag.Arg(0)
	files := getAllFiles(folder)

	for _, file := range files {
		filename := file.Name()
		fileExt, valid := getPostIsoDateSuffix(filename)
		if !valid {
			continue
		}

		date := parseIsoDate(filename[:10])
		newFolder := filepath.Join(folder, fmt.Sprintf("%d/%02d", date.Year(), date.Month()))
		newPath := filepath.Join(newFolder, fmt.Sprintf("%02d%s", date.Day(), fileExt))
		fmt.Printf("%s -> %s\n", filepath.Join(folder, filename), newPath)

		if *dryRun {
			continue
		}

		os.MkdirAll(newFolder, os.ModePerm)

		err := os.Rename(filepath.Join(folder, filename), newPath)
		if err != nil {
			log.Fatal(err)
		}
	}

}

func getPostIsoDateSuffix(filename string) (string, bool) {
	// 2026-04-12-foo.txt

	if len(filename) <= 10 {
		return "", false
	}
	_, err := time.Parse("2006-01-02", filename[:10])
	if err != nil {
		return "", false
	}
	return filename[10:], true
}

func parseIsoDate(s string) time.Time {
	layout := "2006-01-02"
	t, err := time.Parse(layout, s)
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func getAllFiles(path string) []os.DirEntry {
	files, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	outFiles := make([]os.DirEntry, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		outFiles = append(outFiles, file)
	}
	return outFiles // Placeholder return - replace with actual file names
}
