package website

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// WebsiteCertificate is the certificate the custom domain serves.
type WebsiteCertificate struct {
	Status   string     `json:"status" enum:"valid,failing,pending,unknown,none"`
	Issuer   string     `json:"issuer,omitempty"`
	RenewsAt *time.Time `json:"renewsAt,omitempty"`
	Detail   string     `json:"detail,omitempty"`
}

// WebsiteAddress is where the site runs.
type WebsiteAddress struct {
	// Domain is the custom domain as typed (any status).
	Domain       string     `json:"domain"`
	DomainStatus string     `json:"domainStatus" enum:"none,pending,verified,failed"`
	DomainDetail string     `json:"domainDetail"`
	CheckedAt    *time.Time `json:"checkedAt,omitempty"`
	VerifiedAt   *time.Time `json:"verifiedAt,omitempty"`
	// Target is the host the domain's CNAME must point at.
	Target string `json:"target"`
	// Host is the site's address without the scheme: the custom domain
	// once verified, else the instance's /p/<workspace>.
	Host string `json:"host"`
	URL  string `json:"url"`
	// Fallback is the instance's address of the site (until the domain checks out).
	Fallback    string             `json:"fallback"`
	Certificate WebsiteCertificate `json:"certificate"`
}

// WebsiteProgressStep is one step of turning the website on.
type WebsiteProgressStep struct {
	Done  int64    `json:"done"`
	Total int64    `json:"total"`
	Names []string `json:"names,omitempty"`
}

// WebsiteProgress is turning the website on, step by step.
type WebsiteProgress struct {
	StartedAt  *time.Time          `json:"startedAt,omitempty"`
	FinishedAt *time.Time          `json:"finishedAt,omitempty"`
	Tags       WebsiteProgressStep `json:"tags"`
	Posts      WebsiteProgressStep `json:"posts"`
	Redirects  WebsiteProgressStep `json:"redirects"`
	// Pages is the home page, sign-in and the account: waiting or done.
	Pages string `json:"pages" enum:"waiting,done"`
}

// WebsiteNumbers are the site's numbers for the last 30 days.
type WebsiteNumbers struct {
	Views    int64     `json:"views"`
	Visitors int64     `json:"visitors"`
	Signups  int64     `json:"signups"`
	Upgrades int64     `json:"upgrades"`
	From     time.Time `json:"from"`
	Until    time.Time `json:"until"`
}

// WebsiteArchiveList is a list in the archive.
type WebsiteArchiveList struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Issues   int64  `json:"issues"`
	Public   bool   `json:"public"`
	Archived bool   `json:"archived"`
}

// WebsiteArchiveFacts are the archive as it stands: what turning the
// website on makes of it.
type WebsiteArchiveFacts struct {
	Enabled bool                 `json:"enabled"`
	Index   bool                 `json:"index"`
	Issues  int64                `json:"issues"`
	Since   int                  `json:"since"`
	URL     string               `json:"url"`
	Feed    string               `json:"feed"`
	Lists   []WebsiteArchiveList `json:"lists"`
	// Tags are the lists with archived campaigns: the tags turning on makes.
	Tags []string `json:"tags"`
}

// WebsiteMembersFacts are the members and sign-up settings in short.
type WebsiteMembersFacts struct {
	SigninOn    bool                `json:"signinOn"`
	SessionDays int                 `json:"sessionDays"`
	LinkMinutes int                 `json:"linkMinutes"`
	SignupOn    bool                `json:"signupOn"`
	SignupLists []WebsiteAccessList `json:"signupLists"`
	// SignedIn are members who signed in since the website was turned on;
	// LastWeek in the last 7 days; Paid the paid members.
	SignedIn int64 `json:"signedIn"`
	LastWeek int64 `json:"lastWeek"`
	Paid     int64 `json:"paid"`
}

// WebsiteOffFacts are what goes offline when the website is turned off.
type WebsiteOffFacts struct {
	WebOnly     int64 `json:"webOnly"`
	PaidMembers int64 `json:"paidMembers"`
	// PublicPages and Shop stay online (the page builder fills them).
	PublicPages int64 `json:"publicPages"`
	Shop        bool  `json:"shop"`
}

