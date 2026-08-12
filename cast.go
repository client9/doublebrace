package doublebrace

import (
	"fmt"
	"math"
	"strconv"
	"text/template"
)

func castFuncMap() template.FuncMap {
	return template.FuncMap{
		"toInt":   ToInt,
		"toFloat": ToFloat,
	}
}

// ToInt converts v to int. Numeric types are converted directly, with floats
// truncated toward zero. Numeric strings are parsed as an integer and, failing
// that, as a float, so "17" and "17.9" behave like 17 and 17.9 do.
//
// A value that does not fit in an int is an error rather than a silently wrapped
// or saturated result: NaN, ±Inf, and anything outside [math.MinInt, math.MaxInt]
// are all rejected.
//
//	toInt 42        → 42
//	toInt 3.9       → 3
//	toInt "17"      → 17
//	toInt "3.9"     → 3
func ToInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int8:
		return int(n), nil
	case int16:
		return int(n), nil
	case int32:
		// int is at least 32 bits, so int8/int16/int32 always fit.
		return int(n), nil
	case int64:
		if n > int64(math.MaxInt) || n < int64(math.MinInt) {
			return 0, fmt.Errorf("toInt: %d overflows int", n)
		}
		return int(n), nil
	case uint8:
		return int(n), nil
	case uint16:
		// uint8 and uint16 always fit; uint32 does not on a 32-bit platform.
		return int(n), nil
	case uint32:
		return intFromUint64(uint64(n))
	case uint:
		return intFromUint64(uint64(n))
	case uint64:
		return intFromUint64(n)
	case float32:
		return intFromFloat64(float64(n))
	case float64:
		return intFromFloat64(n)
	case string:
		// Atoi first: ParseFloat would lose precision above 2^53, so a string
		// holding a large exact integer must not go through the float path.
		if i, err := strconv.Atoi(n); err == nil {
			return i, nil
		}
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, fmt.Errorf("toInt: %w", err)
		}
		return intFromFloat64(f)
	default:
		return 0, fmt.Errorf("toInt: cannot convert %T", v)
	}
}

// intFromUint64 converts u to int, reporting an error rather than wrapping
// around when it does not fit.
func intFromUint64(u uint64) (int, error) {
	if u > uint64(math.MaxInt) {
		return 0, fmt.Errorf("toInt: %d overflows int", u)
	}
	return int(u), nil
}

// intFromFloat64 truncates f toward zero, reporting an error when the result
// would not fit in an int. Converting an out-of-range float is
// implementation-defined in Go — it yields a different wrong answer per
// architecture rather than panicking — so the range must be checked first.
//
// The upper bound is written as -float64(math.MinInt) rather than the more
// obvious float64(math.MaxInt): it is exactly 2^(bits-1), one past the largest
// int, and is exactly representable on both 32- and 64-bit platforms.
// float64(math.MaxInt) is not — on 64-bit it rounds up to 2^63, making a >=
// comparison correct there but off by one on 32-bit, where it is exact.
func intFromFloat64(f float64) (int, error) {
	const overMax = -float64(math.MinInt)
	if math.IsNaN(f) || f >= overMax || f < float64(math.MinInt) {
		return 0, fmt.Errorf("toInt: %v is not representable as int", f)
	}
	return int(f), nil
}

// ToFloat converts v to float64. Numeric types are converted directly.
// Numeric strings are parsed with strconv.ParseFloat.
//
//	toFloat 42      → 42
//	toFloat "3.14"  → 3.14
func ToFloat(v any) (float64, error) {
	f, err := toFloat64(v)
	if err != nil {
		return 0, fmt.Errorf("toFloat: %w", err)
	}
	return f, nil
}
