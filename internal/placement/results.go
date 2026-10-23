package placement

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/placement/seedbox"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// PlacementNotChecked is a seed whose copy couldn't be looked for.
type PlacementNotChecked struct {
	Provider string `json:"provider"`
	Name     string `json:"name" doc:"The seed's mailbox provider (iCloud)"`
	Email    string `json:"email"`
	Reason   string `json:"reason" enum:"signin_failed,signin_expired,seed_removed,stopped,unsent"`
}

// PlacementCounts are copies by where they landed.
type PlacementCounts struct {
	// Copies went out or are about to; Found were found in a seed.
	Copies     int `json:"copies"`
	Found      int `json:"found"`
	Inbox      int `json:"inbox"`
	Tab        int `json:"tab"`
	Spam       int `json:"spam"`
	NotArrived int `json:"notArrived"`
	NotChecked int `json:"notChecked"`
	// Waiting were sent (or are queued) and aren't found yet; Failed the
	// provider refused; Unsent never went because the test was stopped.
	Waiting int `json:"waiting"`
	Failed  int `json:"failed"`
	Unsent  int `json:"unsent"`
	// Tabs are the inbox tabs copies landed in ("Promotions").
	Tabs            []string              `json:"tabs"`
	NotCheckedSeeds []PlacementNotChecked `json:"notCheckedSeeds"`
}

// countCopies counts copies, of one provider when pid is set.
func countCopies(copies []store.PlacementCopy, pid *uuid.UUID) PlacementCounts {
	c := PlacementCounts{Tabs: []string{}, NotCheckedSeeds: []PlacementNotChecked{}}
	for _, x := range copies {
		if pid != nil && (x.ProviderID == nil || *x.ProviderID != *pid) {
			continue
		}
		switch x.Status {
		case "queued", "sent":
			c.Copies++
			c.Waiting++
		case "failed":
			c.Copies++
			c.Failed++
		case "found":
			c.Copies++
			c.Found++
			switch x.Folder {
			case seedbox.RoleSpam:
				c.Spam++
			case seedbox.RoleTab:
				c.Tab++
				if x.Tab != "" && !slices.Contains(c.Tabs, x.Tab) {
					c.Tabs = append(c.Tabs, x.Tab)
				}
			default:
				c.Inbox++
			}
		case "not_arrived":
			c.Copies++
			c.NotArrived++
		case "not_checked":
			if x.NotChecked == ReasonUnsent {
				c.Unsent++
				continue
			}
			if x.HandedAt != nil {
				c.Copies++
			}
			c.NotChecked++
			p, _ := seedbox.ProviderOf(x.SeedProvider)
			nc := PlacementNotChecked{Provider: x.SeedProvider, Name: p.Name, Email: x.SeedEmail, Reason: x.NotChecked}
			if !slices.ContainsFunc(c.NotCheckedSeeds, func(o PlacementNotChecked) bool { return o.Email == nc.Email }) {
				c.NotCheckedSeeds = append(c.NotCheckedSeeds, nc)
			}
		}
	}
	return c
}

// PlacementTiming is a copy's time to arrive, with where it went.
type PlacementTiming struct {
	Ms               int64  `json:"ms"`
	SeedProvider     string `json:"seedProvider"`
	SeedProviderName string `json:"seedProviderName"`
	Folder           string `json:"folder" enum:"inbox,tab,spam"`
	FolderName       string `json:"folderName"`
}

// PlacementAuthCount is one check across copies.
type PlacementAuthCount struct {
	Pass int `json:"pass"`
	Fail int `json:"fail"`
}

// PlacementAuthSummary is SPF, DKIM and DMARC across a provider's copies,
// from the mailboxes' own Authentication-Results.
type PlacementAuthSummary struct {
	// Checked copies carried Authentication-Results.
	Checked int                `json:"checked"`
	SPF     PlacementAuthCount `json:"spf"`
	DKIM    PlacementAuthCount `json:"dkim"`
	DMARC   PlacementAuthCount `json:"dmarc"`
	// The signing domain and selector, and the SPF domain, most copies showed.
	DKIMDomain   string `json:"dkimDomain,omitempty"`
	DKIMSelector string `json:"dkimSelector,omitempty"`
	SPFDomain    string `json:"spfDomain,omitempty"`
}

// AllPass reports whether every checked copy passed all three.
func (a PlacementAuthSummary) AllPass() bool {
	return a.Checked > 0 && a.SPF.Fail == 0 && a.DKIM.Fail == 0 && a.DMARC.Fail == 0
}

func passed(r string) bool { return r == "pass" }

func failed(r string) bool {
	switch r {
	case "fail", "softfail", "permerror", "temperror", "none", "neutral", "policy":
		return true
	}
	return false
}

