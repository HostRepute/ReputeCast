package flex_test

import (
	"encoding/json"
	"testing"

	"github.com/hostrepute/reputecast/internal/providers/flex"
)

func TestInt(t *testing.T) {
	for in, want := range map[string]flex.Int{`10000`: 10000, `"10000"`: 10000, `"10,000"`: 10000, `" 300 "`: 300, `""`: 0, `null`: 0, `1e4`: 10000} {
		var v struct {
			N flex.Int `json:"n"`
		}
		if err := json.Unmarshal([]byte(`{"n":`+in+`}`), &v); err != nil || v.N != want {
			t.Fatalf("%s: %d, %v", in, v.N, err)
		}
	}
	for _, in := range []string{`"lots"`, `1.5`, `true`} {
		var v struct {
			N flex.Int `json:"n"`
		}
		if err := json.Unmarshal([]byte(`{"n":`+in+`}`), &v); err == nil {
			t.Fatalf("%s must not decode", in)
		}
	}
}
