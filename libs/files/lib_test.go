package files

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGetAllFilesOfType(t *testing.T) {
	// Setup
	var (
		basePath = "../test_files"
		fileType = ".txt"
		paths    = []string{
			"/file1.txt",
			"/file2.txt/abc.raf",
			"/file3/abc.txt",
			"/file4.txt/abc.txt",
			"/file5.txt",
			"/file6.tx",
			"/file7t.xt",
		} // Create files
		expectedFilePaths = []string{
			"file1.txt",
			"/file3/abc.txt",
			"/file4.txt/abc.txt",
			"/file5.txt",
		}
	)
	defer os.RemoveAll(basePath)
	for _, path := range paths {
		err := CreateEmptyFileWithFolders(
			filepath.Join(basePath, path),
		)
		if err != nil {
			t.Errorf("Error creating test files: %v", err)
		}
	}
	// Test
	files, err := GetAllFilesOfType(basePath, fileType)
	if err != nil {
		t.Errorf("Error getting files of type: %v", err)
	}
	if len(files) != len(expectedFilePaths) {
		t.Errorf("Expected %d files, got %d", len(expectedFilePaths), len(files))
	}
	if slices.Equal(expectedFilePaths, files) {
		t.Errorf("Expected %v, got %v", expectedFilePaths, files)
	}
}
