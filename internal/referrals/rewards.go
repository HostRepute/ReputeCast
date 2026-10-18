package referrals

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// webhookBackoff spreads a webhook reward's attempts over 6 hours, as
// outgoing webhooks do.
var webhookBackoff = webhooks.Backoff

var (
	mNotManual  = msgs.M("reward.notManual")
	mNotFulfill = msgs.M("reward.notFulfill")
	mCantResend = msgs.M("reward.cantResend")
	mNoClaim    = msgs.M("reward.noClaim")
	mNotFailed  = msgs.M("reward.notFailed")
	mNoEmail    = msgs.M("milestone.noEmail")
)

// issueReward gives a referrer a milestone's reward, once: a download is
// available at once, a code is taken from the list (or waits for codes), a
// manual reward waits to be sent by hand, and a coupon or a webhook goes
// out from the rewards job. "Reached a milestone" automations start either
// way.
func (s *Service) issueReward(ctx context.Context, tx pgx.Tx, ws *store.Workspace, m store.ReferralMilestone, linkID, contactID uuid.UUID, counted int64, at time.Time) error {
	q := store.New(tx)
	status := RewardSent
	switch m.RewardType {
	case RewardManual:
		status = RewardToFulfill
	case RewardCoupon, RewardWebhook:
		status = RewardPending
	case RewardCodes:
		status = RewardWaitingCodes
	}
	mid := m.ID
	id := uuid.Must(uuid.NewV7())
	if _, err := q.ReferralRewardInsert(ctx, store.ReferralRewardInsertParams{ID: id, WorkspaceID: ws.ID, ProgramID: m.ProgramID, MilestoneID: &mid,
		ContactID: contactID, LinkID: &linkID, MilestoneCount: m.Count, RewardName: m.RewardName, RewardType: m.RewardType, Status: status, Now: at}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // given before
		}
		return err
	}
	if status == RewardSent {
		if err := q.ReferralRewardSet(ctx, store.ReferralRewardSetParams{ID: id, Status: RewardSent, SentAt: &at, Now: at}); err != nil {
			return err
		}
	}
	if m.RewardType == RewardCodes {
		var err error
		if status, err = s.takeCode(ctx, tx, id, m.ID, at); err != nil {
			return err
		}
	}
	link := linkID
	contact := contactID
	if err := writeLog(ctx, q, ws.ID, logEntry{program: m.ProgramID, link: &link, contact: &contact, kind: EventMilestoneReached, at: at,
		detail: map[string]any{"milestone": m.ID.String(), "count": m.Count, "reward": m.RewardName, "rewardId": id.String(), "counted": counted}}); err != nil {
		return err
	}
	referrer := s.contactHook(ctx, tx, contactID)
	if err := emitHook(ctx, tx, ws, HookMilestoneReached, map[string]any{"program": m.ProgramID.String(),
		"milestone": map[string]any{"id": m.ID.String(), "count": m.Count, "reward": m.RewardName}, "referrer": referrer, "counted": counted,
		"reward": map[string]any{"id": id.String(), "type": m.RewardType, "status": status}}); err != nil {
		return err
	}
	if err := emitMilestoneTrigger(ctx, q, ws.ID, contactID, m.ProgramID, m, at); err != nil {
		return err
	}
	if status == RewardSent || status == RewardToFulfill {
		return s.rewardOut(ctx, tx, ws, id, at)
	}
	return nil
}

func (s *Service) contactHook(ctx context.Context, tx pgx.Tx, id uuid.UUID) hookContact {
	h := hookContact{ContactID: id.String()}
	_ = tx.QueryRow(ctx, `SELECT email, first_name FROM contacts WHERE id = $1`, id).Scan(&h.Email, &h.FirstName)
	return h
}

