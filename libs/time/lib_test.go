package time

import "testing"

func TestParseTimeNoErrorRFC3339(t *testing.T) {
	s := "2006-01-02T15:04:05Z07:00"
	ParseTimeNoErrorRFC3339(s)
}
