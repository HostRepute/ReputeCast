package push

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/push/webpush"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Shown and click tracking (brief §3.8): the service worker reports when
// the browser shows or closes a notification, and a click goes through
// ReputeCast's link (/t/p/…), which records it and goes on to the target.
// Each notification carries a random token both have to name.

func init() {
	webhooks.Events = append(webhooks.Events,
		webhooks.WebhookEventInfo{Name: "push.clicked", Group: "Push", Description: "A reader clicked a notification or one of its buttons (once per notification).", DescriptionText: msgs.M("webhook.push.clicked").Keep(nil)},
	)
}

// ErrUnknownNotification is a report or click for no notification of ours.
var ErrUnknownNotification = errors.New("push: unknown notification")

// message loads a notification and checks its token.
func (s *Service) message(ctx context.Context, id uuid.UUID, tok string) (store.PushMessageRow, error) {
	m, err := s.Q.PushMessage(ctx, id)
	if err != nil {
		return m, ErrUnknownNotification
	}
	if subtle.ConstantTimeCompare([]byte(m.Token), []byte(tok)) != 1 {
		return m, ErrUnknownNotification
	}
	return m, nil
}

// Report records what the service worker reports: "shown" or "closed".
func (s *Service) Report(ctx context.Context, ws *store.Workspace, id uuid.UUID, tok, event string) error {
	m, err := s.message(ctx, id, tok)
	if err != nil || m.WorkspaceID != ws.ID {
		return ErrUnknownNotification
	}
	now := s.now()
	return s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		var n int64
		var err error
		kind := ""
		switch event {
		case "shown":
			n, err = q.PushMessageShown(ctx, store.PushMessageShownParams{MessageID: id, At: &now})
			kind = "shown"
			if err == nil && n > 0 {
				_, err = tx.Exec(ctx, `UPDATE channel_messages SET status = 'delivered', delivered_at = $2, updated_at = $2
					WHERE id = $1 AND status IN ('sent','sending')`, id, now)
			}
		case "closed":
			n, err = q.PushMessageClosed(ctx, store.PushMessageClosedParams{MessageID: id, At: &now})
			kind = "closed"
		default:
			return nil
		}
		if err != nil || n == 0 {
			return err
		}
		if m.SubscriptionID != nil && m.Kind != "test" {
			_ = q.PushSubscriptionTouch(ctx, store.PushSubscriptionTouchParams{ID: *m.SubscriptionID, At: &now})
		}
		return q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, MessageID: id, Kind: kind,
			Detail: json.RawMessage(`{}`), OccurredAt: now})
	})
}

// clickTarget is where a notification's click (action 0) or button goes,
// with UTM parameters. Only the notification's own links are a target.
func (s *Service) clickTarget(ctx context.Context, id uuid.UUID, tok string, action int) (store.PushMessageRow, string, error) {
	m, err := s.message(ctx, id, tok)
	if err != nil {
		return m, "", err
	}
	var c Content
	if m.Body != nil {
		_ = json.Unmarshal([]byte(*m.Body), &c)
	}
	if c.Link == "" && m.CampaignID != nil {
		// The content went with the retention: the campaign still has it.
		if row, err := s.Q.PushCampaign(ctx, store.PushCampaignParams{ID: *m.CampaignID, WorkspaceID: m.WorkspaceID}); err == nil {
			c.Link, c.UTM = row.Link, utmCampaign(row.Name)
			_ = json.Unmarshal(row.Buttons, &c.Buttons)
		}
	}
	target := c.Link
	if action > 0 {
		if action > len(c.Buttons) {
			return m, "", ErrUnknownNotification
		}
		target = c.Buttons[action-1].URL
	}
	if target == "" {
		return m, "", ErrUnknownNotification
	}
	return m, withUTM(target, c.UTM), nil
}

// ClickTarget is where a click goes, without counting it: link previews
// and HEAD requests aren't readers.
func (s *Service) ClickTarget(ctx context.Context, id uuid.UUID, tok string, action int) (string, error) {
	_, target, err := s.clickTarget(ctx, id, tok, action)
	return target, err
}

