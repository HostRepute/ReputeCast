package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/notify"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		notify.Register(h, &notify.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Mail: a.notifyMailer})
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		river.AddWorker(a.JobReg.Workers, &notify.EmailWorker{Pool: a.Pool, Mail: a.notifyMailer, PublicURL: a.notifyPublicURL, Log: a.Log})
		a.JobReg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return notify.EmailArgs{}, nil }, false)
		river.AddWorker(a.JobReg.Workers, &notify.CleanupWorker{Pool: a.Pool, Log: a.Log})
		a.JobReg.Every(6*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return notify.CleanupArgs{}, nil }, true)
	})
}

// notifyMailer returns the current system mailer: the System-stream sender
// is wired by a start hook after workers and operations are registered.
func (a *App) notifyMailer() mailer.Sender {
	if a.Mail == nil {
		return mailer.Unavailable{}
	}
	return a.Mail
}

// notifyPublicURL is the instance's public URL (instance settings, then config).
func (a *App) notifyPublicURL(ctx context.Context) string {
	if st, err := a.Q.GetInstanceSettings(ctx); err == nil && st.PublicUrl != "" {
		return st.PublicUrl
	}
	return a.Cfg.HTTP.PublicURL
}
