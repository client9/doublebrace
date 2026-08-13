package doublebrace

import (
	"math"
	"strings"
	"testing"
	"text/template"
)

func TestRepeat(t *testing.T) {
	ok := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"basic", "ab", 3, "ababab"},
		{"zero", "-", 0, ""},
		{"one", "-", 1, "-"},
		{"empty string", "", 5, ""},
		// The limit counts runes, not bytes, so a multi-byte string gets the
		// same count as an ASCII one.
		{"at the limit", "x", MaxRepeatLen, strings.Repeat("x", MaxRepeatLen)},
		{"multibyte at the limit", "日", MaxRepeatLen, strings.Repeat("日", MaxRepeatLen)},
	}
	for _, c := range ok {
		got, err := Repeat(c.s, c.n)
		if err != nil {
			t.Errorf("%s: Repeat(%q, %d): %v", c.name, c.s, c.n, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: Repeat(%q, %d) = %q, want %q", c.name, c.s, c.n, got, c.want)
		}
	}

	bad := []struct {
		name string
		s    string
		n    int
	}{
		// strings.Repeat panics on these; a template author reaching them from
		// data must get an error instead.
		{"negative", "-", -1},
		{"negative with empty string", "", -1},
		{"one past the limit", "x", MaxRepeatLen + 1},
		{"multibyte one past the limit", "日", MaxRepeatLen + 1},
		{"multi-rune string past the limit", "ab", MaxRepeatLen/2 + 1},
		// The product overflows int and wraps negative, which would compare as
		// comfortably under the limit if the check multiplied instead of divided.
		{"count overflows int", "ab", math.MaxInt},
		{"count is MaxInt on a single rune", "x", math.MaxInt},
	}
	for _, c := range bad {
		if got, err := Repeat(c.s, c.n); err == nil {
			t.Errorf("%s: Repeat(%q, %d) = %q, want an error", c.name, c.s, c.n, got)
		}
	}
}

// The guardrail has to hold through the template, which is the only way a count
// reaches repeat from data in the first place.
func TestRepeat_throughTemplate(t *testing.T) {
	tmpl := template.Must(template.New("t").Funcs(FuncMap()).Parse(`{{ repeat "-" . }}`))

	var ok strings.Builder
	if err := tmpl.Execute(&ok, 3); err != nil {
		t.Fatalf("repeat 3: %v", err)
	}
	if ok.String() != "---" {
		t.Errorf("repeat 3 = %q, want %q", ok.String(), "---")
	}

	for _, n := range []int{-1, MaxRepeatLen + 1, math.MaxInt} {
		var sb strings.Builder
		if err := tmpl.Execute(&sb, n); err == nil {
			t.Errorf("repeat %d: expected an error, wrote %d bytes", n, sb.Len())
		}
	}
}

