package agency

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

// Needs attention: health below 80, an incident in the last 7 days, or an
// approval waiting over 24 hours.
const (
	attentionHealth = 80
	attentionWait   = 24 * time.Hour
	incidentWindow  = 7 * 24 * time.Hour
	sendWindow      = 7 * 24 * time.Hour
)

// AgencyHealth is a client's sender health: the Overview's score, its
// 30-day trend and HostRepute's view where it's connected.
type AgencyHealth struct {
	// State is ready, or pending while the client has no sending domain.
	State string `json:"state" enum:"ready,pending,unknown"`
	Score int    `json:"score"`
	Tone  string `json:"tone"`
	// Trend is one score per day, oldest first (up to 30).
	Trend []int `json:"trend"`
	// HostRepute is clean, listed or stale; empty when it isn't connected.
	HostRepute string    `json:"hostRepute,omitempty"`
	Listings   int       `json:"listings,omitempty"`
	CheckedAt  time.Time `json:"checkedAt"`
}

// AgencyClientRow is a client in the overview's table and the switcher.
type AgencyClientRow struct {
	Workspace AgencyWorkspaceRef `json:"workspace"`
	// AccountManager is the client's contact at the agency.
	AccountManager *AgencyPersonRef `json:"accountManager,omitempty"`
	Contacts       int64            `json:"contacts"`
	Attention      bool             `json:"attention"`
	// AttentionReasons: health, incident, approval.
	AttentionReasons []string      `json:"attentionReasons"`
	Health           *AgencyHealth `json:"health,omitempty"`
	// Mine: you're assigned to this client.
	Mine bool `json:"mine"`
	// Role is your role there, when you're assigned.
	Role       *AgencyRoleRef `json:"role,omitempty"`
	NextSendAt *time.Time     `json:"nextSendAt,omitempty"`
	// SettingUp: a new client whose setup steps aren't done yet.
	SettingUp bool `json:"settingUp"`
	// CreatedAt is when it joined the agency.
	CreatedAt time.Time `json:"createdAt"`
	Template  string    `json:"template,omitempty"`
}

// AgencyPersonRef names a person.
type AgencyPersonRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarColor string `json:"avatarColor"`
}

