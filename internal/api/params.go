package api

import (
	"fmt"
	"reflect"
)

// paramTags are the struct tags Huma reads request parameters from.
var paramTags = []string{"path", "query", "header", "cookie"}

// checkInput panics when a request parameter of an operation's input can't
// reach its handler: Huma sets parameters only through exported fields, so
// one in an unexported field, or in an embedded field of an unexported or
// aliased-to-unexported type, stays empty and the handler answers 404 or
// worse (review 2026-11 CA-11). Every operation is registered at start-up
// and in tests, so this fails CI rather than a request.
func checkInput(t reflect.Type, opID string) {
	if where := hiddenParam(t, t.String(), false); where != "" {
		panic(fmt.Sprintf("api: operation %s: parameter %s is in an unexported field, so its handler never receives it; embed an exported type", opID, where))
	}
}

// hiddenParam returns the first parameter field of t that Huma can't set,
// as "Input.field", or "". hidden says t is reached through an unexported
// embedded field.
func hiddenParam(t reflect.Type, path string, hidden bool) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Name == "Body" && !f.Anonymous {
			continue
		}
		at := path + "." + f.Name
		if f.Anonymous {
			if where := hiddenParam(f.Type, at, hidden || !f.IsExported()); where != "" {
				return where
			}
			continue
		}
		for _, tag := range paramTags {
			if _, ok := f.Tag.Lookup(tag); ok && (hidden || !f.IsExported()) {
				return at
			}
		}
	}
	return ""
}
