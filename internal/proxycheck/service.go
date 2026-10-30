package proxycheck

import (
	"context"
	"net/netip"
	"sort"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// KindUntrustedProxy is the instance admins' notification.
const KindUntrustedProxy = "untrusted_proxy"

// Service tells the instance's admins about untrusted proxies and lists
// them for the admin API.
type Service struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Trusted []netip.Prefix
	Now     func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Notify tells every instance admin about each proxy seen often enough in
// the last day that they weren't told about this week, and forgets
// proxies that are trusted now or gone. It tells about maxTold proxies a
// run and maxToldDaily a day, those on a private network first. It
// returns how many proxies the admins were told about.
func (s *Service) Notify(ctx context.Context) (int, error) {
	now := s.now()
	told := 0
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		told = 0
		q := store.New(tx)
		if err := q.ProxyForgetOld(ctx, now.Add(-forgetAfter)); err != nil {
			return err
		}
		before := now.Add(-remindAfter)
		rows, err := q.ProxyDue(ctx, store.ProxyDueParams{MinRequests: minRequests, SeenSince: now.Add(-24 * time.Hour), NotifiedBefore: &before})
		if err != nil || len(rows) == 0 {
			return err
		}
		today, err := q.ProxyToldSince(ctx, now.Add(-24*time.Hour))
		if err != nil {
			return err
		}
		admins, err := q.InstanceAdminAdminNames(ctx)
		if err != nil {
			return err
		}
		// A proxy on a private network is the one to add: it goes first.
		sort.SliceStable(rows, func(i, j int) bool { return private(rows[i].Peer) && !private(rows[j].Peer) })
		for _, r := range rows {
			if trusted(r.Peer, s.Trusted) {
				if err := q.ProxyForget(ctx, r.Peer); err != nil {
					return err
				}
				continue
			}
			if told >= maxTold || today+int64(told) >= maxToldDaily {
				continue // a later run
			}
			title, body := msgs.M("notify.title"), msgs.M("notify.body")
			if !private(r.Peer) {
				// Anyone can send forwarding headers: a public address is
				// one to check, never one to add.
				title, body = msgs.M("notify.publicTitle"), msgs.M("notify.publicBody")
			}
			args := i18n.Args{"peer": r.Peer.String(), "header": r.Header, "setting": Setting, "env": Env}
			for _, a := range admins {
				if err := notify.ToUser(ctx, tx, a.ID, notify.Notification{Kind: KindUntrustedProxy, Severity: "warn",
					TitleText: title.Keep(nil), BodyText: body.Keep(args), Link: "/admin/instance",
					Data: map[string]any{"peer": r.Peer.String(), "setting": Setting, "env": Env}}); err != nil {
					return err
				}
			}
			if err := q.ProxyNotified(ctx, store.ProxyNotifiedParams{Peer: r.Peer, At: &now}); err != nil {
				return err
			}
			told++
		}
		return nil
	})
	return told, err
}

// UntrustedProxies is what the admin API shows: the proxies seen, the
// setting that fixes it, and what is trusted now.
type UntrustedProxies struct {
	Proxies []UntrustedProxy `json:"proxies"`
	Setting string           `json:"setting" doc:"The configuration setting that lists trusted proxies"`
	Env     string           `json:"env" doc:"The same setting as an environment variable"`
	Trusted []string         `json:"trusted" doc:"The proxies trusted now"`
	Advice  string           `json:"advice" doc:"What to do, in the reader's language"`
}

// UntrustedProxy is one proxy seen forwarding requests.
type UntrustedProxy struct {
	Address    string     `json:"address"`
	Header     string     `json:"header" doc:"The forwarding header it sent"`
	Requests   int64      `json:"requests"`
	FirstSeen  time.Time  `json:"firstSeen"`
	LastSeen   time.Time  `json:"lastSeen"`
	NotifiedAt *time.Time `json:"notifiedAt,omitempty" doc:"When admins were last told"`
}

// List returns the proxies seen that aren't trusted now.
func (s *Service) List(ctx context.Context) (UntrustedProxies, error) {
	l := i18n.LangFrom(ctx)
	out := UntrustedProxies{Proxies: []UntrustedProxy{}, Setting: Setting, Env: Env, Trusted: []string{}}
	for _, p := range s.Trusted {
		out.Trusted = append(out.Trusted, p.String())
	}
	rows, err := s.Q.ProxyList(ctx)
	if err != nil {
		return out, err
	}
	public := false
	for _, r := range rows {
		if trusted(r.Peer, s.Trusted) {
			continue
		}
		public = public || !private(r.Peer)
		out.Proxies = append(out.Proxies, UntrustedProxy{Address: r.Peer.String(), Header: r.Header, Requests: r.Requests, FirstSeen: r.FirstSeen,
			LastSeen: r.LastSeen, NotifiedAt: r.NotifiedAt})
	}
	switch {
	case len(out.Proxies) == 0:
		out.Advice = msgs.M("advice.none").Text(l, nil)
	case public:
		// Never "add these addresses" when one of them is public.
		out.Advice = msgs.M("advice.public").Text(l, i18n.Args{"setting": Setting, "env": Env})
	default:
		out.Advice = msgs.M("advice.some").Text(l, i18n.Args{"setting": Setting, "env": Env})
	}
	return out, nil
}

// Register adds the instance admin's view of untrusted proxies.
func Register(h huma.API, s *Service) {
	api.Get(h, api.Op{ID: "list-untrusted-proxies", Path: "/admin/untrusted-proxies",
		Summary:     "Proxies that forward requests to ReputeCast without being trusted, and the setting that fixes it",
		Description: "Requests through a proxy that http.trusted_proxies doesn't list carry the proxy's address instead of the reader's, so the bot filter and rate limits can't see readers.",
		Tags:        []string{"Instance admin · Tracking"}, Access: api.InstanceAdmin},
		func(ctx context.Context, _ *struct{}) (*api.Body[UntrustedProxies], error) {
			out, err := s.List(ctx)
			return api.OK(out), err
		})
}

// NotifyArgs tells admins about untrusted proxies (periodic).
type NotifyArgs struct{}

// Kind implements river.JobArgs.
func (NotifyArgs) Kind() string { return "proxycheck.notify" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (NotifyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 5 * time.Minute}}
}

// NotifyWorker runs NotifyArgs.
type NotifyWorker struct {
	river.WorkerDefaults[NotifyArgs]
	S *Service
}

// Work implements river.Worker.
func (w *NotifyWorker) Work(ctx context.Context, _ *river.Job[NotifyArgs]) error {
	_, err := w.S.Notify(ctx)
	return err
}
