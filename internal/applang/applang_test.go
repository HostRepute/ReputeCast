package applang_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/team"
)

type meView struct {
	User struct {
		Preferences applang.MemberLanguage `json:"preferences"`
	} `json:"user"`
	Instance struct {
		DefaultLanguage     string                   `json:"defaultLanguage"`
		DefaultRegionFormat string                   `json:"defaultRegionFormat"`
		Languages           []applang.LanguageOption `json:"languages"`
	} `json:"instance"`
}

func choose(lang, region string) map[string]any {
	return map[string]any{"language": lang, "regionFormat": region, "hourCycle": "auto", "weekStart": "auto"}
}

func auditRows(t *testing.T, env *apptest.Env, action string) []map[string]any {
	t.Helper()
	rows, err := env.App.Pool.Query(context.Background(), `SELECT coalesce(before, '{}'), coalesce(after, '{}'), coalesce(metadata, '{}') FROM audit_events WHERE action = $1 ORDER BY occurred_at, id`, action)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var before, after, meta []byte
		if err := rows.Scan(&before, &after, &meta); err != nil {
			t.Fatal(err)
		}
		row := map[string]any{}
		for k, b := range map[string][]byte{"before": before, "after": after, "metadata": meta} {
			var v map[string]any
			_ = json.Unmarshal(b, &v)
			row[k] = v
		}
		out = append(out, row)
	}
	return out
}

func TestMemberLanguage(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	c := env.Client()
	c.SignIn("linh@acme.io", "")

	var me meView
	c.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if p := me.User.Preferences; p != (applang.MemberLanguage{Language: "default", RegionFormat: "auto", HourCycle: "auto", WeekStart: "auto"}) {
		t.Fatalf("nothing chosen yet: %+v", p)
	}
	if me.Instance.DefaultLanguage != "en" || me.Instance.DefaultRegionFormat != "standard" || len(me.Instance.Languages) != 2 ||
		me.Instance.Languages[1] != (applang.LanguageOption{Code: "vi", Name: "Tiếng Việt"}) {
		t.Fatalf("instance languages: %+v", me.Instance)
	}

	var saved applang.MemberLanguage
	c.Do("PUT", "/me/language", map[string]any{"language": "vi", "regionFormat": "auto", "hourCycle": "h12", "weekStart": "sunday"}).MustStatus(t, 200).JSON(t, &saved)
	if saved != (applang.MemberLanguage{Language: "vi", RegionFormat: "auto", HourCycle: "h12", WeekStart: "sunday"}) {
		t.Fatalf("saved: %+v", saved)
	}
	c.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if me.User.Preferences != saved {
		t.Fatalf("/me after saving: %+v", me.User.Preferences)
	}
	rows := auditRows(t, env, "account.language_update")
	if len(rows) != 1 || rows[0]["before"].(map[string]any)["language"] != "default" || rows[0]["after"].(map[string]any)["language"] != "vi" {
		t.Fatalf("audit: %+v", rows)
	}
	// Saving the same choices again changes nothing and logs nothing.
	c.Do("PUT", "/me/language", map[string]any{"language": "vi", "regionFormat": "auto", "hourCycle": "h12", "weekStart": "sunday"}).MustStatus(t, 200)
	if n := len(auditRows(t, env, "account.language_update")); n != 1 {
		t.Fatalf("an unchanged save was audited (%d entries)", n)
	}

	// The pseudo-locale is off outside development; unknown values are refused.
	p := c.Do("PUT", "/me/language", choose("en-XA", "auto")).MustStatus(t, 422).Problem(t)
	if !strings.Contains(p["detail"].(string), "i18n.pseudo_locale") {
		t.Fatalf("pseudo-locale: %v", p)
	}
	c.Do("PUT", "/me/language", choose("de", "auto")).MustStatus(t, 422)
	c.Do("PUT", "/me/language", choose("en", "xx-XX")).MustStatus(t, 422)
	env.Client().Do("PUT", "/me/language", choose("en", "auto")).MustStatus(t, 401)

	// Back to the instance default: stored as "not chosen".
	c.Do("PUT", "/me/language", choose("default", "auto")).MustStatus(t, 200).JSON(t, &saved)
	if saved.Language != "default" {
		t.Fatalf("back to default: %+v", saved)
	}
}

