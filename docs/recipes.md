# Doublebrace Recipes

Tricks and techniques using doublebrace functions.

Each recipe shows what it renders, using the `expr → result` convention of the
package documentation. Every one is run against this data:

```
$list = ["a" "b" "c"]
$str  = "hello WORLD"
$line = "text\r\n"
$val  = 42
$date = 2024-03-15 09:30:00 UTC
```

`TestRecipes` in `recipes_test.go` executes each snippet below against exactly
that data and checks the output shown here, so a recipe that stops working — or
one whose output was written down wrong — fails the build.

**Descending sort**
```
{{ reverse (sort $list) }}  → [c b a]
```

**Capitalize first letter only** (leave rest unchanged)
```
{{ firstUpper $str }}  → "Hello WORLD"
```

**Capitalize first letter, lowercase the rest** (like Jinja2 `capitalize`)
```
{{ firstUpper (lower $str) }}  → "Hello world"
```

**Last N elements**
```
{{ take $list -2 }}  → [b c]
```

**All but last N elements**
```
{{ drop $list -2 }}  → [a]
```

**Join with special last separator** ("a, b and c")
```
{{ printf "%s and %s" (join (drop $list -1) ", ") (last $list) }}  → "a, b and c"
```
Needs two or more elements: a one-element list drops to empty and renders
" and a", and an empty list is an error from `last`.

**Add trailing separator**
```
{{ printf "%s," (join $list ", ") }}  → "a, b, c,"
```

**Chomp** (remove trailing newline)
```
{{ trimRight $line "\r\n" }}  → "text"
```

**toString**
```
{{ printf "%v" $val }}  → "42"
```

**Date formatting**
```
{{ $date.Format "2006-01-02" }}  → "2024-03-15"
```

**String Pad Left** (right-align in a 20-column field)
```
{{ printf "%20s" $str }}  → "         hello WORLD"
```

**String Pad Right** (left-align in a 20-column field)
```
{{ printf "%-20s" $str }}  → "hello WORLD         "
```

Both pad to a minimum width and never truncate; use `truncate` for an upper
bound. The width counts runes rather than bytes, so `"日本"` pads to the same
column as `"ab"`.

**UTC time**
```
{{ now.UTC }}
```
The only recipe with no output pinned above, because it does not have a fixed
one. The test checks that it executes without error.

**Simple loop counter** (Go 1.24+)
```
{{ range 10 }}{{ . }}{{ end }}  → 0123456789
```
outputs 0–9. Use `seq` for a non-zero start or a step: `{{ range (seq 3 7) }}`.

Ranging over an integer is a `text/template` feature, not a doublebrace one, and
it arrived after the Go version in `go.mod`. The test therefore skips it on the
declared floor rather than pinning output the minimum toolchain cannot produce.
