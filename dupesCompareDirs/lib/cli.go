package dupescomparedirs

import (
	"flag"
	"fmt"
	"os"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
	producerconsumer "github.com/sander-skjulsvik/tools/dupes/lib/producerConsumer"
	"github.com/sander-skjulsvik/tools/dupes/lib/singleThread"
	"github.com/sander-skjulsvik/tools/libs/progressbar"
)

func RunComparison(comparisonFunc ComparisonFunc) {
	outputJson := flag.Bool("json", false, "If set to true Output as json")
	withProgressBar := flag.Bool("withProgressBar", true, "If set to true display progress bar")
	runnerMode := flag.String("runMode", "singleThread", "possible run modes: singleThread, producerConsumer and nThreads")
	nThreads := flag.Int("nThreads", 0, "number of threads to use, only used witt runMode nThreads")
	dir1 := flag.String("dir1", "", "Path to 1st dir")
	dir2 := flag.String("dir2", "", "Path to 2nd dir")
	flag.Parse()

	// Check if directory paths are provided
	if *dir1 == "" || *dir2 == "" {
		fmt.Println("Please provide directory paths to compare")
		os.Exit(1)
	}

	// Progress bar
	var pbCollection progressbar.ProgressBarCollection
	switch *withProgressBar {
	case true:
		pbCollection = progressbar.NewUiPCollection()
	case false:
		pbCollection = progressbar.ProgressBarCollectionMoc{}
	}

	// Runner
	var runFunc common.Run
	switch *runnerMode {
	case "singleThread":
		runFunc = singleThread.Run
	case "producerConsumer":
		runFunc = producerconsumer.Run
	case "nThreads":
		runFunc = producerconsumer.GetRunNThreads(*nThreads)
	}

	comparator := NewComparator(
		[]string{*dir1, *dir2}, runFunc, comparisonFunc, pbCollection,
	)
	dupes := comparator.Run()

	if *outputJson {
		fmt.Println(string(dupes.GetJSON()))
	} else {
		dupes.Present(false)
	}
}
