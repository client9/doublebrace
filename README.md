# doublebrace - additional functions for Go templates

The default functions in `text/template` and `html/template` are minimal. This extends them.

[![Go Reference](https://pkg.go.dev/badge/github.com/client9/doublebrace.svg)](https://pkg.go.dev/github.com/client9/doublebrace)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/client9/doublebrace/actions/workflows/go.yml/badge.svg)](https://github.com/client9/doublebrace/actions)

## Quick Start

Requires Go >= 1.23.

```go
import (
	"text/template"

	"github.com/client9/doublebrace"
)

t := template.New("foo").Funcs(doublebrace.FuncMap())
```

To combine with your own functions — later maps win, so these are overridable:

```go
fns := doublebrace.Merge(doublebrace.FuncMap(), template.FuncMap{
	"myFunc": myFunc,
})
t := template.New("foo").Funcs(fns)
```

## What's Included?

- strings — case, trim, search, replace, split/join, truncate, rune-aware length
- math — arithmetic, rounding, min/max, clamp, pow
- cast — `toInt`, `toFloat` for frontmatter values that arrive as strings
- encoding — `jsonify` for JSON output and `data-` attributes (inside `<script>`, `html/template` already emits JSON from a bare `{{ . }}`)
- date and time — `now`, `parseTime`; use `time.Time` methods for formatting
- url / safe types — `urlEncode`, `urlPathEscape`, `safeHTML`, `safeCSS`, etc.
- path — `pathBase`, `pathDir`, `pathExt`, `pathJoin`, `pathClean`
- lists (slices) — `list`, `seq`, `first`, `last`, `take`, `drop`, `sort`, `sortNum`, `reverse`, `concat`, `compact`
- dicts (maps) — `dict`, `keys`, `values`, `merge`
- filtering and tests — `where` to filter by field, `in` for membership, `default` and `cond` for fallbacks

The full list, with the argument order and behavior of each, is in the
[package documentation](https://pkg.go.dev/github.com/client9/doublebrace).

## Argument Types

Template data loses the static type of everything it passes, so the functions
are deliberately permissive in three specific ways:

- **Text** — anywhere a function takes text, any value of string kind works: a
  plain `string`, a named type (`type Slug string`), or an `html/template` typed
  value. The result is always a plain string, which for a safe type is a
  security property: a truncated `template.URL` is no longer the URL that was
  vouched for, so it gets escaped like any other text.
- **Counts** — the `n` of `take`, `drop`, `truncate`, `repeat`, and `replace`,
  and the bounds of `seq`, accept any numeric type or numeric string. This is
  what makes `take $list (div $n 2)` work at all: the math functions return
  `float64`, and templates require an assignable type rather than converting.
- **Equality** — `where`, `in`, and `compact` compare numbers by value across
  types and strings by their text across string kinds, so a `1` decoded from
  JSON as `float64` matches an integer literal, and a named type matches the
  literal it was written from. The type a value arrives as is an accident of the
  decoder, and a filter that quietly returns nothing is indistinguishable from
  data that did not match.

## Goals

- **Independent and exportable** — serves as a base, or for use in different templating systems
- **Stdlib only** — keep it simple; functions requiring external deps go in a different module
- **Not pipeline-based** — pipeline order looks elegant for single-argument functions, then gets confusing. Argument order follows Go stdlib (subject first).
- **Prefer separate functions over extra arguments** — `sort` and `sortNum` instead of a mode flag
- **Immutable data structures** — collection functions always return newly allocated values and never mutate or alias their arguments, so one template can render over shared data from many goroutines. The guarantee is shallow: the container is fresh, its elements are still shared.
- **Wrong answers are errors** — a missing field in `where`, a `nil` element in `sort`, a count that overflows. Silently returning nothing, or an arbitrary order, is the one outcome a template author cannot debug.

## Alternatives

[Masterminds/sprig](https://github.com/Masterminds/sprig) — appears semi-abandoned, pipeline-based, has a number of unusual functions and dependencies.

[Hugo](https://gohugo.io/) — the static site generator has many functions, but inconsistent design and argument order optimized for pipelines. Implementation is tightly coupled to Hugo internals.

## Not Included

- **Internationalization / titlecase** — requires `golang.org/x/text`; good for a separate module. Note: all string operations here are rune-aware.
- **Regular expressions** — defer until use cases in templates are better understood
- **Base64 encoding** — two competing encodings (standard vs URL-safe); add when use case is clear
- **Random / shuffle** — non-deterministic output is problematic for static site generators.
- **Checksum and hashes** — limited uses, many variations; good for a separate module
- **Cryptography** — limited use, many variations; belongs in application code, not a template
- **OS and environment** — pass these as data to the template instead
- **Math trig** — limited utility in HTML templates
