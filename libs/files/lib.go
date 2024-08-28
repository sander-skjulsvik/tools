package files

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func CreateEmptyFileWithFolders(path string) error {
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create folders: %w", err)
	}
	return CreateEmptyFile(path)
}

func CreateEmptyFile(path string) error {
	d := []byte("")
	return os.WriteFile(filepath.Clean(path), d, 0o644)
}

func CreateFile(path, content string) error {
	return os.WriteFile(filepath.Clean(path), []byte(content), 0o644)
}

func GetAllFilesOfType(path string, fileType string) ([]string, error) {
	var files []string
	err := filepath.Walk(
		path,
		func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("unable to walk path: %w", err)
			}
			if info == nil {
				return fmt.Errorf("file info is nil")
			}
			if info.IsDir() {
				return nil
			}
			if filepath.Ext(path) == fileType {
				files = append(files, path)
			}
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to get all files of type: %w", err)
	}
	return files, nil
}

func GetNumberOfFiles(path string) (int, error) {
	n := 0
	err := filepath.Walk(
		path,
		func(path string, info fs.FileInfo, err error) error {
			isFile := IsFile(info)
			if !isFile {
				return nil
			}
			n++
			return nil
		},
	)
	if err != nil {
		return 0, fmt.Errorf("unable find number of files in dir: %w", err)
	}
	return n, nil
}

func IsFile(f os.FileInfo) bool {
	if f == nil {
		panic(fmt.Errorf("file info is nil"))
	}
	return f.Mode().IsRegular()
}

func GetSizeOfDirMb(path string) (int, error) {
	var size int64 = 0
	err := filepath.Walk(
		path,
		func(path string, info fs.FileInfo, err error) error {
			if info == nil {
				log.Printf("File info is nil for %s\n", path)
				return nil
			}
			isFile := IsFile(info)
			if !isFile {
				return nil
			}
			size += info.Size()
			return nil
		},
	)
	if err != nil {
		return 0, fmt.Errorf("unable find size of dir: %w", err)
	}
	return int(size / 1e6), nil
}
