package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	lagMetric  = "reputecast_ai_assistant_event_lag_seconds"
	ttftMetric = "reputecast_ai_assistant_ttft_overhead_seconds"
)

// dist is a distribution's size and quantiles, in milliseconds.
type dist struct {
	N   int     `json:"n"`
	P50 float64 `json:"p50Ms"`
	P95 float64 `json:"p95Ms"`
	P99 float64 `json:"p99Ms"`
	Max float64 `json:"maxMs"`
}

func (d dist) String() string {
	if d.N == 0 {
		return "none"
	}
	s := fmt.Sprintf("n=%d p50=%.1f p95=%.1f p99=%.1f", d.N, d.P50, d.P95, d.P99)
	if d.Max > 0 {
		s += fmt.Sprintf(" max=%.1f", d.Max)
	}
	return s
}

// quantiles are the nearest-rank quantiles of v.
func quantiles(v []float64) dist {
	if len(v) == 0 {
		return dist{}
	}
	slices.Sort(v)
	q := func(p float64) float64 { return v[min(len(v)-1, int(math.Ceil(p*float64(len(v))))-1)] }
	return dist{N: len(v), P50: q(.50), P95: q(.95), P99: q(.99), Max: v[len(v)-1]}
}

// summary is a run's report, also written to -out.
type summary struct {
	Role     string    `json:"role"`
	Machine  string    `json:"machine"`
	Streams  int       `json:"streams"`
	Turns    int       `json:"turns"`
	Rounds   int       `json:"rounds"`
	Spread   string    `json:"spread"`
	Took     string    `json:"took"`
	From     time.Time `json:"from,omitzero"` // the database's clock
	To       time.Time `json:"to,omitzero"`
	Errors   int       `json:"errors"`
	Gaps     int       `json:"gaps"`               // events missing between two received
	Missing  int       `json:"missing"`            // events in the database not received (-1: not checked)
	Lag      *histo    `json:"serverDeltaLag"`     // the authoritative measure
	LagAll   *histo    `json:"serverEventLag"`     // every kind
	TTFTHist *histo    `json:"serverTTFTOverhead"` // coarse buckets
	TTFTDB   dist      `json:"ttftDatabase"`       // message committed to first delta committed
	TTFTSeen dist      `json:"ttftClient"`         // message sent to first delta received, this machine's clock
	Posts    dist      `json:"messagePost"`
	Deltas   *dist     `json:"deltaCommitToClient,omitempty"` // one clock only
	All      *dist     `json:"eventCommitToClient,omitempty"`
}

