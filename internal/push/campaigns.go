package push

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/store"
)

// Push campaigns (brief §3.8, board PushComposer): a title, a message, a
// link, the site's icon, an image (Chrome on Windows and Android), up to two
// buttons (Chrome and Edge), an expiry and a tag; the audience is sites,
// optionally narrowed to subscribers linked to contacts on lists or in
// segments.

// Recommended lengths.
const (
	TitleMax   = 50
	MessageMax = 120
	// DefaultTTL is the default expiry: 4 hours.
	DefaultTTL = 240
)

// PushButton is a notification's button.
type PushButton struct {
	Label string `json:"label" maxLength:"40"`
	URL   string `json:"url" maxLength:"2000"`
}

// PushImage is the notification's image.
type PushImage struct {
	MediaID string `json:"mediaId"`
	URL     string `json:"url"`
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Bytes   int    `json:"bytes"`
}

// PushAudienceSite is a site in the audience picker.
type PushAudienceSite struct {
	ID          string `json:"id"`
	Host        string `json:"host"`
	Subscribers int64  `json:"subscribers"`
	Selected    bool   `json:"selected"`
}

// PushAudienceRef is a list or segment narrowing the audience.
type PushAudienceRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind" enum:"list,segment"`
}

// PushAudience is how many subscriptions the campaign reaches.
type PushAudience struct {
	// Total is every live subscription of the chosen sites (narrowed by
	// lists and segments when there are any).
	Total     int64 `json:"total"`
	Linked    int64 `json:"linked"`
	Anonymous int64 `json:"anonymous"`
	// OverCap already got the frequency cap's notifications today.
	OverCap int64 `json:"overCap"`
	// Final is who it goes to.
	Final int64 `json:"final"`
	// Narrowed is true when lists or segments narrow the sites'
	// subscribers to linked ones.
	Narrowed bool `json:"narrowed"`
}

// PushTestResult is the last "Send to my browser".
type PushTestResult struct {
	At       time.Time `json:"at"`
	Browser  string    `json:"browser"`
	Platform string    `json:"platform"`
	Status   string    `json:"status" enum:"queued,sent,shown,clicked,failed"`
	Detail   string    `json:"detail,omitempty"`
}

// PushAccess is what the viewer may do with a push campaign.
type PushAccess struct {
	CanEdit bool `json:"canEdit"`
	CanSend bool `json:"canSend"`
	// Missing are the permissions sending needs that the viewer lacks.
	Missing []string `json:"missing"`
	Helpers []string `json:"helpers"`
}

