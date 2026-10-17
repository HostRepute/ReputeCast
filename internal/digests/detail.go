package digests

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// List returns the workspace's digests for Campaigns › Recurring.
func (s *Service) List(ctx context.Context, ws *store.Workspace) (DigestList, error) {
	rows, err := s.Q.DigestList(ctx, ws.ID)
	if err != nil {
		return DigestList{}, err
	}
	out := DigestList{Items: make([]DigestListItem, 0, len(rows))}
	for _, r := range rows {
		d := digestOf(r)
		it := DigestListItem{ID: r.ID.String(), Name: r.Name, State: r.State, Mode: r.Mode, Schedule: ParseSchedule(r.Schedule, ws.Timezone),
			Source: s.sourceRef(ctx, ws, d)}
		if r.OwnerID != nil {
			it.Owner = person(*r.OwnerID, r.OwnerName, r.OwnerColor)
		}
		issues, err := s.Issues(ctx, ws, r.ID)
		if err != nil {
			return DigestList{}, err
		}
		for _, is := range issues.Items {
			switch {
			case it.Next == nil && (is.Status == "collecting" || is.Status == "preparing" || is.Status == "prepared" || is.Status == "waiting" || is.Status == "held" || is.Status == "draft"):
				n := 0
				if num, err := s.Q.DigestNextNumber(ctx, r.ID); err == nil {
					n = int(num)
				}
				it.Next = &DigestIssueRef{ID: is.ID, Number: n, Status: is.Status, SendAt: is.SendAt, Posts: is.Posts,
					NeedsApproval: is.NeedsApproval || (is.Approval != nil && is.Approval.Status == "pending")}
			case it.Last == nil && is.Status == "sent" && is.Number != nil:
				it.Last = &DigestIssueRef{ID: is.ID, Number: *is.Number, Status: is.Status, SendAt: is.SendAt, Posts: is.Posts, HumanOpens: is.HumanOpens,
					HumanOpensDelta: is.HumanOpensDelta}
			}
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// DigestNextAcross is the soonest issue among a workspace's digests that
// are on, for the overview's "Next scheduled campaign".
type DigestNextAcross struct {
	DigestID, DigestName string
	Number               int
	SendAt               time.Time
	Timezone             string
	Posts                int
	Status               string
	NeedsApproval        bool
}

// NextAcross finds the workspace's soonest digest issue with a send time.
func (s *Service) NextAcross(ctx context.Context, ws *store.Workspace) (*DigestNextAcross, error) {
	list, err := s.List(ctx, ws)
	if err != nil {
		return nil, err
	}
	var best *DigestNextAcross
	for _, d := range list.Items {
		n := d.Next
		if d.State != "on" || n == nil || n.SendAt == nil {
			continue
		}
		if best == nil || n.SendAt.Before(best.SendAt) {
			best = &DigestNextAcross{DigestID: d.ID, DigestName: d.Name, Number: n.Number, SendAt: *n.SendAt, Timezone: d.Schedule.Timezone,
				Posts: n.Posts, Status: n.Status, NeedsApproval: n.NeedsApproval}
		}
	}
	return best, nil
}

func digestOf(r store.DigestListRow) store.Digest {
	return store.Digest{ID: r.ID, WorkspaceID: r.WorkspaceID, Name: r.Name, TemplateID: r.TemplateID, State: r.State, Mode: r.Mode, Source: r.Source, FeedUrl: r.FeedUrl,
		SiteID: r.SiteID, Categories: r.Categories, Tags: r.Tags, MaxAgeDays: r.MaxAgeDays, StartMode: r.StartMode, StartLatest: r.StartLatest, Schedule: r.Schedule,
		PrepareHours: r.PrepareHours, LateHours: r.LateHours, Archive: r.Archive, OwnerID: r.OwnerID, FirstOnAt: r.FirstOnAt, FeedTitle: r.FeedTitle}
}

func (s *Service) sourceRef(ctx context.Context, ws *store.Workspace, d store.Digest) DigestSourceRef {
	ref := DigestSourceRef{Kind: d.Source, Label: hostOf(d.FeedUrl), Categories: nonNil(d.Categories)}
	if d.Source == "wordpress" && d.SiteID != nil {
		if site, err := s.siteRow(ctx, ws.ID, *d.SiteID); err == nil && site != nil {
			ref.Label = hostOf(firstNonEmpty(site.SiteUrl, site.Address))
		}
	}
	return ref
}

// Detail is a digest with what its overview and settings show.
func (s *Service) Detail(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (DigestDetail, error) {
	r, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return DigestDetail{}, api.MapError(err, "Digest")
	}
	d := r.Digest
	sched := ParseSchedule(d.Schedule, ws.Timezone)
	out := DigestDetail{ID: d.ID.String(), Name: d.Name, State: d.State, Mode: d.Mode, Archive: d.Archive, Source: d.Source, FeedURL: d.FeedUrl,
		Schedule: sched, PrepareHours: int(d.PrepareHours), LateHours: int(d.LateHours), OwnerSince: d.OwnerSince, FirstOnAt: d.FirstOnAt,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt, CategoryOptions: []string{}, TagOptions: []string{}, SendHelpers: []string{}, EditHelpers: []string{},
		PauseHelpers: []string{}}
	out.Filters = DigestFilters{Categories: nonNil(d.Categories), Tags: nonNil(d.Tags), StartMode: d.StartMode, StartLatest: int(d.StartLatest)}
	if d.MaxAgeDays != nil {
		n := int(*d.MaxAgeDays)
		out.Filters.MaxAgeDays = &n
	}
	if d.OwnerID != nil {
		out.Owner = person(*d.OwnerID, r.OwnerName, r.OwnerColor)
	}
	if d.CreatedBy != nil {
		out.CreatedBy = person(*d.CreatedBy, r.CreatedByName, "")
	}
	if d.SiteID != nil {
		if site, err := s.site(ctx, ws, *d.SiteID); err == nil {
			out.Site = site
		}
	}
	ms, err := s.members(ctx, ws.ID)
	if err != nil {
		return DigestDetail{}, err
	}
	out.SendHelpers = helpers(ms, authz.CampaignsSend, d.OwnerID)
	out.EditHelpers = helpers(ms, authz.CampaignsEdit, d.OwnerID)
	out.PauseHelpers = helpers(ms, authz.CampaignsPause, d.OwnerID)
	if d.State == "paused" {
		out.Pause = pauseOf(r)
	}
	// The template: audience, sender, limits.
	tpl, err := s.template(ctx, d)
	if err != nil {
		return DigestDetail{}, err
	}
	block := DefaultBlock()
	if tpl != nil {
		out.TemplateID = tpl.ID.String()
		out.TemplateVersion = int(tpl.Version)
		out.TemplateSavedAt = tpl.AutosavedAt
		block, _ = BlockOf(tpl.Mjml)
		if tpl.SenderID != nil {
			out.Sender = &DigestSender{ID: tpl.SenderID.String(), Name: tpl.SenderName, Email: tpl.SenderEmail}
		}
		out.Thresholds = thresholdsOf(tpl.Thresholds, ws.SendThresholds)
		if out.Audience, err = s.audienceOf(ctx, ws, tpl.Audience); err != nil {
			return DigestDetail{}, err
		}
		if out.Route, err = s.routeOf(ctx, ws, tpl.RouteOverride, s.now()); err != nil {
			return DigestDetail{}, err
		}
	} else {
		out.Thresholds = thresholdsOf(nil, ws.SendThresholds)
		out.Audience = DigestAudience{ListIDs: []string{}, SegmentIDs: []string{}, Empty: true}
		out.Route = DigestRoute{Shares: []DigestRouteShare{}}
	}
	// Who approves, and what every issue matches.
	out.Approvals = s.approvalsPreview(ctx, ws, d, out.Audience, out.Route, ms)
	// The feed, and its terms.
	all, err := s.items(ctx, d, 400, false)
	if err != nil {
		return DigestDetail{}, err
	}
	out.Feed = s.feedStatus(ctx, ws, d, all)
	out.CategoryOptions, out.TagOptions = terms(all, d)
	// The next issue.
	if next, err := s.nextIssue(ctx, ws, d, all, block, out.Audience); err != nil {
		return DigestDetail{}, err
	} else {
		out.Next = next
	}
	if d.State != "on" {
		if t, _, _ := s.nextTimes(ctx, d, s.now()); t != nil {
			out.NextSend = t
		}
	} else if out.Next != nil {
		out.NextSend = out.Next.SendAt
	}
	if last, err := s.Q.DigestLastSentIssue(ctx, d.ID); err == nil && last.Number != nil {
		out.SentIssues = int(*last.Number)
	}
	if first, err := s.firstSent(ctx, d.ID); err == nil {
		out.FirstSentAt = first
	}
	return out, nil
}

func (s *Service) firstSent(ctx context.Context, id uuid.UUID) (*time.Time, error) {
	r, err := s.Q.DigestFirstSent(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	switch {
	case r.SentAt != nil:
		return r.SentAt, nil
	case r.StartedAt != nil:
		return r.StartedAt, nil
	}
	return r.SendAt, nil
}

func pauseOf(r store.DigestGetRow) *DigestPause {
	p := &DigestPause{At: r.Digest.PausedAt, By: r.PausedByName, Reason: firstNonEmpty(r.Digest.PauseReason, "person"), Note: r.Digest.PauseNote, Notified: []string{}, Skips: []DigestFeedSkip{}}
	var detail struct {
		From     string   `json:"from"`
		To       string   `json:"to"`
		Left     bool     `json:"left"`
		Notified []string `json:"notified"`
		Skips    []struct {
			SendAt time.Time `json:"send_at"`
			Kind   string    `json:"kind"`
		} `json:"skips"`
	}
	_ = json.Unmarshal(r.Digest.PauseDetail, &detail)
	p.From, p.To, p.Left = detail.From, detail.To, detail.Left
	if detail.Notified != nil {
		p.Notified = detail.Notified
	}
	for _, sk := range detail.Skips {
		p.Skips = append(p.Skips, DigestFeedSkip{SendAt: sk.SendAt, Kind: sk.Kind})
	}
	sort.Slice(p.Skips, func(i, j int) bool { return p.Skips[i].SendAt.Before(p.Skips[j].SendAt) })
	return p
}

func thresholdsOf(campaign, workspace json.RawMessage) DigestThresholds {
	t := readiness.ParseThresholds(campaign, workspace)
	custom := len(campaign) > 0 && string(campaign) != "{}" && string(campaign) != "null"
	return DigestThresholds{Bounce: t.Bounce, Complaint: t.MailboxComplaint, Scope: t.Scope, Custom: custom}
}

// audienceOf counts who an issue goes to now.
func (s *Service) audienceOf(ctx context.Context, ws *store.Workspace, raw json.RawMessage) (DigestAudience, error) {
	def := readiness.ParseAudience(raw)
	out := DigestAudience{ListIDs: nonNil(def.Lists), SegmentIDs: nonNil(def.Segments), Empty: def.Empty()}
	if def.Empty() || s.checks() == nil {
		return out, nil
	}
	a, err := s.checks().Audience(ctx, ws, raw)
	if err != nil {
		return out, err
	}
	out.Label, out.Base, out.Suppressed, out.Role, out.Duplicates, out.Stopped, out.Final = a.BaseLabel, a.Base, a.Suppressed, a.Role, a.Duplicates, a.Stopped, a.Final
	return out, nil
}

// routeOf is the route issues take: the digest's own split, or the
// Broadcasts stream's, with each provider's state and warm-up.
func (s *Service) routeOf(ctx context.Context, ws *store.Workspace, override json.RawMessage, now time.Time) (DigestRoute, error) {
	out := DigestRoute{Shares: []DigestRouteShare{}}
	set, err := routing.Load(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	splits := []routing.Split{}
	if o := readiness.ParseRouteOverride(override); o != nil {
		splits, out.Override = o.Splits, true
	} else if def, ok := set.Default(providers.StreamBroadcast); ok {
		splits = def.Splits
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	total := 0
	for _, sp := range splits {
		total += max(sp.Weight, 0)
	}
	loc := location(ws.Timezone)
	for _, sp := range splits {
		p, ok := byID[sp.ProviderID]
		if !ok || sp.Weight <= 0 {
			continue
		}
		sh := DigestRouteShare{ProviderID: p.ID.String(), Name: p.Name, Short: state.Short(p.Name, p.Adapter), Adapter: p.Adapter, Status: p.Status}
		if total > 0 {
			sh.Percent = math.Round(float64(sp.Weight)*1000/float64(total)) / 10
		}
		if plan, err := s.Q.WarmupActiveForProvider(ctx, p.ID); err == nil {
			day := min(max(state.WarmupDay(plan.StartDate, now, loc), 1), int(plan.Days))
			sh.WarmupDay, sh.WarmupDays = day, int(plan.Days)
			end := plan.StartDate.AddDate(0, 0, int(plan.Days)-1)
			sh.WarmupEnds = &end
			if p.Status == state.Healthy {
				sh.Status = state.WarmingUp
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
		out.Shares = append(out.Shares, sh)
	}
	return out, nil
}

// approvalsPreview says up front which approval rules every issue will
// match: the audience as it stands and the route, sent by the owner (or by
// the person who would turn it on).
func (s *Service) approvalsPreview(ctx context.Context, ws *store.Workspace, d store.Digest, a DigestAudience, r DigestRoute, ms []member) DigestApprovals {
	out := DigestApprovals{Rules: []DigestRuleMatch{}, Approvers: []string{}}
	policy := approvals.ParsePolicy(ws.ApprovalPolicy)
	shares := []approvals.RuleShare{}
	until := map[string]*time.Time{}
	for _, sh := range r.Shares {
		recipients := int64(math.Round(float64(a.Final) * sh.Percent / 100))
		shares = append(shares, approvals.RuleShare{Short: sh.Short, Recipients: recipients, WarmingUp: sh.Status == state.WarmingUp, WarmupDay: sh.WarmupDay,
			WarmupDays: sh.WarmupDays})
		if sh.WarmupEnds != nil {
			end := sh.WarmupEnds.AddDate(0, 0, 1)
			until[sh.Short] = &end
		}
	}
	ownerName, ownerRole, canSend := "", "", true
	if d.OwnerID != nil {
		for _, m := range ms {
			if m.ID == *d.OwnerID {
				ownerName, ownerRole, canSend = m.Name, m.RoleName, m.Perms.Has(authz.CampaignsSend)
			}
		}
	}
	facts := approvals.IssueRulesFacts(a.Final, shares, canSend, ownerName, ownerRole, s.now())
	for _, m := range policy.Evaluate(facts) {
		rm := DigestRuleMatch{Key: m.Key, Label: m.Label, Short: m.Short}
		if m.Key == approvals.RuleAudienceMin {
			rm.Threshold = policy.AudienceMin.N
		}
		if m.Key == approvals.RuleWarmingProvider {
			for short, t := range until {
				if strings.Contains(m.Label, short) {
					rm.Until = t
				}
			}
		}
		out.Rules = append(out.Rules, rm)
	}
	out.Needed = len(out.Rules) > 0
	for _, m := range ms {
		if m.Perms.Has(authz.CampaignsApprove) && (d.OwnerID == nil || m.ID != *d.OwnerID) {
			out.Approvers = append(out.Approvers, m.Name)
		}
	}
	return out
}

// needsApproval is whether the next issue would need approval.
func (s *Service) needsApproval(ctx context.Context, ws *store.Workspace, d store.Digest) bool {
	tpl, err := s.template(ctx, d)
	if err != nil || tpl == nil {
		return false
	}
	a, err := s.audienceOf(ctx, ws, tpl.Audience)
	if err != nil {
		return false
	}
	r, err := s.routeOf(ctx, ws, tpl.RouteOverride, s.now())
	if err != nil {
		return false
	}
	ms, err := s.members(ctx, ws.ID)
	if err != nil {
		return false
	}
	return s.approvalsPreview(ctx, ws, d, a, r, ms).Needed
}

// feedStatus is the feed as last read, with what waits for the next issue.
func (s *Service) feedStatus(ctx context.Context, ws *store.Workspace, d store.Digest, all []item) DigestFeedStatus {
	out := DigestFeedStatus{Status: d.FeedStatus, URL: s.feedAddress(ctx, d), Kind: d.FeedKind, Title: d.FeedTitle, Items: int(d.FeedItems), Bytes: int(d.FeedBytes),
		LastReadAt: d.LastReadAt, LastReadVia: d.LastReadVia, LastOkAt: d.LastOkAt, MovedTo: d.FeedMovedTo}
	if d.FeedStatus == "unreachable" || d.FeedStatus == "invalid" {
		out.Error = decodeFeedError(d.FeedError)
	}
	since := time.Time{}
	if d.FirstOnAt != nil {
		since = *d.FirstOnAt
	}
	if last, err := s.Q.DigestLastSentIssue(ctx, d.ID); err == nil && last.StartedAt != nil {
		since = *last.StartedAt
	}
	for _, it := range all {
		switch {
		case it.Status == "new":
			out.Waiting++
		case it.Status == "left_out" && it.when().After(since):
			out.LeftOut++
		}
	}
	return out
}

// terms are the categories and tags of the feed's items, most used first,
// and the ones picked.
func terms(all []item, d store.Digest) ([]string, []string) {
	count := func(get func(item) []string, picked []string) []string {
		n := map[string]int{}
		order := []string{}
		for _, it := range all {
			for _, t := range get(it) {
				key := strings.ToLower(t)
				if _, ok := n[key]; !ok {
					order = append(order, t)
				}
				n[key]++
			}
		}
		for _, p := range picked {
			if _, ok := n[strings.ToLower(p)]; !ok {
				order = append(order, p)
				n[strings.ToLower(p)] = 0
			}
		}
		sort.SliceStable(order, func(i, j int) bool { return n[strings.ToLower(order[i])] > n[strings.ToLower(order[j])] })
		return order
	}
	cats := count(func(it item) []string { return it.Categories }, d.Categories)
	tags := count(func(it item) []string { return it.Tags }, d.Tags)
	return cats, tags
}

// nextIssue is the issue being collected or prepared, as it stands.
func (s *Service) nextIssue(ctx context.Context, ws *store.Workspace, d store.Digest, all []item, block DigestFeedBlock, aud DigestAudience) (*DigestNextIssue, error) {
	issue, err := s.Q.DigestIssueOpen(ctx, d.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	number, err := s.Q.DigestNextNumber(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	out := &DigestNextIssue{ID: issue.ID.String(), Number: int(number), Status: issue.Status, SendAt: issue.SendAt, PrepareAt: issue.PrepareAt, Deadline: issue.Deadline,
		Fit: block.Max, Recipients: aud.Final, Items: []DigestFeedItem{}, Subjects: []DigestSubject{}, FeedTries: feedTries(issue.FeedTries),
		HoldReason: issue.HoldReason, HeldAt: issue.HeldAt, PreparedAt: issue.PreparedAt}
	if issue.CampaignID != nil {
		out.CampaignID = issue.CampaignID.String()
	}
	if len(out.FeedTries) > 0 {
		next := out.FeedTries[len(out.FeedTries)-1].At.Add(FeedRetry)
		out.NextTry = &next
	}
	// The posts: a prepared issue's own, else the new ones as they stand.
	var posts []BlockItem
	if issue.Status != "collecting" && issue.Status != "preparing" {
		rows, err := s.Q.DigestIssueItems(ctx, issue.ID)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			posts = append(posts, blockItemOf(r))
			for _, it := range all {
				if it.ID == r.ID {
					out.Items = append(out.Items, feedItem(it, block))
				}
			}
		}
		if issue.CampaignID != nil {
			if st, err := s.Q.DigestCampaignState(ctx, *issue.CampaignID); err == nil && st.AudienceCount != nil && *st.AudienceCount > 0 {
				out.Recipients = *st.AudienceCount
			}
			if r, err := s.Q.DigestOpenApproval(ctx, issue.CampaignID); err == nil {
				out.Approval = &DigestIssueApproval{ID: r.ID.String(), Status: r.Status, RequestedAt: r.RequestedAt, DecidedAt: r.DecidedAt, Rules: nonNil(r.MatchedRules)}
			}
			if ch := s.checks(); ch != nil {
				if run, err := ch.Latest(ctx, *issue.CampaignID); err == nil && run != nil {
					rd := &DigestIssueReadiness{RunID: run.ID, Verdict: run.Verdict, Passed: run.Counts.Passed, Warnings: run.Counts.Warnings, Blocks: run.Counts.Blocks,
						RanAt: run.RanAt, Blocking: []string{}}
					for _, c := range run.Checks {
						if c.Status == readiness.Block {
							rd.Blocking = append(rd.Blocking, firstNonEmpty(c.FlatTitle, c.Title))
						}
					}
					out.Readiness = rd
				}
			}
		}
	} else {
		fresh := newItems(all)
		bis := make([]BlockItem, 0, len(fresh))
		for _, it := range fresh {
			bis = append(bis, blockItem(it.DigestItemsRow))
		}
		shown, _ := block.Select(bis)
		posts = shown
		for _, sh := range shown {
			for _, it := range fresh {
				if it.ID.String() == sh.ID {
					out.Items = append(out.Items, feedItem(it, block))
				}
			}
		}
	}
	out.Status = displayStatus(issue.Status, out.Approval)
	out.Posts = len(posts)
	// The subject as it stands, and with 1, 3 and 6 items.
	tpl, err := s.template(ctx, d)
	if err != nil || tpl == nil {
		return out, err
	}
	loc := location(ParseSchedule(d.Schedule, ws.Timezone).Timezone)
	data := IssueData{FeedTitle: d.FeedTitle, FeedURL: d.FeedUrl, Number: int(number), Items: posts, Loc: loc}
	if issue.SendAt != nil {
		data.Date = *issue.SendAt
	}
	out.Subject, _ = RenderIssue(tpl.Subject, data, false)
	out.Preheader, _ = RenderIssue(tpl.Preheader, data, false)
	if issue.CampaignID != nil && issue.Status != "collecting" {
		if c, err := s.Q.DigestCampaignTemplate(ctx, *issue.CampaignID); err == nil {
			out.Subject, out.Preheader = c.Subject, c.Preheader
		}
	}
	out.Subjects = subjects(tpl.Subject, d, all, block, int(number), loc)
	return out, nil
}

// subjects renders a subject with 1, 3 and 6 of the newest posts that pass
// the filters (sent or not), so "…and 0 more" is caught early.
func subjects(subject string, d store.Digest, all []item, block DigestFeedBlock, number int, loc *time.Location) []DigestSubject {
	pool := []BlockItem{}
	for _, it := range all {
		if it.Status == "new" || it.Status == "sent" {
			pool = append(pool, blockItem(it.DigestItemsRow))
		}
	}
	out := []DigestSubject{}
	for _, n := range []int{1, 3, 6} {
		items := pool
		if len(items) > n {
			items = items[:n]
		}
		for len(items) < n && len(items) > 0 {
			items = append(items, items[len(items)-1])
		}
		text, err := RenderIssue(subject, IssueData{FeedTitle: d.FeedTitle, Number: number, Items: items, Loc: loc}, false)
		if err != nil {
			text = subject
		}
		text = strings.TrimSpace(text)
		out = append(out, DigestSubject{Items: n, Text: text, ZeroMore: n == 1 && SaysZeroMore(text)})
	}
	return out
}

// FeedPreview is the newest items with their badges, and what the older
// ones did.
func (s *Service) FeedPreview(ctx context.Context, ws *store.Workspace, id uuid.UUID, limit int) (DigestFeedPreview, error) {
	r, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return DigestFeedPreview{}, api.MapError(err, "Digest")
	}
	all, err := s.items(ctx, r.Digest, 400, false)
	if err != nil {
		return DigestFeedPreview{}, err
	}
	block := DefaultBlock()
	if tpl, err := s.template(ctx, r.Digest); err == nil && tpl != nil {
		block, _ = BlockOf(tpl.Mjml)
	}
	if limit <= 0 {
		limit = 6
	}
	out := DigestFeedPreview{Items: []DigestFeedItem{}, Total: len(all), OlderSent: []DigestSentCount{}}
	sent := map[int]int{}
	for i, it := range all {
		if i < limit {
			out.Items = append(out.Items, feedItem(it, block))
			continue
		}
		switch it.Status {
		case "sent":
			sent[int(it.SentIn)]++
		case "left_out":
			out.OlderLeftOut++
		case "before_start":
			out.OlderBefore++
		default:
			out.OlderNew++
		}
	}
	for n, c := range sent {
		out.OlderSent = append(out.OlderSent, DigestSentCount{Number: n, Count: c})
	}
	sort.Slice(out.OlderSent, func(i, j int) bool { return out.OlderSent[i].Number > out.OlderSent[j].Number })
	return out, nil
}
