package referrals_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// fixture is a workspace with a list, a sender and a referrer (Sam), and a
// referrals service on a clock the test moves.
type fixture struct {
	t     *testing.T
	env   *apptest.Env
	s     *referrals.Service
	hook  referrals.Hook
	ws    store.Workspace
	owner store.User
	list  uuid.UUID
	other uuid.UUID
	sam   uuid.UUID

	mu  sync.Mutex
	now time.Time
}

func (f *fixture) clock() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fixture) advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.mu.Unlock()
}

func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		f.t.Fatalf("%s: %v", sql, err)
	}
}

type mailer struct{ e *engine.Engine }

func (m mailer) QueueMilestoneEmail(ctx context.Context, tx pgx.Tx, ws uuid.UUID, emailID, contactID uuid.UUID, to string) (uuid.UUID, error) {
	q, err := engine.AutomationMailer{E: m.e}.QueueAutomationEmail(ctx, tx, automations.OutgoingEmail{WorkspaceID: ws, ContactID: contactID, To: to, EmailID: emailID, NodeID: "milestone"})
	return q.MessageID, err
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	env := apptest.New(t)
	f := &fixture{t: t, env: env, now: time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)}
	f.owner = env.CreateUser(apptest.UserOpts{Email: "sofia@grow.test", Name: "Sofia Rossi"})
	f.ws = env.CreateWorkspace("Grow Test", f.owner.ID)
	f.list, f.other = uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	f.exec(`INSERT INTO lists (id, workspace_id, name) VALUES ($1, $3, 'Weekly Digest'), ($2, $3, 'Product Updates')`, f.list, f.other, f.ws.ID)
	dom, sender := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	f.exec(`INSERT INTO sending_domains (id, workspace_id, domain, status) VALUES ($1, $2, 'news.grow.test', 'verified')`, dom, f.ws.ID)
	f.exec(`INSERT INTO senders (id, workspace_id, domain_id, name, email, verification) VALUES ($1, $2, $3, 'Grow Test', 'hello@news.grow.test', '{"demo": "verified"}')`, sender, f.ws.ID, dom)
	f.exec(`UPDATE workspaces SET default_sender_id = $2 WHERE id = $1`, f.ws.ID, sender)
	ws, err := env.App.Q.GetWorkspace(context.Background(), f.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.ws = ws
	f.s = &referrals.Service{
		Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Links: env.App.Links, Now: f.clock,
		Pages: func(context.Context, *store.Workspace) (forms.PageLinks, error) {
			return forms.LinksOn("https://news.grow.test", f.ws.Slug, true), nil
		},
		Emails:       func() referrals.MilestoneSender { return mailer{e: env.App.SendingEngine()} },
		AllowPrivate: func() bool { return true },
		Base:         func() string { return "https://reputecast.grow.test" },
	}
	f.hook = referrals.Hook{S: func() *referrals.Service { return f.s }}
	f.sam = f.subscriber("sam.rivera@outlook.com", "Sam", "Rivera", f.list)
	return f
}

// subscriber is a confirmed subscriber of lists.
func (f *fixture) subscriber(email, first, last string, lists ...uuid.UUID) uuid.UUID {
	f.t.Helper()
	id := uuid.Must(uuid.NewV7())
	f.exec(`INSERT INTO contacts (id, workspace_id, email, first_name, last_name, created_at) VALUES ($1, $2, $3, $4, $5, $6)`, id, f.ws.ID, email, first, last, f.clock().Add(-30*24*time.Hour))
	for _, l := range lists {
		f.exec(`INSERT INTO list_memberships (list_id, contact_id, status, subscribed_at, created_at) VALUES ($1, $2, 'subscribed', $3, $3)`, l, id, f.clock().Add(-30*24*time.Hour))
	}
	return id
}

