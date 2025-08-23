package anki

import (
	"testing"
	"time"
)

// TestTimeZero tests the timeZero function.
func TestTimeZero(t *testing.T) {
	got := timeZero()
	if got.Unix() != 0 {
		t.Errorf("timeZero() = %v, want Unix time 0", got)
	}
}

// TestTimeUnix tests the timeUnix function.
func TestTimeUnix(t *testing.T) {
	tests := []struct {
		name string
		time int64
		want int64
	}{
		{"positive time", 1000, 1000},
		{"zero time", 0, 0},
		{"negative time", -1000, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeUnix(time.Unix(tt.time, 0))
			if got != tt.want {
				t.Errorf("timeUnix(%v) = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}
