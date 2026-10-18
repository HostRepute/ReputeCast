package earn_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/visibility"
)

// paidSend is the demo with a list of the test's own: one reader of each
// kind of paid access, an ended member, a free subscriber and a test paid
// member.
type paidSend struct {
	t        *testing.T
	env      *apptest.Env
	e        *engine.Engine
	ctx      context.Context
	ws, linh uuid.UUID
	list     uuid.UUID
	now      time.Time
	readers  map[string]uuid.UUID
	insider  uuid.UUID
	founding uuid.UUID
}

func (p *paidSend) one(sql string, args ...any) uuid.UUID {
	p.t.Helper()
	var id uuid.UUID
	if err := p.env.App.Pool.QueryRow(p.ctx, sql, args...).Scan(&id); err != nil {
		p.t.Fatalf("%s: %v", sql, err)
	}
	return id
}

func (p *paidSend) exec(sql string, args ...any) {
	p.t.Helper()
	if _, err := p.env.App.Pool.Exec(p.ctx, sql, args...); err != nil {
		p.t.Fatalf("%s: %v", sql, err)
	}
}

func newPaidSend(t *testing.T) *paidSend {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	p := &paidSend{t: t, env: env, e: env.App.SendingEngine(), ctx: context.Background(), readers: map[string]uuid.UUID{}}
	p.ws = p.one(`SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`)
	p.linh = p.one(`SELECT id FROM users WHERE email = 'linh@acme.io'`)
	p.insider = p.one(`SELECT id FROM earn_tiers WHERE workspace_id = $1 AND name = 'Weekly Digest Insider'`, p.ws)
	p.founding = p.one(`SELECT id FROM earn_tiers WHERE workspace_id = $1 AND name = 'Founding member'`, p.ws)
	access := `s.ended_at IS NULL AND (s.status IN ('trialing','active','complimentary','gift') OR (s.status = 'past_due' AND NOT s.access_paused))`
	pick := func(key, where string) {
		p.readers[key] = p.one(`SELECT s.contact_id FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id AND c.status = 'active'
			WHERE s.workspace_id = $1 AND `+where+` AND NOT EXISTS (SELECT 1 FROM suppressions x WHERE x.workspace_id = c.workspace_id AND x.email = c.email)
			AND (SELECT count(*) FROM paid_subscriptions o WHERE o.contact_id = s.contact_id) = 1 ORDER BY c.email LIMIT 1`, p.ws)
	}
	pick("insider", `s.mode = 'live' AND s.tier_id = '`+p.insider.String()+`' AND s.status = 'active' AND s.cancel_at IS NULL AND `+access)
	pick("founding", `s.mode = 'live' AND s.tier_id = '`+p.founding.String()+`' AND s.status = 'active' AND `+access)
	pick("pastDue", `s.mode = 'live' AND s.status = 'past_due' AND NOT s.access_paused AND s.ended_at IS NULL`)
	pick("trialing", `s.mode = 'live' AND s.status = 'trialing' AND s.ended_at IS NULL`)
	pick("comp", `s.mode = 'live' AND s.kind = 'complimentary' AND `+access)
	pick("ended", `s.mode = 'live' AND s.ended_at IS NOT NULL`)
	pick("test", `s.mode = 'test' AND `+access)
	p.readers["free"] = p.one(`SELECT c.id FROM contacts c WHERE c.workspace_id = $1 AND c.status = 'active'
		AND NOT EXISTS (SELECT 1 FROM paid_subscriptions s WHERE s.contact_id = c.id)
		AND NOT EXISTS (SELECT 1 FROM suppressions x WHERE x.workspace_id = c.workspace_id AND x.email = c.email)
		AND c.email NOT LIKE 'info@%' AND NOT c.role_address ORDER BY c.email LIMIT 1`, p.ws)
	p.list = uuid.Must(uuid.NewV7())
	p.exec(`INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, 'Paid send test')`, p.list, p.ws)
	for _, id := range p.readers {
		p.exec(`INSERT INTO list_memberships (list_id, contact_id, status, subscribed_at) VALUES ($1, $2, 'subscribed', now() - interval '30 days')`, p.list, id)
		// Nothing about the readers but their paid access decides.
		p.exec(`UPDATE contacts SET tier = 'active', broadcasts_stopped_at = NULL WHERE id = $1`, id)
		p.exec(`DELETE FROM contact_preferences WHERE contact_id = $1`, id)
	}
	p.now = time.Now().Truncate(time.Second)
	now := p.now
	p.e.Now = func() time.Time { return now }
	return p
}

