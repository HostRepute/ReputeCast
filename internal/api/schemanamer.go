package api

import (
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/danielgtaylor/huma/v2"
)

// SchemaNamer names API schemas after their Go type, like Huma's default
// namer, and keeps names unique across domains: when a second, different
// type asks for a name already taken (FieldOption in two packages), it gets
// its package name as a prefix (AutomationsFieldOption) instead of Huma
// panicking at startup. Domains should still pick distinct type names.
func SchemaNamer() func(t reflect.Type, hint string) string {
	var mu sync.Mutex
	byType := map[reflect.Type]string{}
	byName := map[string]reflect.Type{}
	return func(t reflect.Type, hint string) string {
		mu.Lock()
		defer mu.Unlock()
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if name, ok := byType[t]; ok {
			return name
		}
		name := huma.DefaultSchemaNamer(t, hint)
		if other, taken := byName[name]; taken && other != t {
			base := name
			if pkg := t.PkgPath(); pkg != "" {
				p := pkg[strings.LastIndex(pkg, "/")+1:]
				base = strings.ToUpper(p[:1]) + p[1:] + name
			}
			name = base
			for i := 2; byName[name] != nil && byName[name] != t; i++ {
				name = base + strconv.Itoa(i)
			}
		}
		byType[t] = name
		byName[name] = t
		return name
	}
}
