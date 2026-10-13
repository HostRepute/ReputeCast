package engine

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// pageSize is how many contacts one audience page reads. Each page is its
// own short transaction, so a 50,000-recipient send never holds a long
// one and never has every recipient in memory.
const pageSize = 2000

// audienceDef is campaigns.audience. The composer writes camelCase keys;
// the base schema documents snake_case ones; both are read.
type audienceDef struct {
	Lists           []uuid.UUID
	Segments        []uuid.UUID
	ExcludeLists    []uuid.UUID
	ExcludeSegments []uuid.UUID
	ExcludeTiers    []string
}

func parseAudience(raw json.RawMessage) audienceDef {
	var in struct {
		Lists            []string `json:"lists"`
		Segments         []string `json:"segments"`
		ExcludeLists     []string `json:"excludeLists"`
		ExcludeLists2    []string `json:"exclude_lists"`
		ExcludeSegments  []string `json:"excludeSegments"`
		ExcludeSegments2 []string `json:"exclude_segments"`
		ExcludeTiers     []string `json:"excludeTiers"`
		ExcludeTiers2    []string `json:"exclude_tiers"`
	}
	_ = json.Unmarshal(raw, &in)
	ids := func(parts ...[]string) []uuid.UUID {
		out := []uuid.UUID{}
		for _, p := range parts {
			for _, s := range p {
				if u, err := uuid.Parse(strings.TrimSpace(s)); err == nil && !slices.Contains(out, u) {
					out = append(out, u)
				}
			}
		}
		return out
	}
	tiers := append(append([]string{}, in.ExcludeTiers...), in.ExcludeTiers2...)
	return audienceDef{Lists: ids(in.Lists), Segments: ids(in.Segments), ExcludeLists: ids(in.ExcludeLists, in.ExcludeLists2),
		ExcludeSegments: ids(in.ExcludeSegments, in.ExcludeSegments2), ExcludeTiers: tiers}
}

// Excluded counts the contacts of the audience who don't receive the send.
type Excluded struct {
	Suppressed   int64 `json:"suppressed"`
	Unsubscribed int64 `json:"unsubscribed"`
	Bounced      int64 `json:"bounced"`
	Complained   int64 `json:"complained"`
	Paused       int64 `json:"paused"`
	Excluded     int64 `json:"excluded"`
	Tiers        int64 `json:"tiers"`
}

// Content is what a send renders, captured at start (campaign_sends.content).
type sendContent struct {
	Subject   string         `json:"subject"`
	Preheader string         `json:"preheader"`
	HTML      string         `json:"html"`
	Text      string         `json:"text"`
	Variants  []abVariant    `json:"variants,omitempty"`
	FromName  string         `json:"from_name"`
	FromEmail string         `json:"from_email"`
	ReplyTo   string         `json:"reply_to"`
	Tracking  Tracking       `json:"tracking"`
	Links     map[string]int `json:"links"` // first link position per variant
}

// abTest is campaigns.ab_test.
type abTest struct {
	Kind          string      `json:"kind"`
	Variants      []abVariant `json:"variants"`
	TestPercent   int         `json:"test_percent"`
	Metric        string      `json:"metric"`
	DurationHours int         `json:"duration_hours"`
	Winner        string      `json:"winner"`
}

type abVariant struct {
	Key       string `json:"key"`
	Subject   string `json:"subject,omitempty"`
	Preheader string `json:"preheader,omitempty"`
	HTML      string `json:"html,omitempty"`
	SendTime  string `json:"send_time,omitempty"`
}

// abState is campaign_sends.ab: the test's phase during the send.
type abState struct {
	Kind        string     `json:"kind,omitempty"`
	Metric      string     `json:"metric,omitempty"`
	TestPercent int        `json:"test_percent,omitempty"`
	Variants    []string   `json:"variants,omitempty"`
	Phase       string     `json:"phase,omitempty"` // testing, decided
	DecideAt    *time.Time `json:"decide_at,omitempty"`
	Winner      string     `json:"winner,omitempty"`
	DecidedAt   *time.Time `json:"decided_at,omitempty"`
	DecidedBy   string     `json:"decided_by,omitempty"`
	// Window overrides the test duration, in seconds (demo sends).
	Window int64 `json:"window,omitempty"`
}

