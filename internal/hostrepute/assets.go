package hostrepute

import (
	"context"
	"encoding/json"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/store"
)

// Asset roles: why an asset belongs to the workspace.
const (
	RoleSendingDomain      = "sending_domain"
	RoleOrganizationDomain = "organization_domain"
	RoleDedicatedIP        = "dedicated_ip"
	RoleServerHostname     = "server_hostname"
	RoleOther              = "other"
)

// Candidate is a domain, IP or hostname of the workspace that HostRepute can
// monitor. Only candidates are ever sent to HostRepute.
type Candidate struct {
	Asset string
	Type  string // domain, ip, hostname
	Role  string
	// Provider sends from the IP or runs on the hostname ("Postal").
	ProviderID *uuid.UUID
	Provider   string
	// Shared is true for what connecting shares by default: sending and
	// organization domains and dedicated IPs. Server hostnames are suggested.
	Shared bool
}

// Candidates lists the workspace's assets: sending domains, their
// organization domains, dedicated IPs and self-hosted server hostnames.
func Candidates(ctx context.Context, q *store.Queries, workspaceID uuid.UUID) ([]Candidate, error) {
	domains, err := q.HostReputeAssetSources(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	providers, err := q.HostReputeActiveProviders(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	var out []Candidate
	seen := map[string]bool{}
	add := func(c Candidate) {
		if c.Asset == "" || seen[c.Asset] {
			return
		}
		seen[c.Asset] = true
		out = append(out, c)
	}
	for _, d := range domains {
		add(Candidate{Asset: d.Domain, Type: AssetDomain, Role: RoleSendingDomain, Shared: true})
	}
	for _, d := range domains {
		if org := dnscheck.OrgDomain(d.Domain); org != d.Domain {
			add(Candidate{Asset: org, Type: AssetDomain, Role: RoleOrganizationDomain, Shared: true})
		}
	}
	for _, p := range providers {
		if p.IpMode != "dedicated" {
			continue
		}
		for _, ip := range p.Ips {
			if a, err := netip.ParseAddr(ip); err == nil {
				id := p.ID
				add(Candidate{Asset: a.String(), Type: AssetIP, Role: RoleDedicatedIP, ProviderID: &id, Provider: p.Name, Shared: true})
			}
		}
	}
	for _, p := range providers {
		if h := ServerHostname(p.Adapter, p.Config); h != "" {
			id := p.ID
			add(Candidate{Asset: h, Type: AssetHostname, Role: RoleServerHostname, ProviderID: &id, Provider: p.Name})
		}
	}
	return out, nil
}

// ServerHostname is the hostname of a self-hosted sending server (Postal's
// server URL, an SMTP relay's host); empty for API providers.
func ServerHostname(adapter string, config json.RawMessage) string {
	var c struct {
		ServerURL string `json:"server_url"`
		Host      string `json:"host"`
	}
	_ = json.Unmarshal(config, &c)
	switch adapter {
	case "postal":
		if u, err := url.Parse(c.ServerURL); err == nil {
			return strings.ToLower(u.Hostname())
		}
	case "smtp":
		return strings.ToLower(strings.TrimSpace(c.Host))
	}
	return ""
}

// monitoredAsset is one entry of hostrepute_connections.monitored.
type monitoredAsset struct {
	ID            string     `json:"id"`
	Asset         string     `json:"asset"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	Score         *int       `json:"score,omitempty"`
	Verdict       string     `json:"verdict,omitempty"`
	Listings      int        `json:"listings"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	LastSyncAt    *time.Time `json:"last_sync_at,omitempty"`
}

func parseMonitored(raw json.RawMessage) []monitoredAsset {
	var out []monitoredAsset
	_ = json.Unmarshal(raw, &out)
	return out
}

func monitoredJSON(assets []Asset, syncedAt time.Time) json.RawMessage {
	list := make([]monitoredAsset, 0, len(assets))
	for _, a := range assets {
		t := syncedAt
		list = append(list, monitoredAsset{ID: a.ID, Asset: a.Asset, Type: a.Type, Status: a.Status, Score: a.Score, Verdict: a.Verdict,
			Listings: a.Listings, LastCheckedAt: a.LastCheckedAt, LastSyncAt: &t})
	}
	b, _ := json.Marshal(list)
	return b
}

func monitoredNames(raw json.RawMessage) []string {
	var out []string
	for _, m := range parseMonitored(raw) {
		out = append(out, m.Asset)
	}
	return out
}

func findCandidate(cands []Candidate, asset string) (Candidate, bool) {
	i := slices.IndexFunc(cands, func(c Candidate) bool { return strings.EqualFold(c.Asset, asset) })
	if i < 0 {
		return Candidate{}, false
	}
	return cands[i], true
}
