package doublebrace

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"math"
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

// isSequence reports whether k is an indexable sequence of elements. Arrays are
// accepted alongside slices: reflect indexes them identically, and template data
// carrying a fixed-size field ([3]string, say) is otherwise unreachable from
// every collection function. This is the single definition of what those
// functions accept — the three places that repeat the check for their own
// reasons all consult it.
func isSequence(k reflect.Kind) bool {
	return k == reflect.Slice || k == reflect.Array
}

// toSlice converts any slice or array to []any. []any is cloned; other types are
// expanded via reflection. The result is never nil, only ever empty, so that
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
	if !rv.IsValid() || !isSequence(rv.Kind()) {
		return nil, fmt.Errorf("expected slice or array, got %T", v)
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

// fieldNumber returns the numeric value of a named field in a map[string]any
// element, ready for number.compare.
func fieldNumber(v any, key string) (number, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return number{}, fmt.Errorf("element is not map[string]any, got %T", v)
	}
	val, exists := m[key]
	if !exists {
		return number{}, fmt.Errorf("field %q not found", key)
	}
	return toNumber(val)
}

// toNumber converts a value to a number for sorting.
//
// A value of numeric kind keeps its own representation, so int64 and uint64 sort
// exactly rather than through float64. Anything else — a numeric string, which
// is what sortNum exists for — falls back to toFloat64 and is float-limited, as
// it necessarily is: the precision was already spent by the time a number was
// written down as text.
func toNumber(v any) (number, error) {
	if n := asNumber(v); n.kind != notNumeric {
		return n, nil
	}
	f, err := toFloat64(v)
	if err != nil {
		return number{}, err
	}
	return number{kind: numFloat, f: f}, nil
}

// numKind classifies a value's numeric representation. The ordering matters:
// number.equal canonicalizes its operands by it so each pair of kinds is handled
// once.
type numKind int

const (
	notNumeric numKind = iota
	numInt
	numUint
	numFloat
)

// number is a numeric value normalized for comparison across types. Only the
// field named by kind holds a meaningful value.
type number struct {
	kind numKind
	i    int64
	u    uint64
	f    float64
}

// asNumber classifies v by reflect kind, so named types (type Weight int) and
// every integer and float width are recognized. Strings are deliberately not
// numeric here: a "1" in the data should not match a 1 in the template, however
// convenient toFloat64 would make that.
func asNumber(v any) number {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return number{}
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return number{kind: numInt, i: rv.Int()}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return number{kind: numUint, u: rv.Uint()}
	case reflect.Float32, reflect.Float64:
		return number{kind: numFloat, f: rv.Float()}
	}
	return number{}
}

// floatIsInt reports whether f is a whole number within int64's range, and
// returns it as an int64.
//
// Comparisons go through here rather than widening the integer to a float
// because float64 cannot represent every int64: converting 9007199254740993 to
// float64 yields 9007199254740992, so widening would report two distinct IDs as
// equal. Narrowing the float instead is exact at every magnitude.
func floatIsInt(f float64) (int64, bool) {
	if f != math.Trunc(f) {
		return 0, false
	}
	// -float64(math.MinInt64) is 2^63, one past the largest int64 and exactly
	// representable; math.MinInt64 itself is exactly representable.
	if f < float64(math.MinInt64) || f >= -float64(math.MinInt64) {
		return 0, false
	}
	return int64(f), true
}

// floatIsUint is floatIsInt for the unsigned range.
func floatIsUint(f float64) (uint64, bool) {
	const twoPow64 = float64(1<<63) * 2 // one past the largest uint64
	if f != math.Trunc(f) || f < 0 || f >= twoPow64 {
		return 0, false
	}
	return uint64(f), true
}

// equal reports whether two numbers are equal in value, across kinds.
func (a number) equal(b number) bool {
	if a.kind > b.kind {
		a, b = b, a // canonical order, so each kind pair appears once below
	}
	switch {
	case a.kind == numInt && b.kind == numInt:
		return a.i == b.i
	case a.kind == numInt && b.kind == numUint:
		return a.i >= 0 && uint64(a.i) == b.u
	case a.kind == numInt && b.kind == numFloat:
		i, ok := floatIsInt(b.f)
		return ok && i == a.i
	case a.kind == numUint && b.kind == numUint:
		return a.u == b.u
	case a.kind == numUint && b.kind == numFloat:
		u, ok := floatIsUint(b.f)
		return ok && u == a.u
	default: // both float
		return a.f == b.f
	}
}

