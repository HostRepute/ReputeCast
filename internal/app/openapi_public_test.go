package app

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/version"
)

// The client SDKs' document (sdk/openapi): what an API key may call, with
// exactly the schemas that references, the binary's version and the
// instance URL as its server, without changing the web app's document.
func TestPublicSpec(t *testing.T) {
	hapi := humachi.New(chi.NewRouter(), NewHumaConfig())
	RegisterOperations(hapi)
	web := hapi.OpenAPI()
	before, err := json.Marshal(web)
	if err != nil {
		t.Fatal(err)
	}

	// The version is the binary's at the time of the call, as a release
	// build stamps it.
	was := version.Version
	version.Version = "v1.2.3"
	t.Cleanup(func() { version.Version = was })

	pub, err := PublicSpec(web)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := json.Marshal(pub)
	if err != nil {
		t.Fatal(err)
	}
	after, err := json.Marshal(web)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("PublicSpec changed the web app's document")
	}

	// The operations: every one an API key may call, except the integration
	// apps' own, each authenticating with a bearer key alone.
	want := map[string]bool{}
	for _, op := range operationsOf(web) {
		_, integration := op.Extensions["x-integration"]
		if hasBearer(op.Security) && !integration {
			want[op.OperationID] = true
		}
	}
	got := map[string]bool{}
	tags := map[string]bool{}
	for _, op := range operationsOf(pub) {
		got[op.OperationID] = true
		if _, ok := op.Extensions["x-integration"]; ok {
			t.Errorf("%s is only for the integration apps (x-integration)", op.OperationID)
		}
		if len(op.Security) != 1 || !hasBearer(op.Security) || len(op.Security[0]) != 1 {
			t.Errorf("%s: security %v, want bearer alone", op.OperationID, op.Security)
		}
		if !strings.HasPrefix(op.Path, "/api/v1/") {
			t.Errorf("%s: path %s lost its /api/v1 prefix", op.OperationID, op.Path)
		}
		for _, tag := range op.Tags {
			tags[tag] = true
		}
	}
	if !maps.Equal(got, want) {
		for id := range want {
			if !got[id] {
				t.Errorf("%s is missing", id)
			}
		}
		for id := range got {
			if !want[id] {
				t.Errorf("%s isn't callable with an API key", id)
			}
		}
	}
	if len(got) < 800 {
		t.Fatalf("only %d operations", len(got))
	}

	// The schemas: every $ref resolves, and every schema is referenced.
	var tree map[string]any
	if err := json.Unmarshal(doc, &tree); err != nil {
		t.Fatal(err)
	}
	schemas := tree["components"].(map[string]any)["schemas"].(map[string]any)
	referenced := map[string]bool{}
	var walk func(v any)
	walk = func(v any) {
		_ = eachRef(v, func(ref string) error {
			name, ok := strings.CutPrefix(ref, schemaRefPrefix)
			if !ok {
				t.Errorf("$ref %s isn't a schema", ref)
				return nil
			}
			if _, ok := schemas[name]; !ok {
				t.Errorf("$ref %s doesn't resolve", ref)
				return nil
			}
			if !referenced[name] {
				referenced[name] = true
				walk(schemas[name])
			}
			return nil
		})
	}
	walk(tree["paths"])
	for name := range schemas {
		if !referenced[name] {
			t.Errorf("schema %s isn't referenced", name)
		}
	}
	if len(schemas) != len(pub.Components.Schemas.Map()) {
		t.Errorf("the registry holds %d schemas, the document %d", len(pub.Components.Schemas.Map()), len(schemas))
	}

	// The document itself.
	if pub.Info.Version != "v1.2.3" || web.Info.Version == "v1.2.3" {
		t.Errorf("info.version %q, want the binary's v1.2.3 (the web document's stays %q)", pub.Info.Version, web.Info.Version)
	}
	if len(pub.Servers) != 1 || pub.Servers[0].URL != "{baseUrl}" || pub.Servers[0].Variables["baseUrl"] == nil ||
		pub.Servers[0].Variables["baseUrl"].Default != PublicBaseURL {
		t.Errorf("servers: %s", mustJSON(t, pub.Servers))
	}
	if s := mustJSON(t, pub.Security); s != `[{"bearer":[]}]` {
		t.Errorf("security: %s", s)
	}
	if s := mustJSON(t, tree["components"].(map[string]any)["securitySchemes"]); !strings.HasPrefix(s, `{"bearer":{`) || strings.Contains(s, "session") {
		t.Errorf("security schemes: %s", s)
	}
	names := make([]string, 0, len(pub.Tags))
	for _, tag := range pub.Tags {
		names = append(names, tag.Name)
	}
	if !slices.IsSorted(names) || !slices.Equal(names, slices.Sorted(maps.Keys(tags))) {
		t.Errorf("tags %v, want the operations' tags sorted", names)
	}
}

func operationsOf(doc *huma.OpenAPI) []*huma.Operation {
	var ops []*huma.Operation
	for _, item := range doc.Paths {
		for _, op := range []*huma.Operation{item.Get, item.Put, item.Post, item.Delete, item.Options, item.Head, item.Patch, item.Trace} {
			if op != nil {
				ops = append(ops, op)
			}
		}
	}
	return ops
}

func hasBearer(security []map[string][]string) bool {
	return slices.ContainsFunc(security, func(s map[string][]string) bool { _, ok := s["bearer"]; return ok })
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
