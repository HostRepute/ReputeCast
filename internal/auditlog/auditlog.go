// Package auditlog serves the audit log (A5 §3.10, §5.14): filtered event
// list, event detail with before/after values, streamed CSV/JSON exports,
// streaming to a webhook and retention. Entries are immutable.
package auditlog

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	// AllowPrivate reports whether endpoints may resolve to private
	// networks (webhooks.allow_private_networks); read at each use.
	AllowPrivate func() bool
	Now          func() time.Time
	// HTTP overrides the stream client (tests).
	HTTP *http.Client
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// AuditActor is who acted.
type AuditActor struct {
	Type        string `json:"type" enum:"user,api_key,mcp_agent,system"`
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	AvatarColor string `json:"avatarColor,omitempty"`
	// Label is "member", "API key", "integration key", "MCP agent" or "automatic".
	Label string `json:"label"`
	// Role is the member's current role in the workspace ("Admin").
	Role string `json:"role,omitempty"`
}

// AuditSource is where the action came from: "web · 203.0.113.9",
// "mcp · rc_agent_7a…c2", "guard · threshold".
type AuditSource struct {
	Channel string `json:"channel"`
	Detail  string `json:"detail,omitempty"`
}

// AuditEventItem is one row of the event list.
type AuditEventItem struct {
	ID         string     `json:"id"`
	OccurredAt time.Time  `json:"occurredAt"`
	Actor      AuditActor `json:"actor"`
	Action     string     `json:"action"`
	TargetType string     `json:"targetType"`
	TargetID   string     `json:"targetId"`
	TargetName string     `json:"targetName"`
	// Target is the line under the action ("Routing rule · Broadcasts").
	Target string `json:"target"`
	// Summary is the details line ("SES 95% / Postal 5% → SES 90% / Postal 10%").
	Summary   string      `json:"summary"`
	Source    AuditSource `json:"source"`
	Result    string      `json:"result" enum:"allowed,denied,error"`
	Reason    string      `json:"reason,omitempty"`
	RequestID string      `json:"requestId,omitempty"`
	// Changes tells that the entry changed an existing object: it has both a
	// before and an after to compare.
	Changes bool `json:"changes"`
}

// AuditEventList is one page of the log.
type AuditEventList struct {
	Items      []AuditEventItem `json:"items"`
	NextCursor string           `json:"nextCursor,omitempty"`
	// Total is the number of entries kept for the workspace.
	Total         int64             `json:"total"`
	RetentionDays int32             `json:"retentionDays"`
	Stream        AuditStreamStatus `json:"stream"`
	Actors        []AuditActorRef   `json:"actors"`
	ActionGroups  []string          `json:"actionGroups"`
	// Categories are the choices of the action filter.
	Categories []AuditCategory `json:"categories"`
}