// program starts a program on the list with milestones, and turns it on.
func (f *fixture) program(milestones ...referrals.ReferralMilestoneInput) referrals.ReferralProgramDetail {
	f.t.Helper()
	ctx := context.Background()
	p, err := f.s.Create(ctx, &f.ws, referrals.ReferralCreateInput{ListID: f.list.String()})
	if err != nil {
		f.t.Fatal(err)
	}
	id := uuid.MustParse(p.ID)
	for _, m := range milestones {
		if _, err := f.s.CreateMilestone(ctx, &f.ws, id, m); err != nil {
			f.t.Fatalf("milestone %d: %v", m.Count, err)
		}
	}
	on, err := f.s.TurnOn(ctx, &f.ws, id)
	if err != nil {
		f.t.Fatalf("turn on: %v", err)
	}
	if on.LinksCreated < 1 {
		f.t.Fatalf("links created: %d", on.LinksCreated)
	}
	return on.Program
}

func (f *fixture) code(programID string, contact uuid.UUID) string {
	f.t.Helper()
	l, err := f.env.App.Q.ReferralLinkOfContact(context.Background(), store.ReferralLinkOfContactParams{ProgramID: uuid.MustParse(programID), ContactID: contact})
	if err != nil {
		f.t.Fatalf("link of %s: %v", contact, err)
	}
	return l.Code
}

// signup is a signup through a referral link: a new contact, pending on the
// list until it confirms.
func (f *fixture) signup(code, email string, ip string) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	id := uuid.Must(uuid.NewV7())
	now := f.clock()
	f.exec(`INSERT INTO contacts (id, workspace_id, email, created_at) VALUES ($1, $2, $3, $4)`, id, f.ws.ID, email, now)
	f.exec(`INSERT INTO list_memberships (list_id, contact_id, status, created_at) VALUES ($1, $2, 'pending', $3)`, f.list, id, now)
	var addr *netip.Addr
	if ip != "" {
		a := netip.MustParseAddr(ip)
		addr = &a
	}
	err := db.Tx(ctx, f.env.App.Pool, func(tx pgx.Tx) error {
		return f.hook.Accepted(ctx, tx, &f.ws, forms.SignupEvent{ContactID: id, Email: email, Created: true, Lists: []uuid.UUID{f.list}, Pending: []uuid.UUID{f.list},
			Page: "https://news.grow.test/join/" + code + "?via=whatsapp", IP: addr, At: now})
	})
	if err != nil {
		f.t.Fatalf("signup %s: %v", email, err)
	}
	return id
}

// confirm confirms a signup.
func (f *fixture) confirm(contact uuid.UUID) {
	f.t.Helper()
	ctx := context.Background()
	now := f.clock()
	f.exec(`UPDATE list_memberships SET status = 'subscribed', subscribed_at = $3 WHERE list_id = $1 AND contact_id = $2`, f.list, contact, now)
	err := db.Tx(ctx, f.env.App.Pool, func(tx pgx.Tx) error {
		return f.hook.Confirmed(ctx, tx, &f.ws, forms.ConfirmEvent{ContactID: contact, Lists: []uuid.UUID{f.list}, At: now})
	})
	if err != nil {
		f.t.Fatalf("confirm: %v", err)
	}
}

