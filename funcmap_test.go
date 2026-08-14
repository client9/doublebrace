package doublebrace

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"text/template"
)

func TestFuncMap_keys(t *testing.T) {
	fm := FuncMap()
	for _, name := range []string{
		"lower", "upper",
		"trim", "trimPrefix", "trimSuffix", "trimLeft", "trimRight",
		"contains", "hasPrefix", "hasSuffix", "count",
		"replaceAll", "repeat",
		"split", "join", "fields",
		"truncate",
		"add", "sub", "mul", "div", "mod",
		"abs", "ceil", "floor", "round",
	} {
		if _, ok := fm[name]; !ok {
			t.Errorf("FuncMap missing %q", name)
		}
	}
}

// documentedFuncs returns the function names listed in doc.go, which documents
// the template-facing API as bullets of the form:
//
//	//   - name(args) result — description
//
// Reading the source is deliberate. doc.go is the reference a user reads before
// writing a template, so it is the list that has to be true; deriving it from
// the code instead would make the test agree with itself.
func documentedFuncs(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("reading doc.go: %v", err)
	}
	re := regexp.MustCompile(`(?m)^//\s+- ([a-zA-Z][a-zA-Z0-9]*)\(`)
	var names []string
	for _, m := range re.FindAllSubmatch(src, -1) {
		name := string(m[1])
		if !slices.Contains(names, name) {
			names = append(names, name) // seq is listed once per arity
		}
	}
	if len(names) < 40 {
		t.Fatalf("parsed only %d names from doc.go; the bullet format likely changed", len(names))
	}
	return names
}

// doc.go and FuncMap must agree in both directions. A function registered but
// undocumented is invisible to users; one documented but unregistered is a
// promise the package does not keep, and fails at parse time with
// "function not defined" rather than anywhere useful.
func TestFuncMap_matchesDocumentation(t *testing.T) {
	fm := FuncMap()
	documented := documentedFuncs(t)

	for _, name := range documented {
		if _, ok := fm[name]; !ok {
			t.Errorf("doc.go documents %q but FuncMap does not register it", name)
		}
	}

	for name := range fm {
		if !slices.Contains(documented, name) {
			t.Errorf("FuncMap registers %q but doc.go does not document it", name)
		}
	}
}

// Every registered function must be callable through a template. This catches a
// signature templates cannot invoke — a second return value that is not error,
// say — which a direct Go call would not reveal.
func TestFuncMap_allNamesParse(t *testing.T) {
	for name := range FuncMap() {
		if _, err := template.New(name).Funcs(FuncMap()).Parse("{{ " + name + " }}"); err != nil {
			t.Errorf("%q: template will not parse: %v", name, err)
		}
	}
}

func TestMerge(t *testing.T) {
	a := template.FuncMap{"foo": func() string { return "a" }}
	b := template.FuncMap{"bar": func() string { return "b" }}
	c := template.FuncMap{"foo": func() string { return "c" }} // overrides a

	merged := Merge(a, b, c)
	if _, ok := merged["foo"]; !ok {
		t.Error("merged missing foo")
	}
	if _, ok := merged["bar"]; !ok {
		t.Error("merged missing bar")
	}
	if got := merged["foo"].(func() string)(); got != "c" {
		t.Errorf("Merge: expected later map to win, got %q", got)
	}
}

func TestMerge_empty(t *testing.T) {
	if got := Merge(); got == nil {
		t.Error("Merge() with no args returned nil")
	}
}

