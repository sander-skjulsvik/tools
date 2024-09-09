package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

func main() {

}

type WingetJsonManifest struct {
	Schema       string    `json:"$schema"`
	CreationDate time.Time `json:"CreationDate"`
	Sources      []struct {
		Packages []struct {
			PackageIdentifier string `json:"PackageIdentifier"`
		} `json:"Packages"`
		SourceDetails struct {
			Argument   string `json:"Argument"`
			Identifier string `json:"Identifier"`
			Name       string `json:"Name"`
			Type       string `json:"Type"`
		} `json:"SourceDetails"`
	} `json:"Sources"`
	WinGetVersion string `json:"WinGetVersion"`
}

func NewWingetJsonManifest(path string) (WingetJsonManifest, error) {
	b, err := os.ReadFile()
	if err != nil {
		return WingetJsonManifest{}, errors.Join(
			fmt.Errorf("failed to open: %s", path),
			err,
		)
	}
	var wingetJsonManifest WingetJsonManifest
	err = json.Unmarshal(b, wingetJsonManifest)
	if err != nil {
		return wingetJsonManifest, errors.Join(
			fmt.Errorf("failed on: %s", path),
			err,
		)
	}
	return wingetJsonManifest, nil
}

func (w *WingetJsonManifest) AddPackage(id string) {

}