func (f *fixture) status(contact uuid.UUID) (status, reason string) {
	f.t.Helper()
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT status, reason FROM referrals WHERE contact_id = $1`, contact).Scan(&status, &reason); err != nil {
		f.t.Fatalf("referral of %s: %v", contact, err)
	}
	return status, reason
}

func (f *fixture) want(contact uuid.UUID, status, reason string) {
	f.t.Helper()
	if s, r := f.status(contact); s != status || r != reason {
		f.t.Fatalf("referral: %s/%s, want %s/%s", s, r, status, reason)
	}
}

func manual(count int, name string) referrals.ReferralMilestoneInput {
	return referrals.ReferralMilestoneInput{Count: count, RewardName: name, Type: referrals.RewardManual, EmailOn: true}
}

// The counting rules of brief §3.2: self-referrals, the referrer's own
// address, disposable domains, readers already on the list, the hold
// period, leaving during the hold and never confirming.
func TestCountingRules(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	p := f.program(manual(1, "Sticker pack"), manual(3, "Mug"))
	code := f.code(p.ID, f.sam)
	if p.Counting.HoldDays != 7 || !p.Counting.SameIP || !p.Counting.Disposable || !p.Counting.Left || p.Leaderboard.On {
		t.Fatalf("defaults: %+v, leaderboard %+v", p.Counting, p.Leaderboard)
	}

	// Self-referral: the referrer's own mailbox, with a +tag.
	self := f.signup(code, "sam.rivera+friend@outlook.com", "")
	f.want(self, referrals.StatusNotCounted, referrals.ReasonSelf)

	// The referrer's own address: Sam opened their hub from 203.0.113.7.
	link, _ := f.env.App.Q.ReferralLinkOfContact(ctx, store.ReferralLinkOfContactParams{ProgramID: uuid.MustParse(p.ID), ContactID: f.sam})
	ip := netip.MustParseAddr("203.0.113.7")
	f.s.SeenReferrer(ctx, link.ID, &ip)
	sameIP := f.signup(code, "jordan@fabrikam.example", "203.0.113.7")
	f.want(sameIP, referrals.StatusAwaiting, "")
	f.confirm(sameIP)
	f.want(sameIP, referrals.StatusNotCounted, referrals.ReasonSameIP)

	// A disposable domain.
	throwaway := f.signup(code, "x7@mailinator.com", "198.51.100.20")
	f.confirm(throwaway)
	f.want(throwaway, referrals.StatusNotCounted, referrals.ReasonDisposable)

	// Already on the list: a reader of Product Updates who was on Weekly
	// Digest before isn't new.
	former := f.subscriber("lee@contoso.example", "Lee", "", f.other)
	f.exec(`INSERT INTO consent_events (id, contact_id, list_id, action, occurred_at) VALUES ($1, $2, $3, 'unsubscribed', $4)`,
		uuid.Must(uuid.NewV7()), former, f.list, f.clock().Add(-24*time.Hour))
	f.exec(`INSERT INTO list_memberships (list_id, contact_id, status, created_at) VALUES ($1, $2, 'pending', $3)`, f.list, former, f.clock())
	err := db.Tx(ctx, f.env.App.Pool, func(tx pgx.Tx) error {
		return f.hook.Accepted(ctx, tx, &f.ws, forms.SignupEvent{ContactID: former, Email: "lee@contoso.example", Lists: []uuid.UUID{f.list},
			Page: "https://news.grow.test/subscribe?ref=" + strings.ToLower(code), At: f.clock()})
	})
	if err != nil {
		t.Fatal(err)
	}
	f.want(former, referrals.StatusNotCounted, referrals.ReasonAlready)

	// The hold: a friend who confirms counts 7 days later.
	friend := f.signup(code, "ava@northwind.example", "198.51.100.21")
	f.advance(time.Hour)
	f.confirm(friend)
	f.want(friend, referrals.StatusHold, "")
	var until time.Time
	_ = f.env.App.Pool.QueryRow(ctx, `SELECT hold_until FROM referrals WHERE contact_id = $1`, friend).Scan(&until)
	if want := f.clock().Add(7 * 24 * time.Hour); !until.Equal(want) {
		t.Fatalf("hold until %s, want %s", until, want)
	}
	// Leaving during the hold: another friend unsubscribes on day 3.
	leaver := f.signup(code, "noor@tailspin.example", "198.51.100.22")
	f.confirm(leaver)
	f.advance(3 * 24 * time.Hour)
	f.exec(`UPDATE list_memberships SET status = 'unsubscribed', unsubscribed_at = $3 WHERE list_id = $1 AND contact_id = $2`, f.list, leaver, f.clock())
	// Never confirmed: a signup that stays unconfirmed for 7 days.
	silent := f.signup(code, "kim@woodgrove.example", "198.51.100.23")
	if n, err := f.s.ProcessHolds(ctx); err != nil || n != 0 {
		t.Fatalf("holds on day 3: %d %v", n, err)
	}
	f.advance(4*24*time.Hour + time.Hour)
	if _, err := f.s.ProcessHolds(ctx); err != nil {
		t.Fatal(err)
	}
	f.want(friend, referrals.StatusCounted, "")
	f.want(leaver, referrals.StatusNotCounted, referrals.ReasonLeft)
	f.want(silent, referrals.StatusAwaiting, "")
	f.advance(3 * 24 * time.Hour)
	if _, err := f.s.ProcessNeverConfirmed(ctx); err != nil {
		t.Fatal(err)
	}
	f.want(silent, referrals.StatusNotCounted, referrals.ReasonNeverConfirmed)

	// One counted referral reached the first milestone: a manual reward to
	// fulfill, the milestone email queued on the Automations stream, the
	// log and the "Reached a milestone" trigger.
	var status, name string
	var msg *uuid.UUID
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT status, reward_name, email_message_id FROM referral_rewards WHERE contact_id = $1`, f.sam).Scan(&status, &name, &msg); err != nil {
		t.Fatal(err)
	}
	if status != referrals.RewardToFulfill || name != "Sticker pack" || msg == nil {
		t.Fatalf("reward: %s %s %v", status, name, msg)
	}
	var stream, subject string
	var automation *uuid.UUID
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT stream, automation_id FROM messages WHERE id = $1`, *msg).Scan(&stream, &automation); err != nil {
		t.Fatal(err)
	}
	if stream != "automation" || automation != nil {
		t.Fatalf("milestone email: %s %v", stream, automation)
	}
	_ = subject
	var events int
	_ = f.env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM automation_events WHERE kind = 'milestone_reached' AND contact_id = $1`, f.sam).Scan(&events)
	if events != 1 {
		t.Fatalf("milestone trigger events: %d", events)
	}
	var kinds []string
	rows, _ := f.env.App.Pool.Query(ctx, `SELECT kind FROM referral_events WHERE program_id = $1 ORDER BY occurred_at, id`, uuid.MustParse(p.ID))
	kinds, _ = pgx.CollectRows(rows, pgx.RowTo[string])
	joined := strings.Join(kinds, ",")
	for _, k := range []string{"signed_up", "confirmed", "not_counted", "counted", "milestone_reached", "reward_sent"} {
		if !strings.Contains(joined, k) {
			t.Fatalf("log has no %s: %s", k, joined)
		}
	}
	// Nothing about counting went to the audit log (it has its own log).
	var audited int
	_ = f.env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action LIKE 'referral.%' AND action NOT IN
		('referral.program_created', 'referral.milestone_created', 'referral.program_turned_on')`, f.ws.ID).Scan(&audited)
	if audited != 0 {
		t.Fatalf("counting wrote %d audit entries", audited)
	}
	// The hub shows the numbers, never names or addresses.
	hub, err := f.s.Hub(ctx, &f.ws, f.sam, f.s.HubToken(f.ws.ID, f.sam, f.clock()), nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(hub)
	if strings.Contains(string(raw), "ava@") || strings.Contains(string(raw), "northwind") || hub.Track.Counted != 1 {
		t.Fatalf("hub: counted %d, %s", hub.Track.Counted, raw)
	}
}

// Suspicion rules hold referrals for a person, who counts them or not,
// with 10 seconds to undo.
func TestReviewQueue(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	p := f.program(manual(5, "Sticker pack"))
	pid := uuid.MustParse(p.ID)
	code := f.code(p.ID, f.sam)
	// Six confirmations within an hour are a burst (threshold 5).
	var friends []uuid.UUID
	for i, e := range []string{"a1@contoso.example", "b2@fabrikam.example", "c3@northwind.example", "d4@tailspin.example", "e5@woodgrove.example", "f6@litware.example"} {
		c := f.signup(code, e, "198.51.100."+string(rune('1'+i)))
		f.confirm(c)
		friends = append(friends, c)
		f.advance(5 * time.Minute)
	}
	for _, c := range friends {
		f.want(c, referrals.StatusReview, "")
	}
	q, err := f.s.Review(ctx, &f.ws, pid, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Groups) != 1 || len(q.Groups[0].Items) != 6 || q.Groups[0].Suggestion == "" {
		t.Fatalf("review groups: %+v", q.Groups)
	}
	ids := []string{}
	for _, it := range q.Groups[0].Items {
		ids = append(ids, it.ID)
	}
	pv, err := f.s.PreviewDecision(ctx, &f.ws, pid, referrals.ReferralDecisionInput{IDs: ids, Decision: "count"})
	if err != nil || pv.Count != 6 {
		t.Fatalf("preview: %+v %v", pv, err)
	}
	d, err := f.s.Decide(ctx, &f.ws, pid, referrals.ReferralDecisionInput{IDs: ids, Decision: "dont_pause", Note: "Same family"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range friends {
		f.want(c, referrals.StatusNotCounted, referrals.ReasonReviewer)
	}
	link, _ := f.env.App.Q.ReferralLinkOfContact(ctx, store.ReferralLinkOfContactParams{ProgramID: pid, ContactID: f.sam})
	if link.Status != "paused" {
		t.Fatalf("link: %s", link.Status)
	}
	// Undo within 10 seconds puts everything back.
	f.advance(5 * time.Second)
	if err := f.s.UndoDecision(ctx, &f.ws, pid, uuid.MustParse(d.ID)); err != nil {
		t.Fatal(err)
	}
	for _, c := range friends {
		f.want(c, referrals.StatusReview, "")
	}
	link, _ = f.env.App.Q.ReferralLinkOfContact(ctx, store.ReferralLinkOfContactParams{ProgramID: pid, ContactID: f.sam})
	if link.Status != "active" {
		t.Fatalf("link after undo: %s", link.Status)
	}
	// Count them: they go into the hold (at least 10 s, for the undo) and
	// count when it ends.
	d, err = f.s.Decide(ctx, &f.ws, pid, referrals.ReferralDecisionInput{IDs: ids, Decision: "count"})
	if err != nil {
		t.Fatal(err)
	}
	f.advance(11 * time.Second)
	if err := f.s.UndoDecision(ctx, &f.ws, pid, uuid.MustParse(d.ID)); err == nil {
		t.Fatal("undo after 10 seconds")
	}
	f.advance(8 * 24 * time.Hour)
	if _, err := f.s.ProcessHolds(ctx); err != nil {
		t.Fatal(err)
	}
	for _, c := range friends {
		f.want(c, referrals.StatusCounted, "")
	}
	var audited int
	_ = f.env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action IN ('referral.review_decided', 'referral.review_undone')`, f.ws.ID).Scan(&audited)
	if audited != 3 {
		t.Fatalf("decision audit entries: %d", audited)
	}
}

