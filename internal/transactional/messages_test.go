package transactional_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type logPage struct {
	Items []struct {
		ID       string
		Email    string
		Status   string
		Raw      bool
		Template *struct {
			Key     string
			Version *int
		}
		Provider  *struct{ ID, Name string }
		Rejection *struct {
			Reason      string
			Suppression *struct{ Reason, Source, Code, ID string }
			Allowed     []string
		}
		DeliveryMs    *int64
		TestDelivered bool
	}
	Counts struct {
		All, Queued, Sent, Delivered, Deferred, Bounced, Complained, Rejected, Failed, Test int
	}
	Next, Newest string
	Filters      struct {
		Templates, Providers, Keys []struct{ ID, Name string }
	}
	CanExport   bool
	ContentDays int
}

type messageDetail struct {
	ID       string
	Status   string
	Contact  *struct{ ID, Name string }
	Timeline []struct{ Kind, Provider, ProviderMessageID string }
	Request  struct {
		Method, Path, IdempotencyKey string
		Body                         map[string]any
	}
	Content struct {
		Available, CanView bool
		Helpers            []string
		Masked             []string
	}
	Resend struct {
		Available bool
		Reason    string
		Sensitive []string
	}
	Route         struct{ Rule, Primary, Fallback string }
	SensitiveSent []string
	Variables     int
	Webhooks      []struct{ Event string }
}

