package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/paidaccess"
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
	// IncludeSunset overrides the sunset policy for this campaign: contacts
	// whose Broadcasts the policy stopped receive it anyway (Review & send),
	// unless the sunset tier is excluded.
	IncludeSunset bool
	// Paid keeps only the paid members of the lists (at a tier and above)
	// or only the free subscribers: a paid-only campaign (Earn).
	Paid paidaccess.Audience
}

// sendsToStopped reports whether contacts the sunset policy stopped receive
// the campaign.
func (d audienceDef) sendsToStopped() bool {
	return d.IncludeSunset && !slices.Contains(d.ExcludeTiers, "sunset")
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
		IncludeSunset    bool     `json:"include_sunset"`
		IncludeSunset2   bool     `json:"includeSunset"`
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
		ExcludeSegments: ids(in.ExcludeSegments, in.ExcludeSegments2), ExcludeTiers: tiers, IncludeSunset: in.IncludeSunset || in.IncludeSunset2,
		Paid: paidaccess.Parse(raw)}
}

// Excluded counts the contacts of the audience who don't receive the send.
type Excluded struct {
	Suppressed   int64 `json:"suppressed"`
	Unsubscribed int64 `json:"unsubscribed"`
	Bounced      int64 `json:"bounced"`
	Complained   int64 `json:"complained"`
	Paused       int64 `json:"paused"`
	Role         int64 `json:"role"`
	Duplicates   int64 `json:"duplicates"`
	Sunset       int64 `json:"sunset"`
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
	// Source is the language the campaign is written in, and Languages its
	// language versions (content languages); empty when they don't apply.
	Source          string        `json:"source,omitempty"`
	DefaultLanguage string        `json:"default_language,omitempty"`
	Fallback        string        `json:"fallback,omitempty"`
	Offered         []string      `json:"offered,omitempty"`
	Languages       []langContent `json:"languages,omitempty"`
}

