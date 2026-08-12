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

// toSlice converts any slice type to []any. []any is cloned; other slice types
// are expanded via reflection. The result is never nil, only ever empty, so that
// every function built on it inherits that guarantee — see the note on empty
// results below.
//
// The clone is deliberate and load-bearing: collection functions must never
// mutate or alias their inputs (see the package doc on concurrent template
// execution), and allocating here is what makes that guarantee hold for every
// caller rather than depending on each one to remember. The copy is shallow —
// elements are shared, only the spine is fresh.
//
// Returning an empty slice rather than nil is likewise load-bearing. Templates
// cannot tell the two apart (range, len, and index treat them identically), but
// encoding/json can: jsonify on a nil slice emits null instead of [], which
// breaks any script consuming the embedded value.
func toSlice(v any) ([]any, error) {
	if s, ok := v.([]any); ok {
		if s == nil {
			// slices.Clone(nil) is nil; see the non-nil invariant above.
			return []any{}, nil
		}
		return slices.Clone(s), nil
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
// map[string]any element. It errors when the element is not a map or the key is
// absent: returning "" instead would make every element compare equal, so a
// mistyped key or a slice of non-maps would silently yield unsorted output
// rather than a diagnosable failure.
func fieldString(v any, key string) (string, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", fmt.Errorf("element is not map[string]any, got %T", v)
	}
	val, exists := m[key]
	if !exists {
		return "", fmt.Errorf("field %q not found", key)
	}
	return fmt.Sprint(val), nil
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
	return slices.Clone(elems)
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

// MaxSeqLen is the largest sequence seq will produce. It is a guardrail against
// a mistyped or mis-parsed bound turning into a multi-gigabyte allocation, not a
// budget to render against: 10000 iterations already emit far more HTML than any
// page wants, so no legitimate template should come near it.
//
// It is a constant rather than a variable because a mutable global would be
// shared state in a package built for concurrent template execution. A caller
// who genuinely needs longer sequences can register their own seq over this one
// with Merge.
const MaxSeqLen = 10000

// seqSpan returns the number of steps from start to end — one less than the
// element count — and whether the sequence has any elements at all.
//
// The span is measured in uint64 because end-start overflows int whenever the
// bounds straddle zero widely enough; that wrap is how seq (math.MinInt)
// (math.MaxInt) used to compute a count of 0 and silently return nothing. The
// span is returned rather than the count because the count itself does not
// always fit either: the full int range holds 2^64 values, one more than uint64
// can represent. Callers bound the span before adding one.
func seqSpan(start, end, step int) (uint64, bool) {
	if step > 0 {
		if start > end {
			return 0, false
		}
		return (uint64(end) - uint64(start)) / uint64(step), true
	}
	if start < end {
		return 0, false
	}
	// -uint64(step) is the magnitude of a negative step, computed in unsigned
	// arithmetic so that math.MinInt, whose negation overflows int, still works.
	return (uint64(start) - uint64(end)) / (-uint64(step)), true
}

// Seq returns a slice of integers. Counting is 1-based by default.
// The sequence may not exceed MaxSeqLen elements.
//
//	seq 5        → [1 2 3 4 5]
//	seq 3 7      → [3 4 5 6 7]
//	seq 1 10 2   → [1 3 5 7 9]
//	seq 5 1 -1   → [5 4 3 2 1]
func Seq(args ...int) ([]int, error) {
	var start, end, step int
	switch len(args) {
	case 1:
		start, end, step = 1, args[0], 1
	case 2:
		start, end, step = args[0], args[1], 1
	case 3:
		start, end, step = args[0], args[1], args[2]
		if step == 0 {
			return nil, fmt.Errorf("seq: step cannot be zero")
		}
	default:
		return nil, fmt.Errorf("seq: expected 1–3 arguments, got %d", len(args))
	}

	span, ok := seqSpan(start, end, step)
	if !ok {
		return []int{}, nil
	}
	if span >= MaxSeqLen { // element count is span+1
		// Report the request rather than the resulting length: the length can
		// exceed uint64, and the arguments are what the author needs to see.
		return nil, fmt.Errorf("seq: %d to %d by %d exceeds the limit of %d elements",
			start, end, step, MaxSeqLen)
	}

	out := make([]int, span+1)
	v := start
	for i := range out {
		out[i] = v
		// Counting the elements up front rather than testing v against end is
		// what keeps this terminating: the final increment can overflow past
		// end and wrap, which turned the old condition-driven loop into an
		// unbounded one. The wrapped value is never used.
		v += step
	}
	return out, nil
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
		start := max(len(r)+n, 0)
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
	start := max(len(elems)+n, 0)
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
		end := max(len(r)+n, 0)
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
	end := max(len(elems)+n, 0)
	return elems[:end], nil
}

// --- sequence transformation ---

// Reverse returns a new slice with the elements in reverse order.
//
//	reverse []int{1, 2, 3} → []any{3, 2, 1}
func Reverse(v any) (any, error) {
	out, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("reverse: %w", err)
	}
	// toSlice already allocated; reversing it in place does not touch the input.
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

// Concat concatenates multiple slices into a single []any. The result is always
// non-nil, so concat with no arguments yields an empty slice rather than nil,
// matching list.
//
//	concat (list 1 2) (list 3 4) → []any{1, 2, 3, 4}
//	concat                       → []any{}
func Concat(ins ...any) ([]any, error) {
	// Validate and measure every argument before allocating, so that a bad
	// argument fails without having built anything and out can be sized exactly.
	total := 0
	for i, v := range ins {
		rv := reflect.ValueOf(v)
		if !rv.IsValid() || rv.Kind() != reflect.Slice {
			return nil, fmt.Errorf("concat: argument %d: expected slice, got %T", i, v)
		}
		total += rv.Len()
	}
	// Copy in directly rather than through toSlice, whose per-argument copy
	// would only be copied again into out. out is freshly allocated here, so
	// nothing aliases the arguments.
	out := make([]any, 0, total)
	for _, v := range ins {
		if s, ok := v.([]any); ok {
			out = append(out, s...)
			continue
		}
		rv := reflect.ValueOf(v)
		for i := range rv.Len() {
			out = append(out, rv.Index(i).Interface())
		}
	}
	return out, nil
}

// keyedElem pairs a collection element with its precomputed sort key, so that
// key extraction happens once per element rather than once per comparison.
type keyedElem struct {
	key  string
	elem any
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
// An optional key names a field for slice-of-maps sorting (always lexicographic);
// it is an error if any element is not a map[string]any or lacks that key.
// For descending order, compose with reverse.
// ISO 8601 date strings ("2006-01-02") sort correctly lexicographically.
//
//	sort (list "banana" "apple" "cherry") → ["apple" "banana" "cherry"]
//	sort (list 10 2 30)                   → [2 10 30]
//	sort $pages "Title"                   → pages A→Z by Title field
func Sort(v any, key ...string) (any, error) {
	// toSlice already allocated, so out can be sorted in place.
	out, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("sort: %w", err)
	}
	if len(key) > 0 {
		k := key[0]
		// Extract the sort keys up front rather than inside the comparator.
		// This reports a bad element or missing field even for inputs too short
		// for the comparator to run, and formats each key once instead of on
		// every comparison.
		keyed := make([]keyedElem, len(out))
		for i, e := range out {
			s, err := fieldString(e, k)
			if err != nil {
				return nil, fmt.Errorf("sort: key %q: %w", k, err)
			}
			keyed[i] = keyedElem{key: s, elem: e}
		}
		slices.SortStableFunc(keyed, func(a, b keyedElem) int {
			return strings.Compare(a.key, b.key)
		})
		for i, p := range keyed {
			out[i] = p.elem
		}
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
	// toSlice already allocated, so out can be sorted in place.
	out, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("sortNum: %w", err)
	}
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
	if len(m) == 0 {
		// slices.Sorted collects into a nil slice, so an empty map would yield
		// nil rather than an empty slice.
		return []string{}, nil
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
//   - map: key existence (val must be assignable to the map's key type)
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
		// Iterate rather than going through toSlice: this is a read-only search
		// that returns a bool, so there is no result to keep unaliased and no
		// reason to copy the whole slice to find one element. (The rule against
		// non-copying fast paths applies to toSlice, which must protect the
		// structures it hands back; nothing escapes from here.)
		if s, ok := v.([]any); ok {
			for _, elem := range s {
				if reflect.DeepEqual(elem, val) {
					return true, nil
				}
			}
			return false, nil
		}
		for i := range rv.Len() {
			if reflect.DeepEqual(rv.Index(i).Interface(), val) {
				return true, nil
			}
		}
		return false, nil
	case reflect.Map:
		// reflect.Value.MapIndex panics if val is not assignable to the map's
		// key type, so check before indexing. Any key type is supported, not
		// just string: in $m 1 works on a map[int]any.
		kt := rv.Type().Key()
		kv := reflect.ValueOf(val)
		if !kv.IsValid() || !kv.Type().AssignableTo(kt) {
			return false, fmt.Errorf("in: map key search requires %s, got %T", kt, val)
		}
		return rv.MapIndex(kv).IsValid(), nil
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
