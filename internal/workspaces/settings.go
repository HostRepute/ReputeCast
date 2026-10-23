package workspaces

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	provstate "github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// RestoreDays is how long a deleted workspace can be restored by an
// instance admin before it is purged.
const RestoreDays = 7

type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// WorkspaceSender is a sender offered as the default.
type WorkspaceSender struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	// VerifiedOn names the sending providers that verified it ("SES", "Postal").
	VerifiedOn []string `json:"verifiedOn"`
}

// WorkspaceApprovals summarizes the send approval policy.
type WorkspaceApprovals struct {
	// Rules are the enabled rules in words ("audience ≥ 10,000").
	Rules             []string `json:"rules"`
	Approvers         []string `json:"approvers"`
	ApprovalsRequired int      `json:"approvalsRequired"`
	Automations       bool     `json:"automations" doc:"Publishing automations needs approval"`
	Bypass            bool     `json:"bypass" doc:"Owners can bypass with a reason"`
	Pending           int64    `json:"pending"`
}

// WorkspaceMembersSummary is the Members and access card.
type WorkspaceMembersSummary struct {
	Active      int64    `json:"active"`
	Invited     int64    `json:"invited"`
	Deactivated int64    `json:"deactivated"`
	Builtin     []string `json:"builtin"`
	Custom      []string `json:"custom"`
	// SignIn is "Google Workspace SSO enforced for acme.io · password kept for the Owner".
	SignIn string `json:"signIn"`
}

// WorkspaceDanger lists what deleting the workspace removes.
type WorkspaceDanger struct {
	Contacts    int64 `json:"contacts"`
	Lists       int64 `json:"lists"`
	Segments    int64 `json:"segments"`
	Campaigns   int64 `json:"campaigns"`
	Automations int64 `json:"automations"`
	Templates   int64 `json:"templates"`
	Providers   int64 `json:"providers"`
	Scheduled   int64 `json:"scheduled"`
	APIKeys     int64 `json:"apiKeys"`
	Agents      int64 `json:"agents"`
	RestoreDays int   `json:"restoreDays"`
}

// WorkspaceTransferCandidate is a member who could become the Owner.
type WorkspaceTransferCandidate struct {
	MemberID string             `json:"memberId"`
	Name     string             `json:"name"`
	Email    string             `json:"email"`
	RoleName string             `json:"roleName"`
	TwoFA    access.MemberTwoFA `json:"twoFA"`
	// Eligible members are active and have two-factor (their own or the IdP's).
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
	// Note explains how the two-factor requirement is met.
	Note string `json:"note,omitempty"`
}

// WorkspaceSettings is Settings › General.
type WorkspaceSettings struct {
	ID              string                       `json:"id"`
	Slug            string                       `json:"slug"`
	Name            string                       `json:"name"`
	Timezone        string                       `json:"timezone"`
	DefaultSenderID string                       `json:"defaultSenderId,omitempty"`
	Senders         []WorkspaceSender            `json:"senders"`
	Approvals       WorkspaceApprovals           `json:"approvals"`
	Members         WorkspaceMembersSummary      `json:"members"`
	Danger          WorkspaceDanger              `json:"danger"`
	Candidates      []WorkspaceTransferCandidate `json:"candidates"`
	CanManage       bool                         `json:"canManage"`
	CanTransfer     bool                         `json:"canTransfer"`
	CanDelete       bool                         `json:"canDelete"`
	// HasPassword says whether transferring asks for the caller's password.
	HasPassword bool `json:"hasPassword"`
	// Reason explains read-only access.
	Reason  string   `json:"reason,omitempty"`
	Helpers []string `json:"helpers"`
	// HiddenSections are the sections hidden from the sidebar (Sidebar sections).
	HiddenSections []string `json:"hiddenSections" enum:"transactional,grow,earn"`
}

