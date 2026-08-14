package doublebrace

import (
	"math"
	"testing"
	"time"

	// Aliased for the reason collections_test.go aliases it: text/template is
	// bound to the name template elsewhere in this test package.
	htmltemplate "html/template"
)

// compare must agree with equal wherever equal reports true, and must be a
// consistent ordering — otherwise sort and where disagree about the same pair.
func TestNumberCompareAgreesWithEqual(t *testing.T) {
	vals := []any{
		-1, 0, 1, 2, int8(1), int64(-1), int64(math.MaxInt64), int64(math.MinInt64),
		uint(0), uint(1), uint64(math.MaxUint64), uint64(math.MaxInt64),
		-1.5, 0.0, 1.0, 1.5, 2.0, float32(1.5),
		9007199254740992.0, int64(9007199254740993),
		math.Inf(1), math.Inf(-1),
	}
	for _, a := range vals {
		for _, b := range vals {
			na, nb := asNumber(a), asNumber(b)
			gotCmp := na.compare(nb)
			if eq := na.equal(nb); eq != (gotCmp == 0) {
				t.Errorf("(%v %T, %v %T): equal=%v but compare=%d", a, a, b, b, eq, gotCmp)
			}
			// Antisymmetry: reversing the operands negates the result.
			if rev := nb.compare(na); rev != -gotCmp {
				t.Errorf("(%v %T, %v %T): compare=%d but reversed=%d", a, a, b, b, gotCmp, rev)
			}
		}
	}
}

