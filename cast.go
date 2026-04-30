package doublebrace

import (
	"fmt"
	"strconv"
	"text/template"
)

func castFuncMap() template.FuncMap {
	return template.FuncMap{
		"toInt":   ToInt,
		"toFloat": ToFloat,
	}
}

// ToInt converts v to int. Numeric types are converted directly (floats are
// truncated toward zero). Numeric strings are parsed with strconv.Atoi.
//
//	toInt 42        → 42
//	toInt 3.9       → 3
//	toInt "17"      → 17
func ToInt(v any) (int, error) {
	switch n := v.(type) {
	case int:
		return n, nil
	case int8:
		return int(n), nil
	case int16:
		return int(n), nil
	case int32:
		return int(n), nil
	case int64:
		return int(n), nil
	case uint:
		return int(n), nil
	case uint8:
		return int(n), nil
	case uint16:
		return int(n), nil
	case uint32:
		return int(n), nil
	case uint64:
		return int(n), nil
	case float32:
		return int(n), nil
	case float64:
		return int(n), nil
	case string:
		i, err := strconv.Atoi(n)
		if err != nil {
			return 0, fmt.Errorf("toInt: %w", err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("toInt: cannot convert %T", v)
	}
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