// takeCode gives a code reward the oldest free code, or leaves it waiting.
func (s *Service) takeCode(ctx context.Context, tx pgx.Tx, rewardID, milestoneID uuid.UUID, at time.Time) (string, error) {
	q := store.New(tx)
	row, err := q.ReferralCodeTake(ctx, store.ReferralCodeTakeParams{RewardID: &rewardID, MilestoneID: milestoneID, Now: at})
	if errors.Is(err, pgx.ErrNoRows) {
		return RewardWaitingCodes, nil
	}
	if err != nil {
		return "", err
	}
	if err := q.ReferralRewardSet(ctx, store.ReferralRewardSetParams{ID: rewardID, Status: RewardSent, SentAt: &at, CodeID: &row.ID, Now: at}); err != nil {
		return "", err
	}
	return RewardSent, nil
}

// rewardOut records that a reward reached its referrer: the referral log,
// the reward_sent webhook event and the milestone email when it has one.
func (s *Service) rewardOut(ctx context.Context, tx pgx.Tx, ws *store.Workspace, rewardID uuid.UUID, at time.Time) error {
	q := store.New(tx)
	rw, err := q.ReferralRewardByID(ctx, rewardID)
	if err != nil {
		return err
	}
	contact := rw.ContactID
	if err := writeLog(ctx, q, ws.ID, logEntry{program: rw.ProgramID, link: rw.LinkID, contact: &contact, kind: EventRewardSent, at: at,
		detail: map[string]any{"rewardId": rw.ID.String(), "reward": rw.RewardName, "milestone": rw.MilestoneID, "count": rw.MilestoneCount, "status": rw.Status}}); err != nil {
		return err
	}
	if err := emitHook(ctx, tx, ws, HookRewardSent, map[string]any{"program": rw.ProgramID.String(),
		"reward":   map[string]any{"id": rw.ID.String(), "milestone": rw.MilestoneCount, "name": rw.RewardName, "type": rw.RewardType, "status": rw.Status},
		"referrer": s.contactHook(ctx, tx, rw.ContactID)}); err != nil {
		return err
	}
	return s.queueEmail(ctx, tx, ws, rw, at)
}

// queueEmail sends a reward's milestone email, when its milestone has one.
func (s *Service) queueEmail(ctx context.Context, tx pgx.Tx, ws *store.Workspace, rw store.ReferralReward, at time.Time) error {
	if rw.MilestoneID == nil || s.Emails == nil || s.Emails() == nil {
		return nil
	}
	q := store.New(tx)
	m, err := q.ReferralMilestoneByID(ctx, *rw.MilestoneID)
	if err != nil || !m.EmailOn || m.EmailID == nil || m.DeletedAt != nil {
		return nil //nolint:nilerr // no email for this milestone
	}
	var email string
	var erased *time.Time
	if err := tx.QueryRow(ctx, `SELECT email, erased_at FROM contacts WHERE id = $1`, rw.ContactID).Scan(&email, &erased); err != nil || erased != nil {
		return nil //nolint:nilerr // the referrer is gone
	}
	msgID, err := s.Emails().QueueMilestoneEmail(ctx, tx, ws.ID, *m.EmailID, rw.ContactID, email)
	if err != nil {
		return err
	}
	return q.ReferralRewardEmailSent(ctx, store.ReferralRewardEmailSentParams{ID: rw.ID, Now: at, MessageID: &msgID})
}

