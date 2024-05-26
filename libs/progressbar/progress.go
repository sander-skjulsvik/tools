package progressbar

import (
	"fmt"
	"log"
	"os"
	"time"

	uiprogress "github.com/gosuri/uiprogress"
	"github.com/sander-skjulsvik/tools/libs/files"
)

// ///////////////////////////////////
// The interfaces
// ///////////////////////////////////

type ProgressBar interface {
	Add(x int)
	// AddFileSize(path)
	AddFileSize(string)
	Add1()
}

type ProgressBarCollection interface {
	Start()
	Stop()
	// header, size
	AddBar(string, int) ProgressBar
	// path
	AddDirectorySizeBar(string) ProgressBar
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

func (pbs ProgressBarCollectionMoc) AddDirectorySizeBar(path string) ProgressBar {
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

func (pb ProgressBarMoc) AddFileSize(string) {
}

// ///////////////////////////////////
// UiProgressBar implementation
// ///////////////////////////////////

type UiPCollection struct {
	bars     *uiProgressBars
	progress *uiprogress.Progress
}

type uiProgressBars struct {
	bars []*UiProgressBar
}

type UiProgressBar struct {
	bar *uiprogress.Bar
}

func NewUiPCollection() UiPCollection {
	progress := uiprogress.New()
	progress.SetRefreshInterval(time.Millisecond * 10)
	return UiPCollection{
		progress: progress,
		bars:     &uiProgressBars{},
	}
}

// AddBar adds a bar to the progress bar instance and returns the bar index
func (uiP UiPCollection) AddBar(name string, total int) ProgressBar {
	newBar := UiProgressBar{
		bar: uiP.progress.AddBar(total).AppendCompleted().PrependElapsed().PrependFunc(func(b *uiprogress.Bar) string {
			return fmt.Sprintf("%s \n    (%d/%d)Mb\t", name, b.Current(), total)
		}),
	}
	uiP.bars.bars = append(uiP.bars.bars, &newBar)
	return newBar
}

func (uiP UiPCollection) AddDirectorySizeBar(path string) ProgressBar {
	log.Printf("Getting size of dir for bar: %s", path)
	dirSize, err := files.GetSizeOfDirMb(path)
	if err != nil {
		panic(fmt.Errorf("unable to determine directory size: %w", err))
	}
	return uiP.AddBar(path, dirSize)
}

func (uiP UiPCollection) Start() {
	uiP.progress.Start()
}

func (uiP UiPCollection) Stop() {
	uiP.progress.Stop()
}

func (uiP UiProgressBar) Add(x int) {
	for i := 0; i < x; i++ {
		uiP.bar.Incr()
	}
}

func (uiP UiProgressBar) Add1() {
	uiP.bar.Incr()
}

func (uiP UiProgressBar) AddFileSize(path string) {
	// Get the fileinfo
	fileInfo, err := os.Stat(path)
	if err != nil {
		panic(fmt.Errorf("addFileSize failed for: %s", path))
	}
	uiP.Add(int(fileInfo.Size()))
}
