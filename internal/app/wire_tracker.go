package app

import (
	"net/http"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/tracker"
)

// The site tracker (M62 G-28): Settings › Integrations › Website tracking,
// the script and its collect endpoint, the contact timeline's site lines
// and the retention job. The click token on links to tracked sites and the
// hub card are wired with the stores' (wire_commerce.go).

var trackerServices sync.Map // *App → *tracker.Service

// Tracker is the site tracker (tests set its clock and flush its writer).
func (a *App) Tracker() *tracker.Service {
	if v, ok := trackerServices.Load(a); ok {
		return v.(*tracker.Service)
	}
	s := &tracker.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Links: a.Links, URLs: a.URLs, Log: a.Log,
		// A host a workspace holds carries only its endpoints; its pages
		// (and the instance's) are never tracked.
		Hosts: a.customHosts(),
		Carries: func(r *http.Request, slug string) bool {
			ok, err := a.customHosts().CarriesSlug(r.Context(), r.Host, slug)
			return err == nil && ok
		},
	}
	if a.Cfg != nil {
		cfg := a.Cfg
		s.Development = cfg.IsDevelopment
	}
	v, _ := trackerServices.LoadOrStore(a, s)
	return v.(*tracker.Service)
}

func init() {
	// A line per day and site on the contact timeline.
	audienceAnswers = chainActivity(audienceAnswers, func(a *App) audience.ActivitySource { return a.Tracker() })
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		tracker.Register(h, a.Tracker())
	})
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		a.Tracker().Routes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		tracker.RegisterJobs(a.JobReg, a.Tracker())
	})
}
