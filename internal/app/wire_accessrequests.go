package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/accessrequests"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		accessrequests.Register(h, &accessrequests.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
	})
}
