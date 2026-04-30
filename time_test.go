package doublebrace

import (
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	before := time.Now()
	got := Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, want between %v and %v", got, before, after)
	}
}

func TestParseTime(t *testing.T) {
	cases := []struct {
		layout string
		s      string
		want   time.Time
	}{
		{"2006-01-02", "2024-03-15", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
		{"2006-01-02 15:04:05", "2023-12-31 23:59:59", time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)},
		{"January 2, 2006", "March 15, 2024", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got, err := ParseTime(c.layout, c.s)
		if err != nil {
			t.Errorf("ParseTime(%q, %q): %v", c.layout, c.s, err)
			continue
		}
		if !got.Equal(c.want) {
			t.Errorf("ParseTime(%q, %q) = %v, want %v", c.layout, c.s, got, c.want)
		}
	}

	if _, err := ParseTime("2006-01-02", "not-a-date"); err == nil {
		t.Error("expected error for invalid date string")
	}
}