func authOf(copies []store.PlacementCopy) PlacementAuthSummary {
	a := PlacementAuthSummary{}
	dk, dks, spf := map[string]int{}, map[string]int{}, map[string]int{}
	for _, c := range copies {
		if c.Status != "found" {
			continue
		}
		var r seedbox.Auth
		if json.Unmarshal(c.Auth, &r) != nil || (r.SPF == nil && r.DKIM == nil && r.DMARC == nil) {
			continue
		}
		a.Checked++
		count := func(ch *seedbox.Check, into *PlacementAuthCount) {
			switch {
			case ch == nil:
			case passed(ch.Result):
				into.Pass++
			case failed(ch.Result):
				into.Fail++
			}
		}
		count(r.SPF, &a.SPF)
		count(r.DKIM, &a.DKIM)
		count(r.DMARC, &a.DMARC)
		if r.DKIM != nil && r.DKIM.Domain != "" {
			dk[r.DKIM.Domain]++
			if r.DKIM.Selector != "" {
				dks[r.DKIM.Domain+"\x00"+r.DKIM.Selector]++
			}
		}
		if r.SPF != nil && r.SPF.Domain != "" {
			spf[r.SPF.Domain]++
		}
	}
	a.DKIMDomain, a.SPFDomain = top(dk), top(spf)
	if sel := top(dks); sel != "" {
		_, a.DKIMSelector, _ = strings.Cut(sel, "\x00")
	}
	return a
}

func top(m map[string]int) string {
	best, n := "", 0
	for k, v := range m {
		if v > n || (v == n && k < best) {
			best, n = k, v
		}
	}
	return best
}

// PlacementProviderResult is one provider's copies in a test.
type PlacementProviderResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Short   string `json:"short"`
	Adapter string `json:"adapter"`
	Color   string `json:"color"`
	Detail  string `json:"detail"`
	IP      string `json:"ip,omitempty"`
	// Status and warm-up day as they were when the test started.
	Status   string               `json:"status"`
	WarmDay  int                  `json:"warmDay,omitempty"`
	WarmDays int                  `json:"warmDays,omitempty"`
	Counts   PlacementCounts      `json:"counts"`
	MedianMs *int64               `json:"medianMs,omitempty"`
	Fastest  *PlacementTiming     `json:"fastest,omitempty"`
	Slowest  *PlacementTiming     `json:"slowest,omitempty"`
	Auth     PlacementAuthSummary `json:"auth"`
	Cost     float64              `json:"cost"`
}

func arriveMs(c store.PlacementCopy) *int64 {
	if c.HandedAt == nil || c.ReceivedAt == nil {
		return nil
	}
	ms := max(c.ReceivedAt.Sub(*c.HandedAt).Milliseconds(), 0)
	return &ms
}

func snapshots(t store.PlacementTest) map[string]providerSnapshot {
	m := map[string]providerSnapshot{}
	_ = json.Unmarshal(t.Providers, &m)
	return m
}

func providerResults(t store.PlacementTest, copies []store.PlacementCopy) []PlacementProviderResult {
	snap := snapshots(t)
	out := []PlacementProviderResult{}
	for _, pid := range t.ProviderIds {
		ps := snap[pid.String()]
		r := PlacementProviderResult{ID: pid.String(), Name: ps.Name, Short: ps.Short, Adapter: ps.Adapter, Color: state.Color(ps.Adapter), Detail: ps.Detail,
			IP: ps.IP, Status: ps.Status, WarmDay: ps.WarmDay, WarmDays: ps.WarmDays, Counts: countCopies(copies, &pid)}
		mine := []store.PlacementCopy{}
		var times []PlacementTiming
		for _, c := range copies {
			if c.ProviderID == nil || *c.ProviderID != pid {
				continue
			}
			mine = append(mine, c)
			if c.Status != "found" {
				continue
			}
			if ms := arriveMs(c); ms != nil {
				p, _ := seedbox.ProviderOf(c.SeedProvider)
				times = append(times, PlacementTiming{Ms: *ms, SeedProvider: c.SeedProvider, SeedProviderName: p.Name, Folder: c.Folder, FolderName: c.FolderName})
			}
		}
		r.Cost = float64(r.Counts.Copies) * ps.Cost
		r.Auth = authOf(mine)
		if len(times) > 0 {
			sort.SliceStable(times, func(i, j int) bool { return times[i].Ms < times[j].Ms })
			med := times[len(times)/2].Ms
			if len(times)%2 == 0 {
				med = (times[len(times)/2-1].Ms + times[len(times)/2].Ms) / 2
			}
			r.MedianMs = &med
			f, s := times[0], times[len(times)-1]
			r.Fastest, r.Slowest = &f, &s
		}
		out = append(out, r)
	}
	return out
}

