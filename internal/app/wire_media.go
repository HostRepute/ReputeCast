package app

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/templates"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		media.Register(h, &media.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
	})
	// Images are public: email clients fetch them without a session.
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		s := &media.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Static: templates.StarterAssets()}
		r.Handle(media.PathPrefix+"*", s)
	})
}