func parseAB(raw json.RawMessage) abState {
	var a abState
	_ = json.Unmarshal(raw, &a)
	return a
}

// Prepare builds a campaign's recipient list: the audience's members minus
// the shared suppression list, unsubscribed, bounced, complained and paused
// contacts and the exclusions, each routed to a sending provider, with A/B
// variants assigned. It pages through contacts with a saved cursor, so it
// can be stopped and run again at any point without sending anyone twice.
func (e *Engine) Prepare(ctx context.Context, campaignID uuid.UUID) error {
	return e.prepare(ctx, campaignID, e.now())
}

func (e *Engine) prepare(ctx context.Context, campaignID uuid.UUID, now time.Time) error {
	cc, err := e.Q.SendingCampaignContext(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if cc.Status != "sending" && cc.Status != "paused" {
		return nil
	}
	snd, err := e.Q.SendingGetSend(ctx, campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if snd.Phase != "preparing" {
		return nil
	}
	def := parseAudience(cc.Audience)
	if len(def.Lists) == 0 && len(def.Segments) == 0 {
		return e.fail(ctx, cc, "The campaign has no audience. Pick a list or segment on Review & send, then start it again.", now)
	}
	if cc.SenderEmail == "" {
		return e.fail(ctx, cc, "The campaign has no sender. Pick one in the composer, then start it again.", now)
	}
	set, err := routing.Load(ctx, e.Q, cc.WorkspaceID)
	if err != nil {
		return err
	}
	provs, err := e.Q.SendingWorkspaceProviders(ctx, cc.WorkspaceID)
	if err != nil {
		return err
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	loc := state.Location(cc.WorkspaceTimezone)
	ab := abTestOf(cc.AbTest)
	content, err := e.snapshotContent(cc, ab)
	if err != nil {
		return e.fail(ctx, cc, "The email can't be rendered: "+err.Error(), now)
	}
	if snd.PrepareCursor == nil {
		if err := e.startPrepare(ctx, cc, def, content, now); err != nil {
			return err
		}
	} else if len(def.Segments)+len(def.ExcludeSegments) > 0 {
		// A restart lost nothing: staged segment members survive in the table,
		// but stage again if they're missing.
		if n, err := e.Q.SendingStageCount(ctx, campaignID); err == nil && n == 0 {
			if err := e.stageSegments(ctx, cc.WorkspaceID, campaignID, def); err != nil {
				return err
			}
		}
	}
	r := &router{e: e, set: set, byID: byID, loc: loc, now: now, override: routeOverride(cc.RouteOverride),
		senderDomain: domainOf(cc.SenderEmail), rates: map[uuid.UUID]providerRate{}, planned: map[uuid.UUID]int64{}}
	excluded := Excluded{}
	_ = json.Unmarshal(snd.Excluded, &excluded)
	cursor := uuid.Nil
	if snd.PrepareCursor != nil {
		cursor = *snd.PrepareCursor
	}
	sendAt := sendTimes(ab, loc, now)
	for {
		rows, err := e.Q.SendingAudiencePage(ctx, store.SendingAudiencePageParams{
			Now: now, ExcludeLists: def.ExcludeLists, CampaignID: campaignID, Lists: def.Lists, WorkspaceID: cc.WorkspaceID, After: cursor, Lim: pageSize,
		})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			break
		}
		msgs := make([][]any, 0, len(rows))
		planned := map[uuid.UUID]int64{}
		for _, c := range rows {
			if why := eligibility(c, def); why != "" {
				excluded.add(why)
				continue
			}
			mbp := c.MailboxProvider
			if mbp == "" {
				mbp = routing.MailboxProvider(c.Email)
			}
			pid, err := r.route(ctx, c, mbp)
			if err != nil {
				return e.fail(ctx, cc, err.Error(), now)
			}
			variant, status, next := assignVariant(ab, c.ID, sendAt)
			msgs = append(msgs, []any{newID(), cc.WorkspaceID, "broadcast", &campaignID, &c.ID, c.Email, variant, &pid, mbp, status, now, next})
			planned[pid]++
		}
		last := rows[len(rows)-1].ID
		exb, _ := json.Marshal(excluded)
		err = db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			// The cursor and the page commit together: a retry continues
			// after the last committed page.
			s, err := q.SendingGetSendForUpdate(ctx, campaignID)
			if err != nil {
				return err
			}
			if s.Phase != "preparing" || (s.PrepareCursor != nil && *s.PrepareCursor != cursor) || (s.PrepareCursor == nil && cursor != uuid.Nil) {
				return errPrepareMoved
			}
			if len(msgs) > 0 {
				if _, err := tx.CopyFrom(ctx, pgx.Identifier{"messages"}, []string{"id", "workspace_id", "stream", "campaign_id", "contact_id", "email", "variant", "provider_id",
					"mailbox_provider", "status", "queued_at", "next_attempt_at"}, pgx.CopyFromRows(msgs)); err != nil {
					return err
				}
			}
			for pid, n := range planned {
				if err := q.SendingAddPlanned(ctx, store.SendingAddPlannedParams{CampaignID: campaignID, ProviderID: pid, Planned: n}); err != nil {
					return err
				}
			}
			return q.SendingSavePrepare(ctx, store.SendingSavePrepareParams{CampaignID: campaignID, Cursor: &last, Added: int64(len(msgs)), Excluded: exb, At: now})
		})
		if errors.Is(err, errPrepareMoved) {
			return nil // another worker is preparing this campaign
		}
		if err != nil {
			return err
		}
		cursor = last
		if len(rows) < pageSize {
			break
		}
	}
	return e.finishPrepare(ctx, cc, ab, now)
}

