package whatsapp_test

// WhatsApp in the demo (brief §4 "WhatsApp"): the account card, the
// templates, consent, Sofia's draft with its preview and cost, Sam Rivera's
// open window, and the Summit flow's push and WhatsApp steps.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/whatsapp"
)

const demoWS = "/workspaces/acme-newsletter"

func TestWhatsAppDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")

	var ov whatsapp.WhatsAppOverview
	linh.Do(http.MethodGet, demoWS+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	a := ov.Account
	if a == nil || a.Status != "connected" || a.BusinessName != "Acme Inc" || a.BusinessHint != "…4410" || a.Number != "+1 312 555 0110" || a.DisplayName != "Acme" ||
		a.NameStatus != "approved" || a.Quality != "high" || a.MessagingLimit != 1000 || a.ConnectedBy != "Marcus Chen" || a.VerifiedAt == nil || !a.Demo ||
		a.VerifyToken != "rc-wa-7Q2mX9vKp4" {
		t.Fatalf("account: %+v", a)
	}
	if ov.Consent != 214 || ov.TemplateCounts.Approved != 2 || ov.TemplateCounts.Pending != 1 || ov.TemplateCounts.Rejected != 1 ||
		ov.RatesAsOf.Format("2006-01-02") != "2026-10-01" {
		t.Fatalf("overview: consent %d, templates %+v, rates %v", ov.Consent, ov.TemplateCounts, ov.RatesAsOf)
	}
	for _, tp := range ov.Templates {
		if tp.Name == "digest_highlights" && (tp.Reason != "Add sample values for every variable." || tp.Buttons[0].Text != "Stop promotions") {
			t.Fatalf("rejected template: %+v", tp)
		}
		if tp.Name == "summit_day_before" && (tp.Variables[0].Label != "first name" || tp.Variables[1].Label != "doors-open time") {
			t.Fatalf("variables: %+v", tp.Variables)
		}
	}
	var evs whatsapp.WhatsAppEventList
	linh.Do(http.MethodGet, demoWS+"/whatsapp/events", nil).MustStatus(t, 200).JSON(t, &evs)
	if len(evs.Items) != 7 || evs.Items[0].Kind != "message" || !strings.Contains(evs.Items[0].Number, "•••") {
		t.Fatalf("events: %+v", evs.Items)
	}

	// Sofia's draft: summit_day_before to 214, $0.86, as Sam reads it.
	var list struct {
		Items []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Channel string `json:"channel"`
		} `json:"items"`
	}
	linh.Do(http.MethodGet, demoWS+"/campaigns?channel=whatsapp&pageSize=50", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Name != "Summit: see you tomorrow" {
		t.Fatalf("WhatsApp campaigns: %+v", list.Items)
	}
	var c whatsapp.WhatsAppCampaign
	linh.Do(http.MethodGet, demoWS+"/whatsapp-campaigns/"+list.Items[0].ID, nil).MustStatus(t, 200).JSON(t, &c)
	if c.Template == nil || c.Template.Name != "summit_day_before" || c.Audience.Final != 214 || c.Audience.Without != 1026 || c.Cost.Amount != 0.86 ||
		c.Schedule.Mode != "local" || c.Schedule.LocalMinute == nil || *c.Schedule.LocalMinute != 540 || c.CreatedBy != "Sofia Rossi" {
		t.Fatalf("draft: template %+v audience %+v cost %+v schedule %+v", c.Template, c.Audience, c.Cost, c.Schedule)
	}
	if c.Preview == nil || c.Preview.Contact != "Sam Rivera" ||
		c.Preview.Text != "Hi Sam, see you tomorrow at Acme Summit 2026. Doors open at 8:30 at Lakeside Hall, Chicago." {
		t.Fatalf("preview: %+v", c.Preview)
	}
	var rv whatsapp.WhatsAppReview
	linh.Do(http.MethodGet, demoWS+"/whatsapp-campaigns/"+c.ID+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Verdict != "ready" {
		t.Fatalf("review: %s %+v", rv.Verdict, rv.Checks)
	}

	// Sam Rivera's window is open; his question is the last message.
	var contacts struct {
		Items []struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"items"`
	}
	linh.Do(http.MethodGet, demoWS+"/contacts?q=sam.rivera@outlook.com", nil).MustStatus(t, 200).JSON(t, &contacts)
	if len(contacts.Items) != 1 {
		t.Fatalf("Sam: %+v", contacts.Items)
	}
	var card whatsapp.WhatsAppContactCard
	linh.Do(http.MethodGet, demoWS+"/contacts/"+contacts.Items[0].ID+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &card)
	if !card.Window.Open || card.Window.LastText != "Is there parking at the venue?" || len(card.Thread) != 2 || !card.CanReply || len(card.Lists) == 0 {
		t.Fatalf("Sam's card: %+v", card)
	}
	for _, l := range card.Lists {
		if l.Name == "Acme Summit 2026" && (l.Status != "subscribed" || l.Version != "1") {
			t.Fatalf("Sam's consent: %+v", l)
		}
	}
	// The Summit flow's WhatsApp step as its panel shows it: summit_livestream
	// to the 214 with consent, $0.86, as Sam reads it; Sam's window is open.
	summit := ""
	for _, l := range ov.ConsentLists {
		if l.Name == "Acme Summit 2026" {
			summit = l.ListID
		}
	}
	var est whatsapp.WhatsAppStepEstimate
	linh.Do(http.MethodPost, demoWS+"/whatsapp/step-estimate", map[string]any{"consentListId": summit, "mode": "template", "template": "summit_livestream",
		"language": "en_US", "variables": []map[string]string{{"key": "1", "source": "text", "text": "go.acme.io/k3P9xq"}}}).MustStatus(t, 200).JSON(t, &est)
	if !est.Connected || est.Business == nil || est.Business.Number != "+1 312 555 0110" || len(est.Templates) != 4 || est.Consent != 214 || est.Cost.Amount != 0.86 ||
		est.Cost.Category != "utility" || est.Open < 1 || est.WindowExample == nil || est.WindowExample.Name != "Sam Rivera" {
		t.Fatalf("step estimate: %+v", est)
	}
	if est.Preview == nil || est.Preview.Contact != "Sam Rivera" || est.Preview.Text != "Acme Summit 2026 starts in 15 minutes. Watch the livestream: go.acme.io/k3P9xq" {
		t.Fatalf("step preview: %+v", est.Preview)
	}
	// Free text has no preview to fill; without a list the reach is empty.
	est = whatsapp.WhatsAppStepEstimate{}
	linh.Do(http.MethodPost, demoWS+"/whatsapp/step-estimate", map[string]any{"mode": "text"}).MustStatus(t, 200).JSON(t, &est)
	if !est.Connected || est.Consent != 0 || est.Preview != nil || len(est.Templates) != 4 {
		t.Fatalf("step estimate without a list: %+v", est)
	}

	// A viewer sees it all, without the verify token.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	ov = whatsapp.WhatsAppOverview{}
	aisha.Do(http.MethodGet, demoWS+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Account == nil || ov.Account.VerifyToken != "" || ov.CanManage {
		t.Fatalf("Aisha: %+v", ov.Account)
	}
}
