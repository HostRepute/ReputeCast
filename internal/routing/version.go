package routing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// streamVersions returns each stream's routing version: a short hash of its
// rules and failover policy as stored. Any writer changes it (this screen,
// the warm-up planner raising a share, an agent), so a save that names the
// version it started from is refused when someone changed that stream
// since, and a save of one stream never trips over a change to another.
// A stream with no rules and no policy has version "0".
func streamVersions(rules []store.RoutingRule, pols []store.StreamPolicy) map[providers.Stream]string {
	type ruleKey struct {
		Priority   int32           `json:"p"`
		Name       string          `json:"n"`
		Conditions json.RawMessage `json:"c"`
		Splits     json.RawMessage `json:"s"`
		Fallback   string          `json:"f"`
		Enabled    bool            `json:"e"`
	}
	type streamKey struct {
		Rules    []ruleKey `json:"r"`
		Failover string    `json:"f"`
		AckAt    string    `json:"a"`
		Priority bool      `json:"p"`
	}
	keys := map[providers.Stream]*streamKey{}
	get := func(s string) *streamKey {
		st := providers.Stream(s)
		if keys[st] == nil {
			keys[st] = &streamKey{}
		}
		return keys[st]
	}
	for _, r := range rules {
		k := ruleKey{Priority: r.Priority, Name: r.Name, Conditions: compact(r.Conditions), Splits: compact(r.Splits), Enabled: r.Enabled}
		if r.FallbackProviderID != nil {
			k.Fallback = r.FallbackProviderID.String()
		}
		sk := get(r.Stream)
		sk.Rules = append(sk.Rules, k)
	}
	for _, p := range pols {
		sk := get(p.Stream)
		sk.Failover, sk.Priority = p.Failover, p.Priority
		if p.BulkTermsAckAt != nil {
			sk.AckAt = p.BulkTermsAckAt.UTC().Format(time.RFC3339Nano)
		}
	}
	out := map[providers.Stream]string{}
	for _, st := range Streams {
		sk := keys[st]
		if sk == nil {
			out[st] = "0"
			continue
		}
		// Rule ids change on every save; the content decides.
		slices.SortFunc(sk.Rules, func(a, b ruleKey) int {
			if a.Priority != b.Priority {
				return int(a.Priority - b.Priority)
			}
			ja, _ := json.Marshal(a)
			jb, _ := json.Marshal(b)
			return strings.Compare(string(ja), string(jb))
		})
		raw, _ := json.Marshal(sk)
		sum := sha256.Sum256(raw)
		out[st] = hex.EncodeToString(sum[:6])
	}
	return out
}

// compact drops insignificant whitespace so the same JSON hashes the same.
func compact(raw json.RawMessage) json.RawMessage {
	var v any
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return raw
	}
	out, _ := json.Marshal(v)
	return out
}
