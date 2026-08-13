package doublebrace

import (
	"encoding/json"
	"fmt"
	"text/template"
)

func encodingFuncMap() template.FuncMap {
	return template.FuncMap{
		"jsonify": Jsonify,
	}
}

// Jsonify marshals v to a JSON string.
//
//	jsonify (dict "name" "Alice" "age" 30) → {"age":30,"name":"Alice"}
//
// In text/template, this is how a value becomes JSON: {{ . }} renders a map in
// Go's map[k:v] notation, which is not JSON.
//
// In html/template the useful place is a data attribute, for the same reason.
// The JSON is HTML-escaped on the way out, which is what makes it safe there,
// and JSON.parse recovers it exactly:
//
//	<div data-page="{{ jsonify $page }}">
//
// Inside <script>, do not use it. html/template already marshals data to JSON
// in script context, so the bare action is what emits an object, while jsonify
// returns a plain string and a plain string is escaped into a JavaScript string
// literal:
//
//	<script>var page = {{ $page }};</script>          // var page = {"n":1};
//	<script>var page = {{ jsonify $page }};</script>  // var page = "{\"n\":1}";
//
// For a value that is already encoded JSON rather than data to encode, safeJS
// is the way in. That bypasses contextual escaping, and is sound here only
// because encoding/json escapes <, > and & by default: a "</script>" in the
// data marshals to a < escape and cannot close the element early.
func Jsonify(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("jsonify: %w", err)
	}
	return string(b), nil
}