// compare orders two numbers across kinds, returning -1, 0, or +1 as a is less
// than, equal to, or greater than b.
//
// NaN sorts before every number and ties with itself, as cmp.Compare defines it.
// That is the one place compare and equal disagree — equal reports NaN != NaN,
// following ==, while a sort needs a total order and cannot leave an element
// unplaced. Everywhere else a compare of 0 and an equal of true coincide.
//
// Ordering is exact at every magnitude, for the reason equal is: widening both
// sides to float64 collapses integers above 2^53, so 9007199254740992 and
// 9007199254740993 compare equal and a stable sort leaves them in input order —
// unsorted output reported as success. Only an int64 or uint64 can meet a
// float64 here, and those two pairs are decided by narrowing the float, which is
// exact at every magnitude.
func (a number) compare(b number) int {
	if a.kind > b.kind {
		return -b.compare(a) // canonical order, so each kind pair appears once
	}
	switch {
	case a.kind == numInt && b.kind == numInt:
		return cmp.Compare(a.i, b.i)
	case a.kind == numInt && b.kind == numUint:
		if a.i < 0 {
			return -1 // no uint64 is negative
		}
		return cmp.Compare(uint64(a.i), b.u)
	case a.kind == numInt && b.kind == numFloat:
		return -compareFloatInt64(b.f, a.i)
	case a.kind == numUint && b.kind == numUint:
		return cmp.Compare(a.u, b.u)
	case a.kind == numUint && b.kind == numFloat:
		return -compareFloatUint64(b.f, a.u)
	default: // both float
		return cmp.Compare(a.f, b.f)
	}
}

// compareFloatInt64 orders f against i without widening i to a float.
//
// Once f is known to be inside int64's range, truncating it splits the
// comparison into an exact integer one and a fractional tiebreak. Truncation is
// toward zero, so a negative f carries a negative fraction and the tiebreak
// reads the same way in both directions.
func compareFloatInt64(f float64, i int64) int {
	switch {
	case math.IsNaN(f):
		return -1
	// -float64(math.MinInt64) is 2^63, one past the largest int64 and exactly
	// representable; math.MinInt64 itself is exactly representable. Infinities
	// land here too.
	case f >= -float64(math.MinInt64):
		return 1
	case f < float64(math.MinInt64):
		return -1
	}
	trunc := math.Trunc(f)
	if ti := int64(trunc); ti != i {
		return cmp.Compare(ti, i)
	}
	return cmp.Compare(f-trunc, 0)
}

// compareFloatUint64 is compareFloatInt64 for the unsigned range.
func compareFloatUint64(f float64, u uint64) int {
	const twoPow64 = float64(1<<63) * 2 // one past the largest uint64
	switch {
	case math.IsNaN(f):
		return -1
	case f < 0:
		return -1 // no uint64 is negative
	case f >= twoPow64:
		return 1
	}
	trunc := math.Trunc(f)
	if tu := uint64(trunc); tu != u {
		return cmp.Compare(tu, u)
	}
	return cmp.Compare(f-trunc, 0)
}

// valuesEqual reports whether a and b are equal for filtering and membership
// tests. Numbers compare by value across types; everything else falls back to
// reflect.DeepEqual.
//
// Numeric comparison exists because the type a number arrives as is an accident
// of the decoder: encoding/json produces float64, TOML int64, YAML int, and a
// template literal is int if written 1 and float64 if written 1.0. Matching only
// on identical types made where and in stricter than the template language's own
// eq, which already unifies integer widths — and unlike eq, which reports an
// int/float mismatch as an error, a filter that silently returns nothing looks
// like data that legitimately did not match.
func valuesEqual(a, b any) bool {
	na, nb := asNumber(a), asNumber(b)
	if na.kind != notNumeric && nb.kind != notNumeric {
		return na.equal(nb)
	}
	return reflect.DeepEqual(a, b)
}

