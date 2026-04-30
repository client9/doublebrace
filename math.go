package doublebrace

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"text/template"
)

func mathFuncMap() template.FuncMap {
	return template.FuncMap{
		"add":     Add,
		"sub":     Sub,
		"mul":     Mul,
		"div":     Div,
		"mod":     Mod,
		"abs":     Abs,
		"ceil":    Ceil,
		"floor":   Floor,
		"round":   Round,
		"min":     Min,
		"max":     Max,
		"pow":     Pow,
		"modBool": ModBool,
		"clamp":   Clamp,
	}
}

// Add returns a + b. Both arguments accept any numeric type or numeric string.
//
//	add 3 4   → 7
//	add 1.5 2 → 3.5
func Add(a, b any) (float64, error) {
	return applyOp(a, b, func(x, y float64) float64 { return x + y })
}

// Sub returns a - b. Both arguments accept any numeric type or numeric string.
//
//	sub 10 3 → 7
func Sub(a, b any) (float64, error) {
	return applyOp(a, b, func(x, y float64) float64 { return x - y })
}

// Mul returns a * b. Both arguments accept any numeric type or numeric string.
//
//	mul 3 4 → 12
func Mul(a, b any) (float64, error) {
	return applyOp(a, b, func(x, y float64) float64 { return x * y })
}

// Div returns a / b. Returns an error on division by zero.
// Both arguments accept any numeric type or numeric string.
//
//	div 10 4 → 2.5
func Div(a, b any) (float64, error) {
	x, err := toFloat64(a)
	if err != nil {
		return 0, err
	}
	y, err := toFloat64(b)
	if err != nil {
		return 0, err
	}
	if y == 0 {
		return 0, fmt.Errorf("div: division by zero")
	}
	return x / y, nil
}

// Mod returns the floating-point remainder of a / b (math.Mod).
// Returns an error on division by zero.
// Both arguments accept any numeric type or numeric string.
//
//	mod 10 3 → 1
func Mod(a, b any) (float64, error) {
	x, err := toFloat64(a)
	if err != nil {
		return 0, err
	}
	y, err := toFloat64(b)
	if err != nil {
		return 0, err
	}
	if y == 0 {
		return 0, fmt.Errorf("mod: division by zero")
	}
	return math.Mod(x, y), nil
}

// Abs returns the absolute value of a.
//
//	abs -7 → 7
//	abs  3 → 3
func Abs(a any) (float64, error) { return applyFunc(a, math.Abs) }

// Ceil returns the least integer value greater than or equal to a.
//
//	ceil 1.2 → 2
//	ceil 2.0 → 2
func Ceil(a any) (float64, error) { return applyFunc(a, math.Ceil) }

// Floor returns the greatest integer value less than or equal to a.
//
//	floor 1.9 → 1
//	floor 2.0 → 2
func Floor(a any) (float64, error) { return applyFunc(a, math.Floor) }

// Round returns the nearest integer, rounding half away from zero.
//
//	round 1.4 → 1
//	round 1.5 → 2
func Round(a any) (float64, error) { return applyFunc(a, math.Round) }

func applyOp(a, b any, op func(float64, float64) float64) (float64, error) {
	x, err := toFloat64(a)
	if err != nil {
		return 0, err
	}
	y, err := toFloat64(b)
	if err != nil {
		return 0, err
	}
	return op(x, y), nil
}

func applyFunc(a any, fn func(float64) float64) (float64, error) {
	x, err := toFloat64(a)
	if err != nil {
		return 0, err
	}
	return fn(x), nil
}

// Pow returns base raised to the power of exp.
//
//	Pow(2, 10) → 1024
//	Pow(9, 0.5) → 3  (square root)
func Pow(base, exp any) (float64, error) {
	return applyOp(base, exp, math.Pow)
}

// ModBool reports whether a is evenly divisible by b (a mod b == 0).
// Useful for alternating row styles: {{if modBool $i 2}}even{{end}}
//
//	ModBool(4, 2) → true
//	ModBool(5, 2) → false
func ModBool(a, b any) (bool, error) {
	x, err := toFloat64(a)
	if err != nil {
		return false, err
	}
	y, err := toFloat64(b)
	if err != nil {
		return false, err
	}
	if y == 0 {
		return false, fmt.Errorf("modBool: division by zero")
	}
	return math.Mod(x, y) == 0, nil
}

// flattenNumbers recursively flattens args, expanding any slice values, and
// converts each element to float64. Nested slices are fully unwound.
func flattenNumbers(args []any) ([]float64, error) {
	var out []float64
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		if v.IsValid() && v.Kind() == reflect.Slice {
			for i := range v.Len() {
				sub, err := flattenNumbers([]any{v.Index(i).Interface()})
				if err != nil {
					return nil, err
				}
				out = append(out, sub...)
			}
		} else {
			f, err := toFloat64(arg)
			if err != nil {
				return nil, err
			}
			out = append(out, f)
		}
	}
	return out, nil
}

// Min returns the smallest value among the given numbers.
// Accepts one or more scalars, slices, or a mix; slices are flattened recursively.
//
//	Min(3, 1, 2)              → 1
//	Min([]int{5, 2, 8})       → 2
//	Min([]int{5, 2}, 1, 9)    → 1
func Min(args ...any) (float64, error) {
	vals, err := flattenNumbers(args)
	if err != nil {
		return 0, err
	}
	if len(vals) == 0 {
		return 0, fmt.Errorf("min: no arguments")
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m, nil
}

// Max returns the largest value among the given numbers.
// Accepts one or more scalars, slices, or a mix; slices are flattened recursively.
//
//	Max(3, 1, 2)              → 3
//	Max([]int{5, 2, 8})       → 8
//	Max([]int{5, 2}, 9, 1)    → 9
func Max(args ...any) (float64, error) {
	vals, err := flattenNumbers(args)
	if err != nil {
		return 0, err
	}
	if len(vals) == 0 {
		return 0, fmt.Errorf("max: no arguments")
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v > m {
			m = v
		}
	}
	return m, nil
}

// Clamp constrains val to the range [min, max]. If val < min, min is returned;
// if val > max, max is returned; otherwise val is returned unchanged.
// All arguments accept any numeric type or numeric string.
//
//	clamp 5 1 10  → 5
//	clamp 0 1 10  → 1
//	clamp 15 1 10 → 10
func Clamp(val, minVal, maxVal any) (float64, error) {
	v, err := toFloat64(val)
	if err != nil {
		return 0, fmt.Errorf("clamp: val: %w", err)
	}
	lo, err := toFloat64(minVal)
	if err != nil {
		return 0, fmt.Errorf("clamp: min: %w", err)
	}
	hi, err := toFloat64(maxVal)
	if err != nil {
		return 0, fmt.Errorf("clamp: max: %w", err)
	}
	if lo > hi {
		return 0, fmt.Errorf("clamp: min %v > max %v", lo, hi)
	}
	return math.Max(lo, math.Min(hi, v)), nil
}

// toFloat64 converts any numeric type or numeric string to float64.
func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int8:
		return float64(n), nil
	case int16:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case uint:
		return float64(n), nil
	case uint8:
		return float64(n), nil
	case uint16:
		return float64(n), nil
	case uint32:
		return float64(n), nil
	case uint64:
		return float64(n), nil
	case string:
		return strconv.ParseFloat(n, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}