// WebsiteOverview is the Website tab's header and its off and turning-on states.
type WebsiteOverview struct {
	Status     string              `json:"status" enum:"off,turning_on,on"`
	Address    WebsiteAddress      `json:"address"`
	TurnedOnAt *time.Time          `json:"turnedOnAt,omitempty"`
	TurnedOnBy string              `json:"turnedOnBy,omitempty"`
	Progress   *WebsiteProgress    `json:"progress,omitempty"`
	Numbers    WebsiteNumbers      `json:"numbers"`
	Posts      WebsitePostCounts   `json:"posts"`
	Archive    WebsiteArchiveFacts `json:"archive"`
	Members    WebsiteMembersFacts `json:"members"`
	Off        WebsiteOffFacts     `json:"off"`
	HomeLayout string              `json:"homeLayout" enum:"featured,list,grid"`
	Title      string              `json:"title"`
	UpdatedAt  time.Time           `json:"updatedAt"`
	UpdatedBy  string              `json:"updatedBy,omitempty"`
	// Helpers are whom to ask for what the caller's role lacks ("Ask Sofia
	// Rossi, Marcus Chen or Linh Nguyen"); empty for what it holds.
	Helpers WebsiteHelpers `json:"helpers"`
}

// WebsiteHelpers are the members holding website:manage and website:publish.
type WebsiteHelpers struct {
	Manage  []string `json:"manage"`
	Publish []string `json:"publish"`
}

// Overview returns the Website tab's header and states.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace, p *authz.Principal) (WebsiteOverview, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return WebsiteOverview{}, err
	}
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return WebsiteOverview{}, err
	}
	out := WebsiteOverview{Status: c.row.Status, TurnedOnAt: c.row.TurnedOnAt, HomeLayout: c.row.HomeLayout, Title: siteTitle(c, set),
		UpdatedAt: c.row.UpdatedAt}
	if by, err := s.Q.WebsiteSettingsUpdatedBy(ctx, ws.ID); err == nil {
		out.UpdatedBy = by
	}
	if c.row.TurnedOnBy != nil {
		if u, err := s.Q.GetUser(ctx, *c.row.TurnedOnBy); err == nil {
			out.TurnedOnBy = u.Name
		}
	}
	out.Address = s.address(ctx, ws, false)
	if c.row.Status == StatusTurningOn || (c.row.Status == StatusOn && len(c.row.Progress) > 2) {
		var pr WebsiteProgress
		if json.Unmarshal(c.row.Progress, &pr) == nil {
			out.Progress = &pr
		}
	}
	now := s.now()
	until := day(now, location(ws))
	from := until.AddDate(0, 0, -29)
	tot, err := s.Q.WebsiteViewTotals(ctx, store.WebsiteViewTotalsParams{WorkspaceID: ws.ID, FromDay: from, UntilDay: until})
	if err != nil {
		return out, err
	}
	conv, err := s.Q.WebsiteConversionTotals(ctx, store.WebsiteConversionTotalsParams{WorkspaceID: ws.ID, FromDay: from, UntilDay: until.AddDate(0, 0, 1)})
	if err != nil {
		return out, err
	}
	out.Numbers = WebsiteNumbers{Views: tot.Views, Visitors: tot.Visitors, Signups: conv.Signups, Upgrades: conv.Upgrades, From: from, Until: until}
	if out.Posts, err = s.counts(ctx, ws); err != nil {
		return out, err
	}
	if out.Archive, err = s.archiveFacts(ctx, ws); err != nil {
		return out, err
	}
	if out.Members, err = s.membersFacts(ctx, ws, c); err != nil {
		return out, err
	}
	out.Off = WebsiteOffFacts{WebOnly: out.Posts.WebOnly, PaidMembers: out.Members.Paid}
	out.Helpers = WebsiteHelpers{Manage: []string{}, Publish: []string{}}
	if p == nil || !p.Can(authz.WebsiteManage) {
		out.Helpers.Manage = s.helpers(ctx, ws, authz.WebsiteManage)
	}
	if p == nil || !p.Can(authz.WebsitePublish) || !p.Can(authz.WebsiteEdit) {
		out.Helpers.Publish = s.helpers(ctx, ws, authz.WebsitePublish)
	}
	return out, nil
}

// address is where the site runs and the state of its domain and certificate.
func (s *Service) address(ctx context.Context, ws *store.Workspace, force bool) WebsiteAddress {
	out := WebsiteAddress{DomainStatus: "none", Certificate: WebsiteCertificate{Status: "none"}}
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return out
	}
	domain, status, detail := set.DomainStateIn(i18n.LangFrom(ctx))
	out.Domain, out.DomainDetail = domain, detail
	if status != "" {
		out.DomainStatus = status
	}
	out.CheckedAt, out.VerifiedAt = set.DomainDates()
	out.Target = instanceHostOf(s.URLs.Base())
	l := s.Forms.Links(ws, set)
	inst := strings.TrimRight(s.URLs.Base(), "/") + "/p/" + ws.Slug
	out.Fallback = hostOnly(inst)
	if l.Custom {
		out.URL = l.Origin() + "/"
		out.Host = hostOnly(l.Origin())
	} else {
		out.URL = inst + "/"
		out.Host = out.Fallback
	}
	switch {
	case domain == "":
		out.Certificate.Status = "none"
	case out.DomainStatus != "verified":
		out.Certificate.Status = "pending"
	default:
		ci := s.cert(ctx, domain, force)
		out.Certificate = WebsiteCertificate{Status: ci.Status, Issuer: ci.Issuer, Detail: ci.Detail}
		if !ci.NotAfter.IsZero() {
			// ACME certificates renew 30 days before they expire.
			r := ci.NotAfter.AddDate(0, 0, -30)
			out.Certificate.RenewsAt = &r
		}
	}
	return out
}

