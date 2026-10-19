// Package campaigns implements campaigns: the list and summary, drafts and
// the composer's storage, versions, scheduling and sending state.
package campaigns

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// Service holds the campaigns domain dependencies.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
	// Templates resolves templates when a draft starts from one.
	Templates *templates.Service
	// Mail sends test emails through the System route (App.Mail, read at call time).
	Mail func() mailer.Sender
	// URLs builds public links (unsubscribe, preferences, view online).
	URLs publicurl.Builder
	// Checker runs the content checks (blocklist and link probing plug in here).
	Checker *contentcheck.Checker
	// AfterChange runs once a save of the content, audience, sender or
	// settings committed (the approvals domain re-checks open approvals).
	AfterChange ChangeHook
	// Translations handles language versions (?language= on the content
	// endpoints); nil without content languages.
	Translations Translations
	// Extras fill in test emails' poll links and cut them to a contact's
	// version of conditional content (conditional.go); nil leaves them.
	Extras Extras
}

// ChangeHook is told which campaign changed. The save already committed, so
// a hook handles its own errors.
type ChangeHook func(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID)

func (s *Service) changed(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
	if s.AfterChange != nil {
		s.AfterChange(ctx, ws, id)
	}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// RouteShare is one sending provider's part of a campaign.
type RouteShare struct {
	ProviderID string `json:"providerId"`
	Provider   string `json:"provider" example:"Amazon SES"`
	// Short is the label used in badges ("SES", "Postal").
	Short   string `json:"short" example:"SES"`
	Adapter string `json:"adapter" example:"ses"`
	// Percent of the planned audience routed to this provider (unsent campaigns).
	Percent float64 `json:"percent"`
	// Planned recipients and, once sending started, how many were sent.
	Planned int64 `json:"planned"`
	Sent    int64 `json:"sent"`
}

// CampaignListItem is one row of the campaigns list.
type CampaignListItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	// Blocked is true for drafts whose latest Send Readiness run is Blocked.
	Blocked bool `json:"blocked"`
	// Audience is a readable label ("Weekly Digest", "All subscribers"); empty when not set.
	Audience    string     `json:"audience"`
	Recipients  *int64     `json:"recipients,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	UpdatedBy   string     `json:"updatedBy"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	// Rates in percent, set once the campaign has sent anything.
	Delivered  *float64 `json:"delivered,omitempty"`
	HumanOpens *float64 `json:"humanOpens,omitempty"`
	Clicks     *float64 `json:"clicks,omitempty"`
	// Routes lists the sending providers; empty for a draft without an
	// audience (it will use the Broadcasts route).
	Routes []RouteShare `json:"routes"`
	// Origin is the importer source for history pulled in by migration ("brevo").
	Origin string `json:"origin,omitempty"`
	// Notes are short annotations: "A/B subject, winner B", "auto-paused once, Yahoo deferrals".
	Notes []string `json:"notes"`
	// Readiness of unsent campaigns: verdict and number of warnings or blocks.
	Readiness *ReadinessSummary `json:"readiness,omitempty"`
	// Digest is set for an issue of a recurring digest: the Digest badge
	// links to it, and the issue's status reads in place of the campaign's.
	Digest *CampaignDigestRef `json:"digest,omitempty"`
	// PaidOnly is a campaign for paid members only (Earn): its "Paid only"
	// badge.
	PaidOnly bool `json:"paidOnly"`
}

// CampaignDigestRef is the digest an issue belongs to (internal/digests).
type CampaignDigestRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// IssueStatus is the issue's: prepared, waiting (for approval), held,
	// draft, sending, sent…
	IssueStatus string     `json:"issueStatus" enum:"collecting,preparing,prepared,waiting,held,draft,sending,sent,skipped,not_sent"`
	Posts       int        `json:"posts"`
	PreparedAt  *time.Time `json:"preparedAt,omitempty"`
	SendAt      *time.Time `json:"sendAt,omitempty"`
	// ApprovalID is the issue's open approval request.
	ApprovalID string `json:"approvalId,omitempty"`
}