var errPrepareMoved = errors.New("prepare moved on")

// startPrepare clears what a failed earlier attempt left, stages segment
// members, captures the content and indexes its links.
func (e *Engine) startPrepare(ctx context.Context, cc store.SendingCampaignContextRow, def audienceDef, content sendContent, now time.Time) error {
	cid := cc.ID
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SendingClearMessages(ctx, &cid); err != nil {
			return err
		}
		if err := q.SendingClearRoutes(ctx, cid); err != nil {
			return err
		}
		if err := q.SendingStageClear(ctx, cid); err != nil {
			return err
		}
		if err := q.SendingClearLinks(ctx, cid); err != nil {
			return err
		}
		links, err := e.indexLinks(content)
		if err != nil {
			return err
		}
		for _, l := range links {
			if err := q.SendingInsertLink(ctx, store.SendingInsertLinkParams{ID: newID(), CampaignID: cid, Url: l.URL, Label: l.Label, Position: int32(l.Position)}); err != nil {
				return err
			}
		}
		b, _ := json.Marshal(content)
		return q.SendingSetContent(ctx, store.SendingSetContentParams{CampaignID: cid, Content: b, At: now})
	})
	if err != nil {
		return err
	}
	e.forgetTemplates(cid)
	return e.stageSegments(ctx, cc.WorkspaceID, cid, def)
}

