// Command assistant is the AI assistant's load test (B6): it opens many
// thread streams (SSE) on a running ReputeCast, sends a message in each
// thread, answers the model requests itself as a fake OpenAI-compatible
// provider that streams, and measures each event from its commit to its
// arrival here.
//
// It is a small Go program, not a k6 script, for two reasons. k6 has no
// SSE client of its own: the extensions (xk6-sse and its forks) need a
// custom k6 binary built with xk6, a third-party build of a third-party
// extension that the CI and the orb would have to fetch and trust. And the
// measure the gate asks for starts at an event's commit, which only the
// database knows (each event's row keeps its created_at, written in its
// committing transaction): this program reads those after the run and
// pairs them with what it timed on arrival, on the same machine's clock.
// It lives under loadtest/ and is not part of the shipped binary.
//
//	REPUTECAST_DATABASE__URL=postgres://… go run ./loadtest/assistant \
//	  -base http://127.0.0.1:8080 -fake 127.0.0.1:9711 -streams 200
//
// The instance needs the Acme sample (reputecast seed), the provider's
// address allowed (providers.allow_private_networks for a loopback fake)
// and enough assistant workers for every turn at once
// (worker.concurrency 400 gives the ai_assistant queue 200). Setup, done
// each run, writes to the database directly, like loadtest/session.sh:
// development and load test instances only. It connects every workspace's
// model to the fake, sets the assistant's limits to 10 turns a person and
// 100 a workspace (the most they take), adds people as Editors and gives
// them sessions.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	mrand "math/rand/v2"
	"net"
	"net/http"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type options struct {
	base, fake, db, metricsURL, metricsToken string
	workspaces                               []string
	streams, rounds, chunks                  int
	every, timeout, spread                   time.Duration
	serverMatch                              string
}

func main() {
	var o options
	var ws string
	flag.StringVar(&o.base, "base", "http://127.0.0.1:8080", "the instance")
	flag.StringVar(&o.fake, "fake", "127.0.0.1:9711", "where the fake provider listens (the instance must reach it)")
	flag.StringVar(&o.db, "db", os.Getenv("REPUTECAST_DATABASE__URL"), "the instance's database (setup and the events' commit times)")
	flag.StringVar(&ws, "workspaces", "acme-newsletter,acme-engineering-blog", "the workspaces the streams are spread over (100 turns at once each, at most)")
	flag.IntVar(&o.streams, "streams", 200, "thread streams open at once")
	flag.IntVar(&o.rounds, "rounds", 3, "messages sent in each thread, one after the other's turn ended")
	flag.IntVar(&o.chunks, "chunks", 150, "pieces of each streamed answer")
	flag.DurationVar(&o.every, "every", 20*time.Millisecond, "time between the answer's pieces")
	flag.DurationVar(&o.timeout, "timeout", 5*time.Minute, "how long the run may take")
	flag.DurationVar(&o.spread, "spread", 0, "each message waits a random time up to this before it is sent (0: every thread at once)")
	flag.StringVar(&o.serverMatch, "server", "reputecast serve", "a part of the instance process's command line, for its CPU")
	flag.StringVar(&o.metricsURL, "metrics", "", "the instance's /metrics, for the server-side histograms (optional)")
	flag.StringVar(&o.metricsToken, "metrics-token", "", "the metrics bearer token")
	flag.Parse()
	o.workspaces = strings.Split(ws, ",")
	if o.db == "" {
		log.Fatal("set -db or REPUTECAST_DATABASE__URL")
	}
	if err := run(context.Background(), o); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, o options) error {
	pool, err := pgxpool.New(ctx, o.db)
	if err != nil {
		return err
	}
	defer pool.Close()
	fake := &provider{chunks: o.chunks, every: o.every}
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", o.fake)
	if err != nil {
		return err
	}
	go func() { _ = http.Serve(ln, fake) }() //nolint:gosec // a load test's local fake provider
	people := (o.streams + 9) / 10
	perWS := (people + len(o.workspaces) - 1) / len(o.workspaces)
	log.Printf("setup: %d streams, %d people over %d workspaces", o.streams, people, len(o.workspaces))
	var all []*person
	for _, slug := range o.workspaces {
		ps, err := setup(ctx, pool, o, slug, perWS)
		if err != nil {
			return fmt.Errorf("setup %s: %w", slug, err)
		}
		all = append(all, ps...)
	}
	// Threads: ten a person at most (the assistant's limit of turns at
	// once a person).
	var threads []*thread
	for i := 0; len(threads) < o.streams; i++ {
		p := all[i%len(all)]
		th, err := p.newThread(ctx)
		if err != nil {
			return err
		}
		threads = append(threads, th)
	}
	log.Printf("%d threads; opening their streams", len(threads))
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	var connected sync.WaitGroup
	for _, th := range threads {
		connected.Add(1)
		go th.stream(ctx, &connected)
	}
	connected.Wait()
	log.Printf("streams open; sending %d rounds", o.rounds)
	cpu0, at0 := cpuSample(o.serverMatch), time.Now()
	before, _ := scrape(o)
	var wg sync.WaitGroup
	for _, th := range threads {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := range o.rounds {
				if o.spread > 0 {
					time.Sleep(time.Duration(mrand.Int64N(int64(o.spread)))) //nolint:gosec // G404: a load test's jitter, not a secret
				}
				if err := th.turn(ctx, r); err != nil {
					th.err(err)
					return
				}
			}
		}()
	}
	wg.Wait()
	took := time.Since(at0)
	cpu1 := cpuSample(o.serverMatch)
	after, _ := scrape(o)
	cancel()
	return report(context.Background(), pool, o, threads, fake, took, cpu0, cpu1, before, after)
}