// campaign copies Weekly Digest #142 (its sender and settings) with the
// given content and audience.
func (p *paidSend) campaign(name, src, audience string) uuid.UUID {
	p.t.Helper()
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		p.t.Fatal(res.Errors)
	}
	id := uuid.Must(uuid.NewV7())
	p.exec(`INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, reply_to, audience, editor_mode, mjml, html, plain_text,
		schedule, thresholds, tracking, created_by, updated_by, content_hash)
		SELECT $1, workspace_id, $2, 'draft', subject, preheader, sender_id, reply_to, $3, editor_mode, $4, $5, '',
		'{"mode":"now"}', thresholds, tracking, created_by, updated_by, content_hash
		FROM campaigns WHERE workspace_id = $6 AND name = 'Weekly Digest #142'`, id, name, audience, src, res.HTML, p.ws)
	return id
}

// send starts a campaign and builds its recipients; it returns each
// recipient's hidden blocks.
func (p *paidSend) send(id uuid.UUID) map[uuid.UUID]string {
	p.t.Helper()
	ctx := sending.WithRecheck(p.ctx, id, sending.Recheck{Verdict: "ready"})
	if err := p.e.StartWith(ctx, id, engine.StartOptions{By: p.linh, Now: true}); err != nil {
		p.t.Fatalf("start: %v", err)
	}
	if err := p.e.DrivePrepare(p.ctx, id, p.now); err != nil {
		p.t.Fatalf("prepare: %v", err)
	}
	out := map[uuid.UUID]string{}
	rows, err := p.env.App.Pool.Query(p.ctx, `SELECT contact_id, hidden_blocks FROM messages WHERE campaign_id = $1`, id)
	if err != nil {
		p.t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var c uuid.UUID
		var hidden string
		if err := rows.Scan(&c, &hidden); err != nil {
			p.t.Fatal(err)
		}
		out[c] = hidden
	}
	return out
}