// stageSegments writes segment members in chunks.
func (e *Engine) stageSegments(ctx context.Context, wsID, campaignID uuid.UUID, def audienceDef) error {
	if len(def.Segments)+len(def.ExcludeSegments) == 0 {
		return nil
	}
	if e.Segments == nil {
		return fmt.Errorf("segments can't be resolved in this process")
	}
	stage := func(ids []uuid.UUID, exclude bool) error {
		for len(ids) > 0 {
			n := min(len(ids), 5000)
			if err := e.Q.SendingStageAdd(ctx, store.SendingStageAddParams{CampaignID: campaignID, ContactIds: ids[:n], Exclude: exclude}); err != nil {
				return err
			}
			ids = ids[n:]
		}
		return nil
	}
	for _, seg := range def.Segments {
		ids, err := e.Segments.ContactIDs(ctx, wsID, seg)
		if err != nil {
			return fmt.Errorf("segment %s: %w", seg, err)
		}
		if err := stage(ids, false); err != nil {
			return err
		}
	}
	for _, seg := range def.ExcludeSegments {
		ids, err := e.Segments.ContactIDs(ctx, wsID, seg)
		if err != nil {
			return fmt.Errorf("segment %s: %w", seg, err)
		}
		if err := stage(ids, true); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) finishPrepare(ctx context.Context, cc store.SendingCampaignContextRow, ab *abTest, now time.Time) error {
	snd, err := e.Q.SendingGetSend(ctx, cc.ID)
	if err != nil {
		return err
	}
	st := parseAB(snd.Ab)
	if ab != nil && ab.TestPercent > 0 && ab.TestPercent < 100 && len(ab.Variants) >= 2 && ab.Winner == "" {
		st.Kind, st.Metric, st.TestPercent, st.Phase = ab.Kind, ab.Metric, ab.TestPercent, "testing"
		if st.Metric == "" {
			st.Metric = "human_opens"
		}
		st.Variants = st.Variants[:0]
		for _, v := range ab.Variants {
			st.Variants = append(st.Variants, v.Key)
		}
		window := time.Duration(max(ab.DurationHours, 1)) * time.Hour
		if st.Window > 0 {
			window = time.Duration(st.Window) * time.Second
		}
		if st.Metric != "manual" {
			at := now.Add(window)
			st.DecideAt = &at
		}
	} else if ab != nil && ab.Winner != "" {
		st.Kind, st.Phase, st.Winner = ab.Kind, "decided", ab.Winner
	}
	abb, _ := json.Marshal(st)
	err = db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SendingFinishPrepare(ctx, store.SendingFinishPrepareParams{CampaignID: cc.ID, Ab: abb, At: now}); err != nil {
			return err
		}
		s, err := q.SendingGetSend(ctx, cc.ID)
		if err != nil {
			return err
		}
		n := s.Recipients
		if err := q.SendingSetAudienceCount(ctx, store.SendingSetAudienceCountParams{ID: cc.ID, N: &n}); err != nil {
			return err
		}
		return q.SendingStageClear(ctx, cc.ID)
	})
	if err != nil {
		return err
	}
	e.kick(ctx, cc.ID)
	return nil
}

// fail stops a send that can't go out and says why.
func (e *Engine) fail(ctx context.Context, cc store.SendingCampaignContextRow, why string, now time.Time) error {
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cid := cc.ID
		if _, err := q.SendingCancelRemaining(ctx, store.SendingCancelRemainingParams{CampaignID: &cid, Reason: why}); err != nil {
			return err
		}
		if err := q.SendingMarkCompleted(ctx, store.SendingMarkCompletedParams{ID: cc.ID, Status: "failed", At: now}); err != nil {
			return err
		}
		if err := q.SendingSetPhase(ctx, store.SendingSetPhaseParams{CampaignID: cc.ID, Phase: "failed", Error: why, FinishedAt: &now, At: now}); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &cc.WorkspaceID, Action: "campaign.send_fail", TargetType: "campaign", TargetID: cc.ID.String(), TargetName: cc.Name,
			Result: audit.Failed, Reason: why, Actor: systemActor})
	})
}

// eligibility says why a contact doesn't receive the campaign ("" = it does).
func eligibility(c store.SendingAudiencePageRow, def audienceDef) string {
	switch {
	case c.Status == "unsubscribed":
		return "unsubscribed"
	case c.Status == "bounced":
		return "bounced"
	case c.Status == "complained":
		return "complained"
	case c.Status != "active", c.Suppressed:
		return "suppressed"
	case c.Paused:
		return "paused"
	case c.ExcludedList, c.ExcludedSegment:
		return "excluded"
	case slices.Contains(def.ExcludeTiers, c.Tier):
		return "tiers"
	}
	return ""
}

func (x *Excluded) add(why string) {
	switch why {
	case "unsubscribed":
		x.Unsubscribed++
	case "bounced":
		x.Bounced++
	case "complained":
		x.Complained++
	case "suppressed":
		x.Suppressed++
	case "paused":
		x.Paused++
	case "excluded":
		x.Excluded++
	case "tiers":
		x.Tiers++
	}
}

func abTestOf(raw json.RawMessage) *abTest {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var a abTest
	if json.Unmarshal(raw, &a) != nil || a.Kind == "" || len(a.Variants) < 2 {
		return nil
	}
	return &a
}

