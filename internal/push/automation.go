package push

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/store"
)

// The Send push step (brief §3.8, §5.8): a notification to the contact's
// linked subscriptions, skipped when there are none. Outside quiet hours
// it waits for the window (default) or skips. The frequency cap counts it
// with every other notification.

// Step sends the "Send push" steps (automations.ChannelStep).
type Step struct{ S *Service }

// Permission is what publishing a flow with a push step needs.
func (Step) Permission() authz.Permission { return authz.PushSend }

// Validate checks the step beyond the engine's own checks.
func (Step) Validate(n automations.AutomationNode) []automations.ChannelIssue {
	st := n.Data.Push
	out := []automations.ChannelIssue{}
	if st == nil {
		return out
	}
	if runes(st.Title) > TitleMax {
		out = append(out, automations.ChannelIssue{Code: "too_long", Message: "Keep the title to 50 characters: longer titles are cut on every platform."})
	}
	if _, ok := normalizeLink(st.Link); !ok {
		out = append(out, automations.ChannelIssue{Code: "invalid_url", Message: "Enter the address the notification opens, like news.acme.io/summit/live."})
	}
	return out
}

// Condition answers nothing: push has no branch conditions of its own.
func (Step) Condition(context.Context, pgx.Tx, automations.ChannelCheck) (bool, time.Time, error) {
	return false, time.Time{}, nil
}

// stepContent is a step's notification.
func stepContent(st *automations.AutomationPushStep, name string) Content {
	link, _ := normalizeLink(st.Link)
	c := Content{Title: strings.TrimSpace(st.Title), Message: strings.TrimSpace(st.Message), Link: link, TTLMinutes: st.ExpiryMinutes, UTM: utmCampaign(name)}
	if c.TTLMinutes <= 0 {
		c.TTLMinutes = DefaultTTL
	}
	if b := strings.TrimSpace(st.Button); b != "" && link != "" {
		c.Buttons = []PushButton{{Label: b, URL: link}}
	}
	return c
}

// Send queues the step's notification for the run's contact.
func (st Step) Send(ctx context.Context, tx pgx.Tx, in automations.ChannelSend) (automations.ChannelSent, error) {
	s := st.S
	step := in.Node.Data.Push
	if step == nil {
		return automations.ChannelSent{Skipped: "no_content"}, nil
	}
	q := store.New(tx)
	ws, err := q.GetWorkspace(ctx, in.WorkspaceID)
	if err != nil {
		return automations.ChannelSent{}, err
	}
	if _, err := s.liveKey(ctx, q, &ws); errors.Is(err, errNoKeys) {
		return automations.ChannelSent{Skipped: "push_off"}, nil
	} else if err != nil {
		return automations.ChannelSent{}, err
	}
	rows, err := tx.Query(ctx, `SELECT s.id, s.site_id, s.browser, s.time_zone, s.contact_id, s.demo, `+workerMissingSQL+` FROM push_subscriptions s
		JOIN push_sites st ON st.id = s.site_id AND st.removed_at IS NULL
		WHERE s.workspace_id = $1 AND s.contact_id = $2 AND s.removed_at IS NULL ORDER BY s.created_at`, in.WorkspaceID, in.ContactID)
	if err != nil {
		return automations.ChannelSent{}, err
	}
	list := []recipient{}
	for rows.Next() {
		var r recipient
		if err := rows.Scan(&r.ID, &r.SiteID, &r.Browser, &r.TimeZone, &r.Contact, &r.Demo, &r.WorkerMissing); err != nil {
			rows.Close()
			return automations.ChannelSent{}, err
		}
		list = append(list, r)
	}
	rows.Close()
	if len(list) == 0 {
		return automations.ChannelSent{Skipped: "no_subscription"}, nil
	}
	g, err := s.guards(ctx, q, &ws)
	if err != nil {
		return automations.ChannelSent{}, err
	}
	// Quiet hours, in the time zone of the contact's first subscription.
	if g.Quiet && !in.Waited {
		loc, _ := zoneOf(list[0], &ws)
		if !g.Window.Inside(in.Now, loc) {
			if step.Outside == "skip" {
				return automations.ChannelSent{Skipped: "quiet_hours"}, nil
			}
			next := g.Window.NextOpen(in.Now, loc)
			return automations.ChannelSent{Wait: &next}, nil
		}
	}
	counts, err := s.todayCounts(ctx, &ws, in.Now)
	if err != nil {
		return automations.ChannelSent{}, err
	}
	var name string
	_ = tx.QueryRow(ctx, `SELECT name FROM automations WHERE id = $1`, in.AutomationID).Scan(&name)
	content := stepContent(step, name)
	queue := []queued{}
	icons := map[uuid.UUID]string{}
	for _, r := range list {
		if counts[r.ID] >= g.Cap {
			continue
		}
		m := queued{id: uuid.Must(uuid.NewV7()), r: r, status: "queued", after: &in.Now, contact: r.Contact, timeZone: r.TimeZone, token: token()}
		if r.WorkerMissing {
			// Held until the site's service worker is back.
			m.status, m.reason = "held", reasonWorkerMissing
		}
		queue = append(queue, m)
		if _, ok := icons[r.SiteID]; !ok {
			if site, err := q.PushSiteAny(ctx, r.SiteID); err == nil {
				icons[r.SiteID] = s.iconURL(ctx, &ws, site)
			}
		}
	}
	if len(queue) == 0 {
		return automations.ChannelSent{Skipped: "frequency_cap"}, nil
	}
	content.Icon = icons[queue[0].r.SiteID]
	body, _ := json.Marshal(content)
	run := in.RunID
	if err := insertMessages(ctx, tx, &ws, "automation", nil, &in.AutomationID, in.Node.ID, &run, body, queue, in.Now); err != nil {
		return automations.ChannelSent{}, err
	}
	s.kick(ctx)
	return automations.ChannelSent{MessageID: queue[0].id}, nil
}