// releaseWaitingCodes gives the rewards that waited for codes the codes
// just uploaded, oldest first.
func (s *Service) releaseWaitingCodes(ctx context.Context, milestoneID uuid.UUID) (int64, error) {
	var released int64
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id, workspace_id FROM referral_rewards WHERE milestone_id = $1 AND status = 'waiting_codes'
			ORDER BY reached_at FOR UPDATE SKIP LOCKED`, milestoneID)
		if err != nil {
			return err
		}
		type waiting struct{ id, ws uuid.UUID }
		var list []waiting
		for rows.Next() {
			var w waiting
			if err := rows.Scan(&w.id, &w.ws); err != nil {
				rows.Close()
				return err
			}
			list = append(list, w)
		}
		rows.Close()
		q := store.New(tx)
		cache := map[uuid.UUID]*store.Workspace{}
		now := s.now()
		for _, w := range list {
			status, err := s.takeCode(ctx, tx, w.id, milestoneID, now)
			if err != nil {
				return err
			}
			if status != RewardSent {
				break
			}
			ws, err := workspaceOf(ctx, q, cache, w.ws)
			if err != nil {
				return err
			}
			if err := s.rewardOut(ctx, tx, ws, w.id, now); err != nil {
				return err
			}
			released++
		}
		return nil
	})
	return released, err
}

// DeliverDue sends the rewards the job owes: webhook calls that are due,
// coupons, and rewards waiting for codes that have arrived since.
func (s *Service) DeliverDue(ctx context.Context) (int, error) {
	now := s.now()
	var due []store.ReferralReward
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		rows, err := store.New(tx).ReferralRewardsDue(ctx, store.ReferralRewardsDueParams{Now: now, Lim: 50})
		if err != nil {
			return err
		}
		due = rows
		// Claim the webhook calls for a minute so another worker leaves them.
		for _, r := range rows {
			if r.Status == RewardPending {
				next := now.Add(time.Minute)
				if _, err := tx.Exec(ctx, `UPDATE referral_rewards SET next_attempt_at = $2 WHERE id = $1`, r.ID, next); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	done := 0
	for _, r := range due {
		var err error
		switch {
		case r.Status == RewardWaitingCodes && r.MilestoneID != nil:
			var n int64
			n, err = s.releaseWaitingCodes(ctx, *r.MilestoneID)
			done += int(n)
		case r.RewardType == RewardWebhook:
			err = s.callWebhook(ctx, r, now)
			done++
		case r.RewardType == RewardCoupon:
			err = s.issueCoupon(ctx, r, now)
			done++
		}
		if err != nil {
			s.log().WarnContext(ctx, "referral reward", "reward", r.ID, "err", err)
		}
	}
	return done, nil
}

// issueCoupon creates a coupon reward's single-use promotion code through
// the payment provider; without one it stays pending ("Connect Stripe").
func (s *Service) issueCoupon(ctx context.Context, r store.ReferralReward, now time.Time) error {
	if s.Coupons == nil || s.Coupons() == nil || r.MilestoneID == nil {
		_, err := s.Pool.Exec(ctx, `UPDATE referral_rewards SET next_attempt_at = $2 WHERE id = $1`, r.ID, now.Add(time.Hour))
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, r.WorkspaceID)
	if err != nil {
		return err
	}
	m, err := s.Q.ReferralMilestoneByID(ctx, *r.MilestoneID)
	if err != nil {
		return err
	}
	c, err := s.Coupons().Issue(ctx, &ws, m.CouponOfferID, r.ContactID, int(m.CouponValidDays))
	if err != nil {
		next := now.Add(15 * time.Minute)
		_, uerr := s.Pool.Exec(ctx, `UPDATE referral_rewards SET next_attempt_at = $2, last_error = $3, attempts = attempts + 1 WHERE id = $1`, r.ID, next, err.Error())
		return errors.Join(err, uerr)
	}
	sealed, err := s.Box.SealString(c.Code, answerAD(r.ID))
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		exp := c.ExpiresAt
		if err := q.ReferralRewardSet(ctx, store.ReferralRewardSetParams{ID: r.ID, Status: RewardSent, SentAt: &now, ClaimUrl: c.ClaimURL, ExpiresAt: &exp,
			Answer: sealed, Attempts: r.Attempts + 1, Now: now}); err != nil {
			return err
		}
		return s.rewardOut(ctx, tx, &ws, r.ID, now)
	})
}

func answerAD(rewardID uuid.UUID) string { return "referral_reward:" + rewardID.String() + ":answer" }

// rewardAnswer is what a webhook reward's endpoint answered: a code or an
// address the milestone email and the hub show.
type rewardAnswer struct {
	Code string `json:"code,omitempty"`
	URL  string `json:"url,omitempty"`
}

// WebhookCall is one call of a webhook reward's endpoint.
type WebhookCall struct {
	Status int
	Answer rewardAnswer
	Err    error
}

// webhookBody is the referral.milestone_reached event a webhook reward
// posts, signed like outgoing webhooks.
func (s *Service) webhookBody(ws *store.Workspace, deliveryID string, m store.ReferralMilestone, referrer hookContact, rewardID string, counted int64, at time.Time) []byte {
	body, _ := json.Marshal(webhooks.Envelope{ID: deliveryID, Type: HookMilestoneReached, CreatedAt: at.UTC(), Workspace: ws.Slug, Data: map[string]any{
		"program":   m.ProgramID.String(),
		"milestone": map[string]any{"id": m.ID.String(), "count": m.Count, "reward": m.RewardName},
		"referrer":  referrer, "counted": counted, "reward": map[string]any{"id": rewardID, "type": RewardWebhook},
	}})
	return body
}

func (s *Service) client() *http.Client {
	if s.HTTP != nil {
		if c := s.HTTP(); c != nil {
			return c
		}
	}
	return netguard.Policy{AllowPrivate: s.allowPrivate()}.Client(15 * time.Second)
}

// post sends a signed body to a webhook reward's endpoint and reads its
// answer.
func (s *Service) post(ctx context.Context, m store.ReferralMilestone, deliveryID string, body []byte, now time.Time) WebhookCall {
	if _, err := (netguard.Policy{AllowPrivate: s.allowPrivate()}).ValidateURL(m.WebhookUrl); err != nil {
		return WebhookCall{Err: err}
	}
	secret := ""
	if len(m.WebhookSecret) > 0 {
		v, err := s.Box.OpenString(m.WebhookSecret, secretAD(m.ID))
		if err != nil {
			return WebhookCall{Err: err}
		}
		secret = v
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.WebhookUrl, bytes.NewReader(body))
	if err != nil {
		return WebhookCall{Err: err}
	}
	ts := now.Unix()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ReputeCast/"+version.Version+" (referral reward)")
	req.Header.Set(webhooks.HeaderEvent, HookMilestoneReached)
	req.Header.Set(webhooks.HeaderDelivery, deliveryID)
	req.Header.Set(webhooks.HeaderTimestamp, strconv.FormatInt(ts, 10))
	if secret != "" {
		req.Header.Set(webhooks.HeaderSignature, webhooks.Sign(secret, ts, body))
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return WebhookCall{Err: err}
	}
	defer resp.Body.Close() //nolint:errcheck
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
	call := WebhookCall{Status: resp.StatusCode}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		call.Err = fmt.Errorf("answered %d", resp.StatusCode)
		return call
	}
	_ = json.Unmarshal(raw, &call.Answer)
	call.Answer.Code = strings.TrimSpace(call.Answer.Code)
	if u := strings.TrimSpace(call.Answer.URL); strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
		call.Answer.URL = u
	} else {
		call.Answer.URL = ""
	}
	if len(call.Answer.Code) > 200 {
		call.Answer.Code = call.Answer.Code[:200]
	}
	return call
}

// callWebhook makes a webhook reward's call: on success the reward goes out
// with what the endpoint answered; failures are retried for 6 hours, then
// the reward is Failed (with Retry).
func (s *Service) callWebhook(ctx context.Context, r store.ReferralReward, now time.Time) error {
	if r.MilestoneID == nil {
		return nil
	}
	m, err := s.Q.ReferralMilestoneByID(ctx, *r.MilestoneID)
	if err != nil {
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, r.WorkspaceID)
	if err != nil {
		return err
	}
	var referrer hookContact
	_ = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error { referrer = s.contactHook(ctx, tx, r.ContactID); return nil })
	counted, _ := s.Q.ReferralCountedOf(ctx, store.ReferralCountedOfParams{ProgramID: r.ProgramID, LinkID: r.LinkID})
	body := s.webhookBody(&ws, r.ID.String(), m, referrer, r.ID.String(), counted, now)
	call := s.post(ctx, m, r.ID.String(), body, now)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var status *int32
		if call.Status > 0 {
			v := int32(call.Status)
			status = &v
		}
		if err := q.ReferralMilestoneWebhookResult(ctx, store.ReferralMilestoneWebhookResultParams{ID: m.ID, Status: status, Now: now, Ok: call.Err == nil}); err != nil {
			return err
		}
		attempts := r.Attempts + 1
		if call.Err == nil {
			var sealed []byte
			if call.Answer.Code != "" || call.Answer.URL != "" {
				raw, _ := json.Marshal(call.Answer)
				var err error
				if sealed, err = s.Box.Seal(raw, answerAD(r.ID)); err != nil {
					return err
				}
			}
			if err := q.ReferralRewardSet(ctx, store.ReferralRewardSetParams{ID: r.ID, Status: RewardSent, SentAt: &now, Answer: sealed, Attempts: attempts, Now: now}); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE referral_milestones SET webhook_alerted_at = NULL WHERE id = $1`, m.ID); err != nil {
				return err
			}
			return s.rewardOut(ctx, tx, &ws, r.ID, now)
		}
		msg := call.Err.Error()
		if netguard.IsBlocked(call.Err) {
			msg = netguard.Describe(ctx, call.Err)
		}
		if int(attempts) >= webhooks.MaxAttempts || now.Sub(r.ReachedAt) > WebhookRetryFor+time.Hour {
			if err := q.ReferralRewardSet(ctx, store.ReferralRewardSetParams{ID: r.ID, Status: RewardFailed, LastError: msg, Attempts: attempts, FailedAt: &now, Now: now}); err != nil {
				return err
			}
			contact := r.ContactID
			if err := writeLog(ctx, q, ws.ID, logEntry{program: r.ProgramID, link: r.LinkID, contact: &contact, kind: EventRewardFailed, at: now,
				detail: map[string]any{"rewardId": r.ID.String(), "reward": r.RewardName, "error": msg}}); err != nil {
				return err
			}
			if err := emitHook(ctx, tx, &ws, HookRewardFailed, map[string]any{"program": r.ProgramID.String(),
				"reward":   map[string]any{"id": r.ID.String(), "milestone": r.MilestoneCount, "name": r.RewardName, "type": r.RewardType, "error": msg},
				"referrer": referrer}); err != nil {
				return err
			}
		} else {
			next := now.Add(webhookBackoff[min(int(attempts)-1, len(webhookBackoff)-1)])
			if err := q.ReferralRewardSet(ctx, store.ReferralRewardSetParams{ID: r.ID, Status: RewardPending, LastError: msg, Attempts: attempts, NextAttemptAt: &next, Now: now}); err != nil {
				return err
			}
		}
		return s.alertWebhook(ctx, tx, &ws, m, call.Status, int(m.WebhookFailures)+1, now)
	})
}

