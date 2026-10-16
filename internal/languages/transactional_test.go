package languages_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

const tbase = ws + "/transactional"

// txnRoute sends the Transactional stream through Postmark, then Amazon
// SES, as phase 1's sample does.
func txnRoute(t *testing.T, c *apptest.Client, primary, fallback string) {
	t.Helper()
	var r struct {
		Streams []struct {
			Stream string
			Splits []struct {
				ProviderID string
				Weight     int
			}
			Fallback   string
			Failover   string
			Conditions []struct {
				Field      string
				Values     []string
				ProviderID string
				Note       string
			}
		}
	}
	c.Do("GET", ws+"/routing", nil).MustStatus(t, 200).JSON(t, &r)
	streams := []map[string]any{}
	for _, st := range r.Streams {
		in := map[string]any{"stream": st.Stream, "fallback": st.Fallback, "failover": st.Failover}
		splits, conds := []map[string]any{}, []map[string]any{}
		for _, sp := range st.Splits {
			splits = append(splits, map[string]any{"providerId": sp.ProviderID, "weight": sp.Weight})
		}
		for _, cd := range st.Conditions {
			conds = append(conds, map[string]any{"field": cd.Field, "values": cd.Values, "providerId": cd.ProviderID, "note": cd.Note})
		}
		if st.Stream == "transactional" {
			splits = []map[string]any{{"providerId": primary, "weight": 100}}
			in["fallback"], in["failover"] = fallback, "automatic"
		}
		in["splits"], in["conditions"] = splits, conds
		streams = append(streams, in)
	}
	c.Do("PUT", ws+"/routing", map[string]any{"streams": streams, "acknowledgeBulkTerms": true}).MustStatus(t, 200)
}

// txnCall calls the send API with a key.
func txnCall(t *testing.T, env *apptest.Env, secret, path string, body any) *apptest.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, env.Server.URL+"/api/v1"+tbase+path, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	return &apptest.Response{Status: resp.StatusCode, Header: resp.Header, Body: raw}
}

type editorTemplate struct {
	Language, SourceLanguage string
	Subject, MJML            string
	DraftRevision            int
	Variables                []map[string]any
}