type policy struct {
	Rules struct {
		AudienceMin struct {
			Enabled bool  `json:"enabled"`
			N       int64 `json:"n"`
		} `json:"audience_min"`
		Warming struct {
			Enabled bool `json:"enabled"`
		} `json:"warming_provider"`
		NewProvider struct {
			Enabled bool `json:"enabled"`
			Days    int  `json:"days"`
		} `json:"new_provider"`
		Warnings struct {
			Enabled bool `json:"enabled"`
		} `json:"ready_with_warnings"`
		LacksSend struct {
			Enabled bool `json:"enabled"`
		} `json:"author_lacks_send"`
		Lists struct {
			Enabled bool `json:"enabled"`
		} `json:"approval_lists"`
	} `json:"rules"`
	ApprovalsRequired int   `json:"approvals_required"`
	Automation        bool  `json:"automation_publish"`
	Bypass            *bool `json:"emergency_bypass"`
	// ClientSignoff is client workspaces' rule (agencies).
	ClientSignoff *struct {
		Enabled bool `json:"enabled"`
	} `json:"client_signoff,omitempty"`
}

// approvalRules are the policy's enabled rules in words, in a language.
func approvalRules(l i18n.Lang, raw json.RawMessage) ([]string, policy) {
	var p policy
	_ = json.Unmarshal(raw, &p)
	rules := []string{}
	rule := func(key string, args i18n.Args) { rules = append(rules, msgs.M(key).Text(l, args)) }
	if p.Rules.AudienceMin.Enabled {
		rule("rule.audience", i18n.Args{"n": p.Rules.AudienceMin.N})
	}
	if p.Rules.Warming.Enabled {
		rule("rule.warming", nil)
	}
	if p.Rules.NewProvider.Enabled {
		d := p.Rules.NewProvider.Days
		if d == 0 {
			d = 14
		}
		rule("rule.newProvider", i18n.Args{"days": d})
	}
	if p.Rules.Warnings.Enabled {
		rule("rule.warnings", nil)
	}
	if p.Rules.LacksSend.Enabled {
		rule("rule.lacksSend", nil)
	}
	if p.Rules.Lists.Enabled {
		rule("rule.lists", nil)
	}
	if p.ClientSignoff != nil && p.ClientSignoff.Enabled {
		rule("rule.clientSignoff", nil)
	}
	return rules, p
}

