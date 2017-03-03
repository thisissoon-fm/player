package build

import (
	"errors"
	"strconv"
	"time"
)

// Unix epoch - -ldflags "-X player/build.timestamp=1482510310"
var timestamp string

// Error returned by Time() when timestamp is blank
var (
	ErrBlankTimestamp   = errors.New("build timestamp not set")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
)

// Returns the build time as a time.Time if not set this will return an error
func Time() (time.Time, error) {
	if timestamp == "" {
		return time.Time{}, ErrBlankTimestamp
	}
	i, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return time.Time{}, ErrInvalidTimestamp
	}
	t := time.Unix(i, 0).UTC()
	return t, nil
}

// Returns the time in string format, if the time is not set this will return n/a
func TimeStr() string {
	t, err := Time()
	if err != nil {
		return "n/a"
	}
	return t.Format("Monday January 2 2006 at 15:04:05 MST")
}
