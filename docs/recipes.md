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
capitalize $str
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

**String Pad Right**
```
{{ printf "%20s" $atr }}
```

**String Pad Left**
```
{{ printf "%-20s" $astr }}
```
