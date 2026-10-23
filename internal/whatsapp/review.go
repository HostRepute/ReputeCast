package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Review & send for WhatsApp (brief §3.9 pre-send check): consent, the
// template's status and category, the quality rating, the messaging limit
// against the audience (who gets it today and who the next day) and the
// cost; then the workspace's approval rules. Sending needs campaigns:send
// and whatsapp:send.

// Check statuses and verdicts.
const (
	Pass  = "pass"
	Warn  = "warn"
	Block = "block"

	VerdictReady    = "ready"
	VerdictWarnings = "warnings"
	VerdictBlocked  = "blocked"
)

// WhatsAppLimitSplit is how a send fits the messaging limit.
type WhatsAppLimitSplit struct {
	Limit int64 `json:"limit"`
	// Used is how many people got a message the business started in the
	// rolling 24 hours; Left how many more it can start.
	Used int64 `json:"used"`
	Left int64 `json:"left"`
	// Now get it now (or at their time); Later the next day from LaterAt.
	Now     int64      `json:"now"`
	Later   int64      `json:"later"`
	LaterAt *time.Time `json:"laterAt,omitempty"`
	// UsedBy is the send that used most of the limit: its name and time.
	UsedBy   string     `json:"usedBy,omitempty"`
	UsedByAt *time.Time `json:"usedByAt,omitempty"`
	UsedByN  int64      `json:"usedByN,omitempty"`
}

// WhatsAppCheck is one pre-send check.
type WhatsAppCheck struct {
	Key    string `json:"key" enum:"account,consent,template,quality,limit,cost,approval"`
	Status string `json:"status" enum:"pass,warn,block"`
	// Account.
	AccountStatus string `json:"accountStatus,omitempty"`
	// Consent.
	Count   int64    `json:"count"`
	Lists   []string `json:"lists"`
	Without int64    `json:"without"`
	// Template.
	Template string `json:"template,omitempty"`
	Category string `json:"category,omitempty"`
	TStatus  string `json:"templateStatus,omitempty"`
	// Quality.
	Quality         string     `json:"quality,omitempty"`
	PreviousQuality string     `json:"previousQuality,omitempty"`
	QualityAt       *time.Time `json:"qualityAt,omitempty"`
	CheckedAt       *time.Time `json:"checkedAt,omitempty"`
	// Messaging limit.
	Split *WhatsAppLimitSplit `json:"split,omitempty"`
	// Cost.
	Cost *WhatsAppCost `json:"cost,omitempty"`
	// Approval.
	Author  string   `json:"author,omitempty"`
	Missing []string `json:"missing"`
	Helpers []string `json:"helpers"`
}