// ReadinessSummary is the latest Send Readiness verdict.
type ReadinessSummary struct {
	Verdict string `json:"verdict" enum:"ready,warnings,blocked"`
	// Issues are the blocking issues of a blocked campaign, otherwise its
	// warnings.
	Issues int64 `json:"issues"`
}

// CampaignStatusCounts feeds the All / Drafts / Scheduled / Sending / Sent tabs.
type CampaignStatusCounts struct {
	All       int64 `json:"all"`
	Drafts    int64 `json:"drafts"`
	Scheduled int64 `json:"scheduled"`
	Sending   int64 `json:"sending"`
	Sent      int64 `json:"sent"`
	// PaidOnly counts the campaigns for paid members only (Earn).
	PaidOnly int64 `json:"paidOnly"`
}

// CampaignList is one page of campaigns.
type CampaignList struct {
	Items    []CampaignListItem   `json:"items"`
	Total    int64                `json:"total" doc:"Campaigns matching the filter"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
	Counts   CampaignStatusCounts `json:"counts"`
}

// ListParams filters the list.
type ListParams struct {
	Status   string
	Query    string
	PaidOnly bool
	Page     int
	PageSize int
}

// List returns one page of campaigns with their route and readiness.
func (s *Service) List(ctx context.Context, ws *store.Workspace, p ListParams) (CampaignList, error) {
	if p.PageSize <= 0 {
		p.PageSize = 25
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Status == "" {
		p.Status = "all"
	}
	q := strings.TrimSpace(p.Query)
	rows, err := s.Q.ListCampaigns(ctx, store.ListCampaignsParams{
		WorkspaceID: ws.ID, StatusGroup: p.Status, Query: q, PaidOnly: p.PaidOnly, Lim: int32(p.PageSize), Off: int32((p.Page - 1) * p.PageSize),
	})
	if err != nil {
		return CampaignList{}, err
	}
	total, err := s.Q.CountCampaigns(ctx, store.CountCampaignsParams{WorkspaceID: ws.ID, StatusGroup: p.Status, Query: q, PaidOnly: p.PaidOnly})
	if err != nil {
		return CampaignList{}, err
	}
	counts, err := s.counts(ctx, ws.ID)
	if err != nil {
		return CampaignList{}, err
	}
	labels, err := s.audienceLabeler(ctx, ws.ID)
	if err != nil {
		return CampaignList{}, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	routes, err := s.routes(ctx, ws.ID, ids)
	if err != nil {
		return CampaignList{}, err
	}
	incidents, err := s.Q.ListCampaignIncidentSummaries(ctx, ids)
	if err != nil {
		return CampaignList{}, err
	}
	l := i18n.LangFrom(ctx)
	notes := map[uuid.UUID][]string{}
	for _, in := range incidents {
		if in.Kind != "threshold" {
			continue
		}
		notes[in.CampaignID] = append(notes[in.CampaignID], autoPausedNote(l, in.N, in.MailboxProvider, in.Metric))
	}
	out := CampaignList{Items: make([]CampaignListItem, 0, len(rows)), Total: total, Page: p.Page, PageSize: p.PageSize, Counts: counts}
	for _, r := range rows {
		it := CampaignListItem{
			ID: r.ID.String(), Name: r.Name, Status: r.Status, Audience: labels(r.Audience), Recipients: r.AudienceCount, PaidOnly: paidaccess.Parse(r.Audience).Who == paidaccess.Paid,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, UpdatedBy: r.UpdatedByName, ScheduledAt: r.ScheduledAt, StartedAt: r.StartedAt,
			Origin: r.Origin, Notes: []string{}, Routes: routes[r.ID],
		}
		if it.Routes == nil {
			it.Routes = []RouteShare{}
		}
		var sched struct {
			Timezone string `json:"timezone"`
		}
		_ = json.Unmarshal(r.Schedule, &sched)
		it.Timezone = sched.Timezone
		if r.Sent > 0 {
			it.Recipients = &r.Sent
			it.Delivered = pct(r.Delivered, r.Sent)
			it.HumanOpens = pct(r.OpenedHuman, r.Delivered)
			it.Clicks = pct(r.Clicked, r.Delivered)
		}
		// A/B notes describe a test that ran; drafts set one up in the composer.
		if len(r.AbTest) > 0 && r.StartedAt != nil {
			var ab struct {
				Kind   string `json:"kind"`
				Winner string `json:"winner"`
			}
			if json.Unmarshal(r.AbTest, &ab) == nil && ab.Kind != "" {
				it.Notes = append(it.Notes, abNote(l, ab.Kind, ab.Winner))
			}
		}
		it.Notes = append(it.Notes, notes[r.ID]...)
		if r.Note != "" {
			it.Notes = append(it.Notes, r.Note)
		}
		if r.ReadinessVerdict != "" {
			it.Readiness = &ReadinessSummary{Verdict: r.ReadinessVerdict, Issues: r.ReadinessIssues}
			it.Blocked = r.ReadinessVerdict == "blocked" && r.Status == "draft"
		}
		if r.DigestID != nil && r.DigestName != "" {
			ref := &CampaignDigestRef{ID: r.DigestID.String(), Name: r.DigestName, IssueStatus: r.DigestIssueStatus, Posts: int(r.DigestPosts),
				PreparedAt: r.DigestPreparedAt, SendAt: r.DigestSendAt}
			ref.ApprovalID = r.DigestApprovalID
			if ref.IssueStatus == "prepared" && r.DigestApprovalStatus == "pending" {
				ref.IssueStatus = "waiting"
			}
			it.Digest = ref
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

func (s *Service) counts(ctx context.Context, wsID uuid.UUID) (CampaignStatusCounts, error) {
	rows, err := s.Q.CountCampaignsByStatus(ctx, wsID)
	if err != nil {
		return CampaignStatusCounts{}, err
	}
	var c CampaignStatusCounts
	for _, r := range rows {
		c.All += r.N
		switch r.Status {
		case "draft", "approval", "approved":
			c.Drafts += r.N
		case "scheduled":
			c.Scheduled += r.N
		case "sending", "paused":
			c.Sending += r.N
		default:
			c.Sent += r.N
		}
	}
	paid, err := s.Q.CountCampaigns(ctx, store.CountCampaignsParams{WorkspaceID: wsID, StatusGroup: "all", PaidOnly: true})
	if err != nil {
		return CampaignStatusCounts{}, err
	}
	c.PaidOnly = paid
	return c, nil
}

// audienceLabeler turns an audience definition into "Weekly Digest" or
// "All subscribers" (every list of the workspace).
func (s *Service) audienceLabeler(ctx context.Context, wsID uuid.UUID) (func(json.RawMessage) string, error) {
	lists, err := s.Q.ListWorkspaceLists(ctx, wsID)
	if err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(lists))
	for _, l := range lists {
		names[l.ID] = l.Name
	}
	l := i18n.LangFrom(ctx)
	return func(raw json.RawMessage) string {
		var a struct {
			Lists    []uuid.UUID `json:"lists"`
			Segments []uuid.UUID `json:"segments"`
		}
		if json.Unmarshal(raw, &a) != nil || (len(a.Lists) == 0 && len(a.Segments) == 0) {
			return ""
		}
		if len(a.Segments) == 0 && len(a.Lists) == len(names) && len(names) > 1 {
			return mAudienceAll.Text(l, nil)
		}
		parts := make([]string, 0, len(a.Lists))
		for _, id := range a.Lists {
			if n, ok := names[id]; ok {
				parts = append(parts, n)
			}
		}
		if len(a.Segments) > 0 {
			parts = append(parts, mAudienceSegments.Text(l, i18n.Args{"count": len(a.Segments)}))
		}
		label := strings.Join(parts, " + ")
		if p := paidaccess.Parse(raw); p.Set() {
			label += " · " + mAudiencePaid.Text(l, i18n.Args{"who": p.Who})
		}
		return label
	}, nil
}

func (s *Service) routes(ctx context.Context, wsID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID][]RouteShare, error) {
	rows, err := s.Q.ListCampaignRouteShares(ctx, ids)
	if err != nil {
		return nil, err
	}
	planned := map[uuid.UUID]int64{}
	for _, r := range rows {
		planned[r.CampaignID] += r.Planned
	}
	out := map[uuid.UUID][]RouteShare{}
	for _, r := range rows {
		share := RouteShare{ProviderID: r.ProviderID.String(), Provider: r.ProviderName, Short: ShortName(r.ProviderName, r.Adapter), Adapter: r.Adapter, Planned: r.Planned, Sent: r.Sent}
		if t := planned[r.CampaignID]; t > 0 {
			share.Percent = math.Round(float64(r.Planned)/float64(t)*1000) / 10
		}
		out[r.CampaignID] = append(out[r.CampaignID], share)
	}
	return out, nil
}

// ShortName is the badge label of a provider: "Amazon SES" → "SES".
func ShortName(name, adapter string) string {
	switch adapter {
	case "ses":
		return "SES"
	case "azureacs", "azure":
		return "Azure"
	}
	if i := strings.Index(name, " ("); i > 0 {
		return name[:i]
	}
	return name
}

func pct(n, d int64) *float64 {
	if d <= 0 {
		return nil
	}
	v := math.Round(float64(n)/float64(d)*1000) / 10
	return &v
}

func mailboxName(k string) string {
	switch k {
	case "gmail":
		return "Gmail"
	case "outlook":
		return "Outlook"
	case "yahoo":
		return "Yahoo"
	case "icloud":
		return "iCloud"
	}
	return ""
}

// CampaignSummary is the KPI row above the list (last 30 days).
type CampaignSummary struct {
	Days        int            `json:"days"`
	Campaigns   int64          `json:"campaigns"`
	Sent        int64          `json:"sent"`
	Delivered   *float64       `json:"delivered,omitempty"`
	Bounces     *float64       `json:"bounces,omitempty"`
	Complaints  *float64       `json:"complaints,omitempty"`
	HumanOpens  *float64       `json:"humanOpens,omitempty"`
	Clicks      *float64       `json:"clicks,omitempty"`
	ClickToOpen *float64       `json:"clickToOpen,omitempty"`
	Cost        float64        `json:"cost" doc:"Dollars"`
	CostBy      []ProviderCost `json:"costBy"`
}

// ProviderCost is one provider's cost in the period.
type ProviderCost struct {
	Provider string  `json:"provider"`
	Short    string  `json:"short"`
	Cost     float64 `json:"cost"`
}

// Summary returns the last-N-days totals across campaigns.
func (s *Service) Summary(ctx context.Context, ws *store.Workspace, days int) (CampaignSummary, error) {
	since := s.now().AddDate(0, 0, -days)
	t, err := s.Q.CampaignTotalsSince(ctx, store.CampaignTotalsSinceParams{WorkspaceID: ws.ID, Since: &since})
	if err != nil {
		return CampaignSummary{}, err
	}
	out := CampaignSummary{Days: days, Campaigns: t.Campaigns, Sent: t.Sent, CostBy: []ProviderCost{}}
	out.Delivered = pct(t.Delivered, t.Sent)
	out.Bounces = pct(t.Bounced, t.Sent)
	if t.Delivered > 0 {
		v := math.Round(float64(t.Complained)/float64(t.Delivered)*10000) / 100
		out.Complaints = &v
	}
	out.HumanOpens = pct(t.OpenedHuman, t.Delivered)
	out.Clicks = pct(t.Clicked, t.Delivered)
	out.ClickToOpen = pct(t.Clicked, t.OpenedHuman)
	costs, err := s.Q.CampaignCostSince(ctx, store.CampaignCostSinceParams{WorkspaceID: ws.ID, Since: &since})
	if err != nil {
		return CampaignSummary{}, err
	}
	for _, c := range costs {
		v := float64(c.CostMicros) / 1e6
		out.Cost += v
		out.CostBy = append(out.CostBy, ProviderCost{Provider: c.ProviderName, Short: ShortName(c.ProviderName, c.Adapter), Cost: math.Round(v*100) / 100})
	}
	out.Cost = math.Round(out.Cost*100) / 100
	return out, nil
}