// PushCampaign is the composer's view of a push campaign.
type PushCampaign struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	Version     int        `json:"version"`
	AutosavedAt *time.Time `json:"autosavedAt,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	UpdatedBy   string     `json:"updatedBy"`
	CreatedBy   string     `json:"createdBy"`

	Title      string             `json:"title"`
	Message    string             `json:"message"`
	Link       string             `json:"link"`
	UTM        string             `json:"utm"`
	Image      *PushImage         `json:"image,omitempty"`
	Buttons    []PushButton       `json:"buttons"`
	TTLMinutes int                `json:"ttlMinutes"`
	Tag        string             `json:"tag"`
	ReplaceTag bool               `json:"replaceTag"`
	Sites      []PushAudienceSite `json:"sites"`
	Lists      []PushAudienceRef  `json:"lists"`
	Segments   []PushAudienceRef  `json:"segments"`
	Audience   PushAudience       `json:"audience"`
	// IconURL is the first chosen site's icon; Host its address (the
	// previews show both).
	IconURL string `json:"iconUrl"`
	Host    string `json:"host"`
	// Guards are the frequency cap and quiet hours as they apply now.
	Guards PushGuards `json:"guards"`
	// Schedule is the send settings Review & send keeps; the composer's
	// quiet-hours choice changes them.
	Schedule PushSchedule     `json:"schedule"`
	Test     *PushTestResult  `json:"test,omitempty"`
	Access   PushAccess       `json:"access"`
	PushOn   bool             `json:"pushOn"`
	Limits   PushLengthLimits `json:"limits"`
}

// PushLengthLimits are the composer's counters.
type PushLengthLimits struct {
	Title   int `json:"title"`
	Message int `json:"message"`
	// TitleWindows is about how much of the title Chrome on Windows shows.
	TitleWindows int `json:"titleWindows"`
}

// PushGuards is the frequency cap and quiet hours for an audience.
type PushGuards struct {
	Cap int `json:"cap"`
	// OverCap already got Cap notifications today; SentToday is what they
	// got.
	OverCap   int64           `json:"overCap"`
	SentToday []PushSentToday `json:"sentToday"`
	Quiet     bool            `json:"quiet"`
	Window    string          `json:"window"`
	// Zones is the audience by time zone, with what quiet hours do to a
	// notification sent now.
	Zones []PushZone `json:"zones"`
	// Held is how many a notification sent now holds until the window
	// opens; Inside how many get it now.
	Held   int64 `json:"held"`
	Inside int64 `json:"inside"`
	// Unknown subscribers' time zones came from nowhere: they're treated
	// as the workspace's.
	Unknown int64 `json:"unknown"`
}

// PushSentToday is a notification sent today.
type PushSentToday struct {
	Name string `json:"name"`
	// Kind is "campaign" or "automation"; By the campaign's sender or the
	// automation's name.
	Kind string    `json:"kind" enum:"campaign,automation"`
	By   string    `json:"by"`
	To   int64     `json:"to"`
	At   time.Time `json:"at"`
}

// PushZone is the audience in one time zone.
type PushZone struct {
	Zone        string    `json:"zone"`
	LocalTime   string    `json:"localTime"`
	Subscribers int64     `json:"subscribers"`
	Inside      bool      `json:"inside"`
	Until       time.Time `json:"until"`
}

// PushCampaignCreate makes a push campaign.
type PushCampaignCreate struct {
	Name string `json:"name" minLength:"1" maxLength:"200"`
	// SiteIDs are the sites it goes to; empty picks every site.
	SiteIDs []string `json:"siteIds,omitempty" maxItems:"20"`
	Title   string   `json:"title,omitempty" maxLength:"200"`
	Message string   `json:"message,omitempty" maxLength:"500"`
	Link    string   `json:"link,omitempty" maxLength:"2000"`
}

// PushCampaignUpdate is an autosave from the composer.
type PushCampaignUpdate struct {
	Name         string       `json:"name" minLength:"1" maxLength:"200"`
	Title        string       `json:"title" maxLength:"200"`
	Message      string       `json:"message" maxLength:"500"`
	Link         string       `json:"link" maxLength:"2000"`
	ImageMediaID string       `json:"imageMediaId,omitempty" maxLength:"40"`
	Buttons      []PushButton `json:"buttons" maxItems:"2"`
	TTLMinutes   int          `json:"ttlMinutes" minimum:"5" maximum:"40320"`
	Tag          string       `json:"tag" maxLength:"64"`
	ReplaceTag   bool         `json:"replaceTag"`
	SiteIDs      []string     `json:"siteIds" maxItems:"20"`
	Lists        []string     `json:"lists" maxItems:"50"`
	Segments     []string     `json:"segments" maxItems:"50"`
	// Version is the version the composer edited; a save from an older one
	// is refused.
	Version int `json:"version" minimum:"0"`
}

// canSend reports whether p may send push: campaigns:send and push:send.
func canSend(p *authz.Principal) bool { return p.Can(authz.CampaignsSend) && p.Can(authz.PushSend) }

func (s *Service) accessFor(ctx context.Context, ws *store.Workspace, p *authz.Principal) PushAccess {
	a := PushAccess{CanEdit: p.Can(authz.CampaignsEdit), CanSend: canSend(p), Missing: []string{}, Helpers: []string{}}
	if !p.Can(authz.CampaignsSend) {
		a.Missing = append(a.Missing, string(authz.CampaignsSend))
	}
	if !p.Can(authz.PushSend) {
		a.Missing = append(a.Missing, string(authz.PushSend))
	}
	if !a.CanSend || !a.CanEdit {
		perm := authz.PushSend
		if !a.CanEdit {
			perm = authz.CampaignsEdit
		}
		a.Helpers = s.helpers(ctx, ws, perm)
	}
	return a
}

func (s *Service) campaign(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.PushCampaignRow, error) {
	c, err := s.Q.PushCampaign(ctx, store.PushCampaignParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return c, api.MapError(err, "Push campaign")
	}
	return c, nil
}

// audienceJSON is the campaign row's audience: its sites, lists and segments.
func audienceJSON(sites, lists, segments []uuid.UUID) json.RawMessage {
	str := func(ids []uuid.UUID) []string {
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			out = append(out, id.String())
		}
		return out
	}
	b, _ := json.Marshal(map[string]any{"push_sites": str(sites), "lists": str(lists), "segments": str(segments)})
	return b
}

// contentHash is what approvals compare: the notification as it would go.
func contentHash(c store.PushCampaignRow) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%v\x00%s\x00%d\x00%s\x00%v", c.Title, c.Message, c.Link, c.ImageMediaID, c.Buttons, c.TtlMinutes, c.Tag, c.ReplaceTag)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// parseIDs reads ids of the workspace's rows.
func parseIDs(raw []string, field, what string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(raw))
	for _, r := range raw {
		id, err := uuid.Parse(strings.TrimSpace(r))
		if err != nil {
			return nil, api.Invalid(field, "Pick "+what+" of this workspace.")
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out, nil
}

// CreateCampaign makes a push campaign, audited like any campaign.
func (s *Service) CreateCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, in PushCampaignCreate) (PushCampaign, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return PushCampaign{}, api.Invalid("body.name", "Give the campaign a name.")
	}
	sites, err := parseIDs(in.SiteIDs, "body.siteIds", "sites")
	if err != nil {
		return PushCampaign{}, err
	}
	all, err := s.Q.PushSites(ctx, ws.ID)
	if err != nil {
		return PushCampaign{}, err
	}
	if len(sites) == 0 {
		for _, st := range all {
			sites = append(sites, st.ID)
		}
	}
	for _, id := range sites {
		if !slices.ContainsFunc(all, func(st store.PushSitesRow) bool { return st.ID == id }) {
			return PushCampaign{}, api.Invalid("body.siteIds", "Pick sites of this workspace's push setup.")
		}
	}
	link, ok := normalizeLink(in.Link)
	if !ok {
		return PushCampaign{}, api.Invalid("body.link", "Enter the address the notification opens, like news.acme.io/p/feature-flags.")
	}
	id := uuid.Must(uuid.NewV7())
	now := s.now()
	row := store.PushCampaignRow{Title: strings.TrimSpace(in.Title), Message: strings.TrimSpace(in.Message), Link: link,
		Buttons: json.RawMessage(`[]`), TtlMinutes: DefaultTTL, ReplaceTag: true}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.PushCampaignInsert(ctx, store.PushCampaignInsertParams{ID: id, WorkspaceID: ws.ID, Name: name, Audience: audienceJSON(sites, nil, nil),
			ContentHash: contentHash(row), By: uuidPtr(p.UserID), At: now}); err != nil {
			return err
		}
		if err := q.PushCampaignDetailInsert(ctx, store.PushCampaignDetailInsertParams{CampaignID: id, WorkspaceID: ws.ID, Title: row.Title, Message: row.Message,
			Link: row.Link, Buttons: row.Buttons, TtlMinutes: DefaultTTL, Tag: "", ReplaceTag: true, SiteIds: sites, Lists: []uuid.UUID{}, Segments: []uuid.UUID{},
			ScheduleMode: "now", Outside: "hold"}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.create", TargetType: "campaign", TargetID: id.String(), TargetName: name,
			After: map[string]any{"name": name, "channel": "push", "sites": len(sites)}})
	})
	if err != nil {
		return PushCampaign{}, err
	}
	return s.Campaign(ctx, ws, p, id)
}

// Campaign is the composer's view.
func (s *Service) Campaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (PushCampaign, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return PushCampaign{}, err
	}
	return s.composerView(ctx, ws, p, c)
}

func (s *Service) composerView(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.PushCampaignRow) (PushCampaign, error) {
	out := PushCampaign{ID: c.ID.String(), Name: c.Name, Status: c.Status, Version: int(c.Version), AutosavedAt: c.AutosavedAt, UpdatedAt: c.UpdatedAt,
		UpdatedBy: c.UpdatedByName, CreatedBy: c.CreatedByName, Title: c.Title, Message: c.Message, Link: c.Link, UTM: utmCampaign(c.Name),
		Buttons: []PushButton{}, TTLMinutes: int(c.TtlMinutes), Tag: c.Tag, ReplaceTag: c.ReplaceTag, Sites: []PushAudienceSite{}, Lists: []PushAudienceRef{},
		Segments: []PushAudienceRef{}, Access: s.accessFor(ctx, ws, p), Limits: PushLengthLimits{Title: TitleMax, Message: MessageMax, TitleWindows: 40},
		Schedule: PushSchedule{Mode: c.ScheduleMode, SendAt: c.SendAt, Zone: tzName(ws), Outside: c.Outside, TTLMinutes: int(c.TtlMinutes)}}
	_ = json.Unmarshal(c.Buttons, &out.Buttons)
	if out.Buttons == nil {
		out.Buttons = []PushButton{}
	}
	_, keyErr := s.liveKey(ctx, s.Q, ws)
	out.PushOn = keyErr == nil
	if c.ImageMediaID != nil {
		out.Image = s.image(ctx, *c.ImageMediaID)
	}
	sites, err := s.sites(ctx, ws)
	if err != nil {
		return out, err
	}
	for _, st := range sites {
		sel := slices.ContainsFunc(c.SiteIds, func(id uuid.UUID) bool { return id.String() == st.ID })
		out.Sites = append(out.Sites, PushAudienceSite{ID: st.ID, Host: st.Host, Subscribers: st.Subscribers, Selected: sel})
		if sel && out.Host == "" {
			out.Host, out.IconURL = st.Host, st.IconURL
		}
	}
	if out.Host == "" && len(sites) > 0 {
		out.Host, out.IconURL = sites[0].Host, sites[0].IconURL
	}
	out.Lists, out.Segments = s.refs(ctx, ws, c.Lists, c.Segments)
	aud, err := s.audience(ctx, ws, c.SiteIds, c.Lists, c.Segments)
	if err != nil {
		return out, err
	}
	g, err := s.guards(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	out.Guards, err = s.guardsFor(ctx, ws, g, aud, s.now())
	if err != nil {
		return out, err
	}
	out.Audience = aud.summary(out.Guards.OverCap)
	out.Test = s.lastTest(ctx, ws, p, c.ID)
	return out, nil
}

// refs names a campaign's lists and segments.
func (s *Service) refs(ctx context.Context, ws *store.Workspace, lists, segments []uuid.UUID) ([]PushAudienceRef, []PushAudienceRef) {
	ls, ss := []PushAudienceRef{}, []PushAudienceRef{}
	if len(lists) > 0 {
		rows, err := s.Pool.Query(ctx, `SELECT id, name FROM lists WHERE workspace_id = $1 AND id = ANY($2) AND deleted_at IS NULL ORDER BY name`, ws.ID, lists)
		if err == nil {
			for rows.Next() {
				var id uuid.UUID
				var n string
				if rows.Scan(&id, &n) == nil {
					ls = append(ls, PushAudienceRef{ID: id.String(), Name: n, Kind: "list"})
				}
			}
			rows.Close()
		}
	}
	if len(segments) > 0 {
		rows, err := s.Pool.Query(ctx, `SELECT id, name FROM segments WHERE workspace_id = $1 AND id = ANY($2) ORDER BY name`, ws.ID, segments)
		if err == nil {
			for rows.Next() {
				var id uuid.UUID
				var n string
				if rows.Scan(&id, &n) == nil {
					ss = append(ss, PushAudienceRef{ID: id.String(), Name: n, Kind: "segment"})
				}
			}
			rows.Close()
		}
	}
	return ls, ss
}

// image reads a media asset as the notification's image.
func (s *Service) image(ctx context.Context, id uuid.UUID) *PushImage {
	var name string
	var w, h, size int
	if err := s.Pool.QueryRow(ctx, `SELECT file_name, width, height, size_bytes FROM media_assets WHERE id = $1`, id).Scan(&name, &w, &h, &size); err != nil {
		return nil
	}
	return &PushImage{MediaID: id.String(), URL: s.origin() + media.URL(id, name), Name: name, Width: w, Height: h, Bytes: size}
}

// recipient is one subscription in an audience.
type recipient struct {
	ID       uuid.UUID
	SiteID   uuid.UUID
	Browser  string
	TimeZone string
	Contact  *uuid.UUID
	Demo     bool
	// WorkerMissing is a snippet site whose service worker the last check
	// didn't find: its notifications are held until the file is back.
	WorkerMissing bool
}

// workerMissingSQL is true for a site whose service worker is missing.
const workerMissingSQL = `(st.kind = 'snippet' AND st.check_status IN ('missing','unreachable','not_a_worker'))`

// audienceSet is a campaign's audience before the guards.
type audienceSet struct {
	list     []recipient
	narrowed bool
}

func (a audienceSet) summary(overCap int64) PushAudience {
	out := PushAudience{Total: int64(len(a.list)), OverCap: overCap, Narrowed: a.narrowed}
	for _, r := range a.list {
		if r.Contact != nil {
			out.Linked++
		}
	}
	out.Anonymous = out.Total - out.Linked
	out.Final = out.Total - overCap
	return out
}

// audience lists the live subscriptions of the sites, narrowed to those
// linked to contacts on the lists or in the segments when there are any.
func (s *Service) audience(ctx context.Context, ws *store.Workspace, sites, lists, segments []uuid.UUID) (audienceSet, error) {
	out := audienceSet{narrowed: len(lists) > 0 || len(segments) > 0}
	if len(sites) == 0 {
		return out, nil
	}
	var contacts []uuid.UUID
	if out.narrowed {
		seen := map[uuid.UUID]bool{}
		if len(lists) > 0 {
			rows, err := s.Pool.Query(ctx, `SELECT DISTINCT contact_id FROM list_memberships WHERE list_id = ANY($1) AND status = 'subscribed'`, lists)
			if err != nil {
				return out, err
			}
			ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
			if err != nil {
				return out, err
			}
			for _, id := range ids {
				seen[id] = true
			}
		}
		for _, seg := range segments {
			if s.Segments == nil {
				break
			}
			ids, err := s.Segments.ContactIDs(ctx, ws.ID, seg)
			if err != nil {
				return out, err
			}
			for _, id := range ids {
				seen[id] = true
			}
		}
		contacts = make([]uuid.UUID, 0, len(seen))
		for id := range seen {
			contacts = append(contacts, id)
		}
	}
	rows, err := s.Pool.Query(ctx, `SELECT s.id, s.site_id, s.browser, s.time_zone, s.contact_id, s.demo, `+workerMissingSQL+` FROM push_subscriptions s
		JOIN push_sites st ON st.id = s.site_id AND st.removed_at IS NULL
		WHERE s.workspace_id = $1 AND s.removed_at IS NULL AND s.site_id = ANY($2)
		  AND (NOT $3::boolean OR s.contact_id = ANY($4::uuid[]))
		ORDER BY s.created_at, s.id`, ws.ID, sites, out.narrowed, contacts)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r recipient
		if err := rows.Scan(&r.ID, &r.SiteID, &r.Browser, &r.TimeZone, &r.Contact, &r.Demo, &r.WorkerMissing); err != nil {
			return out, err
		}
		out.list = append(out.list, r)
	}
	return out, rows.Err()
}

// todayCounts is how many notifications each subscription got today, in
// its own time zone (the workspace's when it has none).
func (s *Service) todayCounts(ctx context.Context, ws *store.Workspace, now time.Time) (map[uuid.UUID]int, error) {
	rows, err := s.Pool.Query(ctx, `SELECT p.subscription_id, count(*) FROM channel_messages m
		JOIN push_messages p ON p.message_id = m.id
		JOIN push_subscriptions s ON s.id = p.subscription_id
		WHERE m.workspace_id = $1 AND m.channel = 'push' AND m.kind IN ('campaign','automation') AND m.status IN ('queued','sending','sent','delivered')
		  AND coalesce(m.sent_at, m.send_after, m.created_at) >= (date_trunc('day', $2::timestamptz AT TIME ZONE coalesce(nullif(s.time_zone, ''), $3))
		      AT TIME ZONE coalesce(nullif(s.time_zone, ''), $3))
		  AND coalesce(m.sent_at, m.send_after, m.created_at) < ((date_trunc('day', $2::timestamptz AT TIME ZONE coalesce(nullif(s.time_zone, ''), $3))
		      + interval '1 day') AT TIME ZONE coalesce(nullif(s.time_zone, ''), $3))
		GROUP BY 1`, ws.ID, now, tzName(ws))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]int{}
	for rows.Next() {
		var id uuid.UUID
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// tzName is the workspace's time zone name.
func tzName(ws *store.Workspace) string {
	if ws.Timezone == "" {
		return "UTC"
	}
	return ws.Timezone
}

// zoneOf is a subscription's zone: the browser's, else the workspace's.
func zoneOf(r recipient, ws *store.Workspace) (*time.Location, bool) {
	if r.TimeZone != "" {
		if l, err := time.LoadLocation(r.TimeZone); err == nil {
			return l, true
		}
	}
	return wsLocation(ws), false
}

// guardsFor applies the frequency cap and quiet hours to an audience as of
// now.
func (s *Service) guardsFor(ctx context.Context, ws *store.Workspace, g Guards, aud audienceSet, now time.Time) (PushGuards, error) {
	out := PushGuards{Cap: g.Cap, Quiet: g.Quiet, Window: g.Window.String(), SentToday: []PushSentToday{}, Zones: []PushZone{}}
	counts, err := s.todayCounts(ctx, ws, now)
	if err != nil {
		return out, err
	}
	zones := map[string]*PushZone{}
	for _, r := range aud.list {
		if counts[r.ID] >= g.Cap {
			out.OverCap++
			continue
		}
		loc, known := zoneOf(r, ws)
		if !known {
			out.Unknown++
		}
		z, ok := zones[loc.String()]
		if !ok {
			inside := !g.Quiet || g.Window.Inside(now, loc)
			z = &PushZone{Zone: loc.String(), LocalTime: now.In(loc).Format("15:04"), Inside: inside}
			if !inside {
				z.Until = g.Window.NextOpen(now, loc)
			}
			zones[loc.String()] = z
		}
		z.Subscribers++
		if z.Inside {
			out.Inside++
		} else {
			out.Held++
		}
	}
	for _, z := range zones {
		out.Zones = append(out.Zones, *z)
	}
	sort.Slice(out.Zones, func(i, j int) bool {
		if out.Zones[i].Subscribers != out.Zones[j].Subscribers {
			return out.Zones[i].Subscribers > out.Zones[j].Subscribers
		}
		return out.Zones[i].Zone < out.Zones[j].Zone
	})
	if out.OverCap > 0 {
		out.SentToday, err = s.sentToday(ctx, ws, now)
	}
	return out, err
}

// sentToday lists today's notifications, campaigns and automation steps,
// in the workspace's day.
func (s *Service) sentToday(ctx context.Context, ws *store.Workspace, now time.Time) ([]PushSentToday, error) {
	loc := wsLocation(ws)
	l := now.In(loc)
	start := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
	rows, err := s.Pool.Query(ctx, `SELECT m.kind, coalesce(c.name, ''), coalesce(cu.name, ''), coalesce(a.name, ''), coalesce(m.automation_step, ''),
			m.automation_id, count(*), min(coalesce(m.sent_at, m.created_at))
		FROM channel_messages m
		LEFT JOIN campaigns c ON c.id = m.campaign_id
		LEFT JOIN users cu ON cu.id = c.started_by
		LEFT JOIN automations a ON a.id = m.automation_id
		WHERE m.workspace_id = $1 AND m.channel = 'push' AND m.kind IN ('campaign','automation') AND m.status IN ('sending','sent','delivered')
		  AND coalesce(m.sent_at, m.created_at) >= $2
		GROUP BY 1, 2, 3, 4, 5, 6 ORDER BY 8`, ws.ID, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PushSentToday{}
	for rows.Next() {
		var kind, campaign, by, auto, step string
		var autoID *uuid.UUID
		var n int64
		var at time.Time
		if err := rows.Scan(&kind, &campaign, &by, &auto, &step, &autoID, &n, &at); err != nil {
			return nil, err
		}
		item := PushSentToday{Name: campaign, Kind: kind, By: by, To: n, At: at}
		if kind == "automation" {
			item.By = auto
			item.Name = auto
			if autoID != nil {
				if t := s.stepTitle(ctx, *autoID, step); t != "" {
					item.Name = t
				}
			}
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// lastTest is the viewer's last "Send to my browser" of this campaign.
func (s *Service) lastTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) *PushTestResult {
	var r PushTestResult
	var status, reason string
	var shown, clicked *time.Time
	err := s.Pool.QueryRow(ctx, `SELECT m.created_at, p.browser, coalesce(t.platform, ''), m.status, m.reason, p.shown_at, p.clicked_at
		FROM channel_messages m JOIN push_messages p ON p.message_id = m.id
		LEFT JOIN push_test_subscriptions t ON t.id = p.subscription_id
		WHERE m.workspace_id = $1 AND m.campaign_id = $2 AND m.kind = 'test' AND m.recipient LIKE $3
		ORDER BY m.created_at DESC LIMIT 1`, ws.ID, campaignID, "test:"+p.UserID.String()+":%").
		Scan(&r.At, &r.Browser, &r.Platform, &status, &reason, &shown, &clicked)
	if err != nil {
		return nil
	}
	switch {
	case clicked != nil:
		r.Status = "clicked"
	case shown != nil:
		r.Status = "shown"
	case status == "failed" || status == "cancelled":
		r.Status, r.Detail = "failed", reason
	case status == "queued" || status == "sending":
		r.Status = "queued"
	default:
		r.Status = "sent"
	}
	return &r
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// utmCampaign is the utm_campaign of a campaign's links: its name, as a slug
// ("New post: Feature flags at scale" → "feature-flags-at-scale" without the
// "New post:" lead-in that names the kind of campaign).
func utmCampaign(name string) string {
	n := strings.ToLower(name)
	if i := strings.Index(n, ":"); i >= 0 && i < 20 && len(n) > i+1 {
		n = n[i+1:]
	}
	s := strings.Trim(slugRe.ReplaceAllString(n, "-"), "-")
	if s == "" {
		return "push"
	}
	return clip(s, 60)
}

// withUTM adds utm_source=push and the campaign to a link.
func withUTM(link, campaign string) string {
	u, err := url.Parse(link)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return link
	}
	q := u.Query()
	if q.Get("utm_source") == "" {
		q.Set("utm_source", "push")
	}
	if q.Get("utm_medium") == "" {
		q.Set("utm_medium", "push")
	}
	if q.Get("utm_campaign") == "" && campaign != "" {
		q.Set("utm_campaign", campaign)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// normalizeLink reads a link as people type it ("news.acme.io/p/x").
func normalizeLink(raw string) (string, bool) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", true
	}
	if !strings.Contains(v, "://") {
		v = "https://" + v
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || !strings.Contains(u.Hostname(), ".") && u.Hostname() != "localhost" {
		return "", false
	}
	return u.String(), true
}

// UpdateCampaign autosaves the composer: the notification, its buttons,
// expiry and tag, and the audience.
func (s *Service) UpdateCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PushCampaignUpdate) (PushCampaign, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return PushCampaign{}, err
	}
	switch c.Status {
	case "draft", "approval", "approved":
	default:
		return PushCampaign{}, api.Conflict("This campaign is " + c.Status + ": it can't be edited any more.")
	}
	if in.Version > 0 && in.Version < int(c.Version) {
		return PushCampaign{}, api.Err(409, "stale_version", fmt.Sprintf("Someone saved version %d since you opened it. Reload to see their changes.", c.Version))
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return PushCampaign{}, api.Invalid("body.name", "Give the campaign a name.")
	}
	link, ok := normalizeLink(in.Link)
	if !ok {
		return PushCampaign{}, api.Invalid("body.link", "Enter the address the notification opens, like news.acme.io/p/feature-flags-at-scale.")
	}
	buttons := []PushButton{}
	for i, b := range in.Buttons {
		lbl := strings.TrimSpace(b.Label)
		if lbl == "" && strings.TrimSpace(b.URL) == "" {
			continue
		}
		u, ok := normalizeLink(b.URL)
		if !ok || u == "" || lbl == "" {
			return PushCampaign{}, api.Invalid(fmt.Sprintf("body.buttons[%d]", i), "A button needs a label and the address it opens.")
		}
		buttons = append(buttons, PushButton{Label: clip(lbl, 40), URL: u})
	}
	sites, err := parseIDs(in.SiteIDs, "body.siteIds", "sites")
	if err != nil {
		return PushCampaign{}, err
	}
	lists, err := parseIDs(in.Lists, "body.lists", "lists")
	if err != nil {
		return PushCampaign{}, err
	}
	segments, err := parseIDs(in.Segments, "body.segments", "segments")
	if err != nil {
		return PushCampaign{}, err
	}
	var image *uuid.UUID
	if in.ImageMediaID != "" {
		mid, err := uuid.Parse(in.ImageMediaID)
		if err != nil {
			return PushCampaign{}, api.Invalid("body.imageMediaId", "Pick an image from the media library.")
		}
		var ok bool
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM media_assets WHERE id = $1 AND workspace_id = $2)`, mid, ws.ID).Scan(&ok); err != nil || !ok {
			return PushCampaign{}, api.Invalid("body.imageMediaId", "Pick an image from the media library.")
		}
		image = &mid
	}
	tag := strings.TrimSpace(in.Tag)
	ttl := in.TTLMinutes
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	bb, _ := json.Marshal(buttons)
	next := c
	next.Title, next.Message, next.Link, next.ImageMediaID, next.Buttons = strings.TrimSpace(in.Title), strings.TrimSpace(in.Message), link, image, bb
	next.TtlMinutes, next.Tag, next.ReplaceTag = int32(ttl), tag, in.ReplaceTag
	hash := contentHash(next)
	version := c.Version
	if hash != c.ContentHash || name != c.Name || !slices.Equal(sites, c.SiteIds) || !slices.Equal(lists, c.Lists) || !slices.Equal(segments, c.Segments) {
		version++
	}
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.PushCampaignSave(ctx, store.PushCampaignSaveParams{ID: id, WorkspaceID: ws.ID, Name: name, Audience: audienceJSON(sites, lists, segments),
			ContentHash: hash, Version: version, By: uuidPtr(p.UserID), At: now}); err != nil {
			return err
		}
		if err := q.PushCampaignDetailSave(ctx, store.PushCampaignDetailSaveParams{CampaignID: id, Title: next.Title, Message: next.Message, Link: link,
			ImageMediaID: image, Buttons: bb, TtlMinutes: int32(ttl), Tag: tag, ReplaceTag: in.ReplaceTag, SiteIds: sites, Lists: lists, Segments: segments}); err != nil {
			return err
		}
		if version == c.Version {
			return nil
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.update", TargetType: "campaign", TargetID: id.String(), TargetName: name,
			Before: map[string]any{"version": c.Version}, After: map[string]any{"version": version, "channel": "push", "title": next.Title}})
	})
	if err != nil {
		return PushCampaign{}, err
	}
	if version != c.Version && s.Changed != nil {
		s.Changed(ctx, ws, id)
	}
	return s.Campaign(ctx, ws, p, id)
}

