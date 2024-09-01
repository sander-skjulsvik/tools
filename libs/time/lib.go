package time

import (
	"fmt"
	"time"
)

func ParseTimeNoError(layout string, value string) *time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic(
			fmt.Sprintf("Error parsing time: %s, with layout: %s, err: %v", value, layout, err),
		)
	}
	return &t
}

func ParseTimeNoErrorRFC3339(value string) *time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(
			fmt.Sprintf("Error parsing time: %s, err: %v", value, err),
		)
	}
	return &t
}
