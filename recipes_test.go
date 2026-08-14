package doublebrace

import (
	"strings"
	"testing"
	"text/template"
	"time"
)

// The recipes in docs/recipes.md are executed here, one table entry per recipe.
//
// The point is not that these functions work — they are tested thoroughly
// elsewhere — but that the document stays true. A cheat sheet exists to be
// copied out of, and two of its entries were wrong when this test was written:
// the pad-left and pad-right examples had their headings swapped, so following
// the doc produced the opposite alignment, and the "a, b and c" recipe silently
// misrendered a one-element list. Neither was a broken function.
//
// That is why the document now shows the output of every recipe and this table
// holds the same strings. Pinning behavior alone would not have caught either
// bug, because both were prose: a test asserting that "%20s" right-aligns
// passes happily while the heading above it says the opposite. Showing the
// output makes a mislabeled recipe visible to a reader, and pinning it here
// keeps what is shown honest.
//
// Each src is copied verbatim from the document, so the two can be diffed by
// eye. Keep them that way: a snippet rewritten to suit the test stops being
// evidence about the snippet a reader will copy.

// recipeSetup declares the variables the recipes name. It matches the data
// block at the top of docs/recipes.md and renders nothing itself.
const recipeSetup = `{{ $list := .list }}{{ $str := .str }}{{ $line := .line }}` +
	`{{ $val := .val }}{{ $date := .date }}`

func recipeData() map[string]any {
	return map[string]any{
		"list": []any{"a", "b", "c"},
		"str":  "hello WORLD",
		"line": "text\r\n",
		"val":  42,
		"date": time.Date(2024, 3, 15, 9, 30, 0, 0, time.UTC),
	}
}

// renderRecipe executes one snippet from docs/recipes.md against the documented
// data. The second result reports whether it ran, so a caller can skip its own
// comparison rather than report a second failure for the same recipe.
func renderRecipe(t *testing.T, src string) (string, bool) {
	t.Helper()
	tmpl, err := template.New("recipe").Funcs(FuncMap()).Parse(recipeSetup + src)
	if err != nil {
		t.Errorf("%s: parse: %v", src, err)
		return "", false
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, recipeData()); err != nil {
		t.Errorf("%s: execute: %v", src, err)
		return "", false
	}
	return buf.String(), true
}

func TestRecipes(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"descending sort", `{{ reverse (sort $list) }}`, "[c b a]"},
		{"capitalize first letter", `{{ firstUpper $str }}`, "Hello WORLD"},
		{"capitalize, lowercase rest", `{{ firstUpper (lower $str) }}`, "Hello world"},
		{"last N elements", `{{ take $list -2 }}`, "[b c]"},
		{"all but last N", `{{ drop $list -2 }}`, "[a]"},
		{
			"join with final separator",
			`{{ printf "%s and %s" (join (drop $list -1) ", ") (last $list) }}`,
			"a, b and c",
		},
		{"trailing separator", `{{ printf "%s," (join $list ", ") }}`, "a, b, c,"},
		{"chomp", `{{ trimRight $line "\r\n" }}`, "text"},
		{"toString", `{{ printf "%v" $val }}`, "42"},
		{"date formatting", `{{ $date.Format "2006-01-02" }}`, "2024-03-15"},
		// The two that were documented backwards. %20s right-aligns, padding on
		// the left; %-20s left-aligns, padding on the right. The expected values
		// are written with their spaces rather than built with strings.Repeat so
		// that the alignment is visible here as it is in the document.
		{"pad left", `{{ printf "%20s" $str }}`, "         hello WORLD"},
		{"pad right", `{{ printf "%-20s" $str }}`, "hello WORLD         "},
	}
	for _, c := range cases {
		got, ok := renderRecipe(t, c.src)
		if !ok {
			continue
		}
		if got != c.want {
			t.Errorf("%s:\n  %s\n  got  %q\n  want %q", c.name, c.src, got, c.want)
		}
	}
}

// now.UTC is the one recipe docs/recipes.md shows without an output, because it
// does not have a fixed one. Asserting merely that it rendered something would
// pass on any string at all, so the output is parsed back: what the recipe
// claims is a UTC instant, and a value that round-trips through the layout
// time.Time prints in, carries a +0000 zone, and lands near now is one.
func TestRecipes_nowUTC(t *testing.T) {
	got, ok := renderRecipe(t, `{{ now.UTC }}`)
	if !ok {
		return
	}
	parsed, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", got)
	if err != nil {
		t.Fatalf("{{ now.UTC }} rendered %q, which is not a time.Time: %v", got, err)
	}
	if _, offset := parsed.Zone(); offset != 0 {
		t.Errorf("{{ now.UTC }} rendered %q, which is not UTC (offset %d)", got, offset)
	}
	if d := time.Since(parsed); d < -time.Minute || d > time.Minute {
		t.Errorf("{{ now.UTC }} rendered %q, which is %v away from now", got, d)
	}
}
