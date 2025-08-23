package anki

import (
	"time"

	"github.com/google/uuid"
)

func timeZero() time.Time {
	return time.Unix(0, 0)
}

func timeUnix(t time.Time) int64 {
	return max(t.Unix(), 0)
}

func randomGUID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func scanValue[T any](_ sqlQueryer, row sqlRow) (T, error) {
	var val T
	return val, row.Scan(&val)
}
