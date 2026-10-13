package connect

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// catalogOrder is the design's order of the built-in adapters; adapters
// added later follow, by name.
var catalogOrder = []string{"ses", "postmark", "mailgun", "sendgrid", "resend", "sparkpost", "mailjet", "brevo", "azureacs", "postal", "smtp"}

// ConnectCatalogEntry is one provider card of the catalog.
type ConnectCatalogEntry struct {
	Adapter     string `json:"adapter"`
	Name        string `json:"name"`
	Short       string `json:"short"`
	Kind        string `json:"kind" enum:"api,smtp,self_hosted,community"`
	Recommended bool   `json:"recommended"`
	// Terms is the badge about bulk marketing ("Bulk marketing allowed").
	Terms     string        `json:"terms"`
	TermsTone string        `json:"termsTone" enum:"ok,warn,neutral"`
	Pricing   string        `json:"pricing"`
	Chips     []ConnectChip `json:"chips"`
	// Filters of the catalog.
	BulkAllowed  bool `json:"bulkAllowed"`
	Webhooks     bool `json:"webhooks"`
	DedicatedIPs bool `json:"dedicatedIps"`
	// Connected counts this workspace's connections of the adapter.
	Connected int `json:"connected"`
}

// ConnectCommunityEntry is an installed community adapter.
type ConnectCommunityEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Maintainer string `json:"maintainer"`
	// Summary is "webhooks, suppression pull".
	Summary string `json:"summary"`
	// Adapter is the registry key to connect it with; empty when this
	// build can't run it.
	Adapter string `json:"adapter"`
	// Reason explains why it can't be connected here.
	Reason string `json:"reason,omitempty"`
	// Configure is true for the generic HTTP adapter (set up on the
	// Adapters page, not connected directly).
	Configure bool `json:"configure"`
}

// ConnectOpenSetup is a setup that wasn't finished.
type ConnectOpenSetup struct {
	ID        string    `json:"id"`
	Adapter   string    `json:"adapter"`
	Name      string    `json:"name"`
	Step      string    `json:"step"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ConnectCatalog is step 1 of the setup: every adapter the registry knows.
type ConnectCatalog struct {
	Builtin   []ConnectCatalogEntry   `json:"builtin"`
	Community []ConnectCommunityEntry `json:"community"`
	// Open setups can be resumed.
	Open []ConnectOpenSetup `json:"open"`
	// CanConnect is false without providers:manage.
	CanConnect bool `json:"canConnect"`
}

// Catalog lists every registered adapter (built-in and community) with its
// capability chips, pricing and bulk-marketing terms. Adapters that aren't
// in this build don't appear; installed community adapters do, with the
// reason when this build can't run them.
func (s *Service) Catalog(ctx context.Context, ws *store.Workspace) (ConnectCatalog, error) {
	out := ConnectCatalog{Builtin: []ConnectCatalogEntry{}, Community: []ConnectCommunityEntry{}, Open: []ConnectOpenSetup{}}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	connected := map[string]int{}
	for _, p := range provs {
		connected[p.Adapter]++
	}
	cat := providers.Catalog()
	slices.SortFunc(cat, func(a, b providers.Descriptor) int {
		ia, ib := slices.Index(catalogOrder, a.Adapter), slices.Index(catalogOrder, b.Adapter)
		if ia < 0 {
			ia = len(catalogOrder)
		}
		if ib < 0 {
			ib = len(catalogOrder)
		}
		if ia != ib {
			return ia - ib
		}
		return strings.Compare(a.Name, b.Name)
	})
	httpAdapter := false
	for _, d := range cat {
		switch {
		case d.Adapter == "httpgeneric":
			httpAdapter = true
			continue
		case d.Kind == "community" || strings.HasPrefix(d.Adapter, "community:"):
			continue
		}
		c := copyFor(d)
		terms, tone := termsBadge(d, c)
		e := ConnectCatalogEntry{Adapter: d.Adapter, Name: c.Display, Short: c.Short, Kind: d.Kind, Recommended: c.Recommended, Terms: terms, TermsTone: tone,
			Pricing: c.Pricing, Chips: chips(d, c), BulkAllowed: d.Capabilities.Bulk != providers.BulkForbidden,
			Webhooks: d.Capabilities.Webhooks != "" && d.Capabilities.Webhooks != "Bounce mailbox", DedicatedIPs: d.Capabilities.DedicatedIPs != "" || d.Kind == "self_hosted",
			Connected: connected[d.Adapter]}
		if e.Kind == "" {
			e.Kind = "api"
		}
		out.Builtin = append(out.Builtin, e)
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, name, kind, version, maintainer, manifest, latest_version FROM installed_adapters ORDER BY installed_at, name`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var c ConnectCommunityEntry
		var id uuid.UUID
		var kind string
		var manifest json.RawMessage
		var latest *string
		if err := rows.Scan(&id, &c.Name, &kind, &c.Version, &c.Maintainer, &manifest, &latest); err != nil {
			return out, err
		}
		c.ID = id.String()
		var m struct {
			Capabilities []string  `json:"capabilities"`
			LastRelease  time.Time `json:"last_release"`
		}
		_ = json.Unmarshal(manifest, &m)
		// Adapters without a release for a year are not offered for new
		// connections (the Adapters page flags them as unmaintained).
		if !m.LastRelease.IsZero() && s.now().Sub(m.LastRelease) > 365*24*time.Hour {
			continue
		}
		caps := []string{}
		for _, cp := range m.Capabilities {
			cp = strings.ReplaceAll(cp, "domain setup via API", "domain via API")
			if !strings.HasPrefix(cp, "batch") {
				caps = append(caps, cp)
			}
		}
		c.Summary = c.Maintainer
		if len(caps) > 0 {
			c.Summary += " · " + strings.Join(caps, ", ")
		}
		key := "community:" + c.Name
		switch {
		case providers.Registered(key):
			c.Adapter = key
		case kind == "http" && httpAdapter:
			c.Adapter = "httpgeneric"
		default:
			c.Reason = c.Name + " is installed, but this build can't run community adapters yet. It becomes available with the adapter runtime."
		}
		out.Community = append(out.Community, c)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if httpAdapter {
		out.Community = append(out.Community, ConnectCommunityEntry{ID: "httpgeneric", Name: "Generic HTTP adapter", Version: "built-in",
			Summary: "Map any HTTP API's send call and webhook without code", Configure: true})
	}
	open, err := s.Q.SetupOpen(ctx, store.SetupOpenParams{WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	for _, o := range open {
		name := o.Adapter
		if d, ok := providers.Lookup(o.Adapter); ok {
			name = copyFor(d).Display
		}
		out.Open = append(out.Open, ConnectOpenSetup{ID: o.ID.String(), Adapter: o.Adapter, Name: name, Step: o.Step, UpdatedAt: o.UpdatedAt})
	}
	if p := api.PrincipalFrom(ctx); p != nil {
		out.CanConnect = p.Can(authz.ProvidersManage)
	}
	return out, nil
}