func TestPseudoLocaleWhenTurnedOn(t *testing.T) {
	env := apptest.New(t)
	i18n.SetPseudo(true)
	t.Cleanup(func() { i18n.SetPseudo(env.App.Cfg.PseudoLocale()) })
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	c := env.Client()
	c.SignIn("linh@acme.io", "")
	c.Do("PUT", "/me/language", choose("en-XA", "auto")).MustStatus(t, 200)
	var me meView
	c.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if me.User.Preferences.Language != "en-XA" || len(me.Instance.Languages) != 3 || me.Instance.Languages[2].Name != "Pseudo (+40%)" {
		t.Fatalf("pseudo-locale: %+v", me)
	}
	// Server messages come bracketed, so text that skipped the catalogs stands out.
	anon := env.Client()
	anon.Header = http.Header{"Accept-Language": {"en-XA"}}
	p := anon.Do("GET", "/me", nil).MustStatus(t, 401).Problem(t)
	if d := p["detail"].(string); !strings.HasPrefix(d, "⟦Ĥáñţé ðéñĝ") && !strings.HasPrefix(d, "⟦") {
		t.Fatalf("pseudo detail: %q", d)
	}
}

func TestInstanceDefaults(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "admin@acme.io", Name: "Linh Nguyen", InstanceAdmin: true})
	member := env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io", Name: "Aisha Bello"})
	c := env.Client()
	c.SignIn("admin@acme.io", "")

	var st map[string]any
	c.Do("GET", "/admin/instance", nil).MustStatus(t, 200).JSON(t, &st)
	if st["defaultLanguage"] != "en" || st["defaultRegionFormat"] != "standard" {
		t.Fatalf("defaults: %v", st)
	}
	save := map[string]any{"name": "Acme ReputeCast", "publicUrl": env.Server.URL, "defaultTimezone": "America/New_York", "defaultLanguage": "vi", "defaultRegionFormat": "vi-VN"}
	c.Do("PUT", "/admin/instance", save).MustStatus(t, 200).JSON(t, &st)
	if st["defaultLanguage"] != "vi" || st["defaultRegionFormat"] != "vi-VN" {
		t.Fatalf("saved: %v", st)
	}
	var meta map[string]any
	env.Client().Do("GET", "/meta", nil).MustStatus(t, 200).JSON(t, &meta)
	if meta["defaultLanguage"] != "vi" || meta["defaultRegionFormat"] != "vi-VN" || len(meta["languages"].([]any)) != 2 {
		t.Fatalf("meta: %v", meta)
	}
	rows := auditRows(t, env, "instance.settings_updated")
	if len(rows) == 0 || !strings.Contains(rows[len(rows)-1]["metadata"].(map[string]any)["summary"].(string), "default language vi") {
		t.Fatalf("audit: %+v", rows)
	}
	// Members who haven't chosen read the instance default.
	if l := applang.ForUser(context.Background(), env.App.Q, member); l != (i18n.Lang{Locale: i18n.Vietnamese, Region: "vi-VN"}) {
		t.Fatalf("member without choices: %+v", l)
	}
	// Omitting the defaults keeps them; the pseudo-locale and unknown languages are refused.
	delete(save, "defaultLanguage")
	delete(save, "defaultRegionFormat")
	c.Do("PUT", "/admin/instance", save).MustStatus(t, 200).JSON(t, &st)
	if st["defaultLanguage"] != "vi" {
		t.Fatalf("omitted: %v", st)
	}
	save["defaultLanguage"] = "en-XA"
	c.Do("PUT", "/admin/instance", save).MustStatus(t, 422)
	// Only instance admins.
	other := env.Client()
	other.SignIn("aisha@acme.io", "")
	other.Do("PUT", "/admin/instance", map[string]any{"name": "X", "publicUrl": env.Server.URL, "defaultTimezone": "UTC", "defaultLanguage": "en"}).MustStatus(t, 403)
}

