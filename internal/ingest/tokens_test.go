package ingest_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/providers/mailgun"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// Mailgun's one-time tokens are tied to their event in the database every
// process shares: of requests with the same token at once one ties it, the
// tie holds while the signature can be accepted, and the sweep drops it
// after.
func TestTokensTieOnce(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	tokens := ingest.Tokens{Q: env.App.Q}
	now := time.Now().UTC().Truncate(time.Second)
	tok := sha256.Sum256([]byte("mailgun\x00tok-1"))
	payloads := [][32]byte{sha256.Sum256([]byte(`{"event":"delivered"}`)), sha256.Sum256([]byte(`{"event":"failed"}`))}
	got := make([][32]byte, 8)
	var wg sync.WaitGroup
	for i := range got {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := tokens.BindToken(ctx, "mailgun", tok, payloads[i%2], now, now.Add(12*time.Hour))
			if err != nil {
				t.Error(err)
			}
			got[i] = b
		}()
	}
	wg.Wait()
	tied := got[0]
	for _, b := range got {
		if b != tied || (b != payloads[0] && b != payloads[1]) {
			t.Fatalf("the token was tied to two events: %x", got)
		}
	}
	other := payloads[0]
	if tied == other {
		other = payloads[1]
	}
	// While it's current, the tie holds.
	if b, err := tokens.BindToken(ctx, "mailgun", tok, other, now.Add(11*time.Hour), now.Add(23*time.Hour)); err != nil || b != tied {
		t.Fatalf("while current: %x, %v", b, err)
	}
	// Another adapter's token is another token.
	if b, err := tokens.BindToken(ctx, "other", tok, other, now, now.Add(time.Hour)); err != nil || b != other {
		t.Fatalf("another scope: %x, %v", b, err)
	}
	// Past its time the token is tied anew: its signature is refused by then.
	if b, err := tokens.BindToken(ctx, "mailgun", tok, other, now.Add(13*time.Hour), now.Add(25*time.Hour)); err != nil || b != other {
		t.Fatalf("past its time: %x, %v", b, err)
	}
	// The sweep drops the ties past their time.
	if n, err := env.App.Q.IngestPurgeTokens(ctx, now.Add(2*time.Hour)); err != nil || n != 1 {
		t.Fatalf("swept %d, %v", n, err)
	}
	var left int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_tokens`).Scan(&left); err != nil || left != 1 {
		t.Fatalf("left %d, %v", left, err)
	}
}

// A Mailgun delivery's token is tied where every process sees it: one
// another process tied to another event is refused here too.
func TestMailgunTokensAreShared(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "owner@mgtokens.test", Name: "Pat Owner"})
	ws := env.CreateWorkspace("Mailgun Tokens Test", u.ID)
	pid := uuid.Must(uuid.NewV7())
	const key, hook = "key-signing-0001", "token-mgtokens-000001"
	creds, err := env.App.Secrets.Seal([]byte(`{"api_key":"key-api-0001","webhook_signing_key":"`+key+`"}`), manage.CredentialsAD(store.Provider{ID: pid}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, ip_mode, ips,
		capabilities, limits, health, pricing, connected_at)
		VALUES ($1, $2, 'mailgun', 'Mailgun', '', '{"domain":"mg.acme.test"}', $3, $4, 'healthy', 'shared', '{}', '{}', '{}', '{}', '{}', now())`,
		pid, ws.ID, creds, hook); err != nil {
		t.Fatal(err)
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	post := func(tok string) int {
		t.Helper()
		body := `{"signature":{"timestamp":"` + ts + `","token":"` + tok + `","signature":"` + mailgun.Sign(key, ts, tok) + `"},` +
			`"event-data":{"event":"delivered","id":"ev-` + tok + `","timestamp":` + ts + `,"recipient":"ana@example.com"}}`
		resp, err := http.Post(env.Server.URL+"/hooks/mailgun/"+hook, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if st := post("tok-0001"); st != http.StatusOK {
		t.Fatalf("a signed delivery: %d", st)
	}
	var tied int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_tokens WHERE scope = 'mailgun' AND expires_at > now() + interval '12 hours'`).Scan(&tied); err != nil || tied != 1 {
		t.Fatalf("tokens tied: %d, %v", tied, err)
	}
	// Another process (or this one before a restart) tied tok-0002 to
	// another event.
	tok := sha256.Sum256([]byte("mailgun\x00tok-0002"))
	var elsewhere [32]byte
	_, _ = rand.Read(elsewhere[:])
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO webhook_tokens (scope, token_hash, payload_hash, expires_at) VALUES ('mailgun', $1, $2, now() + interval '1 hour')`,
		tok[:], elsewhere[:]); err != nil {
		t.Fatal(err)
	}
	if st := post("tok-0002"); st != http.StatusUnauthorized {
		t.Fatalf("a token tied to another event elsewhere: %d", st)
	}
}
