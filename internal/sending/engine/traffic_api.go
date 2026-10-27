package engine

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/sending/rules"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Traffic rules screen's API (Sending providers › Traffic rules; C9):
// a workspace's thresholds by bounce type (review GS-11), its mailbox
// provider response rules (GS-12), its canary defaults (GS-10) and the
// complaint watch after a send (GS-6), with what the rules did in the
// last 30 days. People who can view providers read it; people who can
// manage them change it, audited.

// TrafficRules is the screen.
type TrafficRules struct {
	Thresholds TrafficThresholds  `json:"thresholds"`
	ReplyRules []TrafficReplyRule `json:"replyRules"`
	Canary     rules.Canary       `json:"canary"`
	Watch      rules.Watch        `json:"watch"`
	// Seeds is how many seed inboxes the workspace connected (the canary
	// adds them).
	Seeds int `json:"seeds"`
	// Resumers are the people who can resume sends: alerts go to them.
	Resumers []string `json:"resumers"`
	// Usual is the median audience of the workspace's last 8 sends (the
	// canary's "over 2× the usual size"); 0 before there are any.
	Usual int64 `json:"usual"`
	// CanManage: the reader can change the rules (providers:manage).
	CanManage bool `json:"canManage"`
	// Recent is what the rules did in the last 30 days, newest first.
	Recent  []TrafficAction `json:"recent"`
	SavedAt *time.Time      `json:"savedAt,omitempty"`
	SavedBy string          `json:"savedBy,omitempty"`
}

// TrafficThresholds are the workspace's thresholds by bounce type and
// complaints per mailbox provider, in percent, each with its action.
type TrafficThresholds struct {
	BlockBounce        float64 `json:"blockBounce" minimum:"0.01" maximum:"20"`
	BlockAction        string  `json:"blockAction" enum:"pause_mailbox,slow,pause_send,warn"`
	UnknownWarn        float64 `json:"unknownWarn" minimum:"0.1" maximum:"50"`
	UnknownWarnAction  string  `json:"unknownWarnAction" enum:"warn,slow,pause_send"`
	UnknownPause       float64 `json:"unknownPause" minimum:"0.1" maximum:"50"`
	UnknownPauseAction string  `json:"unknownPauseAction" enum:"pause_send,warn,slow"`
	DeferralSlow       float64 `json:"deferralSlow" minimum:"1" maximum:"100"`
	DeferralSlowWindow int     `json:"deferralSlowWindow" enum:"5,10,15,30"`
	DeferralSlowAction string  `json:"deferralSlowAction" enum:"slow,pause_mailbox,warn"`
	Deferral           float64 `json:"deferral" minimum:"1" maximum:"100"`
	DeferralWindow     int     `json:"deferralWindow" enum:"5,10,15,30"`
	DeferralAction     string  `json:"deferralAction" enum:"pause_mailbox,slow,warn"`
	Complaint          float64 `json:"complaint" minimum:"0.01" maximum:"5" doc:"Complaints per mailbox provider, where they're reported"`
	ComplaintAction    string  `json:"complaintAction" enum:"pause_mailbox,pause_send,warn"`
}

// TrafficReplyRule is a mailbox provider response rule.
type TrafficReplyRule struct {
	// ID is empty for a rule the request adds.
	ID       string `json:"id,omitempty"`
	Mailbox  string `json:"mailbox" enum:"gmail,outlook,yahoo,icloud,other"`
	Contains string `json:"contains" minLength:"2" maxLength:"120" doc:"Matched as plain text, not case-sensitive, anywhere in the reply"`
	// Count replies within WindowMinutes; 1 acts on the first reply.
	Count         int    `json:"count" minimum:"1" maximum:"10000"`
	WindowMinutes int    `json:"windowMinutes" minimum:"1" maximum:"1440"`
	Action        string `json:"action" enum:"slow,pause,pause_route,alert"`
	// DurationMinutes is how long the action lasts; 0 until someone
	// resumes.
	DurationMinutes int    `json:"durationMinutes" minimum:"0" maximum:"10080"`
	Enabled         bool   `json:"enabled"`
	Note            string `json:"note,omitempty" maxLength:"300" doc:"What the reply means, in the rule's own words"`
	// Means is what the reply means: the note, or what ReputeCast knows of
	// the reply, in the reader's language (read only).
	Means string `json:"means,omitempty" readOnly:"true"`
	// LastMatchedAt and LastCampaign are when it last acted, and in which
	// send (read only).
	LastMatchedAt *time.Time `json:"lastMatchedAt,omitempty" readOnly:"true"`
	LastCampaign  string     `json:"lastCampaign,omitempty" readOnly:"true"`
}