// report prints and (with -out) writes the run's numbers. sameClock: the
// clients and the instance share a clock, so an event's commit and its
// arrival compare.
func report(ctx context.Context, pool *pgxpool.Pool, o options, threads []*thread, fake *provider, res result, sameClock bool) error {
	s := summary{Role: "clients", Machine: machine(), Streams: len(threads), Rounds: o.rounds, Spread: o.spread.String(), Took: res.took.Round(time.Millisecond).String(),
		From: res.from, To: res.to, Missing: -1}
	if sameClock {
		s.Role = "all"
	}
	ids := make([]uuid.UUID, len(threads))
	byID := map[uuid.UUID]*thread{}
	firstDelta := map[uuid.UUID][]time.Time{}
	for i, t := range threads {
		ids[i], byID[t.id] = t.id, t
		t.mu.Lock()
		seqs := make([]int64, 0, len(t.got))
		for seq, a := range t.got {
			seqs = append(seqs, seq)
			if a.kind == "delta" {
				firstDelta[t.id] = append(firstDelta[t.id], a.at)
			}
		}
		t.mu.Unlock()
		slices.Sort(seqs)
		slices.SortFunc(firstDelta[t.id], func(a, b time.Time) int { return a.Compare(b) })
		if len(seqs) > 0 {
			s.Gaps += int(seqs[len(seqs)-1]-t.seq) - len(seqs)
		}
		for _, e := range t.errs {
			if s.Errors++; s.Errors <= 5 {
				log.Printf("thread %s: %v", t.id, e)
			}
		}
		s.Turns += len(t.sent)
	}
	// The client's time to first token: from sending each message to the
	// first delta received after it.
	var ttft, posts []float64
	for _, t := range threads {
		posts = append(posts, t.post...)
		for _, sent := range t.sent {
			if i := slices.IndexFunc(firstDelta[t.id], func(at time.Time) bool { return at.After(sent) }); i >= 0 {
				ttft = append(ttft, firstDelta[t.id][i].Sub(sent).Seconds()*1000)
			}
		}
	}
	s.TTFTSeen, s.Posts = quantiles(ttft), quantiles(posts)
	if res.before != nil && res.after != nil {
		s.Lag, s.LagAll, s.TTFTHist = histogram(res.before, res.after, lagMetric, `kind="delta"`), histogram(res.before, res.after, lagMetric, ""),
			histogram(res.before, res.after, ttftMetric, "")
	}
	if pool != nil {
		var err error
		if s.Missing, s.Deltas, s.All, err = commits(ctx, pool, ids, byID, sameClock); err != nil {
			return err
		}
		if s.TTFTDB, _, err = dbTTFT(ctx, pool, ids, res.from, res.to); err != nil {
			return err
		}
	}

	fmt.Printf("\n%s: %s, Go %s, %s\n", s.Role, s.Machine, runtime.Version(), time.Now().UTC().Format(time.RFC3339))
	shape := ""
	if fake != nil {
		shape = fmt.Sprintf(", %d model streams answered by the fake (%d pieces every %s)", fake.streams.Load(), o.chunks, o.every)
	}
	fmt.Printf("run: %d streams, %d turns (%d rounds, each message up to %s after the last turn)%s, %s\n", s.Streams, s.Turns, o.rounds, o.spread, shape, s.Took)
	missing := "not checked (no -db)"
	if s.Missing >= 0 {
		missing = strconv.Itoa(s.Missing)
	}
	fmt.Printf("errors: %d; events skipped between two received: %d; events in the database not received: %s\n", s.Errors, s.Gaps, missing)
	fmt.Println("\nserver side, the instance's clock (authoritative):")
	if s.Lag == nil {
		fmt.Println("  histograms: not read (set -metrics-token, and -metrics if /metrics isn't at -base)")
	} else {
		fmt.Printf("  delta events, commit to written to the stream (%s{kind=\"delta\"}): %s\n", lagMetric, s.Lag)
		fmt.Printf("  every event kind, commit to written to the stream: %s\n", s.LagAll)
		fmt.Printf("  time to first token overhead (%s, buckets 50/100/250 ms): %s\n", ttftMetric, s.TTFTHist)
	}
	if pool != nil {
		fmt.Printf("  time to first token by the database, message committed to its first delta committed (ms): %s\n", s.TTFTDB)
	}
	fmt.Printf("\nclient side, this machine's clock only (secondary):\n")
	fmt.Printf("  time to first token, message sent to its first delta received (ms): %s\n", s.TTFTSeen)
	fmt.Printf("  message POST, sent to answered (ms): %s\n", s.Posts)
	if s.Deltas != nil {
		fmt.Printf("  delta events, commit to client (ms): %s\n", *s.Deltas)
		fmt.Printf("  all events, commit to client (ms):   %s\n", *s.All)
	}
	if sameClock {
		secs := res.took.Seconds()
		fmt.Printf("  CPU: server %.1f s (%.2f cores), Postgres %.1f s (%.2f cores), over %.1f s\n", res.cpu1.server-res.cpu0.server, (res.cpu1.server-res.cpu0.server)/secs,
			res.cpu1.postgres-res.cpu0.postgres, (res.cpu1.postgres-res.cpu0.postgres)/secs, secs)
	} else if !s.From.IsZero() {
		fmt.Printf("\nthe instance's CPU over the run, on its machine:\n  assistantload collect -from %s -to %s\n", s.From.UTC().Format(time.RFC3339Nano), s.To.UTC().Format(time.RFC3339Nano))
	}
	if o.out != "" {
		raw, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(o.out, raw, 0o600); err != nil {
			return err
		}
		log.Printf("wrote %s", o.out)
	}
	if s.Missing > 0 || s.Errors > 0 || s.Gaps > 0 {
		return fmt.Errorf("%d events not received, %d skipped, %d errors", max(s.Missing, 0), s.Gaps, s.Errors)
	}
	return nil
}

// commits counts the events in the database the streams didn't receive
// and, on one clock, times each received event from its commit.
func commits(ctx context.Context, pool *pgxpool.Pool, ids []uuid.UUID, byID map[uuid.UUID]*thread, sameClock bool) (int, *dist, *dist, error) {
	rows, err := pool.Query(ctx, `SELECT thread_id, seq, kind, created_at FROM ai_assistant_events WHERE thread_id = ANY($1) ORDER BY thread_id, seq`, ids)
	if err != nil {
		return 0, nil, nil, err
	}
	defer rows.Close()
	missing := 0
	var deltas, all []float64
	for rows.Next() {
		var id uuid.UUID
		var seq int64
		var kind string
		var at time.Time
		if err := rows.Scan(&id, &seq, &kind, &at); err != nil {
			return 0, nil, nil, err
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
		}
	}
	if err := rows.Err(); err != nil || !sameClock {
		return missing, nil, nil, err
	}
	d, a := quantiles(deltas), quantiles(all)
	return missing, &d, &a, nil
}

