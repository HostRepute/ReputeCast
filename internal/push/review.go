package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Review & send for push (brief §3.8, board PushStates): the keys, the
// service worker on each site, the audience, the frequency cap, quiet
// hours, the link, lengths per platform and the image, then approvals as
// for other campaigns.

// Check statuses.
const (
	Pass  = "pass"
	Warn  = "warn"
	Block = "block"
)

// Verdicts.
const (
	VerdictReady    = "ready"
	VerdictWarnings = "warnings"
	VerdictBlocked  = "blocked"
)

// Platform is how a platform shows a notification: about how many
// characters of the title fit, and how many lines of the message of about
// how many characters. The composer's previews cut the text the same way.
type Platform struct {
	Key          string
	TitleChars   int
	MessageLines int
	LineChars    int
	Image        bool
	Buttons      bool
}

// Platforms are the previews' platforms, in the composer's order.
var Platforms = []Platform{
	{Key: "chrome_windows", TitleChars: 40, MessageLines: 3, LineChars: 47, Image: true, Buttons: true},
	{Key: "chrome_mac", TitleChars: 34, MessageLines: 2, LineChars: 31, Buttons: true},
	{Key: "safari_mac", TitleChars: 40, MessageLines: 2, LineChars: 43},
	{Key: "chrome_android", TitleChars: 38, MessageLines: 1, LineChars: 40, Image: true, Buttons: true},
	{Key: "safari_ios", TitleChars: 40, MessageLines: 4, LineChars: 44},
}

// Cut is what a platform shows of a text: lines of words up to its width,
// and the rest hidden.
func Cut(text string, lines, width int) (shown, hidden string) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return "", ""
	}
	line, used := 1, 0
	for i, w := range words {
		n := len([]rune(w))
		need := n
		if used > 0 {
			need++
		}
		if used+need > width && used > 0 {
			line++
			used = 0
			need = n
		}
		if line > lines {
			return strings.Join(words[:i], " "), strings.Join(words[i:], " ")
		}
		used += need
	}
	return strings.Join(words, " "), ""
}

// cutTitle is what a platform shows of a title (one line).
func cutTitle(title string, chars int) (shown, hidden string) {
	r := []rune(title)
	if len(r) <= chars {
		return title, ""
	}
	return strings.TrimSpace(string(r[:chars])), string(r[chars:])
}

// PushCheckSite is a site in the service worker check.
type PushCheckSite struct {
	Host        string `json:"host"`
	Status      string `json:"status" enum:"built_in,found,old_version,missing,unreachable,not_a_worker,unchecked"`
	Subscribers int64  `json:"subscribers"`
	URL         string `json:"url"`
	Version     *int   `json:"version,omitempty"`
	HTTPStatus  *int   `json:"httpStatus,omitempty"`
}

// PushCut is a platform that cuts the message or the title.
type PushCut struct {
	Platform string `json:"platform" enum:"chrome_windows,chrome_mac,safari_mac,chrome_android,safari_ios"`
	// Lines is how many lines of the message it shows.
	Lines int  `json:"lines"`
	Title bool `json:"title"`
}

// PushCheck is one pre-send check, with what it found. The app words it.
type PushCheck struct {
	Key    string `json:"key" enum:"keys,worker,audience,frequency_cap,quiet_hours,link,lengths,image,approval"`
	Status string `json:"status" enum:"pass,warn,block"`
	// Keys.
	KeysCreatedAt  *time.Time `json:"keysCreatedAt,omitempty"`
	KeysAcceptedAt *time.Time `json:"keysAcceptedAt,omitempty"`
	// Service worker: each site of the audience; OtherSites is how many
	// push sites aren't in it.
	Sites      []PushCheckSite `json:"sites"`
	OtherSites int             `json:"otherSites"`
	// Audience: Count is who it goes to, Linked and Anonymous split them.
	Count     int64 `json:"count"`
	Linked    int64 `json:"linked"`
	Anonymous int64 `json:"anonymous"`
	// Frequency cap: the cap, how many are over it, how many had any today.
	Cap      int   `json:"cap"`
	OverCap  int64 `json:"overCap"`
	HadToday int64 `json:"hadToday"`
	// Quiet hours.
	QuietOff  bool       `json:"quietOff"`
	Window    string     `json:"window"`
	At        time.Time  `json:"at"`
	AtZone    string     `json:"atZone"`
	Inside    int64      `json:"inside"`
	Held      int64      `json:"held"`
	HeldZones []PushZone `json:"heldZones"`
	Skipped   int64      `json:"skipped"`
	// Link: what it answered.
	Link       string `json:"link,omitempty"`
	LinkStatus int    `json:"linkStatus,omitempty"`
	LinkMillis int    `json:"linkMillis,omitempty"`
	LinkError  string `json:"linkError,omitempty"`
	// LinkDemo: a demo site's address, which is never fetched.
	LinkDemo bool `json:"linkDemo,omitempty"`
	// Lengths.
	TitleLength   int       `json:"titleLength"`
	MessageLength int       `json:"messageLength"`
	Cuts          []PushCut `json:"cuts"`
	// Image.
	Image *PushImage `json:"image,omitempty"`
	Ratio string     `json:"ratio,omitempty"`
	// Approval.
	Author     string   `json:"author,omitempty"`
	AuthorRole string   `json:"authorRole,omitempty"`
	Missing    []string `json:"missing"`
	Helpers    []string `json:"helpers"`
	Rules      []string `json:"rules"`
}

