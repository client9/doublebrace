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
