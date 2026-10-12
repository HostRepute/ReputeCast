package hostrepute

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// HostReputeConnection is a workspace's connection to HostRepute as shown in
// Settings › Integrations and the connect flow. Tokens are never included.
type HostReputeConnection struct {
	Status string `json:"status" enum:"not_connected,connected,revoked,expired,error"`
	Method string `json:"method,omitempty" enum:"oauth,api_key"`
	// Demo is true for connections served by the built-in demo HostRepute.
	Demo         bool       `json:"demo"`
	TeamName     string     `json:"teamName,omitempty"`
	AccountEmail string     `json:"accountEmail,omitempty"`
	Plan         string     `json:"plan,omitempty"`
	ConnectedAt  *time.Time `json:"connectedAt,omitempty"`
	ConnectedBy  string     `json:"connectedBy,omitempty"`
	// TokenRejectedAt is when HostRepute rejected the token: data is stale since.
	TokenRejectedAt *time.Time `json:"tokenRejectedAt,omitempty"`
	LastSyncAt      *time.Time `json:"lastSyncAt,omitempty"`
	// LastCheckAt is HostRepute's last full check of the monitored assets.
	LastCheckAt *time.Time `json:"lastCheckAt,omitempty"`
	// SyncError is set while HostRepute can't be reached ("Couldn't reach
	// api.hostrepute.com (timeout after 30 s, 3 attempts)").
	SyncError   string     `json:"syncError,omitempty"`
	SyncErrorAt *time.Time `json:"syncErrorAt,omitempty"`
	// Syncing is true while a sync (the first one after connecting) runs.
	Syncing              bool              `json:"syncing"`
	Scopes               []HostReputeScope `json:"scopes"`
	MonitorLimit         *int32            `json:"monitorLimit,omitempty"`
	MonitorUsed          int               `json:"monitorUsed"`
	Sources              int32             `json:"sources"`
	CheckIntervalMinutes int32             `json:"checkIntervalMinutes"`
	DMARCRua             string            `json:"dmarcRua,omitempty"`
	// AlertsToNotifications: HostRepute alerts become ReputeCast notifications.
	AlertsToNotifications bool `json:"alertsToNotifications"`
	// Assets are the workspace's domains, IPs and hostnames with their
	// monitoring status (monitored ones first, then suggestions).
	Assets []HostReputeAsset `json:"assets"`
	// Shared is exactly what connecting shares with HostRepute.
	Shared HostReputeShared `json:"shared"`
}

// HostReputeScope is a granted (or requested) OAuth scope.
type HostReputeScope struct {
	Key     string `json:"key"`
	Use     string `json:"use"`
	Granted bool   `json:"granted"`
}

// HostReputeAsset is one row of the monitored assets table.
type HostReputeAsset struct {
	Asset string `json:"asset"`
	Type  string `json:"type" enum:"domain,ip,hostname"`
	Role  string `json:"role" enum:"sending_domain,organization_domain,dedicated_ip,server_hostname,other"`
	// Provider sends from the IP or runs on the hostname ("Postal").
	Provider   string     `json:"provider,omitempty"`
	Monitored  bool       `json:"monitored"`
	Status     string     `json:"status,omitempty" enum:"synced,pending,error"`
	LastSyncAt *time.Time `json:"lastSyncAt,omitempty"`
	Score      *int       `json:"score,omitempty"`
	Verdict    string     `json:"verdict,omitempty"`
	// Sources is how many blocklists check the asset; Listings names the
	// ones it is listed on.
	Sources              int      `json:"sources"`
	Listings             []string `json:"listings"`
	PostmasterReputation string   `json:"postmasterReputation,omitempty"`
	DMARCPolicy          string   `json:"dmarcPolicy,omitempty"`
	DMARCAligned         *float64 `json:"dmarcAligned,omitempty" doc:"Percent of DMARC-aligned mail in the last 7 days"`
}

