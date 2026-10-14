package forms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// SiteverifyURL is Cloudflare Turnstile's server-side verification endpoint
// (a variable so tests can point it at a fake).
var SiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// TurnstileScript is the widget script a challenge page loads (only when a
// submission looked automated; the pages load no third-party code otherwise).
const TurnstileScript = "https://challenges.cloudflare.com/turnstile/v0/api.js"

// TurnstileOrigin is allowed in the challenge page's CSP.
const TurnstileOrigin = "https://challenges.cloudflare.com"

// Turnstile verifies Cloudflare Turnstile answers server-side. Client is an
// SSRF-guarded client (internal/netguard).
type Turnstile struct {
	Client    *http.Client
	VerifyURL string
}

// TurnstileResult is Cloudflare's answer.
type TurnstileResult struct {
	Success  bool     `json:"success"`
	Codes    []string `json:"error-codes"`
	Hostname string   `json:"hostname"`
	Action   string   `json:"action"`
}

// Verify checks a widget answer with the secret key.
func (t *Turnstile) Verify(ctx context.Context, secret, response, remoteIP, idempotencyKey string) (TurnstileResult, error) {
	if t == nil || t.Client == nil {
		return TurnstileResult{}, errors.New("turnstile: not configured")
	}
	endpoint := t.VerifyURL
	if endpoint == "" {
		endpoint = SiteverifyURL
	}
	form := url.Values{"secret": {secret}, "response": {response}}
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}
	if idempotencyKey != "" {
		form.Set("idempotency_key", idempotencyKey)
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return TurnstileResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.Client.Do(req)
	if err != nil {
		return TurnstileResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return TurnstileResult{}, err
	}
	var out TurnstileResult
	if err := json.Unmarshal(body, &out); err != nil {
		return TurnstileResult{}, fmt.Errorf("turnstile: unexpected answer (HTTP %d)", resp.StatusCode)
	}
	return out, nil
}

// SecretRejected reports whether Cloudflare refused the secret key itself.
func (r TurnstileResult) SecretRejected() bool {
	return slices.Contains(r.Codes, "invalid-input-secret") || slices.Contains(r.Codes, "missing-input-secret")
}

// turnstileErrorWords explains widget error codes the challenge page reports.
func turnstileErrorWords(code string) string {
	switch {
	case strings.HasPrefix(code, "1102"):
		return fmt.Sprintf("Cloudflare rejected the site key (error %s)", code)
	case strings.HasPrefix(code, "1101"):
		return fmt.Sprintf("Cloudflare doesn't know this site key (error %s)", code)
	case strings.HasPrefix(code, "1105"), strings.HasPrefix(code, "1106"):
		return fmt.Sprintf("The Turnstile widget couldn't load (error %s)", code)
	}
	return fmt.Sprintf("Turnstile reported error %s", code)
}

// PublicTurnstileInput saves the Turnstile keys. An empty secret keeps the
// stored one (secrets are write-only and never shown again).
type PublicTurnstileInput struct {
	SiteKey string `json:"siteKey" minLength:"1" maxLength:"100" example:"0x4AAAAAAA…"`
	Secret  string `json:"secret,omitempty" maxLength:"200" doc:"Write-only; omit to keep the stored secret"`
}

// SaveTurnstile stores the keys and checks the secret with Cloudflare.
func (s *Service) SaveTurnstile(ctx context.Context, ws *store.Workspace, in PublicTurnstileInput) (PublicPagesSettings, error) {
	site := strings.TrimSpace(in.SiteKey)
	if site == "" || strings.ContainsAny(site, " \t\"'<>") {
		return PublicPagesSettings{}, api.Invalid("body.siteKey", "Paste the site key from the Cloudflare dashboard (it starts with 0x).")
	}
	set, err := s.settings(ctx, ws)
	if err != nil {
		return PublicPagesSettings{}, err
	}
	secret := strings.TrimSpace(in.Secret)
	enc := set.row.TurnstileSecretEnc
	if secret == "" && len(enc) == 0 {
		return PublicPagesSettings{}, api.Invalid("body.secret", "Paste the secret key too; it's stored encrypted and never shown again.")
	}
	if secret != "" {
		if enc, err = s.Box.SealString(secret, "turnstile:"+ws.ID.String()+":secret"); err != nil {
			return PublicPagesSettings{}, err
		}
	} else if secret, err = s.Box.OpenString(enc, "turnstile:"+ws.ID.String()+":secret"); err != nil {
		return PublicPagesSettings{}, api.Invalid("body.secret", "The stored secret can't be read any more. Paste it again.")
	}
	status, problem := "ok", ""
	if s.Turnstile != nil {
		// A real secret answers "invalid-input-response" to a made-up token.
		res, err := s.Turnstile.Verify(ctx, secret, "reputecast-key-check", "", "")
		switch {
		case err != nil:
			s.log().WarnContext(ctx, "turnstile key check", "err", err)
		case res.SecretRejected():
			status, problem = "error", "Cloudflare rejected the secret key ("+strings.Join(res.Codes, ", ")+")"
		}
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PublicSettingsSetTurnstile(ctx, store.PublicSettingsSetTurnstileParams{WorkspaceID: ws.ID, SiteKey: site, SecretEnc: enc,
			Status: status, Error: problem, CheckedAt: &now, UpdatedBy: actor(ctx), At: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "public_pages.turnstile", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"siteKey": set.row.TurnstileSiteKey, "status": set.row.TurnstileStatus},
			After:  map[string]any{"siteKey": site, "status": status, "secretReplaced": in.Secret != ""}})
	})
	if err != nil {
		return PublicPagesSettings{}, err
	}
	return s.PublicPages(ctx, ws)
}

