package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
	"github.com/hostrepute/reputecast/internal/transactional/importer"
)

// Importing transactional templates (X03): the Import templates dialog, and
// the templates a Notifuse contact import brings along.
func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		importer.Register(h, templateImporter(a))
	})
}

// templateImporter reads the providers through the SSRF guard. In
// development the sample accounts answer from their recordings; tests
// never leave the recordings.
func templateImporter(a *App) *importer.Service {
	guard := netguard.Policy{AllowPrivate: a.Cfg != nil && a.Cfg.Providers.AllowPrivateNetworks}
	hc := guard.Client(60 * time.Second)
	samples := false
	switch {
	case a.Cfg != nil && a.Cfg.Env == "test":
		hc.Transport, samples = importer.RecordedTransport(), true
	case a.Cfg != nil && a.Cfg.IsDevelopment():
		hc.Transport, samples = importer.SampleTransport(hc.Transport), true
	}
	s := &importer.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit,
		Drafts:  transactionalService(a),
		Saved:   &templates.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
		HTTP:    hc,
		Guard:   guard,
		Samples: samples,
	}
	if a.Secrets != nil {
		s.Box = a.Secrets
	}
	return s
}

// notifuseTemplates is the contact importer's TemplateImporter.
type notifuseTemplates struct{ a *App }

// ImportNotifuseTemplates implements imports.TemplateImporter.
func (n notifuseTemplates) ImportNotifuseTemplates(ctx context.Context, ws *store.Workspace, by *uuid.UUID, baseURL, apiKey string) (imports.ImportTemplateStats, error) {
	st, err := templateImporter(n.a).ImportNotifuse(ctx, ws, by, baseURL, apiKey)
	return imports.ImportTemplateStats{Transactional: st.Transactional, Saved: st.Saved, Manual: st.Manual, Skipped: st.Skipped}, err
}
