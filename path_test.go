package doublebrace

import (
	"testing"
)

func TestPathFuncs(t *testing.T) {
	fm := FuncMap()

	base := fm["pathBase"].(func(string) string)
	dir := fm["pathDir"].(func(string) string)
	ext := fm["pathExt"].(func(string) string)
	join := fm["pathJoin"].(func(...string) string)
	clean := fm["pathClean"].(func(string) string)

	cases := []struct {
		name string
		got  string
		want string
	}{
		// pathBase
		{"base: file", base("foo/bar.html"), "bar.html"},
		{"base: dir trailing slash", base("foo/bar/"), "bar"},
		{"base: root", base("/"), "/"},
		{"base: no dir", base("file.txt"), "file.txt"},

		// pathDir
		{"dir: nested", dir("foo/bar/baz.html"), "foo/bar"},
		{"dir: single", dir("foo/bar.html"), "foo"},
		{"dir: no dir", dir("file.txt"), "."},
		{"dir: trailing slash", dir("foo/bar/"), "foo/bar"},

		// pathExt
		{"ext: html", ext("bar.html"), ".html"},
		{"ext: double", ext("archive.tar.gz"), ".gz"},
		{"ext: none", ext("Makefile"), ""},
		{"ext: dotfile", ext(".gitignore"), ".gitignore"}, // Go treats leading dot as separator

		// pathJoin
		{"join: two", join("foo", "bar"), "foo/bar"},
		{"join: three", join("foo", "bar", "baz.html"), "foo/bar/baz.html"},
		{"join: cleans dotdot", join("foo/bar", "..", "baz"), "foo/baz"},
		{"join: empty segment", join("foo", "", "bar"), "foo/bar"},

		// pathClean
		{"clean: double slash", clean("foo//bar"), "foo/bar"},
		{"clean: dotdot", clean("foo/bar/../baz"), "foo/baz"},
		{"clean: trailing slash", clean("foo/bar/"), "foo/bar"},
		{"clean: dot", clean("foo/./bar"), "foo/bar"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.got != c.want {
				t.Errorf("got %q, want %q", c.got, c.want)
			}
		})
	}
}