// AuditActorRef is an actor seen in the log, for the actor filter.
type AuditActorRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AuditCategory is one choice of the action filter: the action groups (the
// part before the dot) it covers.
type AuditCategory struct {
	Key    string   `json:"key"`
	Label  string   `json:"label"`
	Groups []string `json:"groups"`
}

// Categories are the action filter's choices, in the order of the board.
var Categories = []AuditCategory{
	{Key: "sending", Label: "Sending", Groups: []string{"campaign", "approval", "send", "report", "template", "automation"}},
	{Key: "routing", Label: "Routing & providers", Groups: []string{"routing", "provider", "warmup", "deliverability", "domain", "sender"}},
	{Key: "audience", Label: "Audience & consent", Groups: []string{"contact", "contacts", "subscriber", "subscribers", "list", "segment", "form", "consent", "suppression", "import", "growth"}},
	{Key: "team", Label: "Team & access", Groups: []string{"team", "member", "role", "apikey", "agent", "auth", "invitation", "user", "session", "sso"}},
	{Key: "settings", Label: "Settings", Groups: []string{"workspace", "webhook", "audit", "settings", "mcp", "instance"}},
	{Key: "integrations", Label: "Integrations", Groups: []string{"integration", "wordpress", "hostrepute"}},
	{Key: "transactional", Label: "Transactional", Groups: []string{"transactional", "smtp_relay"}},
	{Key: "scoring", Label: "Scoring", Groups: []string{"scoring"}},
	// Recurring digests (design expansion, phase 3).
	{Key: "digests", Label: "Digests", Groups: []string{"digest"}},
	{Key: "polls", Label: "Polls & surveys", Groups: []string{"poll", "survey"}},
	// The AI assistant's connection, key, cap, prices and features (phase 3).
	{Key: "ai", Label: "AI", Groups: []string{"ai"}},
	// Earn (design expansion, phase 6).
	{Key: "earn", Label: "Earn", Groups: []string{"payments", "earn", "paid_member", "sponsors"}},
	{Key: "verification", Label: "Verification", Groups: []string{"verification"}},
	{Key: "placement", Label: "Placement tests", Groups: []string{"placement"}},
	// The publication website (design expansion, phase 7).
	{Key: "website", Label: "Website", Groups: []string{"website", "post", "tag", "page"}},
	// Grow (design expansion, phase 5): referral programs, recommendations
	// and partner links.
	{Key: "grow", Label: "Grow", Groups: []string{"referral", "recommendation", "partner"}},
	// Digital products and their orders (design expansion, phase 7).
	{Key: "products", Label: "Products", Groups: []string{"product", "order"}},
	// Shopify and WooCommerce (design expansion, phase 7).
	{Key: "commerce", Label: "Commerce", Groups: []string{"store"}},
	// SMS (design expansion, phase 8): providers, senders, registrations,
	// routes and rules, the short-link domain, suppressions, the terms page
	// and test numbers. Text campaigns are campaigns (Sending).
	{Key: "sms", Label: "SMS", Groups: []string{"sms_provider", "sms_sender", "sms_registration", "sms_routes", "sms_rules", "sms_short_domain",
		"sms_suppression", "sms_terms", "sms_test_number", "sms"}},
	// Web push and WhatsApp (design expansion, phase 8, X22).
	{Key: "push", Label: "Push", Groups: []string{"push_keys", "push_site", "push_prompt", "push_settings", "push_subscription"}},
	{Key: "whatsapp", Label: "WhatsApp", Groups: []string{"whatsapp", "whatsapp_templates", "whatsapp_rates", "whatsapp_settings"}},
	// Agency accounts (design expansion, phase 8).
	{Key: "agency", Label: "Agency", Groups: []string{"agency"}},
}

// Filter narrows the log.
type Filter struct {
	Search    string
	ActorType string
	ActorID   string
	Action    string // a group ("routing") or an action ("routing.updated")
	Category  string // one of Categories ("sending"); Action wins when both are set
	Result    string
	Range     string // 24h, 7d, 30d, 90d, all
	Since     *time.Time
	Until     *time.Time
	Cursor    string
	Limit     int
}

func (f Filter) params(l i18n.Lang, wsID uuid.UUID, now time.Time) (store.AuditLogListParams, error) {
	p := store.AuditLogListParams{WorkspaceID: &wsID, Lim: int32(f.Limit)}
	if p.Lim <= 0 {
		p.Lim = 50
	}
	if v := strings.TrimSpace(f.Search); v != "" {
		p.Search = &v
	}
	if f.ActorType != "" {
		p.ActorType = &f.ActorType
	}
	if f.ActorID != "" {
		id, err := uuid.Parse(f.ActorID)
		if err != nil {
			return p, api.Invalid("query.actor", say(l, "problem.actor", nil))
		}
		p.ActorID = &id
	}
	if f.Action != "" {
		p.Actions = []string{f.Action}
	} else if f.Category != "" {
		for _, c := range Categories {
			if c.Key == f.Category {
				p.Actions = c.Groups
			}
		}
	}
	if f.Result != "" {
		p.Result = &f.Result
	}
	since := f.Since
	if since == nil {
		var d time.Duration
		switch f.Range {
		case "24h":
			d = 24 * time.Hour
		case "", "7d":
			d = 7 * 24 * time.Hour
		case "30d":
			d = 30 * 24 * time.Hour
		case "90d":
			d = 90 * 24 * time.Hour
		}
		if d > 0 {
			t := now.Add(-d)
			since = &t
		}
	}
	p.Since = since
	p.Until = f.Until
	if f.Cursor != "" {
		at, id, err := decodeCursor(f.Cursor)
		if err != nil {
			return p, api.Invalid("query.cursor", say(l, "problem.cursor", nil))
		}
		p.CursorAt, p.CursorID = &at, &id
	}
	return p, nil
}

func encodeCursor(at time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id.String()))
}

