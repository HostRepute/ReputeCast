package push

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Sending providers › Push › Setup (brief §5.9, board PushSetup): the keys,
// the frequency cap and quiet hours, the sites with their service worker,
// the subscribers by browser and site, and the prompts with their numbers.

// PushKeysView is the keys card. The private key is never returned.
type PushKeysView struct {
	PublicKey      string     `json:"publicKey"`
	CreatedAt      time.Time  `json:"createdAt"`
	CreatedBy      string     `json:"createdBy"`
	Contact        string     `json:"contact"`
	LastAcceptedAt *time.Time `json:"lastAcceptedAt,omitempty"`
}

// PushCount is a count by browser or site.
type PushCount struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// PushListHighlight is the newest list with linked subscribers: "612
// belong to Acme Summit 2026 registrants".
type PushListHighlight struct {
	ListID string `json:"listId"`
	Name   string `json:"name"`
	Count  int64  `json:"count"`
}

// PushSubscribers is the subscribers card.
type PushSubscribers struct {
	Total     int64              `json:"total"`
	Linked    int64              `json:"linked"`
	Anonymous int64              `json:"anonymous"`
	ByBrowser []PushCount        `json:"byBrowser"`
	BySite    []PushCount        `json:"bySite"`
	Highlight *PushListHighlight `json:"highlight,omitempty"`
}

// PushPromptCard is a prompt in Setup's Prompts card.
type PushPromptCard struct {
	FormID  string           `json:"formId"`
	Name    string           `json:"name"`
	Site    string           `json:"site"`
	Status  string           `json:"status" enum:"draft,live,paused"`
	Config  PushPromptConfig `json:"config"`
	Version int              `json:"version"`
	Stats   PushPromptStats  `json:"stats"`
}

// PushSnippet is what another site installs.
type PushSnippet struct {
	// Script is the snippet's src; ImportLine the line for a site that
	// already has a worker.
	Script        string `json:"script"`
	WorkerFile    string `json:"workerFile"`
	WorkerBytes   int    `json:"workerBytes"`
	WorkerVersion int    `json:"workerVersion"`
	WorkerURL     string `json:"workerUrl"`
	ImportURL     string `json:"importUrl"`
}

// PushMonth is this month's sending.
type PushMonth struct {
	Sent      int64     `json:"sent"`
	Campaigns int64     `json:"campaigns"`
	ShownPct  float64   `json:"shownPct"`
	Month     time.Time `json:"month"`
}

// PushSetup is Sending providers › Push › Setup.
type PushSetup struct {
	On          bool             `json:"on"`
	Keys        *PushKeysView    `json:"keys,omitempty"`
	Cap         int              `json:"cap"`
	Quiet       bool             `json:"quiet"`
	Window      string           `json:"window"`
	Sites       []PushSite       `json:"sites"`
	CheckedAt   *time.Time       `json:"checkedAt,omitempty"`
	Subscribers PushSubscribers  `json:"subscribers"`
	Month       PushMonth        `json:"month"`
	Prompts     []PushPromptCard `json:"prompts"`
	Snippet     PushSnippet      `json:"snippet"`
	// Website is the publication website's address when it isn't a push
	// site yet: one click adds it.
	Website   string   `json:"website,omitempty"`
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
	// CanEditPrompts is forms:edit, which the prompts need.
	CanEditPrompts bool `json:"canEditPrompts"`
}