func TestAccessErrorsInTheReadersLanguage(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	viewer := env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io", Name: "Aisha Bello"})
	ws := env.CreateWorkspace("Acme Newsletter", owner.ID)
	env.AddMember(ws, viewer.ID, "viewer")
	c := env.Client()
	c.SignIn("aisha@acme.io", "")
	path := "/workspaces/" + ws.Slug + "/invitations/options"

	// API clients get English unless they send Accept-Language.
	p := c.Do("GET", path, nil).MustStatus(t, 403).Problem(t)
	if p["detail"] != "Inviting members needs the team:invite permission. Ask Linh Nguyen." || p["code"] != "permission_denied" {
		t.Fatalf("english: %v", p)
	}
	c.Header = http.Header{"Accept-Language": {"vi-VN,vi;q=0.9,en;q=0.8"}}
	p = c.Do("GET", path, nil).MustStatus(t, 403).Problem(t)
	if p["detail"] != "Để mời thành viên, bạn cần quyền team:invite. Hãy hỏi Linh Nguyen." || p["permission"] != "team:invite" {
		t.Fatalf("vietnamese: %v", p)
	}
	c.Header = http.Header{"Accept-Language": {"fr-FR"}}
	if p = c.Do("GET", path, nil).MustStatus(t, 403).Problem(t); !strings.HasPrefix(p["detail"].(string), "Inviting members") {
		t.Fatalf("a language ReputeCast doesn't have: %v", p)
	}
	// Without the header, a member reads their own language.
	c.Header = nil
	c.Do("PUT", "/me/language", choose("vi", "auto")).MustStatus(t, 200)
	if p = c.Do("GET", path, nil).MustStatus(t, 403).Problem(t); !strings.HasPrefix(p["detail"].(string), "Để mời thành viên") {
		t.Fatalf("member's language: %v", p)
	}
	// Sign-in problems too.
	anon := env.Client()
	anon.Header = http.Header{"Accept-Language": {"vi"}}
	if p = anon.Do("GET", "/me", nil).MustStatus(t, 401).Problem(t); p["detail"] != "Hãy đăng nhập để tiếp tục." || p["code"] != "unauthenticated" {
		t.Fatalf("sign in: %v", p)
	}
}

type fakeMail struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (f *fakeMail) SendSystem(_ context.Context, m mailer.Message) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return "Postmark", nil
}

func (f *fakeMail) SystemAvailable(context.Context, *uuid.UUID) bool { return true }

func (f *fakeMail) last(t *testing.T) mailer.Message {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) == 0 {
		t.Fatal("no email was sent")
	}
	return f.sent[len(f.sent)-1]
}

// withMail serves accounts and team with a fake mailer on env's database.
func withMail(t *testing.T, env *apptest.Env, m mailer.Sender) *apptest.Env {
	t.Helper()
	r := chi.NewRouter()
	r.Use(server.AuditContext)
	h := humachi.New(r, app.NewHumaConfig())
	h.UseMiddleware(env.App.Auth.Middleware(h))
	accounts.Register(h, &accounts.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Cfg: env.App.Cfg,
		Mail: m, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Breach: authn.NewBreachChecker()})
	team.Register(h, &team.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Mail: m,
		PublicURL: func(context.Context) string { return env.Server.URL }})
	applang.Register(h, &applang.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &apptest.Env{T: t, App: env.App, Server: srv}
}

