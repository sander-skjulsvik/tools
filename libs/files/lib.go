package files

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
)

func GetNumberOfFiles(path string) (int, error) {
	n := 0
	err := filepath.Walk(
		path,
		func(path string, info fs.FileInfo, err error) error {
			isFile := common.IsFile(info)
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

func GetNumbeSizeOfDirMb(path string) (int, error) {
	var size int64 = 0
	err := filepath.Walk(
		path,
		func(path string, info fs.FileInfo, err error) error {
			if info == nil {
				panic(fmt.Errorf("GetNumbeSizeOfDirMb: fileinfo is nil for: %s", path))
			}
			isFile := common.IsFile(info)
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