// HostReputeShared lists what is sent to HostRepute. Contacts, email content
// and reports are never sent.
type HostReputeShared struct {
	Domains []string `json:"domains"`
	IPs     []string `json:"ips"`
}

// IntegrationsOverview is Settings › Integrations.
type IntegrationsOverview struct {
	HostRepute HostReputeConnection `json:"hostRepute"`
	Webhooks   WebhooksSummary      `json:"webhooks"`
	MCP        MCPSummary           `json:"mcp"`
	// Postmaster and SNDS say where that data comes from.
	Postmaster DataSourceSummary `json:"postmaster"`
	SNDS       DataSourceSummary `json:"snds"`
}

// WebhooksSummary counts outgoing webhook endpoints.
type WebhooksSummary struct {
	Endpoints int64 `json:"endpoints"`
	Failing   int64 `json:"failing"`
}

// MCPSummary counts active MCP agents.
type MCPSummary struct {
	Agents int64 `json:"agents"`
}

// DataSourceSummary says where Postmaster or SNDS data comes from.
type DataSourceSummary struct {
	Via string `json:"via" enum:"hostrepute,direct,none"`
}

// SyncQueued is the check run a sync fills in.
type SyncQueued struct {
	RunID     string    `json:"runId"`
	StartedAt time.Time `json:"startedAt"`
}