// PushSchedule is when the campaign goes.
type PushSchedule struct {
	Mode   string     `json:"mode" enum:"now,at"`
	SendAt *time.Time `json:"sendAt,omitempty"`
	Zone   string     `json:"zone"`
	// Outside quiet hours: hold until the window opens, or skip.
	Outside    string `json:"outside" enum:"hold,skip"`
	TTLMinutes int    `json:"ttlMinutes"`
}

// PushReviewSite is a site's part of the audience.
type PushReviewSite struct {
	Host        string `json:"host"`
	Subscribers int64  `json:"subscribers"`
}

// PushReview is Review & send for a push campaign.
type PushReview struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	Status    string                   `json:"status"`
	Version   int                      `json:"version"`
	Verdict   string                   `json:"verdict" enum:"ready,warnings,blocked"`
	CheckedAt time.Time                `json:"checkedAt"`
	Checks    []PushCheck              `json:"checks"`
	Sites     []PushReviewSite         `json:"sites"`
	Audience  PushAudience             `json:"audience"`
	Guards    PushGuards               `json:"guards"`
	Schedule  PushSchedule             `json:"schedule"`
	Test      *PushTestResult          `json:"test,omitempty"`
	Approval  approvals.ReviewApproval `json:"approval"`
	Access    PushAccess               `json:"access"`
	// Send is set once the campaign was scheduled or sent.
	Send *PushSendState `json:"send,omitempty"`
}

// PushSendState is a scheduled or running send.
type PushSendState struct {
	Phase       string     `json:"phase" enum:"scheduled,sending,completed,cancelled"`
	ScheduledAt time.Time  `json:"scheduledAt"`
	Recipients  int64      `json:"recipients"`
	StartedBy   string     `json:"startedBy,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// review is the checks of a campaign as it stands.
type review struct {
	c       store.PushCampaignRow
	aud     audienceSet
	guards  Guards
	g       PushGuards
	checks  []PushCheck
	verdict string
	at      time.Time
}

func (rv *review) counts() (warns, blocks int) {
	for _, c := range rv.checks {
		switch c.Status {
		case Warn:
			warns++
		case Block:
			blocks++
		}
	}
	return
}

// linkCheck fetches the link the notification opens. Links on the demo's
// sites are never fetched.
func (s *Service) linkCheck(ctx context.Context, ws *store.Workspace, link string) (status, millis int, errText string, demo bool) {
	u, err := url.Parse(link)
	if err != nil {
		return 0, 0, "invalid", false
	}
	var isDemo bool
	_ = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM push_sites WHERE workspace_id = $1 AND demo AND removed_at IS NULL AND host = $2)`,
		ws.ID, strings.ToLower(u.Host)).Scan(&isDemo)
	if isDemo {
		return 0, 0, "", true
	}
	client := s.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	cctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, link, nil)
	if err != nil {
		return 0, 0, "invalid", false
	}
	req.Header.Set("User-Agent", "ReputeCast link check (+https://reputecast.com)")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, clip(err.Error(), 160), false
	}
	_ = resp.Body.Close()
	return resp.StatusCode, int(time.Since(start) / time.Millisecond), "", false
}