// Every count argument — the n of take, drop, truncate, repeat, and replace,
// and the bounds of seq — is typed any rather than int, and this is the test
// that says why. text/template does not convert an argument, it requires an
// assignable type, so with an int parameter every line below failed at
// execution with "wrong type for value; expected int; got float64". The math
// functions all return float64, an int64 field is an int64, and JSON decodes a
// number to float64 — which left a count, the most likely thing in a template
// to be computed rather than written down, as the one place arithmetic could
// not reach.
func TestCountArgs_acceptComputedValues(t *testing.T) {
	data := map[string]any{
		"L": []any{1, 2, 3, 4},
		"N": int64(2), // a field from Go data
		"F": 2.0,      // a number decoded from JSON
		"S": "2",      // a count from frontmatter
	}

	cases := []struct {
		src  string
		want string
	}{
		{`{{ take .L (add 1 1) }}`, "[1 2]"},
		{`{{ drop .L (sub 4 1) }}`, "[4]"},
		{`{{ take .L (div 4 2) }}`, "[1 2]"},
		{`{{ take .L .N }}`, "[1 2]"},
		{`{{ take .L .F }}`, "[1 2]"},
		{`{{ take .L .S }}`, "[1 2]"},
		{`{{ seq (add 1 2) }}`, "[1 2 3]"},
		{`{{ seq 1 (mul 2 2) 2 }}`, "[1 3]"},
		{`{{ truncate "hello world" (add 4 4) }}`, "hello w…"},
		{`{{ repeat "-" (mul 2 2) }}`, "----"},
		{`{{ replace "aabbaa" "a" "x" (add 1 2) }}`, "xxbbxa"},
		// Truncation toward zero comes with toInt, so a division that does not
		// come out even still counts.
		{`{{ take .L (div 7 2) }}`, "[1 2 3]"},
	}
	for _, c := range cases {
		tmpl, err := template.New("t").Funcs(FuncMap()).Parse(c.src)
		if err != nil {
			t.Errorf("%s: parse: %v", c.src, err)
			continue
		}
		var buf strings.Builder
		if err := tmpl.Execute(&buf, data); err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if got := buf.String(); got != c.want {
			t.Errorf("%s = %q, want %q", c.src, got, c.want)
		}
	}
}

// A count that is not a number at all is still an error, and it names the
// function rather than surfacing as a bare conversion failure.
func TestCountArgs_rejectNonNumeric(t *testing.T) {
	for _, src := range []string{
		`{{ take .L "two" }}`,
		`{{ drop .L true }}`,
		`{{ seq "many" }}`,
		`{{ truncate "hello" "eight" }}`,
		`{{ repeat "-" "four" }}`,
		`{{ replace "aa" "a" "x" "three" }}`,
	} {
		tmpl := template.Must(template.New("t").Funcs(FuncMap()).Parse(src))
		var buf strings.Builder
		if err := tmpl.Execute(&buf, map[string]any{"L": []any{1, 2}}); err == nil {
			t.Errorf("%s: expected an error, wrote %q", src, buf.String())
		}
	}
}

