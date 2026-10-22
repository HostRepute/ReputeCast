package polls

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/permcheck"
	"github.com/hostrepute/reputecast/internal/store"
)

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// TooFew is the fewest answers a breakdown row compares: fewer read "Too
// few to compare".
const TooFew = 30

// PollAnswerCount is one answer's count and share of the answers.
type PollAnswerCount struct {
	Key     int     `json:"key"`
	Label   string  `json:"label"`
	Emoji   string  `json:"emoji,omitempty"`
	Tag     string  `json:"tag,omitempty"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

// PollNPS is a 0–10 poll's Net Promoter Score: promoters (9–10) minus
// detractors (0–6), as percentages of the answers.
type PollNPS struct {
	Score            int     `json:"score"`
	Promoters        int64   `json:"promoters"`
	Passives         int64   `json:"passives"`
	Detractors       int64   `json:"detractors"`
	PromoterPercent  float64 `json:"promoterPercent"`
	PassivePercent   float64 `json:"passivePercent"`
	DetractorPercent float64 `json:"detractorPercent"`
}

// PollAutomatedVisits are the visits ignored as a machine's.
type PollAutomatedVisits struct {
	Total    int64 `json:"total"`
	Gateways int64 `json:"gateways"`
	Prefetch int64 `json:"prefetch"`
}

// PollBreakdownRow is one row of a breakdown: a tier, a list or a
// language, with its answers and, when it has enough, its NPS or shares.
type PollBreakdownRow struct {
	Key     string            `json:"key"`
	Label   string            `json:"label"`
	Answers int64             `json:"answers"`
	Percent float64           `json:"percent"`
	TooFew  bool              `json:"tooFew"`
	NPS     *int              `json:"nps,omitempty"`
	Counts  []PollAnswerCount `json:"counts"`
}

// PollBreakdowns are the answers by engagement tier, list and language.
type PollBreakdowns struct {
	Tiers     []PollBreakdownRow `json:"tiers"`
	Lists     []PollBreakdownRow `json:"lists"`
	Languages []PollBreakdownRow `json:"languages"`
}

// PollOverTime is the cumulative answers by hour over the first 72 hours.
type PollOverTime struct {
	Since *time.Time `json:"since,omitempty"`
	// Cumulative has one point per hour: answers by the end of hour i.
	Cumulative []int64 `json:"cumulative"`
	// FirstSixPercent is the share that came in the first 6 hours; After
	// the answers after 72 hours.
	FirstSixPercent float64 `json:"firstSixPercent"`
	After           int64   `json:"after"`
}

// PollResults is a poll's section of the campaign report.
type PollResults struct {
	ID         string `json:"id"`
	BlockID    string `json:"blockId"`
	CampaignID string `json:"campaignId,omitempty"`
	Campaign   string `json:"campaign"`
	Kind       string `json:"kind" enum:"choice,scale,emoji"`
	Question   string `json:"question"`
	NPSWording bool   `json:"npsWording"`
	Low        string `json:"low,omitempty"`
	High       string `json:"high,omitempty"`
	Save       string `json:"save" enum:"none,tag,field"`
	Field      string `json:"field,omitempty"`
	// Status: open (closes at ClosesAt, or by hand) or closed (by a person,
	// or as scheduled).
	Status            string     `json:"status" enum:"open,closed"`
	ClosesAt          *time.Time `json:"closesAt,omitempty"`
	ClosedAt          *time.Time `json:"closedAt,omitempty"`
	ClosedBy          string     `json:"closedBy,omitempty"`
	ClosedAsScheduled bool       `json:"closedAsScheduled"`
	ReopenedAt        *time.Time `json:"reopenedAt,omitempty"`
	StartedAt         *time.Time `json:"startedAt,omitempty"`
	// Live is a poll still collecting its first 72 hours of answers.
	Live bool `json:"live"`
	// Answers are contacts' answers; Percent their share of Delivered.
	Answers     int64               `json:"answers"`
	Delivered   int64               `json:"delivered"`
	Percent     float64             `json:"percent"`
	NonContacts int64               `json:"nonContacts"`
	Tests       int64               `json:"tests"`
	Changed     int64               `json:"changed"`
	Comments    int64               `json:"comments"`
	Automated   PollAutomatedVisits `json:"automated"`
	Counts      []PollAnswerCount   `json:"counts"`
	NPS         *PollNPS            `json:"nps,omitempty"`
	Breakdown   PollBreakdowns      `json:"breakdown"`
	OverTime    PollOverTime        `json:"overTime"`
	// AnswerPage is where its answer links go (the publisher's domain),
	// with "…" for the signed part; Config is the Poll block's settings
	// (rc-poll), for a preview of the answer page.
	AnswerPage string `json:"answerPage"`
	Config     string `json:"config"`
	// Names is true when comments carry the contacts' names
	// (subscribers:view).
	Names bool `json:"names"`
	// CanClose (campaigns:edit) and CanExportAnswers (subscribers:export)
	// are the viewer's; the helpers are members to ask when not.
	CanClose         bool     `json:"canClose"`
	CloseHelpers     []string `json:"closeHelpers"`
	CanExportAnswers bool     `json:"canExportAnswers"`
	ExportHelpers    []string `json:"exportHelpers"`
}

// PollResultsList are a campaign's polls with their results.
type PollResultsList struct {
	Items []PollResults `json:"items"`
}

func percent(n, of int64) float64 {
	if of <= 0 {
		return 0
	}
	return math.Round(float64(n)/float64(of)*1000) / 10
}

// npsOf scores 0–10 answers.
func npsOf(counts map[int]int64) *PollNPS {
	var p, pa, d, total int64
	for k, n := range counts {
		total += n
		switch {
		case k >= 9:
			p += n
		case k >= 7:
			pa += n
		default:
			d += n
		}
	}
	if total == 0 {
		return &PollNPS{}
	}
	pp, dp := float64(p)/float64(total)*100, float64(d)/float64(total)*100
	return &PollNPS{Score: int(math.Round(pp - dp)), Promoters: p, Passives: pa, Detractors: d,
		PromoterPercent: percent(p, total), PassivePercent: percent(pa, total), DetractorPercent: percent(d, total)}
}

func countsOf(answers []Answer, counts map[int]int64) []PollAnswerCount {
	var total int64
	for _, n := range counts {
		total += n
	}
	out := make([]PollAnswerCount, 0, len(answers))
	for _, a := range answers {
		out = append(out, PollAnswerCount{Key: a.Key, Label: a.Label, Emoji: a.Emoji, Tag: a.Tag, Count: counts[a.Key], Percent: percent(counts[a.Key], total)})
	}
	return out
}

// CampaignPolls are the results of a campaign's polls, in the email's order.
func (s *Service) CampaignPolls(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) (PollResultsList, error) {
	out := PollResultsList{Items: []PollResults{}}
	c, err := s.Q.PollCampaignContent(ctx, store.PollCampaignContentParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return out, api.MapError(err, "Campaign")
	}
	rows, err := s.Q.PollsOfCampaign(ctx, store.PollsOfCampaignParams{WorkspaceID: ws.ID, CampaignID: &campaignID})
	if err != nil {
		return out, err
	}
	order := map[string]int{}
	for i, b := range Read(c.Mjml) {
		order[b.Config.ID] = i
	}
	sort.SliceStable(rows, func(i, j int) bool {
		oi, iok := order[rows[i].BlockID]
		oj, jok := order[rows[j].BlockID]
		if iok != jok {
			return iok
		}
		return oi < oj
	})
	for _, r := range rows {
		res, err := s.PollResultsOf(ctx, ws, p, r.ID)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, res)
	}
	return out, nil
}

// PollResultsOf is one poll's results.
func (s *Service) PollResultsOf(ctx context.Context, ws *store.Workspace, p *authz.Principal, pollID uuid.UUID) (PollResults, error) {
	r, err := s.Q.PollByID(ctx, store.PollByIDParams{WorkspaceID: ws.ID, ID: pollID})
	if err != nil {
		return PollResults{}, api.MapError(err, "Poll")
	}
	cfg, answers := configOf(r.BlockID, r.Kind, r.Question, r.Answers, r.Settings)
	now := s.now()
	out := PollResults{ID: r.ID.String(), BlockID: r.BlockID, Campaign: r.CampaignName, Kind: r.Kind, Question: r.Question, NPSWording: cfg.NPS,
		Save: cfg.Save, Field: cfg.Field, Status: "open", ClosesAt: r.ClosesAt, ReopenedAt: r.ReopenedAt, StartedAt: r.CampaignStartedAt,
		Counts: []PollAnswerCount{}, Names: p.Can(authz.SubscribersView), CanClose: p.Can(authz.CampaignsEdit), CloseHelpers: []string{},
		CanExportAnswers: p.Can(authz.SubscribersExport), ExportHelpers: []string{}}
	if !out.CanClose {
		out.CloseHelpers = nonNil(permcheck.Helpers(ctx, s.Q, ws.ID, authz.CampaignsEdit))
	}
	if !out.CanExportAnswers {
		out.ExportHelpers = nonNil(permcheck.Helpers(ctx, s.Q, ws.ID, authz.SubscribersExport))
	}
	if cfg.Kind == KindScale {
		out.Low, out.High = cfg.Low, cfg.High
	}
	if r.CampaignID != nil {
		out.CampaignID = r.CampaignID.String()
	}
	switch {
	case r.ClosedAt != nil:
		out.Status, out.ClosedAt, out.ClosedBy = "closed", r.ClosedAt, r.ClosedByName
	case r.ClosesAt != nil && !now.Before(*r.ClosesAt):
		out.Status, out.ClosedAt, out.ClosedAsScheduled = "closed", r.ClosesAt, true
	}
	out.AnswerPage = strings.TrimSuffix(s.pageLinks(ctx, ws).Poll("x"), "x") + "…"
	out.Config = cfg.Encode()
	sum, err := s.Q.PollSummary(ctx, r.ID)
	if err != nil {
		return out, err
	}
	out.Answers, out.NonContacts, out.Tests, out.Changed, out.Comments = sum.Contacts, sum.NonContacts, sum.Tests, sum.Changed, sum.Comments
	if r.CampaignID != nil {
		if d, err := s.Q.PollDelivered(ctx, *r.CampaignID); err == nil {
			out.Delivered = d.Delivered
		}
	}
	out.Percent = percent(out.Answers, out.Delivered)
	if auto, err := s.Q.PollAutomated(ctx, r.ID); err == nil {
		for _, a := range auto {
			out.Automated.Total += a.N
			if a.Reason == "gateway" {
				out.Automated.Gateways += a.N
			} else {
				out.Automated.Prefetch += a.N
			}
		}
	}
	counts := map[int]int64{}
	if rows, err := s.Q.PollAnswerCounts(ctx, r.ID); err == nil {
		for _, c := range rows {
			counts[int(c.Answer)] = c.N
		}
	}
	out.Counts = countsOf(answers, counts)
	if cfg.Kind == KindScale {
		out.NPS = npsOf(counts)
	}
	out.Breakdown = s.breakdowns(ctx, r.ID, cfg, answers, out.Answers)
	if r.CampaignStartedAt != nil {
		out.OverTime = s.overTime(ctx, r.ID, *r.CampaignStartedAt, out.Answers)
		out.Live = out.Status == "open" && now.Sub(*r.CampaignStartedAt) < 72*time.Hour
	} else {
		out.OverTime = PollOverTime{Cumulative: []int64{}}
	}
	return out, nil
}

func (s *Service) breakdowns(ctx context.Context, pollID uuid.UUID, cfg PollConfig, answers []Answer, total int64) PollBreakdowns {
	l := i18n.LangFrom(ctx)
	out := PollBreakdowns{Tiers: []PollBreakdownRow{}, Lists: []PollBreakdownRow{}, Languages: []PollBreakdownRow{}}
	type acc struct {
		key, label string
		counts     map[int]int64
	}
	finish := func(rows []*acc) []PollBreakdownRow {
		res := make([]PollBreakdownRow, 0, len(rows))
		for _, a := range rows {
			var n int64
			for _, c := range a.counts {
				n += c
			}
			row := PollBreakdownRow{Key: a.key, Label: a.label, Answers: n, Percent: percent(n, total), TooFew: n < TooFew, Counts: countsOf(answers, a.counts)}
			if cfg.Kind == KindScale && !row.TooFew {
				score := npsOf(a.counts).Score
				row.NPS = &score
			}
			res = append(res, row)
		}
		return res
	}
	// Tiers in their order, the ones with answers.
	tiers := map[string]*acc{}
	if rows, err := s.Q.PollBreakdownTier(ctx, pollID); err == nil {
		for _, r := range rows {
			a := tiers[r.Tier]
			if a == nil {
				a = &acc{key: r.Tier, label: tierLabel(l, r.Tier), counts: map[int]int64{}}
				tiers[r.Tier] = a
			}
			a.counts[int(r.Answer)] += r.N
		}
	}
	var tierRows []*acc
	for _, t := range []string{"active", "cooling", "at_risk", "sunset", ""} {
		if a := tiers[t]; a != nil {
			if t == "" {
				a.label = mLabelNoTier.Text(l, nil)
			}
			tierRows = append(tierRows, a)
		}
	}
	out.Tiers = finish(tierRows)
	lists := map[uuid.UUID]*acc{}
	var listOrder []uuid.UUID
	if rows, err := s.Q.PollBreakdownList(ctx, pollID); err == nil {
		for _, r := range rows {
			a := lists[r.ListID]
			if a == nil {
				a = &acc{key: r.ListID.String(), label: r.ListName, counts: map[int]int64{}}
				lists[r.ListID] = a
				listOrder = append(listOrder, r.ListID)
			}
			a.counts[int(r.Answer)] += r.N
		}
	}
	var listRows []*acc
	for _, id := range listOrder {
		listRows = append(listRows, lists[id])
	}
	sort.SliceStable(listRows, func(i, j int) bool { return sumOf(listRows[i].counts) > sumOf(listRows[j].counts) })
	out.Lists = finish(listRows)
	langs := map[string]*acc{}
	if rows, err := s.Q.PollBreakdownLanguage(ctx, pollID); err == nil {
		for _, r := range rows {
			a := langs[r.Language]
			if a == nil {
				label := languageLabel(l, r.Language)
				if r.Language == "" {
					label = mLabelNotSet.Text(l, nil)
				}
				a = &acc{key: r.Language, label: label, counts: map[int]int64{}}
				langs[r.Language] = a
			}
			a.counts[int(r.Answer)] += r.N
		}
	}
	var langRows []*acc
	for _, a := range langs {
		langRows = append(langRows, a)
	}
	sort.SliceStable(langRows, func(i, j int) bool {
		si, sj := sumOf(langRows[i].counts), sumOf(langRows[j].counts)
		if si != sj {
			return si > sj
		}
		return langRows[i].label < langRows[j].label
	})
	out.Languages = finish(langRows)
	return out
}

func sumOf(m map[int]int64) int64 {
	var n int64
	for _, v := range m {
		n += v
	}
	return n
}

func (s *Service) overTime(ctx context.Context, pollID uuid.UUID, since time.Time, total int64) PollOverTime {
	out := PollOverTime{Since: &since, Cumulative: make([]int64, 72)}
	rows, err := s.Q.PollAnswersByHour(ctx, store.PollAnswersByHourParams{PollID: pollID, Since: since})
	if err != nil {
		return out
	}
	byHour := map[int]int64{}
	for _, r := range rows {
		if r.Hour >= 72 {
			out.After += r.N
			continue
		}
		byHour[int(r.Hour)] += r.N
	}
	var run, six int64
	for h := 0; h < 72; h++ {
		run += byHour[h]
		out.Cumulative[h] = run
		if h < 6 {
			six += byHour[h]
		}
	}
	// Hours that haven't happened yet stay out of the line.
	if elapsed := int(s.now().Sub(since) / time.Hour); elapsed < 71 && elapsed >= 0 {
		out.Cumulative = out.Cumulative[:elapsed+1]
	}
	out.FirstSixPercent = percent(six, total)
	return out
}

// PollComment is one comment with its answer.
type PollComment struct {
	ID     string `json:"id"`
	Answer int    `json:"answer"`
	Label  string `json:"label"`
	// Group of a 0–10 answer: promoter, passive or detractor.
	Group   string    `json:"group,omitempty" enum:"promoter,passive,detractor"`
	Comment string    `json:"comment"`
	At      time.Time `json:"at"`
	// Contact is the contact's name (people with subscribers:view), empty
	// otherwise; NonContact marks answers from people who aren't contacts.
	Contact    string `json:"contact,omitempty"`
	ContactID  string `json:"contactId,omitempty"`
	NonContact bool   `json:"nonContact"`
}

// PollCommentCounts are the comments by group, for the filter.
type PollCommentCounts struct {
	All        int64 `json:"all"`
	Promoters  int64 `json:"promoters"`
	Passives   int64 `json:"passives"`
	Detractors int64 `json:"detractors"`
}

// PollCommentList is a page of comments, newest first.
type PollCommentList struct {
	Items  []PollComment     `json:"items"`
	Total  int64             `json:"total"`
	Counts PollCommentCounts `json:"counts"`
	Names  bool              `json:"names"`
}

func groupOf(kind string, answer int) string {
	if kind != KindScale {
		return ""
	}
	switch {
	case answer >= 9:
		return "promoter"
	case answer >= 7:
		return "passive"
	}
	return "detractor"
}

// PollCommentsFilter narrows the comments.
type PollCommentsFilter struct {
	Group  string
	Search string
	Offset int
	Limit  int
}

// Comments are a poll's comments, newest first, with names for people
// with subscribers:view.
func (s *Service) Comments(ctx context.Context, ws *store.Workspace, p *authz.Principal, pollID uuid.UUID, f PollCommentsFilter) (PollCommentList, error) {
	r, err := s.Q.PollByID(ctx, store.PollByIDParams{WorkspaceID: ws.ID, ID: pollID})
	if err != nil {
		return PollCommentList{}, api.MapError(err, "Poll")
	}
	_, answers := configOf(r.BlockID, r.Kind, r.Question, r.Answers, r.Settings)
	names := p.Can(authz.SubscribersView)
	out := PollCommentList{Items: []PollComment{}, Names: names}
	if counts, err := s.Q.PollCommentsCount(ctx, r.ID); err == nil {
		for _, c := range counts {
			out.Counts.All += c.N
			switch groupOf(r.Kind, int(c.Answer)) {
			case "promoter":
				out.Counts.Promoters += c.N
			case "passive":
				out.Counts.Passives += c.N
			case "detractor":
				out.Counts.Detractors += c.N
			}
		}
	}
	lo, hi := int32(-1), int32(100)
	switch f.Group {
	case "promoters":
		lo, hi, out.Total = 9, 10, out.Counts.Promoters
	case "passives":
		lo, hi, out.Total = 7, 8, out.Counts.Passives
	case "detractors":
		lo, hi, out.Total = 0, 6, out.Counts.Detractors
	default:
		out.Total = out.Counts.All
	}
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	search := strings.TrimSpace(f.Search)
	rows, err := s.Q.PollComments(ctx, store.PollCommentsParams{PollID: r.ID, MinAnswer: lo, MaxAnswer: hi, Search: search, Names: names, Lim: int32(limit), Off: int32(max(f.Offset, 0))})
	if err != nil {
		return out, err
	}
	if search != "" {
		out.Total = 0
		if len(rows) > 0 {
			out.Total = rows[0].Total
		}
	}
	label := map[int]string{}
	for _, a := range answers {
		label[a.Key] = a.Label
	}
	for _, c := range rows {
		item := PollComment{ID: c.ID.String(), Answer: int(c.Answer), Label: label[int(c.Answer)], Group: groupOf(r.Kind, int(c.Answer)), Comment: c.Comment, At: c.At,
			NonContact: c.Kind == AnswererNonContact}
		if names && c.ContactID != nil {
			item.Contact = strings.TrimSpace(c.FirstName + " " + c.LastName)
			if item.Contact == "" {
				item.Contact = c.Email
			}
			item.ContactID = c.ContactID.String()
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}
