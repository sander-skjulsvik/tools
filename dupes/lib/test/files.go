package test

import (
	"os"
	"path/filepath"
	"slices"

	files "github.com/sander-skjulsvik/tools/libs/files"
)

type Folder struct {
	Name    string
	Files   []File
	Folders []Folder
}

type File struct {
	Name    string
	Content string
	Hash    string
}

func (folder *Folder) Generate(parents string) {
	// Create the folder if it doesn't exist
	os.MkdirAll(
		filepath.Join(parents, folder.Name),
		os.ModePerm,
	)

	// Create files in the folder
	for _, file := range folder.Files {
		filePath := filepath.Join(parents, folder.Name, file.Name)
		files.CreateFile(filePath, file.Content)
	}

	// Create child folders
	for _, childFolder := range folder.Folders {
		childFolder.Generate(
			filepath.Join(parents, folder.Name),
		)
	}
}

func (folder *Folder) Clean() {
	os.RemoveAll(folder.Name)
}

func (folder *Folder) GetFullFilePaths(parents string) []string {
	paths := []string{}
	for _, file := range folder.Files {
		paths = append(paths, filepath.Join(parents, folder.Name, file.Name))
	}
	for _, childFolder := range folder.Folders {
		paths = slices.Concat(paths, childFolder.GetFullFilePaths(filepath.Join(parents, folder.Name)))
	}
	return paths
}
