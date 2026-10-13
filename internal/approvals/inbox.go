package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// ApprovalListItem is one row of the approval inbox.
type ApprovalListItem struct {
	ID          string `json:"id"`
	SubjectType string `json:"subjectType" enum:"campaign,automation"`
	CampaignID  string `json:"campaignId,omitempty"`
	// AutomationID is set for automation publish requests.
	AutomationID string `json:"automationId,omitempty"`
	Name         string `json:"name"`
	// Rules is "rules: audience ≥ 10,000 · warming-up provider".
	Rules    string         `json:"rules"`
	Author   ApprovalPerson `json:"author"`
	Audience int64          `json:"audience"`
	// AudienceNow is set when the audience moved after approval (15,002 → 17,110).
	AudienceNow *int64 `json:"audienceNow,omitempty"`
	Route       string `json:"route"`
	// Schedule: the snapshot's schedule, or when it was sent.
	ScheduleMode string     `json:"scheduleMode" enum:"now,at,best_time"`
	ScheduleAt   *time.Time `json:"scheduleAt,omitempty"`
	SentAt       *time.Time `json:"sentAt,omitempty"`
	RequestedAt  time.Time  `json:"requestedAt"`
	Status       string     `json:"status" enum:"pending,approved,changes_requested,rejected,invalidated,bypassed,sent,cancelled"`
	DecidedBy    string     `json:"decidedBy,omitempty"`
	DecidedAt    *time.Time `json:"decidedAt,omitempty"`
	// Note is the line under the status ("Marcus Chen: “exclude 180-day unengaged”").
	Note string `json:"note,omitempty"`
}

// ApprovalCounts feeds the inbox tabs.
type ApprovalCounts struct {
	Pending          int `json:"pending"`
	Approved         int `json:"approved"`
	ChangesRequested int `json:"changesRequested"`
	Rejected         int `json:"rejected"`
	Invalidated      int `json:"invalidated"`
	All              int `json:"all"`
}

// ApprovalFilterOption is an author or approver to filter by.
type ApprovalFilterOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ApprovalPolicySummary is the "Policy in force" card.
type ApprovalPolicySummary struct {
	Rules             []string `json:"rules"`
	ApprovalsRequired int      `json:"approvalsRequired"`
	Approvers         []string `json:"approvers"`
	MedianWaitMinutes *int     `json:"medianWaitMinutes,omitempty"`
}

// ApprovalList is the approval inbox.
type ApprovalList struct {
	Items     []ApprovalListItem     `json:"items"`
	Counts    ApprovalCounts         `json:"counts"`
	Authors   []ApprovalFilterOption `json:"authors"`
	Approvers []ApprovalFilterOption `json:"approvers"`
	Policy    ApprovalPolicySummary  `json:"policy"`
	// AllRequests is true for approvers (they see every request).
	AllRequests bool `json:"allRequests"`
	CanManage   bool `json:"canManage"`
}

// ListParams filter the inbox.
type ListParams struct {
	Status   string
	Author   string
	Approver string
	Days     int
}

