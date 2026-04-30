package doublebrace

import (
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"text/template"
	"time"
)

func collectionsFuncMap() template.FuncMap {
	return template.FuncMap{
		// Constructors
		"list": List,
		"dict": Dict,
		"seq":  Seq,
		// Sequence access
		"first": First,
		"last":  Last,
		"take":  Take,
		"drop":  Drop,
		// Sequence transformation
		"reverse": Reverse,
		"compact": Compact,
		"concat":  Concat,
		"sort":    Sort,
		"sortNum": SortNum,
		"where":   Where,
		// Map operations
		"keys":   Keys,
		"values": Values,
		"merge":  MergeMaps,
		// General
		"in":      In,
		"default": Default,
		"cond":    Cond,
	}
}

// --- internal helpers ---

// toSlice converts any slice type to []any. []any is returned directly (fast
// path); other slice types are expanded via reflection.
func toSlice(v any) ([]any, error) {
	if s, ok := v.([]any); ok {
		return s, nil
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("expected slice, got %T", v)
	}
	out := make([]any, rv.Len())
	for i := range rv.Len() {
		out[i] = rv.Index(i).Interface()
	}
	return out, nil
}

// fieldString returns the string representation of a named field in a
// map[string]any element. Returns "" if the element is not a map or the key
// is absent.
func fieldString(v any, key string) string {
	if m, ok := v.(map[string]any); ok {
		if val, exists := m[key]; exists {
			return fmt.Sprint(val)
		}
	}
	return ""
}

// fieldFloat returns the float64 value of a named field in a map[string]any element.
func fieldFloat(v any, key string) (float64, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("element is not map[string]any, got %T", v)
	}
	val, exists := m[key]
	if !exists {
		return 0, fmt.Errorf("field %q not found", key)
	}
	return toFloat64(val)
}

// isZero reports whether v is the zero value for its type.
// nil, false, 0, "", and empty slices/maps are all considered zero.
func isZero(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.String:
		return rv.String() == ""
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return rv.IsNil()
	}
	return false
}

// --- constructors ---

// List creates a []any from the given values.
//
//	list "a" "b" "c" → []any{"a", "b", "c"}
func List(elems ...any) []any {
	if elems == nil {
		return []any{}
	}
	return elems
}

// Dict creates a map[string]any from alternating key-value arguments.
// Returns an error if the argument count is odd or a key is not a string.
//
//	dict "name" "Alice" "age" 30 → map[string]any{"name": "Alice", "age": 30}
func Dict(kvs ...any) (map[string]any, error) {
	if len(kvs)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments (%d)", len(kvs))
	}
	m := make(map[string]any, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		k, ok := kvs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key at position %d must be string, got %T", i, kvs[i])
		}
		m[k] = kvs[i+1]
	}
	return m, nil
}

// Seq returns a slice of integers. Counting is 1-based by default.
//
//	seq 5        → [1 2 3 4 5]
//	seq 3 7      → [3 4 5 6 7]
//	seq 1 10 2   → [1 3 5 7 9]
//	seq 5 1 -1   → [5 4 3 2 1]
func Seq(args ...int) ([]int, error) {
	switch len(args) {
	case 1:
		n := args[0]
		if n < 1 {
			return []int{}, nil
		}
		out := make([]int, n)
		for i := range n {
			out[i] = i + 1
		}
		return out, nil
	case 2:
		start, end := args[0], args[1]
		if start > end {
			return []int{}, nil
		}
		out := make([]int, end-start+1)
		for i := range out {
			out[i] = start + i
		}
		return out, nil
	case 3:
		start, end, step := args[0], args[1], args[2]
		if step == 0 {
			return nil, fmt.Errorf("seq: step cannot be zero")
		}
		var out []int
		if step > 0 {
			for v := start; v <= end; v += step {
				out = append(out, v)
			}
		} else {
			for v := start; v >= end; v += step {
				out = append(out, v)
			}
		}
		if out == nil {
			return []int{}, nil
		}
		return out, nil
	default:
		return nil, fmt.Errorf("seq: expected 1–3 arguments, got %d", len(args))
	}
}