// WhatsAppSendState is a scheduled or running send.
type WhatsAppSendState struct {
	Phase       string     `json:"phase" enum:"scheduled,sending,completed,cancelled"`
	ScheduledAt time.Time  `json:"scheduledAt"`
	Recipients  int64      `json:"recipients"`
	StartedBy   string     `json:"startedBy,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// WhatsAppReview is Review & send.
type WhatsAppReview struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	Status    string                   `json:"status"`
	Version   int                      `json:"version"`
	Verdict   string                   `json:"verdict" enum:"ready,warnings,blocked"`
	CheckedAt time.Time                `json:"checkedAt"`
	Checks    []WhatsAppCheck          `json:"checks"`
	Audience  WhatsAppAudience         `json:"audience"`
	Schedule  WhatsAppSchedule         `json:"schedule"`
	OverLimit string                   `json:"overLimit" enum:"next_day,skip"`
	Cost      WhatsAppCost             `json:"cost"`
	Approval  approvals.ReviewApproval `json:"approval"`
	Access    WhatsAppAccess           `json:"access"`
	Send      *WhatsAppSendState       `json:"send,omitempty"`
}

// review is a campaign's checks, built once for Review, Send and approvals.
type review struct {
	checks  []WhatsAppCheck
	verdict string
	aud     audienceSet
	cost    WhatsAppCost
	tpl     *store.WhatsappTemplate
	split   WhatsAppLimitSplit
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
	return warns, blocks
}

// firstSend is when a campaign's first message goes: now, its moment, or
// the earliest of its local times.
func (s *Service) firstSend(c store.WhatsAppCampaignRow, ws *store.Workspace, aud audienceSet, rl rules, now time.Time) time.Time {
	at := now
	switch c.ScheduleMode {
	case "at":
		if c.SendAt != nil && c.SendAt.After(now) {
			at = *c.SendAt
		}
	case "local":
		if c.LocalDate == nil || c.LocalMinute == nil {
			return now
		}
		d := *c.LocalDate
		first := time.Time{}
		for _, r := range aud.list {
			loc, _ := zoneOf(r, ws)
			t := time.Date(d.Year(), d.Month(), d.Day(), int(*c.LocalMinute)/60, int(*c.LocalMinute)%60, 0, 0, loc)
			if first.IsZero() || t.Before(first) {
				first = t
			}
		}
		if first.IsZero() {
			first = time.Date(d.Year(), d.Month(), d.Day(), int(*c.LocalMinute)/60, int(*c.LocalMinute)%60, 0, 0, wsLocation(ws))
		}
		if first.After(now) {
			at = first
		}
	}
	return at
}

// buildReview runs the checks.
func (s *Service) buildReview(ctx context.Context, ws *store.Workspace, p *authz.Principal, c store.WhatsAppCampaignRow) (*review, error) {
	rv := &review{at: s.now()}
	add := func(ch WhatsAppCheck) {
		if ch.Lists == nil {
			ch.Lists = []string{}
		}
		if ch.Missing == nil {
			ch.Missing = []string{}
		}
		if ch.Helpers == nil {
			ch.Helpers = []string{}
		}
		rv.checks = append(rv.checks, ch)
	}
	a, aerr := s.account(ctx, s.Q, ws)
	if aerr != nil && !errors.Is(aerr, errNoAccount) {
		return nil, aerr
	}
	// The account.
	ac := WhatsAppCheck{Key: "account", Status: Pass}
	switch {
	case aerr != nil:
		ac.Status, ac.AccountStatus = Block, "not_connected"
	case a.Status == "waiting":
		ac.Status, ac.AccountStatus = Block, "waiting"
	case a.Status == "token_rejected":
		ac.Status, ac.AccountStatus = Block, "token_rejected"
	default:
		ac.AccountStatus = a.Status
	}
	add(ac)
	// Consent.
	lists, segments := audienceOf(c.Audience)
	aud, err := s.audience(ctx, ws, lists, segments)
	if err != nil {
		return nil, err
	}
	rv.aud = aud
	cc := WhatsAppCheck{Key: "consent", Status: Pass, Count: int64(len(aud.list)), Without: aud.without}
	if opts, _, err := s.listConsent(ctx, ws); err == nil {
		for _, id := range lists {
			for _, o := range opts {
				if o.ListID == id.String() {
					cc.Lists = append(cc.Lists, o.Name)
				}
			}
		}
	}
	if len(aud.list) == 0 {
		cc.Status = Block
	}
	add(cc)
	// The template.
	tc := WhatsAppCheck{Key: "template", Status: Pass}
	if c.TemplateID != nil {
		if t, err := s.Q.WhatsAppTemplate(ctx, store.WhatsAppTemplateParams{ID: *c.TemplateID, WorkspaceID: ws.ID}); err == nil && t.RemovedAt == nil {
			rv.tpl = &t
			tc.Template, tc.Category, tc.TStatus = t.Name, t.Category, t.Status
		}
	}
	if rv.tpl == nil || rv.tpl.Status != "approved" {
		tc.Status = Block
	}
	add(tc)
	// Quality.
	qc := WhatsAppCheck{Key: "quality", Status: Pass}
	if aerr == nil {
		qc.Quality, qc.PreviousQuality, qc.QualityAt, qc.CheckedAt = a.Quality, a.PreviousQuality, a.QualityChangedAt, a.CheckedAt
		if a.Quality == "medium" || a.Quality == "low" {
			qc.Status = Warn
		}
	}
	add(qc)
	// The messaging limit.
	lc := WhatsAppCheck{Key: "limit", Status: Pass}
	if aerr == nil {
		rl := s.rules(ctx, s.Q, ws)
		first := s.firstSend(c, ws, aud, rl, rv.at)
		lp, err := s.limitPlan(ctx, ws, int(a.MessagingLimit), first)
		if err != nil {
			return nil, err
		}
		final := int64(len(aud.list))
		sp := WhatsAppLimitSplit{Limit: int64(a.MessagingLimit), Used: int64(lp.Used), Left: int64(min(lp.Left, 1<<30))}
		if a.MessagingLimit <= 0 {
			sp.Left = final
		}
		sp.Now = min(final, sp.Left)
		sp.Later = final - sp.Now
		if sp.Later > 0 {
			lc.Status = Warn
			t := lp.later(0, first)
			if rl.Quiet {
				t = rl.Window.NextOpen(t, wsLocation(ws))
			}
			sp.LaterAt = &t
		}
		if lp.Used > 0 {
			var name string
			var at time.Time
			var n int64
			if err := s.Pool.QueryRow(ctx, `SELECT coalesce(c.name, a.name, ''), min(coalesce(m.sent_at, m.created_at)), count(DISTINCT m.recipient)
				FROM channel_messages m JOIN whatsapp_messages w ON w.message_id = m.id AND w.kind = 'template'
				LEFT JOIN campaigns c ON c.id = m.campaign_id LEFT JOIN automations a ON a.id = m.automation_id
				WHERE m.workspace_id = $1 AND m.channel = 'whatsapp' AND m.status IN ('queued','sending','sent','delivered')
				  AND coalesce(m.sent_at, m.send_after, m.created_at) > $2::timestamptz - interval '24 hours'
				GROUP BY 1 ORDER BY 3 DESC LIMIT 1`, ws.ID, first).Scan(&name, &at, &n); err == nil && name != "" {
				sp.UsedBy, sp.UsedByAt, sp.UsedByN = name, &at, n
			}
		}
		rv.split = sp
		lc.Split = &sp
	}
	add(lc)
	// The cost.
	cost := WhatsAppCost{ByCountry: []WhatsAppCostCountry{}, RatesAsOf: s.rateCard(ctx, s.Q, ws).asOf}
	if rv.tpl != nil {
		cost = s.rateCard(ctx, s.Q, ws).estimate(rv.tpl.Category, aud.countries)
	}
	rv.cost = cost
	add(WhatsAppCheck{Key: "cost", Status: Pass, Cost: &cost})
	// Approval.
	ap := WhatsAppCheck{Key: "approval", Status: Pass}
	if p != nil {
		if u, err := s.Q.GetUser(ctx, p.UserID); err == nil {
			ap.Author = u.Name
		}
		if !canSend(p) {
			ap.Status = Warn
			if !p.Can(authz.CampaignsSend) {
				ap.Missing = append(ap.Missing, string(authz.CampaignsSend))
			}
			if !p.Can(authz.WhatsAppSend) {
				ap.Missing = append(ap.Missing, string(authz.WhatsAppSend))
			}
			ap.Helpers = s.helpers(ctx, ws, authz.WhatsAppSend)
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

// Review is Review & send.
func (s *Service) Review(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (WhatsAppReview, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return WhatsAppReview{}, err
	}
	rv, err := s.buildReview(ctx, ws, p, c)
	if err != nil {
		return WhatsAppReview{}, err
	}
	opts, _, _ := s.listConsent(ctx, ws)
	lists, _ := audienceOf(c.Audience)
	out := WhatsAppReview{ID: c.ID.String(), Name: c.Name, Status: c.Status, Version: int(c.Version), Verdict: rv.verdict, CheckedAt: rv.at, Checks: rv.checks,
		Audience: summary(rv.aud, opts, lists), Schedule: scheduleOf(langOf(ctx), c, ws), OverLimit: c.OverLimit, Cost: rv.cost, Access: s.accessFor(ctx, ws, p)}
	if s.Approvals != nil {
		if ap, err := s.Approvals.ChannelReview(ctx, ws, p, id); err == nil {
			out.Approval = ap
		}
	}
	if snd, err := s.Q.WhatsAppSend(ctx, id); err == nil {
		out.Send = &WhatsAppSendState{Phase: snd.Phase, ScheduledAt: snd.ScheduledAt, Recipients: int64(snd.Recipients), StartedBy: c.StartedByName,
			CompletedAt: snd.CompletedAt}
	}
	return out, nil
}

// WhatsAppSendInput sends a campaign as reviewed.
type WhatsAppSendInput struct {
	// Version is the version Review & send showed; a newer one is refused.
	Version int `json:"version" minimum:"1"`
	// OverLimit is what happens to the people over the messaging limit:
	// the next day (default) or skipped.
	OverLimit string `json:"overLimit,omitempty" enum:"next_day,skip,"`
}

// WhatsAppSendResult is what Send did.
type WhatsAppSendResult struct {
	Status      string    `json:"status" enum:"scheduled,sending"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Recipients  int64     `json:"recipients"`
	Now         int64     `json:"now"`
	Later       int64     `json:"later"`
}