func TestValuesEqual_numericAcrossTypes(t *testing.T) {
	cases := []struct {
		a, b any
		want bool
	}{
		// integer widths and named types unify, as eq does
		{int(1), int64(1), true},
		{int64(1), int(1), true},
		{int32(1), int8(1), true},
		{Weight(1), int(1), true},
		{uint(1), int(1), true},
		{uint8(1), uint64(1), true},
		// int/float, which eq rejects outright
		{float64(1), int(1), true},
		{int(1), float64(1), true},
		{float32(1), int64(1), true},
		{float64(1.5), int(1), false},
		{float64(1.5), float64(1.5), true},
		// signedness edges
		{int(-1), uint64(1), false},
		{int(-1), float64(-1), true},
		{float64(-1), uint(1), false},
		// unequal values of differing types stay unequal
		{int(1), int64(2), false},
		{float64(2), int(1), false},
		// non-numbers are untouched
		{"1", int(1), false}, // a string is not a number here
		{"a", "a", true},
		{true, int(1), false},
		{nil, int(0), false},
		{[]any{1}, []any{1}, true},
	}
	for _, c := range cases {
		if got := valuesEqual(c.a, c.b); got != c.want {
			t.Errorf("valuesEqual(%#v, %#v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// asString decides what a string is everywhere else in the package, so equality
// reads one the same way. reflect.DeepEqual is type-strict, and while it held
// this alone a named type never equalled the literal it was written from: the
// filter came back empty, with no error, which is the outcome the missing-field
// check in Where exists to prevent — reached through a field that is present.
func TestValuesEqual_stringsAcrossKinds(t *testing.T) {
	type Section string
	cases := []struct {
		a, b any
		want bool
	}{
		{Section("blog"), "blog", true},
		{"blog", Section("blog"), true},
		{Section("blog"), Slug("blog"), true}, // two named types, same text
		{Section("blog"), "docs", false},
		{htmltemplate.HTML("<b>"), "<b>", true},
		{"", Section(""), true},
		// A string is still not a number, and not any other kind.
		{Section("1"), 1, false},
		{Section("true"), true, false},
		{Section("x"), nil, false},
		{Section("x"), []any{"x"}, false},
	}
	for _, c := range cases {
		if got := valuesEqual(c.a, c.b); got != c.want {
			t.Errorf("valuesEqual(%#v, %#v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// Comparing must narrow the float to an integer, not widen the integer to a
// float: float64 cannot represent every int64, so widening would report two
// distinct IDs as equal.
func TestValuesEqual_largeIntegersAreExact(t *testing.T) {
	const big = int64(9007199254740993) // 2^53+1, not representable as float64
	if valuesEqual(big, float64(big)) {
		t.Error("valuesEqual(2^53+1, float64(2^53+1)) = true; float64 cannot hold that value")
	}
	if !valuesEqual(big, big-0) {
		t.Error("valuesEqual should still match an identical int64")
	}
	if valuesEqual(big, big-1) {
		t.Error("distinct int64 values must not compare equal")
	}
	// Values that are exactly representable still match.
	const exact = int64(1) << 52
	if !valuesEqual(exact, float64(exact)) {
		t.Error("an exactly representable integer should match its float64")
	}
	// Out-of-range floats never match an integer.
	if valuesEqual(int64(0), 1e300) {
		t.Error("an out-of-range float must not match")
	}
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if valuesEqual(int64(0), f) {
			t.Errorf("valuesEqual(0, %v) = true, want false", f)
		}
	}
}

func TestIsZero_structs(t *testing.T) {
	type page struct {
		Title string
		Count int
	}

	cases := []struct {
		name string
		v    any
		want bool
	}{
		{"zero time", time.Time{}, true},
		{"zero time in UTC", time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC), true},
		// t.In and t.Local set the location field, so the struct is no longer
		// all-zero by reflection even though the time itself still is. The
		// IsZero method is what gets these right.
		{"zero time in a fixed zone", time.Time{}.In(time.FixedZone("X", 3600)), true},
		{"zero time localized", time.Time{}.Local(), true},
		{"non-zero time", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), false},
		{"empty struct type", struct{}{}, true},
		{"struct with zero fields", page{}, true},
		{"struct with one field set", page{Title: "x"}, false},
		{"struct with zero-valued field set", page{Count: 0}, true},
		{"pointer to zero struct", &page{}, false}, // a non-nil pointer is not zero
	}
	for _, c := range cases {
		if got := isZero(c.v); got != c.want {
			t.Errorf("%s: isZero(%#v) = %v, want %v", c.name, c.v, got, c.want)
		}
	}
}

// Arrays go by zero-ness rather than emptiness, so that isZero's stated
// definition — the zero value for its type — holds for them. An array's length
// is fixed, so the emptiness test slices and maps use would mean an array can
// only ever be zero when its type has length 0.
func TestIsZero_arrays(t *testing.T) {
	type page struct{ Title string }

	cases := []struct {
		name string
		v    any
		want bool
	}{
		{"zero-length array", [0]int{}, true},
		{"all-zero ints", [3]int{}, true},
		{"all-zero ints written out", [3]int{0, 0, 0}, true},
		{"first element set", [3]int{1, 0, 0}, false},
		{"last element set", [3]int{0, 0, 1}, false},
		{"all-empty strings", [2]string{}, true},
		{"one non-empty string", [2]string{"a", ""}, false},
		{"array of zero structs", [2]page{}, true},
		{"array of one non-zero struct", [2]page{{Title: "x"}}, false},
	}
	for _, c := range cases {
		if got := isZero(c.v); got != c.want {
			t.Errorf("%s: isZero(%#v) = %v, want %v", c.name, c.v, got, c.want)
		}
	}

	// Slices keep emptiness semantics: unlike an array, a slice of three zeros
	// is a different thing from an empty slice.
	if isZero([]int{0, 0, 0}) {
		t.Error("isZero([]int{0,0,0}) = true, want false: slices go by length")
	}
	if !isZero([]int{}) {
		t.Error("isZero([]int{}) = false, want true")
	}
}

// The struct change must not disturb the kinds isZero already handled.
func TestIsZero_nonStructsUnchanged(t *testing.T) {
	cases := []struct {
		v    any
		want bool
	}{
		{nil, true},
		{false, true}, {true, false},
		{0, true}, {42, false}, {-1, false},
		{uint(0), true}, {uint(1), false},
		{0.0, true}, {1.5, false},
		{"", true}, {"x", false},
		{[]any{}, true}, {[]int{1}, false},
		{map[string]any{}, true}, {map[string]any{"a": 1}, false},
		{(*int)(nil), true},
		{time.Duration(0), true}, {time.Second, false},
	}
	for _, c := range cases {
		if got := isZero(c.v); got != c.want {
			t.Errorf("isZero(%#v) = %v, want %v", c.v, got, c.want)
		}
	}
}

// Kinds that no template is likely to carry, pinned because the per-kind switch
// these replaced answered "not zero" for every one of them — a nil chan and a
// nil func included. Deferring to reflect.Value.IsZero is what makes the answer
// follow the documented definition rather than the list of kinds someone
// remembered to write down.
func TestIsZero_uncommonKinds(t *testing.T) {
	nilChan := (chan int)(nil)
	openChan := make(chan int)
	nilFunc := (func())(nil)

	cases := []struct {
		name string
		v    any
		want bool
	}{
		{"nil chan", nilChan, true},
		{"open chan", openChan, false},
		{"nil func", nilFunc, true},
		{"non-nil func", func() {}, false},
		{"zero complex", complex(0, 0), true},
		{"non-zero complex", complex(1, 0), false},
	}
	for _, c := range cases {
		if got := isZero(c.v); got != c.want {
			t.Errorf("%s: isZero = %v, want %v", c.name, got, c.want)
		}
	}
}

// nilIsZeroer has a value-receiver IsZero, so *nilIsZeroer satisfies the same
// interface time.Time does and dereferences its receiver when called.
type nilIsZeroer struct{ n int }

func (z nilIsZeroer) IsZero() bool { return z.n == 0 }

// A nil pointer must report as zero rather than panic. Its type can carry an
// IsZero method — a value receiver is promoted into the pointer's method set —
// and calling that method dereferences the nil pointer. An optional date held
// as a *time.Time is the case that motivates this: {{ default "TBD" .Date }}
// failed the render instead of falling back.
func TestIsZero_nilPointerWithIsZeroMethod(t *testing.T) {
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	zeroDate := time.Time{}

	cases := []struct {
		name string
		v    any
		want bool
	}{
		{"nil *time.Time", (*time.Time)(nil), true},
		{"nil *nilIsZeroer", (*nilIsZeroer)(nil), true},
		// A non-nil pointer still consults the method, which is what keeps a
		// pointer to a zero time reporting zero.
		{"pointer to zero time", &zeroDate, true},
		{"pointer to a real date", &date, false},
		{"pointer to zero nilIsZeroer", &nilIsZeroer{}, true},
		{"pointer to non-zero nilIsZeroer", &nilIsZeroer{n: 1}, false},
	}
	for _, c := range cases {
		if got := isZero(c.v); got != c.want {
			t.Errorf("%s: isZero = %v, want %v", c.name, got, c.want)
		}
	}
}
