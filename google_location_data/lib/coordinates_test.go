package lib

import (
	"strings"
	"testing"
)

func TestCoordDMS(t *testing.T) {
	cs := NewCoordinatesE7(1e7, 1e7*2)
	calcDMS := cs.CoordFuji()
	expectedDMS := `1 deg 0' 0.000000" N,2 deg 0' 0.000000" E`
	if !strings.EqualFold(calcDMS, expectedDMS) {
		t.Errorf("ExpectedDMS: %s  does not look like calcDMS: %s", expectedDMS, calcDMS)
	}
}