// AgencyMe is what the shell shows about the agency: the switcher's agency
// entry and clients, the More sheet and the agency area's sidebar.
type AgencyMe struct {
	Enabled  bool   `json:"enabled"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	Host     string `json:"host"`
	// Staff is set for the agency's staff; client people get only Enabled
	// and Name.
	Staff *AgencyMeStaff `json:"staff,omitempty"`
	// Clients are the clients the staff member works in (every one for
	// agency managers), with their health dots.
	Clients []AgencyClientRow `json:"clients"`
	// AgencyWorkspace is the agency's own workspace.
	AgencyWorkspace *AgencyClientRow `json:"agencyWorkspace,omitempty"`
	ClientCount     int              `json:"clientCount"`
	// Home is the workspace the Agency area opens in (its URLs live under
	// a workspace, like Instance admin's): the agency's own when you're a
	// member there, else your first client.
	Home string `json:"home,omitempty"`
}

// AgencyMeStaff is the signed-in person's place on the staff.
type AgencyMeStaff struct {
	Level       string   `json:"level" enum:"agency_manager,account_manager"`
	Title       string   `json:"title,omitempty"`
	Permissions []string `json:"permissions"`
}

// Me describes the agency for the shell.
func (s *Service) Me(ctx context.Context) (AgencyMe, error) {
	u := api.UserFrom(ctx)
	if u == nil {
		return AgencyMe{}, api.NotFound("Agency")
	}
	st, err := s.settings(ctx, s.Q)
	if err != nil {
		return AgencyMe{}, err
	}
	out := AgencyMe{Enabled: st.Enabled, Name: st.Name, Initials: Initials(st.Name), Host: host(st.PublicURL), Clients: []AgencyClientRow{}}
	if !st.Enabled {
		return out, nil
	}
	me, err := s.staffOf(ctx, u)
	if err != nil || me == nil {
		return out, err
	}
	out.Staff = &AgencyMeStaff{Level: me.Level, Title: me.Title, Permissions: me.Permissions.Strings()}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return out, err
	}
	rows, err := s.clientRows(ctx, me, set)
	if err != nil {
		return out, err
	}
	for i := range rows {
		r := rows[i]
		if r.Workspace.Kind == "agency" {
			out.AgencyWorkspace = &r
			if r.Mine {
				out.Home = r.Workspace.Slug
			}
			continue
		}
		out.Clients = append(out.Clients, r)
		if out.Home == "" && r.Mine {
			out.Home = r.Workspace.Slug
		}
	}
	out.ClientCount = len(out.Clients)
	if out.AgencyWorkspace != nil && out.AgencyWorkspace.Mine {
		out.Home = out.AgencyWorkspace.Workspace.Slug
	}
	return out, nil
}

// clientRows are the visible workspaces with the cheap facts: attention,
// contacts, health from the latest snapshot, the next send's time.
func (s *Service) clientRows(ctx context.Context, me *Staff, set workspaceSet) ([]AgencyClientRow, error) {
	l := i18n.LangFrom(ctx)
	ids := set.ids()
	now := s.now()
	facts, err := s.Q.AgencyAttentionFacts(ctx, store.AgencyAttentionFactsParams{WorkspaceIds: ids, Since: now.Add(-incidentWindow), Now: &now})
	if err != nil {
		return nil, err
	}
	byWS := map[uuid.UUID]store.AgencyAttentionFactsRow{}
	for _, f := range facts {
		byWS[f.WorkspaceID] = f
	}
	health, err := s.healthOf(ctx, set.visible)
	if err != nil {
		return nil, err
	}
	setup, err := s.setupOf(ctx, set.visible)
	if err != nil {
		return nil, err
	}
	out := make([]AgencyClientRow, 0, len(set.visible))
	for _, w := range set.visible {
		f := byWS[w.ID]
		row := AgencyClientRow{Workspace: wsRef(w), Contacts: f.Contacts, AttentionReasons: []string{}, Health: health[w.ID], NextSendAt: unixTime(f.NextSendUnix),
			CreatedAt: w.AddedAt, Template: w.TemplateName}
		if w.AccountManagerID != nil {
			row.AccountManager = &AgencyPersonRef{ID: w.AccountManagerID.String(), Name: displayName(w.AccountManagerName, w.AccountManagerEmail), Email: w.AccountManagerEmail, AvatarColor: w.AccountManagerColor}
		}
		if m, ok := set.mine[w.ID]; ok {
			row.Mine = true
			ref := roleRef(m.RoleID, m.RoleKey, api.RoleName(l, m.RoleKey, m.RoleName), m.RoleBuiltin)
			row.Role = &ref
		}
		if st := setup[w.ID]; st != nil && st.Done < st.Total {
			row.SettingUp = true
		}
		if h := row.Health; h != nil && h.State == "ready" && h.Score < attentionHealth {
			row.AttentionReasons = append(row.AttentionReasons, "health")
		}
		if f.Incidents > 0 {
			row.AttentionReasons = append(row.AttentionReasons, "incident")
		}
		if t := unixTime(f.OldestApprovalUnix); t != nil && now.Sub(*t) > attentionWait {
			row.AttentionReasons = append(row.AttentionReasons, "approval")
		}
		row.Attention = len(row.AttentionReasons) > 0
		out = append(out, row)
	}
	return out, nil
}

// healthOf reads the latest snapshot of each workspace and its trend.
// A workspace without a snapshot yet (a new client) is scored now.
func (s *Service) healthOf(ctx context.Context, rows []store.AgencyWorkspacesRow) (map[uuid.UUID]*AgencyHealth, error) {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	since := dateOnly(s.now()).AddDate(0, 0, -29)
	snaps, err := s.Q.AgencyHealthSince(ctx, store.AgencyHealthSinceParams{WorkspaceIds: ids, Since: since})
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]*AgencyHealth{}
	for _, sn := range snaps {
		h := out[sn.WorkspaceID]
		if h == nil {
			h = &AgencyHealth{Trend: []int{}}
			out[sn.WorkspaceID] = h
		}
		if sn.State == "ready" {
			h.Trend = append(h.Trend, int(sn.Score))
		}
		h.State, h.Score, h.Tone, h.HostRepute, h.Listings, h.CheckedAt = sn.State, int(sn.Score), sn.Tone, sn.Hostrepute, int(sn.Listings), sn.ComputedAt
	}
	for _, r := range rows {
		if out[r.ID] != nil {
			continue
		}
		h, err := s.refreshHealth(ctx, r.ID)
		if err != nil || h == nil {
			out[r.ID] = &AgencyHealth{State: "unknown", Tone: "neutral", Trend: []int{}}
			continue
		}
		out[r.ID] = h
	}
	return out, nil
}

// AgencySetup is a new client's setup: the steps its template asked for.
type AgencySetup struct {
	Done  int                `json:"done"`
	Total int                `json:"total"`
	Steps []AgencySetupStep  `json:"steps"`
}

// AgencySetupStep is one step: provider, domain, invite or shopify.
type AgencySetupStep struct {
	Key  string `json:"key" enum:"provider,domain,invite,shopify"`
	Done bool   `json:"done"`
	// Detail is the domain to verify, when there is one.
	Detail string `json:"detail,omitempty"`
}

func (s *Service) setupOf(ctx context.Context, rows []store.AgencyWorkspacesRow) (map[uuid.UUID]*AgencySetup, error) {
	out := map[uuid.UUID]*AgencySetup{}
	for _, r := range rows {
		if len(r.SetupSteps) == 0 {
			continue
		}
		f, err := s.Q.AgencySetupFacts(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		st := &AgencySetup{Steps: []AgencySetupStep{}}
		for _, k := range r.SetupSteps {
			step := AgencySetupStep{Key: k}
			switch k {
			case "provider":
				step.Done = f.Providers > 0
			case "domain":
				step.Done, step.Detail = f.VerifiedDomains > 0, f.FirstDomain
			case "invite":
				step.Done = f.ClientMembers+f.ClientInvites > 0
			case "shopify":
				step.Done = false
			default:
				continue
			}
			st.Steps = append(st.Steps, step)
			st.Total++
			if step.Done {
				st.Done++
			}
		}
		out[r.ID] = st
	}
	return out, nil
}

// AgencyOverviewView is the agency overview: the summary, the three cards
// and the clients, cheap facts only (each row's details load on their own).
type AgencyOverviewView struct {
	Clients         []AgencyClientRow `json:"clients"`
	AgencyWorkspace *AgencyClientRow  `json:"agencyWorkspace,omitempty"`
	Counts          AgencyCounts      `json:"counts"`
	// Approvals waiting, oldest first.
	Approvals []AgencyApproval `json:"approvals"`
	// Incidents in the last 7 days, newest first.
	Incidents []AgencyIncident `json:"incidents"`
	// Sends in the next 7 days across every workspace, soonest first, and
	// the first one after that.
	Sends     []AgencySend `json:"sends"`
	NextAfter *AgencySend  `json:"nextAfter,omitempty"`
	// CanCreate: you can create clients (agency:manage).
	CanCreate bool `json:"canCreate"`
	Manager   bool `json:"manager"`
	// Timezone is yours, for writing send times.
	Timezone string `json:"timezone"`
}

// AgencyCounts are the filter chips' numbers.
type AgencyCounts struct {
	All       int `json:"all"`
	Attention int `json:"attention"`
	Mine      int `json:"mine"`
	Sends     int `json:"sends"`
}

// AgencyApproval is a send waiting for approval in a client.
type AgencyApproval struct {
	ID        string             `json:"id"`
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Campaign  string             `json:"campaign"`
	CampaignID string            `json:"campaignId"`
	// WaitingFor are the people whose approval it waits for: the client's
	// approvers under client sign-off.
	WaitingFor  []string   `json:"waitingFor"`
	RequestedBy string     `json:"requestedBy"`
	RequestedAt time.Time  `json:"requestedAt"`
	SendAt      *time.Time `json:"sendAt,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
	Recipients  int64      `json:"recipients"`
	// ClientSignoff: the request waits for the client's own approver.
	ClientSignoff bool `json:"clientSignoff"`
}

