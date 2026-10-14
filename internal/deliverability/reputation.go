package deliverability

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/store"
)

// DeliverabilityReputation is the "Sender reputation" input of Send Readiness (spec 5.1):
// blocklist listings of the workspace's domains and IPs.
type DeliverabilityReputation struct {
	// Status: pass (clean), warn (low or medium impact listing), block
	// (high-impact listing) or unknown (no current data: HostRepute rejected
	// the token or can't be reached and no fallback ran). Unknown is never
	// treated as a Block.
	Status    string                            `json:"status" enum:"pass,warn,block,unknown"`
	Via       string                            `json:"via" enum:"hostrepute,builtin,none"`
	Sources   int                               `json:"sources"`
	Listings  []DeliverabilityReputationListing `json:"listings"`
	CheckedAt *time.Time                        `json:"checkedAt,omitempty"`
	// Reason explains an unknown status.
	Reason string `json:"reason,omitempty"`
}

type DeliverabilityReputationListing struct {
	Asset     string     `json:"asset"`
	Source    string     `json:"source"`
	Impact    string     `json:"impact"`
	Since     *time.Time `json:"since,omitempty"`
	DelistURL string     `json:"delistUrl,omitempty"`
}

// Reputation returns the blocklist verdict for Send Readiness.
func (s *Service) Reputation(ctx context.Context, ws *store.Workspace) (DeliverabilityReputation, error) {
	st, err := s.load(ctx, ws)
	if err != nil {
		return DeliverabilityReputation{}, err
	}
	out := DeliverabilityReputation{Status: "pass", Listings: []DeliverabilityReputationListing{}}
	via := st.blocklistVia()
	if st.hostReputeStale() {
		// Prefer fresh built-in results over stale HostRepute ones.
		if st.lastBuiltin != nil && st.conn.LastSyncAt != nil && st.lastBuiltin.FinishedAt != nil && st.lastBuiltin.FinishedAt.After(*st.conn.LastSyncAt) {
			via = "builtin"
		} else {
			out.Status, out.Via = "unknown", via
			out.Reason = "HostRepute data can't be refreshed: reconnect HostRepute or wait for the built-in check."
			if st.conn.TokenRejectedAt != nil {
				out.Reason = "HostRepute rejected the token on " + st.conn.TokenRejectedAt.In(st.loc()).Format("Jan 2") + "; reconnect to resume blocklist checks."
			}
			return out, nil
		}
	}
	out.Via = via
	out.Sources = st.sourcesFor(via, len(s.lists()))
	if via == "none" {
		out.Status, out.Reason = "unknown", "No blocklist check has run yet."
		return out, nil
	}
	for _, b := range st.blocklists {
		if b.Via != via {
			continue
		}
		if out.CheckedAt == nil || b.CheckedAt.After(*out.CheckedAt) {
			t := b.CheckedAt
			out.CheckedAt = &t
		}
		if !b.Listed {
			continue
		}
		out.Listings = append(out.Listings, DeliverabilityReputationListing{Asset: b.Asset, Source: b.Source, Impact: b.Impact, Since: b.ListedSince, DelistURL: b.DelistUrl})
		switch {
		case b.Impact == "high":
			out.Status = "block"
		case out.Status == "pass":
			out.Status = "warn"
		}
	}
	return out, nil
}

// AssetStatuses is what Deliverability knows per asset, for Settings ›
// Integrations (it has the signature hostrepute.Service.Status expects).
func (s *Service) AssetStatuses(ctx context.Context, ws uuid.UUID) (map[string]hostrepute.AssetStatus, error) {
	w, err := s.Q.GetWorkspace(ctx, ws)
	if err != nil {
		return nil, err
	}
	st, err := s.load(ctx, &w)
	if err != nil {
		return nil, err
	}
	out := map[string]hostrepute.AssetStatus{}
	for _, b := range st.blocklists {
		if b.Via != "hostrepute" {
			continue
		}
		a := out[b.Asset]
		a.Sources++
		if b.Listed {
			a.Listings = append(a.Listings, b.Source)
		}
		out[b.Asset] = a
	}
	for _, c := range st.cands {
		if c.Type != hostrepute.AssetDomain {
			continue
		}
		a := out[c.Asset]
		a.PostmasterReputation = st.latestPostmaster(c.Asset).DomainReputation
		if chk, ok := st.checks[strings.ToLower(c.Asset)]; ok {
			var r struct {
				Auth  *dnscheck.DomainAuth `json:"auth"`
				DMARC *dnscheck.DMARC      `json:"dmarc"`
			}
			if json.Unmarshal(chk.Result, &r) == nil {
				switch {
				case r.DMARC != nil && r.DMARC.Status == dnscheck.StatusPass:
					a.DMARCPolicy = r.DMARC.Policy
				case r.Auth != nil && r.Auth.DMARC.Status == dnscheck.StatusPass:
					a.DMARCPolicy = r.Auth.DMARC.Policy
				}
			}
		}
		a.DMARCAligned = st.dmarcAligned(c.Asset)
		out[c.Asset] = a
	}
	return out, nil
}

