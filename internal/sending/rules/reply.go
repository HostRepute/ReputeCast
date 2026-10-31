package rules

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// ReplyRule is a mailbox provider response rule (review GS-12): when a
// mailbox provider's replies contain a text Count times within
// WindowMinutes, the send acts on that mailbox provider for a while.
type ReplyRule struct {
	ID       uuid.UUID `json:"id"`
	Mailbox  string    `json:"mailbox"`
	Contains string    `json:"contains"`
	// Count replies within WindowMinutes; 1 acts on the first reply.
	Count         int `json:"count"`
	WindowMinutes int `json:"windowMinutes"`
	// Action: slow (to half rate), pause (the mailbox provider on every
	// route), pause_route (on the route that got the replies, with an
	// alert) or alert.
	Action string `json:"action"`
	// DurationMinutes is how long a slowdown or pause lasts; 0 until
	// someone resumes.
	DurationMinutes int        `json:"durationMinutes"`
	Enabled         bool       `json:"enabled"`
	Note            string     `json:"note"`
	LastMatchedAt   *time.Time `json:"lastMatchedAt,omitempty"`
	LastCampaignID  *uuid.UUID `json:"lastCampaignId,omitempty"`
}

// Window is the rule's window.
func (r ReplyRule) Window() time.Duration { return time.Duration(r.WindowMinutes) * time.Minute }

// Duration is how long the rule's action lasts, 0 until someone resumes.
func (r ReplyRule) Duration() time.Duration { return time.Duration(r.DurationMinutes) * time.Minute }

// Matches reports whether a reply's text contains the rule's text (plain
// text, not case-sensitive).
func (r ReplyRule) Matches(text string) bool {
	return r.Contains != "" && strings.Contains(strings.ToLower(text), strings.ToLower(r.Contains))
}

// Mailboxes a rule can name.
var Mailboxes = []string{"gmail", "outlook", "yahoo", "icloud", "other"}

// ReplyActions are the actions a rule can take.
var ReplyActions = []string{"slow", "pause", "pause_route", "alert"}

// ReplyRules are a workspace's response rules, in their order.
func ReplyRules(ctx context.Context, q *store.Queries, ws uuid.UUID) ([]ReplyRule, error) {
	rows, err := q.TrafficReplyRules(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make([]ReplyRule, 0, len(rows))
	for _, r := range rows {
		rr := ReplyRule{ID: r.ID, Mailbox: r.MailboxProvider, Contains: r.Contains, Count: int(r.MinCount), WindowMinutes: int(r.WindowSeconds / 60),
			Action: r.Action, Enabled: r.Enabled, Note: r.Note, LastMatchedAt: r.LastMatchedAt, LastCampaignID: r.LastCampaignID}
		if r.DurationSeconds != nil {
			rr.DurationMinutes = int(*r.DurationSeconds / 60)
		}
		out = append(out, rr)
	}
	return out, nil
}

// KnownReplyKey names a reply mailbox providers are known for, so the
// screen and the incident can say what it means in the reader's language
// ("" for one it doesn't know). Yahoo's TS04 and TSS04 both defer mail for
// unexpected volume or complaints.
func KnownReplyKey(contains string) string {
	return map[string]string{
		"tss04": "tss04", "ts04": "tss04", "s3150": "s3150", "s3140": "s3150", "421-4.7.28": "gmail_rate", "4.7.28": "gmail_rate",
		"4.7.650": "outlook_throttle", "5.7.26": "dmarc", "4.7.0": "temporary",
	}[strings.ToLower(strings.TrimSpace(contains))]
}

// DefaultReplyRules are the response rules every workspace starts with
// (review GS-12): the replies mailbox providers are known for, acting as
// the Traffic rules board does. Outlook's S3150 means the sending IP is on
// Microsoft's block list, so its first one pauses Outlook on that route
// until someone resumes; Yahoo's TSS04 (and the older TS04) defers mail
// for volume or complaints, so ten in ten minutes pause Yahoo for two
// hours; Gmail's 421-4.7.28 rate-limits the sender, so five in ten minutes
// slow Gmail to half rate for 30 minutes. A rule matches plain text, so
// TS04 is its own rule (TSS04 doesn't contain it). People change or
// delete them on Traffic rules like any other rule.
var DefaultReplyRules = []ReplyRule{
	{Mailbox: "yahoo", Contains: "TSS04", Count: 10, WindowMinutes: 10, Action: "pause", DurationMinutes: 120, Enabled: true},
	{Mailbox: "yahoo", Contains: "TS04", Count: 10, WindowMinutes: 10, Action: "pause", DurationMinutes: 120, Enabled: true},
	{Mailbox: "outlook", Contains: "S3150", Count: 1, WindowMinutes: 10, Action: "pause_route", Enabled: true},
	{Mailbox: "gmail", Contains: "421-4.7.28", Count: 5, WindowMinutes: 10, Action: "slow", DurationMinutes: 30, Enabled: true},
}

// AddDefaultReplyRules gives a new workspace the default response rules,
// in the caller's transaction.
func AddDefaultReplyRules(ctx context.Context, q *store.Queries, ws uuid.UUID, now time.Time) error {
	for i, r := range DefaultReplyRules {
		var dur *int32
		if r.DurationMinutes > 0 {
			d := int32(r.DurationMinutes * 60)
			dur = &d
		}
		if err := q.TrafficReplyRuleInsert(ctx, store.TrafficReplyRuleInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, MailboxProvider: r.Mailbox,
			Contains: r.Contains, MinCount: int32(r.Count), WindowSeconds: int32(r.WindowMinutes * 60), Action: r.Action, DurationSeconds: dur,
			Enabled: r.Enabled, Position: int32(i), CreatedAt: now}); err != nil {
			return err
		}
	}
	return nil
}

// Normalized is the canary with its values in range.
func (c Canary) Normalized() Canary {
	b, _ := json.Marshal(c)
	return ParseCanary(b)
}

// Normalized is the watch with its values in range.
func (w Watch) Normalized() Watch {
	b, _ := json.Marshal(w)
	return ParseWatch(b)
}
