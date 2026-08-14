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

// asString reports whether v is a string, returning its value. It is the
// package-wide definition of "is this a string", the counterpart to isSequence:
// a named type (type Slug string) or an html/template typed value is a string
// here, because reflect classifies by kind and template data routinely carries
// both.
//
// A type assertion to string is what this replaces. That accepted only the
// unnamed type, which is why in — which reached the same value through a
// reflect kind switch — could search a Slug that first, last, take, and drop
// all rejected as "expected slice or array".
//
// The functions built on this return a plain string rather than the input's own
// type. For an html/template value that is a security property, not a
// convenience: slicing markup by runes can cut a tag or an entity in half, and
// half a tag is not markup. Downgrading makes html/template escape the result
// for whatever context it lands in, which is the fail-closed direction — a
// truncated template.URL holding "javascript:" is rejected as a plain string
// where the untruncated value would have been emitted as trusted.
// TestSequenceAccess_htmlTemplateDowngrade pins this across every safe type.
func asString(v any) (string, bool) {
	if s, ok := v.(string); ok {
		return s, true
	}
	rv := reflect.ValueOf(v)
	if rv.IsValid() && rv.Kind() == reflect.String {
		return rv.String(), true
	}
	return "", false
}

// isNilValue reports whether v is nil, including a typed nil: a (*Page)(nil)
// carried in an interface is not == nil, though it is just as absent, and
// fmt.Sprint renders both as "<nil>". This is the package-wide definition of
// nil, alongside isSequence, asString, and numKindOf.
//
// A nil slice or map is deliberately not nil here. Those are ordinary empty
// containers — they render as "[]" and "map[]", have a length, and range over
// nothing — so isZero already answers for them by emptiness, and sorting them
// as values is well defined.
func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	// Only Pointer can reach here: reflect.ValueOf unwraps an interface, so a
	// value of interface kind is never produced from an any.
	return rv.Kind() == reflect.Pointer && rv.IsNil()
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

// toElems is toSlice for the four functions that also accept a string — first,
// last, take, and drop, which reach a string through asString before they get
// here. It differs only in the error.
//
// toSlice says "expected slice or array", which is the truth for the functions
// that only take a sequence but names two of the three shapes these four accept.
// first 42 reported "first: expected slice or array, got int", which reads as
// though a string would not have worked either, and the one thing an author in
// that position needs to know is which of the shapes they actually have.
//
// Replacing the message rather than wrapping it loses nothing: toSlice has a
// single error, and this restates the same fact more completely.
func toElems(fn string, v any) ([]any, error) {
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("%s: expected a slice, array, or string, got %T", fn, v)
	}
	return elems, nil
}

// fieldValue reads the field named key from a collection element, for the
// key-taking forms of where, sort, and sortNum.
//
// An element may be a map with string-kind keys or a struct, or a pointer to
// either. Requiring map[string]any meant a plain []Page — the ordinary way to
// hand Go data to a template — could not be filtered or sorted by field at all,
// though it is exactly the shape the doc examples read as though they accept.
//
// Struct fields match by exact name and must be exported. FieldByName also
// resolves promoted fields of embedded structs, so a field reached through
// embedding works without naming the embedded type. Method calls are
// deliberately not supported: reading a field is inert, whereas invoking
// arbitrary methods named by template data is a different thing to hand a
// template author.
//
// An absent key is an error rather than a zero value. Returning "" or nil would
// make every element compare equal, so a mistyped key would silently yield
// unsorted output or an empty filter instead of a diagnosable failure.
func fieldValue(elem any, key string) (any, error) {
	if m, ok := elem.(map[string]any); ok {
		val, exists := m[key]
		if !exists {
			return nil, fmt.Errorf("field %q not found", key)
		}
		return val, nil
	}
	rv := reflect.ValueOf(elem)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, fmt.Errorf("field %q: element is a nil %s", key, rv.Type())
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		kt := rv.Type().Key()
		if kt.Kind() != reflect.String {
			return nil, fmt.Errorf("field %q: element has %s keys, want string", key, kt)
		}
		// Convert so a named key type (map[Slug]string) indexes correctly.
		got := rv.MapIndex(reflect.ValueOf(key).Convert(kt))
		if !got.IsValid() {
			return nil, fmt.Errorf("field %q not found", key)
		}
		return got.Interface(), nil
	case reflect.Struct:
		f := rv.FieldByName(key)
		if !f.IsValid() {
			return nil, fmt.Errorf("field %q not found", key)
		}
		if !f.CanInterface() {
			// reflect finds unexported fields but panics on Interface().
			// Reporting beats faulting midway through rendering a page.
			return nil, fmt.Errorf("field %q is unexported", key)
		}
		return f.Interface(), nil
	}
	return nil, fmt.Errorf("element is not a map or struct, got %T", elem)
}