// Setup is the Setup tab.
func (s *Service) Setup(ctx context.Context, ws *store.Workspace, p *authz.Principal) (PushSetup, error) {
	out := PushSetup{Sites: []PushSite{}, Prompts: []PushPromptCard{}, Helpers: []string{}, CanManage: p.Can(authz.ProvidersManage),
		CanEditPrompts: p.Can(authz.FormsEdit), Subscribers: PushSubscribers{ByBrowser: []PushCount{}, BySite: []PushCount{}}}
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws, authz.ProvidersManage)
	}
	src := s.origin() + "/push/v1/prompt.js"
	out.Snippet = PushSnippet{Script: src, WorkerFile: WorkerFile, WorkerBytes: len(HostedWorker()), WorkerVersion: WorkerVersion,
		WorkerURL: s.origin() + "/push/v1/" + WorkerFile, ImportURL: s.origin() + "/push/v1/sw.js"}
	g, err := s.guards(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	out.Cap, out.Quiet, out.Window = g.Cap, g.Quiet, g.Window.String()
	key, err := s.liveKey(ctx, s.Q, ws)
	if err != nil && err != errNoKeys {
		return out, err
	}
	if err == nil {
		out.On = true
		out.Keys = &PushKeysView{PublicKey: key.PublicKey, CreatedAt: key.CreatedAt, CreatedBy: key.CreatedByName, Contact: key.Contact, LastAcceptedAt: key.LastAcceptedAt}
	}
	website := ""
	if s.Website != nil {
		website = s.Website(ctx, ws)
	}
	out.Sites, err = s.sites(ctx, ws)
	if err != nil {
		return out, err
	}
	hasWebsite := false
	for _, st := range out.Sites {
		if st.Worker.CheckedAt != nil && (out.CheckedAt == nil || st.Worker.CheckedAt.After(*out.CheckedAt)) {
			out.CheckedAt = st.Worker.CheckedAt
		}
		if st.Origin == website {
			hasWebsite = true
		}
		out.Subscribers.BySite = append(out.Subscribers.BySite, PushCount{Key: st.Host, Count: st.Subscribers})
		out.Subscribers.Total += st.Subscribers
		out.Subscribers.Linked += st.Linked
	}
	if website != "" && !hasWebsite {
		out.Website = website
	}
	out.Subscribers.Anonymous = out.Subscribers.Total - out.Subscribers.Linked
	browsers, err := s.Q.PushSubscriptionsByBrowser(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	counts := map[string]int64{}
	for _, b := range browsers {
		counts[b.Browser] = b.N
	}
	for _, k := range []string{BrowserChrome, BrowserSafari, BrowserChromeAndroid, BrowserEdge, BrowserFirefox, BrowserSafariIOS, BrowserOther} {
		if counts[k] > 0 {
			out.Subscribers.ByBrowser = append(out.Subscribers.ByBrowser, PushCount{Key: k, Count: counts[k]})
		}
	}
	// The newest list whose contacts have linked subscriptions.
	var h PushListHighlight
	var lid uuid.UUID
	if err := s.Pool.QueryRow(ctx, `SELECT l.id, l.name, count(DISTINCT s.id) FROM lists l
		JOIN list_memberships lm ON lm.list_id = l.id AND lm.status = 'subscribed'
		JOIN push_subscriptions s ON s.contact_id = lm.contact_id AND s.removed_at IS NULL
		WHERE l.workspace_id = $1 AND l.deleted_at IS NULL
		GROUP BY l.id, l.name, l.created_at ORDER BY l.created_at DESC LIMIT 1`, ws.ID).Scan(&lid, &h.Name, &h.Count); err == nil {
		h.ListID = lid.String()
		out.Subscribers.Highlight = &h
	}
	// This month's notifications.
	loc := wsLocation(ws)
	out.Month.Month = monthStart(s.now(), loc)
	var shown int64
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE m.sent_at IS NOT NULL), count(*) FILTER (WHERE p.shown_at IS NOT NULL),
			count(DISTINCT m.campaign_id) FILTER (WHERE m.kind = 'campaign' AND m.sent_at IS NOT NULL)
		FROM channel_messages m JOIN push_messages p ON p.message_id = m.id
		WHERE m.workspace_id = $1 AND m.channel = 'push' AND m.kind IN ('campaign','automation') AND m.created_at >= $2`, ws.ID, out.Month.Month).
		Scan(&out.Month.Sent, &shown, &out.Month.Campaigns); err != nil {
		return out, err
	}
	out.Month.ShownPct = pct(shown, out.Month.Sent)
	// The prompts.
	prompts, err := s.Q.PushPrompts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, pr := range prompts {
		card := PushPromptCard{FormID: pr.FormID.String(), Name: pr.Name, Status: pr.Status, Config: parseConfig(pr.Draft), Version: int(pr.LiveVersion)}
		if live, _ := s.liveConfig(ctx, pr.FormID, pr.LiveVersion); live != nil {
			card.Config = *live
		}
		if pr.SiteID != nil {
			if st, err := s.Q.PushSiteAny(ctx, *pr.SiteID); err == nil {
				card.Site = st.Host
			}
		}
		row, err := s.Q.PushPrompt(ctx, store.PushPromptParams{FormID: pr.FormID, WorkspaceID: ws.ID})
		if err == nil {
			card.Stats, _ = s.promptStats(ctx, ws, row)
		}
		out.Prompts = append(out.Prompts, card)
	}
	return out, nil
}

// PushEvent is one line of the Events tab: what happened to a
// notification, raw beside what ReputeCast made of it.
type PushEvent struct {
	At time.Time `json:"at"`
	// Kind is the normalized event: sent (accepted by the push service),
	// gone, failed, shown, clicked, closed.
	Kind     string `json:"kind" enum:"sent,gone,failed,shown,clicked,closed,queued,delivered,undelivered,replied,opted_out,opted_in,help,read"`
	Raw      string `json:"raw"`
	Browser  string `json:"browser"`
	Site     string `json:"site"`
	Source   string `json:"source"`
	SourceID string `json:"sourceId,omitempty"`
	// SourceKind is campaign, automation or test.
	SourceKind string `json:"sourceKind" enum:"campaign,automation,test,confirmation,reply"`
	Message    string `json:"message"`
}

// PushEventList is a page of events.
type PushEventList struct {
	Items []PushEvent `json:"items"`
	// Removed is how many subscriptions the push services reported gone in
	// the last 30 days.
	Removed int64 `json:"removed"`
	More    bool  `json:"more"`
}

// Events lists the latest events, newest first.
func (s *Service) Events(ctx context.Context, ws *store.Workspace, before *time.Time, limit int) (PushEventList, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	out := PushEventList{Items: []PushEvent{}}
	b := s.now().Add(time.Minute)
	if before != nil {
		b = *before
	}
	rows, err := s.Pool.Query(ctx, `SELECT e.occurred_at, e.kind, e.provider_code, e.detail, p.browser, coalesce(st.host, ''), m.kind,
			coalesce(c.name, a.name, ''), coalesce(m.campaign_id, m.automation_id), m.id
		FROM channel_message_events e
		JOIN channel_messages m ON m.id = e.message_id
		JOIN push_messages p ON p.message_id = m.id
		LEFT JOIN push_sites st ON st.id = p.site_id
		LEFT JOIN campaigns c ON c.id = m.campaign_id
		LEFT JOIN automations a ON a.id = m.automation_id
		WHERE e.workspace_id = $1 AND m.channel = 'push' AND e.occurred_at < $2
		ORDER BY e.occurred_at DESC, e.id DESC LIMIT $3`, ws.ID, b, limit+1)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var e PushEvent
		var detail []byte
		var src *uuid.UUID
		var msg uuid.UUID
		if err := rows.Scan(&e.At, &e.Kind, &e.Raw, &detail, &e.Browser, &e.Site, &e.SourceKind, &e.Source, &src, &msg); err != nil {
			return out, err
		}
		if src != nil {
			e.SourceID = src.String()
		}
		e.Message = msg.String()
		if e.Raw == "" {
			switch e.Kind {
			case "shown", "closed":
				e.Raw = fmt.Sprintf(`{"kind":"notification","event":%q}`, e.Kind)
			case "clicked":
				var d map[string]any
				_ = json.Unmarshal(detail, &d)
				e.Raw = fmt.Sprintf("GET /t/p/… · button %v", d["action"])
			}
		}
		if len(out.Items) == limit {
			out.More = true
			break
		}
		out.Items = append(out.Items, e)
	}
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM push_subscriptions WHERE workspace_id = $1 AND removed_reason = 'gone' AND removed_at >= $2`,
		ws.ID, s.now().AddDate(0, 0, -30)).Scan(&out.Removed)
	return out, rows.Err()
}

// snippetHTML is the snippet a site adds to every page.
func (s *Service) snippetHTML(ws *store.Workspace, host string) string {
	return fmt.Sprintf("<script async src=%q\n  data-workspace=%q\n  data-site=%q></script>", s.origin()+"/push/v1/prompt.js",
		html.EscapeString(ws.Slug), html.EscapeString(host))
}
