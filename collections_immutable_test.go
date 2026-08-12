package doublebrace

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"text/template"
)

// Collection functions must never mutate or alias their arguments. See the
// Immutability section of the package doc for why: html/template executes
// concurrently over shared data, so in-place mutation is a data race.
//
// These tests guard both halves of that guarantee. Every function is run
// against a []any input — the type that would take a non-copying fast path if
// one were ever reintroduced into toSlice — and the input is checked for
// modification afterward. Functions returning a slice are additionally checked
// for backing-array overlap with the input, since an aliased result lets a
// downstream call corrupt an upstream argument.

// overlaps reports whether two slices share any backing memory.
func overlaps(a, b []any) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for i := range a {
		for j := range b {
			if &a[i] == &b[j] {
				return true
			}
		}
	}
	return false
}

// immutableCases covers every collection function that accepts a slice.
// Inputs deliberately contain adjacent duplicates and out-of-order elements so
// that in-place compaction or sorting would be visible in the input afterward.
var immutableCases = []struct {
	name string
	in   []any
	fn   func(any) (any, error)
}{
	{"First", []any{3, 1, 1, 2}, First},
	{"Last", []any{3, 1, 1, 2}, Last},
	{"Take positive", []any{3, 1, 1, 2}, func(v any) (any, error) { return Take(v, 2) }},
	{"Take negative", []any{3, 1, 1, 2}, func(v any) (any, error) { return Take(v, -2) }},
	{"Take overlong", []any{3, 1, 1, 2}, func(v any) (any, error) { return Take(v, 99) }},
	{"Drop positive", []any{3, 1, 1, 2}, func(v any) (any, error) { return Drop(v, 2) }},
	{"Drop negative", []any{3, 1, 1, 2}, func(v any) (any, error) { return Drop(v, -2) }},
	{"Drop overlong", []any{3, 1, 1, 2}, func(v any) (any, error) { return Drop(v, 99) }},
	{"Reverse", []any{3, 1, 1, 2}, func(v any) (any, error) { return Reverse(v) }},
	{"Compact", []any{1, 1, 2, 3, 3, 1}, func(v any) (any, error) { return Compact(v) }},
	{"Concat", []any{3, 1, 1, 2}, func(v any) (any, error) { return Concat(v, []any{9}) }},
	{"Sort numeric", []any{3, 1, 1, 2}, func(v any) (any, error) { return Sort(v) }},
	{"Sort lexical", []any{"c", "a", "a", "b"}, func(v any) (any, error) { return Sort(v) }},
	{"Sort by key", []any{
		map[string]any{"T": "c"}, map[string]any{"T": "a"},
	}, func(v any) (any, error) { return Sort(v, "T") }},
	{"SortNum", []any{"10", "9", "9", "2"}, func(v any) (any, error) { return SortNum(v) }},
	{"SortNum by key", []any{
		map[string]any{"N": 10}, map[string]any{"N": 2},
	}, func(v any) (any, error) { return SortNum(v, "N") }},
	{"Where", []any{
		map[string]any{"K": 1}, map[string]any{"K": 2}, map[string]any{"K": 1},
	}, func(v any) (any, error) { return Where(v, "K", 1) }},
	{"In", []any{3, 1, 1, 2}, func(v any) (any, error) { return In(v, 1) }},
}

func TestCollectionsDoNotMutateInput(t *testing.T) {
	for _, tc := range immutableCases {
		t.Run(tc.name, func(t *testing.T) {
			before := make([]any, len(tc.in))
			copy(before, tc.in)

			if _, err := tc.fn(tc.in); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tc.in, before) {
				t.Errorf("input mutated: got %v, want %v", tc.in, before)
			}
		})
	}
}

func TestCollectionsDoNotAliasInput(t *testing.T) {
	for _, tc := range immutableCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.fn(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			out, ok := got.([]any)
			if !ok {
				return // scalar result, nothing to alias
			}
			if overlaps(tc.in, out) {
				t.Errorf("result shares backing array with input")
			}
		})
	}
}