// Send sends or schedules the campaign after the checks and the approval
// rules (logged).
func (s *Service) Send(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WhatsAppSendInput) (WhatsAppSendResult, error) {
	if err := s.requireSend(ctx, ws, p); err != nil {
		return WhatsAppSendResult{}, err
	}
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return WhatsAppSendResult{}, err
	}
	l := langOf(ctx)
	switch c.Status {
	case "draft", "approval", "approved":
	default:
		return WhatsAppSendResult{}, api.Conflict(mAlreadyStatus.Text(l, i18n.Args{"status": c.Status}))
	}
	if in.Version != int(c.Version) {
		return WhatsAppSendResult{}, api.Err(409, "stale_version", mStaleReviewed.Text(l, i18n.Args{"version": itoa(int(c.Version))}))
	}
	rv, err := s.buildReview(ctx, ws, p, c)
	if err != nil {
		return WhatsAppSendResult{}, err
	}
	if _, blocks := rv.counts(); blocks > 0 {
		return WhatsAppSendResult{}, api.Err(409, "readiness_blocked", mBlocking.Text(l, nil))
	}
	var approval *store.ApprovalRequest
	if s.Approvals != nil {
		if approval, err = s.Approvals.ChannelGate(ctx, ws, p, id); err != nil {
			return WhatsAppSendResult{}, err
		}
	}
	now := s.now()
	rl := s.rules(ctx, s.Q, ws)
	at := s.firstSend(c, ws, rv.aud, rl, now)
	over := c.OverLimit
	if in.OverLimit != "" {
		over = in.OverLimit
	}
	final := int64(len(rv.aud.list))
	nowN, later := rv.split.Now, rv.split.Later
	if over == "skip" {
		final, later = nowN, 0
	}
	status := "scheduled"
	if !at.After(now.Add(time.Minute)) {
		status = "sending"
	}
	var approvedBy *uuid.UUID
	if approval != nil && approval.DecidedBy != nil {
		approvedBy = approval.DecidedBy
	}
	left := map[string]int64{}
	if over == "skip" {
		left["over_limit_skip"] = 1
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if over != c.OverLimit {
			if _, err := tx.Exec(ctx, `UPDATE whatsapp_campaigns SET over_limit = $2 WHERE campaign_id = $1`, id, over); err != nil {
				return err
			}
		}
		if err := q.WhatsAppSendUpsert(ctx, store.WhatsAppSendUpsertParams{CampaignID: id, WorkspaceID: ws.ID, Phase: "scheduled", Recipients: int32(final),
			LeftOut: jsonOf(left), StartedBy: uuidPtr(p.UserID), ApprovedBy: approvedBy, ScheduledAt: at}); err != nil {
			return err
		}
		if err := q.WhatsAppCampaignSetStatus(ctx, store.WhatsAppCampaignSetStatusParams{Status: "scheduled", ScheduledAt: &at, StartedBy: uuidPtr(p.UserID),
			AudienceCount: &final, At: now, ID: id}); err != nil {
			return err
		}
		if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "campaign.scheduled", map[string]any{"campaign_id": id.String(), "name": c.Name, "channel": "whatsapp",
			"scheduled_at": at, "recipients": final}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.scheduled", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "scheduled"},
			Metadata: map[string]any{"channel": "whatsapp", "recipients": final, "scheduled_at": at, "template": c.TemplateName, "over_limit": over,
				"later": later, "estimated_cost": rv.cost.Amount}})
	})
	if err != nil {
		return WhatsAppSendResult{}, err
	}
	s.kickStart(ctx)
	return WhatsAppSendResult{Status: status, ScheduledAt: at, Recipients: final, Now: nowN, Later: later}, nil
}