// AgencyIncident is an incident in a client in the last 7 days.
type AgencyIncident struct {
	ID        string             `json:"id"`
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Kind      string             `json:"kind" enum:"auto_pause,provider_down,webhooks_silent,sms_budget_reached,store_disconnected"`
	// State: ongoing, or how it ended (resumed, cancelled, recovered).
	State      string     `json:"state" enum:"ongoing,resumed,cancelled,recovered"`
	OccurredAt time.Time  `json:"occurredAt"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
	Campaign   string     `json:"campaign,omitempty"`
	Provider   string     `json:"provider,omitempty"`
	// What tripped an auto-pause: "yahoo", "deferral", 4.1, 3.
	MailboxProvider string  `json:"mailboxProvider,omitempty"`
	Metric          string  `json:"metric,omitempty"`
	Value           float64 `json:"value,omitempty"`
	Threshold       float64 `json:"threshold,omitempty"`
	// How it ended: who resumed, and whether with changes.
	ResolvedBy  string `json:"resolvedBy,omitempty"`
	WithChanges bool   `json:"withChanges,omitempty"`
	// Failovers are the streams that moved to another provider meanwhile.
	Failovers []AgencyFailover `json:"failovers"`
	// BroadcastHeld: a Broadcast was held while the provider was down.
	BroadcastHeld bool   `json:"broadcastHeld"`
	Title         string `json:"title"`
	Detail        string `json:"detail"`
}

// AgencyFailover is a stream that moved to another provider.
type AgencyFailover struct {
	Stream string `json:"stream"`
	To     string `json:"to"`
}

// AgencySend is a send with a time.
type AgencySend struct {
	Workspace  AgencyWorkspaceRef `json:"workspace"`
	CampaignID string             `json:"campaignId"`
	Name       string             `json:"name"`
	Status     string             `json:"status"`
	SendAt     time.Time          `json:"sendAt"`
	Timezone   string             `json:"timezone"`
	Recipients int64              `json:"recipients"`
	// Health is the client's sender health, for the warning badge.
	Health *int `json:"health,omitempty"`
}

// Overview returns the agency overview for the caller.
func (s *Service) Overview(ctx context.Context) (AgencyOverviewView, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyOverviewView{}, err
	}
	u := api.UserFrom(ctx)
	out := AgencyOverviewView{Clients: []AgencyClientRow{}, Approvals: []AgencyApproval{}, Incidents: []AgencyIncident{}, Sends: []AgencySend{},
		CanCreate: me.Can(authz.AgencyManage), Manager: me.Manager(), Timezone: u.Timezone}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return out, err
	}
	rows, err := s.clientRows(ctx, me, set)
	if err != nil {
		return out, err
	}
	health := map[uuid.UUID]*AgencyHealth{}
	for i := range rows {
		r := rows[i]
		id := uuid.MustParse(r.Workspace.ID)
		health[id] = r.Health
		if r.Workspace.Kind == "agency" {
			out.AgencyWorkspace = &r
			continue
		}
		out.Clients = append(out.Clients, r)
		out.Counts.All++
		if r.Attention {
			out.Counts.Attention++
		}
		if r.Mine {
			out.Counts.Mine++
		}
	}
	sortClients(out.Clients)
	ids := set.ids()
	now := s.now()
	if out.Approvals, err = s.approvals(ctx, set, ids); err != nil {
		return out, err
	}
	if out.Incidents, err = s.incidents(ctx, set, ids, now.Add(-incidentWindow)); err != nil {
		return out, err
	}
	sends, err := s.sends(ctx, set, ids, now)
	if err != nil {
		return out, err
	}
	for i := range sends {
		if h := health[uuid.MustParse(sends[i].Workspace.ID)]; h != nil && h.State == "ready" {
			v := h.Score
			sends[i].Health = &v
		}
		if sends[i].SendAt.Before(now.Add(sendWindow)) {
			out.Sends = append(out.Sends, sends[i])
		} else if out.NextAfter == nil {
			n := sends[i]
			out.NextAfter = &n
		}
	}
	out.Counts.Sends = len(out.Sends)
	return out, nil
}

// sortClients: Needs attention first, then by the next send, soonest
// first; clients with nothing scheduled last, by name.
func sortClients(rows []AgencyClientRow) {
	slices.SortStableFunc(rows, func(a, b AgencyClientRow) int {
		if a.Attention != b.Attention {
			if a.Attention {
				return -1
			}
			return 1
		}
		switch {
		case a.NextSendAt != nil && b.NextSendAt != nil:
			if c := a.NextSendAt.Compare(*b.NextSendAt); c != 0 {
				return c
			}
		case a.NextSendAt != nil:
			return -1
		case b.NextSendAt != nil:
			return 1
		}
		return strings.Compare(strings.ToLower(a.Workspace.Name), strings.ToLower(b.Workspace.Name))
	})
}

// approvals are the sends waiting for approval in the given workspaces.
func (s *Service) approvals(ctx context.Context, set workspaceSet, ids []uuid.UUID) ([]AgencyApproval, error) {
	out := []AgencyApproval{}
	rows, err := s.Q.AgencyPendingApprovals(ctx, ids)
	if err != nil || len(rows) == 0 {
		return out, err
	}
	reqIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		reqIDs = append(reqIDs, r.ID)
	}
	decisions, err := s.Q.AgencyApprovalDecisionsFor(ctx, reqIDs)
	if err != nil {
		return out, err
	}
	decided := map[uuid.UUID]map[uuid.UUID]bool{}
	for _, d := range decisions {
		if d.ApproverID == nil {
			continue
		}
		if decided[d.RequestID] == nil {
			decided[d.RequestID] = map[uuid.UUID]bool{}
		}
		decided[d.RequestID][*d.ApproverID] = true
	}
	members, err := s.Q.AgencyApprovers(ctx, ids)
	if err != nil {
		return out, err
	}
	userIDs := []uuid.UUID{}
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}
	staffIDs, err := s.Q.AgencyStaffIn(ctx, userIDs)
	if err != nil {
		return out, err
	}
	staff := map[uuid.UUID]bool{}
	for _, id := range staffIDs {
		staff[id] = true
	}
	for _, r := range rows {
		w, ok := set.find(r.WorkspaceID)
		if !ok || r.CampaignID == nil {
			continue
		}
		a := AgencyApproval{ID: r.ID.String(), Workspace: wsRef(w), Campaign: r.CampaignName, CampaignID: r.CampaignID.String(), WaitingFor: []string{},
			RequestedBy: r.RequestedByName, RequestedAt: r.RequestedAt, Recipients: r.AudienceCount, ClientSignoff: slices.Contains(r.MatchedRules, "client_signoff")}
		if n := snapshotAudience(r.Snapshot); n > 0 {
			a.Recipients = n
		}
		sch := readiness.ParseSchedule(r.Schedule)
		a.Timezone = firstNonEmpty(sch.Timezone, w.Timezone)
		if r.ScheduledAt != nil {
			t := *r.ScheduledAt
			a.SendAt = &t
		} else if sch.At != "" {
			if t, err := time.Parse(time.RFC3339, sch.At); err == nil {
				a.SendAt = &t
			}
		}
		for _, m := range members {
			if m.WorkspaceID != r.WorkspaceID || (r.RequestedBy != nil && m.UserID == *r.RequestedBy) || decided[r.ID][m.UserID] {
				continue
			}
			if !authz.Effective(m.RoleKey, m.Builtin, m.Permissions).Has(authz.CampaignsApprove) {
				continue
			}
			if a.ClientSignoff && staff[m.UserID] {
				continue
			}
			a.WaitingFor = append(a.WaitingFor, m.Name)
		}
		out = append(out, a)
	}
	return out, nil
}

// snapshotAudience is the recipients an approval covers.
func snapshotAudience(raw json.RawMessage) int64 {
	var sn struct {
		AudienceCount int64 `json:"audience_count"`
	}
	_ = json.Unmarshal(raw, &sn)
	return sn.AudienceCount
}

// sends are the sends with a time from now on.
func (s *Service) sends(ctx context.Context, set workspaceSet, ids []uuid.UUID, now time.Time) ([]AgencySend, error) {
	rows, err := s.Q.AgencyPlannedSends(ctx, store.AgencyPlannedSendsParams{WorkspaceIds: ids, Since: &now})
	if err != nil {
		return nil, err
	}
	out := []AgencySend{}
	for _, r := range rows {
		w, ok := set.find(r.WorkspaceID)
		if !ok {
			continue
		}
		sch := readiness.ParseSchedule(r.Schedule)
		send := AgencySend{Workspace: wsRef(w), CampaignID: r.ID.String(), Name: r.Name, Status: r.Status, SendAt: r.SendAt,
			Timezone: firstNonEmpty(sch.Timezone, w.Timezone)}
		if r.AudienceCount != nil {
			send.Recipients = *r.AudienceCount
		}
		out = append(out, send)
	}
	return out, nil
}

// incidents are the auto-pauses, outages, silent webhooks, SMS budgets and
// disconnected stores in the given workspaces since a time.
func (s *Service) incidents(ctx context.Context, set workspaceSet, ids []uuid.UUID, since time.Time) ([]AgencyIncident, error) {
	rows, err := s.Q.AgencyRecentIncidents(ctx, store.AgencyRecentIncidentsParams{WorkspaceIds: ids, Since: since})
	if err != nil {
		return nil, err
	}
	out := []AgencyIncident{}
	campaigns := []uuid.UUID{}
	for _, r := range rows {
		if r.Kind == "auto_pause" && r.CampaignID != nil {
			campaigns = append(campaigns, *r.CampaignID)
		}
	}
	pauses := map[uuid.UUID][]store.AgencyAutoPausesRow{}
	if len(campaigns) > 0 {
		ps, err := s.Q.AgencyAutoPauses(ctx, campaigns)
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			pauses[p.CampaignID] = append(pauses[p.CampaignID], p)
		}
	}
	for _, r := range rows {
		w, ok := set.find(r.WorkspaceID)
		if !ok {
			continue
		}
		in := AgencyIncident{ID: r.ID.String(), Workspace: wsRef(w), Kind: r.Kind, State: "ongoing", OccurredAt: r.OccurredAt, ResolvedAt: r.ResolvedAt,
			Campaign: r.CampaignName, Provider: r.ProviderName, Failovers: []AgencyFailover{}, Title: r.Title, Detail: r.Detail}
		switch r.Kind {
		case "auto_pause":
			if r.CampaignID != nil {
				for _, p := range pauses[*r.CampaignID] {
					if p.TrippedAt.Sub(r.OccurredAt).Abs() > 10*time.Minute {
						continue
					}
					in.MailboxProvider, in.Metric = p.MailboxProvider, p.Metric
					in.Value, in.Threshold = floatOf(p.Value), floatOf(p.Threshold)
					if p.ResolvedAt != nil {
						t := *p.ResolvedAt
						in.ResolvedAt = &t
						in.ResolvedBy = p.ResolvedByName
						switch p.Resolution {
						case "cancelled":
							in.State = "cancelled"
						default:
							in.State = "resumed"
							in.WithChanges = p.Resolution == "resumed_with_changes"
						}
					}
					break
				}
			}
		case "provider_down", "webhooks_silent":
			if r.ResolvedAt != nil {
				in.State = "recovered"
			}
			if r.ProviderID != nil {
				to := s.now()
				if r.ResolvedAt != nil {
					to = *r.ResolvedAt
				}
				fs, err := s.Q.AgencyFailoversBetween(ctx, store.AgencyFailoversBetweenParams{ProviderID: *r.ProviderID, FromAt: r.OccurredAt.Add(-5 * time.Minute), ToAt: to})
				if err != nil {
					return nil, err
				}
				for _, f := range fs {
					if f.Mode == "held" {
						if f.Stream == "broadcast" {
							in.BroadcastHeld = true
						}
						continue
					}
					in.Failovers = append(in.Failovers, AgencyFailover{Stream: f.Stream, To: f.FallbackName})
				}
			}
		default:
			if r.ResolvedAt != nil {
				in.State = "recovered"
			}
		}
		out = append(out, in)
	}
	return out, nil
}

// unixTime turns Unix seconds into a time; 0 means none.
func unixTime(sec int64) *time.Time {
	if sec == 0 {
		return nil
	}
	t := time.Unix(sec, 0).UTC()
	return &t
}

func floatOf(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// AgencyClientPulse is one row's details: health, the next send, the
// approvals and incidents, and a new client's setup.
type AgencyClientPulse struct {
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Contacts  int64              `json:"contacts"`
	Health    *AgencyHealth      `json:"health,omitempty"`
	NextSend  *AgencySend        `json:"nextSend,omitempty"`
	// Draft names the latest draft when nothing is scheduled ("Holiday
	// catalog is a draft").
	Draft     string           `json:"draft,omitempty"`
	Approvals []AgencyApproval `json:"approvals"`
	Incidents []AgencyIncident `json:"incidents"`
	Setup     *AgencySetup     `json:"setup,omitempty"`
}

// Pulse returns one client's row details.
func (s *Service) Pulse(ctx context.Context, wsID string) (AgencyClientPulse, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyClientPulse{}, err
	}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return AgencyClientPulse{}, err
	}
	id, err := uuid.Parse(wsID)
	if err != nil {
		return AgencyClientPulse{}, api.NotFound("Client")
	}
	w, ok := set.find(id)
	if !ok {
		return AgencyClientPulse{}, api.NotFound("Client")
	}
	one := workspaceSet{all: set.all, mine: set.mine, visible: []store.AgencyWorkspacesRow{w}}
	ids := []uuid.UUID{id}
	now := s.now()
	out := AgencyClientPulse{Workspace: wsRef(w), Approvals: []AgencyApproval{}, Incidents: []AgencyIncident{}}
	facts, err := s.Q.AgencyAttentionFacts(ctx, store.AgencyAttentionFactsParams{WorkspaceIds: ids, Since: now.Add(-incidentWindow), Now: &now})
	if err != nil {
		return out, err
	}
	if len(facts) > 0 {
		out.Contacts = facts[0].Contacts
	}
	health, err := s.healthOf(ctx, one.visible)
	if err != nil {
		return out, err
	}
	out.Health = health[id]
	sends, err := s.sends(ctx, one, ids, now)
	if err != nil {
		return out, err
	}
	if len(sends) > 0 {
		n := sends[0]
		out.NextSend = &n
	} else {
		drafts, err := s.Q.AgencyLatestDrafts(ctx, ids)
		if err != nil {
			return out, err
		}
		if len(drafts) > 0 {
			out.Draft = drafts[0].Name
		}
	}
	if out.Approvals, err = s.approvals(ctx, one, ids); err != nil {
		return out, err
	}
	if out.Incidents, err = s.incidents(ctx, one, ids, now.Add(-incidentWindow)); err != nil {
		return out, err
	}
	setup, err := s.setupOf(ctx, one.visible)
	if err != nil {
		return out, err
	}
	if st := setup[id]; st != nil && st.Done < st.Total {
		out.Setup = st
	}
	return out, nil
}