// PlacementTestRow is a test in the list and the trend.
type PlacementTestRow struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	ContentKind    string     `json:"contentKind" enum:"campaign,template,digest,transactional"`
	ContentID      string     `json:"contentId,omitempty"`
	ContentName    string     `json:"contentName"`
	ContentNote    string     `json:"contentNote" doc:"draft, scheduled, sent, template or the transactional template's key"`
	ContentKey     string     `json:"contentKey,omitempty"`
	ContentVersion int        `json:"contentVersion,omitempty"`
	ContentAt      *time.Time `json:"contentAt,omitempty" doc:"When the campaign is scheduled or was sent"`
	// Personalization is the test data set ("" for the fallbacks).
	Personalization string                    `json:"personalization"`
	Origin          string                    `json:"origin" enum:"placement,review,composer,warmup,template"`
	Through         string                    `json:"through" enum:"route,provider,compare"`
	Status          string                    `json:"status" enum:"sending,waiting,finished,stopped"`
	StartedAt       time.Time                 `json:"startedAt"`
	StartedBy       string                    `json:"startedBy"`
	StartedByID     string                    `json:"startedById,omitempty"`
	StartedByColor  string                    `json:"startedByColor,omitempty"`
	LastSentAt      *time.Time                `json:"lastSentAt,omitempty"`
	WaitUntil       *time.Time                `json:"waitUntil,omitempty" doc:"Copies missing then are Not arrived"`
	LateUntil       *time.Time                `json:"lateUntil,omitempty" doc:"Late copies found until then update the result"`
	FinishedAt      *time.Time                `json:"finishedAt,omitempty"`
	Providers       []PlacementProviderResult `json:"providers"`
	Counts          PlacementCounts           `json:"counts"`
}

func testRow(l i18n.Lang, t store.PlacementTest, startedBy, color string, copies []store.PlacementCopy) PlacementTestRow {
	r := PlacementTestRow{ID: t.ID.String(), Name: NameIn(l, t.Name, t.ContentName), ContentKind: t.ContentKind, ContentName: t.ContentName, ContentNote: t.ContentNote,
		ContentKey: t.ContentKey, ContentVersion: int(t.ContentVersion), ContentAt: t.ContentAt, Personalization: t.Personalization, Origin: t.Origin,
		Through: t.Through, Status: t.Status, StartedAt: t.StartedAt, StartedBy: startedBy, StartedByColor: color, LastSentAt: t.LastSentAt, WaitUntil: t.WaitUntil,
		LateUntil: t.LateUntil, FinishedAt: t.FinishedAt, Providers: providerResults(t, copies), Counts: countCopies(copies, nil)}
	if t.ContentID != nil {
		r.ContentID = t.ContentID.String()
	}
	if t.StartedBy != nil {
		r.StartedByID = t.StartedBy.String()
	}
	return r
}

// PlacementFacet is a filter choice of the list.
type PlacementFacet struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PlacementTestList is Deliverability › Placement tests.
type PlacementTestList struct {
	Tests []PlacementTestRow `json:"tests"`
	// Seeds: connected and all, and the ones whose sign-in failed or expired.
	SeedsConnected int                 `json:"seedsConnected"`
	SeedsTotal     int                 `json:"seedsTotal"`
	Attention      []PlacementSeedView `json:"attention"`
	DeleteAfter    string              `json:"deleteAfter" enum:"24h,finished,never"`
	// Filters: the providers tests went through, and who started them.
	Providers     []PlacementFacet `json:"providers"`
	Starters      []PlacementFacet `json:"starters"`
	CanRun        bool             `json:"canRun"`
	RunHelpers    []string         `json:"runHelpers"`
	CanManage     bool             `json:"canManage"`
	ManageHelpers []string         `json:"manageHelpers"`
}

// Tests lists the workspace's tests, newest first (every one started by a person).
func (s *Service) Tests(ctx context.Context, ws *store.Workspace) (PlacementTestList, error) {
	p := api.PrincipalFrom(ctx)
	out := PlacementTestList{Tests: []PlacementTestRow{}, Attention: []PlacementSeedView{}, Providers: []PlacementFacet{}, Starters: []PlacementFacet{},
		CanRun: p.Can(authz.PlacementRun), RunHelpers: Helpers(ctx, s.Q, ws.ID, authz.PlacementRun),
		CanManage: p.Can(authz.PlacementManage), ManageHelpers: Helpers(ctx, s.Q, ws.ID, authz.PlacementManage)}
	rows, err := s.Q.PlacementTests(ctx, store.PlacementTestsParams{WorkspaceID: ws.ID, Lim: 200})
	if err != nil {
		return out, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.PlacementTest.ID
	}
	all, err := s.Q.PlacementCopiesOfTests(ctx, ids)
	if err != nil {
		return out, err
	}
	byTest := map[uuid.UUID][]store.PlacementCopy{}
	for _, c := range all {
		byTest[c.TestID] = append(byTest[c.TestID], c)
	}
	seenP, seenS := map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		row := testRow(i18n.LangFrom(ctx), r.PlacementTest, r.StartedByName, r.StartedByColor, byTest[r.PlacementTest.ID])
		out.Tests = append(out.Tests, row)
		for _, pr := range row.Providers {
			if !seenP[pr.ID] {
				seenP[pr.ID] = true
				out.Providers = append(out.Providers, PlacementFacet{ID: pr.ID, Name: pr.Name})
			}
		}
		if row.StartedByID != "" && !seenS[row.StartedByID] {
			seenS[row.StartedByID] = true
			out.Starters = append(out.Starters, PlacementFacet{ID: row.StartedByID, Name: row.StartedBy})
		}
	}
	sort.SliceStable(out.Providers, func(i, j int) bool { return out.Providers[i].Name < out.Providers[j].Name })
	sort.SliceStable(out.Starters, func(i, j int) bool { return out.Starters[i].Name < out.Starters[j].Name })
	seeds, err := s.Q.PlacementSeeds(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, sd := range seeds {
		out.SeedsTotal++
		switch sd.PlacementSeed.Status {
		case "connected":
			out.SeedsConnected++
		case "failed", "expired":
			out.Attention = append(out.Attention, seedView(sd.PlacementSeed, sd.AddedByName))
		}
	}
	if out.DeleteAfter, err = s.Q.PlacementSettings(ctx, ws.ID); err != nil {
		return out, err
	}
	return out, nil
}

