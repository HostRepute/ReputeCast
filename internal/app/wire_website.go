package app

import (
	"context"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
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
		// Publishing a post may need approval (Settings › Approvals).
		Approvals: func() website.Approvals { return approvals.WebsiteGate{S: newApprovals(a)} },
		// "Send as a campaign" makes a campaign draft of a web-only post.
		Campaigns: func() website.CampaignDrafts {
			return &campaigns.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, AfterChange: CampaignChangeHook(a)}
		},
		Certs: func() website.CertChecker { return websiteCerts(a) },
	}
	v, _ := websiteServices.LoadOrStore(a, s)
	return v.(*website.Service)
}

// websiteCerts checks the certificate the custom domain serves, through the
// outbound guard. In development and tests the demo's domains answer
// offline, and nothing else is dialled in tests.
func websiteCerts(a *App) website.CertChecker {
	var next website.CertChecker
	policy := netguard.Policy{}
	if a.Cfg != nil {
		policy.AllowPrivate = a.Cfg.Providers.AllowPrivateNetworks
	}
	next = website.TLSChecker{Policy: policy, Timeout: 8 * time.Second}
	if a.Cfg == nil || a.Cfg.Env == "test" {
		return demo.WebsiteCertOverlay(nil)
	}
	if a.Cfg.IsDevelopment() {
		return demo.WebsiteCertOverlay(next)
	}
	return next
}

// Website is the website domain (tests set its clock).
func (a *App) Website() *website.Service { return websiteService(a) }

// websiteReadiness gives Send Readiness the web checks of a campaign that
// publishes to the website with the send.
type websiteReadiness struct{ a *App }

// WebFacts implements readiness.WebsiteChecker.
func (w websiteReadiness) WebFacts(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) readiness.WebFacts {
	s := websiteService(w.a)
	p := systemPrincipal(ws, authz.CampaignsView, authz.WebsiteView)
	web, err := s.CampaignWeb(ctx, ws, p, campaignID)
	if err != nil || !web.SiteOn || !web.Post.WebOn || web.Post.Status == website.PostPublished || web.Post.Status == website.PostUnpublished {
		return readiness.WebFacts{}
	}
	if web.Post.PublishMode != website.WhenWithSend {
		return readiness.WebFacts{}
	}
	out := readiness.WebFacts{Ran: true}
	href := "/w/" + ws.Slug + "/campaigns/" + campaignID.String() + "/edit?panel=web"
	for _, c := range web.Post.Checks {
		title, detail := website.CheckText(ctx, c)
		out.Checks = append(out.Checks, readiness.WebCheck{Key: c.Key, Status: c.Status, Title: title, Detail: detail, Href: href})
	}
	return out
}

func init() {
	readinessWebsite = func(a *App) readiness.WebsiteChecker { return websiteReadiness{a} }
	approvalsWebsite = func(a *App, s *approvals.Service) {
		s.Website = func() approvals.WebsitePosts { return websiteService(a) }
		// The approval review's web preview, drawn by the public pages.
		s.WebsitePreview = func(ctx context.Context, ws *store.Workspace, postID uuid.UUID) (string, error) {
			return a.publicPages().ApprovalPreview(ctx, ws, postID)
		}
	}
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		website.Register(h, websiteService(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		website.RegisterJobs(a.JobReg, websiteService(a))
	})
}