// DuplicateCampaign copies a push campaign as a new draft.
func (s *Service) DuplicateCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (PushCampaign, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return PushCampaign{}, err
	}
	nid := uuid.Must(uuid.NewV7())
	name := clip("Copy of "+c.Name, 200)
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.PushCampaignInsert(ctx, store.PushCampaignInsertParams{ID: nid, WorkspaceID: ws.ID, Name: name, Audience: audienceJSON(c.SiteIds, c.Lists, c.Segments),
			ContentHash: contentHash(c), By: uuidPtr(p.UserID), At: now}); err != nil {
			return err
		}
		if err := q.PushCampaignDetailInsert(ctx, store.PushCampaignDetailInsertParams{CampaignID: nid, WorkspaceID: ws.ID, Title: c.Title, Message: c.Message,
			Link: c.Link, ImageMediaID: c.ImageMediaID, Buttons: c.Buttons, TtlMinutes: c.TtlMinutes, Tag: c.Tag, ReplaceTag: c.ReplaceTag, SiteIds: c.SiteIds,
			Lists: c.Lists, Segments: c.Segments, ScheduleMode: "now", Outside: c.Outside}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.duplicate", TargetType: "campaign", TargetID: nid.String(), TargetName: name,
			After: map[string]any{"from": c.ID.String(), "channel": "push"}})
	})
	if err != nil {
		return PushCampaign{}, err
	}
	return s.Campaign(ctx, ws, p, nid)
}

// runes counts characters as people do.
func runes(s string) int { return utf8.RuneCountInString(s) }
