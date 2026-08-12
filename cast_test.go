package doublebrace

import (
	"math"
	"strconv"
	"testing"
)

func TestToInt(t *testing.T) {
	cases := []struct {
		in   any
		want int
	}{
		{42, 42},
		{int8(10), 10},
		{int64(-5), -5},
		{uint(3), 3},
		{float32(3.9), 3},
		{float64(2.1), 2},
		{"17", 17},
		{"-3", -3},
	}
	for _, c := range cases {
		got, err := ToInt(c.in)
		if err != nil {
			t.Errorf("ToInt(%v): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ToInt(%v) = %d, want %d", c.in, got, c.want)
		}
	}

	if _, err := ToInt("abc"); err == nil {
		t.Error("expected error for non-numeric string")
	}
	if _, err := ToInt(true); err == nil {
		t.Error("expected error for unsupported type")
	}
}

// A value that does not fit in an int must be an error, never a wrapped or
// saturated result. Converting an out-of-range float is implementation-defined
// in Go, so an unchecked conversion returns a different wrong answer per
// architecture instead of failing.
func TestToInt_outOfRange(t *testing.T) {
	cases := []struct {
		name string
		in   any
	}{
		{"uint64 max wraps to -1 unchecked", uint64(math.MaxUint64)},
		{"uint64 one past MaxInt", uint64(math.MaxInt) + 1},
		{"uint max", uint(math.MaxUint)},
		{"float above int range", 1e30},
		{"float below int range", -1e30},
		{"float32 above int range", float32(1e30)},
		{"NaN", math.NaN()},
		{"positive infinity", math.Inf(1)},
		{"negative infinity", math.Inf(-1)},
		{"max float64", math.MaxFloat64},
		{"string above int range", "99999999999999999999999"},
		{"string infinity", "Inf"},
		{"string NaN", "NaN"},
	}
	for _, c := range cases {
		got, err := ToInt(c.in)
		if err == nil {
			t.Errorf("%s: ToInt(%v) = %d, want an error", c.name, c.in, got)
		}
	}
}

// The boundary itself must still convert: rejecting one value too many is as
// wrong as accepting one too few.
func TestToInt_boundariesAreInclusive(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want int
	}{
		{"MaxInt as int", math.MaxInt, math.MaxInt},
		{"MinInt as int", math.MinInt, math.MinInt},
		{"MaxInt as uint64", uint64(math.MaxInt), math.MaxInt},
		{"MinInt as float", float64(math.MinInt), math.MinInt},
		// The largest float64 strictly below 2^(bits-1); float64(math.MaxInt)
		// itself rounds up on 64-bit and would overflow.
		{"largest representable float", math.Nextafter(-float64(math.MinInt), 0),
			int(math.Nextafter(-float64(math.MinInt), 0))},
	}
	for _, c := range cases {
		got, err := ToInt(c.in)
		if err != nil {
			t.Errorf("%s: ToInt(%v) unexpected error: %v", c.name, c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: ToInt(%v) = %d, want %d", c.name, c.in, got, c.want)
		}
	}
}

// Strings must convert exactly like the values they spell, so that a template
// does not behave differently depending on whether YAML quoted the number.
func TestToInt_stringMatchesNumeric(t *testing.T) {
	cases := []struct {
		in         string
		want       int
		crossCheck bool // compare against ToInt of the same value parsed as a float
	}{
		{"17", 17, true},
		{"3.9", 3, true},    // matches ToInt(3.9)
		{"-3.9", -3, true},  // truncates toward zero, not floor
		{"1e3", 1000, true}, // exponent notation
		{"0", 0, true},
		// Atoi handles this exactly. ParseFloat would round it to 2^63 and
		// overflow, which is precisely why the integer parse is tried first —
		// so this is the one case where the two paths legitimately differ.
		{"9223372036854775807", math.MaxInt, false},
	}
	for _, c := range cases {
		if c.want == math.MaxInt && math.MaxInt != math.MaxInt64 {
			continue // 32-bit platform: this string genuinely overflows
		}
		got, err := ToInt(c.in)
		if err != nil {
			t.Errorf("ToInt(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ToInt(%q) = %d, want %d", c.in, got, c.want)
		}
		// The string and the number it spells must agree.
		if f, ferr := strconv.ParseFloat(c.in, 64); c.crossCheck && ferr == nil {
			numeric, nerr := ToInt(f)
			if nerr != nil {
				t.Errorf("ToInt(%v): %v", f, nerr)
				continue
			}
			if numeric != got {
				t.Errorf("ToInt(%q) = %d but ToInt(%v) = %d; string and numeric paths disagree",
					c.in, got, f, numeric)
			}
		}
	}
}

func TestToFloat(t *testing.T) {
	cases := []struct {
		in   any
		want float64
	}{
		{42, 42},
		{float32(1.5), float64(float32(1.5))},
		{float64(3.14), 3.14},
		{"3.14", 3.14},
		{"-1", -1},
	}
	for _, c := range cases {
		got, err := ToFloat(c.in)
		if err != nil {
			t.Errorf("ToFloat(%v): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ToFloat(%v) = %f, want %f", c.in, got, c.want)
		}
	}

	if _, err := ToFloat("abc"); err == nil {
		t.Error("expected error for non-numeric string")
	}
}
