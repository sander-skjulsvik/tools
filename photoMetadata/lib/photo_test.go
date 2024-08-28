package lib

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/sander-skjulsvik/tools/libs/files"
)

func TestNewPhotoCollectionFromPath(t *testing.T) {
	// Setup
	basePath := "TestNewPhotoCollectionFromPath"
	defer os.RemoveAll(basePath)

	// Test with single file
	path := path.Join(
		basePath,
		"singleFile",
		fmt.Sprintf("testfile%s", SUPPORTED_FILE_TYPES[0]),
	)
	files.CreateEmptyFileWithFolders(path)

	photoCollection, err := NewPhotoCollectionFromPath(path)
	if err != nil {
		t.Errorf("Failed to create photo collection from path: %v", err)
	}
	if len(photoCollection.Photos) != 1 {
		t.Errorf("Expected photo collection to have 1 photo, got %v", len(photoCollection.Photos))
	}
	calcPath := filepath.Clean(path)
	if photoCollection.Photos[0].Path != calcPath {
		t.Errorf("Expected photo path to be %v, got %v", calcPath, photoCollection.Photos[0].Path)
	}
}
