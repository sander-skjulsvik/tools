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
	)

	flag.StringVar(&method, "method", "single", "Method (single or producerConsumer)")
	flag.StringVar(&path, "path", ".", "File path")
	flag.BoolVar(&presentOnlyDupes, "onlyDupes", true, "Only present dupes")

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

	Run(path, method, presentOnlyDupes)
}

func Run(path, method string, presentOnlyDupes bool) {
	var dupes *common.Dupes
	switch {
	case method == "single":
		dupes = singleThread.Run(path, progressbar.ProgressBarMoc{})
	case method == "producerconsumer":
		dupes = producerConsumer.Run(path, progressbar.ProgressBarMoc{})
	}
	dupes.Present(presentOnlyDupes)
}