// PlacementEvent is a provider's event for a copy.
type PlacementEvent struct {
	Type       string    `json:"type" doc:"delivered, deferred, soft_bounce, hard_bounce, complaint, rejected…"`
	At         time.Time `json:"at"`
	Code       string    `json:"code,omitempty"`
	Diagnostic string    `json:"diagnostic,omitempty"`
	// Host is the receiving server the provider named ("mx-ha03.web.de").
	Host string `json:"host,omitempty"`
}

// PlacementCopyView is one copy: a cell of the matrix.
type PlacementCopyView struct {
	ID               string `json:"id"`
	SeedID           string `json:"seedId,omitempty"`
	SeedEmail        string `json:"seedEmail"`
	SeedProvider     string `json:"seedProvider"`
	SeedProviderName string `json:"seedProviderName"`
	ProviderID       string `json:"providerId,omitempty"`
	ProviderName     string `json:"providerName"`
	Round            int    `json:"round"`
	// First is set in a comparison when this provider's copy went first.
	First      bool       `json:"first"`
	Status     string     `json:"status" enum:"queued,sent,failed,found,not_arrived,not_checked"`
	Folder     string     `json:"folder,omitempty" enum:"inbox,tab,spam,"`
	FolderName string     `json:"folderName,omitempty" doc:"The mailbox's own name: Junk Email, Bulk, Spamverdacht"`
	Tab        string     `json:"tab,omitempty" doc:"Primary, Promotions, Focused, Other"`
	Late       bool       `json:"late"`
	SendAt     time.Time  `json:"sendAt"`
	HandedAt   *time.Time `json:"handedAt,omitempty"`
	ReceivedAt *time.Time `json:"receivedAt,omitempty"`
	ArriveMs   *int64     `json:"arriveMs,omitempty"`
	SPF        string     `json:"spf,omitempty"`
	DKIM       string     `json:"dkim,omitempty"`
	DMARC      string     `json:"dmarc,omitempty"`
	// Reason says why it wasn't looked for; ReasonAt is when the seed's
	// sign-in failed, or when the test stopped.
	Reason   string     `json:"reason,omitempty" enum:"signin_failed,signin_expired,seed_removed,stopped,unsent,"`
	ReasonAt *time.Time `json:"reasonAt,omitempty"`
	// Deferrals and the provider's latest event while it isn't found.
	Deferrals int             `json:"deferrals"`
	LastEvent *PlacementEvent `json:"lastEvent,omitempty"`
	Error     string          `json:"error,omitempty" doc:"Why the provider refused the copy"`
	// LateUntil is how long ReputeCast keeps looking for it.
	LateUntil *time.Time `json:"lateUntil,omitempty"`
}

func copyView(l i18n.Lang, c store.PlacementCopy, t store.PlacementTest, seeds map[uuid.UUID]store.PlacementSeed, events []store.DeliveryEvent) PlacementCopyView {
	p, _ := seedbox.ProviderOf(c.SeedProvider)
	v := PlacementCopyView{ID: c.ID.String(), SeedEmail: c.SeedEmail, SeedProvider: c.SeedProvider, SeedProviderName: p.Name, ProviderName: c.ProviderName,
		Round: int(c.Round), First: t.Through == "compare" && c.Round == 1, Status: c.Status, Folder: c.Folder, FolderName: c.FolderName, Tab: c.Tab,
		Late: c.Late, SendAt: c.SendAt, HandedAt: c.HandedAt, ReceivedAt: c.ReceivedAt, ArriveMs: arriveMs(c), Reason: c.NotChecked, Error: copyError(l, c.Error)}
	if c.SeedID != nil {
		v.SeedID = c.SeedID.String()
	}
	if c.ProviderID != nil {
		v.ProviderID = c.ProviderID.String()
	}
	if c.Status == "found" {
		var r seedbox.Auth
		_ = json.Unmarshal(c.Auth, &r)
		if r.SPF != nil {
			v.SPF = r.SPF.Result
		}
		if r.DKIM != nil {
			v.DKIM = r.DKIM.Result
		}
		if r.DMARC != nil {
			v.DMARC = r.DMARC.Result
		}
	}
	switch c.NotChecked {
	case ReasonSignInFailed, ReasonSignInExpired:
		if c.SeedID != nil {
			if sd, ok := seeds[*c.SeedID]; ok {
				v.ReasonAt = sd.StatusAt
			}
		}
	case ReasonStopped:
		v.ReasonAt = t.StoppedAt
	}
	if c.HandedAt != nil && (c.Status == "not_arrived" || c.Status == "sent") && c.NotChecked == "" {
		at := c.HandedAt.Add(LateWithin)
		v.LateUntil = &at
	}
	for _, e := range events {
		if e.Type == "deferred" || e.Type == "soft_bounce" {
			v.Deferrals++
		}
		if c.Status != "found" && e.Type != "delivered" {
			v.LastEvent = eventView(e)
		}
	}
	return v
}

