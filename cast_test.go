package doublebrace

import (
	"testing"
)

func TestToInt(t *testing.T) {
	cases := []struct {
		in   any
		want int
	}{
		{42, 42},
		{int8(10), 10},
		{int64(-5), -5},
		{uint(3), 3},
		{float32(3.9), 3},
		{float64(2.1), 2},
		{"17", 17},
		{"-3", -3},
	}
	for _, c := range cases {
		got, err := ToInt(c.in)
		if err != nil {
			t.Errorf("ToInt(%v): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ToInt(%v) = %d, want %d", c.in, got, c.want)
		}
	}

	if _, err := ToInt("abc"); err == nil {
		t.Error("expected error for non-numeric string")
	}
	if _, err := ToInt(true); err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestToFloat(t *testing.T) {
	cases := []struct {
		in   any
		want float64
	}{
		{42, 42},
		{float32(1.5), float64(float32(1.5))},
		{float64(3.14), 3.14},
		{"3.14", 3.14},
		{"-1", -1},
	}
	for _, c := range cases {
		got, err := ToFloat(c.in)
		if err != nil {
			t.Errorf("ToFloat(%v): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ToFloat(%v) = %f, want %f", c.in, got, c.want)
		}
	}

	if _, err := ToFloat("abc"); err == nil {
		t.Error("expected error for non-numeric string")
	}
}
