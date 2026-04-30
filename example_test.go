package doublebrace_test

import (
	"fmt"
	"html/template"
	"time"

	"github.com/client9/doublebrace"
)

// ---- math ----

func ExampleAdd() {
	v, _ := doublebrace.Add(3, 4)
	fmt.Println(v)
	// Output:
	// 7
}

func ExampleSub() {
	v, _ := doublebrace.Sub(10, 3)
	fmt.Println(v)
	// Output:
	// 7
}

func ExampleMul() {
	v, _ := doublebrace.Mul(3, 4)
	fmt.Println(v)
	// Output:
	// 12
}

func ExampleDiv() {
	v, _ := doublebrace.Div(10, 4)
	fmt.Println(v)
	// Output:
	// 2.5
}

func ExampleMod() {
	v, _ := doublebrace.Mod(10, 3)
	fmt.Println(v)
	// Output:
	// 1
}

func ExampleAbs() {
	v, _ := doublebrace.Abs(-7)
	fmt.Println(v)
	// Output:
	// 7
}

func ExampleCeil() {
	v, _ := doublebrace.Ceil(1.2)
	fmt.Println(v)
	// Output:
	// 2
}

func ExampleFloor() {
	v, _ := doublebrace.Floor(1.9)
	fmt.Println(v)
	// Output:
	// 1
}

func ExampleRound() {
	lo, _ := doublebrace.Round(1.4)
	hi, _ := doublebrace.Round(1.5)
	fmt.Println(lo, hi)
	// Output:
	// 1 2
}

func ExampleClamp() {
	within, _ := doublebrace.Clamp(5, 1, 10)
	below, _ := doublebrace.Clamp(0, 1, 10)
	above, _ := doublebrace.Clamp(15, 1, 10)
	fmt.Println(within, below, above)
	// Output:
	// 5 1 10
}

func ExamplePow() {
	v, _ := doublebrace.Pow(2, 10)
	fmt.Println(v)
	// Output:
	// 1024
}

func ExamplePow_sqrt() {
	v, _ := doublebrace.Pow(9, 0.5)
	fmt.Println(v)
	// Output:
	// 3
}

func ExampleModBool() {
	even, _ := doublebrace.ModBool(4, 2)
	odd, _ := doublebrace.ModBool(5, 2)
	fmt.Println(even, odd)
	// Output:
	// true false
}

func ExampleMin() {
	v, _ := doublebrace.Min(3, 1, 4, 1, 5, 9)
	fmt.Println(v)
	// Output:
	// 1
}

func ExampleMin_slice() {
	v, _ := doublebrace.Min([]int{7, 2, 8})
	fmt.Println(v)
	// Output:
	// 2
}

func ExampleMax() {
	v, _ := doublebrace.Max(3, 1, 4, 1, 5, 9)
	fmt.Println(v)
	// Output:
	// 9
}

// ---- collections: constructors ----

func ExampleList() {
	s := doublebrace.List("a", "b", "c")
	fmt.Println(s)
	// Output:
	// [a b c]
}

func ExampleDict() {
	m, _ := doublebrace.Dict("name", "Alice", "age", 30)
	fmt.Println(m["name"], m["age"])
	// Output:
	// Alice 30
}

func ExampleSeq() {
	s, _ := doublebrace.Seq(5)
	fmt.Println(s)
	// Output:
	// [1 2 3 4 5]
}

func ExampleSeq_range() {
	s, _ := doublebrace.Seq(3, 7)
	fmt.Println(s)
	// Output:
	// [3 4 5 6 7]
}

func ExampleSeq_step() {
	s, _ := doublebrace.Seq(1, 9, 2)
	fmt.Println(s)
	// Output:
	// [1 3 5 7 9]
}

// ---- collections: sequence access ----

func ExampleFirst() {
	v, _ := doublebrace.First([]string{"a", "b", "c"})
	fmt.Println(v)
	// Output:
	// a
}

func ExampleFirst_string() {
	v, _ := doublebrace.First("café")
	fmt.Println(v)
	// Output:
	// c
}

func ExampleLast() {
	v, _ := doublebrace.Last([]string{"a", "b", "c"})
	fmt.Println(v)
	// Output:
	// c
}

func ExampleLast_string() {
	v, _ := doublebrace.Last("café")
	fmt.Println(v)
	// Output:
	// é
}

func ExampleTake() {
	v, _ := doublebrace.Take([]int{1, 2, 3, 4, 5}, 3)
	fmt.Println(v)
	// Output:
	// [1 2 3]
}

func ExampleTake_negative() {
	v, _ := doublebrace.Take([]int{1, 2, 3, 4, 5}, -2)
	fmt.Println(v)
	// Output:
	// [4 5]
}

func ExampleTake_string() {
	v, _ := doublebrace.Take("日本語", 2)
	fmt.Println(v)
	// Output:
	// 日本
}

