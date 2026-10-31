package renders

import "testing"

// UseEndpoints points the services at fakes for one test: keys are litmus,
// mailgun_inspect/us and mailgun_inspect/eu.
func UseEndpoints(t testing.TB, m map[string]string) {
	t.Helper()
	endpointsMu.Lock()
	old := map[string]string{}
	for k, v := range endpoints {
		old[k] = v
	}
	for k, v := range m {
		endpoints[k] = v
	}
	endpointsMu.Unlock()
	t.Cleanup(func() {
		endpointsMu.Lock()
		endpoints = old
		endpointsMu.Unlock()
	})
}

// LitmusName is litmusName for tests.
var LitmusName = litmusName