func TestMemberEmailsInTheirLanguage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	mail := &fakeMail{}
	menv := withMail(t, env, mail)

	// A member who chose Tiếng Việt gets their password reset in Vietnamese.
	aisha := menv.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("PUT", "/me/language", choose("vi", "auto")).MustStatus(t, 200)
	menv.Client().Do("POST", "/auth/password-reset", map[string]any{"email": "aisha@acme.io"}).MustStatus(t, 200)
	msg := mail.last(t)
	if msg.Lang.Locale != i18n.Vietnamese || !strings.HasPrefix(msg.Subject, "Đặt lại mật khẩu") ||
		!strings.Contains(msg.Text, "Đặt mật khẩu mới: "+env.Server.URL+"/reset-password/") || !strings.Contains(msg.HTML, "<b>aisha@acme.io</b>") {
		t.Fatalf("reset email: %+v", msg)
	}

	// Invitations go out in the invitation language, the instance default unless picked.
	linh := menv.Client()
	linh.SignInDemo("linh@acme.io")
	var opts struct {
		Languages       []applang.LanguageOption `json:"languages"`
		DefaultLanguage string                   `json:"defaultLanguage"`
		DefaultRoleID   string                   `json:"defaultRoleId"`
	}
	linh.Do("GET", "/workspaces/acme-newsletter/invitations/options", nil).MustStatus(t, 200).JSON(t, &opts)
	if opts.DefaultLanguage != "en" || len(opts.Languages) != 2 {
		t.Fatalf("invite options: %+v", opts)
	}
	var res struct {
		Invitations []struct{ ID string }
	}
	linh.Do("POST", "/workspaces/acme-newsletter/invitations", map[string]any{"emails": []string{"minh@example.vn"}, "roleId": opts.DefaultRoleID, "language": "vi"}).MustStatus(t, 201).JSON(t, &res)
	msg = mail.last(t)
	if msg.To != "minh@example.vn" || msg.Lang.Locale != i18n.Vietnamese || !strings.Contains(msg.Subject, "đã mời bạn vào Acme Newsletter") ||
		!strings.Contains(msg.Text, "Chấp nhận lời mời: ") || !strings.Contains(msg.Text, "thg") {
		t.Fatalf("vietnamese invitation: %+v", msg)
	}
	if rows := auditRows(t, env, "team.invited"); rows[len(rows)-1]["after"].(map[string]any)["language"] != "vi" {
		t.Fatalf("the audit entry names the language: %+v", rows[len(rows)-1])
	}
	// A resend keeps the invitation's language.
	linh.Do("POST", "/workspaces/acme-newsletter/invitations/"+res.Invitations[0].ID+"/resend", nil).MustStatus(t, 200)
	if msg = mail.last(t); msg.Lang.Locale != i18n.Vietnamese {
		t.Fatalf("resend: %+v", msg)
	}
	linh.Do("POST", "/workspaces/acme-newsletter/invitations", map[string]any{"emails": []string{"sam.rivera@outlook.com"}, "roleId": opts.DefaultRoleID}).MustStatus(t, 201)
	if msg = mail.last(t); msg.Lang.Locale != i18n.English || msg.Subject != "Linh Nguyen invited you to Acme Newsletter" {
		t.Fatalf("english invitation: %+v", msg)
	}
	linh.Do("POST", "/workspaces/acme-newsletter/invitations", map[string]any{"emails": []string{"x@example.com"}, "roleId": opts.DefaultRoleID, "language": "en-XA"}).MustStatus(t, 422)

	// The invitation page reads the invitation's language.
	token := linkAfter(t, mail.sent[len(mail.sent)-3].Text, "/invite/")
	var view map[string]any
	menv.Client().Do("GET", "/invitations/"+token, nil).MustStatus(t, 200).JSON(t, &view)
	if view["language"] != "vi" {
		t.Fatalf("invitation view language: %v", view["language"])
	}
}

func linkAfter(t *testing.T, text, prefix string) string {
	t.Helper()
	i := strings.LastIndex(text, prefix)
	if i < 0 {
		t.Fatalf("no %q in %q", prefix, text)
	}
	tok := text[i+len(prefix):]
	if j := strings.IndexAny(tok, " \n"); j >= 0 {
		tok = tok[:j]
	}
	return tok
}

// The browser's suggestion is offered once per member, whatever the
// browser: answering it is remembered on the server and audited once.
func TestLanguageSuggestionAnsweredOnce(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	c := env.Client()
	c.SignIn("linh@acme.io", "")

	var me struct {
		User struct {
			Answered bool `json:"languageSuggestionAnswered"`
		} `json:"user"`
	}
	c.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if me.User.Answered {
		t.Fatal("a new member hasn't answered the suggestion")
	}
	c.Do("PUT", "/me/language/suggestion", map[string]any{"answer": "keep", "language": "vi"}).MustStatus(t, 204)
	c.Do("PUT", "/me/language/suggestion", map[string]any{"answer": "dismiss", "language": "vi"}).MustStatus(t, 204)
	c.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if !me.User.Answered {
		t.Fatal("/me should say the suggestion was answered")
	}
	rows := auditRows(t, env, "account.language_suggestion_answered")
	if len(rows) != 1 || rows[0]["metadata"].(map[string]any)["answer"] != "keep" || rows[0]["metadata"].(map[string]any)["language"] != "vi" {
		t.Fatalf("audit: %+v", rows)
	}
	c.Do("PUT", "/me/language/suggestion", map[string]any{"answer": "maybe", "language": "vi"}).MustStatus(t, 422)
	env.Client().Do("PUT", "/me/language/suggestion", map[string]any{"answer": "keep", "language": "vi"}).MustStatus(t, 401)
}