// Every string argument accepts any value of string kind, so a named type
// works with all of them or none. It used to be half: take, drop, first, last,
// and in went through asString, while the functions registered straight from
// the stdlib and this package's own string functions took a plain string and
// failed with "wrong type for value; expected string; got Slug" — the same
// value accepted by one function and rejected by the next, with nothing in the
// name to say which.
func TestStringArgs_acceptStringKinds(t *testing.T) {
	type slug string

	cases := []struct {
		src  string
		want string
	}{
		{`{{ lower .S }}`, "hello world"},
		{`{{ upper .S }}`, "HELLO WORLD"},
		{`{{ trim .Pad }}`, "hi"},
		{`{{ trimPrefix .S .Pre }}`, " world"},
		{`{{ trimSuffix .S .Suf }}`, "hello "},
		{`{{ trimLeft .S .Pre }}`, " world"},
		{`{{ trimRight .S .Suf }}`, "hello "},
		{`{{ contains .S .Pre }}`, "true"},
		{`{{ hasPrefix .S .Pre }}`, "true"},
		{`{{ hasSuffix .S .Suf }}`, "true"},
		{`{{ count .S .Pre }}`, "1"},
		{`{{ replace .S .Pre "howdy" }}`, "howdy world"},
		{`{{ replaceAll .S .Pre "howdy" }}`, "howdy world"},
		{`{{ repeat .Pre 2 }}`, "hellohello"},
		{`{{ split .S .Sp }}`, "[hello world]"},
		{`{{ join (split .S .Sp) .Sp }}`, "hello world"},
		{`{{ fields .S }}`, "[hello world]"},
		{`{{ lenRunes .S }}`, "11"},
		{`{{ truncate .S 8 }}`, "hello w…"},
		{`{{ firstUpper .S }}`, "Hello world"},
		{`{{ urlEncode .S }}`, "hello+world"},
		{`{{ urlPathEscape .S }}`, "hello%20world"},
		// The sequence-access functions already accepted these; they are here so
		// the two halves are checked as one set.
		{`{{ take .S 5 }}`, "hello"},
		{`{{ drop .S 6 }}`, "world"},
		{`{{ in .S .Pre }}`, "true"},
		// Paths and layouts are text too. These registered their plain string
		// parameters directly until the strFn adapters were extended to them,
		// which made pathBase reject the value lower had just accepted.
		{`{{ pathBase .Path }}`, "bar.html"},
		{`{{ pathDir .Path }}`, "foo"},
		{`{{ pathExt .Path }}`, ".html"},
		{`{{ pathClean .Dirty }}`, "foo/bar.html"},
		{`{{ pathJoin .Pre .Suf }}`, "hello/world"},
		{`{{ (parseTime .Layout .Date).Year }}`, "2024"},
		// A key is text as well, on either side of the lookup: the named type
		// names an ordinary string key, which index then finds with a plain one.
		{`{{ index (dict .Pre 1) "hello" }}`, "1"},
		// Equality reads a string the same way, so a named type matches the
		// literal it was written from rather than silently missing it.
		{`{{ in (list .Pre) "hello" }}`, "true"},
	}

	data := map[string]any{
		"S":      slug("hello world"),
		"Pre":    slug("hello"),
		"Suf":    slug("world"),
		"Sp":     slug(" "),
		"Pad":    slug("  hi  "),
		"Path":   slug("foo/bar.html"),
		"Dirty":  slug("foo/./bar.html"),
		"Layout": slug("2006-01-02"),
		"Date":   slug("2024-03-15"),
	}
	for _, c := range cases {
		tmpl, err := template.New("t").Funcs(FuncMap()).Parse(c.src)
		if err != nil {
			t.Errorf("%s: parse: %v", c.src, err)
			continue
		}
		var buf strings.Builder
		if err := tmpl.Execute(&buf, data); err != nil {
			t.Errorf("%s: %v", c.src, err)
			continue
		}
		if got := buf.String(); got != c.want {
			t.Errorf("%s = %q, want %q", c.src, got, c.want)
		}
	}
}

// A value that is not a string of any kind is still rejected, and the error
// names the function rather than reporting a bare argument-type mismatch.
func TestStringArgs_rejectNonStrings(t *testing.T) {
	for _, src := range []string{
		`{{ lower 42 }}`,
		`{{ trimPrefix "abc" 42 }}`,
		`{{ contains 42 "a" }}`,
		// Every argument position is checked, not just the first.
		`{{ replaceAll 42 "a" "b" }}`,
		`{{ replaceAll "abc" 42 "b" }}`,
		`{{ replaceAll "abc" "a" 42 }}`,
		`{{ replace 42 "a" "b" }}`,
		`{{ replace "abc" 42 "b" }}`,
		`{{ replace "abc" "a" 42 }}`,
		`{{ join (list "a") 42 }}`,
		`{{ truncate 42 5 }}`,
		`{{ repeat 42 2 }}`,
		`{{ urlEncode 42 }}`,
		`{{ lower .Nil }}`,
		`{{ pathBase 42 }}`,
		`{{ pathClean .Nil }}`,
		// Every position of the variadic is checked, not just the first.
		`{{ pathJoin "a" 42 }}`,
		`{{ parseTime 42 "2024-03-15" }}`,
		`{{ parseTime "2006-01-02" 42 }}`,
		`{{ dict 42 "v" }}`,
	} {
		tmpl := template.Must(template.New("t").Funcs(FuncMap()).Parse(src))
		var buf strings.Builder
		if err := tmpl.Execute(&buf, map[string]any{"Nil": nil}); err == nil {
			t.Errorf("%s: expected an error, wrote %q", src, buf.String())
		}
	}
}

func TestFuncMap_inTemplate(t *testing.T) {
	tmpl := template.Must(template.New("t").Funcs(FuncMap()).Parse(
		`{{ upper .S }} {{ lower .S }} {{ truncate .S 5 }}`,
	))
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{"S": "hello world"}); err != nil {
		t.Fatalf("template execute: %v", err)
	}
	if got := buf.String(); got != "HELLO WORLD hello world hell…" {
		t.Errorf("unexpected output: %q", got)
	}
}
