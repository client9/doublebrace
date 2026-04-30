package doublebrace

import (
	"html/template"
	"testing"
)

func TestSafeFuncs(t *testing.T) {
	const input = `<b class="x">hello</b>`

	t.Run("SafeCSS", func(t *testing.T) {
		got, err := SafeCSS(input)
		if err != nil {
			t.Fatal(err)
		}
		if got != template.CSS(input) {
			t.Errorf("got %q, want %q", got, input)
		}
	})

	t.Run("SafeHTML", func(t *testing.T) {
		got, err := SafeHTML(input)
		if err != nil {
			t.Fatal(err)
		}
		if got != template.HTML(input) {
			t.Errorf("got %q, want %q", got, input)
		}
	})

	t.Run("SafeHTMLAttr", func(t *testing.T) {
		const attr = `class="x"`
		got, err := SafeHTMLAttr(attr)
		if err != nil {
			t.Fatal(err)
		}
		if got != template.HTMLAttr(attr) {
			t.Errorf("got %q, want %q", got, attr)
		}
	})

	t.Run("SafeJS", func(t *testing.T) {
		const js = `console.log("hi")`
		got, err := SafeJS(js)
		if err != nil {
			t.Fatal(err)
		}
		if got != template.JS(js) {
			t.Errorf("got %q, want %q", got, js)
		}
	})

	t.Run("SafeJSStr", func(t *testing.T) {
		const js = `hello\nworld`
		got, err := SafeJSStr(js)
		if err != nil {
			t.Fatal(err)
		}
		if got != template.JSStr(js) {
			t.Errorf("got %q, want %q", got, js)
		}
	})

	t.Run("SafeURL", func(t *testing.T) {
		const u = `https://example.com/path?q=1`
		got, err := SafeURL(u)
		if err != nil {
			t.Fatal(err)
		}
		if got != template.URL(u) {
			t.Errorf("got %q, want %q", got, u)
		}
	})
}

func TestSafeString_inputs(t *testing.T) {
	// already-typed values are unwrapped without double-encoding
	cases := []struct {
		name  string
		input any
		want  string
	}{
		{"string", "hello", "hello"},
		{"[]byte", []byte("hello"), "hello"},
		{"template.HTML", template.HTML("<b>hi</b>"), "<b>hi</b>"},
		{"template.CSS", template.CSS("color:red"), "color:red"},
		{"template.HTMLAttr", template.HTMLAttr(`id="x"`), `id="x"`},
		{"template.JS", template.JS("x()"), "x()"},
		{"template.JSStr", template.JSStr("str"), "str"},
		{"template.URL", template.URL("/foo"), "/foo"},
		{"int via Sprint", 42, "42"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := safeString(c.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}

	t.Run("nil errors", func(t *testing.T) {
		if _, err := safeString(nil); err == nil {
			t.Error("expected error for nil input")
		}
	})
}

func TestURLEscape(t *testing.T) {
	fm := FuncMap()

	urlEncode := fm["urlEncode"].(func(string) string)
	urlPathEscape := fm["urlPathEscape"].(func(string) string)

	if got := urlEncode("a b&c=d"); got != "a+b%26c%3Dd" {
		t.Errorf("urlEncode: got %q", got)
	}
	// PathEscape targets a single segment: / is encoded too
	if got := urlPathEscape("foo bar/baz"); got != "foo%20bar%2Fbaz" {
		t.Errorf("urlPathEscape: got %q", got)
	}
}