// List returns the approval inbox: approvers see every request, others
// their own.
func (s *Service) List(ctx context.Context, ws *store.Workspace, p *authz.Principal, f ListParams) (ApprovalList, error) {
	out := ApprovalList{Items: []ApprovalListItem{}, Authors: []ApprovalFilterOption{}, Approvers: []ApprovalFilterOption{}, AllRequests: p.Can(authz.CampaignsApprove),
		CanManage: p.Can(authz.WorkspaceManage)}
	// Open requests are re-checked so the inbox never lists a stale approval.
	if open, err := s.Q.ApprovalOpenRequests(ctx); err == nil {
		for _, r := range open {
			if r.WorkspaceID == ws.ID && r.CampaignID != nil {
				_, _ = s.Revalidate(ctx, ws, *r.CampaignID)
			}
		}
	}
	days := f.Days
	since := time.Time{}
	if days > 0 {
		since = s.now().AddDate(0, 0, -days)
	}
	params := store.ApprovalListParams{WorkspaceID: ws.ID, AllRequests: out.AllRequests, Viewer: p.UserID, Since: since, HideReplaced: false}
	if id, err := uuid.Parse(f.Author); err == nil {
		params.Author = &id
	}
	if id, err := uuid.Parse(f.Approver); err == nil {
		params.Approver = &id
	}
	rows, err := s.Q.ApprovalList(ctx, params)
	if err != nil {
		return out, err
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	loc := state.Location(ws.Timezone)
	authors := map[string]bool{}
	for _, r := range rows {
		switch r.Status {
		case "pending":
			out.Counts.Pending++
		case "approved":
			out.Counts.Approved++
		case "changes_requested":
			out.Counts.ChangesRequested++
		case "rejected":
			out.Counts.Rejected++
		case "invalidated":
			out.Counts.Invalidated++
		}
		out.Counts.All++
		if r.RequestedBy != nil && !authors[r.RequestedBy.String()] {
			authors[r.RequestedBy.String()] = true
			out.Authors = append(out.Authors, ApprovalFilterOption{ID: r.RequestedBy.String(), Name: r.RequestedByName})
		}
		if f.Status != "" && f.Status != "all" && r.Status != f.Status {
			continue
		}
		var rules []ApprovalRuleMatch
		_ = json.Unmarshal(r.Rules, &rules)
		var route []SnapshotShare
		_ = json.Unmarshal(r.Route, &route)
		sched := readiness.ParseSchedule(r.Schedule)
		it := ApprovalListItem{ID: r.ID.String(), SubjectType: r.SubjectType, Name: r.SubjectName, Rules: rulesLine(rules), Route: routeText(route),
			ScheduleMode: sched.Mode, RequestedAt: r.RequestedAt, Status: r.Status, DecidedBy: r.DecidedByName, DecidedAt: r.DecidedAt,
			Author: ApprovalPerson{Name: r.RequestedByName, Email: r.RequestedByEmail, Initials: Initials(r.RequestedByName), Color: r.RequestedByColor}}
		it.Audience = r.AudienceCount
		if r.RequestedBy != nil {
			it.Author.ID = r.RequestedBy.String()
		}
		if r.CampaignID != nil {
			it.CampaignID = r.CampaignID.String()
		}
		if r.AutomationID != nil {
			it.AutomationID = r.AutomationID.String()
		}
		if t, err := time.Parse(time.RFC3339, sched.At); err == nil && sched.Mode != "now" {
			it.ScheduleAt = &t
		}
		if r.CampaignStartedAt != nil && (r.CampaignStatus == "sent" || r.CampaignStatus == "sending" || r.CampaignStatus == "paused") {
			it.SentAt = r.CampaignStartedAt
		}
		switch r.Status {
		case "changes_requested", "rejected":
			if r.LastComment != "" {
				it.Note = r.DecidedByName + ": “" + r.LastComment + "”"
			}
		case "invalidated":
			it.Note = r.InvalidatedReason
			var changes []ApprovalChange
			_ = json.Unmarshal(r.Changes, &changes)
			for _, ch := range changes {
				if ch.Field == "audience_count" {
					after := ch.After
					if i := strings.Index(after, " ("); i > 0 {
						after = after[:i]
					}
					if n, ok := parseCount(after); ok {
						it.AudienceNow = &n
						// The route the moved audience would take ("SES 17,110").
						it.Route = routeText(scaleRoute(route, n))
					}
				}
			}
		case "cancelled":
			it.Note = "cancelled by the author"
		}
		out.Items = append(out.Items, it)
	}
	_ = loc
	for _, a := range approversOf(ms) {
		out.Approvers = append(out.Approvers, ApprovalFilterOption{ID: a.ID.String(), Name: a.Name})
	}
	policy := ParsePolicy(ws.ApprovalPolicy)
	out.Policy = ApprovalPolicySummary{Rules: policy.RuleLabels(), ApprovalsRequired: policy.ApprovalsRequired, Approvers: []string{}, MedianWaitMinutes: s.medianWait(ctx, ws)}
	for _, a := range approversOf(ms) {
		out.Policy.Approvers = append(out.Policy.Approvers, a.Name)
	}
	return out, nil
}

func parseCount(s string) (int64, bool) {
	var n int64
	digits := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			n = n*10 + int64(r-'0')
			digits++
		case r == ',':
		default:
			return 0, false
		}
	}
	return n, digits > 0
}