func decodeCursor(c string) (time.Time, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	at, id, ok := strings.Cut(string(b), "|")
	if !ok {
		return time.Time{}, uuid.Nil, fmt.Errorf("bad cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	u, err := uuid.Parse(id)
	return t, u, err
}

type meta struct {
	Summary  string `json:"summary"`
	Detail   string `json:"detail"`
	Source   string `json:"source"`
	Via      string `json:"via"`
	What     string `json:"what"`
	Target   string `json:"target"`
	Related  string `json:"related"`
	Perm     string `json:"permission"`
	Tool     string `json:"tool"`
	Note     string `json:"note"`
	LinkPath string `json:"link"`
	// ActorLabel replaces the actor type's label ("integration key").
	ActorLabel string `json:"actor_label"`
}

func parseMeta(raw json.RawMessage) meta {
	var m meta
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	return m
}

func source(actorType string, ip string, m meta) AuditSource {
	if m.Source != "" {
		return AuditSource{Channel: m.Source, Detail: m.Via}
	}
	switch actorType {
	case "user":
		return AuditSource{Channel: "web", Detail: ip}
	case "api_key":
		return AuditSource{Channel: "api", Detail: ip}
	case "mcp_agent":
		if m.Via != "" {
			return AuditSource{Channel: "mcp", Detail: m.Via}
		}
		return AuditSource{Channel: "mcp", Detail: ip}
	}
	return AuditSource{Channel: "system", Detail: m.Via}
}

// targetLine is the line under the action: the target the entry kept, its
// name, or what it is ("audit log", in the reader's language).
func targetLine(l i18n.Lang, targetType, targetName string, m meta, kept audit.Kept) string {
	if m.Target != "" {
		return audit.In(kept.Target, l, m.Target)
	}
	if targetName != "" {
		return targetName
	}
	if l.Locale == i18n.English {
		return strings.ReplaceAll(targetType, "_", " ")
	}
	return typeIn(l, targetType)
}

// item is an entry of the list, its words in a language.
func item(l i18n.Lang, id uuid.UUID, at time.Time, actorType string, actorID *uuid.UUID, actorName, color, role, action, targetType, targetID, targetName, result, reason string, ip string, requestID string, rawMeta json.RawMessage, before, after json.RawMessage) AuditEventItem {
	m := parseMeta(rawMeta)
	kept := audit.KeptIn(rawMeta)
	it := AuditEventItem{
		ID: id.String(), OccurredAt: at, Action: action, TargetType: targetType, TargetID: targetID, TargetName: targetName,
		Target: targetLine(l, targetType, targetName, m, kept), Result: result, Reason: reasonIn(l, reason), RequestID: requestID,
		Actor:  AuditActor{Type: actorType, Name: actorName, AvatarColor: color, Label: actorLabelIn(l, m.ActorLabel, actorType), Role: role},
		Source: source(actorType, ip, m),
	}
	if actorID != nil {
		it.Actor.ID = actorID.String()
	}
	// Automatic entries are ReputeCast's own: "System", stored or not.
	if it.Actor.Name == "" || actorType == "system" && it.Actor.Name == "System" {
		it.Actor.Name = say(l, "actor.system", nil)
	}
	switch {
	case m.Summary != "":
		it.Summary = kept.SummaryIn(l, m.Summary)
	case m.Detail != "":
		it.Summary = audit.In(kept.Detail, l, m.Detail)
	case reason != "":
		it.Summary = reasonIn(l, reason)
	default:
		it.Summary = diffSummary(before, after)
	}
	return it
}

// List returns one page of the workspace log.
func (s *Service) List(ctx context.Context, ws *store.Workspace, f Filter) (AuditEventList, error) {
	l := i18n.LangFrom(ctx)
	p, err := f.params(l, ws.ID, s.now())
	if err != nil {
		return AuditEventList{}, err
	}
	p.Lim++ // one extra row tells whether there is a next page
	rows, err := s.Q.AuditLogList(ctx, p)
	if err != nil {
		return AuditEventList{}, err
	}
	out := AuditEventList{Items: []AuditEventItem{}, RetentionDays: ws.AuditRetentionDays, Actors: []AuditActorRef{}, ActionGroups: []string{}, Categories: categoriesIn(l)}
	limit := int(p.Lim) - 1
	for i, r := range rows {
		if i == limit {
			last := rows[i-1]
			out.NextCursor = encodeCursor(last.OccurredAt, last.ID)
			break
		}
		it := item(l, r.ID, r.OccurredAt, r.ActorType, r.ActorID, r.ActorName, r.ActorColor, r.ActorRole, r.Action, r.TargetType, r.TargetID, r.TargetName, r.Result, r.Reason, ipString(r.Ip), r.RequestID, r.Metadata, nil, nil)
		it.Changes = r.HasChanges
		out.Items = append(out.Items, it)
	}
	if out.Total, err = s.Q.AuditLogCount(ctx, &ws.ID); err != nil {
		return AuditEventList{}, err
	}
	actors, err := s.Q.AuditLogActors(ctx, &ws.ID)
	if err != nil {
		return AuditEventList{}, err
	}
	for _, a := range actors {
		if a.ActorID == nil {
			continue
		}
		out.Actors = append(out.Actors, AuditActorRef{Type: a.ActorType, ID: a.ActorID.String(), Name: a.ActorName})
	}
	sort.Slice(out.Actors, func(i, j int) bool { return out.Actors[i].Name < out.Actors[j].Name })
	groups, err := s.Q.AuditLogActionGroups(ctx, &ws.ID)
	if err != nil {
		return AuditEventList{}, err
	}
	for _, g := range groups {
		out.ActionGroups = append(out.ActionGroups, g.Grp)
	}
	out.Stream = s.streamStatus(ws)
	return out, nil
}

func ipString(ip *netip.Addr) string {
	if ip == nil || !ip.IsValid() {
		return ""
	}
	return ip.String()
}

// AuditDiffLine is one line of the before → after view.
type AuditDiffLine struct {
	Kind string `json:"kind" enum:"same,del,add"`
	Text string `json:"text"`
}

// AuditEventDetail is the event detail panel.
type AuditEventDetail struct {
	Event AuditEventItem `json:"event"`
	// What is a sentence: "Changed the Broadcasts routing rule".
	What      string `json:"what"`
	Workspace string `json:"workspace"`
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	// Client is the user agent in words: "Chrome 141 on macOS 26".
	Client string `json:"client,omitempty"`
	// Permission is the permission the action needed (providers:manage).
	Permission string          `json:"permission,omitempty"`
	Before     json.RawMessage `json:"before,omitempty"`
	After      json.RawMessage `json:"after,omitempty"`
	Diff       []AuditDiffLine `json:"diff"`
	Related    []string        `json:"related"`
	// Link is the app path of the target, with its label ("Open routing").
	Link      string `json:"link,omitempty"`
	LinkLabel string `json:"linkLabel,omitempty"`
	// Restore undoes the delete the entry records while that is still
	// possible: a deleted list, for 30 days.
	Restore *AuditRestore `json:"restore,omitempty"`
	// JSON is the full entry for "Copy JSON".
	JSON json.RawMessage `json:"json"`
}

// AuditRestore is a delete an entry can undo, until when.
type AuditRestore struct {
	Kind  string    `json:"kind" enum:"list"`
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Until time.Time `json:"until"`
}

// Get returns one entry of the workspace log.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, id uuid.UUID) (AuditEventDetail, error) {
	e, err := s.Q.AuditLogGet(ctx, store.AuditLogGetParams{ID: id, WorkspaceID: &ws.ID})
	if err != nil {
		return AuditEventDetail{}, api.MapError(err, "Audit entry")
	}
	l := i18n.LangFrom(ctx)
	m := parseMeta(e.Metadata)
	kept := audit.KeptIn(e.Metadata)
	it := item(l, e.ID, e.OccurredAt, e.ActorType, e.ActorID, e.ActorName, e.ActorColor, e.ActorRole, e.Action, e.TargetType, e.TargetID, e.TargetName, e.Result, e.Reason, ipString(e.Ip), e.RequestID, e.Metadata, e.Before, e.After)
	out := AuditEventDetail{
		Event: it, Workspace: e.WorkspaceName, IP: ipString(e.Ip), UserAgent: e.UserAgent, Client: ClientIn(l, e.UserAgent),
		Permission: m.Perm, Before: nonEmpty(e.Before), After: nonEmpty(e.After), Diff: Diff(e.Before, e.After), Related: []string{},
	}
	if out.Diff == nil {
		out.Diff = []AuditDiffLine{}
	}
	out.What = audit.In(kept.What, l, m.What)
	if out.What == "" {
		out.What = DescribeIn(l, e.Action, e.TargetType, e.TargetName)
	}
	if m.Related != "" {
		out.Related = append(out.Related, m.Related)
	} else {
		rel, err := s.Q.AuditLogRelated(ctx, store.AuditLogRelatedParams{WorkspaceID: &ws.ID, ID: e.ID, TargetID: e.TargetID, TargetType: e.TargetType, TargetName: e.TargetName,
			Since: e.OccurredAt.Add(-24 * time.Hour), Until: e.OccurredAt.Add(24 * time.Hour)})
		if err == nil {
			for _, r := range rel {
				out.Related = append(out.Related, relatedLine(l, r.Action, r.ActorName, r.OccurredAt.In(loc(ws.Timezone)), r.Metadata))
			}
		}
	}
	out.Link, out.LinkLabel = link(l, ws.Slug, e.TargetType, e.TargetID, m.LinkPath)
	if e.Action == "list.delete" {
		out.Restore = s.listRestore(ctx, ws.ID, e.TargetID, e.TargetName)
	}
	full, _ := json.Marshal(map[string]any{
		"id": e.ID, "occurred_at": e.OccurredAt, "workspace": ws.Slug, "actor": map[string]any{"type": e.ActorType, "id": e.ActorID, "name": e.ActorName},
		"action": e.Action, "target": map[string]any{"type": e.TargetType, "id": e.TargetID, "name": e.TargetName},
		"result": e.Result, "reason": e.Reason, "ip": out.IP, "user_agent": e.UserAgent, "request_id": e.RequestID,
		"before": nonEmpty(e.Before), "after": nonEmpty(e.After), "metadata": nonEmpty(e.Metadata),
	})
	out.JSON = full
	return out, nil
}

// listRestore offers Restore list on a list.delete entry while the list is
// still deleted and inside its restore window.
func (s *Service) listRestore(ctx context.Context, wsID uuid.UUID, targetID, name string) *AuditRestore {
	id, err := uuid.Parse(targetID)
	if err != nil {
		return nil
	}
	at, err := s.Q.ListDeletedAt(ctx, store.ListDeletedAtParams{WorkspaceID: wsID, ID: id})
	if err != nil || at == nil {
		return nil
	}
	until := at.AddDate(0, 0, audience.RestoreDays)
	if !s.now().Before(until) {
		return nil
	}
	return &AuditRestore{Kind: "list", ID: id.String(), Name: name, Until: until}
}

func nonEmpty(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

func loc(tz string) *time.Location {
	if l, err := time.LoadLocation(tz); err == nil {
		return l
	}
	return time.UTC
}

// link maps a target to the screen that shows it, with its button's label
// in a language.
func link(l i18n.Lang, ws, targetType, targetID, override string) (string, string) {
	base := "/w/" + ws
	if override != "" {
		return base + override, say(l, "link.open", nil)
	}
	path, key := "", ""
	switch targetType {
	case "routing_rule", "routing":
		path, key = "/providers/routing", "link.routing"
	case "provider":
		path, key = "/providers", "link.providers"
	case "campaign":
		if targetID != "" {
			path, key = "/campaigns/"+targetID+"/report", "link.campaign"
		}
	case "member":
		if targetID != "" {
			path, key = "/settings/team/"+targetID, "link.member"
		}
	case "invitation":
		path, key = "/settings/team", "link.team"
	case "role":
		if targetID != "" {
			path, key = "/settings/roles/"+targetID, "link.role"
		}
	case "api_key", "mcp_agent":
		path, key = "/settings/keys", "link.keys"
	case "webhook", "webhook_delivery":
		path, key = "/settings/webhooks", "link.webhooks"
	case "workspace":
		path, key = "/settings", "link.settings"
	case "approval_request":
		if targetID != "" {
			path, key = "/approvals/"+targetID, "link.approval"
		}
	case "contacts", "contact":
		path, key = "/audience", "link.audience"
	case "transactional_template":
		if targetID != "" {
			path, key = "/transactional/templates/"+targetID, "link.template"
		}
	case "transactional_message":
		if targetID != "" {
			path, key = "/transactional/messages/"+targetID, "link.message"
		}
	case "transactional_log":
		path, key = "/transactional/messages", "link.messages"
	case "transactional_category", "transactional_settings":
		path, key = "/transactional/settings", "link.settings"
	case "ai_assistant":
		path, key = "/settings/integrations/ai", "link.ai"
	case "store":
		if targetID != "" {
			path, key = "/settings/integrations/stores/"+targetID, "link.store"
		}
	}
	if key == "" {
		return "", ""
	}
	return base + path, say(l, key, nil)
}

// verbs map the last part of an action to its past-tense phrase, the
// message of the What line (what.<key>).
var verbs = map[string]string{
	"created": "created", "updated": "changed", "deleted": "deleted", "revoked": "revoked", "rotated": "rotated", "replaced": "replaced",
	"invited": "invited", "requested": "requestedApproval", "approved": "approved", "rejected": "rejected", "read": "read",
	"export": "exported", "exported": "exported", "paused": "paused", "resumed": "resumed", "auto_paused": "autoPaused",
	"completed": "finishedSending", "scheduled": "scheduled", "role_changed": "changedRole", "deactivated": "deactivated",
	"reactivated": "reactivated", "removed": "removed", "signed_out": "signedOut", "invite_revoked": "revokedInvitation",
	"invite_resent": "resentInvitation", "transferred": "transferredOwnership", "sign_in": "signedInTo", "tested": "tested",
	"retried": "retriedDeliveries", "secret_rotated": "rotatedSecret", "settings_updated": "changedSettings",
	// Actions named in the present tense.
	"create": "created", "update": "changed", "delete": "deleted", "remove": "removed", "rename": "renamed", "pause": "paused",
	"resume": "resumed", "restore": "restored", "restored": "restored", "unscheduled": "unscheduled", "cancel": "cancelled",
	"cancelled": "cancelled", "revoke": "revoked", "publish": "published", "upload": "uploaded", "connect": "connected",
	"disconnect": "disconnected", "duplicate": "duplicated", "erase": "erased", "add": "added",
	// Transactional email.
	"template_created": "created", "template_updated": "changed", "template_saved": "savedDraft", "template_published": "published",
	"template_rolled_back": "rolledBack", "template_archived": "archived", "template_restored": "restored", "template_deleted": "deleted",
	"category_created": "created", "category_updated": "changed", "category_deleted": "deleted", "content_viewed": "viewedContent",
	"message_resent": "resent", "log_exported": "exported", "enabled": "turnedOn", "disabled": "turnedOff",
	// The AI assistant.
	"connection_created": "connected", "connection_updated": "changed", "connection_removed": "disconnected", "key_replaced": "replacedKey",
	"model_changed": "changedModel", "cap_updated": "changedCap", "prices_updated": "changedPrices",
	"feature_enabled": "featureOn", "feature_disabled": "featureOff",
	// Stores (phase 7).
	"connect_started": "startedConnecting", "keys_saved": "savedAppKeys", "installed": "installedApp", "connected": "connected",
	"reconnected": "reconnected", "disconnected": "disconnected", "data_deleted": "deletedStoreData", "list_updated": "changedList",
	"writeback_updated": "changedWriteback", "attribution_updated": "changedAttribution",
	"sync_updated": "changedSync", "sync_started": "startedSync", "webhooks_registered": "registeredWebhooks",
}

// Describe writes "Changed the Broadcasts routing rule" style sentences
// from an action, in English.
func Describe(action, targetType, targetName string) string {
	return DescribeIn(i18n.Default, action, targetType, targetName)
}

// DescribeIn writes an entry's What line in a language: a whole sentence
// per action's verb (what.<key>), with what was acted on (type.<type>) and
// its name. An action whose verb has no sentence reads as its words, in
// English, as audit action names do.
func DescribeIn(l i18n.Lang, action, targetType, targetName string) string {
	noun, verb, ok := strings.Cut(action, ".")
	if !ok {
		return strings.ReplaceAll(action, "-", " ")
	}
	obj := typeIn(l, targetType)
	if obj == "" {
		obj = typeIn(l, noun)
	}
	key, known := verbs[verb]
	if !known {
		v := strings.ToUpper(verb[:1]) + strings.ReplaceAll(verb[1:], "_", " ")
		en := TypeLabel(targetType, false)
		if en == "" {
			en = TypeLabel(noun, false)
		}
		if targetName != "" {
			return fmt.Sprintf("%s %s %s", v, en, quoteName(targetName))
		}
		return fmt.Sprintf("%s %s", v, en)
	}
	if targetName != "" && targetType == "routing_rule" {
		if key == "changed" {
			return say(l, "what.routingRuleChanged", i18n.Args{"name": targetName})
		}
		// No other verb names a routing rule today: as before, in English.
		v := strings.TrimSpace(say(i18n.Default, "what."+key, i18n.Args{"hasName": "no", "type": ""}))
		return fmt.Sprintf("%s the %s routing rule", v, targetName)
	}
	args := i18n.Args{"hasName": "no", "type": obj, "name": ""}
	if targetName != "" {
		args["hasName"], args["name"] = "yes", quoteName(targetName)
	}
	return say(l, "what."+key, args)
}

// acronyms keep their case in type names: api_key → "API key".
var acronyms = map[string]string{"api": "API", "mcp": "MCP", "sso": "SSO", "dns": "DNS", "smtp": "SMTP", "ip": "IP", "url": "URL", "ai": "AI"}

// TypeLabel turns a target type into words: "routing_rule" → "routing rule"
// ("Routing rule" with capital), "api_key" → "API key".
func TypeLabel(t string, capital bool) string {
	words := strings.FieldsFunc(t, func(r rune) bool { return r == '_' || r == '-' })
	for i, w := range words {
		if a, ok := acronyms[strings.ToLower(w)]; ok {
			words[i] = a
		} else if i == 0 && capital {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func quoteName(n string) string {
	if strings.ContainsAny(n, "@/") || strings.HasPrefix(n, "“") {
		return n
	}
	return "“" + n + "”"
}

var (
	reChrome  = regexp.MustCompile(`(?:Chrome|CriOS)/(\d+)`)
	reEdge    = regexp.MustCompile(`Edg(?:e|A|iOS)?/(\d+)`)
	reFirefox = regexp.MustCompile(`(?:Firefox|FxiOS)/(\d+)`)
	reSafari  = regexp.MustCompile(`Version/(\d+)(?:\.\d+)*.*Safari/`)
	reMac     = regexp.MustCompile(`Mac OS X (\d+)[_.](\d+)`)
	reIOS     = regexp.MustCompile(`(?:iPhone|CPU) OS (\d+)`)
	reAndroid = regexp.MustCompile(`Android (\d+)`)
	reWindows = regexp.MustCompile(`Windows NT (\d+\.\d+)`)
)

// Client turns a User-Agent into "Chrome 141 on macOS 26".
func Client(ua string) string { return ClientIn(i18n.Default, ua) }

// ClientIn is Client in a language: "Chrome 141 trên macOS 26".
func ClientIn(l i18n.Lang, ua string) string {
	if ua == "" {
		return ""
	}
	browser := ""
	if m := reEdge.FindStringSubmatch(ua); m != nil {
		browser = "Edge " + m[1]
	} else if m := reFirefox.FindStringSubmatch(ua); m != nil {
		browser = "Firefox " + m[1]
	} else if m := reChrome.FindStringSubmatch(ua); m != nil {
		browser = "Chrome " + m[1]
	} else if m := reSafari.FindStringSubmatch(ua); m != nil {
		browser = "Safari " + m[1]
	}
	osName := ""
	switch {
	case reIOS.MatchString(ua):
		osName = "iOS " + reIOS.FindStringSubmatch(ua)[1]
	case reMac.MatchString(ua):
		m := reMac.FindStringSubmatch(ua)
		if m[1] == "10" {
			osName = "macOS"
		} else {
			osName = "macOS " + m[1]
		}
	case reAndroid.MatchString(ua):
		osName = "Android " + reAndroid.FindStringSubmatch(ua)[1]
	case reWindows.MatchString(ua):
		osName = "Windows"
	case strings.Contains(ua, "Linux"):
		osName = "Linux"
	}
	switch {
	case browser != "" && osName != "":
		return say(l, "client.on", i18n.Args{"browser": browser, "os": osName})
	case browser != "":
		return browser
	case osName != "":
		return osName
	}
	tok := strings.Fields(ua)[0]
	if len(tok) > 60 {
		tok = tok[:60]
	}
	return tok
}

// diffSummary is a short line for entries without a summary: "role: Editor → Viewer".
func diffSummary(before, after json.RawMessage) string {
	b, a := orderedObject(before), orderedObject(after)
	var parts []string
	for _, kv := range a {
		bv, ok := lookup(b, kv.key)
		if ok && bytes.Equal(compact(bv), compact(kv.val)) {
			continue
		}
		if ok {
			parts = append(parts, kv.key+": "+short(bv)+" → "+short(kv.val))
		} else {
			parts = append(parts, kv.key+": "+short(kv.val))
		}
		if len(parts) == 3 {
			break
		}
	}
	return strings.Join(parts, " · ")
}

func short(v json.RawMessage) string {
	var s string
	if json.Unmarshal(v, &s) == nil {
		if len(s) > 60 {
			s = s[:60] + "…"
		}
		return s
	}
	out := Format(v)
	if len(out) > 60 {
		out = out[:60] + "…"
	}
	return out
}

type kv struct {
	key string
	val json.RawMessage
}

// orderedObject reads a JSON object keeping key order.
func orderedObject(raw json.RawMessage) []kv {
	if len(raw) == 0 {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil
	}
	var out []kv
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return out
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return out
		}
		out = append(out, kv{fmt.Sprint(k), v})
	}
	return out
}

func lookup(obj []kv, key string) (json.RawMessage, bool) {
	for _, e := range obj {
		if e.key == key {
			return e.val, true
		}
	}
	return nil, false
}

func compact(v json.RawMessage) []byte {
	var b bytes.Buffer
	if err := json.Compact(&b, v); err != nil {
		return v
	}
	return b.Bytes()
}

// Format renders a JSON value on one line with the spacing of the detail
// panel: { "ses_us_east_1": 95, "postal_mta": 5 }.
func Format(v json.RawMessage) string {
	v = bytes.TrimSpace(v)
	if len(v) == 0 {
		return "null"
	}
	switch v[0] {
	case '{':
		obj := orderedObject(v)
		if len(obj) == 0 {
			return "{}"
		}
		parts := make([]string, len(obj))
		for i, e := range obj {
			parts[i] = strconv.Quote(e.key) + ": " + Format(e.val)
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	case '[':
		var arr []json.RawMessage
		if json.Unmarshal(v, &arr) != nil || len(arr) == 0 {
			return "[]"
		}
		parts := make([]string, len(arr))
		for i, e := range arr {
			parts[i] = Format(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return string(compact(v))
}

// Diff lists the top-level keys of before and after, one per line, marking
// changed keys as a removed and an added line.
func Diff(before, after json.RawMessage) []AuditDiffLine {
	b, a := orderedObject(before), orderedObject(after)
	if b == nil && a == nil {
		return nil
	}
	keys := []string{}
	seen := map[string]bool{}
	for _, e := range a {
		if !seen[e.key] {
			seen[e.key] = true
			keys = append(keys, e.key)
		}
	}
	for _, e := range b {
		if !seen[e.key] {
			seen[e.key] = true
			keys = append(keys, e.key)
		}
	}
	var out []AuditDiffLine
	for i, k := range keys {
		comma := ","
		if i == len(keys)-1 {
			comma = ""
		}
		bv, inB := lookup(b, k)
		av, inA := lookup(a, k)
		switch {
		case inB && inA && bytes.Equal(compact(bv), compact(av)):
			out = append(out, AuditDiffLine{Kind: "same", Text: strconv.Quote(k) + ": " + Format(av) + comma})
		default:
			if inB {
				out = append(out, AuditDiffLine{Kind: "del", Text: strconv.Quote(k) + ": " + Format(bv) + comma})
			}
			if inA {
				out = append(out, AuditDiffLine{Kind: "add", Text: strconv.Quote(k) + ": " + Format(av) + comma})
			}
		}
	}
	return out
}

// AuditStreamStatus describes the webhook stream of the log.
type AuditStreamStatus struct {
	Enabled bool `json:"enabled"`
	// Host is where entries go ("warehouse.acme.io"); the full URL is in Settings.
	Host         string     `json:"host,omitempty"`
	URL          string     `json:"url,omitempty"`
	HasSecret    bool       `json:"hasSecret"`
	LastOKAt     *time.Time `json:"lastOkAt,omitempty"`
	FailingSince *time.Time `json:"failingSince,omitempty"`
	LastError    string     `json:"lastError,omitempty"`
	Delivered    int64      `json:"delivered"`
}

type streamState struct {
	LastOKAt     *time.Time `json:"last_ok_at,omitempty"`
	FailingSince *time.Time `json:"failing_since,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
	Delivered    int64      `json:"delivered"`
}

func (s *Service) streamStatus(ws *store.Workspace) AuditStreamStatus {
	st := AuditStreamStatus{Enabled: ws.AuditStreamEnabled, HasSecret: len(ws.AuditStreamSecretEnc) > 0}
	if ws.AuditStreamUrl != nil {
		st.URL = *ws.AuditStreamUrl
		if u, err := url.Parse(*ws.AuditStreamUrl); err == nil {
			st.Host = u.Hostname()
		}
	}
	var state streamState
	_ = json.Unmarshal(ws.AuditStreamStatus, &state)
	st.LastOKAt, st.FailingSince, st.LastError, st.Delivered = state.LastOKAt, state.FailingSince, state.LastError, state.Delivered
	return st
}

// AuditSettings is the retention and stream configuration.
type AuditSettings struct {
	RetentionDays int32             `json:"retentionDays"`
	Stream        AuditStreamStatus `json:"stream"`
	Total         int64             `json:"total"`
	CanManage     bool              `json:"canManage" doc:"Changing settings needs workspace:manage"`
}

// Settings returns the audit settings of the workspace.
func (s *Service) Settings(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AuditSettings, error) {
	total, err := s.Q.AuditLogCount(ctx, &ws.ID)
	if err != nil {
		return AuditSettings{}, err
	}
	return AuditSettings{RetentionDays: ws.AuditRetentionDays, Stream: s.streamStatus(ws), Total: total, CanManage: p.Can(authz.WorkspaceManage)}, nil
}

// AuditSettingsInput changes retention and the stream.
type AuditSettingsInput struct {
	RetentionDays *int32            `json:"retentionDays,omitempty" minimum:"30" maximum:"3650"`
	Stream        *AuditStreamInput `json:"stream,omitempty"`
}

// AuditStreamInput configures streaming the log to a webhook.
type AuditStreamInput struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url,omitempty" maxLength:"500"`
	// RotateSecret issues a new signing secret (shown once).
	RotateSecret bool `json:"rotateSecret,omitempty"`
}

// AuditSettingsResult carries a new stream secret, shown once.
type AuditSettingsResult struct {
	Settings AuditSettings `json:"settings"`
	Secret   string        `json:"secret,omitempty" doc:"New signing secret, shown only in this response"`
}

// UpdateSettings changes retention and the stream configuration.
func (s *Service) UpdateSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AuditSettingsInput) (AuditSettingsResult, error) {
	var res AuditSettingsResult
	before := map[string]any{"retention_days": ws.AuditRetentionDays, "stream_enabled": ws.AuditStreamEnabled, "stream_url": ws.AuditStreamUrl}
	after := map[string]any{}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	q := store.New(tx)
	if in.RetentionDays != nil {
		if err := q.AuditSetRetention(ctx, store.AuditSetRetentionParams{ID: ws.ID, Days: *in.RetentionDays}); err != nil {
			return res, err
		}
		after["retention_days"] = *in.RetentionDays
	}
	if in.Stream != nil {
		u := strings.TrimSpace(in.Stream.URL)
		if u == "" && ws.AuditStreamUrl != nil {
			u = *ws.AuditStreamUrl
		}
		if in.Stream.Enabled && u == "" {
			return res, api.Invalid("body.stream.url", say(i18n.LangFrom(ctx), "problem.streamURL", nil))
		}
		if u != "" {
			pu, err := s.policy().ValidateURL(u)
			if err != nil {
				msg := err.Error()
				return res, api.Invalid("body.stream.url", strings.ToUpper(msg[:1])+msg[1:]+".")
			}
			u = pu.String()
		}
		var enc []byte
		if in.Stream.RotateSecret || (in.Stream.Enabled && len(ws.AuditStreamSecretEnc) == 0) {
			res.Secret = newSecret()
			if enc, err = s.Box.SealString(res.Secret, streamAD(ws.ID)); err != nil {
				return res, err
			}
		}
		var urlPtr *string
		if u != "" {
			urlPtr = &u
		}
		reset := in.Stream.Enabled && (!ws.AuditStreamEnabled || ws.AuditStreamUrl == nil || *ws.AuditStreamUrl != u)
		if err := q.AuditStreamConfigure(ctx, store.AuditStreamConfigureParams{ID: ws.ID, Enabled: in.Stream.Enabled, Url: urlPtr, SecretEnc: enc, ResetCursor: reset}); err != nil {
			return res, err
		}
		after["stream_enabled"] = in.Stream.Enabled
		after["stream_url"] = u
		if res.Secret != "" {
			after["stream_secret"] = "rotated"
		}
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "audit.settings_updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
		Before: before, After: after, Metadata: map[string]any{"permission": string(authz.WorkspaceManage)}}); err != nil {
		return res, err
	}
	if err := tx.Commit(ctx); err != nil {
		return res, err
	}
	fresh, err := s.Q.GetWorkspace(ctx, ws.ID)
	if err != nil {
		return res, err
	}
	res.Settings, err = s.Settings(ctx, &fresh, p)
	return res, err
}

func streamAD(wsID uuid.UUID) string { return "workspace:" + wsID.String() + ":audit_stream_secret" }

func (s *Service) policy() netguard.Policy {
	return netguard.Policy{AllowPrivate: s.AllowPrivate != nil && s.AllowPrivate()}
}

// relatedLine is a related entry in a language: "routing.updated by
// Marcus Chen at 17:20", after its summary when it has one.
func relatedLine(l i18n.Lang, action, actor string, at time.Time, metadata json.RawMessage) string {
	args := i18n.Args{"action": action, "name": actor, "at": at}
	if rm := parseMeta(metadata); rm.Summary != "" {
		args["summary"] = audit.KeptIn(metadata).SummaryIn(l, rm.Summary)
		return say(l, "related.withSummary", args)
	}
	return say(l, "related.entry", args)
}