// buildReview runs the checks.
func (s *Service) buildReview(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.PushCampaignRow, withLink bool) (*review, error) {
	rv := &review{c: c, at: s.now()}
	g, err := s.guards(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	rv.guards = g
	rv.aud, err = s.audience(ctx, ws, c.SiteIds, c.Lists, c.Segments)
	if err != nil {
		return nil, err
	}
	send := rv.at
	if c.ScheduleMode == "at" && c.SendAt != nil && c.SendAt.After(send) {
		send = *c.SendAt
	}
	rv.g, err = s.guardsFor(ctx, ws, g, rv.aud, send)
	if err != nil {
		return nil, err
	}
	add := func(ch PushCheck) {
		if ch.Sites == nil {
			ch.Sites = []PushCheckSite{}
		}
		if ch.Cuts == nil {
			ch.Cuts = []PushCut{}
		}
		if ch.HeldZones == nil {
			ch.HeldZones = []PushZone{}
		}
		if ch.Missing == nil {
			ch.Missing = []string{}
		}
		if ch.Helpers == nil {
			ch.Helpers = []string{}
		}
		if ch.Rules == nil {
			ch.Rules = []string{}
		}
		rv.checks = append(rv.checks, ch)
	}
	// Keys.
	key, kerr := s.liveKey(ctx, s.Q, ws)
	if kerr != nil && !errors.Is(kerr, errNoKeys) {
		return nil, kerr
	}
	if errors.Is(kerr, errNoKeys) {
		add(PushCheck{Key: "keys", Status: Block})
	} else {
		add(PushCheck{Key: "keys", Status: Pass, KeysCreatedAt: &key.CreatedAt, KeysAcceptedAt: key.LastAcceptedAt})
	}
	// Service worker on each site of the audience.
	sites, err := s.sites(ctx, ws)
	if err != nil {
		return nil, err
	}
	wk := PushCheck{Key: "worker", Status: Pass}
	missing, inAudience := 0, 0
	for _, st := range sites {
		in := false
		for _, id := range c.SiteIds {
			if id.String() == st.ID {
				in = true
			}
		}
		if !in {
			wk.OtherSites++
			continue
		}
		inAudience++
		cs := PushCheckSite{Host: st.Host, Status: st.Worker.Status, Subscribers: st.Subscribers, URL: st.Worker.URL, Version: st.Worker.Version, HTTPStatus: st.Worker.HTTPStatus}
		wk.Sites = append(wk.Sites, cs)
		switch st.Worker.Status {
		case WorkerMissing, WorkerUnreach, WorkerNotAWorker:
			missing++
			wk.Status = Warn
		case WorkerOld, WorkerUnchecked:
			if wk.Status == Pass {
				wk.Status = Warn
			}
		}
	}
	if inAudience == 0 || missing == inAudience {
		wk.Status = Block
	}
	add(wk)
	// Audience.
	sum := rv.aud.summary(rv.g.OverCap)
	ac := PushCheck{Key: "audience", Status: Pass, Count: sum.Total, Linked: sum.Linked, Anonymous: sum.Anonymous}
	if sum.Final <= 0 {
		ac.Status = Block
	}
	add(ac)
	// Frequency cap.
	fc := PushCheck{Key: "frequency_cap", Status: Pass, Cap: g.Cap, OverCap: rv.g.OverCap}
	if rv.g.OverCap > 0 {
		fc.Status = Warn
	} else {
		counts, err := s.todayCounts(ctx, ws, send)
		if err != nil {
			return nil, err
		}
		for _, r := range rv.aud.list {
			if counts[r.ID] > 0 {
				fc.HadToday++
			}
		}
	}
	add(fc)
	// Quiet hours.
	qc := PushCheck{Key: "quiet_hours", Status: Pass, QuietOff: !g.Quiet, Window: g.Window.String(), At: send, AtZone: tzName(ws),
		Inside: rv.g.Inside, Held: rv.g.Held}
	for _, z := range rv.g.Zones {
		if !z.Inside {
			qc.HeldZones = append(qc.HeldZones, z)
		}
	}
	if c.Outside == "skip" && rv.g.Held > 0 {
		qc.Skipped, qc.Held = rv.g.Held, 0
		qc.Status = Warn
	}
	add(qc)
	// Link.
	lc := PushCheck{Key: "link", Status: Pass, Link: c.Link}
	switch {
	case c.Link == "":
		lc.Status = Block
	case withLink:
		lc.LinkStatus, lc.LinkMillis, lc.LinkError, lc.LinkDemo = s.linkCheck(ctx, ws, c.Link)
		if !lc.LinkDemo && (lc.LinkError != "" || lc.LinkStatus >= 400 || lc.LinkStatus == 0) {
			lc.Status = Warn
		}
	}
	add(lc)
	// Lengths per platform.
	tl, ml := runes(c.Title), runes(c.Message)
	lg := PushCheck{Key: "lengths", Status: Pass, TitleLength: tl, MessageLength: ml}
	for _, pl := range Platforms {
		_, hiddenMsg := Cut(c.Message, pl.MessageLines, pl.LineChars)
		_, hiddenTitle := cutTitle(c.Title, pl.TitleChars)
		if hiddenMsg != "" || hiddenTitle != "" {
			lg.Cuts = append(lg.Cuts, PushCut{Platform: pl.Key, Lines: pl.MessageLines, Title: hiddenTitle != ""})
		}
	}
	switch {
	case strings.TrimSpace(c.Title) == "":
		lg.Status = Block
	case tl > TitleMax || ml > MessageMax:
		lg.Status = Warn
	case tl > 40:
		// Chrome on Windows cuts the title at about 40 characters.
		lg.Status = Warn
	}
	add(lg)
	// Image.
	ic := PushCheck{Key: "image", Status: Pass}
	if c.ImageMediaID != nil {
		ic.Image = s.image(ctx, *c.ImageMediaID)
		if ic.Image != nil {
			ratio := float64(ic.Image.Width) / math.Max(1, float64(ic.Image.Height))
			ic.Ratio = ratioLabel(ic.Image.Width, ic.Image.Height)
			if ratio < 1.8 || ratio > 2.2 || ic.Image.Bytes > 1<<20 || ic.Image.Width < 360 {
				ic.Status = Warn
			}
		}
	}
	add(ic)
	// Approval.
	ap := PushCheck{Key: "approval", Status: Pass}
	if p != nil {
		if u, err := s.Q.GetUser(ctx, p.UserID); err == nil {
			ap.Author = u.Name
		}
		if !canSend(p) {
			ap.Status = Warn
			if !p.Can(authz.CampaignsSend) {
				ap.Missing = append(ap.Missing, string(authz.CampaignsSend))
			}
			if !p.Can(authz.PushSend) {
				ap.Missing = append(ap.Missing, string(authz.PushSend))
			}
			ap.Helpers = s.helpers(ctx, ws, authz.PushSend)
		}
	}
	add(ap)
	warns, blocks := rv.counts()
	switch {
	case blocks > 0:
		rv.verdict = VerdictBlocked
	case warns > 0:
		rv.verdict = VerdictWarnings
	default:
		rv.verdict = VerdictReady
	}
	return rv, nil
}

// ratioLabel writes an image's shape: "2:1", or "1.6:1".
func ratioLabel(w, h int) string {
	if h <= 0 {
		return ""
	}
	r := float64(w) / float64(h)
	if math.Abs(r-math.Round(r)) < 0.03 {
		return fmt.Sprintf("%d:1", int(math.Round(r)))
	}
	return fmt.Sprintf("%.1f:1", r)
}

// Review is Review & send.
func (s *Service) Review(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (PushReview, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return PushReview{}, err
	}
	rv, err := s.buildReview(ctx, ws, p, c, true)
	if err != nil {
		return PushReview{}, err
	}
	return s.reviewView(ctx, ws, p, rv)
}

func (s *Service) reviewView(ctx context.Context, ws *store.Workspace, p *authz.Principal, rv *review) (PushReview, error) {
	c := rv.c
	out := PushReview{ID: c.ID.String(), Name: c.Name, Status: c.Status, Version: int(c.Version), Verdict: rv.verdict, CheckedAt: rv.at, Checks: rv.checks,
		Sites: []PushReviewSite{}, Audience: rv.aud.summary(rv.g.OverCap), Guards: rv.g, Access: s.accessFor(ctx, ws, p),
		Schedule: PushSchedule{Mode: c.ScheduleMode, SendAt: c.SendAt, Zone: tzName(ws), Outside: c.Outside, TTLMinutes: int(c.TtlMinutes)}}
	bySite := map[uuid.UUID]int64{}
	for _, r := range rv.aud.list {
		bySite[r.SiteID]++
	}
	sites, err := s.Q.PushSites(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, st := range sites {
		if n, ok := bySite[st.ID]; ok {
			out.Sites = append(out.Sites, PushReviewSite{Host: st.Host, Subscribers: n})
		}
	}
	out.Test = s.lastTest(ctx, ws, p, c.ID)
	if s.Approvals != nil {
		a, err := s.Approvals.ChannelReview(ctx, ws, p, c.ID)
		if err != nil {
			return out, err
		}
		out.Approval = a
		if a.Required {
			for i := range out.Checks {
				if out.Checks[i].Key == "approval" {
					out.Checks[i].Status = Warn
					for _, r := range a.Rules {
						out.Checks[i].Rules = append(out.Checks[i].Rules, r.Label)
					}
					if len(out.Checks[i].Helpers) == 0 {
						for _, ap := range a.Approvers {
							out.Checks[i].Helpers = append(out.Checks[i].Helpers, ap.Name)
						}
					}
				}
			}
		}
	}
	if snd, err := s.Q.PushSend(ctx, c.ID); err == nil {
		st := &PushSendState{Phase: snd.Phase, ScheduledAt: snd.ScheduledAt, Recipients: int64(snd.Recipients), CompletedAt: snd.CompletedAt}
		if snd.StartedBy != nil {
			if u, err := s.Q.GetUser(ctx, *snd.StartedBy); err == nil {
				st.StartedBy = u.Name
			}
		}
		out.Send = st
	}
	return out, nil
}

// PushSendSettings is the schedule Review & send picks.
type PushSendSettings struct {
	Mode    string     `json:"mode" enum:"now,at"`
	SendAt  *time.Time `json:"sendAt,omitempty"`
	Outside string     `json:"outside" enum:"hold,skip"`
}

// SaveSendSettings keeps the schedule and what quiet hours do.
func (s *Service) SaveSendSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PushSendSettings) (PushReview, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return PushReview{}, err
	}
	l := i18n.LangFrom(ctx)
	switch c.Status {
	case "draft", "approval", "approved":
	default:
		return PushReview{}, api.Conflict(mProblemScheduleLocked.Text(l, i18n.Args{"status": c.Status}))
	}
	var at *time.Time
	if in.Mode == "at" {
		if in.SendAt == nil || !in.SendAt.After(s.now()) {
			return PushReview{}, api.Invalid("body.sendAt", mProblemFutureTime.Text(l, nil))
		}
		at = in.SendAt
	}
	outside := in.Outside
	if outside != "skip" {
		outside = "hold"
	}
	if err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		return q.PushCampaignScheduleSave(ctx, store.PushCampaignScheduleSaveParams{CampaignID: id, ScheduleMode: in.Mode, SendAt: at, Outside: outside})
	}); err != nil {
		return PushReview{}, err
	}
	if s.Changed != nil && (in.Mode != c.ScheduleMode || !equalTime(at, c.SendAt)) {
		s.Changed(ctx, ws, id)
	}
	return s.Review(ctx, ws, p, id)
}

func equalTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

// scheduleLabel is the schedule as approvals show it.
func scheduleLabel(c store.PushCampaignRow, loc *time.Location) string {
	if c.ScheduleMode == "at" && c.SendAt != nil {
		return c.SendAt.In(loc).Format("Mon 2 Jan · 15:04") + " " + loc.String()
	}
	return "Now"
}

// DescribeForApproval describes a push campaign to the approvals domain
// (approvals.ChannelCampaigns).
func (s *Service) DescribeForApproval(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (approvals.ChannelCampaign, error) {
	c, err := s.Q.PushCampaign(ctx, store.PushCampaignParams{ID: campaignID, WorkspaceID: ws.ID})
	if err != nil {
		return approvals.ChannelCampaign{}, err
	}
	rv, err := s.buildReview(ctx, ws, nil, c, false)
	if err != nil {
		return approvals.ChannelCampaign{}, err
	}
	warns, blocks := rv.counts()
	sum := rv.aud.summary(rv.g.OverCap)
	label := scheduleLabel(c, wsLocation(ws))
	text := c.Title + "\n" + c.Message + "\n" + c.Link
	snap := approvals.Snapshot{Version: int(c.Version), ContentHash: c.ContentHash, Audience: c.Audience, AudienceLabels: s.siteLabels(ctx, ws, c.SiteIds),
		AudienceCount: sum.Final, AudienceBaseline: sum.Final, Route: []approvals.SnapshotShare{}, Channel: "push", Text: text, ScheduleLabel: label,
		Readiness: approvals.SnapshotReadiness{Verdict: rv.verdict, Passed: len(rv.checks) - warns - blocks, Warnings: warns, Blocks: blocks, Total: len(rv.checks), RanAt: rv.at},
		Push: &approvals.ApprovalPushSnapshot{Title: c.Title, Message: c.Message, Link: c.Link, Buttons: buttonsOf(c.Buttons), TTLMinutes: int(c.TtlMinutes),
			Tag: c.Tag, Sites: s.siteLabels(ctx, ws, c.SiteIds), Recipients: sum.Final, Schedule: label}}
	if c.ImageMediaID != nil {
		if im := s.image(ctx, *c.ImageMediaID); im != nil {
			snap.Push.Image = im.URL
		}
	}
	if sites, err := s.sites(ctx, ws); err == nil {
		for _, st := range sites {
			for _, id := range c.SiteIds {
				if id.String() == st.ID && snap.Push.Icon == "" {
					snap.Push.Icon, snap.Push.Host = st.IconURL, st.Host
				}
			}
		}
	}
	cur := approvals.Current{Version: int(c.Version), ContentHash: c.ContentHash, Audience: c.Audience, AudienceNames: snap.AudienceLabels, AudienceCount: sum.Final,
		Text: text, ScheduleLabel: label}
	verdict := "ready"
	switch rv.verdict {
	case VerdictWarnings:
		verdict = "warnings"
	case VerdictBlocked:
		verdict = "blocked"
	}
	return approvals.ChannelCampaign{Name: c.Name, Status: c.Status, Version: int(c.Version), Snapshot: snap, Current: cur, Blocks: blocks,
		SendPermission: authz.PushSend, Facts: approvals.RuleFacts{Recipients: sum.Final, Verdict: verdict, Warnings: warns, Channel: "push"}}, nil
}

func buttonsOf(raw json.RawMessage) []approvals.ApprovalPushButton {
	var bs []PushButton
	_ = json.Unmarshal(raw, &bs)
	out := []approvals.ApprovalPushButton{}
	for _, b := range bs {
		out = append(out, approvals.ApprovalPushButton{Label: b.Label, URL: b.URL})
	}
	return out
}

// siteLabels names a campaign's sites.
func (s *Service) siteLabels(ctx context.Context, ws *store.Workspace, ids []uuid.UUID) []string {
	out := []string{}
	rows, err := s.Q.PushSites(ctx, ws.ID)
	if err != nil {
		return out
	}
	for _, st := range rows {
		for _, id := range ids {
			if id == st.ID {
				out = append(out, st.Host)
			}
		}
	}
	return out
}

// PushSendInput sends or schedules a push campaign as reviewed.
type PushSendInput struct {
	// Version is the version Review & send showed; a newer one is refused.
	Version int `json:"version" minimum:"1"`
}

// PushSendResult is what Send did.
type PushSendResult struct {
	Status      string    `json:"status" enum:"scheduled,sending"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Recipients  int64     `json:"recipients"`
}

// Send sends the campaign now or schedules it, after the checks and the
// approval rules. Sending needs campaigns:send and push:send.
func (s *Service) Send(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in PushSendInput) (PushSendResult, error) {
	if !canSend(p) {
		perm := authz.PushSend
		if !p.Can(authz.CampaignsSend) {
			perm = authz.CampaignsSend
		}
		return PushSendResult{}, api.Forbidden(perm, s.helpers(ctx, ws, perm))
	}
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return PushSendResult{}, err
	}
	l := i18n.LangFrom(ctx)
	switch c.Status {
	case "draft", "approval", "approved":
	default:
		return PushSendResult{}, api.Conflict(mProblemAlreadySent.Text(l, i18n.Args{"status": c.Status}))
	}
	// A scoped member must narrow the sites' subscribers to their lists.
	if len(p.Scope.Lists) > 0 && (len(c.Lists) == 0 || !access.AudienceInScope(p.Scope, c.Lists, len(c.Segments))) {
		return PushSendResult{}, api.Err(http.StatusForbidden, api.CodePermissionDenied, access.AudienceScopeProblem(l))
	}
	if in.Version != int(c.Version) {
		return PushSendResult{}, api.Err(409, "stale_version", mProblemChanged.Text(l, i18n.Args{"version": strconv.Itoa(int(c.Version))}))
	}
	rv, err := s.buildReview(ctx, ws, p, c, false)
	if err != nil {
		return PushSendResult{}, err
	}
	if _, blocks := rv.counts(); blocks > 0 {
		return PushSendResult{}, api.Err(409, "readiness_blocked", mProblemBlocked.Text(l, nil))
	}
	var approval *store.ApprovalRequest
	if s.Approvals != nil {
		approval, err = s.Approvals.ChannelGate(ctx, ws, p, id)
		if err != nil {
			return PushSendResult{}, err
		}
	}
	now := s.now()
	at := now
	if c.ScheduleMode == "at" && c.SendAt != nil && c.SendAt.After(now) {
		at = *c.SendAt
	}
	sum := rv.aud.summary(rv.g.OverCap)
	var approvedBy *uuid.UUID
	if approval != nil && approval.DecidedBy != nil {
		approvedBy = approval.DecidedBy
	}
	status := "scheduled"
	if !at.After(now.Add(time.Minute)) {
		status = "sending"
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.PushSendUpsert(ctx, store.PushSendUpsertParams{CampaignID: id, WorkspaceID: ws.ID, Phase: "scheduled", Recipients: int32(sum.Final),
			LeftOut: json.RawMessage(`{}`), StartedBy: uuidPtr(p.UserID), ApprovedBy: approvedBy, ScheduledAt: at}); err != nil {
			return err
		}
		if err := q.PushCampaignSetStatus(ctx, store.PushCampaignSetStatusParams{ID: id, Status: "scheduled", ScheduledAt: &at, StartedBy: uuidPtr(p.UserID),
			AudienceCount: &sum.Final, At: now}); err != nil {
			return err
		}
		if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "campaign.scheduled", map[string]any{"campaign_id": id.String(), "name": c.Name, "channel": "push",
			"scheduled_at": at, "recipients": sum.Final}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.scheduled", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "scheduled"},
			Metadata: map[string]any{"channel": "push", "recipients": sum.Final, "scheduled_at": at, "sites": s.siteLabels(ctx, ws, c.SiteIds)}})
	})
	if err != nil {
		return PushSendResult{}, err
	}
	s.kickStart(ctx)
	return PushSendResult{Status: status, ScheduledAt: at, Recipients: sum.Final}, nil
}

// Unschedule takes a scheduled campaign back to a draft.
func (s *Service) Unschedule(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	snd, err := s.Q.PushSend(ctx, id)
	if c.Status != "scheduled" || err != nil || snd.Phase != "scheduled" {
		return api.Conflict(mProblemUnschedule.Text(i18n.LangFrom(ctx), nil))
	}
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM push_sends WHERE campaign_id = $1 AND phase = 'scheduled'`, id); err != nil {
			return err
		}
		if err := q.PushCampaignSetStatus(ctx, store.PushCampaignSetStatusParams{ID: id, Status: "draft", At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.unscheduled", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"status": "scheduled"}, After: map[string]any{"status": "draft"}, Metadata: map[string]any{"channel": "push"}})
	})
}

// Cancel stops a sending campaign: notifications not yet handed to the
// push services don't go.
func (s *Service) Cancel(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	if c.Status != "sending" && c.Status != "scheduled" {
		return api.Conflict(mProblemCancel.Text(i18n.LangFrom(ctx), nil))
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'cancelled', reason = 'cancelled', updated_at = $2
			WHERE campaign_id = $1 AND status IN ('queued','held')`, id, now); err != nil {
			return err
		}
		if err := q.PushSendSetPhase(ctx, store.PushSendSetPhaseParams{CampaignID: id, Phase: "cancelled", CompletedAt: &now}); err != nil {
			return err
		}
		if err := q.PushCampaignSetStatus(ctx, store.PushCampaignSetStatusParams{ID: id, Status: "cancelled", CompletedAt: &now, At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.cancelled", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "cancelled"}, Metadata: map[string]any{"channel": "push"}})
	})
}
