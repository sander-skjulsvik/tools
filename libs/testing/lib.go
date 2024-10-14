package testing

import (
	"fmt"
	"runtime/debug"
	"testing"
)

func ErrorfStackTrace(t *testing.T, format string, args ...any) {
	t.Errorf(format, args)
	fmt.Printf("\n%s\n", debug.Stack())
}
