package producerConsumer

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
)

func TestMain(m *testing.T) {
	common.TestRun("test_main_producer_consumer", Run)
}

func TestGetFiles(t *testing.T) {
	baseDir := "test_get_files/"
	defer os.RemoveAll(filepath.Clean(baseDir))

	// Nesting
	{
		workDir := baseDir + "test_nesting/"
		os.MkdirAll(workDir+filepath.Clean("/folder/folder/"), 0755)
		expectedFilePaths := []string{
			workDir + "nesting_file_name",
			workDir + "folder/" + "nesting_file_name",
			workDir + "folder/" + "folder/" + "nesting_file_name",
		}
		for _, file := range expectedFilePaths {
			common.CreateFile(file, "nesting_file_content")
		}
		calculatedFilePaths := make(chan string)
		go getFiles(workDir, calculatedFilePaths)
		ind := 0
		for calculatedPath := range calculatedFilePaths {
			if !slices.Contains(expectedFilePaths, filepath.ToSlash(calculatedPath)) {
				t.Errorf("calculatedPath: %s, not in expectedPaths: %v", calculatedPath, expectedFilePaths)
			}

			ind++
			if ind > 3 {
				t.Errorf(
					"Expected to find one file, fund %d", ind,
				)
			}
		}
	}

	// Empty file
	{
		workDir := baseDir + "test_emtpy_file/"
		os.MkdirAll(filepath.Clean(workDir), 0755)
		common.CreateEmptyFile(workDir + "empty_file")
		common.CreateFile(workDir+"not_empty_file", "not_empty_file")

		calculatedFilePaths := make(chan string)
		go getFiles(workDir, calculatedFilePaths)
		ind := 0
		for range calculatedFilePaths {
			ind++
		}
		if ind != 2 {
			t.Errorf("Expected 2 files, but got: %d", ind)
		}
	}

	// Symlink
	{
		workDir := baseDir + "test_symlink/"
		os.MkdirAll(filepath.Clean(workDir), 0755)
		common.CreateEmptyFile(workDir + "source_file")
		os.Symlink(workDir+"source_file", workDir+"destination_file")

		calculatedFilePaths := make(chan string)
		go getFiles(workDir, calculatedFilePaths)
		ind := 0
		for calculatedPath := range calculatedFilePaths {
			if filepath.ToSlash(calculatedPath) == workDir+"destination_file" {
				t.Errorf(
					"Calculated path is a symlink: %s", calculatedPath,
				)
			}
			if filepath.ToSlash(calculatedPath) != workDir+"source_file" {
				t.Errorf(
					"Fund file is not source the source file: %s", calculatedPath,
				)
			}
			ind++
			if ind != 1 {
				t.Errorf(
					"Expected to find one file, fund %d", ind,
				)
			}
		}
	}
}

func TestAppendFileTreadSafe(t *testing.T) {
	baseDir := "test_append_file_thread_safe/"
	defer os.RemoveAll(filepath.Clean(baseDir))

	// Append a single file
	{
		workDir := baseDir + "single_file/"
		os.MkdirAll(filepath.Clean(workDir), 0755)
		d := common.Dupes.New(common.Dupes{})

		path := workDir + "single_file"
		lock := sync.Mutex{}
		common.CreateFile(path, "I am a single file")
		expectedHash := "1be3d7cfb6df7ff4ed6235a70603dc3ee8fa636a5e44a5c2ea8ffbcd38b41bd0"

		appendFileTreadSafe(&d, filepath.Clean(path), &lock)
		ind := 0
		for key, val := range d.D {
			if key != expectedHash {
				t.Errorf("Append single file gave the wrong hash, expected: %s, got: %s", expectedHash, key)
			}
			if len(val.Paths) != 1 {
				t.Errorf("Append single file did not give 1 path, got: %d", len(val.Paths))
			}
			if filepath.ToSlash(*val.Paths[0]) != path {
				t.Errorf("Append single file gave the wrong path, expected: %s, got: %s", path, filepath.ToSlash(*val.Paths[0]))
			}
			ind++
			if ind > 1 {
				t.Errorf("Appending single file have more than one hash: %d", ind)
			}
		}

	}

	// Adding many equal files in parallel
	{
		workDir := baseDir + "many_equal_files/"
		os.MkdirAll(filepath.Clean(workDir), 0755)
		d := common.Dupes.New(common.Dupes{})
		n := 1000
		for i := 0; i < n; i++ {
			common.CreateFile(workDir+strconv.Itoa(i), "I am one of many files")
		}

		lock := sync.Mutex{}
		expectedHash := "50b253e70fe2d6ad4c93c902c923d55d89ffdcbd32a63065e9500b51f2a9388b"

		wg := sync.WaitGroup{}
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				appendFileTreadSafe(&d, filepath.Clean(workDir+strconv.Itoa(i)), &lock)
				wg.Done()
			}()
		}
		wg.Wait()
		ind := 0
		for key, val := range d.D {
			if key != expectedHash {
				t.Errorf("Append many equal files gave the wrong hash, expected: %s, got: %s", expectedHash, key)
			}
			if len(val.Paths) != n {
				t.Errorf("Append %d equal files did not the same ammount of paths under the hash, got: %d", n, len(val.Paths))
			}
			ind++
			if ind > 1 {
				t.Errorf("Append many equal files gave more than one hash: %d", ind)
			}

		}
	}

	// Adding many different files
	{
		workDir := baseDir + "many_different_files/"
		os.MkdirAll(filepath.Clean(workDir), 0755)
		d := common.Dupes.New(common.Dupes{})
		n := 1000
		for i := 0; i < n; i++ {
			common.CreateFile(workDir+strconv.Itoa(i), "I am one of many files: "+strconv.Itoa(i))
		}

		lock := sync.Mutex{}
		wg := sync.WaitGroup{}
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				appendFileTreadSafe(&d, filepath.Clean(workDir+strconv.Itoa(i)), &lock)
				wg.Done()
			}()
		}
		wg.Wait()
		for key, val := range d.D {
			if len(val.Paths) != 1 {
				t.Errorf("Append %d uniqe files gave the same hash: %s", n, key)
			}
		}
		if len(d.D) != n {
			t.Errorf("Append many uniqe files did not give n: %d, amount of hashes, got: %d", n, len(d.D))
		}
	}
	// Append empty files
	{
		{
			workDir := baseDir + "empty_files/"
			os.MkdirAll(filepath.Clean(workDir), 0755)
			d := common.Dupes.New(common.Dupes{})

			lock := sync.Mutex{}
			n := 10
			for i := 0; i < n; i++ {
				common.CreateFile(workDir+strconv.Itoa(i), "")
			}
			expectedHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

			for i := 0; i < n; i++ {
				appendFileTreadSafe(&d, filepath.Clean(workDir+strconv.Itoa(i)), &lock)
			}

			for key, val := range d.D {
				if key != expectedHash {
					t.Errorf("Append n: %d, empty files gave the wrong hash, expected: %s, got: %s", n, expectedHash, key)
				}
				if len(val.Paths) != n {
					t.Errorf("Append n: %d, empty files did not give n paths, got: %d", n, len(val.Paths))
				}
			}
			if len(d.D) != 1 {
				t.Errorf("Append n: %d, empty files did not result in 1 hash, got: %d", n, len(d.D))
			}
		}
	}
}