// ApprovalCampaignView is the campaign part of a request.
type ApprovalCampaignView struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Subject   string `json:"subject"`
	Preheader string `json:"preheader"`
	From      string `json:"from"`
	Version   int    `json:"version"`
	// CurrentVersion is the campaign's version now: after edits it is past
	// the snapshot's ("What changed · v9 → v10").
	CurrentVersion int        `json:"currentVersion"`
	EditedBy       string     `json:"editedBy,omitempty"`
	EditedAt       *time.Time `json:"editedAt,omitempty"`
	// ScheduledAt is set while the campaign is scheduled.
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
}

// ApprovalReadinessView is the Send Readiness result a request was given on.
type ApprovalReadinessView struct {
	Verdict   string                     `json:"verdict" enum:"ready,warnings,blocked"`
	Passed    int                        `json:"passed"`
	Warnings  int                        `json:"warnings"`
	Blocks    int                        `json:"blocks"`
	Total     int                        `json:"total"`
	CheckedAt time.Time                  `json:"checkedAt"`
	Issues    []readiness.ReadinessCheck `json:"issues"`
}

// ApprovalAudienceView is the audience part of a request.
type ApprovalAudienceView struct {
	Count int64    `json:"count"`
	Lists []string `json:"lists"`
	// Excluded is "301 suppressed · 0 bounced in 30 days · role addresses".
	Excluded string `json:"excluded"`
	// Current is today's count and ChangePct how far it moved since the request.
	Current   int64   `json:"current"`
	ChangePct float64 `json:"changePct"`
}

// ApprovalRouteView is the route part of a request.
type ApprovalRouteView struct {
	SplitLabel string          `json:"splitLabel"`
	Shares     []ApprovalShare `json:"shares"`
}

// ApprovalShare is one provider of a request's route.
type ApprovalShare struct {
	ProviderID  string  `json:"providerId"`
	Name        string  `json:"name"`
	Short       string  `json:"short"`
	Label       string  `json:"label,omitempty"`
	Color       string  `json:"color" enum:"ses,postal,postmark,other"`
	Recipients  int64   `json:"recipients"`
	Cost        float64 `json:"cost"`
	PriceLabel  string  `json:"priceLabel"`
	Status      string  `json:"status"`
	StatusLabel string  `json:"statusLabel"`
}

// ApprovalScheduleView is the schedule part of a request.
type ApprovalScheduleView struct {
	Mode     string     `json:"mode" enum:"now,at,best_time"`
	At       *time.Time `json:"at,omitempty"`
	Timezone string     `json:"timezone"`
	Minutes  int        `json:"minutes"`
	// AutoPause is "Bounces 2% · complaints 0.10% per mailbox provider".
	AutoPause string `json:"autoPause"`
}

// ApprovalCommentView is one comment of the discussion.
type ApprovalCommentView struct {
	ID     string         `json:"id"`
	Author ApprovalPerson `json:"author"`
	Body   string         `json:"body"`
	At     time.Time      `json:"at"`
}

// ApprovalCan says what the viewer may do on a request.
type ApprovalCan struct {
	Decide       bool   `json:"decide"`
	DecideReason string `json:"decideReason,omitempty"`
	Comment      bool   `json:"comment"`
	Cancel       bool   `json:"cancel"`
	Remind       bool   `json:"remind"`
	Restore      bool   `json:"restore"`
	Schedule     bool   `json:"schedule"`
	IsAuthor     bool   `json:"isAuthor"`
}