// RemoveTurnstile deletes the keys: suspicious submissions are then refused.
func (s *Service) RemoveTurnstile(ctx context.Context, ws *store.Workspace) (PublicPagesSettings, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.PublicSettingsEnsure(ctx, ws.ID); err != nil {
			return err
		}
		if err := q.PublicSettingsSetTurnstile(ctx, store.PublicSettingsSetTurnstileParams{WorkspaceID: ws.ID, Status: "off", UpdatedBy: actor(ctx), At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "public_pages.turnstile_remove", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name})
	})
	if err != nil {
		return PublicPagesSettings{}, err
	}
	return s.PublicPages(ctx, ws)
}

// ReportTurnstileError records an error the widget showed on a challenge
// page (a site key Cloudflare doesn't accept for this domain). Reports count
// only from the page of a submission still held for its check, so nobody can
// switch Turnstile off from outside.
func (s *Service) ReportTurnstileError(ctx context.Context, ws *store.Workspace, challengeID, code string) error {
	code = strings.TrimSpace(code)
	if len(code) < 3 || len(code) > 12 || strings.Trim(code, "0123456789") != "" {
		return nil
	}
	// Only configuration errors (bad key, domain not allowed) are the publisher's to fix.
	if !strings.HasPrefix(code, "110") {
		return nil
	}
	id, ok := parseID(challengeID)
	if !ok {
		return nil // not from a challenge page: ignored
	}
	live, err := s.Q.FormChallengeLive(ctx, store.FormChallengeLiveParams{ID: id, WorkspaceID: ws.ID, At: s.now()})
	if err != nil || !live {
		return err
	}
	return s.turnstileFailed(ctx, ws, turnstileErrorWords(code), map[string]any{"widgetError": code})
}

// turnstileFailed marks the Turnstile keys as failing (spam protection
// falls back to the honeypot and timing check) and writes it to the audit
// log as Cloudflare's doing, once per change of the message.
func (s *Service) turnstileFailed(ctx context.Context, ws *store.Workspace, problem string, meta map[string]any) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		before, err := q.PublicSettingsGet(ctx, ws.ID)
		if err != nil {
			return err
		}
		if err := q.PublicSettingsTurnstileError(ctx, store.PublicSettingsTurnstileErrorParams{WorkspaceID: ws.ID, Error: problem, At: ptrTime(s.now())}); err != nil {
			return err
		}
		if before.TurnstileStatus == "error" && before.TurnstileError == problem {
			return nil // already recorded
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "public_pages.turnstile_error", TargetType: "workspace", TargetID: ws.ID.String(),
			TargetName: ws.Name, Actor: &audit.Actor{Type: authz.ActorSystem, Name: "Cloudflare Turnstile"},
			Before: map[string]any{"status": before.TurnstileStatus}, After: map[string]any{"status": "error", "error": problem}, Metadata: meta})
	})
}

// parseID reads a uuid from a form field.
func parseID(s string) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(s))
	return id, err == nil
}

// turnstileSecret opens the stored secret.
func (s *Service) turnstileSecret(set *Settings) (string, bool) {
	if set.TurnstileSiteKey() == "" || s.Box == nil {
		return "", false
	}
	secret, err := s.Box.OpenString(set.row.TurnstileSecretEnc, "turnstile:"+set.Workspace.ID.String()+":secret")
	return secret, err == nil && secret != ""
}
