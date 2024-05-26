package common_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	common "github.com/sander-skjulsvik/tools/dupes/lib/common"
)

var TEST_DIR string = "test_dir/"

type TestFile struct {
	Content string
	Paths   []string
	Hash    string
}

var FILES = []TestFile{
	{"a93a1ffa-2674-436d-ad10-94dde9c697ea", []string{"1", "a/1", "a/a/1"}, "ad347adf1c9f742644a7f0906e153f4f1609dac98081f8ff8d3aeb33a34c9aa9"},
	{"49c2b6ec-4832-494e-a1d0-83670523fe32", []string{"2", "a/2", "b/a/2"}, "a24f2b5877ea7b1d6c3b9b0d30f317446e321d62fd4d24ca87a326663d3f7936"},
	{"6a6ec3af-a3c0-4005-8dfb-6cf9ef0416be", []string{"b/b/1"}, "858df248eff2912257bb67f9a88b926a6715523614fbd4e58dd3ad8e81f16cb6"},
	{"a3ea3e1f-9eea-4b25-8ec7-bda1dda06731", []string{"b/a/b/a/1", "a/3"}, "9492cbb32af35939f7873464e328b4544fe1c5451e128a636a26eefbd9b23821"},
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func extractDir(path string) string {
	pathParts := strings.Split(path, "/")
	rightParts := pathParts[:len(pathParts)-1]
	return strings.Join(rightParts, "/")
}

func cleanUp() {
	os.RemoveAll(TEST_DIR)
}

func setup() {
	for _, file := range FILES {
		for _, path := range file.Paths {
			err := os.MkdirAll(extractDir(TEST_DIR+path), os.ModePerm)
			if err != nil {
				fmt.Print(err)
				panic(err)
			}
			f, err := os.Create(TEST_DIR + string(path))
			check(err)
			defer f.Close()
			f.WriteString(file.Content + "\n")
			f.Sync()
		}
	}
}

func TestAppend(t *testing.T) {
	defer cleanUp()
	setup()
	d := common.NewDupes()
	if _, err := d.Append(TEST_DIR + "1"); err != nil {
		t.Errorf("Append returned error: %e", err)
		t.FailNow()
	}
	if _, err := d.Append(TEST_DIR + "b/b/1"); err != nil {
		t.Errorf("Append returned error: %e", err)
		t.FailNow()
	}
	if len(d.D) != 2 {
		t.Errorf("Running append on 2 different files and did not get 2 entries in Dupes, got %d", len(d.D))
	}
	if _, err := d.Append(TEST_DIR + "a/1"); err != nil {
		t.Errorf("Append returned error: %e", err)
		t.FailNow()
	}
	if len(d.D) != 2 {
		t.Errorf("Running append on 2 different files and one duplicate and did not get 2 entries in Dupes, got %d", len(d.D))
		t.FailNow()
	}
	count := 0
	for _, dupe := range d.D {
		count += len(dupe.Paths)
	}
	if count != 3 {
		t.Errorf("Running append on 2 different files and one duplicate and the sum of the paths was not 3, got %d", count)
		t.FailNow()
	}
}

func TestHashFile(t *testing.T) {
	defer cleanUp()
	setup()

	for _, file := range FILES {
		for _, path := range file.Paths {
			hash, err := common.HashFile(TEST_DIR + path)
			if err != nil {
				t.Errorf("Failed to hash file, %s, err: %e", TEST_DIR+path, err)
				t.FailNow()
			}
			if hash != file.Hash {
				t.Errorf("Hash given by hash file is not correct, expected: %s, got: %s, on content: %s, path: %s", file.Hash, hash, file.Content, file.Paths[0])
				t.FailNow()
			}
		}
	}
}
