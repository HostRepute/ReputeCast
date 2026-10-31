package app

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailbox"
)

// Mailbox providers of company domains, from their MX records
// (internal/mailbox): the sweep, the per-domain lookup and the classifier
// the send paths share.

func init() {
	jobRegistrars = append(jobRegistrars, func(a *App) {
		c := a.mailboxClassifier()
		river.AddWorker(a.JobReg.Workers, &mailbox.SweepWorker{C: c})
		river.AddWorker(a.JobReg.Workers, &mailbox.LookupWorker{C: c})
		a.JobReg.Every(2*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return mailbox.SweepArgs{}, nil }, true)
	})
}

// mailboxClassifiers keeps one classifier per App: its memory of recent
// answers is shared by every service that classifies addresses.
var mailboxClassifiers sync.Map // *App → *mailbox.Classifier

func (a *App) mailboxClassifier() *mailbox.Classifier {
	if c, ok := mailboxClassifiers.Load(a); ok {
		return c.(*mailbox.Classifier)
	}
	c := &mailbox.Classifier{
		Q:    a.Q,
		MX:   &mailbox.MXChecker{Q: a.Q, Resolver: dnsResolver(a.Cfg)},
		Jobs: func() *jobs.Client { return a.Jobs },
		Log:  a.Log,
	}
	actual, _ := mailboxClassifiers.LoadOrStore(a, c)
	return actual.(*mailbox.Classifier)
}

// dnsResolver is the system resolver, or the configured DNS servers when
// the instance sets them.
func dnsResolver(cfg *config.Config) *net.Resolver {
	if cfg == nil || len(cfg.DNS.Servers) == 0 {
		return net.DefaultResolver
	}
	server := cfg.DNS.Servers[0]
	if _, _, err := net.SplitHostPort(server); err != nil {
		server = net.JoinHostPort(server, "53")
	}
	d := &net.Dialer{Timeout: cfg.DNS.Timeout}
	return &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		return d.DialContext(ctx, network, server)
	}}
}