// ApprovalDetail is the approval review screen.
type ApprovalDetail struct {
	ApprovalRequestView
	Name       string                      `json:"name"`
	Author     ApprovalPerson              `json:"author"`
	Approvers  []ApprovalPerson            `json:"approvers"`
	Campaign   *ApprovalCampaignView       `json:"campaign,omitempty"`
	Readiness  *ApprovalReadinessView      `json:"readiness,omitempty"`
	Audience   *ApprovalAudienceView       `json:"audience,omitempty"`
	Route      *ApprovalRouteView          `json:"route,omitempty"`
	Schedule   *ApprovalScheduleView       `json:"schedule,omitempty"`
	Automation *ApprovalAutomationSnapshot `json:"automation,omitempty"`
	// ChangesSince lists what changed since the previous request this one
	// follows (empty for a first request).
	ChangesSince []ApprovalChange `json:"changesSince"`
	// Next is the request that followed this one (after an invalidation).
	NextID   string                `json:"nextId,omitempty"`
	Comments []ApprovalCommentView `json:"comments"`
	Held     *ReviewHeld           `json:"held,omitempty"`
	Can      ApprovalCan           `json:"can"`
}

// Detail returns a request for its review screen.
func (s *Service) Detail(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (ApprovalDetail, error) {
	r, err := s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ApprovalDetail{}, api.MapError(err, "Approval request")
	}
	if !canSee(p, r.RequestedBy) {
		return ApprovalDetail{}, api.NotFound("Approval request")
	}
	if r.CampaignID != nil && (r.Status == "pending" || r.Status == "approved") {
		if changes, err := s.Revalidate(ctx, ws, *r.CampaignID); err != nil {
			return ApprovalDetail{}, err
		} else if len(changes) > 0 {
			if r, err = s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: id}); err != nil {
				return ApprovalDetail{}, err
			}
		}
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return ApprovalDetail{}, err
	}
	base := store.ApprovalRequest{ID: r.ID, WorkspaceID: r.WorkspaceID, SubjectType: r.SubjectType, CampaignID: r.CampaignID, AutomationID: r.AutomationID, RequestedBy: r.RequestedBy,
		RequestedAt: r.RequestedAt, Note: r.Note, Status: r.Status, MatchedRules: r.MatchedRules, ApprovalsRequired: r.ApprovalsRequired, Snapshot: r.Snapshot,
		InvalidatedAt: r.InvalidatedAt, InvalidatedReason: r.InvalidatedReason, Changes: r.Changes, BypassReason: r.BypassReason, DecidedAt: r.DecidedAt,
		CampaignVersion: r.CampaignVersion, ReadinessRunID: r.ReadinessRunID, DecidedBy: r.DecidedBy, InvalidatedBy: r.InvalidatedBy, RemindedAt: r.RemindedAt,
		PreviousID: r.PreviousID, InvalidatedFrom: r.InvalidatedFrom}
	view, err := s.requestView(ctx, ws, base, ms)
	if err != nil {
		return ApprovalDetail{}, err
	}
	snap := ParseSnapshot(r.Snapshot)
	out := ApprovalDetail{ApprovalRequestView: view, Name: r.SubjectName, Author: view.RequestedBy, Approvers: []ApprovalPerson{}, ChangesSince: []ApprovalChange{},
		Comments: []ApprovalCommentView{}, Automation: snap.Automation}
	out.Author.Role = firstNonEmptyStr(snap.AuthorRole, out.Author.Role)
	for _, a := range approversOf(ms) {
		out.Approvers = append(out.Approvers, s.person(a))
	}
	if r.CampaignID != nil {
		out.Campaign = &ApprovalCampaignView{ID: r.CampaignID.String(), Status: r.CampaignStatus, Subject: snap.Subject, Preheader: snap.Preheader, From: snap.From, Version: snap.Version,
			CurrentVersion: int(r.CampaignCurrentVersion)}
		if r.CampaignStatus == "scheduled" {
			out.Campaign.ScheduledAt = r.CampaignScheduledAt
		}
		if v, err := s.Q.ApprovalCampaignVersion(ctx, store.ApprovalCampaignVersionParams{CampaignID: *r.CampaignID, Version: int32(snap.Version)}); err == nil {
			t := v.SavedAt
			out.Campaign.EditedBy, out.Campaign.EditedAt = v.SavedByName, &t
		}
		rd := snap.Readiness
		out.Readiness = &ApprovalReadinessView{Verdict: firstNonEmptyStr(rd.Verdict, readiness.VerdictReady), Passed: rd.Passed, Warnings: rd.Warnings, Blocks: rd.Blocks,
			Total: rd.Total, CheckedAt: rd.RanAt, Issues: rd.Issues}
		excluded := []string{readiness.Count(snap.Suppressed) + " suppressed"}
		excluded = append(excluded, "0 bounced in 30 days")
		if snap.Bounced > 0 {
			excluded[1] = "recently bounced"
		}
		excluded = append(excluded, "role addresses")
		aud := &ApprovalAudienceView{Count: snap.AudienceCount, Lists: snap.AudienceLabels, Excluded: strings.Join(excluded, " · "), Current: snap.AudienceCount}
		if c, err := s.Readiness.Campaign(ctx, ws, *r.CampaignID); err == nil {
			if a, err := s.Readiness.Audience(ctx, ws, c.Audience); err == nil && !a.Empty && snap.Baseline() > 0 {
				aud.Current = snap.AudienceCount + a.Final - snap.Baseline()
				aud.ChangePct = math.Round(float64(a.Final-snap.Baseline())/float64(snap.Baseline())*1000) / 10
			}
		}
		out.Audience = aud
		rv := &ApprovalRouteView{SplitLabel: snap.SplitLabel, Shares: []ApprovalShare{}}
		for _, sh := range snap.Route {
			if sh.Recipients == 0 && len(snap.Route) > 1 {
				continue
			}
			rv.Shares = append(rv.Shares, ApprovalShare{ProviderID: sh.ProviderID, Name: sh.Name, Short: sh.Short, Label: sh.Label, Color: state.Color(sh.Adapter),
				Recipients: sh.Recipients, Cost: sh.Cost, PriceLabel: sh.PriceLabel, Status: sh.Status, StatusLabel: sh.StatusLabel})
		}
		out.Route = rv
		sv := readiness.ScheduleView(snap.Schedule, ws.Timezone)
		out.Schedule = &ApprovalScheduleView{Mode: sv.Mode, At: sv.At, Timezone: sv.Timezone, Minutes: snap.Minutes,
			AutoPause: fmt.Sprintf("Bounces %s · complaints %.2f%% per mailbox provider", readiness.Pct(snap.Thresholds.Bounce), snap.Thresholds.MailboxComplaint)}
		if h, err := s.Q.ApprovalOpenHold(ctx, *r.CampaignID); err == nil {
			out.Held = s.heldView(ctx, h)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
	}
	// What changed since the request this one follows.
	chain, err := s.Q.ApprovalChain(ctx, id)
	if err != nil {
		return out, err
	}
	if r.PreviousID != nil {
		if prev, err := s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: *r.PreviousID}); err == nil {
			var changes []ApprovalChange
			_ = json.Unmarshal(prev.Changes, &changes)
			if len(changes) == 0 {
				changes = Diff(ParseSnapshot(prev.Snapshot), currentFromSnapshot(snap), state.Location(ws.Timezone))
			}
			if changes != nil {
				out.ChangesSince = changes
			}
		}
	}
	if next, err := s.nextOf(ctx, id); err == nil && next != uuid.Nil {
		out.NextID = next.String()
	}
	comments, err := s.Q.ApprovalComments(ctx, chain)
	if err != nil {
		return out, err
	}
	for _, c := range comments {
		cv := ApprovalCommentView{ID: c.ID.String(), Body: c.Body, At: c.CreatedAt, Author: ApprovalPerson{Name: c.AuthorName, Initials: Initials(c.AuthorName), Color: c.AuthorColor}}
		if c.AuthorID != nil {
			cv.Author.ID = c.AuthorID.String()
		}
		out.Comments = append(out.Comments, cv)
	}
	isAuthor := r.RequestedBy != nil && *r.RequestedBy == p.UserID
	out.Can = ApprovalCan{IsAuthor: isAuthor, Comment: true, Cancel: (isAuthor || p.RoleKey == authz.RoleOwner) && (r.Status == "pending" || r.Status == "approved"),
		Remind: isAuthor && r.Status == "pending", Restore: r.Status == "invalidated" && out.NextID == "" && r.SubjectType == "campaign",
		Schedule: r.Status == "approved" && r.CampaignStatus != "scheduled" && (isAuthor || p.Can(authz.CampaignsSend))}
	switch {
	case !p.Can(authz.CampaignsApprove):
		out.Can.DecideReason = authz.DeniedMessage(authz.CampaignsApprove, approverNames(ms, nil, authz.CampaignsApprove))
	case isAuthor:
		out.Can.DecideReason = "You can't approve your own campaign."
	case !p.IsHuman():
		out.Can.DecideReason = "Only people can decide on approvals."
	case r.Status == "pending" || (r.Status == "invalidated" && out.NextID == ""):
		decided := false
		for _, d := range view.Decisions {
			if d.Approver.ID == p.UserID.String() {
				decided = true
			}
		}
		if decided && r.Status == "pending" {
			out.Can.DecideReason = "You already decided; waiting for another approver."
		} else {
			out.Can.Decide = true
		}
	default:
		out.Can.DecideReason = "This request is " + statusWord(r.Status) + "."
	}
	return out, nil
}

