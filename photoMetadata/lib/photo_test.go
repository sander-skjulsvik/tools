package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/sander-skjulsvik/tools/libs/files"
	"gotest.tools/assert"
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

func TestGetDateTimeOriginal(t *testing.T) {

	p := Photo{
		Path: filepath.Clean("testData/fuji_gps.RAF"),
	}

	pTime, err := p.GetDateTimeOriginal()
	if err != nil {
		t.Errorf("Failed to get date time for: %s, err: %v", p.Path, err)
	}
	// 2024:05:18 19:38:57+02:00
	cest, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(fmt.Errorf("failed to get cest time location: %v", err))
	}
	expected := time.Date(2024, 05, 18, 19, 38, 57, 0, cest)
	if !expected.Equal(pTime) {
		t.Errorf("Photo time not as expected: expected: %s, got: %s", expected, pTime)
	}
}

func TestWriteExifGPSLocation(t *testing.T) {
	testDir := "TestWriteExifGPSLocation"
	_ = TestingSetup(testDir)
	defer os.RemoveAll(testDir)
	pWithGPS := Photo{
		Path: filepath.Clean(filepath.Join(testDir, "fuji_gps.RAF")),
	}
	pNoGPS := Photo{
		Path: filepath.Clean(filepath.Join(testDir, "fuji_no_gps.RAF")),
	}

	expectedLocation, err := pWithGPS.GetLocationRecord()
	if err != nil {
		t.Errorf("failed to get location record from photo with gps: %s", err)
	}
	pNoGPS.WriteExifGPSLocation(expectedLocation.Coordinates)
	calcLocation, err := pNoGPS.GetLocationRecord()
	if err != nil {
		t.Errorf("failed to get location after writing location to the photo: %s", err)
	}
	if !expectedLocation.Equal(calcLocation) {
		t.Errorf(
			"expected location is not equal calculated location: calc: %s, expected: %s",
			calcLocation, expectedLocation,
		)
	}
}

func TestWriteDatetime(t *testing.T) {
	testDir := "TestWriteDatetimeDir"
	testVars := TestingSetup(testDir)
	defer testVars.Clean()

	photo := testVars.PhotoCollection.Photos[0]

	origTime, err := photo.GetDateTimeOriginal()
	assert.NilError(t, err, fmt.Sprintf("failed to get orignal time from photo: %s", photo.Path))
	targetTime := origTime.Add(1 * time.Hour)

	photo.WriteDateTime(targetTime)
	calcTime, err := photo.GetDateTimeOriginal()
	assert.NilError(t, err, fmt.Sprintf("failed to get changed time from photo: %s", photo.Path))
	if targetTime.Equal(origTime) {
		t.Fatal("Target time and orig time is the same")
	}
	if !targetTime.Equal(calcTime) {
		t.Fatalf("targetTime != calcTime\n\tcalcTime: %s\n\ttargetTime: %s\n\torigTime: %s", 
		calcTime, targetTime, origTime)
	}
	assert.Equal(
		t, targetTime, calcTime,
		fmt.Errorf("targetTime != calcTime\n\tcalcTime: %s\n\ttargetTime: %s\n\torigTime: %s", 
		calcTime, targetTime, origTime))
}