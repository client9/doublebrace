package doublebrace

import (
	"errors"
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
		return 0, errors.New("div: division by zero")
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
		return 0, errors.New("mod: division by zero")
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
//	pow 2 10  → 1024
//	pow 9 0.5 → 3     (square root)
func Pow(base, exp any) (float64, error) {
	return applyOp(base, exp, math.Pow)
}

// ModBool reports whether a is evenly divisible by b (a mod b == 0).
// Useful for alternating row styles: {{if modBool $i 2}}even{{end}}
//
//	modBool 4 2 → true
//	modBool 5 2 → false
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
		return false, errors.New("modBool: division by zero")
	}
	return math.Mod(x, y) == 0, nil
}

// maxFlattenDepth bounds how deeply min and max descend into nested sequences.
// It is the guardrail MaxSeqLen and MaxRepeatLen are, for the one input that
// reaches this package as a shape rather than as a count: a sequence that
// contains itself.
//
//	a := make([]any, 1)
//	a[0] = a
//
// The descent had no bound, and what it produced was not an error. It recursed
// until the goroutine stack gave out, and a Go stack overflow is a fatal error
// rather than a panic — text/template's recover cannot turn it into an
// execution error, and nothing else can catch it either. It ends the process,
// which in the server this package is written for takes every other request in
// flight with it.
//
// Only Go data can be that shape; template syntax has no way to build a cycle.
// That is what makes 100 generous rather than restrictive: nesting in real
// template data runs two or three deep, and list (list ...) has to be typed out
// a level at a time.
//
// Unlike the other two limits this one is unexported. Those bound a result a
// legitimate template can approach, so an author needs the number to write
// against; this bounds a shape no correct input has, so there is nothing to
// tune. A caller who hits it has a cycle, not a large document.
//
// The fmt.Sprint in Sort's lexicographic mode is the same descent and is
// deliberately left alone: fmt overflows on a cyclic value on its own, so a
// bare {{ . }} with no functions registered already ends the process the same
// way. That is a property of rendering such a value at all, not something this
// package introduces or can meaningfully guard.
const maxFlattenDepth = 100

// flattenNumbers flattens args, expanding any slice or array values, and
// converts each element to float64. Nesting is unwound to maxFlattenDepth.
//
// isSequence lives in collections.go but is the package-wide definition of an
// indexable sequence; using it here is what keeps min and max accepting the same
// inputs the collection functions do.
func flattenNumbers(args []any) ([]float64, error) {
	var out []float64
	for _, arg := range args {
		var err error
		if out, err = appendNumber(out, arg, 0); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// appendNumber appends arg to out, expanding a sequence element by element.
// depth counts the sequences entered so far, so a scalar argument is at 0.
//
// Recursing on one value rather than on a []any holding it is what lets the
// result accumulate into a single slice. The version that called flattenNumbers
// back allocated a one-element []any and a fresh []float64 for every element at
// every level, then copied each one into its parent.
func appendNumber(out []float64, arg any, depth int) ([]float64, error) {
	v := reflect.ValueOf(arg)
	if !v.IsValid() || !isSequence(v.Kind()) {
		f, err := toFloat64(arg)
		if err != nil {
			return nil, err
		}
		return append(out, f), nil
	}
	if depth >= maxFlattenDepth {
		return nil, fmt.Errorf("nested sequences exceed the depth limit of %d", maxFlattenDepth)
	}
	for i := range v.Len() {
		var err error
		if out, err = appendNumber(out, v.Index(i).Interface(), depth+1); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Min returns the smallest value among the given numbers.
// Accepts one or more scalars, slices, arrays, or a mix; sequences are flattened
// recursively.
//
//	min 3 1 2       → 1
//	min $nums       → 2    ($nums is a slice or array of numbers)
//	min $nums 1 9   → 1
func Min(args ...any) (float64, error) {
	vals, err := flattenNumbers(args)
	if err != nil {
		return 0, err
	}
	if len(vals) == 0 {
		return 0, errors.New("min: no arguments")
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
// Accepts one or more scalars, slices, arrays, or a mix; sequences are flattened
// recursively.
//
//	max 3 1 2       → 3
//	max $nums       → 8    ($nums is a slice or array of numbers)
//	max $nums 9 1   → 9
func Max(args ...any) (float64, error) {
	vals, err := flattenNumbers(args)
	if err != nil {
		return 0, err
	}
	if len(vals) == 0 {
		return 0, errors.New("max: no arguments")
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
//
// The type switch below is a fast path for the concrete types template data
// usually arrives as; it is not the definition of what counts as numeric.
// Named types (type Weight int) do not match a concrete case, so anything that
// falls through is classified by reflect kind instead — the same rule asNumber,
// isZero, and In already apply. Without that fallback a template could compare
// a Weight with in but not add to it, and sort would silently order it as text.
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
	}
	rv := reflect.ValueOf(v)
	if rv.IsValid() {
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return float64(rv.Int()), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return float64(rv.Uint()), nil
		case reflect.Float32, reflect.Float64:
			return rv.Float(), nil
		case reflect.String:
			return strconv.ParseFloat(rv.String(), 64)
		}
	}
	return 0, fmt.Errorf("cannot convert %T to float64", v)
}
