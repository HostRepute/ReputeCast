package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	mrand "math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// thread is a thread and what its stream received, by this machine's
// clock.
type thread struct {
	p    *person
	id   uuid.UUID
	seq  int64
	mu   sync.Mutex
	got  map[int64]arrival
	ends chan string // a turn ended: its state
	sent []time.Time // each round's message, when it was sent
	post []float64   // each round's message, how long its POST took (ms)
	errs []error
}

type arrival struct {
	kind string
	at   time.Time
}

func (t *thread) err(err error) {
	t.mu.Lock()
	t.errs = append(t.errs, err)
	t.mu.Unlock()
}

// refresh starts the thread's stream at its last event now, so a plan
// can be run again.
func (t *thread) refresh(ctx context.Context) error {
	var th struct {
		EventSeq int64 `json:"eventSeq"`
	}
	if err := t.p.do(ctx, http.MethodGet, "/workspaces/"+t.p.slug+"/ai/assistant/threads/"+t.id.String(), nil, &th); err != nil {
		return err
	}
	t.seq = th.EventSeq
	return nil
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
		if res.StatusCode != http.StatusOK {
			t.err(fmt.Errorf("stream: %d", res.StatusCode))
			_ = res.Body.Close()
			time.Sleep(time.Second)
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
	sent := time.Now()
	t.mu.Lock()
	t.sent = append(t.sent, sent)
	t.mu.Unlock()
	for {
		at := time.Now()
		err := t.p.do(ctx, http.MethodPost, "/workspaces/"+t.p.slug+"/ai/assistant/threads/"+t.id.String()+"/messages",
			map[string]any{"text": "Load test question " + strconv.Itoa(r), "requestKey": strings.ReplaceAll(uuid.NewString(), "-", "")}, nil)
		if err == nil {
			t.mu.Lock()
			t.post = append(t.post, time.Since(at).Seconds()*1000)
			t.mu.Unlock()
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

// result is a run as the clients saw it.
type result struct {
	took          time.Duration
	from, to      time.Time // the database's clock; zero without it
	before, after map[string]float64
	cpu0, cpu1    cpu
}

// drive opens every thread's stream, then sends o.rounds messages in each
// (each once the thread's previous turn ended), snapshotting the
// instance's metrics before the first message and after the last turn.
func drive(ctx context.Context, o options, pool *pgxpool.Pool, threads []*thread) (result, error) {
	var res result
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	var connected sync.WaitGroup
	for _, th := range threads {
		connected.Add(1)
		go th.stream(ctx, &connected)
	}
	connected.Wait()
	log.Printf("%d streams open; sending %d rounds", len(threads), o.rounds)
	var err error
	if res.before, err = scrape(o); err != nil {
		log.Printf("metrics before the run: %v", err)
	}
	if pool != nil {
		if err := pool.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&res.from); err != nil {
			return res, err
		}
	}
	res.cpu0 = cpuSample(o.serverMatch)
	at0 := time.Now()
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
	res.took = time.Since(at0)
	res.cpu1 = cpuSample(o.serverMatch)
	if pool != nil {
		if err := pool.QueryRow(context.Background(), `SELECT clock_timestamp()`).Scan(&res.to); err != nil {
			return res, err
		}
	}
	if res.after, err = scrape(o); err != nil {
		log.Printf("metrics after the run: %v", err)
	}
	return res, nil
}

// runClients is the clients role: the plan's threads (or setup's, with
// -db and no -plan), driven at -base, reported with what this machine's
// clock and the instance's metrics and database say.
func runClients(ctx context.Context, o options) error {
	var pool *pgxpool.Pool
	if o.db != "" {
		var err error
		if pool, err = pgxpool.New(ctx, o.db); err != nil {
			return err
		}
		defer pool.Close()
	}
	var p *plan
	switch {
	case o.plan != "":
		var err error
		if p, err = readPlan(o.plan); err != nil {
			return err
		}
	case pool != nil:
		var err error
		if p, err = setupPlan(ctx, pool, o); err != nil {
			return err
		}
	default:
		return errors.New("set -plan (setup's file) or -db (to set up here)")
	}
	raiseFileLimit(len(p.Threads)*2 + 256)
	threads := p.threads(o.base)
	for _, th := range threads {
		if err := th.refresh(ctx); err != nil {
			return fmt.Errorf("thread %s at %s: %w", th.id, o.base, err)
		}
	}
	res, err := drive(ctx, o, pool, threads)
	if err != nil {
		return err
	}
	return report(context.Background(), pool, o, threads, nil, res, false)
}

// runAll is the all-in-one mode: the fake, setup, the clients and the
// report in one process, on one machine's clock.
func runAll(ctx context.Context, o options) error {
	if o.db == "" {
		return errors.New("set -db or REPUTECAST_DATABASE__URL")
	}
	pool, err := pgxpool.New(ctx, o.db)
	if err != nil {
		return err
	}
	defer pool.Close()
	fake := &provider{chunks: o.chunks, every: o.every}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	serve, err := fake.listen(ctx, o.fake)
	if err != nil {
		return err
	}
	go func() { _ = serve() }()
	p, err := setupPlan(ctx, pool, o)
	if err != nil {
		return err
	}
	threads := p.threads(o.base)
	log.Printf("%d threads; opening their streams", len(threads))
	res, err := drive(ctx, o, pool, threads)
	if err != nil {
		return err
	}
	return report(context.Background(), pool, o, threads, fake, res, true)
}
