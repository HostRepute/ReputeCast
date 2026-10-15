package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// App languages: Language & region, the pseudo-locale.

func init() {
	startHooks = append(startHooks, func(a *App) error {
		i18n.SetPseudo(a.Cfg.PseudoLocale())
		return nil
	})
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		applang.Register(h, &applang.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
	})
}
