package app

import (
	"sync"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/website"
)

// The publication website (design expansion phase 7, X18): its settings,
// posts, tags, members and sign-in, and the data of the site's pages, which
// internal/publicpages renders (wire_forms.go gives them the service).

// The service is shared by the API, the public pages and the jobs of one
// App (tests set its clock).
var websiteServices sync.Map // *App → *website.Service

func websiteService(a *App) *website.Service {
	if v, ok := websiteServices.Load(a); ok {
		return v.(*website.Service)
	}
	s := &website.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, URLs: a.URLs,
		Forms: a.formsService(), Audience: a.audience(),
		Earn: func() *earn.Service { return earnService(a) },
		Mail: func() mailer.Sender { return a.Mail },
		Jobs: func() website.Inserter {
			if a.Jobs == nil {
				return nil
			}
			return a.Jobs
		},
	}
	v, _ := websiteServices.LoadOrStore(a, s)
	return v.(*website.Service)
}

// Website is the website domain (tests set its clock).
func (a *App) Website() *website.Service { return websiteService(a) }

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		_ = websiteService(a)
	})
}
