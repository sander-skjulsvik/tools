package producerconsumer

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
	"github.com/sander-skjulsvik/tools/dupes/lib/test"
	"github.com/sander-skjulsvik/tools/libs/progressbar"
	"gotest.tools/assert"
)

func TestRun(t *testing.T) {
	testDir := "test_main_producer_consumer/"
	defer os.RemoveAll(filepath.Clean(testDir))
	t.Logf("running test main producer consumer. testDir: %s", testDir)
	test.TestRun(testDir, Run, t)
}

func TestGetFiles(t *testing.T) {
	baseDir := "test_get_files/"
	defer os.RemoveAll(filepath.Clean(baseDir))

	// Nesting
	{
		workDir := baseDir + "test_nesting/"
		os.MkdirAll(workDir+filepath.Clean("/folder/folder/"), 0o755)
		expectedFilePaths := []string{
			workDir + "nesting_file_name",
			workDir + "folder/" + "nesting_file_name",
			workDir + "folder/" + "folder/" + "nesting_file_name",
		}
		for _, file := range expectedFilePaths {
			test.CreateFile(file, "nesting_file_content")
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
		os.MkdirAll(filepath.Clean(workDir), 0o755)
		test.CreateEmptyFile(workDir + "empty_file")
		test.CreateFile(workDir+"not_empty_file", "not_empty_file")

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
		os.MkdirAll(filepath.Clean(workDir), 0o755)
		test.CreateEmptyFile(workDir + "source_file")
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
	// Sleeping before consuming
	// This test is to cach if the program closes the channel without locking or panicing
	{
		workDir := baseDir + "sleeping_before_consuming/"
		os.MkdirAll(filepath.Clean(workDir), 0o755)
		test.CreateFile(filepath.Join(workDir, "1"), "1")
		test.CreateFile(filepath.Join(workDir, "2"), "2")
		test.CreateFile(filepath.Join(workDir, "3"), "3")

		calculatedFilePathsChan := make(chan string)
		calculatedFilePathsSlice := []string{}
		go getFiles(workDir, calculatedFilePathsChan)
		time.Sleep(10 * time.Second)

		for calcPath := range calculatedFilePathsChan {
			calculatedFilePathsSlice = append(calculatedFilePathsSlice, calcPath)
		}
		assert.Assert(t, len(calculatedFilePathsSlice) == 3)
	}
}

func TestAppendFileTreadSafe(t *testing.T) {
	baseDir := "test_append_file_thread_safe/"
	defer os.RemoveAll(filepath.Clean(baseDir))

	// Append a single file
	{
		workDir := baseDir + "single_file/"
		os.MkdirAll(filepath.Clean(workDir), 0o755)
		d := common.NewDupes()

		path := workDir + "single_file"
		lock := sync.Mutex{}
		test.CreateFile(path, "I am a single file")
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
			if filepath.ToSlash(val.Paths[0]) != path {
				t.Errorf("Append single file gave the wrong path, expected: %s, got: %s", path, filepath.ToSlash(val.Paths[0]))
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
		os.MkdirAll(filepath.Clean(workDir), 0o755)
		d := common.NewDupes()
		n := 1000
		for i := 0; i < n; i++ {
			test.CreateFile(workDir+strconv.Itoa(i), "I am one of many files")
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
		os.MkdirAll(filepath.Clean(workDir), 0o755)
		d := common.NewDupes()
		n := 1000
		for i := 0; i < n; i++ {
			test.CreateFile(workDir+strconv.Itoa(i), "I am one of many files: "+strconv.Itoa(i))
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
			os.MkdirAll(filepath.Clean(workDir), 0o755)
			d := common.NewDupes()

			lock := sync.Mutex{}
			n := 10
			for i := 0; i < n; i++ {
				test.CreateFile(workDir+strconv.Itoa(i), "")
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

func TestProcessFiles(t *testing.T) {
	baseDir := "test_process_files/"
	defer os.RemoveAll(filepath.Clean(baseDir))

	// Process 1 file
	{
		workDir := baseDir + "one_file/"
		os.MkdirAll(filepath.Clean(workDir), 0o755)
		path := workDir + "single_file"

		test.CreateFile(path, "I am a single file")
		expectedHash := "1be3d7cfb6df7ff4ed6235a70603dc3ee8fa636a5e44a5c2ea8ffbcd38b41bd0"

		filePaths := make(chan string)
		wg := sync.WaitGroup{}
		var d *common.Dupes
		wg.Add(1)
		go func() {
			d = ProcessFiles(filePaths, progressbar.ProgressBarMoc{})
			wg.Done()
		}()
		filePaths <- filepath.Clean(path)
		close(filePaths)
		wg.Wait()

		if len(d.D) != 1 {
			t.Errorf("TestProcessFiles: process 1 file, did not get 1 hash, got %d", len(d.D))
		}
		for key, val := range d.D {
			if key != expectedHash {
				t.Errorf("TestProcessFiles: process 1 file, got wrong hash, expected: %s, got: %s", expectedHash, key)
			}
			if len(val.Paths) != 1 {
				t.Errorf("TestProcessFiles: process 1 file, hash got more than one path: %d", len(val.Paths))
			}
		}
	}

	// Process mix of nested files
	{
		type File struct {
			Path    string
			Content string
			Hash    string
		}
		workDir := baseDir + "nested_files/"
		os.MkdirAll(filepath.Clean(workDir), 0o755)

		os.MkdirAll(workDir+filepath.Clean("/folder/folder/"), 0o755)
		expectedFilePaths := []File{{
			Path:    workDir + "nesting_file_name3123",
			Content: "I am unique",
		}, {
			Path:    workDir + "folder/" + "nesting_file_name",
			Content: "I am not unique",
		}, {
			Path:    workDir + "folder/" + "folder/" + "nesting_file_name",
			Content: "I am not unique",
		}, {
			Path:    workDir + "folder/" + "folder/" + "nesting_file_name_1",
			Content: "I am not unique",
		}, {
			Path:    workDir + "folder/" + "nesting_file_name_2",
			Content: "I am not unique",
		}}
		for _, file := range expectedFilePaths {
			test.CreateFile(file.Path, file.Content)
		}

		filePaths := make(chan string)
		wg := sync.WaitGroup{}
		var d *common.Dupes
		wg.Add(1)
		go func() {
			d = ProcessFiles(filePaths, progressbar.ProgressBarMoc{})
			wg.Done()
		}()
		wgAdd := sync.WaitGroup{}
		wgAdd.Add(len(expectedFilePaths))
		for _, f := range expectedFilePaths {
			go func() {
				filePaths <- filepath.Clean(f.Path)
				wgAdd.Done()
			}()
		}
		wgAdd.Wait()
		close(filePaths)
		wg.Wait()

		if len(d.D) != 2 {
			t.Errorf("TestProcessFiles: process nested files, expected 2 unique files, got: %d", len(d.D))
		}
		for hash, val := range d.D {
			if len(val.Paths) == 1 {
				if hash != "e83ada05c293a82c303c0348fb1003d886cb64578e60cc50971d86538b7c67fd" {
					t.Errorf("TestProcessFiles: processing nested files, unique file hash wrog hash: file: %s, expected hash %s, got: %s",
						val.Paths[0], "e83ada05c293a82c303c0348fb1003d886cb64578e60cc50971d86538b7c67fd", hash,
					)
				}
			} else if len(val.Paths) == 4 {
				if hash != "5789c6f31463a1cfc7fc5f2b1a593b2970b73f203efbd235d6d3b5a6d93c425f" {
					t.Errorf("TestProcessFiles: processing nested files, not unique file hash wrog hash: file: %s, expected hash %s, got: %s",
						val.Paths[0], "5789c6f31463a1cfc7fc5f2b1a593b2970b73f203efbd235d6d3b5a6d93c425f", hash,
					)
				}
				if len(val.Paths) != 4 {
					t.Errorf("TestProcessFiles: processing nested files, not unique file got wrong number of paths, expected: %d, got: %d", 4, len(val.Paths))
				}
			} else {
				t.Errorf("TestProcessFiles: processing nested files, got the wrong duber of dupes, expected a dupe with 1 path and a dupe with 4 paths")
			}
		}
	}
}

func TestProcessFilesNConsumers(t *testing.T) {
	baseDir := "test_process_files_N_consumers/"
	defer os.RemoveAll(filepath.Clean(baseDir))

	// Process 1 file
	{
		workDir := baseDir + "one_file/"
		os.MkdirAll(filepath.Clean(workDir), 0o755)
		path := workDir + "single_file"

		test.CreateFile(path, "I am a single file")
		expectedHash := "1be3d7cfb6df7ff4ed6235a70603dc3ee8fa636a5e44a5c2ea8ffbcd38b41bd0"

		filePaths := make(chan string)
		wg := sync.WaitGroup{}
		var d *common.Dupes
		wg.Add(1)
		doneWg := sync.WaitGroup{}
		doneWg.Add(1)
		go func() {
			d = ProcessFilesNConsumers(filePaths, 3, progressbar.ProgressBarMoc{})
			wg.Done()
		}()
		filePaths <- filepath.Clean(path)
		close(filePaths)
		wg.Wait()

		if len(d.D) != 1 {
			t.Errorf("TestProcessFiles: process 1 file, did not get 1 hash, got %d", len(d.D))
		}
		for key, val := range d.D {
			if key != expectedHash {
				t.Errorf("TestProcessFiles: process 1 file, got wrong hash, expected: %s, got: %s", expectedHash, key)
			}
			if len(val.Paths) != 1 {
				t.Errorf("TestProcessFiles: process 1 file, hash got more than one path: %d", len(val.Paths))
			}
		}
	}

	// Process mix of nested files
	{
		type File struct {
			Path    string
			Content string
			Hash    string
		}
		workDir := baseDir + "nested_files/"
		os.MkdirAll(filepath.Clean(workDir), 0o755)

		os.MkdirAll(workDir+filepath.Clean("/folder/folder/"), 0o755)
		expectedFilePaths := []File{{
			Path:    workDir + "nesting_file_name3123",
			Content: "I am unique",
		}, {
			Path:    workDir + "folder/" + "nesting_file_name",
			Content: "I am not unique",
		}, {
			Path:    workDir + "folder/" + "folder/" + "nesting_file_name",
			Content: "I am not unique",
		}, {
			Path:    workDir + "folder/" + "folder/" + "nesting_file_name_1",
			Content: "I am not unique",
		}, {
			Path:    workDir + "folder/" + "nesting_file_name_2",
			Content: "I am not unique",
		}}
		for _, file := range expectedFilePaths {
			test.CreateFile(file.Path, file.Content)
		}

		filePaths := make(chan string)
		wg := sync.WaitGroup{}
		var d *common.Dupes
		wg.Add(1)
		doneWg := sync.WaitGroup{}
		doneWg.Add(1)
		go func() {
			d = ProcessFilesNConsumers(filePaths, 3, progressbar.ProgressBarMoc{})
			wg.Done()
		}()
		wgAdd := sync.WaitGroup{}
		wgAdd.Add(len(expectedFilePaths))
		for _, f := range expectedFilePaths {
			go func() {
				filePaths <- filepath.Clean(f.Path)
				wgAdd.Done()
			}()
		}
		wgAdd.Wait()
		close(filePaths)
		wg.Wait()

		if len(d.D) != 2 {
			t.Errorf("TestProcessFiles: process nested files, expected 2 unique files, got: %d", len(d.D))
		}
		for hash, val := range d.D {
			if len(val.Paths) == 1 {
				if hash != "e83ada05c293a82c303c0348fb1003d886cb64578e60cc50971d86538b7c67fd" {
					t.Errorf("TestProcessFiles: processing nested files, unique file hash wrog hash: file: %s, expected hash %s, got: %s",
						val.Paths[0], "e83ada05c293a82c303c0348fb1003d886cb64578e60cc50971d86538b7c67fd", hash,
					)
				}
			} else if len(val.Paths) == 4 {
				if hash != "5789c6f31463a1cfc7fc5f2b1a593b2970b73f203efbd235d6d3b5a6d93c425f" {
					t.Errorf("TestProcessFiles: processing nested files, not unique file hash wrog hash: file: %s, expected hash %s, got: %s",
						val.Paths[0], "5789c6f31463a1cfc7fc5f2b1a593b2970b73f203efbd235d6d3b5a6d93c425f", hash,
					)
				}
				if len(val.Paths) != 4 {
					t.Errorf("TestProcessFiles: processing nested files, not unique file got wrong number of paths, expected: %d, got: %d", 4, len(val.Paths))
				}
			} else {
				t.Errorf("TestProcessFiles: processing nested files, got the wrong duber of dupes, expected a dupe with 1 path and a dupe with 4 paths")
			}
		}
	}
}
