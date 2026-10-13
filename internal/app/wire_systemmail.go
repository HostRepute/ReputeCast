package app

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/systemmail"
)

// W10 · System mail: App.Mail sends through the System stream's route (see
// internal/systemmail). In development the demo's sample providers deliver
// to Mailpit (mail.dev_smtp).

// deliverer sends one message through a provider and records it.
func (a *App) deliverer() *systemmail.Deliverer {
	d := &systemmail.Deliverer{Pool: a.Pool, Q: a.Q, Factory: a.providerFactory(), Log: a.Log}
	if a.Cfg != nil && a.Cfg.IsDevelopment() {
		d.DevSMTP = a.Cfg.Mail.DevSMTP
	}
	if a.Pool != nil {
		n := a.normalizer()
		d.ApplyEvent = func(ctx context.Context, tx pgx.Tx, p store.Provider, ev providers.Event) error {
			_, err := n.Apply(ctx, tx, p, nil, "api", time.Now(), ev)
			return err
		}
	}
	return d
}

// instanceBrand is the instance name and public URL for system mail.
func (a *App) instanceBrand(ctx context.Context) (string, string) {
	name := ""
	if st, err := a.Q.GetInstanceSettings(ctx); err == nil {
		name = st.Name
	}
	return name, a.URLs.Base()
}

func init() {
	startHooks = append(startHooks, func(a *App) error {
		a.Mail = &systemmail.Mailer{Q: a.Q, Deliver: a.deliverer(), Instance: a.instanceBrand, Log: a.Log}
		return nil
	})
}

// forgetSystemAvailability drops the system mailer's cached availability
// after routing or providers change.
func (a *App) forgetSystemAvailability() {
	if m, ok := a.Mail.(*systemmail.Mailer); ok {
		m.Forget()
	}
}
