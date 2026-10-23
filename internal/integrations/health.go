package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// Reasons an integration needs attention (brief 3.1).
const (
	ReasonKeyRevoked        = "key_revoked"
	ReasonKeyExpired        = "key_expired"
	ReasonOwnerLeft         = "owner_left"
	ReasonKeyExpiring       = "key_expiring"
	ReasonDeliveriesFailing = "deliveries_failing"
	ReasonCallsFailing      = "calls_failing"
	ReasonSiteSilent        = "site_silent"
)

// Thresholds of the passive health checks.
const (
	expiringWithin  = 7 * 24 * time.Hour
	failingFor      = 30 * time.Minute
	callsWindow     = time.Hour
	callsMinFailing = 4
)

// IntegrationReason is why an integration needs attention, with what the
// sentence about it needs.
type IntegrationReason struct {
	Code string `json:"code" enum:"key_revoked,key_expired,owner_left,key_expiring,deliveries_failing,calls_failing,site_silent"`
	// Since is when it started (revoked, expired, left, failing, last heard from).
	Since *time.Time `json:"since,omitempty"`
	// By names who revoked the key, or the member who left.
	By string `json:"by,omitempty"`
	// Site or endpoint host ("staging.acme.io", "n8n.acme.io").
	Host string `json:"host,omitempty"`
	// ConnectionID is the connection (a WordPress site) it is about.
	ConnectionID string `json:"connectionId,omitempty"`
	// Rejected counts calls refused since; Status and StatusText are the
	// failing endpoint's answer; Retrying and NextAt its queue.
	Rejected   int64      `json:"rejected,omitempty"`
	Status     int32      `json:"status,omitempty"`
	StatusText string     `json:"statusText,omitempty"`
	Retrying   int64      `json:"retrying,omitempty"`
	NextAt     *time.Time `json:"nextAt,omitempty"`
	Calls      int64      `json:"calls,omitempty"`
	Failed     int64      `json:"failed,omitempty"`
	// Check is the last "Check now" of a silent site.
	Check *WordPressCheck `json:"check,omitempty"`
}

// lastCheck reads a site's last "Check now", in a reader's language.
func lastCheck(l i18n.Lang, site *store.WordpressSite) *WordPressCheck {
	if site == nil || site.LastCheckAt == nil || len(site.LastCheck) == 0 {
		return nil
	}
	var c WordPressCheck
	if json.Unmarshal(site.LastCheck, &c) != nil {
		return nil
	}
	c = c.in(l)
	c.CheckedAt = *site.LastCheckAt
	return &c
}

var rejectedCodes = []string{"key_revoked", "key_expired", "owner_deactivated"}

// reasons computes why a connection needs attention from what ReputeCast
// already knows: its key, its managed webhooks, its recent calls and, for a
// site, when it last reported in. Nothing is polled.
func (s *Service) reasons(ctx context.Context, wsID uuid.UUID, conn store.Integration, keys []store.IntegrationKeysRow, hooks []store.IntegrationWebhooksRow, site *store.WordpressSite) []IntegrationReason {
	now := s.now()
	out := []IntegrationReason{}
	cur, _ := current(keys, now)
	if cur != nil {
		switch {
		case cur.RevokedAt != nil && (cur.GraceUntil == nil || !cur.GraceUntil.After(now)):
			r := IntegrationReason{Code: ReasonKeyRevoked, Since: cur.RevokedAt}
			if by, err := s.Q.IntegrationKeyRevokedBy(ctx, store.IntegrationKeyRevokedByParams{WorkspaceID: &wsID, KeyID: cur.ID.String()}); err == nil {
				r.By = by.ActorName
			}
			r.Rejected = s.rejectedSince(ctx, conn.ID, *cur.RevokedAt)
			out = append(out, r)
		case cur.ExpiresAt != nil && !cur.ExpiresAt.After(now):
			out = append(out, IntegrationReason{Code: ReasonKeyExpired, Since: cur.ExpiresAt, Rejected: s.rejectedSince(ctx, conn.ID, *cur.ExpiresAt)})
		case cur.OwnerStatus != "active":
			r := IntegrationReason{Code: ReasonOwnerLeft, By: cur.OwnerName, Since: cur.OwnerLeftAt}
			if r.Since != nil {
				r.Rejected = s.rejectedSince(ctx, conn.ID, *r.Since)
			}
			out = append(out, r)
		case cur.ExpiresAt != nil && cur.ExpiresAt.Sub(now) < expiringWithin:
			out = append(out, IntegrationReason{Code: ReasonKeyExpiring, Since: cur.ExpiresAt})
		}
	}
	for _, h := range hooks {
		if h.FailingSince == nil || now.Sub(*h.FailingSince) < failingFor || h.Demo && h.Retrying == 0 && h.Dead > 0 {
			continue
		}
		r := IntegrationReason{Code: ReasonDeliveriesFailing, Since: h.FailingSince, Host: hostOf(h.Url), Retrying: h.Retrying, NextAt: asTime(h.NextAttemptAt)}
		if h.LastStatus != nil {
			r.Status = *h.LastStatus
			r.StatusText = http.StatusText(int(*h.LastStatus))
		}
		out = append(out, r)
		break
	}
	if hc, err := s.Q.IntegrationCallsHealth(ctx, store.IntegrationCallsHealthParams{IntegrationID: conn.ID, Since: now.Add(-callsWindow)}); err == nil &&
		hc.Calls >= callsMinFailing && hc.Failed*2 > hc.Calls {
		since := hc.FirstFailed
		out = append(out, IntegrationReason{Code: ReasonCallsFailing, Since: &since, Calls: hc.Calls, Failed: hc.Failed})
	}
	if site != nil && site.LastReportAt != nil && now.Sub(*site.LastReportAt) >= SiteSilentAfter {
		out = append(out, IntegrationReason{Code: ReasonSiteSilent, Since: site.LastReportAt, Host: siteBareHost(firstNonEmpty(site.SiteUrl, site.Address)),
			ConnectionID: conn.ID.String(), Check: lastCheck(lang(ctx), site)})
	}
	return out
}

