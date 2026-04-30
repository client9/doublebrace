package doublebrace

import (
	"maps"
	"text/template"
)

// FuncMap returns a template.FuncMap containing all standard functions.
// See the package documentation for the full list.
func FuncMap() template.FuncMap {
	out := stringFuncMap()
	maps.Copy(out, mathFuncMap())
	maps.Copy(out, pathFuncMap())
	maps.Copy(out, safeFuncMap())
	maps.Copy(out, collectionsFuncMap())
	maps.Copy(out, timeFuncMap())
	maps.Copy(out, castFuncMap())
	maps.Copy(out, encodingFuncMap())
	return out
}

// Merge combines multiple template.FuncMaps into one new map.
// Later maps win on key collision, so user-defined functions override defaults:
//
//	fns := funcs.Merge(funcs.FuncMap(), template.FuncMap{"myFunc": myFunc})
func Merge(fms ...template.FuncMap) template.FuncMap {
	out := make(template.FuncMap)
	for _, m := range fms {
		maps.Copy(out, m)
	}
	return out
}