// mxHost finds the receiving server a diagnostic names.
func mxHost(diag string) string {
	for _, f := range strings.FieldsFunc(diag, func(r rune) bool {
		return r == ' ' || r == '(' || r == ')' || r == '[' || r == ']' || r == ';' || r == ','
	}) {
		f = strings.TrimSuffix(strings.ToLower(f), ".")
		if strings.Count(f, ".") >= 2 && strings.HasPrefix(f, "mx") && !strings.Contains(f, "@") {
			return f
		}
	}
	return ""
}

func eventView(e store.DeliveryEvent) *PlacementEvent {
	host := mxHost(e.Diagnostic)
	diag := e.Diagnostic
	if host != "" {
		// The host shows on its own ("Deferred by mx-ha03.web.de").
		diag = strings.NewReplacer("("+host+")", "", "· "+host, "", host+":", "", host, "").Replace(diag)
		diag = strings.Trim(strings.Join(strings.Fields(diag), " "), " ·:-")
	}
	return &PlacementEvent{Type: e.Type, At: e.OccurredAt, Code: e.OriginalCode, Diagnostic: diag, Host: host}
}

// PlacementResultSeed is a row of the matrix.
type PlacementResultSeed struct {
	ID           string     `json:"id,omitempty"`
	Email        string     `json:"email"`
	Provider     string     `json:"provider"`
	ProviderName string     `json:"providerName"`
	Position     int        `json:"position"`
	Status       string     `json:"status" doc:"The seed's status now (removed when it's gone)"`
	StatusAt     *time.Time `json:"statusAt,omitempty"`
}

// PlacementAuthReadiness is Send Readiness's latest verdict on the campaign.
type PlacementAuthReadiness struct {
	Passed   int      `json:"passed"`
	Warnings int      `json:"warnings"`
	Blocks   int      `json:"blocks"`
	Blocking []string `json:"blocking" doc:"The blocking checks, in the reader's language"`
	Evidence []string `json:"evidence" doc:"One sentence per blocking check"`
}

// PlacementOther is the same email through another provider in the test.
type PlacementOther struct {
	ProviderID string `json:"providerId"`
	Name       string `json:"name"`
	Inbox      int    `json:"inbox"`
	Spam       int    `json:"spam"`
	Copies     int    `json:"copies"`
}

// PlacementEarlier is an earlier test through the same provider.
type PlacementEarlier struct {
	TestID      string    `json:"testId"`
	ContentName string    `json:"contentName"`
	Inbox       int       `json:"inbox"`
	Copies      int       `json:"copies"`
	StartedAt   time.Time `json:"startedAt"`
}

// PlacementGuidance explains a provider's spam results with ReputeCast's
// own rules (never AI): the likeliest cause and what to look at.
type PlacementGuidance struct {
	ProviderID   string `json:"providerId"`
	ProviderName string `json:"providerName"`
	Spam         int    `json:"spam"`
	Copies       int    `json:"copies"`
	// At names the mailbox providers where copies went to spam, when few.
	At []string `json:"at"`
	// Cause picks the explanation: Send Readiness blocks the email, the
	// authentication failed, a warming-up IP, or the sender's reputation.
	Cause string `json:"cause" enum:"readiness,auth,warmup,reputation"`
	// Auth is across every provider of the test; AuthProviders how many.
	Auth          PlacementAuthSummary    `json:"auth"`
	AuthProviders int                     `json:"authProviders"`
	Readiness     *PlacementAuthReadiness `json:"readiness,omitempty"`
	Other         *PlacementOther         `json:"other,omitempty"`
	Earlier       *PlacementEarlier       `json:"earlier,omitempty"`
	// The provider then: status, warm-up day and IP.
	Status     string `json:"status"`
	WarmDay    int    `json:"warmDay,omitempty"`
	WarmDays   int    `json:"warmDays,omitempty"`
	IP         string `json:"ip,omitempty"`
	HostRepute bool   `json:"hostRepute"`
}

