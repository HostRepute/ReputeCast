package providers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// RuntimePrefix starts the registry key of every adapter installed while
// ReputeCast runs: community adapters are "community:<id>".
const RuntimePrefix = "community:"

// runtimeOwners maps each runtime adapter's key to whoever registered it
// (guarded by registryMu).
var runtimeOwners = map[string]any{}

// SetRuntime makes ds the runtime adapters registered by owner. Community
// adapters are installed, updated and removed while ReputeCast runs
// (internal/providers/remote registers them), so each owner swaps its set as
// a whole instead of registering from init: keys of owner missing from ds
// are removed, keys in ds are added or replaced. Keys must start with
// RuntimePrefix, so a runtime adapter can never take the place of a built-in
// one. owner must be comparable (a pointer).
func SetRuntime(owner any, ds []Descriptor) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	seen := map[string]bool{}
	for _, d := range ds {
		switch {
		case !strings.HasPrefix(d.Adapter, RuntimePrefix) || len(d.Adapter) == len(RuntimePrefix):
			return fmt.Errorf("providers: runtime adapter key %q must start with %q", d.Adapter, RuntimePrefix)
		case seen[d.Adapter]:
			return fmt.Errorf("providers: duplicate runtime adapter %s", d.Adapter)
		case d.New == nil:
			return fmt.Errorf("providers: runtime adapter %s has no constructor", d.Adapter)
		}
		if _, exists := registry[d.Adapter]; exists && runtimeOwners[d.Adapter] == nil {
			return fmt.Errorf("providers: %s is a built-in adapter", d.Adapter)
		}
		seen[d.Adapter] = true
	}
	for k, o := range runtimeOwners {
		if o == owner && !seen[k] {
			delete(registry, k)
			delete(runtimeOwners, k)
		}
	}
	for _, d := range ds {
		registry[d.Adapter] = d
		runtimeOwners[d.Adapter] = owner
	}
	return nil
}

// CommunityKey is the registry key of an installed community adapter that
// is known by its name (HTTP mappings): RuntimePrefix and the name in
// lowercase letters and digits joined by dashes, so it can appear in a
// webhook URL ("Example ESP" → "community:example-esp"). Remote adapters
// use their manifest id instead.
func CommunityKey(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			dash = false
		default:
			dash = true
		}
		if b.Len() >= 60 {
			break
		}
	}
	slug := strings.TrimRight(b.String(), "-")
	if slug == "" {
		sum := sha256.Sum256([]byte(name))
		slug = "adapter-" + hex.EncodeToString(sum[:4])
	}
	return RuntimePrefix + slug
}

// RuntimeAdapters returns the keys of the runtime adapters, sorted.
func RuntimeAdapters() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return slices.Sorted(maps.Keys(runtimeOwners))
}
