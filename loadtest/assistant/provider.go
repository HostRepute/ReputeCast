package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// provider is an OpenAI-compatible model: it passes the tool-calling
// check and streams every answer as pieces at a steady pace. As the
// provider role it also samples the CPU of the instance, of Postgres and
// of the whole machine, and answers /loadtest/cpu with what they used
// over a window (the collect role asks it).
type provider struct {
	chunks   int
	every    time.Duration
	requests atomic.Int64
	streams  atomic.Int64

	match   string // the instance's command line, for its CPU
	mu      sync.Mutex
	samples []sample
}

// sample is the CPU used so far (seconds) and the models streams begun.
type sample struct {
	At       time.Time `json:"at"`
	Server   float64   `json:"server"`
	Postgres float64   `json:"postgres"`
	Busy     float64   `json:"busy"` // the machine's, every CPU
	Load1    float64   `json:"load1"`
	Streams  int64     `json:"streams"`
}

// listen serves the fake on addr until ctx ends.
func (p *provider) listen(ctx context.Context, addr string) (func() error, error) {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	srv := &http.Server{Handler: p, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	return func() error { return srv.Serve(ln) }, nil
}

// runProvider is the provider role: the fake, and the CPU sampler.
func runProvider(ctx context.Context, o options) error {
	p := &provider{chunks: o.chunks, every: o.every, match: o.serverMatch}
	serve, err := p.listen(ctx, o.fake)
	if err != nil {
		return err
	}
	go p.sample(ctx, 500*time.Millisecond, 4*time.Hour)
	log.Printf("fake provider on %s: %d pieces every %s; CPU of %q and Postgres sampled every 500ms", o.fake, o.chunks, o.every, o.serverMatch)
	return serve()
}

// sample records a sample every interval, keeping the last keep.
func (p *provider) sample(ctx context.Context, interval, keep time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		c := cpuSample(p.match)
		s := sample{At: time.Now(), Server: c.server, Postgres: c.postgres, Busy: machineBusy(), Load1: load1(), Streams: p.streams.Load()}
		p.mu.Lock()
		p.samples = append(p.samples, s)
		if n := int(keep / interval); len(p.samples) > n {
			p.samples = append(p.samples[:0], p.samples[len(p.samples)-n:]...)
		}
		p.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// usage is what the CPU did over a window.
type usage struct {
	From, To               time.Time
	CPUs                   int
	Server, Postgres       float64 // CPU seconds
	Busy                   float64 // the machine's CPU seconds, every process
	MaxLoad1               float64
	Streams                int64 // model streams begun
	Chunks                 int
	Every                  string
	SampledFrom, SampledTo time.Time
}

// window is the usage between the last sample at or before from and the
// first at or after to.
func (p *provider) window(from, to time.Time) (usage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	u := usage{From: from, To: to, CPUs: runtime.NumCPU(), Chunks: p.chunks, Every: p.every.String()}
	if len(p.samples) < 2 {
		return u, fmt.Errorf("%d CPU samples so far", len(p.samples))
	}
	a, b := 0, len(p.samples)-1
	for i, s := range p.samples {
		if !s.At.After(from) {
			a = i
		}
		if !s.At.Before(to) {
			b = i
			break
		}
	}
	sa, sb := p.samples[a], p.samples[b]
	u.SampledFrom, u.SampledTo = sa.At, sb.At
	u.Server, u.Postgres, u.Busy, u.Streams = sb.Server-sa.Server, sb.Postgres-sa.Postgres, sb.Busy-sa.Busy, sb.Streams-sa.Streams
	for _, s := range p.samples[a : b+1] {
		u.MaxLoad1 = max(u.MaxLoad1, s.Load1)
	}
	return u, nil
}

func (p *provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/loadtest/cpu" {
		from, err1 := time.Parse(time.RFC3339Nano, r.URL.Query().Get("from"))
		to, err2 := time.Parse(time.RFC3339Nano, r.URL.Query().Get("to"))
		if err1 != nil || err2 != nil {
			http.Error(w, "from and to: RFC 3339 times", http.StatusBadRequest)
			return
		}
		u, err := p.window(from, to)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(u)
		return
	}
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

type cpu struct{ server, postgres float64 }

// cpuSample is the CPU time (user and system) of the instance's processes
// and Postgres's, in seconds (Linux; zero elsewhere).
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
		if len(f) < 15 {
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

// machineBusy is the CPU time every CPU of the machine spent busy, in
// seconds (/proc/stat: user, nice, system, irq, softirq and steal).
func machineBusy() float64 {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	line, _, _ := strings.Cut(string(raw), "\n")
	f := strings.Fields(line)
	busy := 0.0
	for _, i := range []int{1, 2, 3, 6, 7, 8} {
		if i < len(f) {
			v, _ := strconv.ParseFloat(f[i], 64)
			busy += v
		}
	}
	return busy / 100
}

func load1() float64 {
	raw, _ := os.ReadFile("/proc/loadavg")
	f := strings.Fields(string(raw))
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[0], 64)
	return v
}
