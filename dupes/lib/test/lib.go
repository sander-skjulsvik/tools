package test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
	files "github.com/sander-skjulsvik/tools/libs/files"
)

func SetupExpectedDupes(path string) {
	expectedDupes := GetExpectedDupes(path)
	// Create the nested folder structure from ExectedDupes
	for hash, dupe := range expectedDupes.D {
		for _, filePath := range dupe.Paths {
			fullPath := filePath
			// Create the parent folders if they don't exist
			CrateParentFolders(fullPath)
			// Create the file with content from ExpectedDupesHashMap
			files.CreateFile(fullPath, ExpectedDupesHashMap[hash])
		}
	}
}

func CrateParentFolders(path string) {
	// Remove  entry in path
	parentPath := filepath.Dir(path)

	// Create the base directory if it doesn't exist
	if err := os.MkdirAll(parentPath, os.ModePerm); err != nil {
		fmt.Println("Error creating base directory:", err)
		return
	}
}

// ExpectedDupesHashMap is a map of hashes to the content of the files with that hash, used for testing
var ExpectedDupesHashMap map[string]string = map[string]string{
	"a22fcc1f0e918f91dcfc42577c4d522247dd30068a4d48abf07309ad9e30fae3": "Group1Content",
	"273f01d16e4462e20fb94153d4b98a2dd36b0c5cd522e02cdeba5b275f4e3a1c": "Group2Content",
	"3edb15fc42ec3928274be9b5b29435130b2d48c86b750c1427aa788a077dd598": "Group3Content",
	"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855": "",
}

// ExpectedDupes is a map of hashes to the paths of the files with that hash, used for testing
func GetExpectedDupes(basePath string) common.Dupes {
	d := common.Dupes{
		D: map[string]*common.Dupe{
			"a22fcc1f0e918f91dcfc42577c4d522247dd30068a4d48abf07309ad9e30fae3": {
				Hash: "a22fcc1f0e918f91dcfc42577c4d522247dd30068a4d48abf07309ad9e30fae3",
				Paths: []string{
					"Folder1/Group1",
					"Folder1/Folder1/Group1",
					"Folder2/Group1",
					"Folder2/Folder1/Folder2/Folder2/Group1",
					"Group1_2",
				},
			}, "273f01d16e4462e20fb94153d4b98a2dd36b0c5cd522e02cdeba5b275f4e3a1c": {
				Hash: "273f01d16e4462e20fb94153d4b98a2dd36b0c5cd522e02cdeba5b275f4e3a1c",
				Paths: []string{
					"Folder1/Group2",
					"Folder1/Folder1/Group2",
					"Folder2/Group2",
					"Folder2/Folder1/Folder2/Folder2/Group2",
					"Folder2/Folder1/Folder2/Folder2/Group2_2",
					// Same name as files in group1
					"Folder2/Folder1/Folder2/Group1",
					"Group2_2",
				},
			}, "3edb15fc42ec3928274be9b5b29435130b2d48c86b750c1427aa788a077dd598": {
				Hash: "3edb15fc42ec3928274be9b5b29435130b2d48c86b750c1427aa788a077dd598",
				Paths: []string{
					// Only one file
					"Folder2/Group3",
				},
			}, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855": {
				Hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				Paths: []string{
					"Folder2/emtpy1",
					"Folder2/Folder1/emtpy1",
					"Folder2/Folder1/Folder2/emtpy1",
				},
			},
		},
	}
	// Add the base path to the paths
	for _, dupe := range d.D {
		for i, path := range dupe.Paths {
			dupe.Paths[i] = filepath.Join(basePath, path)
		}
	}
	return d
}