// Code rewards wait when the codes run out and go out when more arrive;
// webhook rewards retry, fail after 6 hours and can be retried.
func TestRewardFulfilment(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	var mu sync.Mutex
	answer, calls := http.StatusInternalServerError, 0
	var lastSig, lastTS string
	var lastBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		lastSig, lastTS = r.Header.Get(webhooks.HeaderSignature), r.Header.Get(webhooks.HeaderTimestamp)
		lastBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(answer)
		if answer == http.StatusOK {
			_, _ = w.Write([]byte(`{"code":"ACME-VIP-7731"}`))
		}
	}))
	defer srv.Close()
	secret := "whsec_test"
	p := f.program(
		referrals.ReferralMilestoneInput{Count: 1, RewardName: "Discount code", Type: referrals.RewardCodes, EmailOn: true},
		referrals.ReferralMilestoneInput{Count: 2, RewardName: "Community access", Type: referrals.RewardWebhook, WebhookURL: srv.URL, WebhookSecret: &secret, EmailOn: true},
	)
	pid := uuid.MustParse(p.ID)
	list, err := f.s.Milestones(ctx, &f.ws, pid)
	if err != nil {
		t.Fatal(err)
	}
	codesID, hookID := list.Items[0].ID, list.Items[1].ID
	if list.Items[1].WebhookSecretSet != true {
		t.Fatalf("secret set: %+v", list.Items[1])
	}
	raw, _ := json.Marshal(list)
	if strings.Contains(string(raw), secret) {
		t.Fatal("the webhook secret came back")
	}
	up, err := f.s.UploadCodes(ctx, &f.ws, uuid.MustParse(codesID), referrals.ReferralCodesInput{Text: "SAVE-0001\nSAVE-0001\n"})
	if err != nil || up.Added != 1 || up.Duplicates != 1 {
		t.Fatalf("upload: %+v %v", up, err)
	}
	// Two referrers reach 1: the first takes the only code, the second waits.
	ava := f.subscriber("ava@northwind.example", "Ava", "", f.list)
	if _, err := f.s.EnsureLinks(ctx); err != nil {
		t.Fatal(err)
	}
	countOne := func(referrer uuid.UUID, email string) {
		c := f.signup(f.code(p.ID, referrer), email, "")
		f.confirm(c)
	}
	countOne(f.sam, "one@contoso.example")
	countOne(ava, "two@fabrikam.example")
	f.advance(7*24*time.Hour + time.Minute)
	if _, err := f.s.ProcessHolds(ctx); err != nil {
		t.Fatal(err)
	}
	rewardOf := func(contact uuid.UUID, milestone string) (status string, attempts int) {
		t.Helper()
		_ = f.env.App.Pool.QueryRow(ctx, `SELECT status, attempts FROM referral_rewards WHERE contact_id = $1 AND milestone_id = $2`, contact, uuid.MustParse(milestone)).Scan(&status, &attempts)
		return
	}
	if s, _ := rewardOf(f.sam, codesID); s != referrals.RewardSent {
		t.Fatalf("Sam's code: %s", s)
	}
	if s, _ := rewardOf(ava, codesID); s != referrals.RewardWaitingCodes {
		t.Fatalf("Ava's code: %s", s)
	}
	if _, err := f.s.UploadCodes(ctx, &f.ws, uuid.MustParse(codesID), referrals.ReferralCodesInput{Text: "code\nSAVE-0002\nSAVE-0003\n"}); err != nil {
		t.Fatal(err)
	}
	if s, _ := rewardOf(ava, codesID); s != referrals.RewardSent {
		t.Fatalf("Ava's code after the upload: %s", s)
	}
	// The codes upload is logged with counts, never the codes.
	var meta string
	_ = f.env.App.Pool.QueryRow(ctx, `SELECT coalesce(after::text, '') || coalesce(metadata::text, '') FROM audit_events WHERE action = 'referral.codes_uploaded' ORDER BY occurred_at DESC LIMIT 1`).Scan(&meta)
	if strings.Contains(meta, "SAVE-") {
		t.Fatalf("codes in the audit log: %s", meta)
	}

	// Sam reaches 2: the webhook fails, is retried for 6 hours, then fails.
	countOne(f.sam, "three@northwind.example")
	f.advance(7*24*time.Hour + time.Minute)
	if _, err := f.s.ProcessHolds(ctx); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if _, err := f.s.DeliverDue(ctx); err != nil {
			t.Fatal(err)
		}
		f.advance(15 * time.Minute)
	}
	st, attempts := rewardOf(f.sam, hookID)
	if st != referrals.RewardFailed || attempts < 3 {
		t.Fatalf("webhook reward: %s after %d attempts", st, attempts)
	}
	// Signed like outgoing webhooks, with the milestone's secret.
	ts, _ := strconv.ParseInt(lastTS, 10, 64)
	if lastSig == "" || lastSig != webhooks.Sign(secret, ts, lastBody) || !strings.Contains(string(lastBody), `"type":"referral.milestone_reached"`) {
		t.Fatalf("webhook signature %q for %s", lastSig, lastBody)
	}
	var notified int
	_ = f.env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE workspace_id = $1 AND kind = 'reward_webhook_failing'`, f.ws.ID).Scan(&notified)
	if notified == 0 {
		t.Fatal("no notification for the failing webhook")
	}
	// Retry once it answers again: the code from its answer is the reward.
	mu.Lock()
	answer = http.StatusOK
	mu.Unlock()
	var rid uuid.UUID
	_ = f.env.App.Pool.QueryRow(ctx, `SELECT id FROM referral_rewards WHERE contact_id = $1 AND milestone_id = $2`, f.sam, uuid.MustParse(hookID)).Scan(&rid)
	if err := f.s.RetryReward(ctx, &f.ws, rid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.DeliverDue(ctx); err != nil {
		t.Fatal(err)
	}
	if st, _ := rewardOf(f.sam, hookID); st != referrals.RewardSent {
		t.Fatalf("after the retry: %s", st)
	}
	hub, err := f.s.Hub(ctx, &f.ws, f.sam, f.s.HubToken(f.ws.ID, f.sam, f.clock()), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range hub.Rewards {
		if r.Code == "ACME-VIP-7731" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the hub doesn't show the webhook's code: %+v", hub.Rewards)
	}
	// The test call reaches the endpoint without changing any reward.
	res, err := f.s.TestWebhook(ctx, &f.ws, uuid.MustParse(hookID))
	if err != nil || !res.OK || res.Code != "ACME-VIP-7731" {
		t.Fatalf("test call: %+v %v", res, err)
	}
}
