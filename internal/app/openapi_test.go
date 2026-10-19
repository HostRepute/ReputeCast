package app

import (
	"regexp"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// The OpenAPI command registers every operation on an App without config or
// database; wiring that reads them must not panic.
func TestRegisterOperationsWithoutConfig(t *testing.T) {
	hapi := humachi.New(chi.NewRouter(), NewHumaConfig())
	RegisterOperations(hapi)
	if n := len(hapi.OpenAPI().Paths); n < 100 {
		t.Fatalf("only %d paths registered", n)
	}
}

var pathParam = regexp.MustCompile(`\{([^}]+)\}`)

// Every {name} in an operation's path is one of its path parameters. Huma
// skips embedded fields of unexported types, so an input that embeds one
// silently loses its path parameters and the handler reads empty values.
func TestOperationsDeclareTheirPathParameters(t *testing.T) {
	hapi := humachi.New(chi.NewRouter(), NewHumaConfig())
	RegisterOperations(hapi)
	for path, item := range hapi.OpenAPI().Paths {
		for _, op := range []*huma.Operation{item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
			if op == nil {
				continue
			}
			declared := map[string]bool{}
			for _, p := range op.Parameters {
				if p.In == "path" {
					declared[p.Name] = true
				}
			}
			for _, m := range pathParam.FindAllStringSubmatch(path, -1) {
				if !declared[m[1]] {
					t.Errorf("%s %s (%s): {%s} isn't a path parameter of its input", op.Method, path, op.OperationID, m[1])
				}
			}
		}
	}
}
