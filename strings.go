package doublebrace

import (
	"fmt"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

// Entries assigned straight from strings are deliberately not wrapped in an
// exported function of this package: an alias that only forwards to
// strings.ToLower is a worse way to call strings.ToLower. Each is named in the
// doc.go listing so a Go caller knows what to call instead. Assigning a stdlib
// function directly is only allowed where it already honors the package's
// guarantees — notably that it never returns nil, which
// TestFunctionsReturnEmptyNotNil pins for split and fields.
//
// Repeat is the counter-example: strings.Repeat panics on a negative count and
// allocates whatever a large one asks for, neither of which a template author
// should be able to trigger from data, so it is wrapped rather than assigned.
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
		"repeat":     Repeat,

		// Split / join
		"split":  strings.Split,
		"join":   Join,
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

// Replace returns a copy of s with occurrences of old replaced by repl.
// The optional n argument limits the number of replacements; if omitted,
// only the first occurrence is replaced. Use replaceAll to replace all.
//
// The replacement is named repl, as in regexp, rather than new as in
// strings.Replace: new is a predeclared identifier, and shadowing it in a
// signature that appears in godoc is worth avoiding even though the stdlib does.
//
//	replace "aabbaa" "a" "x"    → "xabbaa"
//	replace "aabbaa" "a" "x" 3  → "xxbbxa"
//	replace "aabbaa" "a" "x" -1 → "xxbbxx"
func Replace(s, old, repl string, n ...int) string {
	count := 1
	if len(n) > 0 {
		count = n[0]
	}
	return strings.Replace(s, old, repl, count)
}

// Join concatenates the elements of v, placing sep between them.
//
// Unlike strings.Join, v may be any slice type, not just []string. This is what
// makes join composable with the collection functions, which all return []any:
// join (sort $list) ", " works, as does join (drop $list -1) ", ".
//
// Elements that are not strings are formatted with fmt.Sprint, matching how
// {{ . }} renders them; a nil element therefore joins as "<nil>".
//
//	join (list "a" "b" "c") ", "  → "a, b, c"
//	join (split "a,b" ",") "-"    → "a-b"
//	join (list 1 2 3) "+"         → "1+2+3"
func Join(v any, sep string) (string, error) {
	if ss, ok := v.([]string); ok {
		return strings.Join(ss, sep), nil
	}
	elems, err := toSlice(v)
	if err != nil {
		return "", fmt.Errorf("join: %w", err)
	}
	parts := make([]string, len(elems))
	for i, e := range elems {
		if s, ok := e.(string); ok {
			parts[i] = s
			continue
		}
		parts[i] = fmt.Sprint(e)
	}
	return strings.Join(parts, sep), nil
}

// FirstUpper returns s with the first rune converted to its Unicode title case.
// All other characters are left unchanged. It is rune-safe: multi-byte leading
// characters such as "é" are handled correctly.
//
// To also lowercase the rest — what Jinja2 and Twig call capitalize — compose:
// firstUpper (lower $s).
//
//	firstUpper "go"           → "Go"
//	firstUpper "hello world"  → "Hello world"
//	firstUpper "élan"         → "Élan"
//	firstUpper "ǳagreb"       → "ǲagreb"  (title case, not the all-caps "Ǳ")
func FirstUpper(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	// ToTitle, not ToUpper. The two agree on every character except the Latin
	// digraphs (ǆ ǉ ǌ ǳ), where the title-case form is the correct one for
	// capitalizing a word: "ǲagreb", not "Ǳagreb", which is the all-caps form.
	return string(unicode.ToTitle(r)) + s[size:]
}

// MaxRepeatLen is the longest string repeat will produce, in runes. Like
// MaxSeqLen it is a guardrail against a mistyped or data-driven count turning
// into an unbounded allocation, and it is set by the same reasoning: 10000
// characters of a repeated string is already far more than any page wants, so no
// legitimate template comes near it. The uses that exist — indentation,
// separator rules, padding, a bar drawn with repeated blocks — run to tens of
// characters.
//
// The limit counts runes rather than bytes because that is how this package
// measures strings everywhere else (lenRunes, truncate, take, drop). It still
// bounds the allocation: a rune is at most four bytes, so the result cannot
// exceed 4*MaxRepeatLen bytes.
//
// It is a constant for the reason MaxSeqLen is: a mutable global would be shared
// state in a package built for concurrent template execution. A caller who needs
// longer output can register their own repeat over this one with Merge.
const MaxRepeatLen = 10000

// Repeat returns n copies of s concatenated.
//
// It differs from strings.Repeat, which is not safe to expose to a template
// directly: that function panics on a negative count, and will allocate however
// much a large one asks for. Both are reachable whenever the count comes from
// data — repeat "─" $width — so both are errors here instead. A panic in a
// template function is recovered by text/template and surfaces as an execution
// error anyway, but an unbounded allocation is not caught by anything.
//
//	repeat "ab" 3   → "ababab"
//	repeat "-" 0    → ""
//	repeat "-" -1   → error
func Repeat(s string, n int) (string, error) {
	if n < 0 {
		return "", fmt.Errorf("repeat: negative count (%d)", n)
	}
	runes := LenRunes(s)
	if runes == 0 {
		// Nothing to repeat, so no count can overflow the limit. Returning
		// early also keeps the division below from dividing by zero.
		return "", nil
	}
	// Divide rather than multiply: runes*n overflows int for a large enough
	// count, and a wrapped product compares as comfortably under the limit.
	if n > MaxRepeatLen/runes {
		return "", fmt.Errorf("repeat: %d copies of a %d-rune string exceeds the limit of %d runes",
			n, runes, MaxRepeatLen)
	}
	return strings.Repeat(s, n), nil
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