// PlacementTestDetail is one test: what it sent, how, and where each copy landed.
type PlacementTestDetail struct {
	PlacementTestRow
	Variant          string `json:"variant,omitempty"`
	Language         string `json:"language,omitempty"`
	FallbackGreeting string `json:"fallbackGreeting,omitempty"`
	// ConditionalVersion is the conditional-content version sent (0:
	// none), ConditionalShows what it shows.
	ConditionalVersion int    `json:"conditionalVersion,omitempty"`
	ConditionalShows   string `json:"conditionalShows,omitempty"`
	Subject            string `json:"subject"`
	FromName           string `json:"fromName"`
	FromEmail          string `json:"fromEmail"`
	Stream             string `json:"stream"`
	// Rounds is how many copies each seed got (one per compared provider),
	// GapMinutes how far apart.
	Rounds     int `json:"rounds"`
	GapMinutes int `json:"gapMinutes"`
	// Copies in the seeds: the setting, when they go, and when they went.
	DeleteAfter     string     `json:"deleteAfter" enum:"24h,finished,never"`
	DeleteAt        *time.Time `json:"deleteAt,omitempty"`
	CopiesDeletedAt *time.Time `json:"copiesDeletedAt,omitempty"`
	CopiesDeletedBy string     `json:"copiesDeletedBy,omitempty"`
	StoppedAt       *time.Time `json:"stoppedAt,omitempty"`
	StoppedBy       string     `json:"stoppedBy,omitempty"`
	// FinishReason: every copy was found, the 15 minutes ran out, or a person stopped it.
	FinishReason string                `json:"finishReason,omitempty" enum:"all_found,timeout,stopped,"`
	Seeds        []PlacementResultSeed `json:"seeds"`
	Copies       []PlacementCopyView   `json:"copies"`
	Guidance     []PlacementGuidance   `json:"guidance"`
	HostRepute   bool                  `json:"hostRepute"`
	CanRun       bool                  `json:"canRun"`
	RunHelpers   []string              `json:"runHelpers"`
}

// Test returns one test with its matrix and guidance.
func (s *Service) Test(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PlacementTestDetail, error) {
	p := api.PrincipalFrom(ctx)
	row, err := s.Q.PlacementTest(ctx, store.PlacementTestParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return PlacementTestDetail{}, api.MapError(err, "Placement test")
	}
	t := row.PlacementTest
	copies, err := s.Q.PlacementCopies(ctx, id)
	if err != nil {
		return PlacementTestDetail{}, err
	}
	seedRows, err := s.Q.PlacementSeeds(ctx, ws.ID)
	if err != nil {
		return PlacementTestDetail{}, err
	}
	seeds := map[uuid.UUID]store.PlacementSeed{}
	for _, r := range seedRows {
		seeds[r.PlacementSeed.ID] = r.PlacementSeed
	}
	ids := make([]uuid.UUID, len(copies))
	for i, c := range copies {
		ids[i] = c.ID
	}
	evs, err := s.Q.PlacementTestEvents(ctx, ids)
	if err != nil {
		return PlacementTestDetail{}, err
	}
	byCopy := map[uuid.UUID][]store.DeliveryEvent{}
	for _, e := range evs {
		if e.MessageID != nil {
			byCopy[*e.MessageID] = append(byCopy[*e.MessageID], e)
		}
	}
	out := PlacementTestDetail{PlacementTestRow: testRow(i18n.LangFrom(ctx), t, row.StartedByName, row.StartedByColor, copies), Variant: t.Variant, Language: t.Language,
		ConditionalVersion: int(t.ConditionalVersion), ConditionalShows: t.ConditionalShows,
		FallbackGreeting: greeting(t.Html), Subject: t.Subject, FromName: t.FromName, FromEmail: t.FromEmail, Stream: t.Stream, Rounds: 1,
		DeleteAfter: t.DeleteAfter, DeleteAt: t.DeleteAt, CopiesDeletedAt: t.CopiesDeletedAt, CopiesDeletedBy: row.CopiesDeletedByName,
		StoppedAt: t.StoppedAt, StoppedBy: row.StoppedByName, Seeds: []PlacementResultSeed{}, Copies: []PlacementCopyView{}, Guidance: []PlacementGuidance{},
		HostRepute: s.hostReputeConnected(ctx, ws), CanRun: p.Can(authz.PlacementRun), RunHelpers: Helpers(ctx, s.Q, ws.ID, authz.PlacementRun)}
	if t.Through == "compare" {
		out.Rounds, out.GapMinutes = len(t.ProviderIds), int(CompareGap/time.Minute)
	}
	switch {
	case t.Status == "stopped":
		out.FinishReason = "stopped"
	case t.Status == "finished" && t.FinishedAt != nil && t.WaitUntil != nil && !t.FinishedAt.Before(*t.WaitUntil):
		out.FinishReason = "timeout"
	case t.Status == "finished":
		out.FinishReason = "all_found"
	}
	seen := map[string]bool{}
	for _, c := range copies {
		out.Copies = append(out.Copies, copyView(i18n.LangFrom(ctx), c, t, seeds, byCopy[c.ID]))
		if seen[c.SeedEmail] {
			continue
		}
		seen[c.SeedEmail] = true
		pv, _ := seedbox.ProviderOf(c.SeedProvider)
		rs := PlacementResultSeed{Email: c.SeedEmail, Provider: c.SeedProvider, ProviderName: pv.Name, Position: int(c.Position), Status: "removed"}
		if c.SeedID != nil {
			rs.ID = c.SeedID.String()
			if sd, ok := seeds[*c.SeedID]; ok {
				rs.Status, rs.StatusAt = sd.Status, sd.StatusAt
			}
		}
		out.Seeds = append(out.Seeds, rs)
	}
	sort.SliceStable(out.Seeds, func(i, j int) bool { return out.Seeds[i].Position < out.Seeds[j].Position })
	if t.Status == "finished" || t.Status == "stopped" {
		out.Guidance = s.guidance(ctx, ws, t, copies, out.Providers, out.HostRepute)
	}
	return out, nil
}