// provider is an OpenAI-compatible model: it passes the tool-calling
// check and streams every answer as pieces at a steady pace.
type provider struct {
	chunks   int
	every    time.Duration
	requests atomic.Int64
	streams  atomic.Int64
}

func (p *provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.requests.Add(1)
	var req struct {
		Stream bool `json:"stream"`
		Tools  []struct {
			Function struct{ Name string } `json:"function"`
		} `json:"tools"`
	}
	raw, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(raw, &req)
	if !req.Stream {
		message := map[string]any{"role": "assistant", "content": "Ready."}
		stop := "stop"
		if len(req.Tools) > 0 && req.Tools[0].Function.Name == "check_tools" {
			message["content"] = nil
			message["tool_calls"] = []any{map[string]any{"id": "call1", "type": "function", "function": map[string]any{"name": "check_tools", "arguments": `{"value":"ready"}`}}}
			stop = "tool_calls"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}},
			"usage": map[string]int{"prompt_tokens": 9, "completion_tokens": 2}})
		return
	}
	p.streams.Add(1)
	w.Header().Set("Content-Type", "text/event-stream")
	f, _ := w.(http.Flusher)
	data := func(v any) {
		b, _ := json.Marshal(v)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		f.Flush()
	}
	tick := time.NewTicker(p.every)
	defer tick.Stop()
	for i := range p.chunks {
		data(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": "word" + strconv.Itoa(i) + " "}}}})
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
		}
	}
	data(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"}}})
	data(map[string]any{"choices": []any{}, "usage": map[string]int{"prompt_tokens": 900, "completion_tokens": p.chunks}})
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	f.Flush()
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
	if err := admin.do(ctx, http.MethodPut, base+"/connection", map[string]any{"provider": "compatible", "baseUrl": "http://" + o.fake + "/v1", "model": "load-model",
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

// thread is a thread and what its stream received.
type thread struct {
	p    *person
	id   uuid.UUID
	seq  int64
	mu   sync.Mutex
	got  map[int64]arrival
	ends chan string // a turn ended: its state
	sent []time.Time // each round's message, when it was sent
	errs []error
}

type arrival struct {
	kind string
	at   time.Time
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

func (t *thread) err(err error) {
	t.mu.Lock()
	t.errs = append(t.errs, err)
	t.mu.Unlock()
}

// stream reads the thread's events until ctx ends, reconnecting from its
// last event when the server cuts it.
func (t *thread) stream(ctx context.Context, connected *sync.WaitGroup) {
	var once sync.Once
	last := t.seq
	for ctx.Err() == nil {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, t.p.base+"/api/v1/workspaces/"+t.p.slug+"/ai/assistant/threads/"+t.id.String()+"/stream", nil)
		req.Header.Set("Cookie", t.p.cookie)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Last-Event-ID", strconv.FormatInt(last, 10))
		res, err := t.p.client.Do(req)
		if err != nil {
			if ctx.Err() == nil {
				t.err(err)
				time.Sleep(200 * time.Millisecond)
			}
			continue
		}
		r := bufio.NewReaderSize(res.Body, 1<<20)
		var id, event string
		var data []byte
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				break
			}
			line = strings.TrimRight(line, "\r\n")
			switch {
			case strings.HasPrefix(line, ": connected"):
				once.Do(connected.Done)
			case line == "":
				if id != "" {
					seq, _ := strconv.ParseInt(id, 10, 64)
					now := time.Now()
					t.mu.Lock()
					if _, dup := t.got[seq]; dup {
						t.errs = append(t.errs, fmt.Errorf("event %d twice", seq))
					}
					t.got[seq] = arrival{kind: event, at: now}
					t.mu.Unlock()
					last = seq
					if event == "turn" {
						var v struct {
							Turn struct{ State, Reason string } `json:"turn"`
						}
						_ = json.Unmarshal(data, &v)
						switch v.Turn.State {
						case "done", "failed", "stopped":
							t.ends <- v.Turn.State + ":" + v.Turn.Reason
						}
					}
				}
				id, event, data = "", "", nil
			case strings.HasPrefix(line, "id: "):
				id = line[4:]
			case strings.HasPrefix(line, "event: "):
				event = line[7:]
			case strings.HasPrefix(line, "data: "):
				data = append(data, line[6:]...)
			}
		}
		_ = res.Body.Close()
	}
	once.Do(connected.Done)
}

// turn sends round r's message and waits for its turn to end.
func (t *thread) turn(ctx context.Context, r int) error {
	t.mu.Lock()
	t.sent = append(t.sent, time.Now())
	t.mu.Unlock()
	for {
		err := t.p.do(ctx, http.MethodPost, "/workspaces/"+t.p.slug+"/ai/assistant/threads/"+t.id.String()+"/messages",
			map[string]any{"text": "Load test question " + strconv.Itoa(r), "requestKey": strings.ReplaceAll(uuid.NewString(), "-", "")}, nil)
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "429") || ctx.Err() != nil {
			return err
		}
		time.Sleep(time.Second)
	}
	select {
	case end := <-t.ends:
		if !strings.HasPrefix(end, "done") {
			return fmt.Errorf("turn ended %s", end)
		}
		return nil
	case <-ctx.Done():
		return errors.New("timed out waiting for the turn")
	}
}

