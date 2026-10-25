package mailbox

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// Classifier keeps contacts' mailbox providers right for company domains.
//
// Contacts are added with the provider their domain's name gives (every
// insert path does that); a company domain starts as Other. The sweep job
// looks those domains up and applies the answer to every contact on them,
// and the send paths call ClassifyPending first so a fresh import is
// routed, throttled and guarded by its real mailbox provider. Addresses
// that aren't contacts (transactional recipients, the route tester) are
// classified with ForEmail from the cache, looking a new domain up in the
// background.
type Classifier struct {
	Q    *store.Queries
	MX   *MXChecker
	Jobs func() *jobs.Client
	Log  *slog.Logger

	mu      sync.Mutex
	known   map[string]remembered
	backoff map[string]time.Time
}

type remembered struct {
	class Class
	at    time.Time
}

const (
	// rememberFor bounds how long ForEmail trusts its memory.
	rememberFor = time.Hour
	// unknownFor is how long ForEmail answers Other for a domain whose
	// lookup was asked for, before checking the cache again.
	unknownFor = 2 * time.Minute
	// retryAfter keeps a domain whose lookup failed (timeout, server
	// failure) out of the next batches.
	retryAfter = time.Hour
	// maxBackoff bounds the skip list sent to Postgres.
	maxBackoff = 500
	batch      = 200
)

// ForEmail classifies an address. Consumer domains answer at once; a
// company domain comes from the MX cache, and a domain never seen is
// looked up in the background while this call answers Other.
func (c *Classifier) ForEmail(ctx context.Context, email string) Class {
	domain := emailaddr.Domain(email)
	s := Static(domain)
	if c == nil || domain == "" || s.Provider != Other {
		return s
	}
	now := time.Now()
	c.mu.Lock()
	if r, ok := c.known[domain]; ok {
		ttl := rememberFor
		if r.class.Provider == Other && r.class.Detail == "" {
			ttl = unknownFor
		}
		if now.Sub(r.at) < ttl {
			c.mu.Unlock()
			return r.class
		}
	}
	c.mu.Unlock()
	cl, cached := c.cached(ctx, domain)
	if !cached {
		c.lookupLater(ctx, domain)
	}
	c.remember(domain, cl, now)
	return cl
}

// cached reads the domain's classification from the MX cache.
func (c *Classifier) cached(ctx context.Context, domain string) (Class, bool) {
	if c.Q == nil {
		return Static(domain), false
	}
	rows, err := c.Q.GetMXCache(ctx, store.GetMXCacheParams{Domains: []string{domain}, FreshAfter: time.Now().Add(-c.MX.ttl())})
	if err != nil || len(rows) == 0 {
		return Static(domain), false
	}
	return FromHosting(domain, rows[0].Provider), true
}

func (c *Classifier) remember(domain string, cl Class, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.known == nil {
		c.known = map[string]remembered{}
	}
	if len(c.known) > 50_000 {
		clear(c.known)
	}
	c.known[domain] = remembered{class: cl, at: at}
}

// lookupLater asks the lookup job to classify the domain (once an hour at
// most per domain).
func (c *Classifier) lookupLater(ctx context.Context, domain string) {
	if c.Jobs == nil {
		return
	}
	cl := c.Jobs()
	if cl == nil {
		return
	}
	_, err := cl.Insert(context.WithoutCancel(ctx), LookupArgs{Domain: domain}, &river.InsertOpts{
		UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: time.Hour},
	})
	if err != nil && c.Log != nil {
		c.Log.WarnContext(ctx, "mailbox lookup not queued", "domain", domain, "err", err)
	}
}

// ClassifyPending classifies the workspace's contacts whose company domain
// was never looked up, until done or budget runs out (a send's audience
// is then built from the answers; what's left keeps Other and is picked
// up by the sweep). It returns how many contacts changed.
func (c *Classifier) ClassifyPending(ctx context.Context, workspaceID uuid.UUID, budget time.Duration) (int64, error) {
	if c == nil {
		return 0, nil
	}
	ws := workspaceID
	return c.run(ctx, &ws, budget, 0)
}

// Sweep classifies waiting contacts across workspaces, at most maxDomains
// domains or budget.
func (c *Classifier) Sweep(ctx context.Context, budget time.Duration, maxDomains int) (int64, error) {
	return c.run(ctx, nil, budget, maxDomains)
}

func (c *Classifier) run(ctx context.Context, ws *uuid.UUID, budget time.Duration, maxDomains int) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	var changed int64
	seen := 0
	for {
		domains, err := c.Q.MailboxPendingDomains(ctx, store.MailboxPendingDomainsParams{WorkspaceID: ws, Skip: c.skipList(), Lim: batch})
		if err != nil {
			if ctx.Err() != nil {
				return changed, nil
			}
			return changed, err
		}
		if len(domains) == 0 {
			return changed, nil
		}
		n, err := c.classify(ctx, domains)
		changed += n
		if err != nil {
			if ctx.Err() != nil {
				return changed, nil
			}
			return changed, err
		}
		seen += len(domains)
		if ctx.Err() != nil || len(domains) < batch || (maxDomains > 0 && seen >= maxDomains) {
			return changed, nil
		}
	}
}

// classify looks the domains up and applies the definite answers to the
// contacts waiting for them; failed lookups are retried later.
func (c *Classifier) classify(ctx context.Context, domains []string) (int64, error) {
	res, err := c.MX.Check(ctx, domains)
	if err != nil {
		return 0, err
	}
	var changed int64
	now := time.Now()
	for _, d := range domains {
		r, ok := res[d]
		if !ok || !r.Checked {
			c.failed(d, now)
			continue
		}
		cl := FromHosting(d, r.Hosting)
		n, err := c.Q.MailboxClassifyContacts(ctx, store.MailboxClassifyContactsParams{Provider: cl.Provider, Detail: cl.Detail, Domain: d})
		if err != nil {
			return changed, err
		}
		changed += n
		c.remember(d, cl, now)
	}
	return changed, nil
}

func (c *Classifier) failed(domain string, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.backoff == nil {
		c.backoff = map[string]time.Time{}
	}
	c.backoff[domain] = at
}

// skipList is the domains whose lookup failed within retryAfter.
func (c *Classifier) skipList() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := []string{}
	now := time.Now()
	for d, at := range c.backoff {
		if now.Sub(at) >= retryAfter {
			delete(c.backoff, d)
			continue
		}
		if len(out) < maxBackoff {
			out = append(out, d)
		}
	}
	return out
}

// SweepArgs is the periodic sweep.
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "mailbox.sweep" }

// InsertOpts runs the sweep on the maintenance queue, one at a time.
func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: time.Minute}}
}

// SweepWorker classifies waiting contacts in the background.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	C *Classifier
}

// Work implements river.Worker.
func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	n, err := w.C.Sweep(ctx, 90*time.Second, 2000)
	if n > 0 && w.C.Log != nil {
		w.C.Log.InfoContext(ctx, "mailbox providers classified from MX records", "contacts", n)
	}
	return err
}

// LookupArgs asks for one domain's classification.
type LookupArgs struct {
	Domain string `json:"domain"`
}

// Kind implements river.JobArgs.
func (LookupArgs) Kind() string { return "mailbox.lookup" }

// LookupWorker looks a domain up and applies it to waiting contacts.
type LookupWorker struct {
	river.WorkerDefaults[LookupArgs]
	C *Classifier
}

// Work implements river.Worker.
func (w *LookupWorker) Work(ctx context.Context, j *river.Job[LookupArgs]) error {
	_, err := w.C.classify(ctx, []string{j.Args.Domain})
	return err
}
