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
		// Every width has its own case in the type switch, so every width needs
		// its own case here: a missing one is a conversion that compiles and is
		// never executed until a caller's data happens to arrive as that type.
		{42, 42},
		{int8(10), 10},
		{int16(-300), -300},
		{int32(70000), 70000},
		{int64(-5), -5},
		{uint(3), 3},
		{uint8(200), 200},
		{uint16(50000), 50000},
		{uint32(3000000000), 3000000000},
		{uint64(7), 7},
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
// A named type is still the number it is built on. The type switch in ToInt
// matches concrete types only, so these reach the reflect fallback — which must
// route each kind through the same range-checked helper the concrete case uses,
// not merely convert and hope.
func TestToInt_namedTypes(t *testing.T) {
	type Rate float32

	cases := []struct {
		name string
		in   any
		want int
	}{
		{"named int", Weight(42), 42},
		{"named int negative", Weight(-42), -42},
		{"named uint", Count(7), 7},
		{"named float truncates", Ratio(3.9), 3},
		{"named float toward zero", Ratio(-3.9), -3},
		// rv.Float widens a float32 exactly, as the concrete float32 case does.
		{"named float32", Rate(3.9), 3},
		{"named string int", Slug("17"), 17},
		{"named string float", Slug("3.9"), 3},
	}
	for _, c := range cases {
		got, err := ToInt(c.in)
		if err != nil {
			t.Errorf("%s: ToInt(%v): %v", c.name, c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: ToInt(%v) = %d, want %d", c.name, c.in, got, c.want)
		}
	}

	// The range and parse checks must survive the reflect path; converting
	// without them would wrap silently, which is the whole point of ToInt.
	bad := []struct {
		name string
		in   any
	}{
		{"named uint overflows int", Count(math.MaxUint)},
		{"named float NaN", Ratio(math.NaN())},
		{"named float infinity", Ratio(math.Inf(1))},
		{"named float out of range", Ratio(1e300)},
		{"named string non-numeric", Slug("abc")},
		{"named bool unsupported", Flag(true)},
	}
	for _, c := range bad {
		if got, err := ToInt(c.in); err == nil {
			t.Errorf("%s: ToInt(%v) = %d, want an error", c.name, c.in, got)
		}
	}
}

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

// ToInt's int64 arm guards against a value too large for an int. On a 64-bit
// platform int is int64, so the guard cannot fire and coverage cannot reach it;
// it exists for 32-bit builds, where an int64 from a JSON decoder routinely
// exceeds 2^31-1. The test is written to run there rather than deleted as dead
// code, because it is only dead on this architecture.
func TestToInt_int64OverflowGuardIs32BitOnly(t *testing.T) {
	if math.MaxInt == math.MaxInt64 {
		t.Skip("int is 64 bits here; the int64 overflow guard is unreachable by construction")
	}
	for _, in := range []int64{math.MaxInt64, math.MinInt64} {
		if got, err := ToInt(in); err == nil {
			t.Errorf("ToInt(int64 %d) = %d, want an overflow error", in, got)
		}
	}
}
