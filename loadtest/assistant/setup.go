package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// plan is what the clients need: the threads, each with its owner's
// session and the stream's starting point. Setup writes it; it holds
// session cookies, so it is written readable by its owner only.
type plan struct {
	// Base is the instance as setup reached it; the clients may reach it
	// at another address (-base).
	Base    string `json:"base"`
	Streams int    `json:"streams"`
	// Created is when setup finished, by the database's clock.
	Created time.Time    `json:"created"`
	Threads []planThread `json:"threads"`
}

type planThread struct {
	Workspace string    `json:"workspace"`
	ID        uuid.UUID `json:"id"`
	Seq       int64     `json:"seq"`
	Cookie    string    `json:"cookie"`
}

// runSetup is the setup role: it writes the plan to -plan.
func runSetup(ctx context.Context, o options) error {
	if o.plan == "" {
		return errors.New("set -plan, the file to write")
	}
	pool, err := pgxpool.New(ctx, o.db)
	if err != nil {
		return err
	}
	defer pool.Close()
	p, err := setupPlan(ctx, pool, o)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(o.plan, raw, 0o600); err != nil {
		return err
	}
	log.Printf("wrote %s: %d threads", o.plan, len(p.Threads))
	return nil
}

func readPlan(path string) (*plan, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // G304: the operator's own file
	if err != nil {
		return nil, err
	}
	var p plan
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(p.Threads) == 0 {
		return nil, fmt.Errorf("%s has no threads", path)
	}
	return &p, nil
}

// setupPlan connects each workspace's model to the fake, adds people with
// sessions and opens o.streams threads, ten a person at most (the
// assistant's limit of turns at once a person).
func setupPlan(ctx context.Context, pool *pgxpool.Pool, o options) (*plan, error) {
	people := (o.streams + 9) / 10
	perWS := (people + len(o.workspaces) - 1) / len(o.workspaces)
	log.Printf("setup: %d streams, %d members over %d workspaces, the model at %s", o.streams, people, len(o.workspaces), o.fakeURL())
	var all []*person
	for _, slug := range o.workspaces {
		ps, err := setup(ctx, pool, o, slug, perWS)
		if err != nil {
			return nil, fmt.Errorf("setup %s: %w", slug, err)
		}
		all = append(all, ps...)
	}
	p := &plan{Base: o.base, Streams: o.streams}
	for i := 0; len(p.Threads) < o.streams; i++ {
		who := all[i%len(all)]
		th, err := who.newThread(ctx)
		if err != nil {
			return nil, err
		}
		p.Threads = append(p.Threads, planThread{Workspace: who.slug, ID: th.id, Seq: th.seq, Cookie: who.cookie})
	}
	if err := pool.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&p.Created); err != nil {
		return nil, err
	}
	return p, nil
}

// threads are the plan's threads, reached at base.
func (p *plan) threads(base string) []*thread {
	out := make([]*thread, len(p.Threads))
	for i, t := range p.Threads {
		who := &person{base: base, slug: t.Workspace, cookie: t.Cookie, client: httpClient}
		out[i] = &thread{p: who, id: t.ID, seq: t.Seq, got: map[int64]arrival{}, ends: make(chan string, 16)}
	}
	return out
}

// person is a member with a session.
type person struct {
	base, slug, cookie string
	client             *http.Client
}

func (p *person) do(ctx context.Context, method, path string, body, out any) error {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.base+"/api/v1"+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", p.cookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	res, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close() //nolint:errcheck
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %d %s", method, path, res.StatusCode, raw)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// session gives a user a session, as loadtest/session.sh does.
func session(ctx context.Context, pool *pgxpool.Pool, user uuid.UUID) (string, error) {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	_, err := pool.Exec(ctx, `INSERT INTO sessions (id, user_id, token_hash, auth_method, mfa_verified, created_at, last_seen_at, expires_at)
		VALUES (gen_random_uuid(), $1, $2, 'password', true, now(), now(), now() + interval '1 day')`, user, sum[:])
	return "rc_session=" + token, err
}

var httpClient = &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 1000, MaxConnsPerHost: 0}}

