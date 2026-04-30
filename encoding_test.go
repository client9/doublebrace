package doublebrace

import (
	"testing"
)

func TestJsonify(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{map[string]any{"a": 1, "b": 2}, `{"a":1,"b":2}`},
		{[]string{"x", "y"}, `["x","y"]`},
		{"hello", `"hello"`},
		{42, `42`},
		{true, `true`},
		{nil, `null`},
	}
	for _, c := range cases {
		got, err := Jsonify(c.in)
		if err != nil {
			t.Errorf("Jsonify(%v): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Jsonify(%v) = %q, want %q", c.in, got, c.want)
		}
	}

	// unmarshalable value
	if _, err := Jsonify(make(chan int)); err == nil {
		t.Error("expected error for unmarshalable value")
	}
}
