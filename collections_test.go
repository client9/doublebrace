package doublebrace

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

// --- constructors ---

func TestList(t *testing.T) {
	got := List("a", "b", "c")
	want := []any{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := List(); got == nil || len(got) != 0 {
		t.Errorf("List() should return empty non-nil slice")
	}
}

func TestDict(t *testing.T) {
	got, err := Dict("name", "Alice", "age", 30)
	if err != nil {
		t.Fatal(err)
	}
	if got["name"] != "Alice" || got["age"] != 30 {
		t.Errorf("unexpected map: %v", got)
	}

	if _, err := Dict("odd"); err == nil {
		t.Error("expected error for odd argument count")
	}
	if _, err := Dict(1, "v"); err == nil {
		t.Error("expected error for non-string key")
	}
}

func TestSeq(t *testing.T) {
	cases := []struct {
		args []int
		want []int
	}{
		{[]int{5}, []int{1, 2, 3, 4, 5}},
		{[]int{1}, []int{1}},
		{[]int{0}, []int{}},
		{[]int{3, 7}, []int{3, 4, 5, 6, 7}},
		{[]int{5, 5}, []int{5}},
		{[]int{7, 3}, []int{}},
		{[]int{1, 10, 2}, []int{1, 3, 5, 7, 9}},
		{[]int{5, 1, -1}, []int{5, 4, 3, 2, 1}},
		{[]int{0, 6, 3}, []int{0, 3, 6}},
	}
	for _, c := range cases {
		got, err := Seq(c.args...)
		if err != nil {
			t.Errorf("Seq(%v): unexpected error: %v", c.args, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Seq(%v) = %v, want %v", c.args, got, c.want)
		}
	}

	if _, err := Seq(); err == nil {
		t.Error("Seq(): expected error for 0 args")
	}
	if _, err := Seq(1, 2, 3, 4); err == nil {
		t.Error("Seq(1,2,3,4): expected error for 4 args")
	}
	if _, err := Seq(1, 10, 0); err == nil {
		t.Error("Seq(1,10,0): expected error for zero step")
	}
}

// seq is bounded so that a mistyped or mis-parsed bound fails fast instead of
// allocating gigabytes. The limit is on the element count, not the numeric
// range, so a wide span with a large step is still fine.
func TestSeq_lengthLimit(t *testing.T) {
	t.Run("at the limit", func(t *testing.T) {
		for _, args := range [][]int{
			{MaxSeqLen},
			{1, MaxSeqLen},
			{0, 2*MaxSeqLen - 2, 2},
		} {
			got, err := Seq(args...)
			if err != nil {
				t.Errorf("Seq(%v): unexpected error: %v", args, err)
				continue
			}
			if len(got) != MaxSeqLen {
				t.Errorf("Seq(%v) returned %d elements, want %d", args, len(got), MaxSeqLen)
			}
		}
	})

	t.Run("one past the limit", func(t *testing.T) {
		for _, args := range [][]int{
			{MaxSeqLen + 1},
			{1, MaxSeqLen + 1},
			{0, 2 * MaxSeqLen, 2},
		} {
			if got, err := Seq(args...); err == nil {
				t.Errorf("Seq(%v) returned %d elements, want an error", args, len(got))
			}
		}
	})

	t.Run("wide range with a large step stays under the limit", func(t *testing.T) {
		got, err := Seq(math.MinInt, math.MaxInt, math.MaxInt)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []int{math.MinInt, -1, math.MaxInt - 1}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Seq(MinInt, MaxInt, MaxInt) = %v, want %v", got, want)
		}
	})
}

// The element count must be computed in a width that cannot overflow. Before
// seqLen, end-start+1 wrapped: seq (math.MinInt) (math.MaxInt) computed a count
// of 0 and silently returned nothing rather than refusing an impossible request.
func TestSeq_countDoesNotOverflow(t *testing.T) {
	cases := [][]int{
		{math.MinInt, math.MaxInt},
		{math.MinInt, math.MaxInt, 1},
		{math.MaxInt, math.MinInt, -1},
		{math.MinInt + 1, math.MaxInt},
		{0, math.MaxInt},
		{math.MinInt, 0},
	}
	for _, args := range cases {
		got, err := Seq(args...)
		if err == nil {
			t.Errorf("Seq(%v) returned %d elements, want an error", args, len(got))
		}
	}
}

// A step that overflows past end must not restart the loop. The old
// condition-driven loop appended forever here, because v += step wrapped from
// MaxInt-1 around to MinInt and satisfied v <= end again.
func TestSeq_stepOverflowTerminates(t *testing.T) {
	cases := []struct {
		args []int
		want []int
	}{
		{[]int{math.MaxInt - 1, math.MaxInt, 2}, []int{math.MaxInt - 1}},
		{[]int{math.MaxInt - 1, math.MaxInt, math.MaxInt}, []int{math.MaxInt - 1}},
		{[]int{math.MinInt + 1, math.MinInt, -2}, []int{math.MinInt + 1}},
		{[]int{math.MinInt, math.MinInt, math.MinInt}, []int{math.MinInt}},
	}
	for _, c := range cases {
		got, err := Seq(c.args...)
		if err != nil {
			t.Errorf("Seq(%v): unexpected error: %v", c.args, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Seq(%v) = %v, want %v", c.args, got, c.want)
		}
	}
}

// --- sequence access ---

func TestFirst(t *testing.T) {
	got, err := First([]int{10, 20, 30})
	if err != nil || got != 10 {
		t.Errorf("first slice: got %v, %v", got, err)
	}
	got, err = First("café")
	if err != nil || got != "c" {
		t.Errorf("first string: got %v, %v", got, err)
	}
	if _, err := First([]int{}); err == nil {
		t.Error("expected error for empty slice")
	}
	if _, err := First(""); err == nil {
		t.Error("expected error for empty string")
	}
}

func TestLast(t *testing.T) {
	got, err := Last([]int{10, 20, 30})
	if err != nil || got != 30 {
		t.Errorf("last slice: got %v, %v", got, err)
	}
	got, err = Last("café")
	if err != nil || got != "é" {
		t.Errorf("last string: got %v, %v", got, err)
	}
	if _, err := Last([]int{}); err == nil {
		t.Error("expected error for empty slice")
	}
}

func TestTake(t *testing.T) {
	cases := []struct {
		v    any
		n    int
		want any
	}{
		{[]int{1, 2, 3, 4, 5}, 3, []any{1, 2, 3}},
		{[]int{1, 2, 3}, 0, []any{}},
		{[]int{1, 2, 3}, 10, []any{1, 2, 3}},    // n > len: clamp
		{[]int{1, 2, 3, 4, 5}, -2, []any{4, 5}}, // last 2
		{[]int{1, 2, 3}, -10, []any{1, 2, 3}},   // |n| > len: clamp
		{"hello", 3, "hel"},
		{"日本語", 2, "日本"},     // rune-aware
		{"hi", 10, "hi"},     // n > len: clamp
		{"日本語", -1, "語"},     // last rune
		{"hello", -3, "llo"}, // last 3 runes
	}
	for _, c := range cases {
		got, err := Take(c.v, c.n)
		if err != nil {
			t.Errorf("Take(%v, %d): %v", c.v, c.n, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Take(%v, %d) = %v, want %v", c.v, c.n, got, c.want)
		}
	}
}

func TestDrop(t *testing.T) {
	cases := []struct {
		v    any
		n    int
		want any
	}{
		{[]int{1, 2, 3, 4, 5}, 2, []any{3, 4, 5}},
		{[]int{1, 2, 3}, 0, []any{1, 2, 3}},
		{[]int{1, 2, 3}, 10, []any{}},              // n > len: empty
		{[]int{1, 2, 3, 4, 5}, -2, []any{1, 2, 3}}, // remove last 2
		{[]int{1, 2, 3}, -10, []any{}},             // |n| > len: empty
		{"hello", 2, "llo"},
		{"日本語", 1, "本語"},    // rune-aware
		{"hi", 10, ""},      // n > len: empty
		{"日本語", -1, "日本"},   // remove last rune
		{"hello", -3, "he"}, // remove last 3 runes
	}
	for _, c := range cases {
		got, err := Drop(c.v, c.n)
		if err != nil {
			t.Errorf("Drop(%v, %d): %v", c.v, c.n, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Drop(%v, %d) = %v, want %v", c.v, c.n, got, c.want)
		}
	}
}

// --- sequence transformation ---

func TestReverse(t *testing.T) {
	got, err := Reverse([]int{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	want := []any{3, 2, 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// original must be unmodified
	orig := []int{1, 2, 3}
	if _, err := Reverse(orig); err != nil {
		t.Errorf("got error %v, expected none", err)
	}
	if orig[0] != 1 {
		t.Error("Reverse must not modify original slice")
	}
}

func TestCompact(t *testing.T) {
	cases := []struct {
		in   []any
		want []any
	}{
		{[]any{1, 1, 2, 3, 3, 1}, []any{1, 2, 3, 1}}, // only consecutive
		{[]any{"a", "a", "b"}, []any{"a", "b"}},
		{[]any{1, 2, 3}, []any{1, 2, 3}}, // no dups
		{[]any{}, []any{}},
	}
	for _, c := range cases {
		got, err := Compact(c.in)
		if err != nil {
			t.Errorf("Compact(%v): %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Compact(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestConcat(t *testing.T) {
	got, err := Concat([]int{1, 2}, []int{3, 4}, []int{5})
	if err != nil {
		t.Fatal(err)
	}
	want := []any{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	// error on non-slice
	if _, err := Concat("not a slice"); err == nil {
		t.Error("expected error for non-slice argument")
	}
}

// Concat must return an empty slice rather than nil, so that its result behaves
// like every other collection result — and like list — for callers that check
// for nil or append to it.
func TestConcat_alwaysNonNil(t *testing.T) {
	cases := []struct {
		name string
		in   []any
	}{
		{"no arguments", nil},
		{"one empty slice", []any{[]any{}}},
		{"several empty slices", []any{[]any{}, []int{}, []string{}}},
	}
	for _, c := range cases {
		got, err := Concat(c.in...)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if got == nil {
			t.Errorf("%s: Concat returned nil, want empty non-nil slice", c.name)
		}
		if len(got) != 0 {
			t.Errorf("%s: Concat = %v, want empty", c.name, got)
		}
	}
}

func TestConcat_mixedSliceTypes(t *testing.T) {
	got, err := Concat([]int{1, 2}, []string{"a"}, []any{true, nil}, Titles{"t"})
	if err != nil {
		t.Fatal(err)
	}
	want := []any{1, 2, "a", true, nil, "t"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestConcat_errorNamesArgumentPosition(t *testing.T) {
	_, err := Concat([]int{1}, "not a slice")
	if err == nil {
		t.Fatal("expected error for non-slice argument")
	}
	if !strings.Contains(err.Error(), "argument 1") {
		t.Errorf("error should name the offending argument position, got: %v", err)
	}
}

// Concat sizes the result from the summed argument lengths in one allocation
// rather than growing it by repeated append. This pins that.
func TestConcat_resultIsExactlySized(t *testing.T) {
	got, err := Concat([]any{1, 2}, []any{3})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != cap(got) {
		t.Errorf("len=%d cap=%d, want equal", len(got), cap(got))
	}
}

func TestSort(t *testing.T) {
	// scalar lex sort
	got, err := Sort([]string{"banana", "apple", "cherry"})
	if err != nil {
		t.Fatal(err)
	}
	want := []any{"apple", "banana", "cherry"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sort scalars: got %v, want %v", got, want)
	}

	// ISO date strings sort correctly lexicographically
	dates := []string{"2024-03-01", "2023-12-31", "2024-01-15"}
	got, err = Sort(dates)
	if err != nil {
		t.Fatal(err)
	}
	wantDates := []any{"2023-12-31", "2024-01-15", "2024-03-01"}
	if !reflect.DeepEqual(got, wantDates) {
		t.Errorf("sort dates: got %v, want %v", got, wantDates)
	}

	// slice-of-maps by key
	pages := []any{
		map[string]any{"Title": "Zebra"},
		map[string]any{"Title": "Apple"},
		map[string]any{"Title": "Mango"},
	}
	got, err = Sort(pages, "Title")
	if err != nil {
		t.Fatal(err)
	}
	gotSlice := got.([]any)
	if gotSlice[0].(map[string]any)["Title"] != "Apple" {
		t.Errorf("sort by key: first element should be Apple, got %v", gotSlice[0])
	}

	// []int sorts numerically, not lexicographically
	got, err = Sort([]int{10, 2, 30, 5})
	if err != nil {
		t.Fatal(err)
	}
	//wantInts := []any{2, 10, 30, 5} // lex would give [10 2 30 5] sorted as [10 2 30 5]→[10 2 30 5]
	// numeric order: 2 5 10 30
	wantInts := []any{2, 5, 10, 30}
	if !reflect.DeepEqual(got, wantInts) {
		t.Errorf("sort []int: got %v, want %v", got, wantInts)
	}

	// []float64 sorts numerically
	got, err = Sort([]float64{3.14, 1.0, 2.71})
	if err != nil {
		t.Fatal(err)
	}
	wantFloats := []any{1.0, 2.71, 3.14}
	if !reflect.DeepEqual(got, wantFloats) {
		t.Errorf("sort []float64: got %v, want %v", got, wantFloats)
	}

	// []time.Time sorts chronologically
	t1 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	got, err = Sort([]time.Time{t1, t2, t3})
	if err != nil {
		t.Fatal(err)
	}
	gotTimes := got.([]any)
	if gotTimes[0].(time.Time) != t2 || gotTimes[1].(time.Time) != t3 || gotTimes[2].(time.Time) != t1 {
		t.Errorf("sort []time.Time: got %v, want [%v %v %v]", gotTimes, t2, t3, t1)
	}

	// []any with int elements sorts numerically
	got, err = Sort([]any{10, 2, 30})
	if err != nil {
		t.Fatal(err)
	}
	wantAnyInts := []any{2, 10, 30}
	if !reflect.DeepEqual(got, wantAnyInts) {
		t.Errorf("sort []any ints: got %v, want %v", got, wantAnyInts)
	}

	// []any with time.Time elements sorts chronologically
	got, err = Sort([]any{t1, t2, t3})
	if err != nil {
		t.Fatal(err)
	}
	gotAnyTimes := got.([]any)
	if gotAnyTimes[0].(time.Time) != t2 {
		t.Errorf("sort []any time.Time: first should be %v, got %v", t2, gotAnyTimes[0])
	}
}

func TestSortNum(t *testing.T) {
	// numeric strings
	got, err := SortNum([]string{"10", "9", "2"})
	if err != nil {
		t.Fatal(err)
	}
	want := []any{"2", "9", "10"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sortNum: got %v, want %v", got, want)
	}

	// slice-of-maps by numeric key
	pages := []any{
		map[string]any{"Year": 2020},
		map[string]any{"Year": 2018},
		map[string]any{"Year": 2022},
	}
	got, err = SortNum(pages, "Year")
	if err != nil {
		t.Fatal(err)
	}
	gotSlice := got.([]any)
	if gotSlice[0].(map[string]any)["Year"] != 2018 {
		t.Errorf("sortNum by key: first should be 2018, got %v", gotSlice[0])
	}

	// error on non-numeric value
	if _, err := SortNum([]string{"a", "b"}); err == nil {
		t.Error("expected error for non-numeric values")
	}
}

func TestWhere(t *testing.T) {
	pages := []any{
		map[string]any{"Title": "Post A", "Draft": false, "Section": "blog"},
		map[string]any{"Title": "Post B", "Draft": true, "Section": "blog"},
		map[string]any{"Title": "Post C", "Draft": false, "Section": "news"},
	}

	// filter by bool
	got, err := Where(pages, "Draft", false)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(got.([]any)); n != 2 {
		t.Errorf("where Draft==false: expected 2, got %d", n)
	}

	// filter by string
	got, err = Where(pages, "Section", "blog")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(got.([]any)); n != 2 {
		t.Errorf("where Section==blog: expected 2, got %d", n)
	}

	// no matches → empty slice
	got, err = Where(pages, "Section", "none")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(got.([]any)); n != 0 {
		t.Errorf("where no match: expected 0, got %d", n)
	}
}

// --- map operations ---

func TestKeys(t *testing.T) {
	m := map[string]any{"b": 2, "a": 1, "c": 3}
	got, err := Keys(m)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("keys: got %v, want %v", got, want)
	}
	if _, err := Keys("not a map"); err == nil {
		t.Error("expected error for non-map")
	}
}

func TestValues(t *testing.T) {
	m := map[string]any{"b": 2, "a": 1}
	got, err := Values(m)
	if err != nil {
		t.Fatal(err)
	}
	// ordered by sorted keys: a→1, b→2
	want := []any{1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("values: got %v, want %v", got, want)
	}
}

func TestMergeMaps(t *testing.T) {
	a := map[string]any{"a": 1, "b": 2}
	b := map[string]any{"b": 99, "c": 3}
	got, err := MergeMaps(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != 1 || got["b"] != 99 || got["c"] != 3 {
		t.Errorf("merge: unexpected result %v", got)
	}
	// originals must be unmodified
	if a["b"] != 2 {
		t.Error("merge must not modify first argument")
	}
	if _, err := MergeMaps("not a map"); err == nil {
		t.Error("expected error for non-map argument")
	}
}

// --- general ---

func TestIn(t *testing.T) {
	// slice membership
	ok, err := In([]string{"a", "b", "c"}, "b")
	if err != nil || !ok {
		t.Errorf("in slice: expected true, got %v %v", ok, err)
	}
	ok, err = In([]int{1, 2, 3}, 4)
	if err != nil || ok {
		t.Errorf("in slice missing: expected false, got %v %v", ok, err)
	}

	// map key
	ok, err = In(map[string]any{"x": 1, "y": 2}, "x")
	if err != nil || !ok {
		t.Errorf("in map: expected true, got %v %v", ok, err)
	}
	ok, err = In(map[string]any{"x": 1}, "z")
	if err != nil || ok {
		t.Errorf("in map missing: expected false, got %v %v", ok, err)
	}

	// string substring
	ok, err = In("hello world", "world")
	if err != nil || !ok {
		t.Errorf("in string: expected true, got %v %v", ok, err)
	}
	ok, err = In("hello", "xyz")
	if err != nil || ok {
		t.Errorf("in string missing: expected false, got %v %v", ok, err)
	}

	// nil → false
	ok, err = In(nil, "x")
	if err != nil || ok {
		t.Errorf("in nil: expected false, got %v %v", ok, err)
	}
}

// Arrays are indexable sequences, so every collection function accepts one.
// Template data reaches them through a fixed-size struct field ([3]string), which
// was otherwise unusable with this package.
func TestCollections_acceptArrays(t *testing.T) {
	cases := []struct {
		name string
		got  func() (any, error)
		want any
	}{
		{"Sort", func() (any, error) { return Sort([3]int{3, 1, 2}) }, []any{1, 2, 3}},
		{"SortNum", func() (any, error) { return SortNum([3]string{"10", "9", "2"}) }, []any{"2", "9", "10"}},
		{"Reverse", func() (any, error) { return Reverse([3]int{1, 2, 3}) }, []any{3, 2, 1}},
		{"Compact", func() (any, error) { return Compact([4]int{1, 1, 2, 2}) }, []any{1, 2}},
		{"Take", func() (any, error) { return Take([4]int{1, 2, 3, 4}, 2) }, []any{1, 2}},
		{"Drop", func() (any, error) { return Drop([4]int{1, 2, 3, 4}, 2) }, []any{3, 4}},
		{"First", func() (any, error) { return First([3]int{7, 8, 9}) }, 7},
		{"Last", func() (any, error) { return Last([3]int{7, 8, 9}) }, 9},
		{"Concat", func() (any, error) { return Concat([2]int{1, 2}, []int{3}) }, []any{1, 2, 3}},
		{"Concat of arrays only", func() (any, error) { return Concat([2]int{1, 2}, [1]int{3}) }, []any{1, 2, 3}},
		{"In present", func() (any, error) { return In([3]int{1, 2, 3}, 2) }, true},
		{"In absent", func() (any, error) { return In([3]int{1, 2, 3}, 9) }, false},
		{"Where", func() (any, error) {
			return Where([2]any{map[string]any{"K": 1}, map[string]any{"K": 2}}, "K", 1)
		}, []any{map[string]any{"K": 1}}},
		{"empty array", func() (any, error) { return Sort([0]int{}) }, []any{}},
	}
	for _, c := range cases {
		got, err := c.got()
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %#v, want %#v", c.name, got, c.want)
		}
	}
}

// Arrays are values in Go, so an array reaching a function through an `any`
// parameter has already been copied and the original cannot be reached. The
// immutability guarantee therefore holds trivially — but only as long as nothing
// starts taking a pointer to the argument, so it is asserted rather than assumed.
func TestCollections_doNotMutateArrayInput(t *testing.T) {
	in := [4]int{3, 1, 1, 2}
	before := in

	for _, fn := range []func(any) (any, error){
		func(v any) (any, error) { return Sort(v) },
		func(v any) (any, error) { return Reverse(v) },
		func(v any) (any, error) { return Compact(v) },
		func(v any) (any, error) { return Take(v, 2) },
		func(v any) (any, error) { return Drop(v, 2) },
		func(v any) (any, error) { return Concat(v, []int{9}) },
	} {
		if _, err := fn(in); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if in != before {
		t.Errorf("array input mutated: got %v, want %v", in, before)
	}
}

func TestToSlice_rejectsNonSequences(t *testing.T) {
	for _, in := range []any{nil, 42, "a string", map[string]any{"a": 1}, struct{}{}} {
		if _, err := toSlice(in); err == nil {
			t.Errorf("toSlice(%#v): expected an error, got nil", in)
		}
	}
}

// Sorting by key must fail loudly on elements it cannot read. Returning the
// input untouched would look like a successful sort while producing wrong
// output — see fieldString.
func TestSort_keyErrors(t *testing.T) {
	type page struct{ Title string }

	cases := []struct {
		name string
		in   []any
	}{
		{"structs, not maps", []any{page{"Zebra"}, page{"Apple"}}},
		{"single struct element", []any{page{"Zebra"}}},
		{"scalars", []any{"Zebra", "Apple"}},
		{"key absent from element", []any{
			map[string]any{"Title": "Zebra"},
			map[string]any{"Other": "Apple"},
		}},
		{"key absent from single element", []any{map[string]any{"Other": "Apple"}}},
	}
	for _, c := range cases {
		got, err := Sort(c.in, "Title")
		if err == nil {
			t.Errorf("%s: Sort(%v, \"Title\") expected an error, got %v", c.name, c.in, got)
		}
	}
}

// sortNum must reject what it cannot convert regardless of how many elements it
// was given. Validating inside the comparator made this length-dependent:
// slices.SortStableFunc never calls a comparator on fewer than two elements, so
// a one-element slice of unconvertible values was silently accepted while the
// same template broke as soon as a second element arrived.
func TestSortNum_errorsAreNotLengthDependent(t *testing.T) {
	type page struct{ Year int }

	cases := []struct {
		name string
		in   []any
		key  []string
	}{
		{"one struct, keyed", []any{page{2020}}, []string{"Year"}},
		{"two structs, keyed", []any{page{2020}, page{2021}}, []string{"Year"}},
		{"one map missing the key", []any{map[string]any{"X": 1}}, []string{"Year"}},
		{"two maps missing the key", []any{
			map[string]any{"X": 1}, map[string]any{"X": 2},
		}, []string{"Year"}},
		{"one map with a non-numeric field", []any{
			map[string]any{"Year": "not a number"},
		}, []string{"Year"}},
		{"one non-numeric value", []any{"abc"}, nil},
		{"two non-numeric values", []any{"abc", "def"}, nil},
		{"one non-numeric among numbers", []any{"abc", 1, 2}, nil},
	}
	for _, c := range cases {
		got, err := SortNum(c.in, c.key...)
		if err == nil {
			t.Errorf("%s: SortNum(%v, %v) = %v, want an error", c.name, c.in, c.key, got)
		}
	}
}

// Sort and SortNum sit next to each other and must agree on bad input, since
// that is how the two drifted apart in the first place.
func TestSortAndSortNum_agreeOnBadInput(t *testing.T) {
	type page struct{ N int }
	in := []any{page{1}} // single element: the case that used to differ

	if _, err := Sort(in, "N"); err == nil {
		t.Error("Sort([1 struct], \"N\"): expected an error")
	}
	if _, err := SortNum(in, "N"); err == nil {
		t.Error("SortNum([1 struct], \"N\"): expected an error")
	}
}

func TestSortNum_emptyInput(t *testing.T) {
	// Nothing to convert, so nothing to fail on.
	for _, key := range [][]string{nil, {"Year"}} {
		got, err := SortNum([]any{}, key...)
		if err != nil {
			t.Errorf("SortNum([], %v) = %v, want no error", key, err)
			continue
		}
		if s := got.([]any); len(s) != 0 {
			t.Errorf("SortNum([], %v) = %v, want empty", key, s)
		}
	}
}

func TestSort_keyMissingOnEmptyInput(t *testing.T) {
	// Nothing to read a field from, so nothing to fail on.
	got, err := Sort([]any{}, "Title")
	if err != nil {
		t.Errorf("Sort([], \"Title\") = %v, want no error", err)
	}
	if s := got.([]any); len(s) != 0 {
		t.Errorf("Sort([], \"Title\") = %v, want empty", s)
	}
}

// In must not panic when the search value cannot be used as a key for the map's
// key type — reflect.Value.MapIndex panics on an unassignable key.
func TestIn_mapKeyTypes(t *testing.T) {
	// Non-string key types are supported.
	ok, err := In(map[int]string{1: "a"}, 1)
	if err != nil || !ok {
		t.Errorf("in map[int]string with int key: expected true, got %v %v", ok, err)
	}
	ok, err = In(map[int]string{1: "a"}, 2)
	if err != nil || ok {
		t.Errorf("in map[int]string missing key: expected false, got %v %v", ok, err)
	}

	// Mismatched key types are an error, not a panic.
	for _, c := range []struct {
		name string
		m    any
		val  any
	}{
		{"string key against map[int]", map[int]string{1: "a"}, "1"},
		{"int key against map[string]", map[string]any{"x": 1}, 1},
		{"nil key", map[string]any{"x": 1}, nil},
	} {
		ok, err := In(c.m, c.val)
		if err == nil {
			t.Errorf("%s: expected an error, got %v", c.name, ok)
		}
	}
}

// No function returns nil for a successful call: an empty result is an empty
// slice or map. Templates cannot distinguish the two — range, len, and index
// treat them identically — but encoding/json can, so a nil result makes jsonify
// emit null where a script expects []. The enforcement points are toSlice, which
// every slice-taking function goes through, and Keys, which does not; the rest
// hold the rule by construction and are pinned here so they keep holding it.
func TestFunctionsReturnEmptyNotNil(t *testing.T) {
	isNilResult := func(v any) bool {
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return true
		}
		switch rv.Kind() {
		case reflect.Slice, reflect.Map:
			return rv.IsNil()
		}
		return false
	}
	// Every slice-taking function, against both an empty and a nil slice. Reusing
	// immutableCases means a function added there is checked here too.
	t.Run("slice input", func(t *testing.T) {
		for _, tc := range immutableCases {
			for _, in := range []struct {
				kind  string
				slice []any
			}{{"empty", []any{}}, {"nil", nil}} {
				got, err := tc.fn(in.slice)
				if err != nil {
					continue // First and Last legitimately reject empty input
				}
				if isNilResult(got) {
					t.Errorf("%s(%s slice) returned a nil slice, want empty", tc.name, in.kind)
				}
			}
		}
	})

	// Everything else that can produce an empty result: functions that do not
	// take a slice, the map constructors, and the string functions registered
	// straight from the stdlib.
	t.Run("other functions", func(t *testing.T) {
		cases := []struct {
			name string
			fn   func() (any, error)
		}{
			{"List", func() (any, error) { return List(), nil }},
			{"Seq n<1", func() (any, error) { return Seq(0) }},
			{"Seq start>end", func() (any, error) { return Seq(5, 1) }},
			{"Seq empty range with step", func() (any, error) { return Seq(5, 1, 1) }},
			{"Concat no args", func() (any, error) { return Concat() }},
			{"Keys empty map", func() (any, error) { return Keys(map[string]any{}) }},
			{"Values empty map", func() (any, error) { return Values(map[string]any{}) }},
			{"Dict no args", func() (any, error) { return Dict() }},
			{"MergeMaps no args", func() (any, error) { return MergeMaps() }},
			{"MergeMaps empty maps", func() (any, error) { return MergeMaps(map[string]any{}) }},
			// split and fields are registered as strings.Split and strings.Fields.
			// They satisfy the rule today; these pin it so that swapping in a
			// custom implementation cannot quietly break it.
			{"split empty string", func() (any, error) { return strings.Split("", ""), nil }},
			{"split empty on sep", func() (any, error) { return strings.Split("", ","), nil }},
			{"fields empty string", func() (any, error) { return strings.Fields(""), nil }},
			{"fields all whitespace", func() (any, error) { return strings.Fields("   "), nil }},
		}
		for _, c := range cases {
			got, err := c.fn()
			if err != nil {
				t.Errorf("%s: unexpected error: %v", c.name, err)
				continue
			}
			if isNilResult(got) {
				t.Errorf("%s returned nil, want an empty slice or map", c.name)
			}
		}
	})

	// The symptom this invariant exists to prevent.
	t.Run("jsonify emits an empty array", func(t *testing.T) {
		ks, err := Keys(map[string]any{})
		if err != nil {
			t.Fatal(err)
		}
		if j, _ := Jsonify(ks); j != "[]" {
			t.Errorf("jsonify (keys $empty) = %s, want []", j)
		}
		sorted, err := Sort([]any(nil))
		if err != nil {
			t.Fatal(err)
		}
		if j, _ := Jsonify(sorted); j != "[]" {
			t.Errorf("jsonify (sort $nil) = %s, want []", j)
		}
	})
}

// The type a number arrives as is an accident of the decoder — encoding/json
// gives float64, TOML int64, YAML int — and a template literal is int if written
// 1 and float64 if written 1.0. Matching on identical types made where and in
// stricter than text/template's own eq, which already unifies integer widths,
// and the mismatch surfaced as an empty result rather than an error.
func TestValuesEqual_numericAcrossTypes(t *testing.T) {
	type Weight int

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

// The end-to-end case: data decoded by encoding/json, filtered with a literal.
func TestWhereAndIn_jsonDecodedData(t *testing.T) {
	var pages []any
	if err := json.Unmarshal([]byte(`[{"W":1,"T":"a"},{"W":2,"T":"b"}]`), &pages); err != nil {
		t.Fatal(err)
	}

	for _, val := range []any{1, 1.0, int64(1)} {
		got, err := Where(pages, "W", val)
		if err != nil {
			t.Errorf("Where(json, \"W\", %v): %v", val, err)
			continue
		}
		if s := got.([]any); len(s) != 1 {
			t.Errorf("Where(json, \"W\", %v) matched %d, want 1", val, len(s))
		}
	}

	var nums []any
	if err := json.Unmarshal([]byte(`[1,2,3]`), &nums); err != nil {
		t.Fatal(err)
	}
	for _, val := range []any{2, 2.0, int64(2)} {
		ok, err := In(nums, val)
		if err != nil || !ok {
			t.Errorf("In(json nums, %v) = %v, %v; want true", val, ok, err)
		}
	}
	if ok, _ := In(nums, 9); ok {
		t.Error("In(json nums, 9) = true, want false")
	}
}

// A missing field must be an error, not an empty result: a typo in the key would
// otherwise be indistinguishable from data that legitimately did not match.
func TestWhere_missingKeyIsAnError(t *testing.T) {
	pages := []any{
		map[string]any{"Section": "blog"},
		map[string]any{"Section": "docs"},
	}
	if got, err := Where(pages, "Sectoin", "blog"); err == nil {
		t.Errorf("Where with a misspelled key = %v, want an error", got)
	}
	// Present on one element but not another is still an error.
	mixed := []any{
		map[string]any{"Draft": true},
		map[string]any{"Other": 1},
	}
	if got, err := Where(mixed, "Draft", true); err == nil {
		t.Errorf("Where over elements missing the field = %v, want an error", got)
	}
	// The ordinary case still works, including a zero-valued field.
	got, err := Where([]any{
		map[string]any{"Draft": false},
		map[string]any{"Draft": true},
	}, "Draft", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s := got.([]any); len(s) != 1 {
		t.Errorf("Where matched %d, want 1", len(s))
	}
}

func TestDefault(t *testing.T) {
	cases := []struct {
		def  any
		val  any
		want any
	}{
		{"anon", "", "anon"},
		{"anon", "Alice", "Alice"},
		{0, 42, 42},
		{0, 0, 0},
		{"x", nil, "x"},
		{"x", false, "x"},
		{99, []int{}, 99},
	}
	for _, c := range cases {
		got := Default(c.def, c.val)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("default(%v, %v) = %v, want %v", c.def, c.val, got, c.want)
		}
	}
}

// A struct with all-zero fields is a zero value, so default and cond must treat
// it as one. The case that motivates this is an unset date: before, every struct
// fell through to "not zero" and {{ default "Draft" .Date }} never fell back.
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
		{[]int{}, true}, {[]int{1}, false},
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

// default and cond share isZero, so both must see the change.
func TestDefaultAndCond_zeroStruct(t *testing.T) {
	if got := Default("Draft", time.Time{}); got != "Draft" {
		t.Errorf("Default(\"Draft\", time.Time{}) = %v, want Draft", got)
	}
	if got := Cond(time.Time{}, "yes", "no"); got != "no" {
		t.Errorf("Cond(time.Time{}, ...) = %v, want no", got)
	}
	now := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if got := Default("Draft", now); got != now {
		t.Errorf("Default(\"Draft\", %v) = %v, want the date", now, got)
	}
}

func TestCond(t *testing.T) {
	cases := []struct {
		ctrl any
		want any
	}{
		{true, "yes"},
		{false, "no"},
		{"", "no"},
		{"x", "yes"},
		{0, "no"},
		{1, "yes"},
		{nil, "no"},
	}
	for _, c := range cases {
		got := Cond(c.ctrl, "yes", "no")
		if got != c.want {
			t.Errorf("cond(%v) = %v, want %v", c.ctrl, got, c.want)
		}
	}
}
