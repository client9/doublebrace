package doublebrace

import (
	"strings"
	"testing"
)

// Named types standing in for the shapes template data actually carries: a
// numeric field declared as its own type, and a string one. They exist to reach
// toFloat64's reflect fallback, which the concrete type switch above it cannot.
type (
	Weight int
	Count  uint
	Ratio  float64
	Slug   string
	Flag   bool // a named type of a kind that is deliberately not numeric
)

func TestToFloat64(t *testing.T) {
	cases := []struct {
		in   any
		want float64
		ok   bool
	}{
		// One case per arm of the type switch. An untested arm is a conversion
		// that only runs once a caller's decoder happens to produce that type.
		{int(3), 3.0, true},
		{int8(-8), -8.0, true},
		{int16(-300), -300.0, true},
		{int32(70000), 70000.0, true},
		{int64(-7), -7.0, true},
		{uint(4), 4.0, true},
		{uint8(200), 200.0, true},
		{uint16(50000), 50000.0, true},
		{uint32(3000000000), 3000000000.0, true},
		{uint64(9), 9.0, true},
		{float32(1.5), 1.5, true},
		{float64(2.5), 2.5, true},
		{"3.14", 3.14, true},
		{"bad", 0, false},
		{true, 0, false},
		// Named types reach the reflect fallback, not the type switch. Template
		// data routinely carries these (type Weight int in a page struct), and
		// without the fallback in accepted them while add and sort did not.
		{Weight(3), 3.0, true},
		{Ratio(1.5), 1.5, true},
		{Count(7), 7.0, true},
		{Slug("2.5"), 2.5, true},
		{Slug("bad"), 0, false},
		{[]byte("3"), 0, false},
	}
	for _, c := range cases {
		got, err := toFloat64(c.in)
		if c.ok && err != nil {
			t.Errorf("toFloat64(%v): unexpected error: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("toFloat64(%v): expected error, got %v", c.in, got)
		}
		if c.ok && got != c.want {
			t.Errorf("toFloat64(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestMathOps(t *testing.T) {
	fm := FuncMap()

	call2 := func(name string, a, b any) float64 {
		t.Helper()
		fn := fm[name].(func(any, any) (float64, error))
		v, err := fn(a, b)
		if err != nil {
			t.Fatalf("%s(%v, %v): %v", name, a, b, err)
		}
		return v
	}
	call1 := func(name string, a any) float64 {
		t.Helper()
		fn := fm[name].(func(any) (float64, error))
		v, err := fn(a)
		if err != nil {
			t.Fatalf("%s(%v): %v", name, a, err)
		}
		return v
	}

	if got := call2("add", 3, 4); got != 7 {
		t.Errorf("add: got %v", got)
	}
	if got := call2("sub", 10, 3); got != 7 {
		t.Errorf("sub: got %v", got)
	}
	if got := call2("mul", 3, 4); got != 12 {
		t.Errorf("mul: got %v", got)
	}
	if got := call2("div", 7, 2); got != 3.5 {
		t.Errorf("div: got %v", got)
	}
	if got := call2("mod", 7, 3); got != 1 {
		t.Errorf("mod: got %v", got)
	}
	if got := call1("abs", -5); got != 5 {
		t.Errorf("abs: got %v", got)
	}
	if got := call1("ceil", 1.2); got != 2 {
		t.Errorf("ceil: got %v", got)
	}
	if got := call1("floor", 1.9); got != 1 {
		t.Errorf("floor: got %v", got)
	}
	if got := call1("round", 1.5); got != 2 {
		t.Errorf("round: got %v", got)
	}
}

func TestMathDiv_byZero(t *testing.T) {
	if _, err := Div(1, 0); err == nil {
		t.Error("expected error for div by zero")
	}
}

func TestMathMod_byZero(t *testing.T) {
	if _, err := Mod(5, 0); err == nil {
		t.Error("expected error for mod by zero")
	}
}

func TestMathOps_mixedTypes(t *testing.T) {
	got, err := Div(float64(7), int(2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 3.5 {
		t.Errorf("div(7.0, 2) = %v, want 3.5", got)
	}
}

func TestMinMax(t *testing.T) {
	cases := []struct {
		name string
		fn   func(...any) (float64, error)
		args []any
		want float64
	}{
		// Min: scalars
		{"min/one", Min, []any{5}, 5},
		{"min/two", Min, []any{3, 7}, 3},
		{"min/many", Min, []any{4, 1, 9, 2}, 1},
		{"min/negative", Min, []any{0, -3, 5}, -3},
		// Min: slice input
		{"min/slice", Min, []any{[]int{5, 2, 8}}, 2},
		// Min: mixed scalars and slice
		{"min/mixed", Min, []any{[]int{5, 2}, 1, 9}, 1},
		// Min: nested slices
		{"min/nested", Min, []any{[]any{[]int{10, 3}, 7}}, 3},
		// Min: mixed types
		{"min/types", Min, []any{float64(1.5), int(3), "2.0"}, 1.5},

		// Max: scalars
		{"max/one", Max, []any{5}, 5},
		{"max/two", Max, []any{3, 7}, 7},
		{"max/many", Max, []any{4, 1, 9, 2}, 9},
		{"max/negative", Max, []any{0, -3, 5}, 5},
		// Max: slice input
		{"max/slice", Max, []any{[]int{5, 2, 8}}, 8},
		// Max: mixed scalars and slice
		{"max/mixed", Max, []any{[]int{5, 2}, 9, 1}, 9},
		// Max: nested slices
		{"max/nested", Max, []any{[]any{[]int{10, 3}, 7}}, 10},
		// Max: mixed types
		{"max/types", Max, []any{float64(1.5), int(3), "2.0"}, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.fn(c.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestPow(t *testing.T) {
	cases := []struct {
		base, exp any
		want      float64
	}{
		{2, 10, 1024},
		{9, 0.5, 3}, // square root
		{5, 0, 1},   // anything^0 == 1
		{0, 0, 1},   // math.Pow(0,0) == 1
		{"2", "8", 256},
	}
	for _, c := range cases {
		got, err := Pow(c.base, c.exp)
		if err != nil {
			t.Errorf("Pow(%v, %v): unexpected error: %v", c.base, c.exp, err)
			continue
		}
		if got != c.want {
			t.Errorf("Pow(%v, %v) = %v, want %v", c.base, c.exp, got, c.want)
		}
	}
}

func TestModBool(t *testing.T) {
	cases := []struct {
		a, b any
		want bool
	}{
		{4, 2, true},
		{5, 2, false},
		{9, 3, true},
		{10, 3, false},
		{0, 5, true},
	}
	for _, c := range cases {
		got, err := ModBool(c.a, c.b)
		if err != nil {
			t.Errorf("ModBool(%v, %v): unexpected error: %v", c.a, c.b, err)
			continue
		}
		if got != c.want {
			t.Errorf("ModBool(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}

	if _, err := ModBool(5, 0); err == nil {
		t.Error("ModBool(5, 0): expected error for division by zero")
	}
}

func TestClamp(t *testing.T) {
	cases := []struct {
		val, min, max any
		want          float64
	}{
		{5, 1, 10, 5},       // within range
		{0, 1, 10, 1},       // below min
		{15, 1, 10, 10},     // above max
		{1, 1, 10, 1},       // at min
		{10, 1, 10, 10},     // at max
		{"5", "1", "10", 5}, // numeric strings
	}
	for _, c := range cases {
		got, err := Clamp(c.val, c.min, c.max)
		if err != nil {
			t.Errorf("Clamp(%v, %v, %v): %v", c.val, c.min, c.max, err)
			continue
		}
		if got != c.want {
			t.Errorf("Clamp(%v, %v, %v) = %v, want %v", c.val, c.min, c.max, got, c.want)
		}
	}
	if _, err := Clamp(5, 10, 1); err == nil {
		t.Error("expected error for min > max")
	}
}

// min and max accept the same sequence kinds the collection functions do, so an
// array reaches them like a slice does. flattenNumbers shares isSequence with
// toSlice to keep the two surfaces from drifting apart.
func TestMinMax_arrays(t *testing.T) {
	cases := []struct {
		name string
		fn   func(...any) (float64, error)
		args []any
		want float64
	}{
		{"min/array", Min, []any{[3]int{5, 2, 8}}, 2},
		{"max/array", Max, []any{[3]int{5, 2, 8}}, 8},
		{"min/array of floats", Min, []any{[2]float64{1.5, 0.5}}, 0.5},
		{"min/array with scalars", Min, []any{[2]int{5, 2}, 1, 9}, 1},
		{"max/array and slice", Max, []any{[2]int{5, 2}, []int{9, 1}}, 9},
		{"min/array nested in slice", Min, []any{[]any{[2]int{10, 3}, 7}}, 3},
		{"max/array nested in array", Max, []any{[2]any{[2]int{10, 3}, 7}}, 10},
		{"min/array of numeric strings", Min, []any{[2]string{"10", "2"}}, 2},
		{"min/single-element array", Min, []any{[1]int{42}}, 42},
	}
	for _, c := range cases {
		got, err := c.fn(c.args...)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s(%v) = %v, want %v", c.name, c.args, got, c.want)
		}
	}

	// An empty array contributes no values, so it is the no-arguments case.
	if _, err := Min([0]int{}); err == nil {
		t.Error("Min([0]int{}): expected error for an empty sequence")
	}
}

func TestMinMax_errors(t *testing.T) {
	if _, err := Min(); err == nil {
		t.Error("Min(): expected error for no args")
	}
	if _, err := Max(); err == nil {
		t.Error("Max(): expected error for no args")
	}
	if _, err := Min("bad"); err == nil {
		t.Error("Min(bad): expected error for non-numeric string")
	}
	if _, err := Max(true); err == nil {
		t.Error("Max(true): expected error for unsupported type")
	}
}

// A value that cannot be converted must surface as an error from every argument
// position of every function, not merely the first. Each toFloat64 call site is
// its own branch: Div checks its divisor after its dividend, Clamp checks three
// arguments in turn, and min/max convert inside a recursion. A site that
// silently substituted zero would produce a plausible wrong number instead of a
// failure, which is the worst outcome available to a template.
func TestMath_conversionErrorsPropagate(t *testing.T) {
	const bad = "not a number"

	binary := []struct {
		name string
		fn   func(a, b any) (float64, error)
	}{
		{"Add", Add}, {"Sub", Sub}, {"Mul", Mul},
		{"Div", Div}, {"Mod", Mod}, {"Pow", Pow},
	}
	for _, c := range binary {
		if got, err := c.fn(bad, 1); err == nil {
			t.Errorf("%s(%q, 1) = %v, want an error", c.name, bad, got)
		}
		if got, err := c.fn(1, bad); err == nil {
			t.Errorf("%s(1, %q) = %v, want an error", c.name, bad, got)
		}
	}

	unary := []struct {
		name string
		fn   func(a any) (float64, error)
	}{
		{"Abs", Abs}, {"Ceil", Ceil}, {"Floor", Floor}, {"Round", Round},
	}
	for _, c := range unary {
		if got, err := c.fn(bad); err == nil {
			t.Errorf("%s(%q) = %v, want an error", c.name, bad, got)
		}
	}

	if got, err := ModBool(bad, 2); err == nil {
		t.Errorf("ModBool(%q, 2) = %v, want an error", bad, got)
	}
	if got, err := ModBool(2, bad); err == nil {
		t.Errorf("ModBool(2, %q) = %v, want an error", bad, got)
	}

	// Clamp names the argument that failed. The label is worth pinning: with
	// three numeric arguments, "cannot convert string to float64" alone does not
	// say which one the template got wrong.
	for _, c := range []struct {
		args [3]any
		want string
	}{
		{[3]any{bad, 1, 10}, "clamp: val:"},
		{[3]any{1, bad, 10}, "clamp: min:"},
		{[3]any{1, 10, bad}, "clamp: max:"},
	} {
		_, err := Clamp(c.args[0], c.args[1], c.args[2])
		if err == nil {
			t.Errorf("Clamp%v: expected an error", c.args)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("Clamp%v: error %q, want it to contain %q", c.args, err, c.want)
		}
	}

	// min and max flatten nested sequences recursively; an element that fails to
	// convert must abort the whole call rather than being skipped.
	if got, err := Min([]any{[]any{bad}}); err == nil {
		t.Errorf("Min(nested %q) = %v, want an error", bad, got)
	}
	if got, err := Max([]any{[]any{bad}}); err == nil {
		t.Errorf("Max(nested %q) = %v, want an error", bad, got)
	}
}