// pushStep is a live automation's push step.
type pushStep struct {
	AutomationID uuid.UUID
	Automation   string
	NodeID       string
	Title        string
}

// liveSteps lists the push steps of live automations.
func (s *Service) liveSteps(ctx context.Context, ws *store.Workspace) ([]pushStep, error) {
	rows, err := s.Pool.Query(ctx, `SELECT a.id, a.name, v.graph FROM automations a
		JOIN automation_versions v ON v.automation_id = a.id AND v.version = a.live_version
		WHERE a.workspace_id = $1 AND a.status = 'live' AND a.deleted_at IS NULL`, ws.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []pushStep{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		var raw []byte
		if err := rows.Scan(&id, &name, &raw); err != nil {
			return nil, err
		}
		var g automations.AutomationGraph
		if json.Unmarshal(raw, &g) != nil {
			continue
		}
		for _, n := range g.Nodes {
			if n.Type == automations.NodePush && n.Data.Push != nil {
				out = append(out, pushStep{AutomationID: id, Automation: name, NodeID: n.ID, Title: n.Data.Push.Title})
			}
		}
	}
	return out, rows.Err()
}

// stepTitle is a push step's notification title.
func (s *Service) stepTitle(ctx context.Context, automationID uuid.UUID, nodeID string) string {
	var raw []byte
	if err := s.Pool.QueryRow(ctx, `SELECT v.graph FROM automations a JOIN automation_versions v ON v.automation_id = a.id
		AND v.version = coalesce(a.live_version, a.draft_version) WHERE a.id = $1`, automationID).Scan(&raw); err != nil {
		return ""
	}
	var g automations.AutomationGraph
	if json.Unmarshal(raw, &g) != nil {
		return ""
	}
	for _, n := range g.Nodes {
		if n.ID == nodeID && n.Data.Push != nil {
			return n.Data.Push.Title
		}
	}
	return ""
}

// PushStepStats are a push step's numbers over 30 days.
type PushStepStats struct {
	NodeID  string           `json:"nodeId"`
	Sent    int64            `json:"sent"`
	Shown   int64            `json:"shown"`
	Clicks  int64            `json:"clicks"`
	Waiting int64            `json:"waiting"`
	Skipped map[string]int64 `json:"skipped"`
	FirstAt *time.Time       `json:"firstAt,omitempty"`
	LastAt  *time.Time       `json:"lastAt,omitempty"`
}

// PushStepStatsList is a flow's push steps' numbers.
type PushStepStatsList struct {
	Steps []PushStepStats `json:"steps"`
}

// StepStats are the flow's push steps' numbers over 30 days.
func (s *Service) StepStats(ctx context.Context, ws *store.Workspace, automationID uuid.UUID) (PushStepStatsList, error) {
	out := PushStepStatsList{Steps: []PushStepStats{}}
	since := s.now().AddDate(0, 0, -30)
	rows, err := s.Pool.Query(ctx, `SELECT m.automation_step, count(*) FILTER (WHERE m.sent_at IS NOT NULL), count(*) FILTER (WHERE p.shown_at IS NOT NULL),
			count(*) FILTER (WHERE p.clicked_at IS NOT NULL), count(*) FILTER (WHERE m.status = 'queued'), min(m.sent_at), max(m.sent_at)
		FROM channel_messages m JOIN push_messages p ON p.message_id = m.id
		WHERE m.workspace_id = $1 AND m.automation_id = $2 AND m.channel = 'push' AND m.created_at >= $3
		GROUP BY 1 ORDER BY 1`, ws.ID, automationID, since)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		st := PushStepStats{Skipped: map[string]int64{}}
		if err := rows.Scan(&st.NodeID, &st.Sent, &st.Shown, &st.Clicks, &st.Waiting, &st.FirstAt, &st.LastAt); err != nil {
			rows.Close()
			return out, err
		}
		out.Steps = append(out.Steps, st)
	}
	rows.Close()
	return out, rows.Err()
}

