package app

import (
	"testing"

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
