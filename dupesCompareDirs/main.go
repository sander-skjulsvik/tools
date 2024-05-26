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
	compMode := flag.String("mode", "", "Mode to run in, modes: onlyInboth, onlyInFirst, all")
	runnerMode := flag.String("runMode", "singleThread", "possible run modes: singleThread, producerConsumer and nThreads")
	nThreads := flag.Int("nThreads", 0, "number of threads to use, only used witt runMode nThreads")
	outputJson := flag.Bool("json", false, "If set to true Output as json")
	withProgressBar := flag.Bool("withProgressBar", true, "If set to true display progress bar")
	dir1 := flag.String("dir1", "", "Path to 1st dir")
	dir2 := flag.String("dir2", "", "Path to 2nd dir")
	flag.Parse()

	errString := ""
	if *dir1 == "" || *dir2 == "" {
		errString = "Please provide `-dir1 <path>` and `-dir2 <path>`\n" + errString
	}
	if *compMode == "" {
		errString = "Please provide `-mode <onlyInBoth|onlyInFirst|all>` flag\n" + errString
	}
	if *compMode == "nThreads" && *nThreads == 0 {
		errString = "If `-mode nThreads` please provide `-nThreads <number of threads>\n" + errString
	}
	if errString != "" {
		panic(fmt.Errorf("failed to start, error with cli flags\n%s", errString))
	}
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