// alertWebhook tells members with grow:edit, once per run of failures, that
// a reward webhook is failing.
func (s *Service) alertWebhook(ctx context.Context, tx pgx.Tx, ws *store.Workspace, m store.ReferralMilestone, status, calls int, now time.Time) error {
	if m.WebhookAlertedAt != nil || calls < 3 {
		return nil
	}
	l := i18n.For(i18n.English)
	st := strconv.Itoa(status)
	if status == 0 {
		st = "nothing"
	}
	n := notify.Notification{Kind: notify.KindRewardWebhookFailing, Severity: "bad",
		Title:   msgs.M("notify.webhookTitle").Text(l, i18n.Args{"reward": m.RewardName}),
		Summary: msgs.M("notify.webhookBody").Text(l, i18n.Args{"url": m.WebhookUrl, "status": st, "calls": calls}),
		Link:    "/w/" + ws.Slug + "/grow/referrals/milestones?edit=" + m.ID.String(),
		Data:    map[string]any{"milestone": m.ID.String(), "status": status, "calls": calls}}
	if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.GrowEdit, n); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE referral_milestones SET webhook_alerted_at = $2 WHERE id = $1`, m.ID, now)
	return err
}

// --- Actions people take on rewards -----------------------------------------------

// RetryReward tries a failed webhook reward again.
func (s *Service) RetryReward(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	rw, err := s.Q.ReferralRewardGet(ctx, store.ReferralRewardGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return api.MapError(err, "Reward")
	}
	if rw.Status != RewardFailed {
		return api.Conflict(s.text(ctx, mNotFailed, nil))
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralRewardSet(ctx, store.ReferralRewardSetParams{ID: id, Status: RewardPending, NextAttemptAt: &now, Now: now}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE referral_rewards SET reached_at = $2 WHERE id = $1`, id, now); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.reward_retried", TargetType: "referral_reward",
			TargetID: id.String(), TargetName: rw.RewardName, Metadata: map[string]any{"related": rw.ContactID.String()}})
	})
}