func ExampleDrop() {
	v, _ := doublebrace.Drop([]int{1, 2, 3, 4, 5}, 2)
	fmt.Println(v)
	// Output:
	// [3 4 5]
}

func ExampleDrop_negative() {
	v, _ := doublebrace.Drop([]int{1, 2, 3, 4, 5}, -2)
	fmt.Println(v)
	// Output:
	// [1 2 3]
}

func ExampleDrop_string() {
	v, _ := doublebrace.Drop("hello", 2)
	fmt.Println(v)
	// Output:
	// llo
}

// ---- collections: sequence transformation ----

func ExampleReverse() {
	v, _ := doublebrace.Reverse([]int{1, 2, 3})
	fmt.Println(v)
	// Output:
	// [3 2 1]
}

func ExampleCompact() {
	v, _ := doublebrace.Compact([]any{1, 1, 2, 3, 3, 1})
	fmt.Println(v)
	// Output:
	// [1 2 3 1]
}

func ExampleConcat() {
	v, _ := doublebrace.Concat([]int{1, 2}, []int{3, 4})
	fmt.Println(v)
	// Output:
	// [1 2 3 4]
}

func ExampleSort() {
	v, _ := doublebrace.Sort([]string{"banana", "apple", "cherry"})
	fmt.Println(v)
	// Output:
	// [apple banana cherry]
}

func ExampleSort_numeric() {
	v, _ := doublebrace.Sort([]int{10, 2, 30, 5})
	fmt.Println(v)
	// Output:
	// [2 5 10 30]
}

func ExampleSort_time() {
	t1 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	v, _ := doublebrace.Sort([]time.Time{t1, t2})
	fmt.Println(v.([]any)[0].(time.Time).Format("2006-01-02"))
	// Output:
	// 2023-12-31
}

func ExampleSortNum() {
	v, _ := doublebrace.SortNum([]string{"10", "9", "2"})
	fmt.Println(v)
	// Output:
	// [2 9 10]
}

func ExampleWhere() {
	pages := []any{
		map[string]any{"Title": "Post A", "Draft": false},
		map[string]any{"Title": "Post B", "Draft": true},
		map[string]any{"Title": "Post C", "Draft": false},
	}
	v, _ := doublebrace.Where(pages, "Draft", false)
	fmt.Println(len(v.([]any)))
	// Output:
	// 2
}

// ---- collections: map operations ----

func ExampleKeys() {
	m := map[string]any{"b": 2, "a": 1, "c": 3}
	k, _ := doublebrace.Keys(m)
	fmt.Println(k)
	// Output:
	// [a b c]
}

func ExampleValues() {
	m := map[string]any{"b": 2, "a": 1}
	v, _ := doublebrace.Values(m)
	fmt.Println(v)
	// Output:
	// [1 2]
}

func ExampleMergeMaps() {
	a := map[string]any{"x": 1, "y": 2}
	b := map[string]any{"y": 99, "z": 3}
	m, _ := doublebrace.MergeMaps(a, b)
	fmt.Println(m["x"], m["y"], m["z"])
	// Output:
	// 1 99 3
}

// ---- collections: general ----

func ExampleIn_slice() {
	ok, _ := doublebrace.In([]string{"a", "b", "c"}, "b")
	fmt.Println(ok)
	// Output:
	// true
}

func ExampleIn_map() {
	ok, _ := doublebrace.In(map[string]any{"x": 1}, "x")
	fmt.Println(ok)
	// Output:
	// true
}

func ExampleIn_string() {
	ok, _ := doublebrace.In("hello world", "world")
	fmt.Println(ok)
	// Output:
	// true
}

func ExampleDefault() {
	fmt.Println(doublebrace.Default("anon", ""))
	fmt.Println(doublebrace.Default("anon", "Alice"))
	// Output:
	// anon
	// Alice
}

func ExampleCond() {
	fmt.Println(doublebrace.Cond(true, "yes", "no"))
	fmt.Println(doublebrace.Cond(false, "yes", "no"))
	fmt.Println(doublebrace.Cond(0, "yes", "no"))
	// Output:
	// yes
	// no
	// no
}

// ---- path ----

func ExamplePathBase() {
	fmt.Println(doublebrace.PathBase("/a/b/c.html"))
	fmt.Println(doublebrace.PathBase("/a/b/"))
	// Output:
	// c.html
	// b
}

func ExamplePathDir() {
	fmt.Println(doublebrace.PathDir("/a/b/c.html"))
	// Output:
	// /a/b
}

func ExamplePathExt() {
	fmt.Println(doublebrace.PathExt("index.html"))
	fmt.Println(doublebrace.PathExt("Makefile"))
	// Output:
	// .html
	//
}

func ExamplePathJoin() {
	fmt.Println(doublebrace.PathJoin("/a", "b", "c.html"))
	// Output:
	// /a/b/c.html
}

