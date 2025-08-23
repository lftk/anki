package anki

import (
	"time"
)

func timeZero() time.Time {
	return time.Unix(0, 0)
}

func timeUnix(t time.Time) int64 {
	return max(t.Unix(), 0)
}
