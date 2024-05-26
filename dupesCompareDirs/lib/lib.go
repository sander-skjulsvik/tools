package dupescomparedirs

import (
	"fmt"
	"log"
	"sync"

	"github.com/sander-skjulsvik/tools/dupes/lib/common"
	"github.com/sander-skjulsvik/tools/libs/progressbar"
)

type ComparisonFunc func([]*common.Dupes) *common.Dupes

// OnlyInAll returns dupes that is present in all directories
func OnlyInAll(inDupes []*common.Dupes) *common.Dupes {
	if len(inDupes) == 0 {
		panic(fmt.Errorf("onlyInAll was given empty inDupes"))
	}
	outDupes := common.NewDupes()
	outDupes.AppendDupes(inDupes[0])
	for _, d := range inDupes[1:] {
		outDupes = *outDupes.OnlyInBoth(d)
	}

	return &outDupes
}

// OnlyInFirst returns dupes that is only present in first directory
func OnlyInFirst(inDupes []*common.Dupes) *common.Dupes {
	if len(inDupes) == 0 {
		panic(fmt.Errorf("onlyInFirst was given empty inDupes"))
	}
	outDupes := common.NewDupes()
	outDupes.AppendDupes(inDupes[0])
	for _, d := range inDupes[1:] {
		outDupes = *outDupes.OnlyInSelf(d)
	}
	return &outDupes
}

// All returns all dupes in all directories
func All(inDupes []*common.Dupes) *common.Dupes {
	if len(inDupes) == 0 {
		panic(fmt.Errorf("all was given empty inDupes"))
	}
	outDupes := common.NewDupes()
	for _, dupe := range inDupes {
		outDupes.AppendDupes(dupe)
	}
	return &outDupes
}

type Comparator struct {
	DupesRunners          []*common.Runner
	CompFunc              ComparisonFunc
	ProgressBarCollection progressbar.ProgressBarCollection
	paths                 []string
}

func NewComparator(paths []string, runFunc common.Run, compFunc ComparisonFunc, barCollection progressbar.ProgressBarCollection) *Comparator {
	runners := []*common.Runner{}
	for _, path := range paths {
		runners = append(runners, common.NewRunner(
			runFunc,
			barCollection.AddDirectorySizeBar(path),
		))
	}

	return &Comparator{
		DupesRunners:          runners,
		CompFunc:              compFunc,
		ProgressBarCollection: barCollection,
		paths:                 paths,
	}
}

func (compr *Comparator) Run() *common.Dupes {
	wg := sync.WaitGroup{}
	wg.Add(len(compr.paths))
	dupesCollection := make([]*common.Dupes, len(compr.paths))

	compr.ProgressBarCollection.Start()

	for ind, path := range compr.paths {
		go func() {
			defer wg.Done()
			log.Printf("Running dupes on: %s", path)
			dupesCollection[ind] = compr.DupesRunners[ind].Run(path)
		}()
	}
	wg.Wait()
	compr.ProgressBarCollection.Stop()

	return compr.CompFunc(dupesCollection)
}