// TrafficAction is one thing the rules did.
type TrafficAction struct {
	At         time.Time `json:"at"`
	CampaignID string    `json:"campaignId"`
	Campaign   string    `json:"campaign"`
	Kind       string    `json:"kind" enum:"threshold,reply_rule,canary,complaint_watch"`
	Text       string    `json:"text"`
}

// TrafficRulesInput is what the screen saves.
type TrafficRulesInput struct {
	Thresholds TrafficThresholds  `json:"thresholds"`
	ReplyRules []TrafficReplyRule `json:"replyRules" maxItems:"50"`
	Canary     rules.Canary       `json:"canary"`
	Watch      rules.Watch        `json:"watch"`
}

type trafficInput struct {
	api.WorkspaceInput
}

type trafficSaveInput struct {
	api.WorkspaceInput
	Body TrafficRulesInput
}

// RegisterTraffic adds the Traffic rules screen's operations.
func RegisterTraffic(h huma.API, e *Engine) {
	tags := []string{"Traffic rules"}
	path := "/workspaces/{workspace}/traffic-rules"
	api.Get(h, api.Op{ID: "get-traffic-rules", Path: path, Summary: "A workspace's traffic rules: thresholds by bounce type, reply rules, canary, complaint watch",
		Tags: tags, Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, _ *trafficInput) (*api.Body[TrafficRules], error) {
			ws := api.WorkspaceFrom(ctx)
			out, err := e.TrafficRules(ctx, ws.ID, i18n.LangFrom(ctx))
			if err != nil {
				return nil, err
			}
			out.CanManage = canManageProviders(ctx)
			return api.OK(out), nil
		})
	api.Put(h, api.Op{ID: "save-traffic-rules", Path: path, Summary: "Save a workspace's traffic rules (audited)", Tags: tags, Access: api.Workspace(authz.ProvidersManage)},
		func(ctx context.Context, in *trafficSaveInput) (*api.Body[TrafficRules], error) {
			ctx = ownAPI(ctx)
			ws := api.WorkspaceFrom(ctx)
			var by uuid.UUID
			if p := api.PrincipalFrom(ctx); p != nil {
				by = p.ID
			}
			if err := e.SaveTrafficRules(ctx, ws.ID, by, in.Body); err != nil {
				var se *StateError
				if errors.As(err, &se) {
					return nil, api.Invalid("body", se.In(i18n.LangFrom(ctx)))
				}
				return nil, err
			}
			out, err := e.TrafficRules(ctx, ws.ID, i18n.LangFrom(ctx))
			if err != nil {
				return nil, err
			}
			out.CanManage = canManageProviders(ctx)
			return api.OK(out), nil
		})
}

// canManageProviders reports whether the caller can change the rules.
func canManageProviders(ctx context.Context) bool {
	p := api.PrincipalFrom(ctx)
	return p != nil && p.Can(authz.ProvidersManage)
}

// trafficThresholds are a workspace's thresholds by type as the screen
// shows them.
func trafficThresholds(th Thresholds) TrafficThresholds {
	return TrafficThresholds{BlockBounce: th.BlockBounce, BlockAction: th.BlockAction, UnknownWarn: th.UnknownWarn, UnknownWarnAction: th.UnknownWarnAction,
		UnknownPause: th.UnknownPause, UnknownPauseAction: th.UnknownPauseAction, DeferralSlow: th.DeferralSlow, DeferralSlowWindow: th.DeferralSlowWindow,
		DeferralSlowAction: th.DeferralSlowAction, Deferral: th.Deferral, DeferralWindow: th.DeferralWindow, DeferralAction: th.DeferralAction,
		Complaint: th.MailboxComplaint, ComplaintAction: th.ComplaintAction}
}