// --- sequence access ---

// First returns the first element of a slice, or the first rune of a string.
//
//	first []int{1, 2, 3} → 1
//	first "café"         → "c"
func First(v any) (any, error) {
	if s, ok := v.(string); ok {
		r := []rune(s)
		if len(r) == 0 {
			return nil, fmt.Errorf("first: empty string")
		}
		return string(r[0]), nil
	}
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("first: %w", err)
	}
	if len(elems) == 0 {
		return nil, fmt.Errorf("first: empty slice")
	}
	return elems[0], nil
}

// Last returns the last element of a slice, or the last rune of a string.
//
//	last []int{1, 2, 3} → 3
//	last "café"         → "é"
func Last(v any) (any, error) {
	if s, ok := v.(string); ok {
		r := []rune(s)
		if len(r) == 0 {
			return nil, fmt.Errorf("last: empty string")
		}
		return string(r[len(r)-1]), nil
	}
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("last: %w", err)
	}
	if len(elems) == 0 {
		return nil, fmt.Errorf("last: empty slice")
	}
	return elems[len(elems)-1], nil
}

// Take returns the first n elements of a slice, or the first n runes of a
// string. If n is negative, it returns the last |n| elements or runes.
// If |n| exceeds the length the full input is returned.
// Rune-aware for strings: multi-byte characters are not split.
//
//	take []int{1, 2, 3, 4, 5} 3  → []any{1, 2, 3}
//	take []int{1, 2, 3, 4, 5} -2 → []any{4, 5}
//	take "日本語" 2                → "日本"
//	take "日本語" -1               → "語"
func Take(v any, n int) (any, error) {
	if s, ok := v.(string); ok {
		r := []rune(s)
		if n >= 0 {
			if n > len(r) {
				n = len(r)
			}
			return string(r[:n]), nil
		}
		// negative: last |n| runes
		start := len(r) + n
		if start < 0 {
			start = 0
		}
		return string(r[start:]), nil
	}
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("take: %w", err)
	}
	if n >= 0 {
		if n > len(elems) {
			n = len(elems)
		}
		return elems[:n], nil
	}
	// negative: last |n| elements
	start := len(elems) + n
	if start < 0 {
		start = 0
	}
	return elems[start:], nil
}

// Drop skips the first n elements of a slice, or the first n runes of a
// string. If n is negative, it removes the last |n| elements or runes.
// If |n| exceeds the length an empty result is returned.
// Rune-aware for strings: multi-byte characters are not split.
//
//	drop []int{1, 2, 3, 4, 5} 2  → []any{3, 4, 5}
//	drop []int{1, 2, 3, 4, 5} -2 → []any{1, 2, 3}
//	drop "日本語" 1                → "本語"
//	drop "日本語" -1               → "日本"
func Drop(v any, n int) (any, error) {
	if s, ok := v.(string); ok {
		r := []rune(s)
		if n >= 0 {
			if n > len(r) {
				n = len(r)
			}
			return string(r[n:]), nil
		}
		// negative: remove last |n| runes
		end := len(r) + n
		if end < 0 {
			end = 0
		}
		return string(r[:end]), nil
	}
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("drop: %w", err)
	}
	if n >= 0 {
		if n > len(elems) {
			n = len(elems)
		}
		return elems[n:], nil
	}
	// negative: remove last |n| elements
	end := len(elems) + n
	if end < 0 {
		end = 0
	}
	return elems[:end], nil
}

// --- sequence transformation ---

// Reverse returns a new slice with the elements in reverse order.
//
//	reverse []int{1, 2, 3} → []any{3, 2, 1}
func Reverse(v any) (any, error) {
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("reverse: %w", err)
	}
	out := slices.Clone(elems)
	slices.Reverse(out)
	return out, nil
}

