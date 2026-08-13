package doublebrace

import (
	"fmt"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

// Every string argument in the FuncMap is registered through one of the strFn
// adapters below, so what counts as a string is asString's answer — any value of
// string kind, including a named type (type Slug string) and an html/template
// typed value.
//
// The alternative was registering strings.ToLower and the package's own
// functions directly, which is how these entries began. That made the accepted
// types an artifact of how a function happened to be registered rather than a
// rule: take $slug and in $slug "x" worked, while lower $slug and truncate
// $slug 5 failed with "wrong type for value; expected string; got Slug" — the
// same value, rejected by half the functions that operate on it. Templates
// carry named string types routinely, and a template author has no way to tell
// which half a given function falls in.
//
// The coercion is deliberately here rather than in the exported Go functions,
// which keep their string parameters. It exists because a template loses the
// static type of every value it passes; a Go caller holding a Slug knows it is
// one and writes string(slug). Widening the exported signatures to any would
// buy nothing for that caller and cost the signature that documents them.
//
// The stdlib entries are still not given exported wrappers of this package: an
// alias forwarding to strings.ToLower remains a worse way to call
// strings.ToLower, and doc.go names the stdlib function for each so a Go caller
// knows what to call instead. Adapting is not aliasing — the adapter is
// unexported and exists for the template, not for Go.
func stringFuncMap() template.FuncMap {
	return template.FuncMap{
		// Case
		"lower": strFn1("lower", strings.ToLower),
		"upper": strFn1("upper", strings.ToUpper),

		// Whitespace
		"trim":       strFn1("trim", strings.TrimSpace),
		"trimPrefix": strFn2("trimPrefix", strings.TrimPrefix),
		"trimSuffix": strFn2("trimSuffix", strings.TrimSuffix),
		"trimLeft":   strFn2("trimLeft", strings.TrimLeft),
		"trimRight":  strFn2("trimRight", strings.TrimRight),

		// Search
		"contains":  strFn2("contains", strings.Contains),
		"hasPrefix": strFn2("hasPrefix", strings.HasPrefix),
		"hasSuffix": strFn2("hasSuffix", strings.HasSuffix),
		"count":     strFn2("count", strings.Count),

		// Transform
		"replace":    replaceFn,
		"replaceAll": strFn3("replaceAll", strings.ReplaceAll),
		"repeat":     strCountFn("repeat", Repeat),

		// Split / join
		"split":  strFn2("split", strings.Split),
		"join":   joinFn,
		"fields": strFn1("fields", strings.Fields),

		// Case — first character only
		"firstUpper": strFn1("firstUpper", FirstUpper),

		// Length
		"lenRunes": strFn1("lenRunes", LenRunes),

		// Truncate with ellipsis
		"truncate": strCountFn("truncate", Truncate),
	}
}

// strArg coerces one template argument to a string, naming fn in the error.
// It is the single point where the FuncMap decides what a string is; everything
// below routes through it, and it routes through asString.
func strArg(fn string, v any) (string, error) {
	s, ok := asString(v)
	if !ok {
		return "", fmt.Errorf("%s: expected a string, got %T", fn, v)
	}
	return s, nil
}

// strFn1, strFn2, and strFn3 adapt a function of one, two, or three string
// arguments into one taking any. They are generic in the result so that the
// bool of contains, the int of count, and the []string of split all adapt
// through the same three functions rather than one per return type.
//
// The result is passed through untouched, so the package's guarantees stay the
// responsibility of the function being adapted — split and fields return a
// non-nil empty slice on their own, which is what TestFunctionsReturnEmptyNotNil
// checks. The zero R returned alongside an error is nil for a slice result, and
// that is allowed: the rule is that nothing returns nil on success.
func strFn1[R any](name string, fn func(string) R) func(any) (R, error) {
	return func(v any) (R, error) {
		s, err := strArg(name, v)
		if err != nil {
			var zero R
			return zero, err
		}
		return fn(s), nil
	}
}

func strFn2[R any](name string, fn func(string, string) R) func(any, any) (R, error) {
	return func(a, b any) (R, error) {
		var zero R
		sa, err := strArg(name, a)
		if err != nil {
			return zero, err
		}
		sb, err := strArg(name, b)
		if err != nil {
			return zero, err
		}
		return fn(sa, sb), nil
	}
}

func strFn3[R any](name string, fn func(string, string, string) R) func(any, any, any) (R, error) {
	return func(a, b, c any) (R, error) {
		var zero R
		sa, err := strArg(name, a)
		if err != nil {
			return zero, err
		}
		sb, err := strArg(name, b)
		if err != nil {
			return zero, err
		}
		sc, err := strArg(name, c)
		if err != nil {
			return zero, err
		}
		return fn(sa, sb, sc), nil
	}
}

// strCountFn adapts a function taking a string and a count — Truncate and
// Repeat, which already report their own errors.
func strCountFn(name string, fn func(string, any) (string, error)) func(any, any) (string, error) {
	return func(v, n any) (string, error) {
		s, err := strArg(name, v)
		if err != nil {
			return "", err
		}
		return fn(s, n)
	}
}

// replaceFn and joinFn are the two shapes no adapter above fits: replace takes
// three strings and a variadic count, and join takes a collection first and its
// separator second.
func replaceFn(s, old, repl any, n ...any) (string, error) {
	ss, err := strArg("replace", s)
	if err != nil {
		return "", err
	}
	sold, err := strArg("replace", old)
	if err != nil {
		return "", err
	}
	srepl, err := strArg("replace", repl)
	if err != nil {
		return "", err
	}
	return Replace(ss, sold, srepl, n...)
}

func joinFn(v, sep any) (string, error) {
	ssep, err := strArg("join", sep)
	if err != nil {
		return "", err
	}
	return Join(v, ssep)
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
// n is any numeric type or numeric string, converted as toInt does.
//
// The replacement is named repl, as in regexp, rather than new as in
// strings.Replace: new is a predeclared identifier, and shadowing it in a
// signature that appears in godoc is worth avoiding even though the stdlib does.
//
//	replace "aabbaa" "a" "x"    → "xabbaa"
//	replace "aabbaa" "a" "x" 3  → "xxbbxa"
//	replace "aabbaa" "a" "x" -1 → "xxbbxx"
func Replace(s, old, repl string, n ...any) (string, error) {
	count := 1
	if len(n) > 0 {
		c, err := toCount("replace", n[0])
		if err != nil {
			return "", err
		}
		count = c
	}
	return strings.Replace(s, old, repl, count), nil
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
// n is any numeric type or numeric string, converted as toInt does.
//
//	repeat "ab" 3   → "ababab"
//	repeat "-" 0    → ""
//	repeat "-" -1   → error
func Repeat(s string, n any) (string, error) {
	count, err := toCount("repeat", n)
	if err != nil {
		return "", err
	}
	if count < 0 {
		return "", fmt.Errorf("repeat: negative count (%d)", count)
	}
	runes := LenRunes(s)
	if runes == 0 {
		// Nothing to repeat, so no count can overflow the limit. Returning
		// early also keeps the division below from dividing by zero.
		return "", nil
	}
	// Divide rather than multiply: runes*count overflows int for a large enough
	// count, and a wrapped product compares as comfortably under the limit.
	if count > MaxRepeatLen/runes {
		return "", fmt.Errorf("repeat: %d copies of a %d-rune string exceeds the limit of %d runes",
			count, runes, MaxRepeatLen)
	}
	return strings.Repeat(s, count), nil
}

// Truncate shortens s to at most n runes. If s is longer it is cut and an
// ellipsis ("…") is appended. n includes the ellipsis, so the result is
// always at most n runes long.
//
// n is any numeric type or numeric string, converted as toInt does — so a
// length computed with the math functions, which all return float64, is usable
// as one: truncate $title (sub $width 4).
//
//	truncate "hello world" 8 → "hello w…"
//	truncate "hi" 8          → "hi"
func Truncate(s string, n any) (string, error) {
	count, err := toCount("truncate", n)
	if err != nil {
		return "", err
	}
	if count <= 0 {
		return "", nil
	}
	runes := []rune(s)
	if len(runes) <= count {
		return s, nil
	}
	if count == 1 {
		return "…", nil
	}
	return string(runes[:count-1]) + "…", nil
}