// TrafficRules reads a workspace's traffic rules.
func (e *Engine) TrafficRules(ctx context.Context, ws uuid.UUID, l i18n.Lang) (TrafficRules, error) {
	out := TrafficRules{ReplyRules: []TrafficReplyRule{}, Recent: []TrafficAction{}, Resumers: []string{}}
	defaults, err := e.Q.SendingWorkspaceDefaults(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Thresholds = trafficThresholds(ParseThresholds(nil, defaults.SendThresholds))
	if out.Canary, out.Watch, err = rules.Load(ctx, e.Q, ws); err != nil {
		return out, err
	}
	if rows, err := e.Q.TrafficRulesGet(ctx, ws); err == nil && len(rows) > 0 {
		at := rows[0].UpdatedAt
		out.SavedAt = &at
		if rows[0].UpdatedBy != nil {
			if u, err := e.Q.GetUser(ctx, *rows[0].UpdatedBy); err == nil {
				out.SavedBy = u.Name
			}
		}
	}
	rrs, err := rules.ReplyRules(ctx, e.Q, ws)
	if err != nil {
		return out, err
	}
	names := map[uuid.UUID]string{}
	for _, r := range rrs {
		tr := TrafficReplyRule{ID: r.ID.String(), Mailbox: r.Mailbox, Contains: r.Contains, Count: r.Count, WindowMinutes: r.WindowMinutes, Action: r.Action,
			DurationMinutes: r.DurationMinutes, Enabled: r.Enabled, Note: r.Note, Means: r.Note, LastMatchedAt: r.LastMatchedAt}
		if k := rules.KnownReplyKey(r.Contains); tr.Means == "" && k != "" {
			tr.Means = text(l, "reply.means."+k, nil)
		}
		if r.LastCampaignID != nil {
			if _, ok := names[*r.LastCampaignID]; !ok {
				if c, err := e.Q.SendingCampaign(ctx, *r.LastCampaignID); err == nil {
					names[*r.LastCampaignID] = c.Name
				}
			}
			tr.LastCampaign = names[*r.LastCampaignID]
		}
		out.ReplyRules = append(out.ReplyRules, tr)
	}
	out.Seeds = e.seedCount(ctx, ws)
	out.Resumers = access.Helpers(ctx, e.Q, ws, authz.CampaignsSend)
	sizes, err := e.Q.TrafficRecentAudiences(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Usual = median(sizes)
	recent, err := e.Q.TrafficRecentActions(ctx, store.TrafficRecentActionsParams{WorkspaceID: ws, Since: e.now().AddDate(0, 0, -30), Lim: 20})
	if err != nil {
		return out, err
	}
	for _, r := range recent {
		out.Recent = append(out.Recent, TrafficAction{At: r.TrippedAt, CampaignID: r.CampaignID.String(), Campaign: r.CampaignName, Kind: r.Kind,
			Text: recentText(l, r)})
	}
	return out, nil
}

// recentText says what a rule did, in the reader's language ("Outlook
// complaints reached 0.18% (limit 0.10%) · paused Outlook").
func recentText(l i18n.Lang, r store.TrafficRecentActionsRow) string {
	d := parseDetail(r.Detail)
	args := i18n.Args{"kind": r.Kind, "scope": r.Scope, "mailbox": r.MailboxProvider, "metric": r.Metric, "action": firstNonEmpty(d.Action, "paused"),
		"events": d.Events, "slowRate": d.SlowedTo, "valuePct": 0.0, "limitPct": 0.0, "hasMailbox": yesNo(r.MailboxProvider != "")}
	if r.Value != nil {
		args["valuePct"] = round2(*r.Value)
	}
	if r.Threshold != nil {
		args["limitPct"] = *r.Threshold
	}
	if r.Kind == "reply_rule" && len(d.Causes) > 0 {
		// The rule's own words: "Yahoo replied TSS04 10 times in 10 min".
		args["lead"] = d.Causes[0].Lead
		if t := d.kept(); len(t.Causes) > 0 {
			args["lead"] = sentence(l, t.Causes[0].Lead)
		}
	}
	return text(l, "recent."+r.Kind, args)
}

// seedCount is how many seed inboxes a workspace connected.
func (e *Engine) seedCount(ctx context.Context, ws uuid.UUID) int {
	n, err := e.Q.SendingSeedCount(ctx, ws)
	if err != nil {
		return 0
	}
	return int(n)
}

// SaveTrafficRules writes a workspace's traffic rules: the thresholds by
// type into its auto-pause defaults, the reply rules as listed (new ones
// added, missing ones removed), the canary and the watch; audited with
// what they were.
func (e *Engine) SaveTrafficRules(ctx context.Context, ws, by uuid.UUID, in TrafficRulesInput) error {
	now := e.now()
	for i, r := range in.ReplyRules {
		r.Contains = strings.TrimSpace(r.Contains)
		if len(r.Contains) < 2 {
			return stateErr("state.replyText", nil)
		}
		if r.ID != "" {
			if _, err := uuid.Parse(r.ID); err != nil {
				return stateErr("state.replyRule", nil)
			}
		}
		in.ReplyRules[i] = r
	}
	if in.Thresholds.UnknownWarn >= in.Thresholds.UnknownPause || in.Thresholds.DeferralSlow >= in.Thresholds.Deferral {
		return stateErr("state.levelsOrder", nil)
	}
	before, err := e.TrafficRules(ctx, ws, i18n.Default)
	if err != nil {
		return err
	}
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		defaults, err := q.SendingWorkspaceDefaults(ctx, ws)
		if err != nil {
			return err
		}
		th := map[string]any{}
		_ = json.Unmarshal(defaults.SendThresholds, &th)
		t := in.Thresholds
		for k, v := range map[string]any{"block_bounce": round2(t.BlockBounce), "block_action": t.BlockAction, "unknown_warn": round2(t.UnknownWarn),
			"unknown_warn_action": t.UnknownWarnAction, "unknown_pause": round2(t.UnknownPause), "unknown_pause_action": t.UnknownPauseAction,
			"deferral_slow": round2(t.DeferralSlow), "deferral_slow_window": t.DeferralSlowWindow, "deferral_slow_action": t.DeferralSlowAction,
			"deferral": round2(t.Deferral), "deferral_window": t.DeferralWindow, "deferral_action": t.DeferralAction,
			"mailbox_complaint": math.Round(t.Complaint*100) / 100, "complaint_action": t.ComplaintAction} {
			th[k] = v
		}
		thb, _ := json.Marshal(th)
		if err := q.ConnectSetSendThresholds(ctx, store.ConnectSetSendThresholdsParams{ID: ws, SendThresholds: thb, UpdatedAt: now}); err != nil {
			return err
		}
		canary, _ := json.Marshal(in.Canary.Normalized())
		watch, _ := json.Marshal(in.Watch.Normalized())
		var byRef *uuid.UUID
		if by != uuid.Nil {
			byRef = &by
		}
		if err := q.TrafficRulesSave(ctx, store.TrafficRulesSaveParams{WorkspaceID: ws, Canary: canary, Watch: watch, UpdatedBy: byRef, UpdatedAt: now}); err != nil {
			return err
		}
		existing, err := q.TrafficReplyRules(ctx, ws)
		if err != nil {
			return err
		}
		keep := map[uuid.UUID]bool{}
		for i, r := range in.ReplyRules {
			var dur *int32
			if r.DurationMinutes > 0 {
				d := int32(r.DurationMinutes * 60)
				dur = &d
			}
			note := strings.TrimSpace(r.Note)
			if r.ID == "" {
				if err := q.TrafficReplyRuleInsert(ctx, store.TrafficReplyRuleInsertParams{ID: newID(), WorkspaceID: ws, MailboxProvider: r.Mailbox, Contains: r.Contains,
					MinCount: int32(r.Count), WindowSeconds: int32(r.WindowMinutes * 60), Action: r.Action, DurationSeconds: dur, Enabled: r.Enabled,
					Position: int32(i), Note: note, CreatedBy: byRef, CreatedAt: now}); err != nil {
					return err
				}
				continue
			}
			id := uuid.MustParse(r.ID)
			keep[id] = true
			n, err := q.TrafficReplyRuleUpdate(ctx, store.TrafficReplyRuleUpdateParams{ID: id, WorkspaceID: ws, MailboxProvider: r.Mailbox, Contains: r.Contains,
				MinCount: int32(r.Count), WindowSeconds: int32(r.WindowMinutes * 60), Action: r.Action, DurationSeconds: dur, Enabled: r.Enabled,
				Position: int32(i), Note: note})
			if err != nil {
				return err
			}
			if n == 0 {
				return stateErr("state.replyRule", nil)
			}
		}
		for _, r := range existing {
			if !keep[r.ID] {
				if _, err := q.TrafficReplyRuleDelete(ctx, store.TrafficReplyRuleDeleteParams{ID: r.ID, WorkspaceID: ws}); err != nil {
					return err
				}
			}
		}
		if !records(ctx) {
			return nil
		}
		ref, name := e.actorRef(ctx, q, by)
		after := in
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws, Action: "traffic_rules.updated", TargetType: "workspace", TargetID: ws.String(),
			TargetName: "Traffic rules", Actor: e.actorFor(ctx, ref, name), Before: trafficAudit(before.Thresholds, before.ReplyRules, before.Canary, before.Watch),
			After: trafficAudit(after.Thresholds, after.ReplyRules, after.Canary, after.Watch), Metadata: map[string]any{"summary": trafficSummary(before, in)}})
	})
}