// hash01 maps an id to 0–99 with a salt, stable across runs.
func hash01(id uuid.UUID, salt string) int {
	h := fnv.New64a()
	_, _ = h.Write([]byte(salt))
	_, _ = h.Write(id[:])
	return int(h.Sum64() % 100)
}

// assignVariant puts a recipient in the A/B sample (with its variant) or
// in the rest, who wait for the winner (held).
func assignVariant(ab *abTest, contact uuid.UUID, sendAt map[string]*time.Time) (string, string, *time.Time) {
	if ab == nil {
		return "", "queued", nil
	}
	if ab.Winner != "" {
		return ab.Winner, "queued", nil
	}
	if ab.TestPercent <= 0 || ab.TestPercent >= 100 {
		return ab.Variants[0].Key, "queued", nil
	}
	if hash01(contact, "ab-sample") >= ab.TestPercent {
		return "", "held", nil
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte("ab-variant"))
	_, _ = h.Write(contact[:])
	v := ab.Variants[int(h.Sum32())%len(ab.Variants)]
	return v.Key, "queued", sendAt[v.Key]
}

// sendTimes are the local send times of a send-time test's variants.
func sendTimes(ab *abTest, loc *time.Location, now time.Time) map[string]*time.Time {
	out := map[string]*time.Time{}
	if ab == nil || ab.Kind != "send_time" {
		return out
	}
	local := now.In(loc)
	for _, v := range ab.Variants {
		var h, m int
		if _, err := fmt.Sscanf(v.SendTime, "%d:%d", &h, &m); err != nil {
			continue
		}
		t := time.Date(local.Year(), local.Month(), local.Day(), h, m, 0, 0, loc)
		if t.Before(now) {
			t = t.AddDate(0, 0, 1)
		}
		out[v.Key] = &t
	}
	return out
}

// snapshotContent captures what the send renders.
func (e *Engine) snapshotContent(cc store.SendingCampaignContextRow, ab *abTest) (sendContent, error) {
	tr := Tracking{Opens: true, Clicks: true}
	if len(cc.Tracking) > 0 {
		_ = json.Unmarshal(cc.Tracking, &tr)
	}
	base := e.URLs.Base()
	c := sendContent{Subject: cc.Subject, Preheader: cc.Preheader, HTML: media.Absolutize(cc.Html, base), Text: cc.PlainText,
		FromName: cc.SenderName, FromEmail: cc.SenderEmail, ReplyTo: firstNonEmpty(cc.ReplyTo, cc.SenderReplyTo), Tracking: tr, Links: map[string]int{}}
	if strings.TrimSpace(c.HTML) == "" && strings.TrimSpace(c.Text) == "" {
		return c, errors.New("the email has no content")
	}
	if ab != nil {
		for _, v := range ab.Variants {
			if v.HTML != "" {
				v.HTML = media.Absolutize(v.HTML, base)
			}
			c.Variants = append(c.Variants, v)
		}
	}
	return c, nil
}

// indexLinks prepares every variant once to number its links: variant A
// (the campaign) first, then content variants after it.
func (e *Engine) indexLinks(c sendContent) ([]TrackedLink, error) {
	out := []TrackedLink{}
	for _, v := range variantKeys(c) {
		t, err := Prepare(c.forVariant(v), c.Tracking)
		if err != nil {
			return nil, err
		}
		off := len(out)
		c.Links[v] = off
		for _, l := range t.Links {
			l.Position += off
			out = append(out, l)
		}
	}
	return out, nil
}

// variantKeys lists the variants with their own content: "" (the campaign,
// also variant A) and any content variant with its own HTML.
func variantKeys(c sendContent) []string {
	keys := []string{""}
	for _, v := range c.Variants {
		if v.HTML != "" && v.Key != "A" {
			keys = append(keys, v.Key)
		}
	}
	return keys
}

// forVariant is the content a variant renders.
func (c sendContent) forVariant(key string) Content {
	out := Content{Subject: c.Subject, Preheader: c.Preheader, HTML: c.HTML, Text: c.Text}
	for _, v := range c.Variants {
		if v.Key != key || key == "" {
			continue
		}
		if v.Subject != "" {
			out.Subject = v.Subject
		}
		if v.Preheader != "" {
			out.Preheader = v.Preheader
		}
		if v.HTML != "" {
			out.HTML, out.Text = v.HTML, ""
		}
	}
	return out
}

