package doublebrace

import (
	"reflect"
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
	// zero args → empty
	empty, err := Concat()
	if err != nil || empty != nil {
		t.Errorf("Concat() should return nil, nil; got %v, %v", empty, err)
	}
	// error on non-slice
	if _, err := Concat("not a slice"); err == nil {
		t.Error("expected error for non-slice argument")
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