// trafficAudit is what the audit log keeps of the rules.
func trafficAudit(th TrafficThresholds, rr []TrafficReplyRule, c rules.Canary, w rules.Watch) map[string]any {
	rs := make([]map[string]any, 0, len(rr))
	for _, r := range rr {
		rs = append(rs, map[string]any{"mailbox": r.Mailbox, "contains": r.Contains, "count": r.Count, "window_minutes": r.WindowMinutes, "action": r.Action,
			"duration_minutes": r.DurationMinutes, "enabled": r.Enabled})
	}
	return map[string]any{"thresholds": th, "reply_rules": rs, "canary": c, "watch": w}
}

// trafficSummary says what a save changed, in the audit log's English.
func trafficSummary(before TrafficRules, in TrafficRulesInput) string {
	var parts []string
	if before.Thresholds != in.Thresholds {
		parts = append(parts, "thresholds by bounce type")
	}
	b := map[string]TrafficReplyRule{}
	for _, r := range before.ReplyRules {
		b[r.ID] = r
	}
	added, removed, changed := 0, 0, 0
	seen := map[string]bool{}
	for _, r := range in.ReplyRules {
		if r.ID == "" {
			added++
			continue
		}
		seen[r.ID] = true
		o := b[r.ID]
		if o.Mailbox != r.Mailbox || o.Contains != r.Contains || o.Count != r.Count || o.WindowMinutes != r.WindowMinutes || o.Action != r.Action ||
			o.DurationMinutes != r.DurationMinutes || o.Enabled != r.Enabled {
			changed++
		}
	}
	for id := range b {
		if !seen[id] {
			removed++
		}
	}
	if added+removed+changed > 0 {
		parts = append(parts, "reply rules ("+strings.Join(nonZero(map[string]int{"added": added, "removed": removed, "changed": changed}), ", ")+")")
	}
	if before.Canary != in.Canary {
		parts = append(parts, "canary defaults")
	}
	if before.Watch != in.Watch {
		parts = append(parts, "complaint watch")
	}
	if len(parts) == 0 {
		return "No change"
	}
	return "Changed " + strings.Join(parts, ", ")
}

func nonZero(m map[string]int) []string {
	var out []string
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if m[k] > 0 {
			out = append(out, strconv.Itoa(m[k])+" "+k)
		}
	}
	return slices.Clip(out)
}

// median is the middle of some sizes (0 for none).
func median(xs []int64) int64 {
	if len(xs) == 0 {
		return 0
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	return s[len(s)/2]
}