func (s *Service) hostReputeConnected(ctx context.Context, ws *store.Workspace) bool {
	c, err := s.Q.GetHostReputeConnection(ctx, ws.ID)
	if err != nil {
		return false
	}
	return c.Status == "connected" || c.Status == "limit_reached"
}

// guidance explains each provider with copies in spam: two or more, or at
// least half of what arrived.
func (s *Service) guidance(ctx context.Context, ws *store.Workspace, t store.PlacementTest, copies []store.PlacementCopy, results []PlacementProviderResult, hr bool) []PlacementGuidance {
	out := []PlacementGuidance{}
	all := authOf(copies)
	var ready *PlacementAuthReadiness
	if t.ContentKind == "campaign" && t.ContentID != nil && s.Readiness != nil {
		if r, ok := s.Readiness.Latest(ctx, ws, *t.ContentID); ok {
			ready = &PlacementAuthReadiness{Passed: r.Passed, Warnings: r.Warnings, Blocks: r.Blocks, Blocking: r.Blocking, Evidence: r.Evidence}
			if ready.Blocking == nil {
				ready.Blocking = []string{}
			}
			if ready.Evidence == nil {
				ready.Evidence = []string{}
			}
		}
	}
	for _, r := range results {
		arrived := r.Counts.Found
		if r.Counts.Spam < 2 && (arrived == 0 || r.Counts.Spam*2 < arrived) {
			continue
		}
		pid := uuid.MustParse(r.ID)
		g := PlacementGuidance{ProviderID: r.ID, ProviderName: r.Name, Spam: r.Counts.Spam, Copies: r.Counts.Copies, At: []string{}, Auth: all,
			AuthProviders: len(results), Readiness: ready, Status: r.Status, WarmDay: r.WarmDay, WarmDays: r.WarmDays, IP: r.IP, HostRepute: hr}
		at := []string{}
		for _, c := range copies {
			if c.ProviderID != nil && *c.ProviderID == pid && c.Status == "found" && c.Folder == seedbox.RoleSpam {
				p, _ := seedbox.ProviderOf(c.SeedProvider)
				if !slices.Contains(at, p.Name) {
					at = append(at, p.Name)
				}
			}
		}
		if r.Counts.Spam*2 < r.Counts.Copies && len(at) <= 3 {
			g.At = at
		}
		mine := []store.PlacementCopy{}
		for _, c := range copies {
			if c.ProviderID != nil && *c.ProviderID == pid {
				mine = append(mine, c)
			}
		}
		own := authOf(mine)
		switch {
		case ready != nil && ready.Blocks > 0:
			g.Cause = "readiness"
		case own.Checked > 0 && !own.AllPass():
			g.Cause = "auth"
		case r.WarmDay > 0:
			g.Cause = "warmup"
		default:
			g.Cause = "reputation"
		}
		for _, o := range results {
			if o.ID != r.ID {
				g.Other = &PlacementOther{ProviderID: o.ID, Name: o.Name, Inbox: o.Counts.Inbox, Spam: o.Counts.Spam, Copies: o.Counts.Copies}
				break
			}
		}
		if g.Other == nil {
			g.Earlier = s.earlier(ctx, ws, t, pid)
		}
		out = append(out, g)
	}
	return out
}

// earlier is the latest earlier finished test through a provider.
func (s *Service) earlier(ctx context.Context, ws *store.Workspace, t store.PlacementTest, pid uuid.UUID) *PlacementEarlier {
	tests, err := s.Q.PlacementTestsForProvider(ctx, store.PlacementTestsForProviderParams{WorkspaceID: ws.ID, ProviderID: pid, Since: t.StartedAt.AddDate(0, 0, -60)})
	if err != nil {
		return nil
	}
	var best *store.PlacementTest
	for i := range tests {
		x := tests[i]
		if x.ID == t.ID || !x.StartedAt.Before(t.StartedAt) || x.Status != "finished" {
			continue
		}
		best = &tests[i]
	}
	if best == nil {
		return nil
	}
	copies, err := s.Q.PlacementCopies(ctx, best.ID)
	if err != nil {
		return nil
	}
	c := countCopies(copies, &pid)
	return &PlacementEarlier{TestID: best.ID.String(), ContentName: best.ContentName, Inbox: c.Inbox, Copies: c.Copies, StartedAt: best.StartedAt}
}

