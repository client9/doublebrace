package doublebrace

import "testing"

func TestReplace(t *testing.T) {
	cases := []struct {
		s, old, new string
		n           []int
		want        string
	}{
		{"aabbaa", "a", "x", nil, "xabbaa"},       // default: first only
		{"aabbaa", "a", "x", []int{1}, "xabbaa"},  // explicit 1
		{"aabbaa", "a", "x", []int{3}, "xxbbxa"},  // limit 3
		{"aabbaa", "a", "x", []int{-1}, "xxbbxx"}, // all
		{"hello", "z", "x", nil, "hello"},         // no match
	}
	for _, c := range cases {
		got := Replace(c.s, c.old, c.new, c.n...)
		if got != c.want {
			t.Errorf("Replace(%q, %q, %q, %v) = %q, want %q", c.s, c.old, c.new, c.n, got, c.want)
		}
	}
}

func TestFirstUpper(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"go", "Go"},
		{"hello world", "Hello world"},
		{"élan", "Élan"},
		{"", ""},
		{"A", "A"},
		{"already Upper", "Already Upper"},
	}
	for _, c := range cases {
		got := FirstUpper(c.in)
		if got != c.want {
			t.Errorf("FirstUpper(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLenRunes(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"hello", 5},
		{"café", 4}, // é is 2 bytes, 1 rune
		{"日本語", 3},
		{"", 0},
	}
	for _, c := range cases {
		got := LenRunes(c.in)
		if got != c.want {
			t.Errorf("LenRunes(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"hello", 10, "hello"},         // shorter than limit
		{"hello", 5, "hello"},          // exact length
		{"hello world", 8, "hello w…"}, // cut with ellipsis
		{"hello", 1, "…"},              // n=1 is just ellipsis
		{"hello", 0, ""},               // n=0 is empty
		{"héllo", 4, "hél…"},           // rune-aware
	}
	for _, tt := range tests {
		got := Truncate(tt.in, tt.n)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
}
