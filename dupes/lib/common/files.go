package common

// Package common provides common functions for finding duplicates.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

type File struct {
	Path string
	Hash string
}

func HashString(b []byte) string {
	return hex.EncodeToString(b)
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
