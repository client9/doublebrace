# Doublebrace Recipes

Tricks and techniques using doublebrace functions.

**Descending sort**
```
reverse (sort $list)
```

**Capitalize first letter only** (leave rest unchanged)
```
firstUpper $str
```

**Capitalize first letter, lowercase the rest** (like Jinja2 `capitalize`)
```
firstUpper (lower $str)
```

**Last N elements**
```
take $list -2
```

**All but last N elements**
```
drop $list -2
```

**Join with special last separator** ("a, b and c")
```
printf "%s and %s" (join (drop $list -1) ", ") (last $list)
```
Needs two or more elements: a one-element list drops to empty and renders
" and a", and an empty list is an error from `last`.

**Add trailing separator**
```
printf "%s," (join $list ", ")
```

**Chomp** (remove trailing newline)
```
trimRight $str "\r\n"
```

**toString**
```
printf "%v" $val
```

**Date formatting**
```
{{ $date.Format "2006-01-02" }}
```

**UTC time**
```
{{ now.UTC }}
```

**Simple loop counter** (Go 1.24+)
```
{{ range 10 }}{{ . }}{{ end }}
```
outputs 0–9. Use `seq` for non-zero start or step: `{{ range (seq 3 7) }}`.

**String Pad Left** (right-align in a 20-column field)
```
{{ printf "%20s" $str }}
```

**String Pad Right** (left-align in a 20-column field)
```
{{ printf "%-20s" $str }}
```

Both pad to a minimum width and never truncate; use `truncate` for an upper
bound. The width counts runes rather than bytes, so `"日本"` pads to the same
column as `"ab"`.
