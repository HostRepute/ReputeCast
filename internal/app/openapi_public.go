package app

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/version"
)

// PublicBaseURL is the default of the public document's baseUrl server
// variable: an SDK is given the instance's own URL.
const PublicBaseURL = "https://reputecast.example.com"

const schemaRefPrefix = "#/components/schemas/"

// PublicSpec returns the document the client SDKs are generated from
// (sdk/openapi): the operations an API key may call, except the ones only
// the integration apps call (x-integration), with only the schemas they
// reference. Its version is the binary's, its server is the instance URL
// (paths keep their /api/v1 prefix), and every operation authenticates
// with a bearer key. spec itself is left as it was: the web app's document
// is generated from it.
func PublicSpec(spec *huma.OpenAPI) (*huma.OpenAPI, error) {
	paths := make(map[string]*huma.PathItem, len(spec.Paths))
	tagSet := map[string]bool{}
	for path, item := range spec.Paths {
		pub := *item
		pub.Get, pub.Put, pub.Post, pub.Delete = publicOperation(item.Get), publicOperation(item.Put), publicOperation(item.Post), publicOperation(item.Delete)
		pub.Options, pub.Head, pub.Patch, pub.Trace = publicOperation(item.Options), publicOperation(item.Head), publicOperation(item.Patch), publicOperation(item.Trace)
		kept := false
		for _, op := range []*huma.Operation{pub.Get, pub.Put, pub.Post, pub.Delete, pub.Options, pub.Head, pub.Patch, pub.Trace} {
			if op == nil {
				continue
			}
			kept = true
			for _, t := range op.Tags {
				tagSet[t] = true
			}
		}
		if kept {
			paths[path] = &pub
		}
	}

	schemas, err := referencedSchemas(paths, spec.Components.Schemas.Map())
	if err != nil {
		return nil, err
	}

	tags := make([]*huma.Tag, 0, len(tagSet))
	for _, name := range slices.Sorted(maps.Keys(tagSet)) {
		tags = append(tags, &huma.Tag{Name: name})
	}

	info := *spec.Info
	info.Version = version.Version
	components := *spec.Components
	components.Schemas = &publicSchemas{Registry: spec.Components.Schemas, schemas: schemas}
	components.SecuritySchemes = map[string]*huma.SecurityScheme{"bearer": spec.Components.SecuritySchemes["bearer"]}

	pub := *spec
	pub.Info = &info
	pub.Paths = paths
	pub.Components = &components
	pub.Servers = []*huma.Server{{
		URL:       "{baseUrl}",
		Variables: map[string]*huma.ServerVariable{"baseUrl": {Default: PublicBaseURL, Description: "The instance's public URL."}},
	}}
	pub.Security = []map[string][]string{{"bearer": {}}}
	pub.Tags = tags
	pub.Webhooks = nil
	return &pub, nil
}

// publicOperation returns a copy of op authenticating with a bearer key
// alone, or nil when an API key can't call op or only the integration apps
// do.
func publicOperation(op *huma.Operation) *huma.Operation {
	if op == nil || op.Hidden {
		return nil
	}
	if _, ok := op.Extensions["x-integration"]; ok {
		return nil
	}
	if !slices.ContainsFunc(op.Security, func(s map[string][]string) bool { _, ok := s["bearer"]; return ok }) {
		return nil
	}
	pub := *op
	pub.Security = []map[string][]string{{"bearer": {}}}
	return &pub
}

// referencedSchemas returns the schemas of all that paths reference,
// directly or through other schemas.
func referencedSchemas(paths map[string]*huma.PathItem, all map[string]*huma.Schema) (map[string]*huma.Schema, error) {
	keep := map[string]*huma.Schema{}
	var queue []string
	add := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		var tree any
		if err := json.Unmarshal(b, &tree); err != nil {
			return err
		}
		return eachRef(tree, func(ref string) error {
			name, ok := strings.CutPrefix(ref, schemaRefPrefix)
			if !ok {
				return fmt.Errorf("openapi: unexpected $ref %q", ref)
			}
			s, ok := all[name]
			if !ok {
				return fmt.Errorf("openapi: $ref %q names no schema", ref)
			}
			if keep[name] == nil {
				keep[name] = s
				queue = append(queue, name)
			}
			return nil
		})
	}
	if err := add(paths); err != nil {
		return nil, err
	}
	for len(queue) > 0 {
		name := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if err := add(keep[name]); err != nil {
			return nil, err
		}
	}
	return keep, nil
}

// eachRef calls visit with every $ref in a decoded JSON value.
func eachRef(v any, visit func(string) error) error {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if ref, ok := val.(string); ok && k == "$ref" {
				if err := visit(ref); err != nil {
					return err
				}
				continue
			}
			if err := eachRef(val, visit); err != nil {
				return err
			}
		}
	case []any:
		for _, val := range t {
			if err := eachRef(val, visit); err != nil {
				return err
			}
		}
	}
	return nil
}

// publicSchemas is the registry of the public document: the server's, of
// which it shows only the referenced schemas.
type publicSchemas struct {
	huma.Registry
	schemas map[string]*huma.Schema
}

func (r *publicSchemas) Map() map[string]*huma.Schema { return r.schemas }

func (r *publicSchemas) SchemaFromRef(ref string) *huma.Schema {
	return r.schemas[strings.TrimPrefix(ref, schemaRefPrefix)]
}

func (r *publicSchemas) MarshalJSON() ([]byte, error) { return json.Marshal(r.schemas) }