func TestJoin(t *testing.T) {
	cases := []struct {
		name string
		in   any
		sep  string
		want string
	}{
		{"[]string", []string{"a", "b", "c"}, ", ", "a, b, c"},
		{"[]any of strings", []any{"a", "b", "c"}, ", ", "a, b, c"},
		{"[]any mixed", []any{1, "b", 2.5}, "-", "1-b-2.5"},
		{"[]int", []int{1, 2, 3}, "+", "1+2+3"},
		{"named slice type", Titles{"a", "b"}, ",", "a,b"},
		{"empty separator", []any{"a", "b"}, "", "ab"},
		{"single element", []any{"a"}, ", ", "a"},
		{"empty slice", []any{}, ", ", ""},
		{"nil element renders as <nil>", []any{"a", nil}, ",", "a,<nil>"},
	}
	for _, c := range cases {
		got, err := Join(c.in, c.sep)
		if err != nil {
			t.Errorf("%s: Join(%v, %q) unexpected error: %v", c.name, c.in, c.sep, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: Join(%v, %q) = %q, want %q", c.name, c.in, c.sep, got, c.want)
		}
	}
}

// Titles exercises Join against a named slice type, which reaches the
// reflection path in toSlice rather than the []string fast path.
type Titles []string

func TestJoin_notASlice(t *testing.T) {
	for _, in := range []any{nil, 42, "already a string", map[string]any{"a": 1}} {
		if _, err := Join(in, ","); err == nil {
			t.Errorf("Join(%#v, \",\") expected an error, got nil", in)
		}
	}
}

// join must compose with the collection functions, all of which return []any.
// Before Join existed, join was bound directly to strings.Join and every one of
// these pipelines failed at execution with "expected []string; got []interface{}".
func TestJoin_composesWithCollections(t *testing.T) {
	list := []any{"c", "a", "b"}

	sorted, err := Sort(list)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := Join(sorted, ", "); got != "a, b, c" {
		t.Errorf("join (sort $list) = %q, want %q", got, "a, b, c")
	}

	dropped, err := Drop(list, -1)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := Join(dropped, ", "); got != "c, a" {
		t.Errorf("join (drop $list -1) = %q, want %q", got, "c, a")
	}

	if got, _ := Join(List("a", "b"), "-"); got != "a-b" {
		t.Errorf("join (list \"a\" \"b\") = %q, want %q", got, "a-b")
	}
}

func TestReplace(t *testing.T) {
	cases := []struct {
		s, old, new string
		n           []any
		want        string
	}{
		{"aabbaa", "a", "x", nil, "xabbaa"},        // default: first only
		{"aabbaa", "a", "x", []any{1}, "xabbaa"},   // explicit 1
		{"aabbaa", "a", "x", []any{3}, "xxbbxa"},   // limit 3
		{"aabbaa", "a", "x", []any{-1}, "xxbbxx"},  // all
		{"hello", "z", "x", nil, "hello"},          // no match
		{"aabbaa", "a", "x", []any{3.0}, "xxbbxa"}, // a float count, as math returns
		{"aabbaa", "a", "x", []any{"3"}, "xxbbxa"}, // a numeric string
		{"aabbaa", "a", "x", []any{int64(3)}, "xxbbxa"},
	}
	for _, c := range cases {
		got, err := Replace(c.s, c.old, c.new, c.n...)
		if err != nil {
			t.Errorf("Replace(%q, %q, %q, %v): %v", c.s, c.old, c.new, c.n, err)
			continue
		}
		if got != c.want {
			t.Errorf("Replace(%q, %q, %q, %v) = %q, want %q", c.s, c.old, c.new, c.n, got, c.want)
		}
	}

	if _, err := Replace("a", "a", "x", "many"); err == nil {
		t.Error("Replace with a non-numeric count: expected an error")
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

// The Latin digraphs are the only characters where Unicode title case differs
// from upper case, and they are the reason FirstUpper uses unicode.ToTitle: when
// capitalizing a word the title-case form is correct, while the upper-case form
// belongs to a word written entirely in capitals.
func TestFirstUpper_titleCaseDigraphs(t *testing.T) {
	cases := []struct {
		in, want, wrong string
	}{
		{"ǳagreb", "ǲagreb", "Ǳagreb"},
		{"ǆungla", "ǅungla", "Ǆungla"},
		{"ǉubav", "ǈubav", "Ǉubav"},
		{"ǌegov", "ǋegov", "Ǌegov"},
	}
	for _, c := range cases {
		got := FirstUpper(c.in)
		if got != c.want {
			t.Errorf("FirstUpper(%q) = %q, want %q (upper case %q is the wrong form here)",
				c.in, got, c.want, c.wrong)
		}
	}
}

// capitalize is deliberately not a function: it composes from the two that
// exist. This pins the composition the recipes document recommends.
func TestFirstUpper_composesToCapitalize(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"hELLO", "Hello"},
		{"hello world", "Hello world"},
		{"HELLO WORLD", "Hello world"},
		{"ÉLAN", "Élan"},
		{"", ""},
	}
	for _, c := range cases {
		if got := FirstUpper(strings.ToLower(c.in)); got != c.want {
			t.Errorf("firstUpper (lower %q) = %q, want %q", c.in, got, c.want)
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
		n    any
		want string
	}{
		{"hello", 10, "hello"},           // shorter than limit
		{"hello", 5, "hello"},            // exact length
		{"hello world", 8, "hello w…"},   // cut with ellipsis
		{"hello", 1, "…"},                // n=1 is just ellipsis
		{"hello", 0, ""},                 // n=0 is empty
		{"héllo", 4, "hél…"},             // rune-aware
		{"hello world", 8.0, "hello w…"}, // a float count, as math returns
		{"hello world", "8", "hello w…"}, // a numeric string
		{"hello world", int64(8), "hello w…"},
		{"hello world", 8.9, "hello w…"}, // truncated toward zero, as toInt does
	}
	for _, tt := range tests {
		got, err := Truncate(tt.in, tt.n)
		if err != nil {
			t.Errorf("truncate(%q, %v): %v", tt.in, tt.n, err)
			continue
		}
		if got != tt.want {
			t.Errorf("truncate(%q, %v) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}

	if _, err := Truncate("hello", "eight"); err == nil {
		t.Error("Truncate with a non-numeric length: expected an error")
	}
}
