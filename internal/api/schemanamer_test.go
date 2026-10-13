package api

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

// Cookie collides by name with net/http.Cookie.
type Cookie struct{ Name string }

func TestSchemaNamerKeepsNamesUnique(t *testing.T) {
	r := huma.NewMapRegistry("#/components/schemas/", SchemaNamer())
	first := r.Schema(reflect.TypeFor[http.Cookie](), true, "")
	second := r.Schema(reflect.TypeFor[Cookie](), true, "")
	again := r.Schema(reflect.TypeFor[*Cookie](), true, "")
	if first.Ref != "#/components/schemas/Cookie" {
		t.Fatalf("first type keeps its name, got %q", first.Ref)
	}
	if second.Ref != "#/components/schemas/ApiCookie" {
		t.Fatalf("second type is prefixed with its package, got %q", second.Ref)
	}
	if again.Ref != second.Ref {
		t.Fatalf("the same type always gets the same name, got %q and %q", again.Ref, second.Ref)
	}
}
