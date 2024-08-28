package lib

import (
	"strings"
	"testing"
)

func TestCoordDMS(t *testing.T) {
	cs := NewCoordinatesE7(1e7, 1e7*2)
	calcDMS := cs.CoordDMS()
	expectedDMS := `1° 0' 0.000000",2° 0' 0.000000"`
	if !strings.EqualFold(calcDMS, expectedDMS) {
		t.Errorf("ExpectedDMS: %s  does not look like calcDMS: %s", expectedDMS, calcDMS)
	}
}