// report pairs arrivals with the events' commit times and prints the
// distribution, the CPU used and the server's own histograms.
func report(ctx context.Context, pool *pgxpool.Pool, o options, threads []*thread, fake *provider, took time.Duration, cpu0, cpu1 cpu, before, after map[string]float64) error {
	ids := make([]uuid.UUID, len(threads))
	byID := map[uuid.UUID]*thread{}
	for i, t := range threads {
		ids[i], byID[t.id] = t.id, t
	}
	rows, err := pool.Query(ctx, `SELECT thread_id, seq, kind, created_at FROM ai_assistant_events WHERE thread_id = ANY($1) ORDER BY thread_id, seq`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	var deltas, all []float64
	missing := 0
	ttft := []float64{}
	firstDelta := map[uuid.UUID][]time.Time{}
	for rows.Next() {
		var id uuid.UUID
		var seq int64
		var kind string
		var at time.Time
		if err := rows.Scan(&id, &seq, &kind, &at); err != nil {
			return err
		}
		t := byID[id]
		if seq <= t.seq {
			continue
		}
		t.mu.Lock()
		a, ok := t.got[seq]
		t.mu.Unlock()
		if !ok {
			missing++
			continue
		}
		lag := a.at.Sub(at).Seconds() * 1000
		all = append(all, lag)
		if kind == "delta" {
			deltas = append(deltas, lag)
			firstDelta[id] = append(firstDelta[id], a.at)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	errs, turns := 0, 0
	for _, t := range threads {
		errs += len(t.errs)
		for _, e := range t.errs {
			if errs <= 5 {
				log.Printf("thread %s: %v", t.id, e)
			}
		}
		turns += len(t.sent)
		// The client's time to first token: from sending each message to
		// the first delta after it.
		for _, sent := range t.sent {
			i := slices.IndexFunc(firstDelta[t.id], func(at time.Time) bool { return at.After(sent) })
			if i >= 0 {
				ttft = append(ttft, firstDelta[t.id][i].Sub(sent).Seconds()*1000)
			}
		}
	}
	cpus, model := machine()
	fmt.Printf("\nmachine: %d CPUs (%s), Go %s, %s\n", cpus, model, runtime.Version(), time.Now().UTC().Format(time.RFC3339))
	fmt.Printf("run: %d streams, %d turns (%d rounds, each message up to %s after the last turn), %d model streams answered by the fake (%d pieces every %s), %s\n",
		len(threads), turns, o.rounds, o.spread, fake.streams.Load(), o.chunks, o.every, took.Round(time.Millisecond))
	fmt.Printf("errors: %d; events not received: %d\n", errs, missing)
	fmt.Printf("delta events, commit to client (ms): %s\n", quantiles(deltas))
	fmt.Printf("all events, commit to client (ms):   %s\n", quantiles(all))
	fmt.Printf("time to first token, message sent to its first delta (ms): %s\n", quantiles(ttft))
	secs := took.Seconds()
	fmt.Printf("CPU: server %.1f s (%.2f cores), Postgres %.1f s (%.2f cores), over %.1f s\n", cpu1.server-cpu0.server, (cpu1.server-cpu0.server)/secs,
		cpu1.postgres-cpu0.postgres, (cpu1.postgres-cpu0.postgres)/secs, secs)
	if before != nil && after != nil {
		for _, h := range []string{"reputecast_ai_assistant_event_lag_seconds{kind=\"delta\"}", "reputecast_ai_assistant_ttft_overhead_seconds"} {
			fmt.Printf("server %s: %s\n", h, bucketQuantiles(before, after, h))
		}
	}
	if missing > 0 || errs > 0 {
		return fmt.Errorf("%d events not received, %d errors", missing, errs)
	}
	return nil
}

func quantiles(v []float64) string {
	if len(v) == 0 {
		return "none"
	}
	slices.Sort(v)
	q := func(p float64) float64 { return v[min(len(v)-1, int(math.Ceil(p*float64(len(v))))-1)] }
	return fmt.Sprintf("n=%d p50=%.1f p95=%.1f p99=%.1f max=%.1f", len(v), q(.50), q(.95), q(.99), v[len(v)-1])
}

type cpu struct{ server, postgres float64 }

// cpuSample is the CPU time (user and system) of the instance's processes
// and Postgres's, in seconds.
func cpuSample(match string) cpu {
	var c cpu
	dirs, _ := os.ReadDir("/proc")
	for _, d := range dirs {
		if _, err := strconv.Atoi(d.Name()); err != nil {
			continue
		}
		cmd, _ := os.ReadFile("/proc/" + d.Name() + "/cmdline")
		stat, err := os.ReadFile("/proc/" + d.Name() + "/stat")
		if err != nil {
			continue
		}
		// The fields after the command's closing parenthesis: utime and
		// stime are the 12th and 13th.
		end := bytes.LastIndexByte(stat, ')')
		f := strings.Fields(string(stat[end+2:]))
		if len(f) < 13 {
			continue
		}
		ut, _ := strconv.ParseFloat(f[11], 64)
		st, _ := strconv.ParseFloat(f[12], 64)
		// Backends that ended count in their parent's children's time.
		cut, _ := strconv.ParseFloat(f[13], 64)
		cst, _ := strconv.ParseFloat(f[14], 64)
		secs := (ut + st + cut + cst) / 100 // USER_HZ
		line := strings.ReplaceAll(string(cmd), "\x00", " ")
		switch {
		case strings.Contains(line, match):
			c.server += secs
		case strings.HasPrefix(line, "postgres") || strings.Contains(string(stat[:end]), "(postgres"):
			c.postgres += secs
		}
	}
	return c
}

func machine() (int, string) {
	raw, _ := os.ReadFile("/proc/cpuinfo")
	model := "unknown CPU"
	for line := range strings.SplitSeq(string(raw), "\n") {
		if strings.HasPrefix(line, "model name") {
			_, model, _ = strings.Cut(line, ": ")
			break
		}
	}
	return runtime.NumCPU(), model
}

// scrape reads the instance's histogram buckets (cumulative counts by
// metric and upper bound).
func scrape(o options) (map[string]float64, error) {
	if o.metricsURL == "" {
		return nil, nil
	}
	req, _ := http.NewRequest(http.MethodGet, o.metricsURL, nil) //nolint:noctx // a one-off scrape
	if o.metricsToken != "" {
		req.Header.Set("Authorization", "Bearer "+o.metricsToken)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close() //nolint:errcheck
	out := map[string]float64{}
	sc := bufio.NewScanner(res.Body)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, "_bucket{") {
			continue
		}
		key, val, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		v, err := strconv.ParseFloat(val, 64)
		if err == nil {
			out[key] = v
		}
	}
	return out, sc.Err()
}

// bucketQuantiles estimates p50 and p95 of a histogram over the run from
// its buckets' counts before and after.
func bucketQuantiles(before, after map[string]float64, name string) string {
	metric, labels, _ := strings.Cut(name, "{")
	labels = strings.TrimSuffix(labels, "}")
	type bucket struct{ le, n float64 }
	var bs []bucket
	for k, v := range after {
		if !strings.HasPrefix(k, metric+"_bucket{") || (labels != "" && !strings.Contains(k, labels)) {
			continue
		}
		i := strings.Index(k, `le="`)
		le, err := strconv.ParseFloat(strings.TrimSuffix(k[i+4:], `"}`), 64)
		if err != nil {
			le = math.Inf(1)
		}
		bs = append(bs, bucket{le, v - before[k]})
	}
	slices.SortFunc(bs, func(a, b bucket) int { return int(math.Copysign(1, a.le-b.le)) })
	if len(bs) == 0 || bs[len(bs)-1].n == 0 {
		return "no samples"
	}
	total := bs[len(bs)-1].n
	at := func(p float64) string {
		for _, b := range bs {
			if b.n >= p*total {
				return "≤" + strconv.FormatFloat(b.le*1000, 'f', -1, 64) + " ms"
			}
		}
		return "?"
	}
	return fmt.Sprintf("n=%.0f p50 %s, p95 %s", total, at(.5), at(.95))
}