// Unschedule takes a scheduled campaign back to a draft (logged).
func (s *Service) Unschedule(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	snd, err := s.Q.WhatsAppSend(ctx, id)
	if c.Status != "scheduled" || err != nil || snd.Phase != "scheduled" {
		return api.Conflict(mUnscheduleOnly.Text(langOf(ctx), nil))
	}
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM whatsapp_sends WHERE campaign_id = $1 AND phase = 'scheduled'`, id); err != nil {
			return err
		}
		if err := q.WhatsAppCampaignSetStatus(ctx, store.WhatsAppCampaignSetStatusParams{Status: "draft", At: s.now(), ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.unscheduled", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"status": "scheduled"}, After: map[string]any{"status": "draft"}, Metadata: map[string]any{"channel": "whatsapp"}})
	})
}

// Cancel stops a campaign: messages not yet sent to Meta don't go (logged).
func (s *Service) Cancel(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return err
	}
	if c.Status != "sending" && c.Status != "scheduled" {
		return api.Conflict(mCancelOnly.Text(langOf(ctx), nil))
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'cancelled', reason = 'cancelled', updated_at = $2
			WHERE campaign_id = $1 AND status IN ('queued','held')`, id, now)
		if err != nil {
			return err
		}
		if err := q.WhatsAppSendSetPhase(ctx, store.WhatsAppSendSetPhaseParams{Phase: "cancelled", CompletedAt: &now, CampaignID: id}); err != nil {
			return err
		}
		if err := q.WhatsAppCampaignSetStatus(ctx, store.WhatsAppCampaignSetStatusParams{Status: "cancelled", CompletedAt: &now, At: now, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.cancelled", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "cancelled"},
			Metadata: map[string]any{"channel": "whatsapp", "not_sent": tag.RowsAffected()}})
	})
}

