// Command assistant is the AI assistant's load test (B6): it opens many
// thread streams (SSE) on a running ReputeCast, sends a message in each
// thread, answers the model requests as a fake OpenAI-compatible provider
// that streams, and measures each event from its commit to its delivery.
//
// It is a small Go program, not a k6 script, for two reasons. k6 has no
// SSE client of its own: the extensions (xk6-sse and its forks) need a
// custom k6 binary built with xk6, a third-party build of a third-party
// extension that the CI and the orb would have to fetch and trust. And the
// measure the gate asks for starts at an event's commit, which only the
// instance knows (each event's row keeps its created_at, written in its
// committing transaction). It lives under loadtest/ and is not part of the
// shipped binary.
//
// All in one process, on one machine (the load competes with the instance
// for its CPUs):
//
//	REPUTECAST_DATABASE__URL=postgres://… go run ./loadtest/assistant \
//	  -base http://127.0.0.1:8080 -fake 127.0.0.1:9711 -streams 200 \
//	  -metrics-token …
//
// Or split in roles, so the streams' clients run on another machine:
//
//	provider  on the instance's machine: the fake model (-fake, -chunks,
//	          -every), and samples of the CPU the instance (-server),
//	          Postgres and the machine use, served at /loadtest/cpu.
//	setup     anywhere that reaches the database and the instance: connects
//	          the workspaces' model to the fake (-fake: its address as the
//	          instance reaches it), adds people with sessions, opens the
//	          threads and writes them to -plan (the clients' input; it
//	          holds sessions).
//	clients   on the other machine: the plan's threads (or, with -db and
//	          no -plan, setup first), driven at -base; prints the server's
//	          own histograms, read from /metrics before and after the run,
//	          as the authoritative measure, the times to first token by the
//	          database (-db), and what this machine's clock alone can time.
//	collect   on the instance's machine: the run's turns by the database
//	          and the CPU used over the run, from the provider's samples.
//
// The clocks of two machines differ, so in split mode no time is taken
// on one and compared with the other: an event's delay from its commit to
// its stream is the instance's reputecast_ai_assistant_event_lag_seconds,
// a turn's time to its first token the database's two timestamps.
//
// The instance needs the Acme sample (reputecast seed), the provider's
// address allowed (providers.allow_private_networks for a loopback fake),
// enough assistant workers for every turn at once (worker.concurrency 400
// gives the ai_assistant queue 200) and, for the histograms, a metrics
// token (metrics.token; /metrics on the main listener). Setup, done each
// run, writes to the database directly, like loadtest/session.sh:
// development and load test instances only. It connects every workspace's
// model to the fake, sets the assistant's limits to 10 turns a person and
// 100 a workspace (the most they take), adds people as Editors and gives
// them sessions.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type options struct {
	base, fake, db, metricsURL, metricsToken string
	workspaces                               []string
	streams, rounds, chunks                  int
	every, timeout, spread                   time.Duration
	serverMatch                              string
	plan, out, from, to, stats               string
	last                                     time.Duration
}

// fakeURL is the fake's API as the instance reaches it.
func (o options) fakeURL() string { return "http://" + o.fake + "/v1" }

const roles = "all (the default), provider, setup, clients or collect"

func main() {
	role, args := "all", os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		role, args = args[0], args[1:]
	}
	fs := flag.NewFlagSet("assistant "+role, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: assistant [role] [flags]; role: %s (see the package's doc)\n", roles)
		fs.PrintDefaults()
	}
	var o options
	var ws string
	fs.StringVar(&o.base, "base", "http://127.0.0.1:8080", "the instance, as this machine reaches it")
	fs.StringVar(&o.fake, "fake", "127.0.0.1:9711", "the fake provider's address: where it listens (all, provider) and how the instance reaches it (all, setup)")
	fs.StringVar(&o.db, "db", os.Getenv("REPUTECAST_DATABASE__URL"), "the instance's database: setup, the events not received and the times to first token")
	fs.StringVar(&ws, "workspaces", "acme-newsletter,acme-engineering-blog", "the workspaces the streams are spread over (100 turns at once each, at most)")
	fs.IntVar(&o.streams, "streams", 200, "thread streams open at once")
	fs.IntVar(&o.rounds, "rounds", 3, "messages sent in each thread, one after the other's turn ended")
	fs.IntVar(&o.chunks, "chunks", 150, "pieces of each streamed answer (all, provider)")
	fs.DurationVar(&o.every, "every", 20*time.Millisecond, "time between the answer's pieces (all, provider)")
	fs.DurationVar(&o.timeout, "timeout", 5*time.Minute, "how long the run may take")
	fs.DurationVar(&o.spread, "spread", 0, "each message waits a random time up to this before it is sent (0: every thread at once)")
	fs.StringVar(&o.serverMatch, "server", "reputecast serve", "a part of the instance process's command line, for its CPU (all, provider)")
	fs.StringVar(&o.metricsURL, "metrics", "", "the instance's /metrics (default: -base's, with -metrics-token)")
	fs.StringVar(&o.metricsToken, "metrics-token", "", "the metrics bearer token (metrics.token)")
	fs.StringVar(&o.plan, "plan", "", "the threads' file: written by setup, read by clients")
	fs.StringVar(&o.out, "out", "", "clients: write the report as JSON here too")
	fs.StringVar(&o.from, "from", "", "collect: the run's start, as clients print it (RFC 3339, the database's clock)")
	fs.StringVar(&o.to, "to", "", "collect: the run's end")
	fs.DurationVar(&o.last, "last", 10*time.Minute, "collect without -from and -to: the load test's turns of this last while")
	fs.StringVar(&o.stats, "stats", "http://127.0.0.1:9711/loadtest/cpu", "collect: the provider role's CPU samples")
	_ = fs.Parse(args)
	o.workspaces = strings.Split(ws, ",")
	run := map[string]func(context.Context, options) error{"all": runAll, "provider": runProvider, "setup": runSetup, "clients": runClients, "collect": runCollect}[role]
	if run == nil {
		fs.Usage()
		os.Exit(2)
	}
	if o.db == "" && role != "provider" && (role != "clients" || o.plan == "") {
		log.Fatal("set -db or REPUTECAST_DATABASE__URL")
	}
	if err := run(context.Background(), o); err != nil {
		log.Fatal(err)
	}
}