// Aliasing is not merely untidy: a result that borrows its argument's memory
// lets a later call in the same pipeline write through to the original. This is
// the concrete failure that motivates returning fresh structures rather than
// only refraining from mutation.
func TestChainedCallsDoNotCorruptInput(t *testing.T) {
	in := []any{1, 1, 2, 9, 9}
	before := []any{1, 1, 2, 9, 9}

	// compact (take $list 3)
	taken, err := Take(in, 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compact(taken); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(in, before) {
		t.Errorf("input corrupted through chained call: got %v, want %v", in, before)
	}
}

// List must copy its variadic slice too, so that spreading a caller's slice
// does not hand back the caller's own memory.
func TestListDoesNotAliasSpreadSlice(t *testing.T) {
	in := []any{1, 2, 3}
	out := List(in...)
	if overlaps(in, out) {
		t.Error("List result shares backing array with spread input")
	}
}

// The guarantee is shallow by design: containers are fresh, elements are
// shared. This documents that boundary so a future change to deep-copy is a
// deliberate decision rather than an accident.
func TestImmutabilityIsShallow(t *testing.T) {
	elem := map[string]any{"K": 1}
	in := []any{elem}

	got, err := Reverse(in)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].(map[string]any)["K"] != 1 {
		t.Fatal("unexpected element value")
	}
	got[0].(map[string]any)["K"] = 2
	if elem["K"] != 2 {
		t.Error("elements are expected to be shared; update this test if deep copying is added")
	}
}

// The immutability guarantee exists so that one template can render over shared
// data from many goroutines at once, which is the shape html/template is built
// for and the shape a web server produces for free. The tests above establish
// that no function mutates its input when called once; this one puts real
// goroutines on the same data so that the race detector has something to
// observe. Without it `go test -race` proves nothing here, however many
// immutability assertions accumulate: the detector reports only races that
// actually occur during a run.
//
// Data lives outside the goroutines and is never copied for them: every
// execution reads the same maps and slices concurrently. A function that sorted
// its argument in place, or returned a result aliasing it, would be reported
// here.
func TestConcurrentTemplateExecution(t *testing.T) {
	pages := []any{
		map[string]any{"Title": "Cherry", "Weight": 3, "Draft": false},
		map[string]any{"Title": "Apple", "Weight": 1, "Draft": false},
		map[string]any{"Title": "Banana", "Weight": 2, "Draft": true},
	}
	site := map[string]any{"Name": "example", "Lang": "en"}
	data := map[string]any{"Pages": pages, "Site": site, "Tags": []any{"b", "a", "a", "c"}}

	tmpl := template.Must(template.New("page").Funcs(FuncMap()).Parse(
		`{{ range sort (where .Pages "Draft" false) "Title" }}{{ .Title }},{{ end }}` +
			`|{{ join (reverse (compact (sort .Tags))) "-" }}` +
			`|{{ join (keys (merge .Site (dict "Extra" 1))) "," }}` +
			`|{{ range sortNum .Pages "Weight" }}{{ .Weight }}{{ end }}`,
	))

	// Rendering once up front both fixes the expected output and proves the
	// template is well-formed, so a failure below is about concurrency rather
	// than a typo in the template above.
	var first strings.Builder
	if err := tmpl.Execute(&first, data); err != nil {
		t.Fatalf("initial execute: %v", err)
	}
	want := first.String()

	const goroutines, iterations = 8, 50
	var wg sync.WaitGroup
	errs := make(chan error, goroutines*iterations)
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				var buf strings.Builder
				if err := tmpl.Execute(&buf, data); err != nil {
					errs <- fmt.Errorf("execute: %w", err)
					continue
				}
				if got := buf.String(); got != want {
					errs <- fmt.Errorf("render mismatch:\n got %q\nwant %q", got, want)
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	// The shared data must be exactly as it started. A mutation that happened to
	// be goroutine-consistent would pass the comparison above but is still a bug.
	if got := pages[0].(map[string]any)["Title"]; got != "Cherry" {
		t.Errorf("input reordered by rendering: pages[0].Title = %v, want Cherry", got)
	}
	if len(site) != 2 {
		t.Errorf("merge modified its argument: site has %d keys, want 2", len(site))
	}
	if got := data["Tags"].([]any); !reflect.DeepEqual(got, []any{"b", "a", "a", "c"}) {
		t.Errorf("sort/compact modified the input slice: %v", got)
	}
}
