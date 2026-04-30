package doublebrace

import (
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