func TestMessageLog(t *testing.T) {
	s := newSendEnv(t)
	s.webhook()
	s.publish("welcome", "Welcome to Acme Cloud", "welcome")
	keyID, secret := s.key("Acme Cloud · production", map[string]any{"preset": "send_only"})
	_, test := s.key("Acme Cloud · staging", map[string]any{"preset": "send_only", "mode": "test", "testAddresses": []string{"qa+*@acme.io"}})
	s.exec(`INSERT INTO suppressions (id, workspace_id, email, reason, source, original_code, detail) VALUES ($1, $2, 'jordan@example.org', 'hard_bounce', 'webhook', '550 5.1.1', 'mailbox does not exist')`,
		uuid.Must(uuid.NewV7()), s.ws)

	var reset, welcome sendResult
	b := resetBody("sam.rivera@outlook.com")
	b["metadata"] = map[string]string{"user_id": "u_48213"}
	s.call(secret, "/messages", "pwreset-7f3a91", b).MustStatus(t, 202).JSON(t, &reset)
	s.clock.add(time.Second)
	s.call(secret, "/messages", "", resetBody("jordan@example.org")).MustStatus(t, 422)
	s.clock.add(time.Second)
	s.call(test, "/messages", "", resetBody("lena@example.com")).MustStatus(t, 202)
	s.clock.add(time.Second)
	wb := map[string]any{"template": "welcome", "to": map[string]any{"email": "priya.shah@example.net", "name": "Priya"},
		"data": map[string]any{"first_name": "Priya", "dashboard_url": "https://app.acme.io/"}, "tags": []string{"onboarding"}}
	s.call(secret, "/messages", "", wb).MustStatus(t, 202).JSON(t, &welcome)
	s.step()

	var page logPage
	s.owner.Do("GET", base+"/messages", nil).MustStatus(t, 200).JSON(t, &page)
	if len(page.Items) != 4 || page.Counts.All != 4 || page.Counts.Delivered != 2 || page.Counts.Rejected != 1 || page.Counts.Test != 1 || !page.CanExport || page.ContentDays != 30 {
		t.Fatalf("log: %d items, counts %+v", len(page.Items), page.Counts)
	}
	if page.Items[0].ID != welcome.ID || page.Items[3].ID != reset.ID || page.Items[3].DeliveryMs == nil || page.Items[3].Provider.Name != "Postmark" {
		t.Fatalf("order: %+v", page.Items)
	}
	rej := page.Items[2]
	if rej.Status != "rejected" || rej.Rejection == nil || rej.Rejection.Reason != "suppressed" || rej.Rejection.Suppression.Reason != "hard_bounce" ||
		rej.Rejection.Suppression.Code != "550 5.1.1" {
		t.Fatalf("rejected row: %+v", rej)
	}
	tst := page.Items[1]
	if tst.Status != "test" || tst.TestDelivered || tst.Rejection == nil || tst.Rejection.Reason != "test_not_allowed" {
		t.Fatalf("test row: %+v", tst)
	}
	if len(page.Filters.Templates) != 3 || page.Filters.Templates[2].ID != "raw" || len(page.Filters.Keys) != 2 {
		t.Fatalf("filters: %+v", page.Filters)
	}

	// Filters and search.
	get := func(q url.Values) logPage {
		t.Helper()
		var p logPage
		s.owner.Do("GET", base+"/messages?"+q.Encode(), nil).MustStatus(t, 200).JSON(t, &p)
		return p
	}
	if p := get(url.Values{"status": {"rejected"}}); len(p.Items) != 1 || p.Counts.All != 4 {
		t.Fatalf("status filter: %d items, counts %+v", len(p.Items), p.Counts)
	}
	if p := get(url.Values{"template": {"welcome"}}); len(p.Items) != 1 || p.Items[0].ID != welcome.ID {
		t.Fatalf("template filter: %+v", p.Items)
	}
	if p := get(url.Values{"key": {keyID}}); len(p.Items) != 3 {
		t.Fatalf("key filter: %d", len(p.Items))
	}
	for _, q := range []string{"pwreset-7f3a91", "u_48213", "onboarding", "sam.rivera", reset.ID} {
		if p := get(url.Values{"q": {q}}); len(p.Items) != 1 {
			t.Fatalf("search %q: %d", q, len(p.Items))
		}
	}
	if p := get(url.Values{"template": {"raw"}}); len(p.Items) != 0 {
		t.Fatalf("raw: %d", len(p.Items))
	}
	// Pages and live updates.
	first := get(url.Values{"limit": {"2"}})
	if len(first.Items) != 2 || first.Next == "" {
		t.Fatalf("page 1: %+v", first)
	}
	second := get(url.Values{"limit": {"2"}, "before": {first.Next}})
	if len(second.Items) != 2 || second.Items[0].ID == first.Items[1].ID {
		t.Fatalf("page 2: %+v", second.Items)
	}
	if p := get(url.Values{"after": {first.Newest}}); len(p.Items) != 0 {
		t.Fatal("nothing is newer yet")
	}
	s.clock.add(time.Second)
	s.call(secret, "/messages", "", resetBody("new@contoso.example")).MustStatus(t, 202)
	if p := get(url.Values{"after": {first.Newest}}); len(p.Items) != 1 {
		t.Fatalf("live: %d", len(p.Items))
	}
	s.owner.Do("GET", base+"/messages?range=custom&from=2020-01-01", nil).MustStatus(t, 422)

	// The detail: summary, timeline, the request as stored.
	var d messageDetail
	s.owner.Do("GET", base+"/messages/"+reset.ID, nil).MustStatus(t, 200).JSON(t, &d)
	kinds := []string{}
	for _, e := range d.Timeline {
		if e.Kind != "webhook" { // webhook deliveries carry the database's clock
			kinds = append(kinds, e.Kind)
		}
	}
	want := "accepted,rendered,routed,handed,delivered"
	if !strings.HasPrefix(strings.Join(kinds, ","), want) || d.Contact == nil || d.Contact.Name != "Sam Rivera" {
		t.Fatalf("timeline %v contact %+v", kinds, d.Contact)
	}
	if d.Request.IdempotencyKey != "pwreset-7f3a91" || d.Request.Body["data"].(map[string]any)["reset_url"] != "••••••" || d.Variables != 3 ||
		strings.Join(d.SensitiveSent, ",") != "reset_url" || d.Route.Primary != "Postmark" || d.Route.Fallback != "Amazon SES" {
		t.Fatalf("detail: %+v", d)
	}
	if !d.Content.Available || !d.Content.CanView || d.Resend.Available || d.Resend.Reason != "sensitive" || len(d.Webhooks) != 2 {
		t.Fatalf("content %+v resend %+v webhooks %+v", d.Content, d.Resend, d.Webhooks)
	}
	// The request reads as the app wrote it: template, to, data in the
	// template's order, tags, metadata.
	var raw struct {
		Request struct{ Body json.RawMessage }
	}
	s.owner.Do("GET", base+"/messages/"+reset.ID, nil).MustStatus(t, 200).JSON(t, &raw)
	order := []string{`"template"`, `"to"`, `"data"`, `"first_name"`, `"reset_url"`, `"expires_in"`, `"tags"`, `"metadata"`}
	at := -1
	for _, k := range order {
		i := strings.Index(string(raw.Request.Body), k)
		if i <= at {
			t.Fatalf("request order at %s: %s", k, raw.Request.Body)
		}
		at = i
	}
	s.owner.Do("GET", base+"/messages/"+uuid.Must(uuid.NewV7()).String(), nil).MustStatus(t, 404)
	s.owner.Do("GET", base+"/messages/not-an-id", nil).MustStatus(t, 404)

	// Content needs transactional:content, and every view is audited.
	aisha := s.env.Client()
	aisha.SignInDemo("aisha@acme.io")
	var ad messageDetail
	aisha.Do("GET", base+"/messages/"+reset.ID, nil).MustStatus(t, 200).JSON(t, &ad)
	if ad.Content.CanView || len(ad.Content.Helpers) == 0 {
		t.Fatalf("viewer content state: %+v", ad.Content)
	}
	aisha.Do("GET", base+"/messages/"+reset.ID+"/content", nil).MustStatus(t, 403)
	aisha.Do("GET", base+"/messages/export", nil).MustStatus(t, 403)
	var content struct {
		HTML    string
		Headers []struct{ Name, Value string }
		Masked  []string
	}
	s.owner.Do("GET", base+"/messages/"+reset.ID+"/content", nil).MustStatus(t, 200).JSON(t, &content)
	if strings.Contains(content.HTML, "tok_9f8e7d6c5b4a") || !strings.Contains(content.HTML, "••••••") || len(content.Headers) < 4 || content.Headers[1].Name != "From" {
		t.Fatalf("content: %+v", content.Headers)
	}
	if n := s.count(`SELECT count(*) FROM audit_events WHERE action = 'transactional.content_viewed' AND target_id = $1`, reset.ID); n != 1 {
		t.Fatalf("content views audited: %d", n)
	}
	// Agents read the log without content.
	var agent struct{ Secret string }
	s.owner.Do("POST", "/workspaces/acme-newsletter/mcp-agents", map[string]any{"name": "Claude", "permissions": []string{"transactional:view", "transactional:content"}}).
		MustStatus(t, 201).JSON(t, &agent)
	ag := s.env.Client()
	ag.Bearer = agent.Secret
	ag.Do("GET", base+"/messages", nil).MustStatus(t, 200)
	ag.Do("GET", base+"/messages/"+reset.ID+"/content", nil).MustStatus(t, 403)

	// Export, audited.
	r := s.owner.Do("GET", base+"/messages/export?range=24h", nil).MustStatus(t, 200)
	csvText := string(r.Body)
	if !strings.HasPrefix(csvText, "accepted_at,message_id,recipient") || !strings.Contains(csvText, "jordan@example.org") || strings.Count(csvText, "\n") != 6 {
		t.Fatalf("export:\n%s", csvText)
	}
	if n := s.count(`SELECT count(*) FROM audit_events WHERE action = 'transactional.log_exported'`); n != 1 {
		t.Fatalf("exports audited: %d", n)
	}

	// Resend: the same data, a new message ID; people only, audited.
	s.call(secret, "/messages/"+welcome.ID+"/resend", "", map[string]any{}).MustStatus(t, 403)
	s.owner.Do("POST", base+"/messages/"+reset.ID+"/resend", nil).MustStatus(t, 409)
	var again sendResult
	s.owner.Do("POST", base+"/messages/"+welcome.ID+"/resend", nil).MustStatus(t, 202).JSON(t, &again)
	if again.ID == "" || again.ID == welcome.ID || again.Status != "queued" {
		t.Fatalf("resend: %+v", again)
	}
	var from string
	s.one(`SELECT resent_from::text || ' ' || source FROM transactional_messages WHERE message_id = $1`, &from, again.ID)
	if from != welcome.ID+" resend" {
		t.Fatalf("resent from %q", from)
	}
	if n := s.count(`SELECT count(*) FROM audit_events WHERE action = 'transactional.message_resent' AND target_id = $1`, again.ID); n != 1 {
		t.Fatal("resend not audited")
	}

	// A contact's activity: its messages and opt-outs.
	var act struct {
		Messages []struct{ ID, Status string }
	}
	var sam string
	s.one(`SELECT id::text FROM contacts WHERE workspace_id = $1 AND email = 'sam.rivera@outlook.com'`, &sam, s.ws)
	s.owner.Do("GET", base+"/contacts/"+sam, nil).MustStatus(t, 200).JSON(t, &act)
	if len(act.Messages) != 1 || act.Messages[0].ID != reset.ID {
		t.Fatalf("contact activity: %+v", act)
	}

	// Retention: content goes after the retention, the log after 90 days.
	s.exec(`UPDATE transactional_messages SET accepted_at = accepted_at - interval '31 days' WHERE message_id = $1`, reset.ID)
	res, err := s.env.App.Transactional().Retain(s.ctx)
	if err != nil || res.Content != 1 {
		t.Fatalf("retention: %+v %v", res, err)
	}
	p := s.owner.Do("GET", base+"/messages/"+reset.ID+"/content", nil).MustStatus(t, 410).Problem(t)
	if p["code"] != "content_removed" {
		t.Fatalf("removed: %v", p)
	}
	s.exec(`UPDATE transactional_messages SET accepted_at = accepted_at - interval '60 days' WHERE message_id = $1`, reset.ID)
	if res, err = s.env.App.Transactional().Retain(s.ctx); err != nil || res.Messages != 1 {
		t.Fatalf("retention: %+v %v", res, err)
	}
	s.owner.Do("GET", base+"/messages/"+reset.ID, nil).MustStatus(t, 404)
}
