package dupescomparedirs

import (
	"fmt"
	"log"
	"sync"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
	producerconsumer "github.com/sander-skjulsvik/tools/dupes/lib/producerConsumer"
	"github.com/sander-skjulsvik/tools/dupes/lib/singleThread"
	"github.com/sander-skjulsvik/tools/libs/files"
	"github.com/sander-skjulsvik/tools/libs/progressbar"
)

type ComparisonFunc func(progressBars progressbar.ProgressBarCollection, paths ...string) *common.Dupes

// OnlyInAll returns dupes that is present in all directories
func OnlyInAll(progressBars progressbar.ProgressBarCollection, paths ...string) *common.Dupes {
	ds := Run(singleThread.Run, progressBars, paths...)
	first := ds[0]

	for _, d := range ds[1:] {
		first = first.OnlyInBoth(d)
	}

	return first
}

// OnlyInFirst returns dupes that is only present in first directory
func OnlyInFirst(progressBarCollection progressbar.ProgressBarCollection, paths ...string) *common.Dupes {
	ds := Run(singleThread.Run, progressBarCollection, paths...)
	first := ds[0]
	for _, d := range ds[1:] {
		first = first.OnlyInSelf(d)
	}
	return first
}

// All returns all dupes in all directories
func All(progressBarCollection progressbar.ProgressBarCollection, paths ...string) *common.Dupes {
	dupes := common.NewDupes()
	for _, dupe := range Run(singleThread.Run, progressBarCollection, paths...) {
		dupes.AppendDupes(dupe)
	}
	return &dupes
}

func runSingleThread(progressBarCollection progressbar.ProgressBarCollection, paths ...string) []*common.Dupes {
	return Run(singleThread.Run, progressBarCollection, paths...)
}

func runMultithread(progressBarCollection progressbar.ProgressBarCollection, nThreads int, paths ...string) []*common.Dupes {
	return Run(producerconsumer.GetRunNThreads(nThreads), progressBarCollection, paths...)
}

func Run(runFunc common.Run, progressBarCollection progressbar.ProgressBarCollection, paths ...string) []*common.Dupes {
	wg := sync.WaitGroup{}
	wg.Add(len(paths))
	dupesCollection := make([]*common.Dupes, len(paths))

	progressBarCollection.Start()

	for ind, path := range paths {
		go func() {
			defer wg.Done()
			log.Printf("Running dupes on: %s", path)
			n, err := files.GetNumbeSizeOfDirMb(path)
			if err != nil {
				panic(fmt.Errorf("unable to get size of directory: %w", err))
			}
			bar := progressBarCollection.AddBar(path, n)
			dupesCollection[ind] = runFunc(path, bar)
		}()
	}
	wg.Wait()
	progressBarCollection.Stop()

	return dupesCollection
}