// Settings returns the general settings with the summaries of the page.
func (s *Service) Settings(ctx context.Context, ws *store.Workspace, p *authz.Principal) (WorkspaceSettings, error) {
	out := WorkspaceSettings{
		ID: ws.ID.String(), Slug: ws.Slug, Name: ws.Name, Timezone: ws.Timezone, Senders: []WorkspaceSender{}, Candidates: []WorkspaceTransferCandidate{},
		CanManage: p.Can(authz.WorkspaceManage), CanTransfer: p.Can(authz.WorkspaceTransfer) && p.IsHuman(), CanDelete: p.Can(authz.WorkspaceDelete) && p.IsHuman(),
		Helpers: access.Helpers(ctx, s.Q, ws.ID, authz.WorkspaceManage),
	}
	out.HiddenSections = HiddenSections(ws)
	if ws.DefaultSenderID != nil {
		out.DefaultSenderID = ws.DefaultSenderID.String()
	}
	l := i18n.LangFrom(ctx)
	if !out.CanManage {
		out.Reason = say(ctx, "reason.manage", nil)
		if len(out.Helpers) > 0 {
			out.Reason = say(ctx, "reason.manageAsk", i18n.Args{"helpers": i18n.OrList(l.Locale, out.Helpers)})
		}
	}
	active, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	// The default sender is for campaigns: "verified on" names the providers
	// that can send them (not a transactional-only stream).
	provs := []store.Provider{}
	for _, pr := range active {
		if provstate.Caps(pr).BulkAllowed() {
			provs = append(provs, pr)
		}
	}
	pname := map[string]string{}
	for _, pr := range provs {
		pname[pr.ID.String()] = shortProvider(pr.Name, pr.Adapter)
	}
	senders, err := s.Q.WsSenders(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, sn := range senders {
		var ver map[string]string
		_ = json.Unmarshal(sn.Verification, &ver)
		on := []string{}
		for _, pr := range provs {
			if ver[pr.ID.String()] == "verified" {
				on = append(on, pname[pr.ID.String()])
			}
		}
		out.Senders = append(out.Senders, WorkspaceSender{ID: sn.ID.String(), Name: sn.Name, Email: sn.Email, VerifiedOn: on})
	}
	counts, err := s.Q.WsCounts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	keys, err := s.Q.KeysCountActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Danger = WorkspaceDanger{Contacts: counts.Contacts, Lists: counts.Lists, Segments: counts.Segments, Campaigns: counts.Campaigns, Automations: counts.Automations,
		Templates: counts.Templates, Providers: counts.Providers, Scheduled: counts.Scheduled, APIKeys: keys.ApiKeys, Agents: keys.Agents, RestoreDays: RestoreDays}
	rules, pol := approvalRules(l, ws.ApprovalPolicy)
	out.Approvals = WorkspaceApprovals{Rules: rules, Approvers: access.Helpers(ctx, s.Q, ws.ID, authz.CampaignsApprove), ApprovalsRequired: max(pol.ApprovalsRequired, 1),
		Automations: pol.Automation, Bypass: pol.Bypass == nil || *pol.Bypass, Pending: counts.PendingApprovals}
	// Every approver (members holding campaigns:approve), Owners first.
	if rows, err := s.Q.ListMembersWithRoles(ctx, ws.ID); err == nil {
		owners, others := []string{}, []string{}
		for _, r := range rows {
			if !authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(authz.CampaignsApprove) {
				continue
			}
			if r.RoleKey == authz.RoleOwner {
				owners = append(owners, r.Name)
			} else {
				others = append(others, r.Name)
			}
		}
		out.Approvals.Approvers = append(owners, others...)
	}
	mc, err := s.Q.WsMemberCounts(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Members = WorkspaceMembersSummary{Active: mc.Active, Invited: mc.Invited, Deactivated: mc.Deactivated, Builtin: []string{}, Custom: []string{}}
	roles, err := s.Q.ListRoles(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range roles {
		if r.Builtin {
			out.Members.Builtin = append(out.Members.Builtin, r.Name)
		} else {
			out.Members.Custom = append(out.Members.Custom, r.Name)
		}
	}
	out.Members.SignIn = s.signIn(ctx, ws)
	if u := api.UserFrom(ctx); u != nil {
		out.HasPassword = u.PasswordHash != nil
	}
	if out.CanTransfer {
		if out.Candidates, err = s.candidates(ctx, ws, p); err != nil {
			return out, err
		}
	}
	return out, nil
}

func shortProvider(name, adapter string) string {
	if adapter == "ses" {
		return "SES"
	}
	if i := strings.Index(name, " ("); i > 0 {
		return name[:i]
	}
	return name
}

// signIn summarizes how members sign in, in the request's language.
func (s *Service) signIn(ctx context.Context, ws *store.Workspace) string {
	text := func(key string, args i18n.Args) string { return say(ctx, key, args) }
	sso, _ := s.Q.TeamEnforcedSSO(ctx)
	var parts []string
	for _, p := range sso {
		parts = append(parts, text("signIn.enforced", i18n.Args{"name": p.Name, "domains": strings.Join(p.EnforceDomains, ", ")}))
	}
	if len(parts) > 0 {
		var breakGlass int
		_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM workspace_members m JOIN users u ON u.id = m.user_id JOIN roles r ON r.id = m.role_id
			WHERE m.workspace_id = $1 AND m.status = 'active' AND u.break_glass AND r.key = 'owner'`, ws.ID).Scan(&breakGlass)
		if breakGlass > 0 {
			parts = append(parts, text("signIn.ownerPassword", nil))
		}
		return strings.Join(parts, " · ")
	}
	// SSO that people of a domain can use without it being enforced (the
	// demo keeps enforcement off so password members can still sign in).
	if jit, err := s.Q.TeamJITProviders(ctx, &ws.ID); err == nil && len(jit) > 0 {
		for _, p := range jit {
			parts = append(parts, text("signIn.sso", i18n.Args{"name": p.Name, "domains": strings.Join(p.AllowedDomains, ", ")}))
		}
		return strings.Join(append(parts, text("signIn.alsoPasswords", nil)), " · ")
	}
	st, err := s.Q.GetInstanceSettings(ctx)
	if err != nil {
		return text("signIn.fallback", nil)
	}
	var methods []string
	for _, m := range st.AllowedMethods {
		switch m {
		case "password":
			methods = append(methods, text("signIn.password", nil))
		case "passkey":
			methods = append(methods, text("signIn.passkeys", nil))
		case "social":
			methods = append(methods, text("signIn.social", nil))
		case "sso":
			methods = append(methods, "SSO")
		}
	}
	out := capitalize(strings.Join(methods, ", "))
	switch st.TwofaEnforcement {
	case "everyone":
		out += " · " + text("signIn.twoFactorEveryone", nil)
	case "sensitive":
		out += " · " + text("signIn.twoFactorSensitive", nil)
	}
	return out
}

// candidates lists who can become the Owner.
func (s *Service) candidates(ctx context.Context, ws *store.Workspace, p *authz.Principal) ([]WorkspaceTransferCandidate, error) {
	rows, err := s.Q.TeamListMembers(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.UserID
	}
	idents, err := s.Q.TeamIdentities(ctx, ids)
	if err != nil {
		return nil, err
	}
	sso := map[uuid.UUID]string{}
	for _, i := range idents {
		if i.Kind == "sso" {
			sso[i.UserID] = i.ProviderName
		}
	}
	out := []WorkspaceTransferCandidate{}
	for _, r := range rows {
		if r.UserID == p.UserID || r.Status != "active" || r.UserStatus != "active" {
			continue
		}
		_, viaIdP := sso[r.UserID]
		t := access.TwoFAStatus(access.TwoFAInput{TOTP: r.TotpEnabledAt != nil, Passkeys: r.PasskeyCount, SSOOnly: viaIdP && !r.HasPassword && r.PasskeyCount == 0, GraceUntil: r.TwofaGraceUntil, Now: s.now()})
		c := WorkspaceTransferCandidate{MemberID: r.ID.String(), Name: displayName(r.UserName, r.Email), Email: r.Email, RoleName: r.RoleName, TwoFA: t}
		switch t.State {
		case "on":
			c.Eligible = true
			c.Note = say(ctx, "note.twoFactor", i18n.Args{"name": c.Name, "method": t.Method})
		case "idp":
			c.Eligible = true
			name := sso[r.UserID]
			if name == "" {
				name = say(ctx, "note.singleSignOn", nil)
			}
			c.Note = say(ctx, "note.idp", i18n.Args{"name": c.Name, "provider": name})
		default:
			c.Reason = say(ctx, "reason.noTwoFactor", i18n.Args{"name": c.Name})
		}
		out = append(out, c)
	}
	return out, nil
}

func displayName(name, email string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return email
}

// WorkspaceGeneralInput changes the general settings.
type WorkspaceGeneralInput struct {
	Name            string `json:"name" minLength:"1" maxLength:"80"`
	Timezone        string `json:"timezone" minLength:"1" maxLength:"64"`
	DefaultSenderID string `json:"defaultSenderId,omitempty"`
}

// UpdateGeneral saves name, timezone and default sender.
func (s *Service) UpdateGeneral(ctx context.Context, ws *store.Workspace, in WorkspaceGeneralInput) error {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return api.Invalid("body.name", say(ctx, "problem.name", nil))
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return api.Invalid("body.timezone", say(ctx, "problem.timezone", nil))
	}
	var sender *uuid.UUID
	if in.DefaultSenderID != "" {
		id, err := uuid.Parse(in.DefaultSenderID)
		if err != nil {
			return api.Invalid("body.defaultSenderId", say(ctx, "problem.sender", nil))
		}
		ok, err := s.Q.WsSenderExists(ctx, store.WsSenderExistsParams{ID: id, WorkspaceID: ws.ID})
		if err != nil {
			return err
		}
		if !ok {
			return api.Invalid("body.defaultSenderId", say(ctx, "problem.sender", nil))
		}
		sender = &id
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.WsUpdateGeneral(ctx, store.WsUpdateGeneralParams{ID: ws.ID, Name: name, Timezone: in.Timezone, DefaultSenderID: sender}); err != nil {
			return err
		}
		before := map[string]any{"name": ws.Name, "timezone": ws.Timezone, "default_sender_id": ws.DefaultSenderID}
		after := map[string]any{"name": name, "timezone": in.Timezone, "default_sender_id": sender}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "workspace.updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: name,
			Before: before, After: after, Metadata: map[string]any{"permission": string(authz.WorkspaceManage)}})
	})
}

// WorkspaceTransferInput hands the workspace to another member.
type WorkspaceTransferInput struct {
	MemberID string `json:"memberId" format:"uuid"`
	Password string `json:"password,omitempty" maxLength:"1024" doc:"The caller's password (re-authentication)"`
}

// Transfer makes another member the Owner; the caller becomes an Admin.
func (s *Service) Transfer(ctx context.Context, ws *store.Workspace, p *authz.Principal, in WorkspaceTransferInput) error {
	u := api.UserFrom(ctx)
	if u == nil {
		return api.Err(http.StatusForbidden, api.CodeHumansOnly, say(ctx, "problem.signedInOwner", nil))
	}
	if u.PasswordHash != nil {
		ok, err := authn.VerifyPassword(in.Password, *u.PasswordHash)
		if err != nil || !ok {
			return api.Invalid("body.password", say(ctx, "problem.password", nil))
		}
	} else if sess := api.SessionFrom(ctx); sess != nil && s.now().Sub(sess.CreatedAt) > 10*time.Minute {
		return api.Err(http.StatusForbidden, "reauth_required", say(ctx, "problem.reauth", nil))
	}
	memberID, err := uuid.Parse(in.MemberID)
	if err != nil {
		return api.Invalid("body.memberId", say(ctx, "problem.pickOwner", nil))
	}
	cands, err := s.candidates(ctx, ws, p)
	if err != nil {
		return err
	}
	var target *WorkspaceTransferCandidate
	for i := range cands {
		if cands[i].MemberID == memberID.String() {
			target = &cands[i]
		}
	}
	if target == nil {
		return api.Invalid("body.memberId", say(ctx, "problem.notMember", nil))
	}
	if !target.Eligible {
		return api.Conflict(say(ctx, "problem.notEligible", i18n.Args{"name": target.Name}))
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		owner, err := q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: ws.ID, Key: authz.RoleOwner})
		if err != nil {
			return err
		}
		admin, err := q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: ws.ID, Key: authz.RoleAdmin})
		if err != nil {
			return err
		}
		m, err := q.TeamGetMember(ctx, store.TeamGetMemberParams{ID: memberID, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "Member")
		}
		if err := q.TeamSetMemberRole(ctx, store.TeamSetMemberRoleParams{ID: m.ID, WorkspaceID: ws.ID, RoleID: owner.ID, Scope: json.RawMessage(`{}`)}); err != nil {
			return err
		}
		me, err := q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: ws.ID, UserID: p.UserID})
		if err != nil {
			return err
		}
		if err := q.TeamSetMemberRole(ctx, store.TeamSetMemberRoleParams{ID: me.ID, WorkspaceID: ws.ID, RoleID: admin.ID, Scope: me.Scope}); err != nil {
			return err
		}
		// Read later by the new Owner, in their own language.
		if err := notify.ToUser(ctx, tx, m.UserID, notify.Notification{WorkspaceID: &ws.ID, Kind: "ownership_transferred", Severity: "info",
			TitleText: keep("notify.ownerTitle", i18n.Args{"name": p.Name, "workspace": ws.Name}), Link: "/w/" + ws.Slug + "/settings"}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "workspace.transferred", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"owner": p.Name},
			After:  map[string]any{"owner": target.Name, "previous_owner_role": "Admin"},
			Metadata: audit.KeepSummary(map[string]any{"permission": string(authz.WorkspaceTransfer)},
				p.Name+" → "+target.Name, keep("audit.nowAdmin", i18n.Args{"name": p.Name}))})
	})
}

// WorkspaceDeleteInput confirms deleting the workspace.
type WorkspaceDeleteInput struct {
	Confirm string `json:"confirm" maxLength:"80" doc:"The workspace name, typed exactly"`
}

// WorkspaceDeleteResult says until when an instance admin can restore it.
type WorkspaceDeleteResult struct {
	RestoreUntil time.Time `json:"restoreUntil"`
	// Cancelled counts scheduled sends that stopped.
	Cancelled int64 `json:"cancelled"`
}

// Delete soft-deletes the workspace: scheduled sends stop at once, keys and
// agents stop working, and the data is purged after the restore hold.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, in WorkspaceDeleteInput) (WorkspaceDeleteResult, error) {
	if in.Confirm != ws.Name {
		return WorkspaceDeleteResult{}, api.Invalid("body.confirm", say(ctx, "problem.confirmName", i18n.Args{"name": ws.Name}))
	}
	res := WorkspaceDeleteResult{RestoreUntil: s.now().Add(RestoreDays * 24 * time.Hour)}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if res.Cancelled, err = q.WsCancelScheduled(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.WsSoftDelete(ctx, ws.ID); err != nil {
			return err
		}
		// Read later by each member, in their own language, the date in the
		// workspace's timezone.
		until := res.RestoreUntil.In(location(ws.Timezone))
		if err := notify.ToMembersWith(ctx, tx, ws.ID, "", notify.Notification{WorkspaceID: nil, Kind: "workspace_deleted", Severity: "warn",
			TitleText: keep("notify.deletedTitle", i18n.Args{"name": ws.Name}), BodyText: keep("notify.deletedBody", i18n.Args{"at": until})}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "workspace.deleted", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"name": ws.Name, "slug": ws.Slug},
			Metadata: audit.KeepSummary(map[string]any{"permission": string(authz.WorkspaceDelete)},
				keep("audit.restorable", i18n.Args{"at": until}), keep("audit.sendsStopped", i18n.Args{"count": res.Cancelled}))})
	})
	return res, err
}

// PurgeArgs deletes workspaces past their restore hold.
type PurgeArgs struct{}

// Kind implements river.JobArgs.
func (PurgeArgs) Kind() string { return "workspaces.purge" }

// InsertOpts keeps one purge at a time.
func (PurgeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// PurgeWorker removes soft-deleted workspaces for good.
type PurgeWorker struct {
	river.WorkerDefaults[PurgeArgs]
	Pool *pgxpool.Pool
	Log  *slog.Logger
}

// Work implements river.Worker.
func (w *PurgeWorker) Work(ctx context.Context, _ *river.Job[PurgeArgs]) error {
	rows, err := store.New(w.Pool).WsPurgeDeleted(ctx)
	if err != nil {
		return err
	}
	for _, r := range rows {
		w.Log.InfoContext(ctx, "purged deleted workspace", "id", r.ID, "name", r.Name)
	}
	return nil
}

// RegisterJobs adds the purge of deleted workspaces (hourly).
func RegisterJobs(reg *jobs.Registry, pool *pgxpool.Pool, log *slog.Logger) {
	river.AddWorker(reg.Workers, &PurgeWorker{Pool: pool, Log: log})
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return PurgeArgs{}, nil }, false)
}
