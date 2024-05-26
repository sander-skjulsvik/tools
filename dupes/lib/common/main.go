package common

// Package common provides common functions for finding duplicates.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/sander-skjulsvik/tools/libs/progressbar"
)

// Run is the main function to run for consumers of this lib.
// First arg is the path to the folder,
type Run func(string, progressbar.ProgressBar) *Dupes

type File struct {
	Path string
	Hash string
}

func HashString(b []byte) string {
	return hex.EncodeToString(b)
}

func IsFile(f os.FileInfo) bool {
	if f == nil {
		panic(fmt.Errorf("file info is nil"))
	}
	return f.Mode().IsRegular()
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open: %s: %w\n", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash: %s: %w\n", path, err)
	}

	return HashString(h.Sum(nil)), nil
}