// Connection returns the connection view (status not_connected when there
// is none, with what connecting would share).
func (s *Service) Connection(ctx context.Context, ws uuid.UUID) (HostReputeConnection, error) {
	out := HostReputeConnection{Status: "not_connected", Scopes: []HostReputeScope{}, Assets: []HostReputeAsset{}}
	cands, err := Candidates(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	out.Shared = HostReputeShared{Domains: []string{}, IPs: []string{}}
	for _, c := range cands {
		if !c.Shared {
			continue
		}
		if c.Type == AssetIP {
			out.Shared.IPs = append(out.Shared.IPs, c.Asset)
		} else {
			out.Shared.Domains = append(out.Shared.Domains, c.Asset)
		}
	}
	conn, err := s.Q.GetHostReputeConnection(ctx, ws)
	if errors.Is(err, pgx.ErrNoRows) {
		for _, k := range Scopes {
			out.Scopes = append(out.Scopes, HostReputeScope{Key: k, Use: ScopeUse[k]})
		}
		for _, c := range cands {
			out.Assets = append(out.Assets, HostReputeAsset{Asset: c.Asset, Type: c.Type, Role: c.Role, Provider: c.Provider, Listings: []string{}})
		}
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Status = conn.Status
	if out.Status == "limit_reached" {
		out.Status = "connected" // the plan's limit only affects adding assets
	}
	out.Method, out.Demo, out.TeamName, out.AccountEmail, out.Plan = conn.Method, conn.ApiUrl == DemoAPIURL, conn.TeamName, conn.AccountEmail, conn.Plan
	ca := conn.ConnectedAt
	out.ConnectedAt = &ca
	if conn.ConnectedBy != nil {
		if u, err := s.Q.UserName(ctx, *conn.ConnectedBy); err == nil {
			out.ConnectedBy = u.Name
			if out.ConnectedBy == "" {
				out.ConnectedBy = u.Email
			}
		}
	}
	out.TokenRejectedAt, out.LastSyncAt, out.LastCheckAt = conn.TokenRejectedAt, conn.LastSyncAt, conn.LastCheckAt
	if conn.LastSyncError != "" {
		out.SyncError, out.SyncErrorAt = conn.LastSyncError, conn.LastSyncAttemptAt
	}
	if run, err := s.Q.RunningCheckRun(ctx, store.RunningCheckRunParams{WorkspaceID: ws, AliveAfter: s.now().Add(-10 * time.Minute)}); err == nil && run.Via == "hostrepute" {
		out.Syncing = true
	}
	for _, k := range Scopes {
		out.Scopes = append(out.Scopes, HostReputeScope{Key: k, Use: ScopeUse[k], Granted: slices.Contains(conn.Scopes, k)})
	}
	for _, k := range conn.Scopes {
		if !slices.Contains(Scopes, k) {
			out.Scopes = append(out.Scopes, HostReputeScope{Key: k, Use: ScopeUse[k], Granted: true})
		}
	}
	out.MonitorLimit, out.Sources, out.CheckIntervalMinutes, out.DMARCRua = conn.MonitorLimit, conn.Sources, conn.CheckIntervalMinutes, conn.DmarcRua
	out.AlertsToNotifications = conn.AlertsToNotifications

	var status map[string]AssetStatus
	if s.Status != nil {
		if status, err = s.Status(ctx, ws); err != nil {
			return out, err
		}
	}
	monitored := parseMonitored(conn.Monitored)
	out.MonitorUsed = len(monitored)
	row := func(name, typ, role, provider string) HostReputeAsset {
		a := HostReputeAsset{Asset: name, Type: typ, Role: role, Provider: provider, Listings: []string{}}
		if st, ok := status[name]; ok {
			a.Sources, a.PostmasterReputation, a.DMARCPolicy, a.DMARCAligned = st.Sources, st.PostmasterReputation, st.DMARCPolicy, st.DMARCAligned
			if st.Listings != nil {
				a.Listings = st.Listings
			}
		}
		return a
	}
	seen := map[string]bool{}
	// Monitored assets in the workspace's own order, then ones only HostRepute knows.
	for _, c := range cands {
		i := slices.IndexFunc(monitored, func(m monitoredAsset) bool { return strings.EqualFold(m.Asset, c.Asset) })
		if i < 0 {
			continue
		}
		m := monitored[i]
		a := row(c.Asset, c.Type, c.Role, c.Provider)
		a.Monitored, a.Status, a.LastSyncAt, a.Score, a.Verdict = true, m.Status, m.LastSyncAt, m.Score, m.Verdict
		out.Assets = append(out.Assets, a)
		seen[strings.ToLower(c.Asset)] = true
	}
	for _, m := range monitored {
		if seen[strings.ToLower(m.Asset)] {
			continue
		}
		typ := m.Type
		if typ == "" {
			typ = GuessType(m.Asset)
		}
		a := row(m.Asset, typ, RoleOther, "")
		a.Monitored, a.Status, a.LastSyncAt, a.Score, a.Verdict = true, m.Status, m.LastSyncAt, m.Score, m.Verdict
		out.Assets = append(out.Assets, a)
		seen[strings.ToLower(m.Asset)] = true
	}
	for _, c := range cands {
		if !seen[strings.ToLower(c.Asset)] {
			out.Assets = append(out.Assets, row(c.Asset, c.Type, c.Role, c.Provider))
		}
	}
	return out, nil
}

// Overview returns Settings › Integrations.
func (s *Service) Overview(ctx context.Context, ws uuid.UUID) (IntegrationsOverview, error) {
	hr, err := s.Connection(ctx, ws)
	if err != nil {
		return IntegrationsOverview{}, err
	}
	out := IntegrationsOverview{HostRepute: hr}
	wh, err := s.Q.CountWorkspaceWebhooks(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Webhooks = WebhooksSummary{Endpoints: wh.Endpoints, Failing: wh.Failing}
	if out.MCP.Agents, err = s.Q.CountWorkspaceAgents(ctx, ws); err != nil {
		return out, err
	}
	via, err := s.Q.DataSourceVia(ctx, ws)
	if err != nil {
		return out, err
	}
	source := func(v string) DataSourceSummary {
		switch {
		case v == "direct":
			return DataSourceSummary{Via: "direct"}
		case v == "hostrepute" && hr.Status != "not_connected":
			return DataSourceSummary{Via: "hostrepute"}
		}
		return DataSourceSummary{Via: "none"}
	}
	out.Postmaster, out.SNDS = source(via.Postmaster), source(via.Snds)
	return out, nil
}