// setup connects a workspace's model to the fake provider, turns the
// assistant on with room for every turn, and adds n people.
func setup(ctx context.Context, pool *pgxpool.Pool, o options, slug string, n int) ([]*person, error) {
	var wsID, owner uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT w.id, m.user_id FROM workspaces w JOIN workspace_members m ON m.workspace_id = w.id JOIN roles r ON r.id = m.role_id
		WHERE w.slug = $1 AND r.key = 'owner' LIMIT 1`, slug).Scan(&wsID, &owner); err != nil {
		return nil, err
	}
	cookie, err := session(ctx, pool, owner)
	if err != nil {
		return nil, err
	}
	admin := &person{base: o.base, slug: slug, cookie: cookie, client: httpClient}
	base := "/workspaces/" + slug + "/ai"
	_ = admin.do(ctx, http.MethodDelete, base+"/connection", nil, nil)
	var s struct {
		Accounts []struct {
			ID     uuid.UUID `json:"id"`
			Models []struct {
				ID uuid.UUID `json:"id"`
			} `json:"models"`
		} `json:"accounts"`
		Assistant struct {
			Revision int64 `json:"revision"`
		} `json:"assistant"`
	}
	price := 0.0
	if err := admin.do(ctx, http.MethodPut, base+"/connection", map[string]any{"provider": "compatible", "baseUrl": o.fakeURL(), "model": "load-model",
		"apiKey": "load-test-key", "capKind": "none", "inputPrice": price, "outputPrice": price}, &s); err != nil {
		return nil, err
	}
	if len(s.Accounts) == 0 || len(s.Accounts[0].Models) == 0 {
		return nil, errors.New("no model after connecting")
	}
	model := base + "/accounts/" + s.Accounts[0].ID.String() + "/models/" + s.Accounts[0].Models[0].ID.String()
	if err := admin.do(ctx, http.MethodPost, model+"/test-tools", nil, nil); err != nil {
		return nil, err
	}
	if err := admin.do(ctx, http.MethodGet, base, nil, &s); err != nil {
		return nil, err
	}
	if err := admin.do(ctx, http.MethodPut, base+"/assistant/settings", map[string]any{"enabled": true, "limits": map[string]any{"toolCalls": 40, "tokens": 2000000, "costMicros": 100000000},
		"perPerson": 10, "perWorkspace": 100, "revision": s.Assistant.Revision}, nil); err != nil {
		return nil, err
	}
	var out []*person
	for i := range n {
		email := fmt.Sprintf("load-%s-%03d@load.test", slug, i)
		var user uuid.UUID
		if err := pool.QueryRow(ctx, `INSERT INTO users (id, email, name) VALUES (gen_random_uuid(), $1, $2)
			ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name RETURNING id`, email, fmt.Sprintf("Load %d", i)).Scan(&user); err != nil {
			return nil, err
		}
		if _, err := pool.Exec(ctx, `INSERT INTO workspace_members (id, workspace_id, user_id, role_id, scope, source)
			SELECT gen_random_uuid(), $1, $2, r.id, '{}', 'invite' FROM roles r WHERE r.workspace_id = $1 AND r.key = 'editor'
			ON CONFLICT DO NOTHING`, wsID, user); err != nil {
			return nil, err
		}
		cookie, err := session(ctx, pool, user)
		if err != nil {
			return nil, err
		}
		out = append(out, &person{base: o.base, slug: slug, cookie: cookie, client: httpClient})
	}
	return out, nil
}

func (p *person) newThread(ctx context.Context) (*thread, error) {
	var th struct {
		ID       uuid.UUID `json:"id"`
		EventSeq int64     `json:"eventSeq"`
	}
	if err := p.do(ctx, http.MethodPost, "/workspaces/"+p.slug+"/ai/assistant/threads", map[string]any{"requestKey": strings.ReplaceAll(uuid.NewString(), "-", "")}, &th); err != nil {
		return nil, err
	}
	return &thread{p: p, id: th.ID, seq: th.EventSeq, got: map[int64]arrival{}, ends: make(chan string, 16)}, nil
}