// DeliverabilityDelistingRequest is the result of "Request delisting".
type DeliverabilityDelistingRequest struct {
	Asset    string                           `json:"asset"`
	Requests []DeliverabilityDelistingOutcome `json:"requests"`
}

// DeliverabilityDelistingOutcome is one source's delisting request.
type DeliverabilityDelistingOutcome struct {
	Source      string    `json:"source"`
	Status      string    `json:"status" enum:"requested,pending,delisted,denied"`
	RequestedAt time.Time `json:"requestedAt"`
	// Already is true when a request was already pending.
	Already bool   `json:"already"`
	URL     string `json:"url,omitempty"`
}

// RequestDelisting asks HostRepute to request delisting of an asset from
// every list it's on (or one source). Without HostRepute, delisting is done
// on each list's own page.
func (s *Service) RequestDelisting(ctx context.Context, ws *store.Workspace, asset, source string) (DeliverabilityDelistingRequest, error) {
	asset = strings.ToLower(strings.TrimSpace(asset))
	rows, err := s.Q.ListDeliverabilityBlocklistResults(ctx, ws.ID)
	if err != nil {
		return DeliverabilityDelistingRequest{}, err
	}
	var listed []store.BlocklistResult
	for _, b := range rows {
		if b.Listed && b.Asset == asset && (source == "" || strings.EqualFold(b.Source, source)) {
			listed = append(listed, b)
		}
	}
	if len(listed) == 0 {
		return DeliverabilityDelistingRequest{}, api.Invalid("body.asset", asset+" isn't listed on any blocklist we check.")
	}
	conn, err := s.Q.GetHostReputeConnection(ctx, ws.ID)
	if err != nil || (conn.Status != "connected" && conn.Status != "limit_reached") {
		return DeliverabilityDelistingRequest{}, api.Err(http.StatusConflict, "hostrepute_required",
			"Delisting requests go through HostRepute. Request it on the list's own page instead: "+listed[0].DelistUrl).WithMeta("delistUrl", listed[0].DelistUrl)
	}
	sort.Slice(listed, func(i, j int) bool { return impactRank(listed[i].Impact) < impactRank(listed[j].Impact) })
	out := DeliverabilityDelistingRequest{Asset: asset, Requests: []DeliverabilityDelistingOutcome{}}
	for _, b := range listed {
		if b.Via != "hostrepute" {
			continue
		}
		if (b.Delisting == "requested" || b.Delisting == "pending") && b.DelistingRequestedAt != nil {
			out.Requests = append(out.Requests, DeliverabilityDelistingOutcome{Source: b.Source, Status: b.Delisting, RequestedAt: *b.DelistingRequestedAt, Already: true, URL: b.DelistUrl})
			continue
		}
		d, err := s.HostRepute.RequestDelisting(ctx, ws.ID, asset, b.Source)
		if err != nil {
			return out, err
		}
		status := d.Status
		if status != "pending" && status != "delisted" && status != "denied" {
			status = "requested"
		}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.SetDeliverabilityBlocklistDelisting(ctx, store.SetDeliverabilityBlocklistDelistingParams{WorkspaceID: ws.ID, Via: "hostrepute", Asset: asset, Source: b.Source,
				Delisting: status, RequestedAt: &d.RequestedAt}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "deliverability.delisting_request", TargetType: "blocklist_listing",
				TargetID: asset + "@" + b.Source, TargetName: asset + " on " + b.Source, After: map[string]any{"status": status}})
		})
		if err != nil {
			return out, err
		}
		out.Requests = append(out.Requests, DeliverabilityDelistingOutcome{Source: b.Source, Status: status, RequestedAt: d.RequestedAt, URL: firstNonEmpty(d.URL, b.DelistUrl)})
	}
	return out, nil
}
