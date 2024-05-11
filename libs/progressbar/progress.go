package progressbar

import (
	"fmt"

	uiprogress "github.com/gosuri/uiprogress"
)

// ///////////////////////////////////
// The interfaces
// ///////////////////////////////////

type ProgressBar interface {
	Add(x int)
	Add1()
}

type ProgressBarCollection interface {
	Start()
	Stop()
	// header, size
	AddBar(string, int) ProgressBar
}

// ///////////////////////////////////
// Moc implementation
// ///////////////////////////////////

type ProgressBarCollectionMoc struct {
	bars *ProgressBarsMoc
}

type ProgressBarsMoc struct {
	bars []*ProgressBarMoc
}

type ProgressBarMoc struct {
}

func NewMocProgressBarCollection() ProgressBarCollectionMoc {
	return ProgressBarCollectionMoc{
		bars: &ProgressBarsMoc{},
	}
}

func (pbs ProgressBarCollectionMoc) AddBar(name string, total int) ProgressBar {
	return ProgressBarMoc{}
}

func (pbs ProgressBarCollectionMoc) Start() {
}

func (pbs ProgressBarCollectionMoc) Stop() {
}

func (pb ProgressBarMoc) Add(x int) {
}

func (pb ProgressBarMoc) Add1() {
}

// ///////////////////////////////////
// UiProgressBar implementation
// ///////////////////////////////////

type UiPCollection struct {
	bars       *uiProgressBars
	uiprogress *uiprogress.Progress
}

type uiProgressBars struct {
	bars []*UiProgressBar
}

type UiProgressBar struct {
	bar *uiprogress.Bar
}

func NewUiPCollection() UiPCollection {
	return UiPCollection{
		bars: &uiProgressBars{},
	}
}

// AddBar adds a bar to the progress bar instance and returns the bar index
func (uiP UiPCollection) AddBar(name string, total int) ProgressBar {
	newBar := UiProgressBar{
		bar: uiP.uiprogress.AddBar(total).AppendCompleted().PrependElapsed().PrependFunc(func(b *uiprogress.Bar) string {
			return fmt.Sprintf("%s \n    (%d/%d)Mb\t", name, b.Current(), total)
		}),
	}
	uiP.bars.bars = append(uiP.bars.bars, &newBar)
	return newBar
}

func (uiP UiPCollection) Start() {
	uiP.uiprogress.Start()
}

func (uiP UiPCollection) Stop() {
	uiP.uiprogress.Stop()
}

func (uiP UiProgressBar) Add(x int) {
	for i := 0; i < x; i++ {
		uiP.bar.Incr()
	}
}

func (uiP UiProgressBar) Add1() {
	uiP.bar.Incr()
}