// DescribeForApproval describes a WhatsApp campaign to the approvals
// domain (approvals.ChannelCampaigns).
func (s *Service) DescribeForApproval(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (approvals.ChannelCampaign, error) {
	c, err := s.Q.WhatsAppCampaign(ctx, store.WhatsAppCampaignParams{ID: campaignID, WorkspaceID: ws.ID})
	if err != nil {
		return approvals.ChannelCampaign{}, err
	}
	rv, err := s.buildReview(ctx, ws, nil, c)
	if err != nil {
		return approvals.ChannelCampaign{}, err
	}
	warns, blocks := rv.counts()
	// The snapshot is kept and compared: in English, whoever asks.
	sched, schedText := scheduleOf(i18n.Default, c, ws).Label, approvalSchedule(c, ws)
	opts, _, _ := s.listConsent(ctx, ws)
	lists, _ := audienceOf(c.Audience)
	labels := []string{}
	for _, id := range lists {
		for _, o := range opts {
			if o.ListID == id.String() {
				labels = append(labels, o.Name)
			}
		}
	}
	final := int64(len(rv.aud.list))
	snapWA := &approvals.ApprovalWhatsAppSnapshot{Template: c.TemplateName, Language: LanguageLabel(c.TemplateLanguage), Buttons: []string{},
		Variables: []approvals.ApprovalWhatsAppValue{}, Recipients: final, Cost: rv.cost.Amount, Schedule: sched}
	text := c.TemplateName
	if rv.tpl != nil {
		snapWA.Category, snapWA.Body = rv.tpl.Category, rv.tpl.Body
		for _, b := range templateView(i18n.Default, *rv.tpl).Buttons {
			snapWA.Buttons = append(snapWA.Buttons, b.Text)
		}
		var vars []WhatsAppVariable
		_ = json.Unmarshal(c.Variables, &vars)
		for _, v := range vars {
			src := v.Field
			if v.Source == "text" {
				src = "“" + v.Text + "”"
			}
			snapWA.Variables = append(snapWA.Variables, approvals.ApprovalWhatsAppValue{Key: v.Key, Source: src, Fallback: v.Fallback})
		}
		if pv := s.preview(ctx, ws, *rv.tpl, vars, rv.aud, uuid.Nil); pv != nil {
			snapWA.Preview = pv.Text
		}
		text = rv.tpl.Name + "\n" + rv.tpl.Body + "\n" + string(c.Variables)
	}
	if a, err := s.account(ctx, s.Q, ws); err == nil {
		snapWA.Business, snapWA.Number = a.DisplayName, phone.Display(a.PhoneNumber)
	}
	snap := approvals.Snapshot{Version: int(c.Version), ContentHash: c.ContentHash, Audience: c.Audience, AudienceLabels: labels, AudienceCount: final,
		AudienceBaseline: final, Route: []approvals.SnapshotShare{}, Channel: "whatsapp", Text: text, ScheduleLabel: sched, ScheduleText: schedText, Cost: rv.cost.Amount,
		Readiness: approvals.SnapshotReadiness{Verdict: rv.verdict, Passed: len(rv.checks) - warns - blocks, Warnings: warns, Blocks: blocks, Total: len(rv.checks),
			RanAt: rv.at},
		WhatsApp: snapWA}
	cur := approvals.Current{Version: int(c.Version), ContentHash: c.ContentHash, Audience: c.Audience, AudienceNames: labels, AudienceCount: final, Text: text,
		ScheduleLabel: sched, ScheduleText: schedText}
	return approvals.ChannelCampaign{Name: c.Name, Status: c.Status, Version: int(c.Version), Snapshot: snap, Current: cur, Blocks: blocks,
		SendPermission: authz.WhatsAppSend, Facts: approvals.RuleFacts{Recipients: final, Verdict: rv.verdict, Warnings: warns, Channel: "whatsapp",
			Cost: rv.cost.Amount}}, nil
} // approvalSchedule is the schedule's message for approvers, with its times
// kept so each approver reads them in their own region format.
func approvalSchedule(c store.WhatsAppCampaignRow, ws *store.Workspace) i18n.Text {
	switch c.ScheduleMode {
	case "at":
		if c.SendAt != nil {
			return msgs.M("approval.at").Keep(i18n.Args{"at": c.SendAt.In(wsLocation(ws)), "zone": tzName(ws)})
		}
	case "local":
		if c.LocalDate != nil && c.LocalMinute != nil {
			m := int(*c.LocalMinute)
			d := *c.LocalDate
			return msgs.M("approval.local").Keep(i18n.Args{"at": time.Date(d.Year(), d.Month(), d.Day(), m/60, m%60, 0, 0, time.UTC)})
		}
	}
	return mScheduleNow.Keep(nil)
}
