package website

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// KindDomainProblem notifies the people who manage the website that its
// custom domain stopped checking out or its certificate is failing.
const KindDomainProblem = notify.KindWebsiteDomain

// TickArgs is the minute job: posts whose time came, and posts of the
// campaigns the archive filed since.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "website.tick" }

// InsertOpts runs once at a time.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

type tickWorker struct {
	river.WorkerDefaults[TickArgs]
	s *Service
}

func (w *tickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error {
	if _, err := w.s.PublishDue(ctx); err != nil {
		return err
	}
	return w.s.syncIssues(ctx)
}

func (w *tickWorker) Timeout(*river.Job[TickArgs]) time.Duration { return 5 * time.Minute }

// HourlyArgs is the hourly job: views past their keeping, used sign-in
// links, and the custom domain's certificate.
type HourlyArgs struct{}

// Kind implements river.JobArgs.
func (HourlyArgs) Kind() string { return "website.hourly" }

// InsertOpts runs on the maintenance queue.
func (HourlyArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Minute}}
}

type hourlyWorker struct {
	river.WorkerDefaults[HourlyArgs]
	s *Service
}

func (w *hourlyWorker) Work(ctx context.Context, _ *river.Job[HourlyArgs]) error {
	if err := w.s.PurgeViews(ctx); err != nil {
		return err
	}
	if err := w.s.PurgeLinks(ctx); err != nil {
		return err
	}
	return w.s.CheckDomains(ctx)
}

func (w *hourlyWorker) Timeout(*river.Job[HourlyArgs]) time.Duration { return 10 * time.Minute }

// RegisterJobs adds the website's workers and periodic jobs.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &turnOnWorker{s: s})
	river.AddWorker(reg.Workers, &tickWorker{s: s})
	river.AddWorker(reg.Workers, &hourlyWorker{s: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, true)
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return HourlyArgs{}, nil }, true)
}

// PurgeLinks deletes sign-in links a day after they could last be used.
func (s *Service) PurgeLinks(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM website_signin_links WHERE expires_at < $1`, s.now().Add(-24*time.Hour))
	return err
}

// CheckDomains looks at the custom domain of every site that is on: when it
// stops checking out or its certificate fails, the people with
// website:manage are told once per problem, until it is fine again.
func (s *Service) CheckDomains(ctx context.Context) error {
	ids, err := s.Q.WebsitesOn(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		ws, err := s.Q.GetWorkspace(ctx, id)
		if err != nil {
			continue
		}
		if err := s.checkDomain(ctx, &ws); err != nil {
			s.log().WarnContext(ctx, "website domain check", "workspace", ws.Slug, "err", err)
		}
	}
	return nil
}

func (s *Service) checkDomain(ctx context.Context, ws *store.Workspace) error {
	addr := s.address(ctx, ws, true)
	problem := ""
	switch {
	case addr.Domain == "":
	case addr.DomainStatus == "failed":
		problem = "domain"
	case addr.Certificate.Status == "failing":
		problem = "certificate"
	}
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return err
	}
	switch {
	case problem == "" && c.row.DomainNotified != "":
		_, err := s.Pool.Exec(ctx, `UPDATE website_settings SET domain_notified = '' WHERE workspace_id = $1`, ws.ID)
		return err
	case problem == "" || c.row.DomainNotified == problem:
		return nil
	}
	// Notifications are written once for every member, in English, as the
	// other packages' are; the words are in the catalog.
	l := i18n.For(i18n.English)
	title := msgs.M("notify.domainTitle").Text(l, nil)
	body := msgs.M("notify.domainBody").Text(l, i18n.Args{"domain": addr.Domain, "fallback": addr.Fallback})
	if problem == "certificate" {
		title = msgs.M("notify.certTitle").Text(l, nil)
		body = msgs.M("notify.certBody").Text(l, i18n.Args{"domain": addr.Domain})
		if addr.Certificate.Detail != "" {
			body = msgs.M("notify.certBodyDetail").Text(l, i18n.Args{"domain": addr.Domain, "detail": addr.Certificate.Detail})
		}
	}
	n := notify.Notification{WorkspaceID: &ws.ID, Kind: KindDomainProblem, Severity: "bad", Title: title, Body: body,
		Link: "/w/" + ws.Slug + "/forms/branding", Data: map[string]any{"domain": addr.Domain, "problem": problem}}
	if err := notify.ToMembersWith(ctx, s.Pool, ws.ID, authz.WebsiteManage, n); err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `UPDATE website_settings SET domain_notified = $2 WHERE workspace_id = $1`, ws.ID, problem)
	return err
}
