package doublebrace

import (
	"path"
	"text/template"
)

func pathFuncMap() template.FuncMap {
	return template.FuncMap{
		"pathBase":  PathBase,
		"pathDir":   PathDir,
		"pathExt":   PathExt,
		"pathJoin":  PathJoin,
		"pathClean": PathClean,
	}
}

// PathBase returns the last element of a slash-separated path.
// Trailing slashes are removed before extracting the last element.
//
//	pathBase "/a/b/c.html" → "c.html"
//	pathBase "/a/b/"       → "b"
func PathBase(p string) string { return path.Base(p) }

// PathDir returns all but the last element of a slash-separated path.
//
//	pathDir "/a/b/c.html" → "/a/b"
//	pathDir "/a/b/"       → "/a/b"
func PathDir(p string) string { return path.Dir(p) }

// PathExt returns the file extension of the last element of a path,
// including the leading dot. Returns "" if there is no extension.
//
//	pathExt "index.html" → ".html"
//	pathExt "Makefile"   → ""
func PathExt(p string) string { return path.Ext(p) }

// PathJoin joins path elements with slashes and cleans the result.
//
//	pathJoin "/a" "b" "c.html" → "/a/b/c.html"
func PathJoin(elems ...string) string { return path.Join(elems...) }

// PathClean returns the shortest equivalent path by applying lexical rules.
//
//	pathClean "/a/b/../c" → "/a/c"
//	pathClean "a//b"      → "a/b"
func PathClean(p string) string { return path.Clean(p) }
