package time

import (
	"fmt"
	"time"
)

var RFC3339 string = "2006-01-02T15:04:05Z07:00"

func ParseTimeNoErrorRFC3339(timeString string) *time.Time {
	t, err := time.Parse(RFC3339, timeString)
	if err != nil {
		panic(
			fmt.Sprintf("Error parsing time: %s, err: %v", timeString, err),
		)
	}
	return &t
}
