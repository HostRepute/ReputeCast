package publicpages_test

import (
	"context"
	"html"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

// Texts on the public pages (SMS, design expansion phase 8): the phone
// field with its country picker and the SMS consent box on the subscribe
// page, "We've also texted…" after the signup, and the preference center's
// Texts switch per list. SMS consent keeps its evidence; email consent never
// signs anyone up for texts.
func TestTextsOnThePublicPages(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	formPath := wsAPI + "/forms/" + ids.signup.String()
	var f struct {
		Name           string           `json:"name"`
		Kind           string           `json:"kind"`
		Fields         []map[string]any `json:"fields"`
		ListIDs        []string         `json:"listIds"`
		DefaultListIDs []string         `json:"defaultListIds"`
		ListChoice     bool             `json:"listChoice"`
		DoubleOptIn    bool             `json:"doubleOptIn"`
		ConsentText    string           `json:"consentText"`
		Spam           map[string]any   `json:"spam"`
		AfterSubmit    string           `json:"afterSubmit"`
		RedirectURL    string           `json:"redirectUrl"`
		AllowedOrigins []string         `json:"allowedOrigins"`
		Content        map[string]any   `json:"content"`
	}
	linh.Do(http.MethodGet, formPath, nil).MustStatus(t, 200).JSON(t, &f)
	defaultText, _ := f.Content["smsConsentText"].(string)
	if !strings.Contains(defaultText, "Msg & data rates may apply") || !strings.Contains(defaultText, "text message terms") || !strings.Contains(defaultText, "STOP") {
		t.Fatalf("default SMS consent text: %q", defaultText)
	}
	input := func(fields []map[string]any, sms bool) map[string]any {
		content := maps.Clone(f.Content)
		content["smsConsent"] = sms
		return map[string]any{"name": f.Name, "kind": f.Kind, "fields": fields, "listIds": f.ListIDs, "defaultListIds": f.DefaultListIDs, "listChoice": f.ListChoice,
			"doubleOptIn": f.DoubleOptIn, "consentText": f.ConsentText, "spam": f.Spam, "afterSubmit": f.AfterSubmit, "redirectUrl": f.RedirectURL,
			"allowedOrigins": f.AllowedOrigins, "content": content, "publish": true}
	}

	t.Run("the SMS box needs the phone field", func(t *testing.T) {
		r := linh.Do(http.MethodPut, formPath, input(f.Fields, true))
		if r.Status != 422 || !strings.Contains(string(r.Body), "body.content.smsConsent") {
			t.Fatalf("SMS consent without a number: %d %s", r.Status, r.Body)
		}
		fields := append(slices.Clone(f.Fields), map[string]any{"key": "phone", "label": "", "required": false})
		var saved struct{ Content map[string]any }
		linh.Do(http.MethodPut, formPath, input(fields, true)).MustStatus(t, 200).JSON(t, &saved)
		if saved.Content["smsConsent"] != true || saved.Content["smsConsentVersion"] != float64(1) {
			t.Fatalf("saved: %v", saved.Content)
		}
	})

	t.Run("the subscribe page asks for the number, and the SMS box starts unchecked", func(t *testing.T) {
		page := get(t, env, "/p/acme-newsletter/subscribe")
		must(t, page, 200, `name="phone_country"`, `type="tel" name="phone"`, `autocomplete="tel"`, `<option value="US" selected>United States +1</option>`,
			`name="sms_consent" value="1"><span>Text me updates from Acme.`, `/sms-terms">text message terms</a>`)
		lacks(t, page, `name="sms_consent" value="1" checked`)
		script := get(t, env, "/p/acme-newsletter/forms/frm_8c2e1a.js")
		must(t, script, 200, `"smsConsent":"Text me updates from Acme.`, `"phone":{"label":"Country"`)
	})

	post := func(v url.Values) resp {
		v.Set("_t", formToken(env, "frm_8c2e1a", time.Minute))
		v.Set("list", ids.digest.String())
		v.Set("consent", "1")
		return do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", v, hdr{"Origin": env.Server.URL, "User-Agent": browserUA})
	}
	var robin uuid.UUID
	t.Run("a signup with SMS consent keeps its evidence and texts the confirmation", func(t *testing.T) {
		r := post(url.Values{"email": {"robin.texts@fastmail.com"}, "phone": {"(312) 555-0199"}, "phone_country": {"US"}, "sms_consent": {"1"},
			"_page": {"https://news.acme.io/subscribe?utm_source=summit"}})
		must(t, r, 200, "Check your inbox", "We've also texted +1 312 ••• 0199 to confirm.")
		var ok bool
		if robin, ok = contactID(t, env, ids.ws, "robin.texts@fastmail.com"); !ok {
			t.Fatal("no contact")
		}
		if ph := one[string](t, env, `SELECT phone || ' ' || phone_country FROM contacts WHERE id = $1`, robin); ph != "+13125550199 US" {
			t.Errorf("number: %q", ph)
		}
		var status, text, version, page, ip, ua string
		if err := env.App.Pool.QueryRow(ctx, `SELECT cc.status, e.consent_text, e.consent_version, e.page, host(e.ip), e.user_agent
			FROM channel_consents cc JOIN channel_consent_events e ON e.contact_id = cc.contact_id AND e.list_id = cc.list_id AND e.channel = 'sms'
			WHERE cc.contact_id = $1 AND cc.list_id = $2 AND cc.channel = 'sms' ORDER BY e.occurred_at LIMIT 1`, robin, ids.digest).
			Scan(&status, &text, &version, &page, &ip, &ua); err != nil {
			t.Fatal(err)
		}
		if status != "subscribed" || text != defaultText || version != "1" || page != "news.acme.io/subscribe" || ip != "127.0.0.1" || ua != browserUA {
			t.Errorf("evidence: %s %q v%s %s %s %s", status, text, version, page, ip, ua)
		}
		if n := one[int](t, env, `SELECT count(*) FROM channel_messages WHERE contact_id = $1 AND kind = 'confirmation' AND recipient = '+13125550199'`, robin); n != 1 {
			t.Errorf("%d confirmation texts", n)
		}
		// Email stays pending until the link is clicked; texts don't wait for it.
		if st := membership(t, env, robin, ids.digest); st != "pending" {
			t.Errorf("email: %s", st)
		}
	})

	t.Run("SMS consent without a number asks for one", func(t *testing.T) {
		r := post(url.Values{"email": {"sky.nophone@fastmail.com"}, "phone": {""}, "sms_consent": {"1"}})
		must(t, r, 422, "Add your mobile number to get texts")
		r = post(url.Values{"email": {"sky.nophone@fastmail.com"}, "phone": {"12"}, "phone_country": {"US"}})
		must(t, r, 422, "That doesn't look like a phone number")
	})

	t.Run("the preference center's Texts switch per list", func(t *testing.T) {
		path := "/p/acme-newsletter/preferences/" + sign(env, linktoken.Preferences, ids.ws, robin, uuid.Nil)
		page := get(t, env, path)
		must(t, page, 200, `name="sms_list" value="`+ids.digest.String()+`" checked`, "Texts go to +1 312 ••• 0199.")
		m := regexp.MustCompile(`name="sms_shown" value="([^"]*)"`).FindStringSubmatch(page.Body)
		e := regexp.MustCompile(`name="shown" value="([^"]*)"`).FindStringSubmatch(page.Body)
		if m == nil || e == nil {
			t.Fatalf("no shown lists:\n%s", clip(page.Body))
		}
		// Texts off for the Digest, on for Product Updates.
		r := do(t, env, http.MethodPost, path, url.Values{"shown": {e[1]}, "list": {ids.digest.String()}, "sms_shown": {m[1]}, "sms_list": {ids.updates.String()}}, hdr{"User-Agent": browserUA})
		if r.Status != 303 {
			t.Fatalf("save: %d\n%s", r.Status, clip(r.Body))
		}
		sms := func(list uuid.UUID) string {
			return one[string](t, env, `SELECT status FROM channel_consents WHERE contact_id = $1 AND list_id = $2 AND channel = 'sms'`, robin, list)
		}
		if sms(ids.digest) != "not_subscribed" || sms(ids.updates) != "subscribed" {
			t.Errorf("texts: digest %s, updates %s", sms(ids.digest), sms(ids.updates))
		}
		text := one[string](t, env, `SELECT consent_text FROM channel_consent_events WHERE contact_id = $1 AND list_id = $2 AND channel = 'sms' AND source = 'preference_center'`, robin, ids.updates)
		if !strings.Contains(text, "Texts about Product Updates from Acme") {
			t.Errorf("consent text: %q", text)
		}
		// Texts on for Product Updates never subscribe the address to its emails.
		if st := membership(t, env, robin, ids.updates); st != "none" {
			t.Errorf("Product Updates' email changed with its texts: %s", st)
		}
		// A number that replied STOP stays off here: only START brings it back.
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO channel_suppressions (id, workspace_id, channel, phone, reason, source, created_at)
			VALUES ($1, $2, 'sms', '+13125550199', 'opted_out', 'keyword', now())`, uuid.Must(uuid.NewV7()), ids.ws); err != nil {
			t.Fatal(err)
		}
		page = get(t, env, path)
		must(t, page, 200, "replied STOP, so texts are off", `name="sms_list" value="`+ids.updates.String()+`" disabled`)
	})
}

// The website's account page has the same Texts switches (brief §3.3),
// recorded with the account page as their source.
func TestTextsOnTheAccountPage(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET phone = '+13125550142', phone_country = 'US', phone_line_type = 'mobile' WHERE id = $1`, ids.sam); err != nil {
		t.Fatal(err)
	}
	sam := env.Client()
	sam.Header = http.Header{"Cookie": {"rc_member=demo-session-sam-" + ids.ws.String()}}
	r := sam.Do(http.MethodGet, "/p/acme-newsletter/account", nil)
	body := html.UnescapeString(string(r.Body))
	if r.Status != 200 || !strings.Contains(body, `id="texts"`) || !strings.Contains(body, "Texts go to +1 312 ••• 0142.") ||
		!strings.Contains(body, `name="sms_list" value="`+ids.digest.String()+`" aria-labelledby`) {
		t.Fatalf("account page: %d\n%s", r.Status, clip(body))
	}
	m := regexp.MustCompile(`name="sms_shown" value="([^"]*)"`).FindStringSubmatch(body)
	if m == nil {
		t.Fatal("no sms_shown")
	}
	saved := do(t, env, http.MethodPost, "/p/acme-newsletter/account/texts", url.Values{"sms_shown": {m[1]}, "sms_list": {ids.digest.String()}},
		hdr{"Cookie": "rc_member=demo-session-sam-" + ids.ws.String(), "Origin": env.Server.URL, "Sec-Fetch-Site": "same-origin"})
	if saved.Status != http.StatusSeeOther || !strings.HasSuffix(saved.Header.Get("Location"), "/account#texts") {
		t.Fatalf("save: %d %v", saved.Status, saved.Header)
	}
	var status, source, page, text string
	if err := env.App.Pool.QueryRow(ctx, `SELECT cc.status, e.source, e.page, e.consent_text FROM channel_consents cc
		JOIN channel_consent_events e ON e.contact_id = cc.contact_id AND e.list_id = cc.list_id AND e.channel = 'sms' AND e.source = 'account_page'
		WHERE cc.contact_id = $1 AND cc.list_id = $2 AND cc.channel = 'sms'`, ids.sam, ids.digest).Scan(&status, &source, &page, &text); err != nil {
		t.Fatal(err)
	}
	if status != "subscribed" || page != "Account page" || !strings.Contains(text, "Texts about Weekly Digest from Acme") {
		t.Errorf("consent: %s %s %q", status, page, text)
	}
	if n := one[int](t, env, `SELECT count(*) FROM channel_messages WHERE contact_id = $1 AND kind = 'confirmation' AND recipient = '+13125550142'`, ids.sam); n != 1 {
		t.Errorf("%d confirmation texts", n)
	}
}
