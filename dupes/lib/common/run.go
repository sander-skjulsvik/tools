package common

import "github.com/sander-skjulsvik/tools/libs/progressbar"

// Run is the main function to run for consumers of this lib.
// First arg is the path to the folder,
type Run func(string, progressbar.ProgressBar) *Dupes

type Runner struct {
	RunFunc     Run
	ProgressBar progressbar.ProgressBar
	OutputJson  bool
}

func NewRunner(runFunc Run, bar progressbar.ProgressBar) *Runner {
	return &Runner{
		RunFunc:     runFunc,
		ProgressBar: bar,
	}
}

func (r *Runner) Run(path string) *Dupes {
	return r.RunFunc(path, r.ProgressBar)
}
