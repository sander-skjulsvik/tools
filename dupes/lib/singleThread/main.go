package singleThread

import (
	"io/fs"
	"log"
	"path/filepath"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
	"github.com/sander-skjulsvik/tools/libs/files"
	"github.com/sander-skjulsvik/tools/libs/progressbar"
)

func Run(src string, bar progressbar.ProgressBar) *common.Dupes {
	dupes := &common.Dupes{
		D: map[string]*common.Dupe{},
	}

	err := filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		isFile := files.IsFile(info)
		if !isFile {
			return nil
		}

		dupes, err = dupes.Append(path)
		bar.Add(int(info.Size() / 1e6))
		if err != nil {
			return nil
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Failed to walk src: %s, with err: %s", src, err.Error())
	}
	return dupes
}