// PlacementAuthCheck is one authentication result of a copy, in full.
type PlacementAuthCheck struct {
	Result   string `json:"result"`
	Domain   string `json:"domain,omitempty"`
	Selector string `json:"selector,omitempty"`
	IP       string `json:"ip,omitempty"`
	Aligned  *bool  `json:"aligned,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// PlacementCopyDetail is a copy's detail: where it landed, its
// authentication and headers, and the provider's events.
type PlacementCopyDetail struct {
	Copy PlacementCopyView `json:"copy"`
	// Path is the folder as the mailbox reports it ("acme.seed@outlook.com › Junk Email").
	Path string `json:"path,omitempty"`
	// ReadVia is how the seed was read.
	ReadVia string              `json:"readVia" enum:"imap,graph,gmail"`
	SPF     *PlacementAuthCheck `json:"spf,omitempty"`
	DKIM    *PlacementAuthCheck `json:"dkim,omitempty"`
	DMARC   *PlacementAuthCheck `json:"dmarc,omitempty"`
	// KeyHeaders are the lines that matter (Authentication-Results, the
	// hand-over Received, Message-ID and X-ReputeCast-Test); Headers all of them.
	KeyHeaders string           `json:"keyHeaders"`
	Headers    string           `json:"headers"`
	Events     []PlacementEvent `json:"events"`
	// ProviderMessageID is the provider's own id for the copy.
	ProviderMessageID string     `json:"providerMessageId,omitempty"`
	WaitUntil         *time.Time `json:"waitUntil,omitempty"`
	TestStatus        string     `json:"testStatus"`
}

// CopyDetail returns one copy.
func (s *Service) CopyDetail(ctx context.Context, ws *store.Workspace, testID, copyID uuid.UUID) (PlacementCopyDetail, error) {
	c, err := s.Q.PlacementCopy(ctx, store.PlacementCopyParams{ID: copyID, WorkspaceID: ws.ID})
	if err != nil || c.TestID != testID {
		if err == nil || errors.Is(err, pgx.ErrNoRows) {
			return PlacementCopyDetail{}, api.NotFound("Copy")
		}
		return PlacementCopyDetail{}, err
	}
	t, err := s.Q.PlacementTestByID(ctx, testID)
	if err != nil {
		return PlacementCopyDetail{}, err
	}
	seeds := map[uuid.UUID]store.PlacementSeed{}
	auth := c.Auth
	if c.SeedID != nil {
		if sd, err := s.Q.PlacementSeed(ctx, store.PlacementSeedParams{ID: *c.SeedID, WorkspaceID: ws.ID}); err == nil {
			seeds[sd.ID] = sd
		}
	}
	evs, err := s.Q.PlacementCopyEvents(ctx, &c.ID)
	if err != nil {
		return PlacementCopyDetail{}, err
	}
	out := PlacementCopyDetail{Copy: copyView(i18n.LangFrom(ctx), c, t, seeds, evs), Events: []PlacementEvent{}, ProviderMessageID: c.ProviderMessageID,
		WaitUntil: t.WaitUntil, TestStatus: t.Status, Headers: c.Headers, ReadVia: "imap"}
	oauth := strings.Contains(string(c.Ref), "graphId")
	if c.SeedID != nil {
		if sd, ok := seeds[*c.SeedID]; ok {
			oauth = sd.Auth == seedbox.AuthOAuth
		}
	}
	switch {
	case seedbox.IsMicrosoft(c.SeedProvider) && oauth:
		out.ReadVia = "graph"
	case seedbox.IsGoogle(c.SeedProvider):
		out.ReadVia = "gmail"
	}
	if c.Status == "found" {
		out.Path = c.SeedEmail + " › " + firstNonEmpty(c.FolderName, c.Folder)
		var r seedbox.Auth
		_ = json.Unmarshal(auth, &r)
		var hop seedbox.Hop
		_ = json.Unmarshal(c.Hop, &hop)
		conv := func(ch *seedbox.Check) *PlacementAuthCheck {
			if ch == nil {
				return nil
			}
			return &PlacementAuthCheck{Result: ch.Result, Domain: ch.Domain, Selector: ch.Selector, Detail: ch.Detail}
		}
		out.SPF, out.DKIM, out.DMARC = conv(r.SPF), conv(r.DKIM), conv(r.DMARC)
		if out.SPF != nil {
			out.SPF.IP = hop.IP
		}
		if out.DMARC != nil && out.DMARC.Result == "pass" {
			yes := true
			out.DMARC.Aligned = &yes
		}
		out.KeyHeaders = keyHeaders(c.Headers, hop.Line)
	}
	for _, e := range evs {
		out.Events = append(out.Events, *eventView(e))
	}
	return out, nil
}

// keyHeaders keeps the header fields the copy detail shows, folded as the
// mailbox sent them: Authentication-Results, the Received line of the
// hand-over (hop), the Message-ID and the test token.
func keyHeaders(raw, hop string) string {
	keep := map[string]bool{"authentication-results": true, "message-id": true, strings.ToLower(seedbox.TestHeader): true}
	type field struct{ name, text string }
	var fields []field
	for _, ln := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if ln == "" {
			break
		}
		if (ln[0] == ' ' || ln[0] == '\t') && len(fields) > 0 {
			fields[len(fields)-1].text += "\n" + ln
			continue
		}
		name, _, _ := strings.Cut(ln, ":")
		fields = append(fields, field{name: strings.ToLower(strings.TrimSpace(name)), text: ln})
	}
	var out []string
	for _, f := range fields {
		switch {
		case keep[f.name]:
			out = append(out, f.text)
		case f.name == "received" && hop != "":
			_, v, _ := strings.Cut(f.text, ":")
			if strings.Join(strings.Fields(v), " ") == hop {
				out = append(out, f.text)
			}
		}
	}
	return strings.Join(out, "\n")
}