// router decides each recipient's provider during the audience build.
type router struct {
	e            *Engine
	set          routing.Set
	byID         map[uuid.UUID]store.Provider
	loc          *time.Location
	now          time.Time
	override     *uuid.UUID
	senderDomain string
	rates        map[uuid.UUID]providerRate
	planned      map[uuid.UUID]int64
}

func (r *router) rate(ctx context.Context, p store.Provider) providerRate {
	if pr, ok := r.rates[p.ID]; ok {
		return pr
	}
	pr := r.e.rateOf(ctx, r.e.Q, p, r.loc, r.now)
	r.rates[p.ID] = pr
	return pr
}

// route picks the provider: the campaign's override, else the Broadcasts
// rules (conditions, then the deterministic weighted split). A warming
// provider only takes the engagement tiers of its day, up to its daily
// cap; the rest goes to the warm provider of the same split.
func (r *router) route(ctx context.Context, c store.SendingAudiencePageRow, mbp string) (uuid.UUID, error) {
	if r.override != nil {
		if _, ok := r.byID[*r.override]; ok {
			r.planned[*r.override]++
			return *r.override, nil
		}
	}
	lists := make([]string, len(c.Lists))
	for i, l := range c.Lists {
		lists[i] = l.String()
	}
	d, ok := r.set.Resolve(routing.Recipient{Stream: providers.StreamBroadcast, Email: c.Email, Lists: lists, SenderDomain: r.senderDomain, MailboxProvider: mbp})
	if !ok {
		return uuid.Nil, errors.New("No sending provider is set up for Broadcasts. Add a route in Sending providers › Routing, then start the send again.")
	}
	pid := d.ProviderID
	p, ok := r.byID[pid]
	if !ok {
		return uuid.Nil, errors.New("The Broadcasts route points at a provider that is no longer connected. Fix the route, then start the send again.")
	}
	if pr := r.rate(ctx, p); pr.WarmingUp {
		allowed := len(pr.Tiers) == 0 || slices.Contains(pr.Tiers, c.Tier)
		capLeft := pr.Cap <= 0 || pr.SentToday+r.planned[pid] < pr.Cap
		if !allowed || !capLeft || pr.Paused {
			if alt, ok := r.warmAlternative(ctx, d.Rule, pid); ok {
				pid = alt
			}
		}
	}
	r.planned[pid]++
	return pid, nil
}

// warmAlternative is the largest share of the same rule that isn't warming
// up, else the rule's fallback.
func (r *router) warmAlternative(ctx context.Context, rule routing.Rule, not uuid.UUID) (uuid.UUID, bool) {
	best, bestW := uuid.Nil, -1
	for _, s := range rule.Splits {
		p, ok := r.byID[s.ProviderID]
		if s.ProviderID == not || !ok || !state.Caps(p).BulkAllowed() {
			continue
		}
		if r.rate(ctx, p).WarmingUp {
			continue
		}
		if s.Weight > bestW {
			best, bestW = s.ProviderID, s.Weight
		}
	}
	if bestW >= 0 {
		return best, true
	}
	if rule.Fallback != nil {
		if _, ok := r.byID[*rule.Fallback]; ok && *rule.Fallback != not {
			return *rule.Fallback, true
		}
	}
	return uuid.Nil, false
}

func routeOverride(raw json.RawMessage) *uuid.UUID {
	var o struct {
		ProviderID string `json:"provider_id"`
		Provider   string `json:"providerId"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &o) != nil {
		return nil
	}
	for _, s := range []string{o.ProviderID, o.Provider} {
		if id, err := uuid.Parse(s); err == nil {
			return &id
		}
	}
	return nil
}

func domainOf(email string) string {
	if i := strings.LastIndexByte(email, '@'); i >= 0 {
		return strings.ToLower(email[i+1:])
	}
	return ""
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			return x
		}
	}
	return ""
}

// uint64Of is a stable number from an id (simulation and sampling).
func uint64Of(id uuid.UUID) uint64 { return binary.BigEndian.Uint64(id[8:]) }