// PushStepEstimate is what the step's panel says about who it reaches.
type PushStepEstimate struct {
	// Contacts are the flow's contacts waiting or in it; Linked those with
	// a linked subscription.
	Contacts      int64 `json:"contacts"`
	Linked        int64 `json:"linked"`
	Subscriptions int64 `json:"subscriptions"`
	PushOn        bool  `json:"pushOn"`
	// Window and Quiet: the quiet hours the step keeps to.
	Window string `json:"window"`
	Quiet  bool   `json:"quiet"`
}

// StepEstimate counts the flow's contacts with a linked subscription.
func (s *Service) StepEstimate(ctx context.Context, ws *store.Workspace, automationID uuid.UUID) (PushStepEstimate, error) {
	out := PushStepEstimate{}
	_, err := s.liveKey(ctx, s.Q, ws)
	out.PushOn = err == nil
	g, err := s.guards(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	out.Window, out.Quiet = g.Window.String(), g.Quiet
	// The flow's contacts: those its trigger will bring in (a date field
	// set, a list's members), and those already in it.
	var trigger []byte
	if err := s.Pool.QueryRow(ctx, `SELECT trigger FROM automations WHERE id = $1 AND workspace_id = $2`, automationID, ws.ID).Scan(&trigger); err != nil {
		return out, err
	}
	var t automations.AutomationTrigger
	_ = json.Unmarshal(trigger, &t)
	err = s.Pool.QueryRow(ctx, `WITH c AS (
			SELECT r.contact_id FROM automation_runs r WHERE r.automation_id = $1 AND r.status IN ('active','waiting')
			UNION
			SELECT ct.id FROM contacts ct WHERE $3 = 'date_field' AND ct.workspace_id = $2 AND ct.status = 'active' AND ct.erased_at IS NULL
			  AND (ct.attributes ->> $4) ~ '^\d{4}-\d{2}-\d{2}'
			UNION
			SELECT lm.contact_id FROM list_memberships lm WHERE $3 = 'joined_list' AND lm.list_id::text = $5 AND lm.status = 'subscribed')
		SELECT (SELECT count(*) FROM c),
			(SELECT count(DISTINCT s.contact_id) FROM push_subscriptions s JOIN c ON c.contact_id = s.contact_id WHERE s.workspace_id = $2 AND s.removed_at IS NULL),
			(SELECT count(*) FROM push_subscriptions s JOIN c ON c.contact_id = s.contact_id WHERE s.workspace_id = $2 AND s.removed_at IS NULL)`,
		automationID, ws.ID, t.Kind, t.Field, t.ListID).Scan(&out.Contacts, &out.Linked, &out.Subscriptions)
	return out, err
}
