package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
	comparedirs "github.com/sander-skjulsvik/tools/dupesCompareDirs/lib"
	"github.com/sander-skjulsvik/tools/libs/progressbar"
)

func main() {
	// Define command-line flags
	mode := flag.String("mode", "all", "Mode to run in, modes: OnlyInboth, onlyInFirst, all")
	outputJson := flag.Bool("json", false, "If set to true Output as json")
	dir1 := flag.String("dir1", "", "Path to 1st dir")
	dir2 := flag.String("dir2", "", "Path to 2nd dir")
	flag.Parse()

	log.Printf("Comparing directories: %s and %s\n", *dir1, *dir2)

	// Progress bar
	pbs := progressbar.NewUiPCollection()

	var newD *common.Dupes
	switch *mode {
	// Show dupes that is present in both directories
	case "OnlyInboth":
		newD = comparedirs.OnlyInAll(pbs, *dir1, *dir2)
	// Show dupes that is only present in first
	case "onlyInFirst":
		newD = comparedirs.OnlyInFirst(pbs, *dir1, *dir2)
		log.Println("Only in first")
		log.Printf("Number of dupes: %d\n", len(newD.D))
	case "all":
		newD = comparedirs.All(pbs, *dir1, *dir2)
	default:
		panic(fmt.Errorf("unknown mode: %s, supported modes: OnlyInboth, onlyInFirst, all ", *mode))
	}

	log.Printf("Sander")
	if *outputJson {
		fmt.Println(string(newD.GetJSON()))
	} else {
		newD.Present(false)
	}
}
