package doublebrace

import (
	"cmp"
	"math"
	"reflect"
)

// This file is the package's value-classification and cross-type comparison
// engine: what counts as a sequence, a string, nil, or a number, and how two
// numbers or two strings compare when their concrete types differ. It exists
// because template data loses type precision that Go code would keep — JSON
// decodes a number as float64, a struct field might be a named type, an
// html/template value wraps a plain string — and every function elsewhere in
// the package that classifies or compares a value does it by calling in here,
// so a type is never a number to one function and text to the next.
//
// collections.go, math.go, and strings.go all call into this file rather than
// repeating these checks locally: isSequence is what lets min, max, and the
// collection functions agree on what a sequence is; asString is what lets
// every text-taking function in the FuncMap accept a named string type or an
// html/template value the same way.

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