// isZero reports whether v is the zero value for its type.
// nil, false, 0, "", empty slices/maps, and all-zero arrays/structs are
// considered zero.
//
// A type that defines IsZero() bool decides for itself. That is what makes a
// zero time.Time report as zero even when it carries a location: t.In(loc) and
// t.Local() leave the seconds at zero but set the location field, so the struct
// is no longer all-zero by reflection while time.Time.IsZero still reports true.
// encoding/json's omitzero resolves this the same way.
func isZero(v any) bool {
	if v == nil {
		return true
	}
	if z, ok := v.(interface{ IsZero() bool }); ok {
		return z.IsZero()
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
	case reflect.Slice, reflect.Map:
		return rv.Len() == 0
	case reflect.Pointer, reflect.Interface:
		return rv.IsNil()
	case reflect.Array, reflect.Struct:
		// Arrays go by zero-ness, not emptiness: an array's length is fixed, so
		// treating [3]int{} as non-zero because it has three slots would
		// contradict the definition above. Slices and maps keep emptiness
		// semantics, where a length of zero is the meaningful test.
		return rv.IsZero()
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
// Hugo caps its equivalent at 1000000 (https://gohugo.io/functions/collections/seq/).
// The lower limit here is deliberate: 1000000 bounds the allocation at 8MB but
// still lets a mistaken bound run a million iterations and emit an unusable
// page, which is the failure this is meant to catch rather than merely survive.
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
			return nil, errors.New("seq: step cannot be zero")
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
			return nil, errors.New("first: empty string")
		}
		return string(r[0]), nil
	}
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("first: %w", err)
	}
	if len(elems) == 0 {
		return nil, errors.New("first: empty slice")
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
			return nil, errors.New("last: empty string")
		}
		return string(r[len(r)-1]), nil
	}
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("last: %w", err)
	}
	if len(elems) == 0 {
		return nil, errors.New("last: empty slice")
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
func Reverse(v any) ([]any, error) {
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
// Elements compare with the same rule as where and in: numbers are equal when
// their values are equal, whatever their types, so a 1 decoded as float64 and a
// 1 written as an int literal are one duplicate rather than two elements.
//
//	compact []int{1, 1, 2, 3, 3, 1} → []any{1, 2, 3, 1}
//	compact []any{1, 1.0, 2}        → []any{1, 2}
func Compact(v any) ([]any, error) {
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("compact: %w", err)
	}
	return slices.CompactFunc(elems, valuesEqual), nil
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
		if !rv.IsValid() || !isSequence(rv.Kind()) {
			return nil, fmt.Errorf("concat: argument %d: expected slice or array, got %T", i, v)
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

// keyedElem pairs a collection element with its precomputed sort key.
type keyedElem[K any] struct {
	key  K
	elem any
}

// sortByKey sorts elems by a key extracted from each element, in place, and
// returns it. elems must already be a private copy; callers get theirs from
// toSlice.
//
// Extracting every key before sorting rather than inside the comparator is what
// makes the error reporting reliable: slices.SortStableFunc never calls a
// comparator on fewer than two elements, so validation done there silently
// passes a one-element slice of elements it should have rejected. It also runs
// keyFn once per element instead of once per comparison.
func sortByKey[K any](elems []any, keyFn func(any) (K, error), cmpFn func(K, K) int) ([]any, error) {
	keyed := make([]keyedElem[K], len(elems))
	for i, e := range elems {
		k, err := keyFn(e)
		if err != nil {
			return nil, err
		}
		keyed[i] = keyedElem[K]{key: k, elem: e}
	}
	slices.SortStableFunc(keyed, func(a, b keyedElem[K]) int {
		return cmpFn(a.key, b.key)
	})
	for i, p := range keyed {
		elems[i] = p.elem
	}
	return elems, nil
}

type sortMode int

const (
	sortLex sortMode = iota
	sortNumeric
	sortTime
)

// inferSortMode inspects the first non-nil element of elems to decide how to sort.
//
// Numbers are recognized through asNumber rather than a type switch on the
// concrete types, so a named type (type Weight int) sorts numerically like the
// int it is. A type switch here would classify it as sortLex and quietly order
// [10 2 30] as text, which is a wrong answer rather than an error — and would
// disagree with in, compact, and where, which reach the same value through
// asNumber and treat it as a number.
//
// Strings stay lexicographic even when they hold digits: asNumber excludes them
// deliberately, and sortNum is the way to ask for "10" to sort after "9".
func inferSortMode(elems []any) sortMode {
	for _, e := range elems {
		if e == nil {
			continue
		}
		if asNumber(e).kind != notNumeric {
			return sortNumeric
		}
		if _, ok := e.(time.Time); ok {
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
// Numeric ordering is exact at every magnitude, including int64 and uint64
// values above 2^53, which a float64 cannot tell apart.
//
//	sort (list "banana" "apple" "cherry") → ["apple" "banana" "cherry"]
//	sort (list 10 2 30)                   → [2 10 30]
//	sort $pages "Title"                   → pages A→Z by Title field
func Sort(v any, key ...string) ([]any, error) {
	// toSlice already allocated, so out can be sorted in place.
	out, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("sort: %w", err)
	}
	if len(key) > 0 {
		k := key[0]
		return sortByKey(out, func(e any) (string, error) {
			s, err := fieldString(e, k)
			if err != nil {
				return "", fmt.Errorf("sort: key %q: %w", k, err)
			}
			return s, nil
		}, cmp.Compare[string])
	}
	switch inferSortMode(out) {
	case sortNumeric:
		return sortByKey(out, func(e any) (number, error) {
			n, err := toNumber(e)
			if err != nil {
				return number{}, errors.New("sort: cannot convert element to number")
			}
			return n, nil
		}, number.compare)
	case sortTime:
		return sortByKey(out, func(e any) (time.Time, error) {
			t, ok := e.(time.Time)
			if !ok {
				return time.Time{}, errors.New("sort: element is not time.Time")
			}
			return t, nil
		}, time.Time.Compare)
	default:
		return sortByKey(out, func(e any) (string, error) {
			return fmt.Sprint(e), nil
		}, cmp.Compare[string])
	}
}

// SortNum returns a new slice sorted numerically, converting each element to a
// number — so numeric strings sort by value rather than as text.
// An optional key names a field for slice-of-maps sorting. It is an error if any
// element cannot be converted, or — with a key — is not a map[string]any or lacks
// that key.
// For descending order, compose with reverse.
//
// Values that are already numeric are ordered exactly, whatever their width; a
// numeric string is parsed as a float64 and so carries that type's precision.
//
//	sortNum (list "10" "9" "2") → ["2" "9" "10"]
//	sortNum $pages "Year"       → pages sorted by Year field, ascending
func SortNum(v any, key ...string) ([]any, error) {
	// toSlice already allocated, so out can be sorted in place.
	out, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("sortNum: %w", err)
	}
	if len(key) > 0 {
		k := key[0]
		return sortByKey(out, func(e any) (number, error) {
			n, err := fieldNumber(e, k)
			if err != nil {
				return number{}, fmt.Errorf("sortNum: cannot convert field %q to number", k)
			}
			return n, nil
		}, number.compare)
	}
	return sortByKey(out, func(e any) (number, error) {
		n, err := toNumber(e)
		if err != nil {
			return number{}, errors.New("sortNum: cannot convert value to number")
		}
		return n, nil
	}, number.compare)
}

// Where filters a slice of map[string]any by field equality.
// Only elements where element[key] equals val are included in the result.
//
// Numbers compare by value regardless of type, so a field decoded from JSON as
// float64 matches an integer literal. It is an error if any element is not a
// map[string]any or lacks the named field — a missing field would otherwise drop
// every element and look like data that simply did not match.
//
//	where $pages "Draft" false    → pages where Draft == false
//	where $pages "Section" "blog" → pages in the blog section
//	where $pages "Weight" 1       → matches whether Weight is int, int64, or float64
func Where(v any, key string, val any) ([]any, error) {
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
		got, exists := m[key]
		if !exists {
			// A missing field would compare unequal and quietly drop the element,
			// making a typo in the key indistinguishable from data that matched
			// nothing. Sort reports this the same way.
			return nil, fmt.Errorf("where: field %q not found", key)
		}
		if valuesEqual(got, val) {
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
//   - slice or array: element membership; numbers compare by value across
//     types, everything else via reflect.DeepEqual
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
	case reflect.Slice, reflect.Array:
		// Iterate rather than going through toSlice: this is a read-only search
		// that returns a bool, so there is no result to keep unaliased and no
		// reason to copy the whole sequence to find one element. (The rule against
		// non-copying fast paths applies to toSlice, which must protect the
		// structures it hands back; nothing escapes from here.)
		if s, ok := v.([]any); ok {
			for _, elem := range s {
				if valuesEqual(elem, val) {
					return true, nil
				}
			}
			return false, nil
		}
		for i := range rv.Len() {
			if valuesEqual(rv.Index(i).Interface(), val) {
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
// Zero values: nil, false, 0, "", empty slices/maps, and all-zero arrays/structs
// — including a zero time.Time, so an unset date falls back.
//
//	default "anon" ""      → "anon"
//	default "anon" "Alice" → "Alice"
//	default 0 42           → 42
//	default "Draft" $date  → "Draft" when $date is the zero time
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