func (s *Service) nextOf(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	var next uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT id FROM approval_requests WHERE previous_id = $1 AND status <> 'bypassed' ORDER BY requested_at DESC LIMIT 1`, id).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, nil
	}
	return next, err
}

func currentFromSnapshot(s Snapshot) Current {
	return Current{Version: s.Version, ContentHash: s.ContentHash, Subject: s.Subject, Preheader: s.Preheader, SenderID: s.SenderID, From: s.From, Audience: s.Audience,
		AudienceNames: s.AudienceLabels, AudienceCount: s.AudienceCount, RouteKey: s.RouteKey, SplitLabel: s.SplitLabel, Schedule: s.Schedule}
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ApprovalPreview is the email a request covers.
type ApprovalPreview struct {
	Subject   string `json:"subject"`
	Preheader string `json:"preheader"`
	HTML      string `json:"html"`
	Version   int    `json:"version"`
}

// Preview renders the version a request covers with the personalization
// fallbacks ("there" for an empty first name).
func (s *Service) Preview(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (ApprovalPreview, error) {
	r, err := s.Q.ApprovalGet(ctx, store.ApprovalGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ApprovalPreview{}, api.MapError(err, "Approval request")
	}
	if !canSee(p, r.RequestedBy) || r.CampaignID == nil {
		return ApprovalPreview{}, api.NotFound("Approval request")
	}
	snap := ParseSnapshot(r.Snapshot)
	v, err := s.Q.ApprovalCampaignVersion(ctx, store.ApprovalCampaignVersionParams{CampaignID: *r.CampaignID, Version: int32(snap.Version)})
	html, subject, pre := "", snap.Subject, snap.Preheader
	if err == nil {
		html = v.Html
	} else if c, cerr := s.Readiness.Campaign(ctx, ws, *r.CampaignID); cerr == nil {
		html = c.Html
	}
	vars := liquid.Vars("", "", "", nil, map[string]string{"unsubscribe_url": "#unsubscribe", "preferences_url": "#preferences", "view_in_browser_url": "#view-in-browser"})
	if out, err := liquid.Render(html, vars); err == nil {
		html = out
	}
	return ApprovalPreview{Subject: subject, Preheader: pre, HTML: html, Version: snap.Version}, nil
}

// scaleRoute spreads a moved audience over a snapshot's route by its split;
// the rounding remainder goes to the largest share.
func scaleRoute(shares []SnapshotShare, total int64) []SnapshotShare {
	out := make([]SnapshotShare, len(shares))
	copy(out, shares)
	var sum, most int64
	largest := -1
	for i := range out {
		r := out[i].Recipients
		sum += r
		if largest < 0 || r > most {
			largest, most = i, r
		}
	}
	if sum <= 0 || largest < 0 {
		return out
	}
	var assigned int64
	for i := range out {
		out[i].Recipients = out[i].Recipients * total / sum
		assigned += out[i].Recipients
	}
	out[largest].Recipients += total - assigned
	return out
}