// dbTTFT is each turn's time to its first token by the database: the
// turn's message committed (its created_at) to its first delta committed,
// for the threads' turns created from from to to. Both times are the
// instance's. The fake streams its first piece at once, so this is what
// the TTFT overhead histogram measures, per turn rather than by bucket.
func dbTTFT(ctx context.Context, pool *pgxpool.Pool, ids []uuid.UUID, from, to time.Time) (dist, map[string]int, error) {
	if to.IsZero() {
		to = time.Now().Add(time.Hour)
	}
	rows, err := pool.Query(ctx, `SELECT t.state, t.created_at,
			(SELECT min(e.created_at) FROM ai_assistant_events e WHERE e.thread_id = t.thread_id AND e.turn_id = t.id AND e.kind = 'delta')
		FROM ai_assistant_turns t WHERE t.thread_id = ANY($1) AND t.created_at >= $2 AND t.created_at <= $3`, ids, from, to)
	if err != nil {
		return dist{}, nil, err
	}
	defer rows.Close()
	states := map[string]int{}
	var v []float64
	for rows.Next() {
		var state string
		var sent time.Time
		var first *time.Time
		if err := rows.Scan(&state, &sent, &first); err != nil {
			return dist{}, nil, err
		}
		states[state]++
		if first != nil {
			v = append(v, first.Sub(sent).Seconds()*1000)
		}
	}
	return quantiles(v), states, rows.Err()
}

// histo is a histogram's counts over the run and its quantiles (ms)
// interpolated within the buckets, as Prometheus's histogram_quantile.
type histo struct {
	N          float64  `json:"n"`
	P50        float64  `json:"p50Ms"`
	P95        float64  `json:"p95Ms"`
	P99        float64  `json:"p99Ms"`
	Under100ms float64  `json:"shareUnder100ms"`
	Buckets    []bucket `json:"buckets"`
}

type bucket struct {
	LE float64
	N  float64
}

// MarshalJSON writes the bound as text: JSON has no +Inf.
func (b bucket) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"le": strconv.FormatFloat(b.LE, 'g', -1, 64), "n": b.N})
}

func (h *histo) String() string {
	if h == nil || h.N == 0 {
		return "no samples"
	}
	return fmt.Sprintf("n=%.0f p50=%.1f p95=%.1f p99=%.1f ms (interpolated); %.1f%% at or under 100 ms", h.N, h.P50, h.P95, h.P99, h.Under100ms*100)
}

// histogram is metric's histogram over the run (after − before), summed
// over its series whose labels contain labels ("" every series).
func histogram(before, after map[string]float64, metric, labels string) *histo {
	sum := map[float64]float64{}
	for k, v := range after {
		if !strings.HasPrefix(k, metric+"_bucket{") || (labels != "" && !strings.Contains(k, labels)) {
			continue
		}
		i := strings.Index(k, `le="`)
		if i < 0 {
			continue
		}
		raw, _, _ := strings.Cut(k[i+4:], `"`)
		le, err := strconv.ParseFloat(raw, 64) // "+Inf" too
		if err != nil {
			continue
		}
		sum[le] += v - before[k]
	}
	h := &histo{}
	for le, n := range sum {
		h.Buckets = append(h.Buckets, bucket{le, n})
	}
	slices.SortFunc(h.Buckets, func(a, b bucket) int {
		switch {
		case a.LE < b.LE:
			return -1
		case a.LE > b.LE:
			return 1
		}
		return 0
	})
	if len(h.Buckets) == 0 {
		return h
	}
	h.N = h.Buckets[len(h.Buckets)-1].N
	if h.N == 0 {
		return h
	}
	h.P50, h.P95, h.P99 = histQuantile(.50, h.Buckets)*1000, histQuantile(.95, h.Buckets)*1000, histQuantile(.99, h.Buckets)*1000
	for _, b := range h.Buckets {
		if b.LE == .1 {
			h.Under100ms = b.N / h.N
		}
	}
	return h
}

// histQuantile is the q-quantile of cumulative buckets sorted by upper
// bound, interpolated linearly within its bucket (the first from 0); in
// the +Inf bucket it is the highest finite bound.
func histQuantile(q float64, bs []bucket) float64 {
	rank := q * bs[len(bs)-1].N
	lower, below := 0.0, 0.0
	for i, b := range bs {
		if b.N >= rank {
			switch {
			case math.IsInf(b.LE, 1):
				if i == 0 {
					return 0
				}
				return bs[i-1].LE
			case b.N == below:
				return b.LE
			}
			return lower + (b.LE-lower)*(rank-below)/(b.N-below)
		}
		lower, below = b.LE, b.N
	}
	return 0
}

