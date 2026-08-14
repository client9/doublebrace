//go:build go1.24

package doublebrace

import "testing"

// Ranging over an integer is a text/template feature, not a doublebrace one, and
// it arrived after the Go version declared in go.mod. The build tag is what lets
// the recipe be pinned like every other one without breaking make test on the
// floor the module claims to support — the CI matrix builds that floor, so an
// untagged entry in the main table would fail there.
//
// The alternative was to leave the recipe unverified everywhere because one
// supported toolchain cannot run it, which gets the trade backwards: it is
// checked on every version that has the feature, and skipped only where it
// could not work.
func TestRecipes_rangeOverInt(t *testing.T) {
	got, ok := renderRecipe(t, `{{ range 10 }}{{ . }}{{ end }}`)
	if !ok {
		return
	}
	if want := "0123456789"; got != want {
		t.Errorf("{{ range 10 }}{{ . }}{{ end }}: got %q, want %q", got, want)
	}
}
