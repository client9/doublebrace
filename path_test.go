package doublebrace

import (
	"testing"
)

// The exported functions are called directly rather than pulled out of the
// FuncMap and asserted to a signature. The registered entries take any, since
// they go through the strFn adapters that accept every string kind, so a type
// assertion here would only pin how they are registered — which is what
// TestStringArgs_acceptStringKinds checks, through a template, where it matters.
func TestPathFuncs(t *testing.T) {
	base, dir, ext := PathBase, PathDir, PathExt
	join, clean := PathJoin, PathClean

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