// ResendReward sends a reward's milestone email again (a coupon's claim
// link, a download's fresh link).
func (s *Service) ResendReward(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	rw, err := s.Q.ReferralRewardGet(ctx, store.ReferralRewardGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return api.MapError(err, "Reward")
	}
	if rw.Status != RewardSent && rw.Status != RewardFulfilled && rw.Status != RewardToFulfill && rw.Status != RewardExpired {
		return api.Conflict(s.text(ctx, mCantResend, nil))
	}
	if rw.MilestoneID == nil {
		return api.Conflict(s.text(ctx, mNoEmail, nil))
	}
	m, err := s.Q.ReferralMilestoneByID(ctx, *rw.MilestoneID)
	if err != nil || !m.EmailOn || m.EmailID == nil {
		return api.Conflict(s.text(ctx, mNoEmail, nil))
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := s.queueEmail(ctx, tx, ws, rw, now); err != nil {
			return err
		}
		if err := q.ReferralRewardResent(ctx, store.ReferralRewardResentParams{ID: id, Now: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.reward_resent", TargetType: "referral_reward",
			TargetID: id.String(), TargetName: rw.RewardName, Metadata: map[string]any{"related": rw.ContactID.String()}})
	})
}

// ReferralFulfillInput marks a manual reward fulfilled, with a note the
// referrer sees (a tracking number).
type ReferralFulfillInput struct {
	Note string `json:"note,omitempty" maxLength:"500"`
}