// Compact removes consecutive duplicate elements, identical to slices.Compact
// semantics. For full deduplication use: compact (sort $list)
//
//	compact []int{1, 1, 2, 3, 3, 1} → []any{1, 2, 3, 1}
func Compact(v any) (any, error) {
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("compact: %w", err)
	}
	return slices.CompactFunc(elems, reflect.DeepEqual), nil
}

// Concat concatenates multiple slices into a single []any.
//
//	concat (list 1 2) (list 3 4) → []any{1, 2, 3, 4}
func Concat(ins ...any) ([]any, error) {
	var out []any
	for i, v := range ins {
		elems, err := toSlice(v)
		if err != nil {
			return nil, fmt.Errorf("concat: argument %d: %w", i, err)
		}
		out = append(out, elems...)
	}
	return out, nil
}

type sortMode int

const (
	sortLex     sortMode = iota
	sortNumeric sortMode = iota
	sortTime    sortMode = iota
)

// inferSortMode inspects the first non-nil element of elems to decide how to sort.
func inferSortMode(elems []any) sortMode {
	for _, e := range elems {
		if e == nil {
			continue
		}
		switch e.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			return sortNumeric
		case time.Time:
			return sortTime
		}
		return sortLex
	}
	return sortLex
}

// Sort returns a new slice sorted by type:
//   - numeric types (int, float64, etc.) sort numerically
//   - time.Time values sort chronologically
//   - everything else sorts lexicographically (string comparison)
//
// For []any, the first non-nil element determines the sort mode.
// An optional key names a field for slice-of-maps sorting (always lexicographic).
// For descending order, compose with reverse.
// ISO 8601 date strings ("2006-01-02") sort correctly lexicographically.
//
//	sort (list "banana" "apple" "cherry") → ["apple" "banana" "cherry"]
//	sort (list 10 2 30)                   → [2 10 30]
//	sort $pages "Title"                   → pages A→Z by Title field
func Sort(v any, key ...string) (any, error) {
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("sort: %w", err)
	}
	out := slices.Clone(elems)
	if len(key) > 0 {
		k := key[0]
		slices.SortStableFunc(out, func(a, b any) int {
			return strings.Compare(fieldString(a, k), fieldString(b, k))
		})
		return out, nil
	}
	var sortErr error
	switch inferSortMode(out) {
	case sortNumeric:
		slices.SortStableFunc(out, func(a, b any) int {
			fa, ea := toFloat64(a)
			fb, eb := toFloat64(b)
			if ea != nil || eb != nil {
				sortErr = fmt.Errorf("sort: cannot convert element to number")
				return 0
			}
			return cmp.Compare(fa, fb)
		})
	case sortTime:
		slices.SortStableFunc(out, func(a, b any) int {
			ta, oka := a.(time.Time)
			tb, okb := b.(time.Time)
			if !oka || !okb {
				sortErr = fmt.Errorf("sort: element is not time.Time")
				return 0
			}
			return ta.Compare(tb)
		})
	default:
		slices.SortStableFunc(out, func(a, b any) int {
			return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
		})
	}
	if sortErr != nil {
		return nil, sortErr
	}
	return out, nil
}

// SortNum returns a new slice sorted numerically using toFloat64 conversion.
// An optional key names a field for slice-of-maps sorting.
// For descending order, compose with reverse.
//
//	sortNum (list "10" "9" "2") → ["2" "9" "10"]
//	sortNum $pages "Year"       → pages sorted by Year field, ascending
func SortNum(v any, key ...string) (any, error) {
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("sortNum: %w", err)
	}
	out := slices.Clone(elems)
	var sortErr error
	if len(key) > 0 {
		k := key[0]
		slices.SortStableFunc(out, func(a, b any) int {
			fa, ea := fieldFloat(a, k)
			fb, eb := fieldFloat(b, k)
			if ea != nil || eb != nil {
				sortErr = fmt.Errorf("sortNum: cannot convert field %q to number", k)
				return 0
			}
			return cmp.Compare(fa, fb)
		})
	} else {
		slices.SortStableFunc(out, func(a, b any) int {
			fa, ea := toFloat64(a)
			fb, eb := toFloat64(b)
			if ea != nil || eb != nil {
				sortErr = fmt.Errorf("sortNum: cannot convert value to number")
				return 0
			}
			return cmp.Compare(fa, fb)
		})
	}
	if sortErr != nil {
		return nil, sortErr
	}
	return out, nil
}

