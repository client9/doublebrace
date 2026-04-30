package doublebrace

import (
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

func stringFuncMap() template.FuncMap {
	return template.FuncMap{
		// Case
		"lower": strings.ToLower,
		"upper": strings.ToUpper,

		// Whitespace
		"trim":       strings.TrimSpace,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"trimLeft":   strings.TrimLeft,
		"trimRight":  strings.TrimRight,

		// Search
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"count":     strings.Count,

		// Transform
		"replace":    Replace,
		"replaceAll": strings.ReplaceAll,
		"repeat":     strings.Repeat,

		// Split / join
		"split":  strings.Split,
		"join":   strings.Join,
		"fields": strings.Fields,

		// Case — first character only
		"firstUpper": FirstUpper,

		// Length
		"lenRunes": LenRunes,

		// Truncate with ellipsis
		"truncate": Truncate,
	}
}

// LenRunes returns the number of runes in s. Unlike the built-in len, which
// counts bytes, LenRunes counts characters — so multi-byte characters such as
// "é" or "日" each count as one.
//
//	lenRunes "café" → 4
//	lenRunes "日本語" → 3
func LenRunes(s string) int {
	return utf8.RuneCountInString(s)
}

// Replace returns a copy of s with occurrences of old replaced by new.
// The optional n argument limits the number of replacements; if omitted,
// only the first occurrence is replaced. Use replaceAll to replace all.
//
//	replace "aabbaa" "a" "x"     → "xabbaa"
//	replace "aabbaa" "a" "x" 3  → "xxbbxa"
//	replace "aabbaa" "a" "x" -1 → "xxbbxx"
func Replace(s, old, new string, n ...int) string {
	count := 1
	if len(n) > 0 {
		count = n[0]
	}
	return strings.Replace(s, old, new, count)
}

// FirstUpper returns s with the first rune converted to its Unicode title case.
// All other characters are left unchanged. It is rune-safe: multi-byte leading
// characters such as "é" are handled correctly.
//
//	firstUpper "go"              → "Go"
//	firstUpper "hello world"    → "Hello world"
//	firstUpper "élan"           → "Élan"
func FirstUpper(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

// Truncate shortens s to at most n runes. If s is longer it is cut and an
// ellipsis ("…") is appended. n includes the ellipsis, so the result is
// always at most n runes long.
//
//	truncate "hello world" 8 → "hello w…"
//	truncate "hi" 8          → "hi"
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(runes[:n-1]) + "…"
}
