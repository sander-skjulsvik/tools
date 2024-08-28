package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sander-skjulsvik/tools/libs/files"
)

func TestNewPhotoCollectionFromPath(t *testing.T) {
	// Setup
	basePath := "TestNewPhotoCollectionFromPath"
	defer os.RemoveAll(basePath)

	// Test with single file
	path := filepath.Join(
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

	// Test with directory
	dirPath := filepath.Join(
		basePath,
		"directory",
	)
	defer os.RemoveAll(dirPath)
	filePaths := []string{
		"testfile1",
		"testfile1.raf",
		"d/d/d/d/d/testfile2.raf",
		"d/d/d/d/d/testfile2.noVaid",
	}
	expectedFilePaths := []string{
		filepath.Join(dirPath, filePaths[1]),
		filepath.Join(dirPath, filePaths[2]),
	}
	for _, file := range filePaths {
		files.CreateEmptyFileWithFolders(filepath.Join(dirPath, file))
	}
	calc, err := NewPhotoCollectionFromPath(dirPath)
	if err != nil {
		t.Errorf("Failed to create photo collection from path: %v", err)
	}
	calcFilePaths := []string{}
	for _, photo := range calc.Photos {
		calcFilePaths = append(calcFilePaths, photo.Path)
	}
	if len(calc.Photos) != 2 {
		t.Errorf("Expected photo collection to have 2 photos, got %v", len(calc.Photos))
	}

	if slices.Equal(calcFilePaths, expectedFilePaths) {
		t.Errorf("Expected photo collection to have paths %v, got %v", expectedFilePaths, calcFilePaths)
	}
}