// scrape reads the instance's histogram buckets (cumulative counts by
// series and upper bound).
func scrape(o options) (map[string]float64, error) {
	u := o.metricsURL
	if u == "" {
		if o.metricsToken == "" {
			return nil, nil
		}
		u = o.base + "/metrics"
	}
	req, _ := http.NewRequest(http.MethodGet, u, nil) //nolint:noctx // a one-off scrape
	if o.metricsToken != "" {
		req.Header.Set("Authorization", "Bearer "+o.metricsToken)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close() //nolint:errcheck
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %d", u, res.StatusCode)
	}
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

// machine is this machine's CPUs and their model.
func machine() string {
	model := "unknown CPU"
	switch runtime.GOOS {
	case "linux":
		raw, _ := os.ReadFile("/proc/cpuinfo")
		for line := range strings.SplitSeq(string(raw), "\n") {
			if strings.HasPrefix(line, "model name") {
				_, model, _ = strings.Cut(line, ": ")
				break
			}
		}
	case "darwin":
		if out, err := exec.CommandContext(context.Background(), "sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
			model = strings.TrimSpace(string(out))
		}
	}
	return fmt.Sprintf("%d CPUs (%s), %s/%s", runtime.NumCPU(), model, runtime.GOOS, runtime.GOARCH)
}

// runCollect is the collect role, on the instance's machine: the load
// test's turns from -from to -to (the clients print both; without them,
// those of the last -last) by the database, and the CPU the instance,
// Postgres and the machine used then, from the provider role's samples.
func runCollect(ctx context.Context, o options) error {
	pool, err := pgxpool.New(ctx, o.db)
	if err != nil {
		return err
	}
	defer pool.Close()
	var from, to time.Time
	if o.from != "" || o.to != "" {
		if from, err = time.Parse(time.RFC3339Nano, o.from); err != nil {
			return fmt.Errorf("-from: %w", err)
		}
		if to, err = time.Parse(time.RFC3339Nano, o.to); err != nil {
			return fmt.Errorf("-to: %w", err)
		}
	} else {
		var first, last *time.Time
		if err := pool.QueryRow(ctx, `SELECT min(t.created_at), max(coalesce(t.ended_at, t.created_at)) FROM ai_assistant_turns t
			JOIN ai_assistant_threads th ON th.id = t.thread_id JOIN users u ON u.id = th.user_id
			WHERE u.email LIKE 'load-%@load.test' AND t.created_at > clock_timestamp() - $1::interval`, o.last.String()).Scan(&first, &last); err != nil {
			return err
		}
		if first == nil {
			return fmt.Errorf("no load test turns in the last %s", o.last)
		}
		from, to = *first, *last
		log.Printf("no -from and -to: every load test turn of the last %s", o.last)
	}
	var ids []uuid.UUID
	rows, err := pool.Query(ctx, `SELECT DISTINCT t.thread_id FROM ai_assistant_turns t JOIN ai_assistant_threads th ON th.id = t.thread_id JOIN users u ON u.id = th.user_id
		WHERE u.email LIKE 'load-%@load.test' AND t.created_at >= $1 AND t.created_at <= $2`, from, to)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	ttft, states, err := dbTTFT(ctx, pool, ids, from, to)
	if err != nil {
		return err
	}
	secs := to.Sub(from).Seconds()
	fmt.Printf("\ninstance: %s, %s\n", machine(), time.Now().UTC().Format(time.RFC3339))
	fmt.Printf("window: %s to %s (%.1f s, the database's clock); %d threads, turns by state %v\n", from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano), secs, len(ids), states)
	fmt.Printf("time to first token by the database, message committed to its first delta committed (ms): %s\n", ttft)
	q := url.Values{"from": {from.UTC().Format(time.RFC3339Nano)}, "to": {to.UTC().Format(time.RFC3339Nano)}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, o.stats+"?"+q.Encode(), nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("the provider role's CPU samples (%s): %w", o.stats, err)
	}
	defer res.Body.Close() //nolint:errcheck
	var u usage
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %d", o.stats, res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(&u); err != nil {
		return err
	}
	w := u.SampledTo.Sub(u.SampledFrom).Seconds()
	fmt.Printf("CPU, sampled %s to %s (%.1f s): server %.1f s (%.2f cores), Postgres %.1f s (%.2f cores), the whole machine %.2f of %d cores busy; load average (1 min) up to %.2f\n",
		u.SampledFrom.UTC().Format("15:04:05.0"), u.SampledTo.UTC().Format("15:04:05.0"), w, u.Server, u.Server/w, u.Postgres, u.Postgres/w, u.Busy/w, u.CPUs, u.MaxLoad1)
	fmt.Printf("model streams the fake began then: %d (%d pieces every %s)\n", u.Streams, u.Chunks, u.Every)
	return nil
}
