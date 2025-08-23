package anki

import (
	"time"

	"github.com/google/uuid"
)

// timeZero returns a zero time.
func timeZero() time.Time {
	return time.Unix(0, 0)
}

// timeUnix returns the Unix timestamp for a given time.
func timeUnix(t time.Time) int64 {
	return max(t.Unix(), 0)
}

// randomGUID generates a random GUID.
func randomGUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// scanValue scans a single value from a database row.
func scanValue[T any](_ sqlQueryer, row sqlRow) (T, error) {
	var val T
	return val, row.Scan(&val)
}