// Click records a click on a notification (action 0) or one of its buttons
// (1, 2) and returns where it goes, with UTM parameters. Only the
// notification's own links are ever a target.
func (s *Service) Click(ctx context.Context, id uuid.UUID, tok string, action int, userAgent string) (string, error) {
	m, target, err := s.clickTarget(ctx, id, tok, action)
	if err != nil {
		return "", err
	}
	now := s.now()
	ws, err := s.Q.GetWorkspace(ctx, m.WorkspaceID)
	if err != nil {
		// The reader still gets the page; the click isn't counted.
		s.log().WarnContext(ctx, "push click workspace", "err", err)
		return target, nil //nolint:nilerr // see above
	}
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.PushClickInsert(ctx, store.PushClickInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: m.WorkspaceID, MessageID: id, Action: int32(action),
			Target: clip(target, 2000), OccurredAt: now}); err != nil {
			return err
		}
		first, err := q.PushMessageClicked(ctx, store.PushMessageClickedParams{MessageID: id, At: &now})
		if err != nil || first == 0 {
			return err
		}
		// A click means the browser showed it, whether or not it said so.
		if _, err := q.PushMessageShown(ctx, store.PushMessageShownParams{MessageID: id, At: &now}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'delivered', delivered_at = coalesce(delivered_at, $2), updated_at = $2
			WHERE id = $1 AND status IN ('sent','sending')`, id, now); err != nil {
			return err
		}
		if err := q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: m.WorkspaceID, MessageID: id,
			Kind: "clicked", Detail: jsonOf(map[string]any{"action": action, "user_agent": clip(userAgent, 200)}), OccurredAt: now}); err != nil {
			return err
		}
		if m.Kind == "test" {
			return nil
		}
		data := map[string]any{"message_id": id.String(), "action": action, "url": target, "browser": m.Browser}
		if m.CampaignID != nil {
			data["campaign_id"] = m.CampaignID.String()
		}
		if m.AutomationID != nil {
			data["automation_id"] = m.AutomationID.String()
		}
		if m.ContactID != nil {
			data["contact_id"] = m.ContactID.String()
		}
		_, err = webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "push.clicked", data)
		return err
	})
	if err != nil {
		s.log().WarnContext(ctx, "push click", "message", id, "err", err)
	}
	return target, nil
}

// ParseClick reads a click link's last part: "<id>.<token>".
func ParseClick(raw string) (uuid.UUID, string, bool) {
	id, tok, ok := strings.Cut(raw, ".")
	if !ok || tok == "" || len(tok) > 40 {
		return uuid.Nil, "", false
	}
	u, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, "", false
	}
	return u, tok, true
}

// PushTestInput is the member's browser subscription, made with the
// workspace's key by "Send to my browser".
type PushTestInput struct {
	Endpoint string `json:"endpoint" minLength:"1" maxLength:"1024"`
	P256dh   string `json:"p256dh" minLength:"1" maxLength:"200"`
	Auth     string `json:"auth" minLength:"1" maxLength:"100"`
}

// demoOnly reports whether every site of the workspace is a demo site:
// then its tests go to the in-process push service too.
func (s *Service) demoOnly(ctx context.Context, ws *store.Workspace) bool {
	var all, demo int64
	_ = s.Pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE demo) FROM push_sites WHERE workspace_id = $1 AND removed_at IS NULL`, ws.ID).Scan(&all, &demo)
	return all > 0 && all == demo
}

// SendTest sends the campaign to the member's own browser (logged).
func (s *Service) SendTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID, in PushTestInput, userAgent string) (PushTestResult, error) {
	c, err := s.campaign(ctx, ws, campaignID)
	if err != nil {
		return PushTestResult{}, err
	}
	l := i18n.LangFrom(ctx)
	key, err := s.liveKey(ctx, s.Q, ws)
	if errors.Is(err, errNoKeys) {
		return PushTestResult{}, api.Conflict(mProblemTestNoKeys.Text(l, nil))
	}
	if err != nil {
		return PushTestResult{}, err
	}
	sub, err := webpush.ParseSubscription(in.Endpoint, in.P256dh, in.Auth)
	if err != nil || !knownPushService(sub.Endpoint) {
		return PushTestResult{}, api.Invalid("body.endpoint", mProblemTestSubscription.Text(l, nil))
	}
	if strings.TrimSpace(c.Title) == "" {
		return PushTestResult{}, api.Invalid("body", mProblemTestTitle.Text(l, nil))
	}
	browser, platform := BrowserOf(userAgent)
	now := s.now()
	testID := uuid.Must(uuid.NewV7())
	msgID := uuid.Must(uuid.NewV7())
	content := s.campaignContent(ctx, ws, c)
	if sites, err := s.sites(ctx, ws); err == nil {
		for _, st := range sites {
			for _, id := range c.SiteIds {
				if id.String() == st.ID && content.Icon == "" {
					content.Icon = st.IconURL
				}
			}
		}
	}
	body, _ := json.Marshal(content)
	demo := s.demoOnly(ctx, ws)
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.PushTestSubscriptionUpsert(ctx, store.PushTestSubscriptionUpsertParams{UserID: p.UserID, WorkspaceID: ws.ID, KeyID: key.ID, ID: testID,
			Endpoint: sub.Endpoint, P256dh: sub.P256dh, Auth: sub.Auth, Browser: browser, Platform: platform, CreatedAt: now}); err != nil {
			return err
		}
		r := recipient{ID: testID, Browser: browser, Demo: demo}
		row := queued{id: msgID, r: r, status: "queued", after: &now, token: token()}
		if err := insertMessages(ctx, tx, ws, "test", &c.ID, nil, "", nil, body, []queued{row}, now); err != nil {
			return err
		}
		// The recipient names the member, so their last test is theirs.
		if _, err := tx.Exec(ctx, `UPDATE channel_messages SET recipient = $2 WHERE id = $1`, msgID, "test:"+p.UserID.String()+":"+testID.String()); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.test_sent", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			After: map[string]any{"channel": "push", "browser": browser, "platform": platform}})
	})
	if err != nil {
		return PushTestResult{}, err
	}
	s.kick(ctx)
	return PushTestResult{At: now, Browser: browser, Platform: platform, Status: "queued"}, nil
}
