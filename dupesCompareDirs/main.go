package main

import (
	"flag"
	"fmt"
	"log"

	comparedirs "github.com/sander-skjulsvik/tools/dupesCompareDirs/lib"
	dupescomparedirs "github.com/sander-skjulsvik/tools/dupesCompareDirs/lib"
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
	pbCollection := dupescomparedirs.SelectProgressBarCollection(*withProgressBar)

	// Comparison mode
	comparatorFunc := comparedirs.SelectComparatorFunc(*compMode)

	// Runner
	runFunc := dupescomparedirs.SelectRunnerFunction(*runnerMode, *nThreads)

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