// Transactional templates' language versions over HTTP: the editor's
// switcher and ?language=, publishing every version together, and the
// version each send and render gets (the language asked for, else the
// linked contact's, else the fallback).
func TestTransactionalLanguages(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	// A Transactional sender, and the route.
	dom := uuid.Must(uuid.NewV7())
	exec(`INSERT INTO sending_domains (id, workspace_id, domain, status, transactional) VALUES ($1, $2, 'notify.acme.io', 'verified', true)`, dom, wsID)
	exec(`INSERT INTO senders (id, workspace_id, domain_id, name, email, reply_to) VALUES ($1, $2, $3, 'Acme Cloud', 'no-reply@notify.acme.io', 'support@acme.io')`,
		uuid.Must(uuid.NewV7()), wsID, dom)
	var postmark, ses string
	if err := env.App.Pool.QueryRow(ctx, `SELECT (SELECT id::text FROM providers WHERE workspace_id = $1 AND adapter = 'postmark'),
		(SELECT id::text FROM providers WHERE workspace_id = $1 AND adapter = 'ses')`, wsID).Scan(&postmark, &ses); err != nil {
		t.Fatal(err)
	}
	txnRoute(t, linh, postmark, ses)
	before := map[string]int{}
	for _, a := range []string{"transactional_template.language.add", "transactional_template.language.save", "transactional_template.language.mark_translated",
		"transactional_template.language.remove"} {
		before[a] = audited(t, env, a)
	}
	logged := func(action string) int { return audited(t, env, action) - before[action] }

	// --- The editor (TransactionalTemplate's language switcher) ---
	var created struct{ DraftRevision int }
	linh.Do("POST", tbase+"/templates", map[string]any{"name": "Password reset", "key": "password-reset", "starter": "password-reset"}).MustStatus(t, 201).JSON(t, &created)
	tpl := tbase + "/templates/password-reset"
	var langs emailLanguages
	linh.Do("GET", tpl+"/languages", nil).MustStatus(t, 200).JSON(t, &langs)
	if langs.Source != "en" || !langs.CanEdit || langs.version(t, "es").Status != "missing" || langs.version(t, "vi").Gets != "en" {
		t.Fatalf("a new template is in English only: %+v", langs)
	}
	linh.Do("POST", tpl+"/languages", map[string]any{"language": "es"}).MustStatus(t, 201).JSON(t, &langs)
	if es := langs.version(t, "es"); es.Status != "draft" || es.Gets != "en" || logged("transactional_template.language.add") != 1 {
		t.Fatalf("Spanish starts as a draft: %+v", es)
	}
	linh.Do("POST", tpl+"/languages", map[string]any{"language": "es"}).MustStatus(t, 409)
	linh.Do("POST", tpl+"/languages", map[string]any{"language": "fr"}).MustStatus(t, 422)

	var source, es editorTemplate
	linh.Do("GET", tpl, nil).MustStatus(t, 200).JSON(t, &source)
	linh.Do("GET", tpl+"?language=es", nil).MustStatus(t, 200).JSON(t, &es)
	if source.Language != "en" || source.SourceLanguage != "en" || es.Language != "es" || es.SourceLanguage != "en" || es.Subject != source.Subject {
		t.Fatalf("the editor opens Spanish as a copy of English: source %+v, es %+v", source, es)
	}
	linh.Do("GET", tpl+"?language=de", nil).MustStatus(t, 404)
	spanish := "Restablece tu contraseña de Acme Cloud"
	linh.Do("PUT", tpl+"/draft?language=es", map[string]any{"baseRevision": es.DraftRevision, "subject": spanish, "mjml": strings.Replace(es.MJML, "Reset", "Restablecer", 1)}).
		MustStatus(t, 200)
	linh.Do("PUT", tpl+"/draft?language=es", map[string]any{"baseRevision": es.DraftRevision, "subject": "Otra"}).MustStatus(t, 409)
	var after editorTemplate
	linh.Do("GET", tpl, nil).MustStatus(t, 200).JSON(t, &after)
	if after.Subject != source.Subject || after.DraftRevision != source.DraftRevision || logged("transactional_template.language.save") != 1 {
		t.Fatalf("saving Spanish leaves the draft alone: %+v (%d saves audited)", after, logged("transactional_template.language.save"))
	}
	// Variables are the template's: changed while editing Spanish, they're
	// saved with the draft.
	linh.Do("GET", tpl+"?language=es", nil).MustStatus(t, 200).JSON(t, &es)
	vars := append(after.Variables, map[string]any{"name": "locale", "type": "text", "required": false, "sensitive": false})
	linh.Do("PUT", tpl+"/draft?language=es", map[string]any{"baseRevision": es.DraftRevision, "subject": spanish, "variables": vars}).MustStatus(t, 200)
	linh.Do("GET", tpl, nil).MustStatus(t, 200).JSON(t, &after)
	if after.DraftRevision != source.DraftRevision+1 || len(after.Variables) != len(vars) || after.Subject != source.Subject {
		t.Fatalf("the new variable is the template's: %+v", after)
	}

	// --- Publishing publishes every version together ---
	var pv struct {
		Languages []struct{ Language, Status string }
	}
	linh.Do("POST", tpl+"/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if len(pv.Languages) != 2 || pv.Languages[0].Language != "en" || pv.Languages[0].Status != "source" || pv.Languages[1].Status != "draft" {
		t.Fatalf("publish dialog languages: %+v", pv.Languages)
	}
	linh.Do("POST", tpl+"/languages/es/mark-translated", map[string]any{}).MustStatus(t, 200).JSON(t, &langs)
	if langs.version(t, "es").Status != "up_to_date" || logged("transactional_template.language.mark_translated") != 1 {
		t.Fatalf("Spanish marked translated: %+v", langs.version(t, "es"))
	}
	linh.Do("POST", tpl+"/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if len(pv.Languages) != 2 || pv.Languages[1].Language != "es" || pv.Languages[1].Status != "up_to_date" {
		t.Fatalf("publish dialog languages: %+v", pv.Languages)
	}
	linh.Do("POST", tpl+"/publish", map[string]any{"baseRevision": after.DraftRevision}).MustStatus(t, 200)

	// --- Sends: the language asked for, else the contact's, else the fallback ---
	var key struct{ Secret string }
	linh.Do("POST", ws+"/api-keys", map[string]any{"name": "App", "permissions": []string{"transactional:send"}, "transactional": map[string]any{"preset": "send_only"}}).
		MustStatus(t, 201).JSON(t, &key)
	data := map[string]any{"first_name": "Lucía", "reset_url": "https://app.acme.io/reset/tok_1", "expires_in": "30 minutes"}
	send := func(to, language string) (lang, contact, reason, subject string) {
		t.Helper()
		body := map[string]any{"template": "password-reset", "to": map[string]any{"email": to}, "data": data}
		if language != "" {
			body["language"] = language
		}
		var res struct{ ID string }
		txnCall(t, env, key.Secret, "/messages", body).MustStatus(t, 202).JSON(t, &res)
		if err := env.App.Pool.QueryRow(ctx, `SELECT m.language, m.contact_language, m.language_reason, t.subject
			FROM messages m JOIN transactional_messages t ON t.message_id = m.id WHERE m.id = $1`, res.ID).Scan(&lang, &contact, &reason, &subject); err != nil {
			t.Fatal(err)
		}
		return
	}
	if l, c, r, s := send("someone@example.com", "es"); l != "es" || c != "es" || r != "match" || s != spanish {
		t.Fatalf("language es: %s %s %s %q", l, c, r, s)
	}
	if l, c, r, s := send("someone@example.com", "pt-BR"); l != "en" || c != "pt-BR" || r != "requested" || s != source.Subject {
		t.Fatalf("pt-BR asked for, English sent: %s %s %s %q", l, c, r, s)
	}
	var spanishContact string
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.email FROM contacts c WHERE c.workspace_id = $1 AND c.language = 'es-MX' AND c.erased_at IS NULL
		AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = lower(c.email)) ORDER BY c.email LIMIT 1`, wsID).
		Scan(&spanishContact); err != nil {
		t.Fatal(err)
	}
	if l, c, r, _ := send(spanishContact, ""); l != "es" || c != "es-MX" || r != "match" {
		t.Fatalf("from the contact: %s %s %s", l, c, r)
	}
	if l, _, r, _ := send("nobody@example.com", ""); l != "en" || r != "no_language" {
		t.Fatalf("no contact, no language: %s %s", l, r)
	}
	var rendered struct{ Subject, Language string }
	txnCall(t, env, key.Secret, "/templates/password-reset/render", map[string]any{"data": data, "language": "es"}).MustStatus(t, 200).JSON(t, &rendered)
	if rendered.Language != "es" || rendered.Subject != spanish {
		t.Fatalf("render in Spanish: %+v", rendered)
	}

	// Removing Spanish from the draft leaves the published version's.
	linh.Do("DELETE", tpl+"/languages/es", nil).MustStatus(t, 200).JSON(t, &langs)
	if langs.version(t, "es").Status != "missing" || logged("transactional_template.language.remove") != 1 {
		t.Fatalf("Spanish removed from the draft: %+v", langs.version(t, "es"))
	}
	if l, _, _, s := send("someone@example.com", "es"); l != "es" || s != spanish {
		t.Fatalf("v1 still sends Spanish: %s %q", l, s)
	}

	// Archived templates can't change.
	linh.Do("POST", tpl+"/archive", nil).MustStatus(t, 200)
	linh.Do("POST", tpl+"/languages", map[string]any{"language": "de"}).MustStatus(t, 409)
}
