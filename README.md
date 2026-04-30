# doublebrace - additional functions for Go templates

The default functions in `text/template` and `html/template` are minimal. This extends them.

## Quick Start

```go
import (
	"github.com/client9/doublebrace"
)

t := template.New("foo").Funcs(doublebrace.FuncMap())

```


## What's Included?

- strings — case, trim, search, replace, split/join, truncate, rune-aware length
- math — arithmetic, rounding, min/max, clamp, pow
- cast — `toInt`, `toFloat` for frontmatter values that arrive as strings
- encoding — `jsonify` for embedding data in `<script>` blocks
- date and time — `now`, `parseTime`; use `time.Time` methods for formatting
- url / safe types — `urlEncode`, `urlPathEscape`, `safeHTML`, `safeCSS`, etc.
- path — `pathBase`, `pathDir`, `pathExt`, `pathJoin`, `pathClean`
- lists (slices) — `list`, `seq`, `take`, `drop`, `sort`, `sortNum`, `reverse`, `concat`, …
- dicts (maps) — `dict`, `keys`, `values`, `merge`

## Goals

- **Independent and exportable** — serves as a base, or for use in different templating systems
- **Stdlib only** — keep it simple; functions requiring external deps go in a different module
- **Not pipeline-based** — pipeline order looks elegant for single-argument functions, then gets confusing. Argument order follows Go stdlib (subject first).
- **Prefer separate functions over extra arguments** — `sort` and `sortNum` instead of a mode flag
- **Immutable data structures** — all functions return new values, never modify inputs

## Alternatives

[Masterminds/sprig](https://github.com/Masterminds/sprig) — appears semi-abandoned, pipeline-based, has a number of unusual functions and dependencies.

[Hugo](https://gohugo.io/) — the static site generator has many functions, but inconsistent design and argument order optimized for pipelines. Implementation is tightly coupled to Hugo internals.

## Not Included

- **Internationalization / titlecase** — requires `golang.org/x/text`; good for a separate module. Note: all string operations here are rune-aware.
- **Regular expressions** — defer until use cases in templates are better understood
- **Base64 encoding** — two competing encodings (standard vs URL-safe); add when use case is clear
- **Random / shuffle** — non-deterministic output is problematic for static site generators.
- **Checksum and hashes** — limited uses, many variations; good for a separate module
- **Cryptography** — limited use, many variations,
- **OS and environment** — pass these as data to the template instead
- **Math trig** — limited utility in HTML templates