// abTest is campaigns.ab_test.
type abTest struct {
	Kind          string      `json:"kind"`
	Variants      []abVariant `json:"variants"`
	TestPercent   int         `json:"test_percent"`
	Metric        string      `json:"metric"`
	DurationHours int         `json:"duration_hours"`
	Winner        string      `json:"winner"`
	// The decision rule (ab.go, contract C14): the fewest recipients per
	// variant and the confidence a winner needs (500 and 0.95 when unset).
	MinSample  int64   `json:"min_sample,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
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
	// Rates are the variants' rates when the test was decided: the monitor
	// keeps showing what the decision was based on.
	Rates map[string]float64 `json:"rates,omitempty"`
	// Window overrides the test duration, in seconds (demo sends).
	Window int64 `json:"window,omitempty"`
	// Rule, Default and Decision are the test's decision rule, its default
	// variant and what it was decided on (ab.go).
	Rule     abRule      `json:"rule,omitzero"`
	Default  string      `json:"default,omitempty"`
	Decision *abDecision `json:"decision,omitempty"`
}

func parseAB(raw json.RawMessage) abState {
	var a abState
	_ = json.Unmarshal(raw, &a)
	return a
}

// Prepare builds a campaign's recipient list: the audience's members minus
// suppressed, unsubscribed, bounced, complained and paused contacts and the
// exclusions, each routed to a provider with its A/B variant. A saved cursor
// lets it stop and run again without sending anyone twice. It holds the
// send's lock (errBusy when another process holds it).
func (e *Engine) Prepare(ctx context.Context, campaignID uuid.UUID) error {
	unlock, err := e.lockSend(ctx, campaignID)
	if err != nil {
		return err
	}
	defer unlock()
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
		return e.fail(ctx, cc, english("fail.noAudience", nil), now)
	}
	if cc.SenderEmail == "" {
		return e.fail(ctx, cc, english("fail.noSender", nil), now)
	}
	// Company domains go out as their mailbox provider (Google Workspace
	// as Gmail…), so routing, throttles and the guard see them right.
	if e.Mailbox != nil {
		if _, err := e.Mailbox.ClassifyPending(ctx, cc.WorkspaceID, classifyBudget); err != nil && e.Log != nil {
			e.Log.WarnContext(ctx, "mailbox classification before the send", "campaign", campaignID, "err", err)
		}
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
		return e.fail(ctx, cc, english("fail.render", i18n.Args{"error": err.Error()}), now)
	}
	if snd.PrepareCursor == nil {
		if err := e.sendLanguages(ctx, cc.WorkspaceID, campaignID, &content); err != nil {
			return err
		}
		e.resolveStore(ctx, cc.WorkspaceID, &content)
	} else if len(snd.Content) > 0 {
		// A build that continues keeps the versions captured when it began.
		var prev sendContent
		if json.Unmarshal(snd.Content, &prev) == nil {
			content.Source, content.DefaultLanguage, content.Fallback, content.Offered, content.Languages = prev.Source, prev.DefaultLanguage, prev.Fallback, prev.Offered, prev.Languages
		}
	}
	chooser, multilingual := content.chooser()
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
		senderDomain: domainOf(cc.SenderEmail), sender: senderVerification(ctx, e.Q, cc.WorkspaceID, cc.SenderID),
		rates: map[uuid.UUID]providerRate{}, planned: map[uuid.UUID]int64{}}
	excluded := Excluded{}
	_ = json.Unmarshal(snd.Excluded, &excluded)
	cursor := uuid.Nil
	if snd.PrepareCursor != nil {
		cursor = *snd.PrepareCursor
	}
	// Aliases reaching a mailbox that another contact of the audience reaches
	// already (computed again when a stopped build continues: the pages
	// already built counted theirs).
	dups, err := e.duplicates(ctx, cc, def)
	if err != nil {
		return err
	}
	// A send-time test's variants go out at their time of day in each
	// recipient's own time zone (ab.go).
	sendAt := newSendTimes(ab, loc, now)
	var sc schedule
	_ = json.Unmarshal(cc.Schedule, &sc)
	bestTime := sc.Mode == "best_time"
	// A paid-only campaign: who has paid access is read once, as the build
	// starts (or continues); contacts it leaves out aren't recipients.
	keep, err := paidaccess.Load(ctx, e.Q, cc.WorkspaceID, def.Paid)
	if err != nil {
		return err
	}
	// Conditional blocks: each page's recipients are decided once, here
	// (their lists, tags, fields and tier as the build reads them).
	var planner Planner
	if e.Conditional != nil {
		if planner, err = e.Conditional.Plan(ctx, cc.WorkspaceID, campaignID); err != nil {
			return e.fail(ctx, cc, english("fail.conditional", i18n.Args{"error": err.Error()}), now)
		}
	}
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
		var hours map[uuid.UUID]int
		if bestTime {
			hours = e.usualOpenHours(ctx, rows, now)
		}
		var hidden map[uuid.UUID]string
		if planner != nil {
			ids := make([]uuid.UUID, 0, len(rows))
			for _, c := range rows {
				ids = append(ids, c.ID)
			}
			if hidden, err = e.hiddenFor(ctx, planner, ids); err != nil {
				return err
			}
		}
		for _, c := range rows {
			if !keep.Keeps(c.ID) {
				continue
			}
			if why := eligibility(c, def, dups[c.ID]); why != "" {
				excluded.add(why)
				continue
			}
			mbp := c.MailboxProvider
			if mbp == "" {
				mbp = e.mailboxFor(ctx, c.Email)
			}
			rt, err := r.route(ctx, c, mbp)
			if err != nil {
				return e.fail(ctx, cc, err.Error(), now)
			}
			pid := rt.provider
			variant, status, next := assignVariant(ab, c.ID, sendAt.in(c.Timezone))
			if h, ok := hours[c.ID]; ok && next == nil {
				// Best time: the hour the contact usually opens at, within
				// 24 hours; contacts who never opened go at the start.
				if t := nextAtHour(now, h); t.After(now) {
					next = &t
				}
			}
			// A random send order: the first minutes of the send are a fair
			// sample of the audience (migration 00652).
			order := rand.Int32() //nolint:gosec // an order, not a secret
			ml := content.pick(chooser, multilingual, c.Language)
			msgs = append(msgs, []any{newID(), cc.WorkspaceID, "broadcast", &campaignID, &c.ID, c.Email, variant, &pid, mbp, status, now, next, rt.note, rt.failover, order,
				ml.Language, ml.Contact, ml.Reason, hidden[c.ID]})
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
					"mailbox_provider", "status", "queued_at", "next_attempt_at", "route_note", "failover", "send_order", "language", "contact_language", "language_reason", "hidden_blocks"},
					pgx.CopyFromRows(msgs)); err != nil {
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
	// The A/B test's rule, default and decision time (ab.go).
	st, err := e.abStart(ctx, cc, ab, parseAB(snd.Ab), now)
	if err != nil {
		return err
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
		if err := emit(ctx, tx, cc.WorkspaceID, "campaign.completed", campaignData(cc.ID, cc.Name, "status", "failed", "reason", why)); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &cc.WorkspaceID, Action: "campaign.send_failed", TargetType: "campaign", TargetID: cc.ID.String(), TargetName: cc.Name,
			Result: audit.Failed, Reason: why, Actor: systemActor})
	})
}

// eligibility says why a contact doesn't receive the campaign ("" = it
// does). The exclusions are those of Send Readiness's audience count, in
// its order (suppressed, role, duplicate, stopped by the sunset policy,
// excluded tier), so the count confirmed on Review & send is what goes out;
// contacts who paused their emails in the preference center are left out
// as well. dup marks an alias of an earlier contact (duplicates).
func eligibility(c store.SendingAudiencePageRow, def audienceDef, dup bool) string {
	switch {
	case c.Gone:
		return "gone" // erased, or not this workspace's: not counted anywhere
	case c.ExcludedList, c.ExcludedSegment:
		return "excluded"
	case c.Status == "unsubscribed":
		return "unsubscribed"
	case c.Status == "bounced":
		return "bounced"
	case c.Status == "complained":
		return "complained"
	case c.Status != "active", c.Suppressed:
		return "suppressed"
	case c.RoleAddress || emailaddr.IsRole(c.Email):
		return "role"
	case dup:
		return "duplicate"
	case c.Paused:
		return "paused"
	case c.SunsetStopped && !def.sendsToStopped():
		return "sunset"
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
	case "role":
		x.Role++
	case "duplicate":
		x.Duplicates++
	case "sunset":
		x.Sunset++
	case "excluded":
		x.Excluded++
	case "tiers":
		x.Tiers++
	}
}

// duplicates finds the audience's aliases: a contact whose address reaches
// the same mailbox as another contact of the audience (plus-tags, dots in
// Gmail addresses) is left out, as Send Readiness counts duplicates. The
// contact kept is one the sunset policy didn't stop, then the oldest.
// Suppressed and role addresses don't count: they never receive the send.
func (e *Engine) duplicates(ctx context.Context, cc store.SendingCampaignContextRow, def audienceDef) (map[uuid.UUID]bool, error) {
	rows, err := e.Q.SendingAudienceAddresses(ctx, store.SendingAudienceAddressesParams{
		Lists: def.Lists, CampaignID: cc.ID, WorkspaceID: cc.WorkspaceID, ExcludeLists: def.ExcludeLists,
	})
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(rows))
	dups := map[uuid.UUID]bool{}
	for _, r := range rows {
		if r.RoleAddress || emailaddr.IsRole(r.Email) {
			continue
		}
		k := mailboxKey(r.Email)
		if seen[k] {
			dups[r.ID] = true
			continue
		}
		seen[k] = true
	}
	return dups, nil
}

// mailboxKey folds the addresses that reach the same mailbox: plus-tags
// anywhere, dots in Gmail addresses (the folding Send Readiness counts
// duplicates with).
func mailboxKey(email string) string {
	e := strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndexByte(e, '@')
	if at < 0 {
		return e
	}
	local, domain := e[:at], e[at+1:]
	if i := strings.IndexByte(local, '+'); i > 0 {
		local = local[:i]
	}
	if domain == "googlemail.com" {
		domain = "gmail.com"
	}
	if domain == "gmail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	return local + "@" + domain
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

// usualOpenHours is the hour of the day (UTC) each contact of a page opens
// at most often, over the last 180 days.
func (e *Engine) usualOpenHours(ctx context.Context, rows []store.SendingAudiencePageRow, now time.Time) map[uuid.UUID]int {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	out := map[uuid.UUID]int{}
	hs, err := e.Q.SendingUsualOpenHours(ctx, store.SendingUsualOpenHoursParams{ContactIds: ids, Since: now.AddDate(0, 0, -180)})
	if err != nil {
		e.log().WarnContext(ctx, "usual open hours", "err", err)
		return out
	}
	for _, h := range hs {
		if h.ContactID != nil {
			out[*h.ContactID] = int(h.Hour)
		}
	}
	return out
}

// nextAtHour is when a contact who usually opens at hour (UTC) gets a send
// starting at now: within that hour it goes at once, otherwise at the hour's
// next start, at most 24 hours later.
func nextAtHour(now time.Time, hour int) time.Time {
	u := now.UTC()
	t := time.Date(u.Year(), u.Month(), u.Day(), hour, 0, 0, 0, time.UTC)
	switch {
	case !now.Before(t) && now.Before(t.Add(time.Hour)):
		return now
	case t.Before(now):
		return t.Add(24 * time.Hour)
	}
	return t
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
// also variant A) and any content variant with its own HTML, then each
// language version's ("de|", "de|B").
func variantKeys(c sendContent) []string {
	keys := []string{""}
	for _, v := range c.Variants {
		if v.HTML != "" && v.Key != "A" {
			keys = append(keys, v.Key)
		}
	}
	return append(keys, languageVariantKeys(c)...)
}

// forVariant is the content a variant renders; "de|B" is variant B of the
// German version.
func (c sendContent) forVariant(key string) Content {
	if lang, variant := splitKey(key); lang != "" {
		if l := c.translation(lang); l != nil {
			return l.forVariant(variant)
		}
		key = variant
	}
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
	// sender is the campaign sender's verification per provider.
	sender  map[string]string
	rates   map[uuid.UUID]providerRate
	planned map[uuid.UUID]int64
}

func (r *router) rate(ctx context.Context, p store.Provider) providerRate {
	if pr, ok := r.rates[p.ID]; ok {
		return pr
	}
	pr := r.e.rateOf(ctx, r.e.Q, p, r.loc, r.now)
	r.rates[p.ID] = pr
	return pr
}

// vet checks whether a fallback can take one more recipient on top of what
// this build already gave it.
func (r *router) vet(ctx context.Context, p store.Provider) state.Vetting {
	load := state.Load{Broadcast: true, Messages: r.planned[p.ID] + 1, Sender: r.sender}
	if pr := r.rate(ctx, p); pr.WarmingUp {
		load.Warmup = &state.WarmupToday{Cap: pr.Cap, Sent: pr.SentToday, Paused: pr.Paused}
	}
	return state.Vet(p, load)
}

// routed is where one recipient goes and why (messages.route_note, shown on
// the contact's activity timeline).
type routed struct {
	provider uuid.UUID
	note     string
	failover bool
}

// route picks the provider: the campaign's override, else the Broadcasts
// rules (conditions, then the deterministic weighted split). A warming
// provider only takes the engagement tiers of its day, up to its daily
// cap; the rest goes to the warm provider of the same split. A provider
// that is down fails over only when the Broadcast policy allows it and the
// fallback is warm; otherwise the share stays and is held when it sends.
func (r *router) route(ctx context.Context, c store.SendingAudiencePageRow, mbp string) (routed, error) {
	if r.override != nil {
		if p, ok := r.byID[*r.override]; ok {
			r.planned[p.ID]++
			return routed{provider: p.ID, note: routing.RouteNote("campaignRoute", i18n.Args{"provider": state.Short(p.Name, p.Adapter)})}, nil
		}
	}
	lists := make([]string, len(c.Lists))
	for i, l := range c.Lists {
		lists[i] = l.String()
	}
	d, ok := r.set.Resolve(routing.Recipient{Stream: providers.StreamBroadcast, Email: c.Email, Lists: lists, SenderDomain: r.senderDomain, MailboxProvider: mbp})
	if !ok {
		return routed{}, sendFailure(english("fail.noRoute", nil))
	}
	pid := d.ProviderID
	p, ok := r.byID[pid]
	if !ok {
		return routed{}, sendFailure(english("fail.routeGone", nil))
	}
	out := routed{provider: pid}
	short := state.Short(p.Name, p.Adapter)
	pr := r.rate(ctx, p)
	// The notes are routing's messages, in English as stored
	// (routing.NoteIn reads them in other languages).
	switch {
	case d.Matched != "" && d.Rule.Name != "":
		out.note = routing.RouteNote("byRule", i18n.Args{"provider": short, "rule": d.Rule.Name})
	case d.Split && pr.WarmingUp:
		out.note = routing.RouteNote("splitWarmup", i18n.Args{"provider": short, "split": splitLabel(d.Rule.Splits), "day": pr.WarmupDay})
	case d.Split:
		out.note = routing.RouteNote("split", i18n.Args{"provider": short, "split": splitLabel(d.Rule.Splits)})
	}
	if pr.WarmingUp {
		allowed := len(pr.Tiers) == 0 || slices.Contains(pr.Tiers, c.Tier)
		capLeft := pr.Cap <= 0 || pr.SentToday+r.planned[pid] < pr.Cap
		if !allowed || !capLeft || pr.Paused {
			if alt, ok := r.warmAlternative(ctx, d.Rule, pid); ok {
				ap := r.byID[alt]
				why := "cap"
				if !allowed {
					why = "tier"
				}
				if pr.Paused {
					why = "paused"
				}
				out = routed{provider: alt, note: routing.RouteNote("instead", i18n.Args{"alt": state.Short(ap.Name, ap.Adapter), "provider": short, "why": why,
					"tier": c.Tier, "day": pr.WarmupDay})}
			}
		}
	}
	if cur := r.byID[out.provider]; !state.Available(cur) {
		target, hold := routing.Failover(r.set.Policy(providers.StreamBroadcast), d.Fallback, func(id uuid.UUID) routing.ProviderState {
			fp, ok := r.byID[id]
			if !ok || id == cur.ID {
				return routing.ProviderState{}
			}
			v := r.vet(ctx, fp)
			return routing.ProviderState{Available: v.Automatic(true), WarmingUp: v.Warming()}
		})
		if !hold && target != nil && r.set.Policy(providers.StreamBroadcast) != routing.PauseAndAsk {
			fb := r.byID[*target]
			out = routed{provider: fb.ID, failover: true,
				note: routing.RouteNote("failover", i18n.Args{"provider": state.Short(cur.Name, cur.Adapter), "fallback": state.Short(fb.Name, fb.Adapter)})}
		}
	}
	r.planned[out.provider]++
	return out, nil
}

// splitLabel is a split's weights: "90/10".
func splitLabel(splits []routing.Split) string {
	parts := make([]string, 0, len(splits))
	for _, sp := range splits {
		parts = append(parts, fmt.Sprint(sp.Weight))
	}
	return strings.Join(parts, "/")
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

// sendFailure is why a send can't go out, in words for the people who
// started it (the monitor shows it).
type sendFailure string

func (f sendFailure) Error() string { return string(f) }