// Where filters a slice of map[string]any by field equality.
// Only elements where element[key] == val are included in the result.
//
//	where $pages "Draft" false    → pages where Draft == false
//	where $pages "Section" "blog" → pages in the blog section
func Where(v any, key string, val any) (any, error) {
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("where: %w", err)
	}
	out := make([]any, 0, len(elems))
	for _, elem := range elems {
		m, ok := elem.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("where: elements must be map[string]any, got %T", elem)
		}
		if reflect.DeepEqual(m[key], val) {
			out = append(out, elem)
		}
	}
	return out, nil
}

// --- map operations ---

// Keys returns the keys of a map[string]any in sorted order.
//
//	keys map[string]any{"b": 2, "a": 1} → ["a" "b"]
func Keys(v any) ([]string, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("keys: expected map[string]any, got %T", v)
	}
	return slices.Sorted(maps.Keys(m)), nil
}

// Values returns the values of a map[string]any in key-sorted order.
//
//	values map[string]any{"b": 2, "a": 1} → [1 2]
func Values(v any) ([]any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("values: expected map[string]any, got %T", v)
	}
	ks := slices.Sorted(maps.Keys(m))
	out := make([]any, len(ks))
	for i, k := range ks {
		out[i] = m[k]
	}
	return out, nil
}

// MergeMaps combines map[string]any maps into a new map. Later maps win on
// key collision. Registered as "merge" in the template FuncMap.
//
//	merge (dict "a" 1 "b" 2) (dict "b" 99 "c" 3) → {"a":1, "b":99, "c":3}
func MergeMaps(mapsIn ...any) (map[string]any, error) {
	out := make(map[string]any)
	for i, v := range mapsIn {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("merge: argument %d must be map[string]any, got %T", i, v)
		}
		maps.Copy(out, m)
	}
	return out, nil
}

// --- general ---

// In reports whether val is present in v.
//
//   - slice: element membership via reflect.DeepEqual
//
//   - map[string]any: key existence (val must be string)
//
//   - string: substring search (val must be string)
//
//     in (list "a" "b" "c") "b"          → true
//     in (dict "x" 1) "x"                → true
//     in "hello world" "world"            → true
func In(v, val any) (bool, error) {
	if v == nil {
		return false, nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		s, ok := val.(string)
		if !ok {
			return false, fmt.Errorf("in: string search requires string value, got %T", val)
		}
		return strings.Contains(rv.String(), s), nil
	case reflect.Slice:
		elems, err := toSlice(v)
		if err != nil {
			return false, err
		}
		for _, elem := range elems {
			if reflect.DeepEqual(elem, val) {
				return true, nil
			}
		}
		return false, nil
	case reflect.Map:
		s, ok := val.(string)
		if !ok {
			return false, fmt.Errorf("in: map key search requires string value, got %T", val)
		}
		return rv.MapIndex(reflect.ValueOf(s)).IsValid(), nil
	default:
		return false, fmt.Errorf("in: unsupported type %T", v)
	}
}

// Default returns val if it is non-zero, otherwise def.
// Zero values: nil, false, 0, "", and empty slices/maps.
//
//	default "anon" ""      → "anon"
//	default "anon" "Alice" → "Alice"
//	default 0 42           → 42
func Default(def, val any) any {
	if isZero(val) {
		return def
	}
	return val
}

// Cond returns a if ctrl is truthy (non-zero), otherwise b.
//
//	cond true  "yes" "no" → "yes"
//	cond false "yes" "no" → "no"
//	cond ""    "yes" "no" → "no"
//	cond 1     "yes" "no" → "yes"
func Cond(ctrl, a, b any) any {
	if !isZero(ctrl) {
		return a
	}
	return b
}