// fieldString returns the string representation of a named field in a
// collection element, for the key form of sort.
//
// A nil field is an error for the reason a nil element is — see requireNoNils.
// Without the check fmt.Sprint would render it "<nil>" and sort it as that text,
// which is the silently-wrong-order outcome the check exists to prevent.
func fieldString(v any, key string) (string, error) {
	val, err := fieldValue(v, key)
	if err != nil {
		return "", err
	}
	if isNilValue(val) {
		return "", fmt.Errorf("field %q is nil", key)
	}
	return fmt.Sprint(val), nil
}

// fieldNumber returns the numeric value of a named field in a collection
// element, ready for number.compare.
//
// The lookup error is returned as fieldValue wrote it, which already names the
// key; only the conversion error needs the key added. Naming it in exactly one
// place is what lets the callers wrap with a bare "sortNum: %w" and still say
// which field failed, without the key appearing twice on one line.
func fieldNumber(v any, key string) (number, error) {
	val, err := fieldValue(v, key)
	if err != nil {
		return number{}, err
	}
	if isNilValue(val) {
		// Reported as nil rather than as a failed conversion, so it reads the
		// same as it does from sort. See fieldString.
		return number{}, fmt.Errorf("field %q is nil", key)
	}
	n, err := toNumber(val)
	if err != nil {
		return number{}, fmt.Errorf("field %q: %w", key, err)
	}
	return n, nil
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

// numKindOf classifies a reflect kind, and is the single definition of which
// kinds are numeric — the counterpart to isSequence and asString. Everything
// that asks the question goes through here or through asNumber, so a type
// cannot be a number to one function and text to the next, which is the way
// inferSortMode once came to sort a []Weight lexicographically.
func numKindOf(k reflect.Kind) numKind {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return numInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return numUint
	case reflect.Float32, reflect.Float64:
		return numFloat
	}
	return notNumeric
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
	switch numKindOf(rv.Kind()) {
	case numInt:
		return number{kind: numInt, i: rv.Int()}
	case numUint:
		return number{kind: numUint, u: rv.Uint()}
	case numFloat:
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
// tests. Numbers compare by value across types, as do strings across string
// kinds; everything else falls back to reflect.DeepEqual.
//
// Numeric comparison exists because the type a number arrives as is an accident
// of the decoder: encoding/json produces float64, TOML int64, YAML int, and a
// template literal is int if written 1 and float64 if written 1.0. Matching only
// on identical types made where and in stricter than the template language's own
// eq, which already unifies integer widths — and unlike eq, which reports an
// int/float mismatch as an error, a filter that silently returns nothing looks
// like data that legitimately did not match.
//
// String comparison is here for the same reason, reached from the other side.
// DeepEqual is type-strict, so a named type never equalled the literal it was
// written from: where $pages "Section" "blog" returned nothing at all when
// Section was a type Section string, which is exactly the silent empty filter
// the paragraph above rejects — and the missing-field check in Where exists to
// prevent. It disagreed with the rest of the package too, since asString is what
// decides whether a value is a string everywhere else, so in $slug "b" searched
// a named string that in $slugs "b" could not find among them.
//
// The trade is the one already made for numbers: Slug("a") and "a" are equal
// here though Go says otherwise. Equality is the only thing this decides — a
// value's own type still survives the collection functions, so sorting a []Slug
// yields Slug values, and the string functions downgrade for the separate
// reason described on asString.
func valuesEqual(a, b any) bool {
	na, nb := asNumber(a), asNumber(b)
	if na.kind != notNumeric && nb.kind != notNumeric {
		return na.equal(nb)
	}
	if sa, ok := asString(a); ok {
		if sb, ok := asString(b); ok {
			return sa == sb
		}
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
//
// Everything else defers to reflect.Value.IsZero, which is already the "zero
// value for its type" test this documents — spelling out a case per kind only
// created kinds to forget, and the ones forgotten (a nil chan, a nil func) fell
// through to a bare "not zero". Slices and maps are the one deliberate
// departure: there, emptiness is the meaningful question, so a non-nil empty
// slice is zero here though reflect says otherwise. Arrays keep zero-ness,
// because an array's length is fixed and an emptiness test would mean [3]int{}
// could never be zero.
func isZero(v any) bool {
	if v == nil {
		return true
	}
	// A nil pointer has to be settled before the IsZero method call below, not
	// merely by the reflect fallback at the end. A value-receiver IsZero is
	// promoted into the pointer type's method set, so (*time.Time)(nil) satisfies
	// the interface and calling the method dereferences the nil pointer. An
	// optional date carried as a *time.Time is exactly the shape default exists
	// for, so the panic is reachable from ordinary template data:
	// {{ default "TBD" .Date }} failed the whole render rather than falling back.
	if isNilValue(v) {
		return true
	}
	if z, ok := v.(interface{ IsZero() bool }); ok {
		return z.IsZero()
	}
	rv := reflect.ValueOf(v)
	if k := rv.Kind(); k == reflect.Slice || k == reflect.Map {
		return rv.Len() == 0
	}
	return rv.IsZero()
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
// A key may be any value of string kind, so a named type (type Slug string)
// names a key — the rule asString sets and the rest of the package follows.
// merge, keys, values, and the key forms of sort and where all reach a named key
// type by converting, so a dict literal that could not take one was the one
// place a Slug in hand could not be used as a key.
//
//	dict "name" "Alice" "age" 30 → map[string]any{"name": "Alice", "age": 30}
func Dict(kvs ...any) (map[string]any, error) {
	if len(kvs)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments (%d)", len(kvs))
	}
	m := make(map[string]any, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		k, ok := asString(kvs[i])
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
// Bounds are any numeric type or numeric string, converted as toInt does — so
// seq (add $n 1) and a count read from data both work.
//
//	seq 5        → [1 2 3 4 5]
//	seq 3 7      → [3 4 5 6 7]
//	seq 1 10 2   → [1 3 5 7 9]
//	seq 5 1 -1   → [5 4 3 2 1]
func Seq(args ...any) ([]int, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("seq: expected 1–3 arguments, got %d", len(args))
	}
	bounds := make([]int, len(args))
	for i, a := range args {
		n, err := toCount(fmt.Sprintf("seq: argument %d", i+1), a)
		if err != nil {
			return nil, err
		}
		bounds[i] = n
	}

	var start, end, step int
	switch len(bounds) {
	case 1:
		start, end, step = 1, bounds[0], 1
	case 2:
		start, end, step = bounds[0], bounds[1], 1
	default:
		start, end, step = bounds[0], bounds[1], bounds[2]
		if step == 0 {
			return nil, errors.New("seq: step cannot be zero")
		}
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
	if s, ok := asString(v); ok {
		r := []rune(s)
		if len(r) == 0 {
			return nil, errors.New("first: empty string")
		}
		return string(r[0]), nil
	}
	elems, err := toElems("first", v)
	if err != nil {
		return nil, err
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
	if s, ok := asString(v); ok {
		r := []rune(s)
		if len(r) == 0 {
			return nil, errors.New("last: empty string")
		}
		return string(r[len(r)-1]), nil
	}
	elems, err := toElems("last", v)
	if err != nil {
		return nil, err
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
// n is any numeric type or numeric string, converted as toInt does — so the
// float64 every math function returns is a usable count: take $list (div $n 2).
//
//	take []int{1, 2, 3, 4, 5} 3  → []any{1, 2, 3}
//	take []int{1, 2, 3, 4, 5} -2 → []any{4, 5}
//	take "日本語" 2                → "日本"
//	take "日本語" -1               → "語"
func Take(v any, n any) (any, error) {
	count, err := toCount("take", n)
	if err != nil {
		return nil, err
	}
	if s, ok := asString(v); ok {
		r := []rune(s)
		if count >= 0 {
			return string(r[:min(count, len(r))]), nil
		}
		// negative: last |n| runes
		start := max(len(r)+count, 0)
		return string(r[start:]), nil
	}
	elems, err := toElems("take", v)
	if err != nil {
		return nil, err
	}
	if count >= 0 {
		return elems[:min(count, len(elems))], nil
	}
	// negative: last |n| elements
	start := max(len(elems)+count, 0)
	return elems[start:], nil
}

// Drop skips the first n elements of a slice, or the first n runes of a
// string. If n is negative, it removes the last |n| elements or runes.
// If |n| exceeds the length an empty result is returned.
// Rune-aware for strings: multi-byte characters are not split.
//
// n is any numeric type or numeric string, converted as toInt does — so the
// float64 every math function returns is a usable count: drop $list (div $n 2).
//
//	drop []int{1, 2, 3, 4, 5} 2  → []any{3, 4, 5}
//	drop []int{1, 2, 3, 4, 5} -2 → []any{1, 2, 3}
//	drop "日本語" 1                → "本語"
//	drop "日本語" -1               → "日本"
func Drop(v any, n any) (any, error) {
	count, err := toCount("drop", n)
	if err != nil {
		return nil, err
	}
	if s, ok := asString(v); ok {
		r := []rune(s)
		if count >= 0 {
			return string(r[min(count, len(r)):]), nil
		}
		// negative: remove last |n| runes
		end := max(len(r)+count, 0)
		return string(r[:end]), nil
	}
	elems, err := toElems("drop", v)
	if err != nil {
		return nil, err
	}
	if count >= 0 {
		return elems[min(count, len(elems)):], nil
	}
	// negative: remove last |n| elements
	end := max(len(elems)+count, 0)
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
// Strings compare the same way across string kinds. The element kept is the one
// that was there — only equality spans types, never the result.
//
//	compact []int{1, 1, 2, 3, 3, 1} → []any{1, 2, 3, 1}
//	compact []any{1, 1.0, 2}        → []any{1, 2}
//	compact []any{Slug("a"), "a"}   → []any{Slug("a")}
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
	//
	// A []any needs neither: it is a sequence by its type, so there is nothing to
	// check and len answers what reflect would. Taking it first here as well as
	// in the copy below is what keeps the ordinary case — every collection
	// function in this package returns []any — off reflect entirely.
	total := 0
	for i, v := range ins {
		if s, ok := v.([]any); ok {
			total += len(s)
			continue
		}
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

// requireNoNils reports the first nil element of elems, if any.
//
// Sorting orders values against each other, and a nil has no position among
// them. It used to get one anyway: the numeric and time modes rejected it, but
// the lexicographic mode ran it through fmt.Sprint and sorted the text "<nil>",
// which lands at ASCII '<' — between the digits and the capitals. So a null in
// the data was silently filed in an arbitrary place, and whether that happened
// at all depended on the type of the other elements, since they are what chooses
// the mode. An error is the answer here for the reason it is everywhere else in
// this file: a wrong order returned as success is worse than a failure that says
// which element is at fault.
//
// Checking up front rather than inside a key function is what makes the index
// available. "element 37 is nil" is actionable against a page list in a way that
// a message naming only the fault is not, and one pass over the slice is nothing
// beside the sort that follows.
//
// A caller who wants nils tolerated has to drop them before the template — the
// package has no filter, deliberately, for the same reason it has no groupBy.
func requireNoNils(fn string, elems []any) error {
	for i, e := range elems {
		if isNilValue(e) {
			return fmt.Errorf("%s: element %d is nil", fn, i)
		}
	}
	return nil
}

// inferSortMode inspects the first element of elems to decide how to sort.
//
// The first element is enough because requireNoNils has already run, so nothing
// here is nil. That check replaced a loop that skipped nils to find something
// informative, which only ever skipped an untyped one: a (*Page)(nil) is not
// == nil, so it fell through to sortLex and quietly ordered a list of numbers as
// text — the very failure the paragraph below is about, reached from the other
// side.
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
	if len(elems) == 0 {
		return sortLex
	}
	e := elems[0]
	if asNumber(e).kind != notNumeric {
		return sortNumeric
	}
	if _, ok := e.(time.Time); ok {
		return sortTime
	}
	return sortLex
}

// Sort returns a new slice sorted by type:
//   - numeric types (int, float64, etc.) sort numerically
//   - time.Time values sort chronologically
//   - everything else sorts lexicographically (string comparison)
//
// For []any, the first element determines the sort mode.
// A nil element is an error in every mode, as is a nil value under key.
// An optional key names a field on each element (always lexicographic); the
// element may be a struct, a pointer to one, or a map with string-kind keys.
// It is an error if any element cannot supply that field — see fieldValue.
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
	if err := requireNoNils("sort", out); err != nil {
		return nil, err
	}
	if len(key) > 0 {
		k := key[0]
		return sortByKey(out, func(e any) (string, error) {
			s, err := fieldString(e, k)
			if err != nil {
				return "", fmt.Errorf("sort: %w", err)
			}
			return s, nil
		}, cmp.Compare[string])
	}
	switch inferSortMode(out) {
	case sortNumeric:
		return sortByKey(out, func(e any) (number, error) {
			n, err := toNumber(e)
			if err != nil {
				return number{}, fmt.Errorf("sort: cannot convert element to number: %w", err)
			}
			return n, nil
		}, number.compare)
	case sortTime:
		return sortByKey(out, func(e any) (time.Time, error) {
			t, ok := e.(time.Time)
			if !ok {
				return time.Time{}, fmt.Errorf("sort: element is not time.Time, got %T", e)
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
// An optional key names a field on each element, which may be a struct, a
// pointer to one, or a map with string-kind keys. It is an error if any element
// cannot be converted, or — with a key — cannot supply that field.
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
	if err := requireNoNils("sortNum", out); err != nil {
		return nil, err
	}
	if len(key) > 0 {
		k := key[0]
		return sortByKey(out, func(e any) (number, error) {
			n, err := fieldNumber(e, k)
			if err != nil {
				return number{}, fmt.Errorf("sortNum: %w", err)
			}
			return n, nil
		}, number.compare)
	}
	return sortByKey(out, func(e any) (number, error) {
		n, err := toNumber(e)
		if err != nil {
			return number{}, fmt.Errorf("sortNum: cannot convert element to number: %w", err)
		}
		return n, nil
	}, number.compare)
}

// Where filters a slice by field equality. Only elements whose named field
// equals val are included in the result. An element may be a struct, a pointer
// to one, or a map with string-kind keys, so a plain []Page filters as readily
// as a []map[string]any decoded from JSON.
//
// Numbers compare by value regardless of type, so a field decoded from JSON as
// float64 matches an integer literal, and strings compare by their text across
// string kinds, so a field declared as a named type matches a plain literal. It
// is an error if any element cannot supply the named field — a missing one would
// otherwise drop every element and look like data that simply did not match.
//
//	where $pages "Draft" false    → pages where Draft == false
//	where $pages "Section" "blog" → pages in the blog section, whether Section is string or a named type
//	where $pages "Weight" 1       → matches whether Weight is int, int64, or float64
func Where(v any, key string, val any) ([]any, error) {
	elems, err := toSlice(v)
	if err != nil {
		return nil, fmt.Errorf("where: %w", err)
	}
	out := make([]any, 0, len(elems))
	for _, elem := range elems {
		// A missing field is an error, not a non-match: it would quietly drop
		// the element, making a typo in the key indistinguishable from data that
		// matched nothing. Sort reports this the same way.
		got, err := fieldValue(elem, key)
		if err != nil {
			return nil, fmt.Errorf("where: %w", err)
		}
		if valuesEqual(got, val) {
			out = append(out, elem)
		}
	}
	return out, nil
}

// --- map operations ---

// stringKeyedMap returns the reflect value of a map with string-kind keys.
//
// Any value type is accepted — map[string]string and map[string]int are as
// ordinary in template data as map[string]any — and so is a named key type
// (map[Slug]int), because reflect classifies by kind the way asString and
// isSequence do.
//
// The key kind must be a string, which is where this stops short of In. In
// accepts any key type because it only probes for one key; keys and values have
// to put the keys in an order, and there is no ordering rule here that spans
// kinds. Widening that means deciding how an int key sorts against a string one,
// and changing what Keys returns.
//
// It validates and nothing more. Ordering belongs to the two callers that need
// it, because merge does not: it writes every key into a map, where the order
// they arrive in cannot be observed, and used to pay for a sort to produce it.
// The error is returned unprefixed, so each caller names itself — merge is the
// reason, since only it has an argument index to report, and formatting one
// eagerly to pass in meant building that string on every successful call.
func stringKeyedMap(v any) (reflect.Value, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return reflect.Value{}, fmt.Errorf("expected a map, got %T", v)
	}
	if kt := rv.Type().Key(); kt.Kind() != reflect.String {
		return reflect.Value{}, fmt.Errorf("map has %s keys, want string", kt)
	}
	return rv, nil
}

// sortedMapKeys returns rv's keys in order, as reflect values, for a caller
// that has to look each one up again.
//
// Keeping the reflect values is what lets Values index the map with a key
// directly. Handing back a []string meant converting each one back —
// reflect.ValueOf(k).Convert(kt) — to reach the entry it had just come from,
// once per key, for a map this function had already walked.
//
// It costs a String call per comparison rather than per key, which is why Keys
// does not use it: that one wants the names themselves, so it sorts them as
// strings and never pays to look anything up.
//
// reflect.Value.MapKeys allocates its result, so this is never nil.
func sortedMapKeys(rv reflect.Value) []reflect.Value {
	keys := rv.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int {
		return cmp.Compare(a.String(), b.String())
	})
	return keys
}

// Keys returns the keys of a map in sorted order. The map may have any value
// type and any string-kind key type.
//
//	keys map[string]any{"b": 2, "a": 1}    → ["a" "b"]
//	keys map[string]string{"b": "2"}       → ["b"]
func Keys(v any) ([]string, error) {
	// The concrete case is the common one and skips reflect entirely.
	if m, ok := v.(map[string]any); ok {
		if len(m) == 0 {
			// slices.Sorted collects into a nil slice, so an empty map would
			// yield nil rather than an empty slice.
			return []string{}, nil
		}
		return slices.Sorted(maps.Keys(m)), nil
	}
	rv, err := stringKeyedMap(v)
	if err != nil {
		return nil, fmt.Errorf("keys: %w", err)
	}
	// Sorted as strings, not through sortedMapKeys. Keys wants the names and
	// nothing else, so extracting them first costs one String call per key,
	// where sorting the reflect values costs one per comparison — measurably
	// slower for the only caller that has no use for the reflect key.
	// make, not append to nil: an empty map must yield an empty slice.
	mk := rv.MapKeys()
	out := make([]string, len(mk))
	for i, k := range mk {
		out[i] = k.String()
	}
	slices.Sort(out)
	return out, nil
}

// Values returns the values of a map in key-sorted order. The map may have any
// value type and any string-kind key type.
//
//	values map[string]any{"b": 2, "a": 1} → [1 2]
func Values(v any) ([]any, error) {
	if m, ok := v.(map[string]any); ok {
		ks := slices.Sorted(maps.Keys(m))
		out := make([]any, len(ks))
		for i, k := range ks {
			out[i] = m[k]
		}
		return out, nil
	}
	rv, err := stringKeyedMap(v)
	if err != nil {
		return nil, fmt.Errorf("values: %w", err)
	}
	keys := sortedMapKeys(rv)
	out := make([]any, len(keys))
	for i, k := range keys {
		out[i] = rv.MapIndex(k).Interface()
	}
	return out, nil
}

// MergeMaps combines maps into a new map[string]any. Later maps win on key
// collision. Registered as "merge" in the template FuncMap.
//
// Arguments may have any value type and any string-kind key type, and need not
// agree with each other: merging a map[string]string with a map[string]any is
// how a typed config map and a dict literal combine. Values widen to any, which
// is why the result is always map[string]any whatever went in.
//
//	merge (dict "a" 1 "b" 2) (dict "b" 99 "c" 3) → {"a":1, "b":99, "c":3}
func MergeMaps(mapsIn ...any) (map[string]any, error) {
	out := make(map[string]any)
	for i, v := range mapsIn {
		if m, ok := v.(map[string]any); ok {
			maps.Copy(out, m)
			continue
		}
		rv, err := stringKeyedMap(v)
		if err != nil {
			return nil, fmt.Errorf("merge: argument %d: %w", i, err)
		}
		// Ranged rather than keyed: the result is a map, so the order these
		// arrive in cannot be observed, and no key slice needs to exist.
		for iter := rv.MapRange(); iter.Next(); {
			out[iter.Key().String()] = iter.Value().Interface()
		}
	}
	return out, nil
}

// --- general ---

// mapKey converts val into a value usable as a key of a map whose key type is
// kt. The second result reports whether val can name a key of that type at all;
// false means the search can only come up empty.
//
// It cannot fail, which is the whole of the rule it follows: a needle no key of
// this type could hold is absent, never an error. A nil, a struct against a
// string key, an int against a string key — each used to be reported as a type
// error, while the slice branch answered false for the same needle against the
// same element type. in $m nil failed the render where in $list nil returned
// false two lines away, and neither one had found anything. The haystack asks
// the question; the needle only answers it.
//
// The case for erroring was that a needle of unrelated kind is likely a typo.
// That is true of the slice branch equally, and it is not the trade this package
// makes there — nor could it be, since a []any legitimately holds mixed types.
// One rule, applied to both, beats a diagnostic that only half the containers
// offer.
//
// Assignability alone is too strict to be the rule. It is what made a
// map[Slug]int unsearchable — keys, values, merge, and fieldValue all reach a
// named key type by converting — and what made in $m 1 fail on a map[int64]any
// while in $list 1 matched an int64 element two lines away. The reason numbers
// compare across types in the first place is that the type a number arrives as
// is an accident of the decoder, and a map key is no less subject to that than
// an element.
//
// Conversion is gated on the kinds rather than on reflect's ConvertibleTo, which
// is far too permissive here: int converts to string in Go, so a
// ConvertibleTo-based rule would quietly turn in $stringMap 65 into a search for
// "A". Only string kind into string kind, and numeric into numeric, are allowed.
//
// A numeric conversion must also round-trip. Converting 1.5 to an int64 key
// truncates, and converting 1e300 is implementation-defined, so either would
// name some unrelated key; the result is checked against the original with the
// same number.equal the slice branch uses. A needle that does not survive the
// trip is absent rather than an error, because in $int64List 1.5 is likewise
// false rather than a failure.
func mapKey(kt reflect.Type, val any) (reflect.Value, bool) {
	kv := reflect.ValueOf(val)
	if !kv.IsValid() {
		// An untyped nil. No map has a key it could name, including one with an
		// interface key type: a nil interface is not a valid key value.
		return reflect.Value{}, false
	}
	if kv.Type().AssignableTo(kt) {
		return kv, true
	}
	numeric := numKindOf(kv.Kind()) != notNumeric && numKindOf(kt.Kind()) != notNumeric
	stringly := kv.Kind() == reflect.String && kt.Kind() == reflect.String
	if numeric || stringly {
		converted := kv.Convert(kt)
		if numeric && !asNumber(converted.Interface()).equal(asNumber(val)) {
			return reflect.Value{}, false
		}
		return converted, true
	}
	return reflect.Value{}, false
}

// In reports whether val is present in v.
//
//   - slice or array: element membership; numbers compare by value across
//     types and strings by their text across string kinds, everything else via
//     reflect.DeepEqual
//
//   - map: key existence; a named key type and a number of any width match,
//     on the same terms elements do. A needle that could not name a key of that
//     type — a nil, a struct, an int against string keys — is absent rather than
//     an error, exactly as it is for a slice
//
//   - string: substring search; val must be a string, and is the one needle
//     reported as an error rather than as a miss
//
//     in (list "a" "b" "c") "b"          → true
//     in (dict "x" 1) "x"                → true
//     in "hello world" "world"            → true
func In(v, val any) (bool, error) {
	if v == nil {
		return false, nil
	}
	// Strings go through asString rather than this function's own kind check, so
	// that what counts as a string is defined in one place for in, first, last,
	// take, and drop alike. The needle is read the same way as the haystack, so
	// a named type works on either side of the search.
	//
	// This is the one branch that reports a needle of the wrong shape as an
	// error, and the exception is deliberate. A slice and a map are containers
	// of values, so a needle they cannot hold is one they do not hold; a string
	// is not a container of values at all, and "is 42 inside this text" has no
	// answer to return rather than a false one. Coercing the needle is the only
	// way to give it one, and that answer would be worse than the error: it
	// would make in "a42b" 42 true.
	if s, ok := asString(v); ok {
		sub, ok := asString(val)
		if !ok {
			return false, fmt.Errorf("in: string search requires string value, got %T", val)
		}
		return strings.Contains(s, sub), nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
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
		// key type, so the key is built before indexing. Any key type is
		// supported, not just string: in $m 1 works on a map[int]any. A needle
		// that cannot name a key of that type is absent, exactly as it is for a
		// slice of the same element type — see mapKey.
		kv, ok := mapKey(rv.Type().Key(), val)
		if !ok {
			return false, nil
		}
		return rv.MapIndex(kv).IsValid(), nil
	default:
		return false, fmt.Errorf("in: expected a slice, array, map, or string, got %T", v)
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