// asTime reads a nullable timestamp sqlc couldn't type.
func asTime(v any) *time.Time {
	if t, ok := v.(time.Time); ok {
		return &t
	}
	return nil
}

func (s *Service) rejectedSince(ctx context.Context, id uuid.UUID, since time.Time) int64 {
	rows, err := s.Q.IntegrationCallsRejected(ctx, store.IntegrationCallsRejectedParams{IntegrationID: id, Since: since, Codes: rejectedCodes})
	if err != nil {
		return 0
	}
	return int64(len(rows))
}

// HealthArgs is the periodic check that notices integrations needing
// attention (a site going silent, a key expiring) and tells the members
// who can fix them.
type healthReport struct {
	conn    store.Integration
	reasons []IntegrationReason
}

// CheckHealth recomputes every connection's reasons and notifies members
// with integrations:manage about new ones, once per reason.
func (s *Service) CheckHealth(ctx context.Context) error {
	conns, err := s.Q.IntegrationsAllActive(ctx)
	if err != nil {
		return err
	}
	for _, c := range conns {
		keys, err := s.Q.IntegrationKeys(ctx, &c.ID)
		if err != nil {
			return err
		}
		hooks, err := s.Q.IntegrationWebhooks(ctx, c.ID)
		if err != nil {
			return err
		}
		var site *store.WordpressSite
		if c.Kind == KindWordPress {
			if st, err := s.Q.WordPressSite(ctx, c.ID); err == nil {
				site = &st
			}
		}
		rs := s.reasons(ctx, c.WorkspaceID, c, keys, hooks, site)
		if err := s.noteAttention(ctx, healthReport{conn: c, reasons: rs}); err != nil {
			s.log().WarnContext(ctx, "integration health", "integration", c.ID, "err", err)
		}
	}
	return nil
}

// noteAttention stores a connection's reasons and notifies new ones.
func (s *Service) noteAttention(ctx context.Context, h healthReport) error {
	codes := []string{}
	for _, r := range h.reasons {
		codes = append(codes, r.Code)
	}
	attention, _ := json.Marshal(h.reasons)
	var fresh []IntegrationReason
	for _, r := range h.reasons {
		if !slices.Contains(h.conn.Notified, r.Code) {
			fresh = append(fresh, r)
		}
	}
	ws, err := s.Q.GetWorkspace(ctx, h.conn.WorkspaceID)
	if err != nil {
		return err
	}
	for _, r := range fresh {
		if err := notify.ToMembersWith(ctx, s.Pool, h.conn.WorkspaceID, authz.IntegrationsManage, notify.Notification{
			Kind: notify.KindIntegrationAttention, Severity: "warn",
			TitleText: msgs.M("notify.attentionTitle").Keep(nil), BodyText: attentionText(h.conn, r, wsLocation(&ws)),
			Link: integrationLink(ws.Slug, h.conn),
			Data: map[string]any{"integrationId": h.conn.ID.String(), "kind": h.conn.Kind, "reason": r.Code},
		}); err != nil {
			return err
		}
	}
	return s.Q.IntegrationSetAttention(ctx, store.IntegrationSetAttentionParams{ID: h.conn.ID, Attention: attention, Notified: codes})
}

func integrationLink(slug string, c store.Integration) string {
	if c.Kind == KindWordPress {
		return "/w/" + slug + "/settings/integrations/wordpress"
	}
	return "/w/" + slug + "/settings/integrations/" + c.Kind
}

// attentionText is the notification's sentence about a reason, kept as a
// message so each member reads it in their own language, with its times in
// the workspace's zone.
func attentionText(c store.Integration, r IntegrationReason, loc *time.Location) i18n.Text {
	args := i18n.Args{"name": Name(c.Kind)}
	if r.Since != nil {
		args["at"] = r.Since.In(loc)
	}
	key := "notify.attention"
	switch r.Code {
	case ReasonKeyRevoked:
		key = "notify.keyRevoked"
	case ReasonKeyExpired:
		key = "notify.keyExpired"
	case ReasonOwnerLeft:
		key, args["owner"] = "notify.ownerLeft", r.By
	case ReasonKeyExpiring:
		key = "notify.keyExpiring"
	case ReasonDeliveriesFailing:
		key, args["host"], args["status"], args["statusText"] = "notify.deliveriesFailing", r.Host, strconv.Itoa(int(r.Status)), r.StatusText
	case ReasonCallsFailing:
		key, args["failed"], args["calls"] = "notify.callsFailing", r.Failed, r.Calls
	case ReasonSiteSilent:
		key, args["host"] = "notify.siteSilent", r.Host
	}
	return msgs.M(key).Keep(args)
}
