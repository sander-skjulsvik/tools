package time

import (
	"errors"
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

func GetCEST() *time.Location {
	cest, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(errors.Join(
			fmt.Errorf("failed to get CEST"),
			err,
		))
	}
	return cest
}

func GetMidpointByWeights(t1, t2 time.Time, weight float64) time.Time {
	return t1.Add(time.Duration(int64(float64(t2.Sub(t1)) * weight)))
}
