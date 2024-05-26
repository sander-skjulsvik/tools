package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
	producerconsumer "github.com/sander-skjulsvik/tools/dupes/lib/producerConsumer"
	"github.com/sander-skjulsvik/tools/dupes/lib/singleThread"
	comparedirs "github.com/sander-skjulsvik/tools/dupesCompareDirs/lib"
	"github.com/sander-skjulsvik/tools/libs/progressbar"
)

func main() {
	// Define command-line flags
	compMode := flag.String("mode", "all", "Mode to run in, modes: OnlyInboth, onlyInFirst, all")
	runnerMode := flag.String("runMode", "singleThread", "possible run modes: singleThread, producerConsumer and nThreads")
	nThreads := flag.Int("nThreads", 0, "number of threads to use, only used witt runMode nThreads")
	outputJson := flag.Bool("json", false, "If set to true Output as json")
	withProgressBar := flag.Bool("withProgressBar", true, "If set to true display progress bar")
	dir1 := flag.String("dir1", "", "Path to 1st dir")
	dir2 := flag.String("dir2", "", "Path to 2nd dir")
	flag.Parse()

	log.Printf("Comparing directories: %s and %s\n", *dir1, *dir2)

	// Progress bar
	var pbCollection progressbar.ProgressBarCollection
	switch *withProgressBar {
	case true:
		pbCollection = progressbar.NewUiPCollection()
	case false:
		pbCollection = progressbar.ProgressBarCollectionMoc{}
	}

	// Comparison mode
	var comparatorFunc comparedirs.ComparisonFunc
	switch *compMode {
	// Show dupes that is present in both directories
	case "OnlyInboth":
		comparatorFunc = comparedirs.OnlyInAll
	// Show dupes that is only present in first
	case "onlyInFirst":
		comparatorFunc = comparedirs.OnlyInFirst

	case "all":
		comparatorFunc = comparedirs.All
	default:
		panic(fmt.Errorf("unknown mode: %s, supported modes: OnlyInboth, onlyInFirst, all ", *compMode))
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

	comparator := comparedirs.NewComparator(
		[]string{*dir1, *dir2}, runFunc, comparatorFunc, pbCollection,
	)

	dupes := comparator.Run()

	switch *outputJson {
	case true:
		fmt.Printf(string(dupes.GetJSON()))
	case false:
		dupes.Present(false)
	}
}
