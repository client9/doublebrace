// Package doublebrace provides a stdlib-only template.FuncMap for use with Go's
// text/template and html/template packages.
//
// All functions follow Go stdlib argument order: the primary value is the first
// argument. This matches direct Go calls and avoids pipeline-optimized argument
// order confusion. Single-argument functions work naturally in pipelines regardless.
//
// No function returns nil on success: a function that produces nothing returns
// an empty slice or map. Templates cannot tell the two apart — range, len, and
// index treat them identically — but encoding/json can, so a nil result would
// make jsonify emit null where a consuming script expects [].
//
// # Usage
//
//	import "github.com/client9/doublebrace"
//
//	t := template.New("page").Funcs(doublebrace.FuncMap())
//
// To combine with your own functions:
//
//	fns := doublebrace.Merge(doublebrace.FuncMap(), template.FuncMap{
//	    "myFunc": myFunc,
//	})
//	t := template.New("page").Funcs(fns)
//
// # Strings
//
//   - lower(s) string — convert to lowercase
//   - upper(s) string — convert to uppercase
//   - trim(s) string — remove leading and trailing whitespace
//   - trimPrefix(s, prefix) string — remove prefix if present
//   - trimSuffix(s, suffix) string — remove suffix if present
//   - trimLeft(s, cutset) string — remove leading characters contained in cutset
//   - trimRight(s, cutset) string — remove trailing characters contained in cutset
//   - contains(s, substr) bool — report whether substr is within s
//   - hasPrefix(s, prefix) bool — report whether s begins with prefix
//   - hasSuffix(s, suffix) bool — report whether s ends with suffix
//   - count(s, substr) int — count non-overlapping instances of substr in s; "" counts runes+1
//   - replace(s, old, new [, n]) string — replace first occurrence of old with new; optional n sets limit (-1 replaces all)
//   - replaceAll(s, old, new) string — replace all occurrences of old with new
//   - repeat(s, n) string — return n copies of s concatenated
//   - split(s, sep) []string — split s into substrings separated by sep
//   - join(elems, sep) string — join elements with sep; accepts any slice type, non-strings via fmt.Sprint
//   - fields(s) []string — split s on whitespace, discarding empty strings
//   - lenRunes(s) int — number of runes in s; unlike built-in len which counts bytes
//   - truncate(s, n) string — shorten to at most n runes; appends "…" if truncated
//   - firstUpper(s) string — uppercase first rune only; all other characters unchanged
//
// # Math
//
// All math functions accept any numeric type or numeric string as input.
// Results are float64; use printf for integer formatting.
//
//   - add(a, b) float64 — a + b
//   - sub(a, b) float64 — a - b
//   - mul(a, b) float64 — a * b
//   - div(a, b) float64 — a / b; error on zero divisor
//   - mod(a, b) float64 — floating-point remainder of a/b; error on zero divisor
//   - modBool(a, b) bool — report whether a is evenly divisible by b; useful for alternating rows
//   - abs(a) float64 — absolute value
//   - ceil(a) float64 — least integer value ≥ a
//   - floor(a) float64 — greatest integer value ≤ a
//   - round(a) float64 — nearest integer, rounding half away from zero
//   - pow(base, exp) float64 — base raised to exp
//   - min(args...) float64 — minimum value; accepts scalars, slices, or a mix
//   - max(args...) float64 — maximum value; accepts scalars, slices, or a mix
//   - clamp(val, min, max) float64 — constrain val to [min, max]
//
// # Encoding
//
//   - jsonify(v) string — marshal v to JSON; useful for <script> data blocks
//
// # Cast
//
// Cast functions convert values between types. Useful when frontmatter values
// arrive as strings and numeric operations are needed.
//
//   - toInt(v) int — convert to int; floats truncate toward zero; strings parse as integer then float ("3.9" → 3); out-of-range values are an error
//   - toFloat(v) float64 — convert to float64
//
// # Time
//
// Time functions return time.Time values. Use Go's time.Time methods directly
// in templates for formatting and field access: {{.Date.Format "2006-01-02"}},
// {{.Date.Year}}, {{.Date.Weekday}}, etc.
//
//   - now() time.Time — current local time; use {{now.UTC}} for UTC
//   - parseTime(layout, s) time.Time — parse s using Go reference-time layout
//
// # Path
//
// Path functions use forward slashes regardless of OS (suitable for URLs).
// They wrap the stdlib path package, not filepath.
//
//   - pathBase(p) string — last element of p; "foo/bar.html" → "bar.html"
//   - pathDir(p) string — all but last element; "foo/bar.html" → "foo"
//   - pathExt(p) string — file extension including dot; "bar.html" → ".html"
//   - pathJoin(elems...) string — join elements and clean the result
//   - pathClean(p) string — normalize: resolve . and .., remove double slashes
//
// # Safe Types
//
// These functions wrap string values in html/template typed aliases, preventing
// the template engine from escaping content that has already been sanitized.
// All accept string, []byte, any html/template typed value, or any type via
// fmt.Sprint. nil is an error.
//
//   - safeCSS(s) template.CSS — mark s safe for style attributes and <style> blocks
//   - safeHTML(s) template.HTML — mark s safe to render as raw HTML without escaping
//   - safeHTMLAttr(s) template.HTMLAttr — mark s safe as an HTML attribute name/value pair
//   - safeJS(s) template.JS — mark s safe for use inside <script> blocks
//   - safeJSStr(s) template.JSStr — mark s safe for interpolation inside JS string literals
//   - safeURL(s) template.URL — mark s safe for use in href/src/action attributes
//
// # URL Encoding
//
//   - urlEncode(s) string — percent-encode for query strings; spaces become +
//   - urlPathEscape(s) string — percent-encode a single path segment; / is encoded too
//
// # Collections — Immutability
//
// Collection functions always return newly allocated structures and never
// mutate or alias their arguments. This is a correctness requirement rather
// than a stylistic one: html/template is safe for concurrent execution, and the
// common server shape renders one template over shared data from many
// goroutines at once. A function that modified its input in place would be a
// data race there, and templates offer no way to express or even notice that a
// result borrows its argument's memory.
//
// The guarantee is shallow. The returned container is fresh, but its elements
// are shared with the input, so sort $pages yields a new slice holding the same
// map values. Mutating an element still affects the original.
//
// # Collections — Constructors
//
//   - list(elems...) []any — create a slice from values: list "a" "b" "c"
//   - dict(k, v, ...) map[string]any — create a map from key-value pairs: dict "name" "Alice"
//   - seq(n) []int — integers 1..n (1-based)
//   - seq(start, end) []int — integers start..end inclusive
//   - seq(start, end, step) []int — with step; negative step counts down
//
// A sequence may not exceed MaxSeqLen elements; a longer request is an error
// rather than an allocation. The limit is on the element count, not the numeric
// range, so a wide span with a large step is fine.
//
// # Collections — Sequence Access
//
// These functions operate on any slice type or string. String operations are
// rune-aware: multi-byte characters are never split.
//
//   - first(v) any — first element of a slice, or first rune of a string
//   - last(v) any — last element of a slice, or last rune of a string
//   - take(v, n) any — first n elements of a slice, or first n runes of a string; negative n takes from the end
//   - drop(v, n) any — skip first n elements of a slice, or first n runes of a string; negative n removes from the end
//
// # Collections — Sequence Transformation
//
//   - reverse(v) []any — new slice in reverse order
//   - compact(v) []any — remove consecutive duplicate elements; for full dedup: compact (sort $list)
//   - concat(slices...) []any — concatenate multiple slices into one
//   - sort(v [, key]) []any — type-aware sort: numeric types sort numerically, time.Time sorts chronologically, everything else sorts lexicographically; for []any the first non-nil element determines mode; key names a field for slice-of-maps (always lexicographic)
//   - sortNum(v [, key]) []any — numeric sort via float64 conversion; key names a field for slice-of-maps
//   - where(v, key, val) []any — filter slice of map[string]any where element[key] == val
//
// For descending order compose with reverse: reverse (sort $pages "Title")
//
// ISO 8601 date strings ("2006-01-02") sort correctly with lexicographic order.
//
// # Collections — Map Operations
//
//   - keys(m) []string — sorted keys of a map[string]any
//   - values(m) []any — values of a map[string]any ordered by sorted keys
//   - merge(maps...) map[string]any — shallow merge; later maps win on key collision
//
// # Collections — General
//
//   - in(v, val) bool — membership test: slice (element), map (key existence), string (substring)
//   - default(def, val) any — return val if non-zero, else def; zero: nil, false, 0, "", empty slice/map, all-zero array/struct (including a zero time.Time)
//   - cond(ctrl, a, b) any — ternary: return a if ctrl is truthy, else b
package doublebrace
