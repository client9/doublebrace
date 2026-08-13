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
// Most functions are also exported as named Go functions — Truncate, Sort,
// Where — so they are callable directly and visible on pkg.go.dev. A function
// whose behavior is exactly a stdlib call is the exception: it has no exported
// counterpart here, because an alias that only forwards to strings.ToLower
// would be a worse way to call strings.ToLower. The listings below name the
// stdlib function in each such case; call it directly. Such a function may only
// be registered if it already satisfies the guarantees above, in particular
// that it never returns nil.
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
//   - lower(s) string — convert to lowercase (strings.ToLower)
//   - upper(s) string — convert to uppercase (strings.ToUpper)
//   - trim(s) string — remove leading and trailing whitespace (strings.TrimSpace)
//   - trimPrefix(s, prefix) string — remove prefix if present (strings.TrimPrefix)
//   - trimSuffix(s, suffix) string — remove suffix if present (strings.TrimSuffix)
//   - trimLeft(s, cutset) string — remove leading characters contained in cutset (strings.TrimLeft)
//   - trimRight(s, cutset) string — remove trailing characters contained in cutset (strings.TrimRight)
//   - contains(s, substr) bool — report whether substr is within s (strings.Contains)
//   - hasPrefix(s, prefix) bool — report whether s begins with prefix (strings.HasPrefix)
//   - hasSuffix(s, suffix) bool — report whether s ends with suffix (strings.HasSuffix)
//   - count(s, substr) int — count non-overlapping instances of substr in s; "" counts runes+1 (strings.Count)
//   - replace(s, old, repl [, n]) string — replace first occurrence of old with repl; optional n sets limit (-1 replaces all)
//   - replaceAll(s, old, repl) string — replace all occurrences of old with repl (strings.ReplaceAll)
//   - repeat(s, n) string — return n copies of s concatenated; at most MaxRepeatLen runes, and a negative n is an error
//   - split(s, sep) []string — split s into substrings separated by sep (strings.Split)
//   - join(elems, sep) string — join elements with sep; accepts any slice or array, non-strings via fmt.Sprint
//   - fields(s) []string — split s on whitespace, discarding empty strings (strings.Fields)
//   - lenRunes(s) int — number of runes in s; unlike built-in len which counts bytes
//   - truncate(s, n) string — shorten to at most n runes; appends "…" if truncated
//   - firstUpper(s) string — uppercase first rune only; all other characters unchanged
//
// # Text arguments
//
// Wherever a function takes text it accepts any value of string kind: a plain
// string, a named type (type Slug string), or an html/template typed value.
// This holds for every string argument in the FuncMap — the functions above,
// urlEncode and urlPathEscape, and first, last, take, drop, and in, which treat
// a string as a sequence of runes. A function is never picky about a type its
// neighbour accepts.
//
// The result is always a plain string, never the input's own type. For an
// html/template value that is a security property rather than a detail:
// lowercasing or truncating markup produces text that is no longer the markup
// that was vouched for, so it is escaped for whatever context it lands in. A
// template.URL that has been through lower is rejected as a URL, which is the
// fail-closed direction.
//
// The exported Go functions keep their string parameters. The coercion is for
// templates, which lose the static type of every value they pass; a Go caller
// holding a Slug knows it is one and writes string(slug).
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
//   - min(args...) float64 — minimum value; accepts scalars, slices, arrays, or a mix
//   - max(args...) float64 — maximum value; accepts scalars, slices, arrays, or a mix
//   - clamp(val, min, max) float64 — constrain val to [min, max]
//
// # Counts
//
// Wherever a function takes a count — the n of take, drop, truncate, repeat,
// and replace, and the bounds of seq — it accepts any numeric type or numeric
// string and converts it the way toInt does, truncating a float toward zero and
// reporting an out-of-range value as an error rather than a wrapped length.
//
// This is what makes a count composable with the arithmetic above. A count is
// the most likely argument in a template to be computed rather than written
// down, and templates do not convert an argument, they require an assignable
// type. An int parameter would therefore reject the float64 every math function
// returns, along with an int64 field and anything decoded from JSON, and
// take $list (div $n 2) could not be written at all.
//
// # Encoding
//
//   - jsonify(v) string — marshal v to JSON; for text/template output and html/template data attributes
//
// Inside a <script> block, use the bare action instead: html/template already
// marshals data to JSON there, whereas jsonify's plain string is escaped into a
// JavaScript string literal, giving a string that holds JSON rather than an
// object. See Jsonify for the details and for the already-encoded case.
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
// Marking a value safe turns off the protection html/template applies to it, so
// the content must be trusted — and for safeHTML, balanced. See SafeHTML.
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
// Both take any value of string kind — see Text arguments above.
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
// range, so a wide span with a large step is fine. repeat is bounded the same
// way by MaxRepeatLen — wherever a count can come from data, a mistyped or
// hostile one is an error rather than an allocation.
//
// seq's bounds are counts, so they accept any numeric type or numeric string —
// see Counts above.
//
// # Collections — Sequence Access
//
// These functions operate on any slice, array, or string. String operations are
// rune-aware: multi-byte characters are never split.
//
// "String" means any value of string kind, the same rule the text functions
// follow — see Text arguments above. The result is a plain string rather than
// the input's own type: slicing markup by runes can cut a tag or an entity in
// half, so a truncated template.HTML is no longer trusted markup and is escaped
// like any other string.
//
// They return any, rather than the []any returned elsewhere, because the result
// follows the input: a string argument yields a string or a rune, not a slice.
//
// The n of take and drop is a count, so it accepts any numeric type or numeric
// string — see Counts above.
//
//   - first(v) any — first element of a slice, or first rune of a string
//   - last(v) any — last element of a slice, or last rune of a string
//   - take(v, n) any — first n elements of a slice, or first n runes of a string; negative n takes from the end
//   - drop(v, n) any — skip first n elements of a slice, or first n runes of a string; negative n removes from the end
//
// # Collections — Sequence Transformation
//
//   - reverse(v) []any — new slice in reverse order
//   - compact(v) []any — remove consecutive duplicate elements; numbers compare by value across types; for full dedup: compact (sort $list)
//   - concat(slices...) []any — concatenate multiple slices into one
//   - sort(v [, key]) []any — type-aware sort: numeric types sort numerically, time.Time sorts chronologically, everything else sorts lexicographically; for []any the first non-nil element determines mode; key names a field on each element (always lexicographic)
//   - sortNum(v [, key]) []any — numeric sort, including numeric strings; key names a field on each element
//   - where(v, key, val) []any — filter a slice by field equality; numbers compare by value across types; a missing field is an error
//
// These take a slice or an array only. A string is not a sequence to them, so
// reverse "abc" is an error rather than "cba" — unlike first, last, take, and
// drop, which do treat a string as a sequence of runes. Reversing text is a
// different job from reversing a list: it needs grapheme clusters, not runes,
// or it takes apart any character built from a combining mark.
//
// For descending order compose with reverse: reverse (sort $pages "Title")
//
// ISO 8601 date strings ("2006-01-02") sort correctly with lexicographic order.
//
// The key forms of sort, sortNum, and where read a named field from each
// element. An element may be a struct or a pointer to one, or a map with any
// string-kind key — so a plain []Page works, not only []map[string]any. Struct
// fields match by exact name, must be exported, and may be promoted from an
// embedded struct. Methods are not called. A missing, unexported, or
// unreadable field is an error rather than a silent non-match.
//
// Numeric ordering is exact at every magnitude. Integers wider than a float64's
// 53-bit mantissa — database IDs and timestamps in nanoseconds, most often —
// order by their real values rather than collapsing into ties. A numeric string
// is the exception: parsing it produces a float64, and that is where its
// precision stops.
//
// # Collections — Map Operations
//
//   - keys(m) []string — sorted keys of a map
//   - values(m) []any — values of a map ordered by sorted keys
//   - merge(maps...) map[string]any — shallow merge; later maps win on key collision
//
// These accept a map with any value type and any string-kind key, so
// map[string]string and map[Slug]int work alongside map[string]any. The key
// must be a string kind because these order the keys; in accepts any key type
// because it only probes for one.
//
// # Collections — General
//
//   - in(v, val) bool — membership test: slice/array (element), map (key existence), string (substring); numbers compare by value across types
//   - default(def, val) any — return val if non-zero, else def; zero: nil, false, 0, "", empty slice/map, all-zero array/struct (including a zero time.Time)
//   - cond(ctrl, a, b) any — ternary: return a if ctrl is truthy, else b
package doublebrace
