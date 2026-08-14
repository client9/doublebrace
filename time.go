package doublebrace

import (
	"fmt"
	"text/template"
	"time"
)

func timeFuncMap() template.FuncMap {
	return template.FuncMap{
		"now": Now,
		// Both arguments are text, so both go through the adapter that defines
		// what text is — see strings.go. A layout is routinely held in a config
		// struct as a named type, and the value being parsed comes from data.
		"parseTime": strFn2Err("parseTime", ParseTime),
	}
}

// Now returns the current local time.
// The returned time.Time value supports method calls in templates:
// {{now.Year}}, {{now.Format "2006-01-02"}}, etc.
func Now() time.Time {
	return time.Now()
}

// ParseTime parses a formatted string and returns the time.Time value it
// represents. The layout defines the format using Go's reference time:
// Mon Jan 2 15:04:05 MST 2006.
//
//	parseTime "2006-01-02" "2024-03-15" → 2024-03-15 00:00:00 +0000 UTC
func ParseTime(layout, s string) (time.Time, error) {
	t, err := time.Parse(layout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parseTime: %w", err)
	}
	return t, nil
}