func (p *paidSend) names(got map[uuid.UUID]string) []string {
	var out []string
	for k, id := range p.readers {
		if _, ok := got[id]; ok {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

func rule(id string, op, value, label string) string {
	r := visibility.VisibilityRule{ID: id, Show: visibility.ShowOnly, Conditions: []visibility.VisibilityCondition{{Kind: earn.KindPaid, Op: op, Value: value, Label: label}}}
	return strings.ReplaceAll(r.Encode(), `"`, "&quot;")
}

// Paid-only content is decided per reader when a send builds its
// recipients: paid access (trials, complimentary access and past due while
// Stripe retries included) sees the paid sections, a tier sees what its
// rank includes, everyone else (ended members, free subscribers, test paid
// members) sees the free version with the Upgrade block. Paid-only
// campaigns go to the paid members on their lists, or to the free
// subscribers.
func TestPaidContentAtSendTime(t *testing.T) {
	p := newPaidSend(t)
	src := `<mjml><mj-body>
<mj-section><mj-column><mj-text>This week at Acme.</mj-text></mj-column></mj-section>
<mj-section rc-name="Insider deep dive" rc-visible="` + rule("paid01", earn.OpPaidAny, "", "") + `"><mj-column><mj-text>The deep dive.</mj-text></mj-column></mj-section>
<mj-section rc-name="Founders' notes" rc-visible="` + rule("fnd001", earn.OpPaidTier, p.founding.String(), "Founding member") + `"><mj-column><mj-text>Notes for founders.</mj-text></mj-column></mj-section>
<mj-section rc-name="Upgrade" rc-visible="` + rule("free01", earn.OpPaidFree, "", "") + `"><mj-column><mj-button href="{{ upgrade_url }}">Start your free trial</mj-button></mj-column></mj-section>
<mj-section><mj-column><mj-text><a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`
	everyone := p.send(p.campaign("Paid content", src, fmt.Sprintf(`{"lists":[%q]}`, p.list)))
	want := map[string]string{
		"insider": "fnd001,free01", "pastDue": "fnd001,free01", "trialing": "fnd001,free01", "comp": "fnd001,free01",
		"founding": "free01",
		"ended":    "fnd001,paid01", "free": "fnd001,paid01", "test": "fnd001,paid01",
	}
	for k, id := range p.readers {
		got, ok := everyone[id]
		if !ok {
			t.Errorf("%s isn't a recipient", k)
			continue
		}
		if got != want[k] {
			t.Errorf("%s: hidden %q, want %q", k, got, want[k])
		}
	}

	// Paid members only: any paid tier, then Founding member and above.
	paid := p.send(p.campaign("Paid members only", src, fmt.Sprintf(`{"lists":[%q],"paid":"paid"}`, p.list)))
	if got := p.names(paid); !slices.Equal(got, []string{"comp", "founding", "insider", "pastDue", "trialing"}) {
		t.Errorf("paid members only: %v", got)
	}
	top := p.send(p.campaign("Founders only", src, fmt.Sprintf(`{"lists":[%q],"paid":"paid","paid_tier":%q}`, p.list, p.founding)))
	if got := p.names(top); !slices.Equal(got, []string{"founding"}) {
		t.Errorf("Founding member and above: %v", got)
	}
	// Free subscribers only: ended members and test paid members are free.
	free := p.send(p.campaign("Free subscribers only", src, fmt.Sprintf(`{"lists":[%q],"paid":"free"}`, p.list)))
	if got := p.names(free); !slices.Equal(got, []string{"ended", "free", "test"}) {
		t.Errorf("free subscribers only: %v", got)
	}

	// The composer counts the same people, and the list's label says who.
	c := p.env.Client()
	c.SignInDemo("linh@acme.io")
	var n struct{ Count int64 }
	c.Do("POST", "/workspaces/acme-newsletter/campaigns/audience-count", map[string]any{"lists": []string{p.list.String()}, "paid": "paid"}).MustStatus(t, 200).JSON(t, &n)
	if n.Count != int64(len(paid)) {
		t.Errorf("composer count %d, send %d", n.Count, len(paid))
	}
	var a earn.EarnAudience
	c.Do("GET", "/workspaces/acme-newsletter/earn/audience?list="+p.list.String(), nil).MustStatus(t, 200).JSON(t, &a)
	if a.Everyone != int64(len(everyone)) || a.Paid != int64(len(paid)) || a.Free != int64(len(free)) || a.PastDue != 1 || a.Trialing != 1 || a.Complimentary != 1 {
		t.Errorf("popover: %+v (sent to %d, %d paid, %d free)", a, len(everyone), len(paid), len(free))
	}
	var list struct {
		Items []struct {
			Name     string
			Audience string
			PaidOnly bool
		}
	}
	c.Do("GET", "/workspaces/acme-newsletter/campaigns?paidOnly=true&pageSize=100", nil).MustStatus(t, 200).JSON(t, &list)
	names := []string{}
	for _, it := range list.Items {
		names = append(names, it.Name)
		if !it.PaidOnly || !strings.HasSuffix(it.Audience, "Paid send test · Paid members only") {
			t.Errorf("paid-only row: %+v", it)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"Founders only", "Paid members only"}) {
		t.Errorf("the Paid only filter lists %v", names)
	}

	// Paid-only campaigns stay out of the public archive; an issue to
	// everyone is filed as the archive is read.
	digest := p.one(`SELECT id FROM lists WHERE workspace_id = $1 AND name = 'Weekly Digest' AND deleted_at IS NULL`, p.ws)
	paidIssue := p.campaign("Insider: the Q4 roadmap call, annotated", src, fmt.Sprintf(`{"lists":[%q],"paid":"paid"}`, digest))
	issue := p.campaign("Weekly Digest #199", src, fmt.Sprintf(`{"lists":[%q]}`, digest))
	p.exec(`UPDATE campaigns SET status = 'sent', started_at = now(), completed_at = now() WHERE id = ANY ($1)`, []uuid.UUID{paidIssue, issue})
	c.Do("GET", "/p/acme-newsletter/archive", nil).MustStatus(t, 200)
	filed := func(id uuid.UUID) int {
		var n int
		if err := p.env.App.Pool.QueryRow(p.ctx, `SELECT count(*) FROM archive_issues WHERE campaign_id = $1`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if filed(paidIssue) != 0 || filed(issue) != 1 {
		t.Errorf("archive: the paid-only issue filed %d times, the issue to everyone %d", filed(paidIssue), filed(issue))
	}
}
