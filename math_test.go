package doublebrace

import "testing"

func TestToFloat64(t *testing.T) {
	cases := []struct {
		in   any
		want float64
		ok   bool
	}{
		{int(3), 3.0, true},
		{int64(-7), -7.0, true},
		{float32(1.5), 1.5, true},
		{float64(2.5), 2.5, true},
		{uint(4), 4.0, true},
		{"3.14", 3.14, true},
		{"bad", 0, false},
		{true, 0, false},
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