func instanceHostOf(base string) string {
	h := hostOnly(strings.TrimRight(base, "/"))
	if i := strings.IndexAny(h, ":/"); i >= 0 {
		h = h[:i]
	}
	return h
}

// archiveFacts is the archive as it stands.
func (s *Service) archiveFacts(ctx context.Context, ws *store.Workspace) (WebsiteArchiveFacts, error) {
	out := WebsiteArchiveFacts{Lists: []WebsiteArchiveList{}, Tags: []string{}}
	set, err := s.Forms.LoadSettings(ctx, ws)
	if err != nil {
		return out, err
	}
	if err := s.Forms.SyncArchive(ctx, ws, set); err != nil {
		s.log().WarnContext(ctx, "sync archive", "err", err)
	}
	out.Enabled, out.Index = set.Pages.Archive.Enabled, set.Pages.Archive.Index
	l := s.Forms.Links(ws, set)
	out.URL, out.Feed = l.Archive(), l.Feed()
	archived, err := s.Forms.ArchiveLists(ctx, s.Q, ws, set)
	if err != nil {
		return out, err
	}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return out, err
	}
	allIDs := make([]uuid.UUID, 0, len(lists))
	for _, li := range lists {
		allIDs = append(allIDs, li.ID)
	}
	counts, err := s.Q.ArchiveListCounts(ctx, store.ArchiveListCountsParams{WorkspaceID: ws.ID, Lists: allIDs})
	if err != nil {
		return out, err
	}
	issues := map[uuid.UUID]int64{}
	for _, c := range counts {
		issues[c.ListID] = c.Issues
	}
	sum, err := s.Q.ArchiveSummary(ctx, store.ArchiveSummaryParams{WorkspaceID: ws.ID, Lists: archived})
	if err != nil {
		return out, err
	}
	out.Issues, out.Since = sum.Issues, sum.FirstAt.Year()
	for _, li := range lists {
		isArchived := false
		for _, a := range archived {
			if a == li.ID {
				isArchived = true
			}
		}
		out.Lists = append(out.Lists, WebsiteArchiveList{ID: li.ID.String(), Name: li.Name, Issues: issues[li.ID], Public: li.Public, Archived: isArchived})
		if isArchived && issues[li.ID] > 0 {
			out.Tags = append(out.Tags, li.Name)
		}
	}
	return out, nil
}

// membersFacts are the members and the sign-up settings in short.
func (s *Service) membersFacts(ctx context.Context, ws *store.Workspace, c *config) (WebsiteMembersFacts, error) {
	out := WebsiteMembersFacts{SigninOn: c.row.SigninOn, SessionDays: int(c.row.SessionDays), LinkMinutes: int(c.row.LinkMinutes), SignupOn: c.row.SignupOn,
		SignupLists: []WebsiteAccessList{}}
	lists, err := s.accessLists(ctx, ws, c)
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		if l.Offered {
			out.SignupLists = append(out.SignupLists, l)
		}
	}
	since := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if c.row.TurnedOnAt != nil {
		since = *c.row.TurnedOnAt
	}
	m, err := s.Q.WebsiteMemberCounts(ctx, store.WebsiteMemberCountsParams{WorkspaceID: ws.ID, Since: since, Week: s.now().AddDate(0, 0, -7)})
	if err != nil {
		return out, err
	}
	out.SignedIn, out.LastWeek = m.Since, m.Week
	if err := s.Pool.QueryRow(ctx, `SELECT count(DISTINCT contact_id) FROM paid_subscriptions
		WHERE workspace_id = $1 AND mode = 'live' AND ended_at IS NULL
		  AND (status IN ('trialing','active','complimentary','gift') OR (status = 'past_due' AND NOT access_paused))`, ws.ID).Scan(&out.Paid); err != nil {
		return out, err
	}
	return out, nil
}
