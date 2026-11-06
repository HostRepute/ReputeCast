package digests_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// scopedEditor adds an Editor limited by scope to the demo workspace and
// signs them in.
func scopedEditor(t *testing.T, env *apptest.Env, email string, scope authz.Scope) (*apptest.Client, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	var wsID, role uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, r.id FROM workspaces w JOIN roles r ON r.workspace_id = w.id AND r.key = 'editor'
		WHERE w.slug = 'acme-newsletter'`).Scan(&wsID, &role); err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: email, Name: "Contractor " + email})
	raw, _ := json.Marshal(scope)
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: role, Scope: raw, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo(email)
	return c, u.ID
}

// A member limited to some lists or sender domains sets a digest up, turns
// it on and releases its issues only within them, whoever set up what they
// keep; an unscoped member does all of it, and a digest turned on within a
// member's scope sends its issues on its own, as any other.
func TestScopedMemberDigests(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	starter := &fakeStarter{}
	env.App.Sending = starter
	n := counter{t, env}
	ctx := context.Background()
	feed := newFeedServer(t, basePosts(time.Now())...)
	svc := env.App.Digests()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	product, beta := n.listID("Product Updates"), n.listID("Beta Program")
	// Kim reaches Beta Program only; Lee sends from acme.io only.
	kim, kimID := scopedEditor(t, env, "kim@contractor.dev", authz.Scope{Lists: []uuid.UUID{uuid.MustParse(beta)}})
	lee, _ := scopedEditor(t, env, "lee@contractor.dev", authz.Scope{SenderDomains: []string{"acme.io"}})
	senderID := func(email string) string {
		t.Helper()
		var id uuid.UUID
		if err := env.App.Pool.QueryRow(ctx, `SELECT s.id FROM senders s JOIN workspaces w ON w.id = s.workspace_id WHERE w.slug = 'acme-newsletter' AND s.email = $1`,
			email).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id.String()
	}
	var segment uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT s.id FROM segments s JOIN workspaces w ON w.id = s.workspace_id WHERE w.slug = 'acme-newsletter' LIMIT 1`).Scan(&segment); err != nil {
		t.Fatal(err)
	}
	const (
		listsOnly  = "Your access is limited to some lists: send only to those lists, not to other lists or a segment."
		newsDomain = "Your access is limited to some sender domains, and this campaign sends from news.acme.io."
	)
	refused := func(what, want string, r *apptest.Response) {
		t.Helper()
		b, _ := json.Marshal(r.MustStatus(t, 403).Problem(t))
		if !strings.Contains(string(b), want) {
			t.Fatalf("%s: %s", what, b)
		}
	}
	create := func(c *apptest.Client, name string) detail {
		t.Helper()
		return decodeDetail(t, c.Do("POST", ws+"/digests", map[string]any{"name": name, "source": "url", "feedUrl": feed.srv.URL + "/feed/"}).MustStatus(t, 201))
	}
	sender := func(id string) string {
		t.Helper()
		var d struct{ Sender *struct{ Email string } }
		linh.Do("GET", ws+"/digests/"+id, nil).MustStatus(t, 200).JSON(t, &d)
		if d.Sender == nil {
			return ""
		}
		return d.Sender.Email
	}

	// Linh's digest: Product Updates, from the workspace's default sender
	// on news.acme.io. Neither Kim nor Lee may save it as it is or turn it
	// on: its issues would reach other lists, or go out from news.acme.io.
	mine := create(linh, "Linh's roundup")
	if got := sender(mine.ID); got != "digest@news.acme.io" {
		t.Fatalf("Linh's digest starts from %q", got)
	}
	setup := settings("Linh's roundup", feed.srv.URL+"/feed/", product, 2)
	linh.Do("PUT", ws+"/digests/"+mine.ID, setup).MustStatus(t, 200)
	updates := n.audit("digest.updated", mine.ID)
	refused("Kim saves Linh's digest", listsOnly, kim.Do("PUT", ws+"/digests/"+mine.ID, setup))
	refused("Lee saves Linh's digest", newsDomain, lee.Do("PUT", ws+"/digests/"+mine.ID, setup))
	toSegment := settings("Linh's roundup", feed.srv.URL+"/feed/", beta, 2)
	toSegment["listIds"], toSegment["segmentIds"] = []string{}, []string{segment.String()}
	refused("Kim picks a segment", listsOnly, kim.Do("PUT", ws+"/digests/"+mine.ID, toSegment))
	if got := n.audit("digest.updated", mine.ID); got != updates {
		t.Fatalf("a refused save is audited: %d entries, were %d", got, updates)
	}
	var audience string
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.audience::text FROM campaigns c WHERE c.id = $1`, uuid.MustParse(mine.TemplateID)).Scan(&audience); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(audience, product) || strings.Contains(audience, segment.String()) {
		t.Fatalf("a refused save changed the audience: %s", audience)
	}
	refused("Kim turns Linh's digest on", listsOnly, kim.Do("POST", ws+"/digests/"+mine.ID+"/turn-on", map[string]any{}))
	refused("Lee turns Linh's digest on", newsDomain, lee.Do("POST", ws+"/digests/"+mine.ID+"/turn-on", map[string]any{}))
	mine = decodeDetail(t, linh.Do("POST", ws+"/digests/"+mine.ID+"/turn-on", map[string]any{}).MustStatus(t, 200))
	if mine.State != "on" || mine.Owner == nil || mine.Owner.Name != "Linh Nguyen" {
		t.Fatalf("Linh turns it on: %+v", mine)
	}

	// Its issue is held (the template lost its footer), then fixed: only
	// Linh may release it.
	var original string
	if err := env.App.Pool.QueryRow(ctx, `SELECT mjml FROM campaigns WHERE id = $1`, uuid.MustParse(mine.TemplateID)).Scan(&original); err != nil {
		t.Fatal(err)
	}
	n.exec(`UPDATE campaigns SET mjml = regexp_replace(mjml, '<!-- rc:footer -->.*?</mj-section>', '') WHERE id = $1`, uuid.MustParse(mine.TemplateID))
	if err := svc.Prepare(ctx, uuid.MustParse(mine.Next.ID)); err != nil {
		t.Fatal(err)
	}
	mine = getDigest(t, linh, mine.ID)
	if mine.Next == nil || mine.Next.Status != "held" {
		t.Fatalf("held: %+v", mine.Next)
	}
	n.exec(`UPDATE campaigns SET mjml = $2 WHERE id = $1`, uuid.MustParse(mine.TemplateID), original)
	linh.Do("POST", ws+"/digests/"+mine.ID+"/issues/"+mine.Next.ID+"/rebuild", nil).MustStatus(t, 200)
	release := ws + "/digests/" + mine.ID + "/issues/" + mine.Next.ID + "/release"
	refused("Kim releases the issue", listsOnly, kim.Do("POST", release, nil))
	refused("Lee releases the issue", newsDomain, lee.Do("POST", release, nil))
	if n.audit("digest.issue_released", mine.ID) != 0 {
		t.Fatal("a refused release is audited")
	}
	linh.Do("POST", release, nil).MustStatus(t, 200)
	if n.audit("digest.issue_released", mine.ID) != 1 {
		t.Fatal("Linh's release isn't audited")
	}

	// Lee's own digest starts from a sender of hers, and she can't move it
	// to news.acme.io.
	lees := create(lee, "Lee's roundup")
	if got := sender(lees.ID); got != "marcus@acme.io" {
		t.Fatalf("Lee's digest starts from %q", got)
	}
	leeSetup := settings("Lee's roundup", feed.srv.URL+"/feed/", product, 2)
	lee.Do("PUT", ws+"/digests/"+lees.ID, leeSetup).MustStatus(t, 200)
	leeSetup["senderId"] = senderID("digest@news.acme.io")
	refused("Lee picks news.acme.io", newsDomain, lee.Do("PUT", ws+"/digests/"+lees.ID, leeSetup))
	if got := sender(lees.ID); got != "marcus@acme.io" {
		t.Fatalf("a refused save changed the sender to %q", got)
	}

	// Kim's digest goes to Beta Program: she sets it up and turns it on, and
	// the schedule prepares and starts its issue on her authority.
	kims := create(kim, "Kim's roundup")
	kimSetup := settings("Kim's roundup", feed.srv.URL+"/feed/", product, 2)
	refused("Kim picks Product Updates", listsOnly, kim.Do("PUT", ws+"/digests/"+kims.ID, kimSetup))
	kimSetup["listIds"] = []string{beta}
	kim.Do("PUT", ws+"/digests/"+kims.ID, kimSetup).MustStatus(t, 200)
	kims = decodeDetail(t, kim.Do("POST", ws+"/digests/"+kims.ID+"/turn-on", map[string]any{}).MustStatus(t, 200))
	if kims.State != "on" || kims.Next == nil {
		t.Fatalf("Kim turns hers on: %+v", kims)
	}
	issueID := uuid.MustParse(kims.Next.ID)
	if err := svc.Prepare(ctx, issueID); err != nil {
		t.Fatal(err)
	}
	kims = getDigest(t, linh, kims.ID)
	if kims.Next == nil || kims.Next.CampaignID == "" || kims.Next.Status == "held" {
		if kims.Next != nil && kims.Next.CampaignID != "" {
			t.Logf("blocks: %s", n.blocks(kims.Next.CampaignID))
		}
		t.Fatalf("Kim's issue, prepared: %+v", kims.Next)
	}
	if a := kims.Next.Approval; a != nil && a.Status == "pending" {
		linh.Do("POST", ws+"/approvals/"+a.ID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	}
	n.exec(`UPDATE digest_issues SET send_at = now() - interval '1 minute', deadline = now() + interval '1 day' WHERE id = $1`, issueID)
	if err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(starter.started) != 1 || starter.started[0].String() != kims.Next.CampaignID || starter.by[0] != kimID {
		t.Fatalf("Kim's issue starts on her authority: %v by %v", starter.started, starter.by)
	}
}
