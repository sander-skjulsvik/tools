package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
	producerConsumer "github.com/sander-skjulsvik/tools/dupes/lib/producerConsumer"
	singleThread "github.com/sander-skjulsvik/tools/dupes/lib/singleThread"
	"github.com/sander-skjulsvik/tools/libs/progressbar"
)

func main() {
	var (
		method           string
		path             string
		presentOnlyDupes bool
		useProgressBar   bool
		presentJson      bool
		nThreads         int
	)

	flag.StringVar(&method, "method", "single", "Method (single, producerConsumer or nThreads)")
	flag.IntVar(&nThreads, "nThreads", 0, "Number of threads to use, ignored unless nThreads method is chosen")
	flag.StringVar(&path, "path", ".", "File path")
	flag.BoolVar(&presentOnlyDupes, "onlyDupes", true, "Only present dupes")
	flag.BoolVar(&presentJson, "json", false, "present json")
	flag.BoolVar(&useProgressBar, "progressBar", false, "Present a progress bar?")

	// Parse the command-line arguments
	flag.Parse()

	// LowerCasing method
	method = strings.ToLower(method)

	// Check if the method is one of the allowed values
	if method != "single" && method != "producerconsumer" {
		fmt.Println("Invalid method. Allowed values are 'single' and 'producerConsumer'.")
		os.Exit(1)
	}

	// At this point, you have valid values for method and path
	fmt.Printf("Method: %s\n", method)
	fmt.Printf("Path: %s\n", path)
	fmt.Printf("PresentOnlyDupes: %t\n", presentOnlyDupes)

	var runFunc common.Run
	switch {
	case method == "single":
		runFunc = singleThread.Run
	case method == "producerConsumer":
		runFunc = producerConsumer.Run
	case method == "nThreads":
		runFunc = producerConsumer.GetRunNThreads(nThreads)
	}

	var bar progressbar.ProgressBar
	switch useProgressBar {
	case true:
		bar = progressbar.UiProgressBar{}
	}

	dupes := NewRunner(runFunc, bar).Run(path)
	switch presentOnlyDupes {
	case true:
		dupes.GetOnlyDupes().Present(presentJson)
	case false:
		dupes.Present(presentJson)
	}

}

type Runner struct {
	RunFunc     common.Run
	ProgressBar progressbar.ProgressBar
	OutputJson  bool
}

func NewRunner(runFunc common.Run, bar progressbar.ProgressBar) *Runner {
	return &Runner{
		RunFunc:     runFunc,
		ProgressBar: bar,
	}
}

func (r *Runner) Run(path string) *common.Dupes {
	return r.RunFunc(path, r.ProgressBar)
}