// FulfillReward marks a manual reward sent by hand. Its shipping address is
// deleted 30 days later.
func (s *Service) FulfillReward(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ReferralFulfillInput) error {
	rw, err := s.Q.ReferralRewardGet(ctx, store.ReferralRewardGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return api.MapError(err, "Reward")
	}
	if rw.RewardType != RewardManual {
		return api.Conflict(s.text(ctx, mNotManual, nil))
	}
	if rw.Status != RewardToFulfill {
		return api.Conflict(s.text(ctx, mNotFulfill, nil))
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralRewardFulfill(ctx, store.ReferralRewardFulfillParams{ID: id, Now: now, By: actorUser(ctx), Note: strings.TrimSpace(in.Note),
			PurgeAt: now.Add(AddressKeptFor)}); err != nil {
			return err
		}
		contact := rw.ContactID
		if err := writeLog(ctx, q, ws.ID, logEntry{program: rw.ProgramID, link: rw.LinkID, contact: &contact, kind: EventRewardFulfilled, at: now,
			detail: map[string]any{"rewardId": id.String(), "reward": rw.RewardName}}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.reward_fulfilled", TargetType: "referral_reward",
			TargetID: id.String(), TargetName: rw.RewardName, Reason: strings.TrimSpace(in.Note), Metadata: map[string]any{"related": rw.ContactID.String()}})
	})
}

