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

// Jsonify marshals v to a JSON string. Useful for embedding data in
// <script> blocks or data attributes.
//
//	jsonify (dict "name" "Alice" "age" 30) → {"age":30,"name":"Alice"}
func Jsonify(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("jsonify: %w", err)
	}
	return string(b), nil
}