func ExamplePathClean() {
	fmt.Println(doublebrace.PathClean("/a/b/../c"))
	fmt.Println(doublebrace.PathClean("a//b"))
	// Output:
	// /a/c
	// a/b
}

// ---- safe types ----

func ExampleSafeHTML() {
	v, _ := doublebrace.SafeHTML("<b>bold</b>")
	fmt.Println(v)
	// Output:
	// <b>bold</b>
}

func ExampleSafeCSS() {
	v, _ := doublebrace.SafeCSS("color: red")
	fmt.Println(v)
	// Output:
	// color: red
}

func ExampleSafeURL() {
	v, _ := doublebrace.SafeURL("https://example.com/path?q=1")
	fmt.Println(v)
	// Output:
	// https://example.com/path?q=1
}

func ExampleSafeJS() {
	v, _ := doublebrace.SafeJS("alert('hi')")
	fmt.Println(v)
	// Output:
	// alert('hi')
}

func ExampleURLEncode() {
	fmt.Println(doublebrace.URLEncode("hello world"))
	fmt.Println(doublebrace.URLEncode("a=1&b=2"))
	// Output:
	// hello+world
	// a%3D1%26b%3D2
}

func ExampleURLPathEscape() {
	fmt.Println(doublebrace.URLPathEscape("hello world"))
	fmt.Println(doublebrace.URLPathEscape("a/b"))
	// Output:
	// hello%20world
	// a%2Fb
}

func ExampleSafeHTMLAttr() {
	v, _ := doublebrace.SafeHTMLAttr(`class="hero"`)
	fmt.Println(v)
	// Output:
	// class="hero"
}

func ExampleSafeJSStr() {
	v, _ := doublebrace.SafeJSStr(`hello\nworld`)
	fmt.Println(v)
	// Output:
	// hello\nworld
}

// ---- strings ----

func ExampleFirstUpper() {
	fmt.Println(doublebrace.FirstUpper("go"))
	fmt.Println(doublebrace.FirstUpper("hello world"))
	fmt.Println(doublebrace.FirstUpper("élan"))
	// Output:
	// Go
	// Hello world
	// Élan
}

func ExampleTruncate() {
	fmt.Println(doublebrace.Truncate("hello world", 8))
	fmt.Println(doublebrace.Truncate("hi", 8))
	// Output:
	// hello w…
	// hi
}

func ExampleLenRunes() {
	fmt.Println(doublebrace.LenRunes("café"))
	fmt.Println(doublebrace.LenRunes("日本語"))
	// Output:
	// 4
	// 3
}

func ExampleReplace() {
	fmt.Println(doublebrace.Replace("aabbaa", "a", "x"))
	// Output:
	// xabbaa
}

func ExampleReplace_count() {
	fmt.Println(doublebrace.Replace("aabbaa", "a", "x", -1))
	// Output:
	// xxbbxx
}

// ---- encoding ----

func ExampleJsonify() {
	v, _ := doublebrace.Jsonify(map[string]any{"name": "Alice", "age": 30})
	fmt.Println(v)
	// Output:
	// {"age":30,"name":"Alice"}
}

func ExampleJsonify_slice() {
	v, _ := doublebrace.Jsonify([]string{"a", "b", "c"})
	fmt.Println(v)
	// Output:
	// ["a","b","c"]
}

// ---- cast ----

func ExampleToInt() {
	v, _ := doublebrace.ToInt("42")
	fmt.Println(v)
	// Output:
	// 42
}

func ExampleToInt_float() {
	v, _ := doublebrace.ToInt(3.9)
	fmt.Println(v)
	// Output:
	// 3
}

func ExampleToFloat() {
	v, _ := doublebrace.ToFloat("3.14")
	fmt.Println(v)
	// Output:
	// 3.14
}

// ---- time ----

func ExampleNow() {
	t := doublebrace.Now()
	_ = t
	// Returns current local time as time.Time.
	// In templates: {{now.Year}}, {{now.Format "2006-01-02"}}
}

func ExampleParseTime() {
	t, _ := doublebrace.ParseTime("2006-01-02", "2024-03-15")
	fmt.Println(t.Format("January 2, 2006"))
	// Output:
	// March 15, 2024
}

// ---- FuncMap / Merge ----

func ExampleFuncMap() {
	fm := doublebrace.FuncMap()
	t := template.Must(template.New("").Funcs(fm).Parse(`{{upper "hello"}}`))
	_ = t.Execute(nil, nil)
	// (FuncMap registers all template functions; use with template.New().Funcs())
}

func ExampleMerge() {
	custom := template.FuncMap{
		"greet": func(name string) string { return "Hello, " + name + "!" },
	}
	fm := doublebrace.Merge(doublebrace.FuncMap(), custom)
	t := template.Must(template.New("").Funcs(fm).Parse(`{{greet "World"}}`))
	_ = t.Execute(nil, nil)
	// (Merge combines FuncMaps; later maps win on key collision)
}