// ReferralClaimLink is a coupon reward's claim link.
type ReferralClaimLink struct {
	URL       string     `json:"url"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// ClaimLink returns a coupon reward's claim link to copy; copying it is
// audited.
func (s *Service) ClaimLink(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ReferralClaimLink, error) {
	rw, err := s.Q.ReferralRewardGet(ctx, store.ReferralRewardGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ReferralClaimLink{}, api.MapError(err, "Reward")
	}
	if rw.ClaimUrl == "" {
		return ReferralClaimLink{}, api.Conflict(s.text(ctx, mNoClaim, nil))
	}
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "referral.claim_link_copied", TargetType: "referral_reward",
		TargetID: id.String(), TargetName: rw.RewardName, Metadata: map[string]any{"related": rw.ContactID.String()}})
	return ReferralClaimLink{URL: rw.ClaimUrl, ExpiresAt: rw.ExpiresAt}, err
}

// openAddress reads a manual reward's sealed shipping address.
func (s *Service) openAddress(rewardID uuid.UUID, sealed []byte) *ReferralShippingAddress {
	if len(sealed) == 0 {
		return nil
	}
	raw, err := s.Box.Open(sealed, addressAD(rewardID))
	if err != nil {
		return nil
	}
	var a ReferralShippingAddress
	if json.Unmarshal(raw, &a) != nil {
		return nil
	}
	return &a
}

func addressAD(rewardID uuid.UUID) string { return "referral_reward:" + rewardID.String() + ":address" }

// DownloadToken is a reward's expiring download link token: the hub makes
// a fresh one each time.
func (s *Service) DownloadToken(wsID, rewardID, contactID uuid.UUID, at time.Time) string {
	if s.Links == nil {
		return ""
	}
	return s.Links.Sign(linktoken.Claims{Kind: linktoken.RewardDownload, WorkspaceID: wsID, MessageID: rewardID, ContactID: contactID, Link: linktoken.IssueDay(at)})
}

// ExpiredLink reports whether a day-stamped token is older than valid.
func ExpiredLink(c linktoken.Claims, valid time.Duration, now time.Time) bool {
	issued := linktoken.IssuedOn(c.Link)
	return now.Sub(issued) > valid+24*time.Hour
}

// rewardCode is the code a reward took, or the code or address its webhook
// answered (a coupon's promotion code is its answer).
func (s *Service) rewardCode(ctx context.Context, rw store.ReferralReward) (code, url string) {
	if rw.CodeID != nil && rw.MilestoneID != nil {
		if sealed, err := s.Q.ReferralCodeOf(ctx, *rw.CodeID); err == nil {
			if c, err := s.Box.OpenString(sealed, codeAD(*rw.MilestoneID)); err == nil {
				code = c
			}
		}
	}
	if len(rw.Answer) == 0 {
		return code, url
	}
	raw, err := s.Box.Open(rw.Answer, answerAD(rw.ID))
	if err != nil {
		return code, url
	}
	if rw.RewardType == RewardCoupon {
		return string(raw), url
	}
	var a rewardAnswer
	if json.Unmarshal(raw, &a) == nil {
		if a.Code != "" {
			code = a.Code
		}
		url = a.URL
	}
	return code, url
}

// TestWebhook posts a sample referral.milestone_reached to a webhook
// reward's endpoint, signed like the real calls, and shows the answer. It
// changes neither the reward's health nor any referrer's reward.
func (s *Service) TestWebhook(ctx context.Context, ws *store.Workspace, milestoneID uuid.UUID) (ReferralWebhookTestResult, error) {
	m, err := s.Q.ReferralMilestoneByID(ctx, milestoneID)
	if err != nil || m.WorkspaceID != ws.ID || m.DeletedAt != nil {
		return ReferralWebhookTestResult{}, api.NotFound("Milestone")
	}
	if m.RewardType != RewardWebhook || m.WebhookUrl == "" {
		return ReferralWebhookTestResult{}, api.Invalid("milestone", s.text(ctx, msgs.M("milestone.webhookMissing"), nil))
	}
	now := s.now()
	id := "test_" + uuid.Must(uuid.NewV7()).String()
	sample := hookContact{ContactID: "00000000-0000-0000-0000-000000000000", Email: "reader@example.com", FirstName: "Sam"}
	body := s.webhookBody(ws, id, m, sample, id, int64(m.Count), now)
	call := s.post(ctx, m, id, body, now)
	out := ReferralWebhookTestResult{OK: call.Err == nil, Status: call.Status, Code: call.Answer.Code, URL: call.Answer.URL}
	if call.Err != nil {
		out.Error = netguard.Describe(ctx, call.Err)
		if call.Status > 0 {
			out.Error = s.text(ctx, msgs.M("milestone.webhookAnswered"), i18n.Args{"status": call.Status})
		}
	}
	result := audit.Allowed
	if !out.OK {
		result = audit.Failed
	}
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "referral.webhook_tested", TargetType: "referral_milestone", TargetID: m.ID.String(),
		TargetName: m.RewardName, Result: result, Metadata: map[string]any{"status": call.Status, "url": m.WebhookUrl}})
	return out, err
}
