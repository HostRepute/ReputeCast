package state

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

func TestVet(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	sender := map[string]string{id.String(): "verified"}
	health := func(h Health) []byte { b, _ := json.Marshal(h); return b }
	caps := func(c providers.Capabilities) []byte { b, _ := json.Marshal(Snapshot{Capabilities: c}); return b }
	ses := func(mod func(*store.Provider)) store.Provider {
		p := store.Provider{ID: id, Adapter: "ses", Name: "Amazon SES", Status: Healthy,
			Capabilities: caps(providers.Capabilities{Bulk: providers.BulkAllowed, DomainAPI: "full", QuotaDiscovery: true}),
			Health:       health(Health{Quota: &Quota{Kind: "rolling_24h", Max: 50000, Used: 20000}})}
		if mod != nil {
			mod(&p)
		}
		return p
	}
	result := func(v Vetting, key string) VetCheck {
		for _, c := range v.Checks {
			if c.Key == key {
				return c
			}
		}
		return VetCheck{}
	}
	now := time.Now()

	ok := Vet(ses(nil), Load{Broadcast: true, Messages: 18000, Sender: sender})
	if !ok.Automatic(false) || !ok.Allowed() || ok.Warming() || len(ok.Checks) != 6 {
		t.Fatalf("a healthy, verified provider with room: %+v", ok)
	}
	if q := result(ok, VetQuota); *q.Left != 30000 || *q.Need != 18000 {
		t.Fatalf("quota numbers: %+v", q)
	}
	// Transactional mail skips the bulk terms.
	if v := Vet(ses(nil), Load{Messages: 1, Sender: sender}); len(v.Checks) != 5 || result(v, VetBulkTerms).Key != "" {
		t.Fatalf("transactional: %+v", v)
	}

	for name, c := range map[string]struct {
		p      store.Provider
		load   Load
		key    string
		result string
		detail string
	}{
		"down":         {ses(func(p *store.Provider) { p.Status = Down }), Load{Sender: sender}, VetAvailable, VetFail, Down},
		"disconnected": {ses(func(p *store.Provider) { p.DisconnectedAt = &now }), Load{Sender: sender}, VetAvailable, VetFail, "disconnected"},
		"no bulk": {ses(func(p *store.Provider) {
			p.Capabilities = caps(providers.Capabilities{Bulk: providers.BulkForbidden, DomainAPI: "full"})
		}), Load{Broadcast: true, Sender: sender}, VetBulkTerms, VetFail, "forbidden"},
		"sandbox status":  {ses(func(p *store.Provider) { p.Status = Sandbox }), Load{Sender: sender}, VetSandbox, VetFail, ""},
		"sandbox health":  {ses(func(p *store.Provider) { p.Health = health(Health{Sandbox: true}) }), Load{Sender: sender}, VetSandbox, VetFail, ""},
		"domain pending":  {ses(nil), Load{Sender: map[string]string{id.String(): "pending"}}, VetDomain, VetFail, "pending"},
		"domain missing":  {ses(nil), Load{Sender: map[string]string{}}, VetDomain, VetFail, "not_set_up"},
		"no sender":       {ses(nil), Load{}, VetDomain, VetUnknown, "no_sender"},
		"over quota":      {ses(nil), Load{Messages: 30001, Sender: sender}, VetQuota, VetFail, "rolling_24h"},
		"quota not read":  {ses(func(p *store.Provider) { p.Health = nil }), Load{Sender: sender}, VetQuota, VetUnknown, "not_read"},
		"warming":         {ses(nil), Load{Messages: 500, Sender: sender, Warmup: &WarmupToday{Cap: 2000, Sent: 1200}}, VetWarm, VetWarn, ""},
		"over warm cap":   {ses(nil), Load{Messages: 900, Sender: sender, Warmup: &WarmupToday{Cap: 2000, Sent: 1200}}, VetWarm, VetFail, ""},
		"warm-up paused":  {ses(nil), Load{Messages: 1, Sender: sender, Warmup: &WarmupToday{Cap: 2000, Paused: true}}, VetWarm, VetFail, "paused"},
		"warming no plan": {ses(func(p *store.Provider) { p.Status = WarmingUp }), Load{Sender: sender}, VetWarm, VetUnknown, ""},
	} {
		v := Vet(c.p, c.load)
		got := result(v, c.key)
		if got.Result != c.result || got.Detail != c.detail {
			t.Errorf("%s: %s = %+v, want %s (%s)", name, c.key, got, c.result, c.detail)
		}
		if v.Automatic(false) {
			t.Errorf("%s: moves automatically", name)
		}
		// A policy that accepts warming providers takes one whose cap
		// covers the mail, nothing else.
		if v.Automatic(true) != (c.key == VetWarm && c.result == VetWarn) {
			t.Errorf("%s: automatic when warming is accepted = %v", name, v.Automatic(true))
		}
		if v.Warming() != (c.key == VetWarm) {
			t.Errorf("%s: warming = %v", name, v.Warming())
		}
		if v.Allowed() != (c.result != VetFail) {
			t.Errorf("%s: allowed = %v", name, v.Allowed())
		}
	}

	// A provider without a domain API (SMTP) can't confirm the sender, and
	// one without quota discovery sets no quota.
	smtp := store.Provider{ID: id, Adapter: "smtp", Name: "Relay", Status: Healthy, Capabilities: caps(providers.Capabilities{Bulk: providers.BulkYourPolicy})}
	v := Vet(smtp, Load{Broadcast: true, Messages: 10, Sender: map[string]string{}})
	if d := result(v, VetDomain); d.Result != VetUnknown || d.Detail != "not_set_up" {
		t.Fatalf("smtp domain: %+v", d)
	}
	if q := result(v, VetQuota); q.Result != VetPass || q.Left != nil {
		t.Fatalf("smtp quota: %+v", q)
	}
	if !v.Allowed() || v.Automatic(true) {
		t.Fatalf("smtp: allowed %v, automatic %v", v.Allowed(), v.Automatic(true))
	}

	// Each finding reads as a sentence in both languages; passes have none.
	en, vi := i18n.Lang{Locale: i18n.English}, i18n.Lang{Locale: "vi"}
	over := result(Vet(ses(nil), Load{Messages: 31000, Sender: sender}), VetQuota)
	if got := over.In(en, "SES"); got != "SES has 30,000 messages left in its quota; 31,000 would move." {
		t.Fatalf("quota text: %q", got)
	}
	if got := over.In(vi, "SES"); !strings.Contains(got, "SES còn") || !strings.Contains(got, "31.000") {
		t.Fatalf("quota text (vi): %q", got)
	}
	if got := result(ok, VetDomain).In(en, "SES"); got != "" {
		t.Fatalf("a passed check has a sentence: %q", got)
	}
	for _, c := range Vet(ses(func(p *store.Provider) { p.Status = Sandbox }), Load{Broadcast: true, Messages: 60000, Warmup: &WarmupToday{Cap: 10}}).Checks {
		if c.Result == VetPass {
			continue
		}
		if got := c.In(en, "SES"); got == "" || strings.Contains(got, "{") {
			t.Errorf("%s: %q", c.Key, got)
		}
		if got := c.In(vi, "SES"); got == "" || strings.Contains(got, "{") {
			t.Errorf("%s (vi): %q", c.Key, got)
		}
	}
}
