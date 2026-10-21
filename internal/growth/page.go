package growth

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// GrowthQuery is what Audience › Growth shows: a period, all lists or one,
// with or without imports.
type GrowthQuery struct {
	Period string `query:"period" enum:"30d,90d,12m,custom," doc:"30d (default), 90d, 12m or custom"`
	From   string `query:"from" maxLength:"10" doc:"First day of a custom period (YYYY-MM-DD)"`
	To     string `query:"to" maxLength:"10" doc:"Last day of a custom period"`
	List   string `query:"list" doc:"A list id: count its members instead of contacts"`
	// Imports adds imported contacts to the joins (a marker on the chart).
	Imports bool `query:"imports"`
	// CohortSource limits cohort retention to one source key.
	CohortSource string `query:"cohortSource" maxLength:"200"`
}

// GrowthPeriod is the period counted, in the workspace timezone.
type GrowthPeriod struct {
	Kind string `json:"kind" enum:"30d,90d,12m,custom"`
	From string `json:"from" doc:"YYYY-MM-DD"`
	To   string `json:"to" doc:"YYYY-MM-DD"`
	Days int    `json:"days"`
}

// GrowthListRef is the list counted in the one-list view.
type GrowthListRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GrowthImportItem is one import of the period.
type GrowthImportItem struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Day   string `json:"day"`
	Count int64  `json:"count"`
}

// GrowthTiles are the four numbers at the top.
type GrowthTiles struct {
	Joined int64 `json:"joined"`
	Left   int64 `json:"left"`
	Net    int64 `json:"net"`
	// NetPct is the net change against the contacts (or members) at the
	// start of the period.
	NetPct      float64            `json:"netPct"`
	Imports     int64              `json:"imports"`
	ImportItems []GrowthImportItem `json:"importItems"`
}

// GrowthSeries is one colour of the chart: one of the four biggest sources,
// or everything else ("other").
type GrowthSeries struct {
	Key    string       `json:"key"`
	Other  bool         `json:"other"`
	Source GrowthSource `json:"source"`
	Joined int64        `json:"joined"`
	Left   int64        `json:"left"`
}

// GrowthChartDay is one day of the chart: joins and departures per series
// (in Series order), and imports when they are included.
type GrowthChartDay struct {
	Day     string  `json:"day"`
	Joined  []int64 `json:"joined"`
	Left    []int64 `json:"left"`
	Imports int64   `json:"imports"`
}

// GrowthChart is joins above the axis and departures below, by day.
type GrowthChart struct {
	Series []GrowthSeries   `json:"series"`
	Days   []GrowthChartDay `json:"days"`
}

// GrowthSourceRow is one row of the sources table.
type GrowthSourceRow struct {
	Source GrowthSource `json:"source"`
	// Joined is nil for an import while imports are counted apart.
	Joined *int64 `json:"joined,omitempty"`
	Left   int64  `json:"left"`
	Net    *int64 `json:"net,omitempty"`
	// Contacts is how many contacts came from the source; with one list
	// picked, how many members it brought to the list in the period.
	Contacts  int64 `json:"contacts"`
	Sent      int64 `json:"sent"`
	Delivered int64 `json:"delivered"`
	// Rates are nil when there are too few to compare.
	BounceRate    *float64 `json:"bounceRate,omitempty"`
	ComplaintRate *float64 `json:"complaintRate,omitempty"`
	HumanOpenRate *float64 `json:"humanOpenRate,omitempty"`
	// TooFew says why rates aren't shown: contacts (under 50) or messages (under 100 delivered).
	TooFew string `json:"tooFew,omitempty" enum:"contacts,messages,"`
	// BounceFlag and ComplaintFlag are how many times the workspace's rate a
	// flagged source has.
	BounceFlag    *float64 `json:"bounceFlag,omitempty"`
	ComplaintFlag *float64 `json:"complaintFlag,omitempty"`
	sortJoined    int64
}

// GrowthTotals is the sources table's last row.
type GrowthTotals struct {
	Joined int64 `json:"joined"`
	Left   int64 `json:"left"`
	Net    int64 `json:"net"`
}

// GrowthRates are the workspace's own rates over the period's sends.
type GrowthRates struct {
	Sent      int64   `json:"sent"`
	Delivered int64   `json:"delivered"`
	Bounce    float64 `json:"bounce"`
	Complaint float64 `json:"complaint"`
	HumanOpen float64 `json:"humanOpen"`
}

// GrowthDepartures are departures by kind.
type GrowthDepartures struct {
	Unsubscribed int64 `json:"unsubscribed"`
	HardBounced  int64 `json:"hardBounced"`
	Complained   int64 `json:"complained"`
	Erased       int64 `json:"erased"`
}

// GrowthUTMRow is one UTM combination.
type GrowthUTMRow struct {
	Source   string `json:"source"`
	Medium   string `json:"medium"`
	Campaign string `json:"campaign"`
	Joins    int64  `json:"joins"`
	// HumanOpenRate is nil until the combination's contacts got emails.
	HumanOpenRate *float64 `json:"humanOpenRate,omitempty"`
}

// GrowthRest is the "Other, N …" row of a table.
type GrowthRest struct {
	Count         int      `json:"count"`
	Joins         int64    `json:"joins"`
	HumanOpenRate *float64 `json:"humanOpenRate,omitempty"`
}

// GrowthUTM is the UTM table.
type GrowthUTM struct {
	CaptureStartedAt *time.Time `json:"captureStartedAt,omitempty"`
	// Joins counts the joins through forms and the subscribe page;
	// Captured those since capture began.
	Joins       int64          `json:"joins"`
	Captured    int64          `json:"captured"`
	NotCaptured int64          `json:"notCaptured"`
	NotCapRate  *float64       `json:"notCapturedHumanOpenRate,omitempty"`
	None        GrowthUTMRow   `json:"none"`
	Rows        []GrowthUTMRow `json:"rows"`
	Other       GrowthRest     `json:"other"`
}

// GrowthReferrerRow is one referring page.
type GrowthReferrerRow struct {
	Page  string `json:"page"`
	Joins int64  `json:"joins"`
}

// GrowthReferrers is the referring pages table.
type GrowthReferrers struct {
	Rows  []GrowthReferrerRow `json:"rows"`
	None  int64               `json:"none"`
	Other GrowthRest          `json:"other"`
	Joins int64               `json:"joins"`
}

// GrowthRetention is one cohort's share still subscribed after a mark.
type GrowthRetention struct {
	// State is done, in_progress (some members haven't reached the mark)
	// or not_started (none have).
	State string   `json:"state" enum:"done,in_progress,not_started"`
	Rate  *float64 `json:"rate,omitempty"`
	// CompleteOn is when every member reaches the mark; From is when the
	// first one does (a source whose first join is in this cohort).
	CompleteOn string `json:"completeOn,omitempty"`
	From       string `json:"from,omitempty"`
}

// GrowthCohort is the contacts who joined in one month.
type GrowthCohort struct {
	Month    string          `json:"month" doc:"YYYY-MM"`
	Contacts int64           `json:"contacts"`
	After30  GrowthRetention `json:"after30"`
	After90  GrowthRetention `json:"after90"`
}

// GrowthPage is Audience › Growth.
type GrowthPage struct {
	Period GrowthPeriod   `json:"period"`
	List   *GrowthListRef `json:"list,omitempty"`
	// Members is true in the one-list view: it counts list members.
	Members        bool               `json:"members"`
	IncludeImports bool               `json:"includeImports"`
	Contacts       int64              `json:"contacts"`
	Tiles          GrowthTiles        `json:"tiles"`
	Chart          GrowthChart        `json:"chart"`
	Sources        []GrowthSourceRow  `json:"sources"`
	Total          GrowthTotals       `json:"total"`
	Workspace      GrowthRates        `json:"workspace"`
	Departures     GrowthDepartures   `json:"departures"`
	UTM            GrowthUTM          `json:"utm"`
	Referrers      GrowthReferrers    `json:"referrers"`
	Cohorts        []GrowthCohort     `json:"cohorts"`
	CohortSource   string             `json:"cohortSource"`
	CohortSources  []GrowthSource     `json:"cohortSources"`
	Lists          []GrowthListRef    `json:"lists"`
	OnlyImports    bool               `json:"onlyImports"`
	Imports        []GrowthImportItem `json:"imports" doc:"Every finished import, for the only-imports state"`
	// MinContacts and MinDelivered are the "too few to compare" limits;
	// FlagFactor is how many times the workspace's rate flags a source.
	MinContacts  int64   `json:"minContacts"`
	MinDelivered int64   `json:"minDelivered"`
	FlagFactor   float64 `json:"flagFactor"`
}

// The quality rules (brief §3.4).
const (
	minContacts  = 50
	minDelivered = 100
	flagFactor   = 2.0
	topSeries    = 4
	topRows      = 5
)

// window is a period's bounds.
type window struct {
	kind     string
	from, to time.Time // [from, to)
	fromDay  time.Time
	days     int
	loc      *time.Location
}

func periodOf(q GrowthQuery, now time.Time, loc *time.Location) (window, error) {
	l := now.In(loc)
	today := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
	w := window{kind: q.Period, loc: loc}
	if w.kind == "" {
		w.kind = "30d"
	}
	switch w.kind {
	case "90d":
		w.days = 90
	case "12m":
		w.days = 365
	case "custom":
		from, err1 := time.ParseInLocation("2006-01-02", q.From, loc)
		to, err2 := time.ParseInLocation("2006-01-02", q.To, loc)
		if err1 != nil || err2 != nil {
			return w, periodError{"query.from", mPeriodPickDays}
		}
		if to.Before(from) {
			return w, periodError{"query.to", mPeriodEndsBefore}
		}
		w.days = int(to.Sub(from).Hours()/24+0.5) + 1
		if w.days > 731 {
			return w, periodError{"query.from", mPeriodTooLong}
		}
		w.fromDay = from
		w.from = from
		w.to = to.AddDate(0, 0, 1)
		return w, nil
	default:
		w.kind, w.days = "30d", 30
	}
	w.fromDay = today.AddDate(0, 0, -(w.days - 1))
	w.from = w.fromDay
	w.to = today.AddDate(0, 0, 1)
	return w, nil
}

func ratio(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

func ptr[T any](v T) *T { return &v }

// Page returns everything Audience › Growth shows.
func (s *Service) Page(ctx context.Context, ws *store.Workspace, in GrowthQuery) (GrowthPage, error) {
	loc := location(ws.Timezone)
	w, err := periodOf(in, s.now(), loc)
	if err != nil {
		var pe periodError
		if errors.As(err, &pe) {
			return GrowthPage{}, api.Invalid(pe.location, pe.msg.Text(i18n.LangFrom(ctx), nil))
		}
		return GrowthPage{}, err
	}
	q := s.Q
	n, err := s.names(ctx, q, ws)
	if err != nil {
		return GrowthPage{}, err
	}
	out := GrowthPage{
		Period:         GrowthPeriod{Kind: w.kind, From: w.fromDay.Format("2006-01-02"), To: w.to.AddDate(0, 0, -1).Format("2006-01-02"), Days: w.days},
		IncludeImports: in.Imports, Sources: []GrowthSourceRow{}, Cohorts: []GrowthCohort{}, CohortSources: []GrowthSource{}, Lists: []GrowthListRef{},
		Imports: []GrowthImportItem{}, MinContacts: minContacts, MinDelivered: minDelivered, FlagFactor: flagFactor, CohortSource: in.CohortSource,
	}
	lists, err := s.Pool.Query(ctx, `SELECT id, name FROM lists WHERE workspace_id = $1 AND deleted_at IS NULL ORDER BY created_at`, ws.ID)
	if err != nil {
		return out, err
	}
	for lists.Next() {
		var id uuid.UUID
		var name string
		if err := lists.Scan(&id, &name); err != nil {
			lists.Close()
			return out, err
		}
		out.Lists = append(out.Lists, GrowthListRef{ID: id.String(), Name: name})
		if in.List == id.String() {
			out.List = &GrowthListRef{ID: id.String(), Name: name}
		}
	}
	lists.Close()
	if in.List != "" && out.List == nil {
		return out, api.NotFound("List")
	}
	out.Members = out.List != nil
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL`, ws.ID).Scan(&out.Contacts); err != nil {
		return out, err
	}
	g := &gather{s: s, ws: ws, w: w, n: n, page: &out, in: in}
	if out.List != nil {
		g.list = uuid.MustParse(out.List.ID)
	}
	steps := []func(context.Context) error{g.joinsAndDepartures, g.quality, g.utm, g.referrers, g.cohorts, g.imports}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return out, err
		}
	}
	return out, nil
}

// gather builds a page step by step.
type gather struct {
	s    *Service
	ws   *store.Workspace
	w    window
	n    *names
	page *GrowthPage
	in   GrowthQuery
	list uuid.UUID
	// counts by source key, for the table
	joined, left map[string]int64
	contacts     map[string]int64
	firstJoin    map[string]time.Time
	hosts        map[string][]string
	utmRates     map[string]rateAgg
}

// rateAgg adds up a source's (or a UTM combination's) sends.
type rateAgg struct{ sent, delivered, bounced, complained, opened int64 }

// sourceExpr is a row's source key in the one-list view: how the member
// joined that list (the preference center, a form, a person), else how the
// contact first joined.
const listSourceExpr = `CASE
    WHEN lm.source = 'preference_center' THEN 'preference_center'
    WHEN lm.source = 'team' THEN 'person'
    WHEN lm.source = 'subscribe_page' THEN 'subscribe_page'
    WHEN lm.source LIKE 'form:%' AND f.id IS NOT NULL THEN 'form:' || f.id::text
    WHEN cg.contact_id IS NOT NULL THEN cg.source_kind || CASE WHEN cg.source_id <> '' THEN ':' || cg.source_id ELSE '' END
    ELSE 'unknown' END`

func (g *gather) joinsAndDepartures(ctx context.Context) error {
	ws, w := g.ws, g.w
	g.joined, g.left, g.contacts, g.firstJoin, g.hosts = map[string]int64{}, map[string]int64{}, map[string]int64{}, map[string]time.Time{}, map[string][]string{}
	type dayRow struct {
		day  time.Time
		key  string
		n    int64
		kind string // left kind, for departures
	}
	var joins, lefts []dayRow
	var err error
	scan := func(sql string, args []any, departures bool) ([]dayRow, error) {
		rows, err := g.s.Pool.Query(ctx, sql, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []dayRow
		for rows.Next() {
			var r dayRow
			if departures {
				err = rows.Scan(&r.day, &r.key, &r.kind, &r.n)
			} else {
				err = rows.Scan(&r.day, &r.key, &r.n)
			}
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, rows.Err()
	}
	tz := w.loc.String()
	if g.list == uuid.Nil {
		joins, err = scan(`SELECT (g.joined_at AT TIME ZONE $4)::date, g.source_kind || CASE WHEN g.source_id <> '' THEN ':' || g.source_id ELSE '' END, count(*)
			FROM contact_growth g WHERE g.workspace_id = $1 AND g.joined_at >= $2 AND g.joined_at < $3 GROUP BY 1, 2`, []any{ws.ID, w.from, w.to, tz}, false)
		if err != nil {
			return err
		}
		lefts, err = scan(`SELECT (g.left_at AT TIME ZONE $4)::date, g.source_kind || CASE WHEN g.source_id <> '' THEN ':' || g.source_id ELSE '' END, g.left_kind, count(*)
			FROM contact_growth g WHERE g.workspace_id = $1 AND g.left_at >= $2 AND g.left_at < $3 GROUP BY 1, 2, 3`, []any{ws.ID, w.from, w.to, tz}, true)
		if err != nil {
			return err
		}
		rows, err := g.s.Pool.Query(ctx, `SELECT g.source_kind || CASE WHEN g.source_id <> '' THEN ':' || g.source_id ELSE '' END, count(*), min(g.joined_at)
			FROM contact_growth g JOIN contacts c ON c.id = g.contact_id AND c.erased_at IS NULL WHERE g.workspace_id = $1 GROUP BY 1`, ws.ID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var k string
			var c int64
			var first time.Time
			if err := rows.Scan(&k, &c, &first); err != nil {
				rows.Close()
				return err
			}
			g.contacts[k], g.firstJoin[k] = c, first
		}
		rows.Close()
	} else {
		from := `FROM list_memberships lm
			JOIN contacts c ON c.id = lm.contact_id
			LEFT JOIN contact_growth cg ON cg.contact_id = lm.contact_id
			LEFT JOIN forms f ON f.workspace_id = c.workspace_id AND lm.source = 'form:' || f.slug`
		joins, err = scan(`SELECT (lm.subscribed_at AT TIME ZONE $4)::date, `+listSourceExpr+`, count(*) `+from+`
			WHERE lm.list_id = $1 AND lm.status = 'subscribed' AND lm.subscribed_at >= $2 AND lm.subscribed_at < $3 GROUP BY 1, 2`, []any{g.list, w.from, w.to, tz}, false)
		if err != nil {
			return err
		}
		lefts, err = scan(`SELECT (x.at AT TIME ZONE $4)::date, x.key, x.kind, count(*) FROM (
				SELECT lm.unsubscribed_at AS at, `+listSourceExpr+` AS key, 'unsubscribed' AS kind `+from+`
				WHERE lm.list_id = $1 AND lm.unsubscribed_at >= $2 AND lm.unsubscribed_at < $3
				UNION ALL
				SELECT cg.left_at, `+listSourceExpr+`, cg.left_kind `+from+`
				WHERE lm.list_id = $1 AND lm.status = 'subscribed' AND cg.left_kind IN ('hard_bounce', 'complaint', 'erased') AND cg.left_at >= $2 AND cg.left_at < $3
			) x GROUP BY 1, 2, 3`, []any{g.list, w.from, w.to, tz}, true)
		if err != nil {
			return err
		}
		rows, err := g.s.Pool.Query(ctx, `SELECT `+listSourceExpr+`, count(*), min(lm.subscribed_at) `+from+`
			WHERE lm.list_id = $1 AND lm.status = 'subscribed' AND c.erased_at IS NULL GROUP BY 1`, g.list)
		if err != nil {
			return err
		}
		for rows.Next() {
			var k string
			var c int64
			var first *time.Time
			if err := rows.Scan(&k, &c, &first); err != nil {
				rows.Close()
				return err
			}
			g.contacts[k] = c
			if first != nil {
				g.firstJoin[k] = *first
			}
		}
		rows.Close()
	}
	// Where each form's signups happened (the source's second line).
	hostRows, err := g.s.Pool.Query(ctx, `SELECT g.source_id, split_part(g.page, '/', 1) AS host, count(*) FROM contact_growth g
		WHERE g.workspace_id = $1 AND g.source_kind = 'form' AND g.page <> '' AND g.joined_at >= $2 AND g.joined_at < $3
		GROUP BY 1, 2 ORDER BY 3 DESC`, ws.ID, w.from, w.to)
	if err != nil {
		return err
	}
	for hostRows.Next() {
		var id, host string
		var c int64
		if err := hostRows.Scan(&id, &host, &c); err != nil {
			hostRows.Close()
			return err
		}
		k := SourceKey(KindForm, id)
		if len(g.hosts[k]) < 3 {
			g.hosts[k] = append(g.hosts[k], host)
		}
	}
	hostRows.Close()

	p := g.page
	imported := func(key string) bool { kind, _ := splitKey(key); return kindOf(kind).Imported }
	// Tiles, departures by kind, and the table's counts.
	for _, r := range joins {
		g.joined[r.key] += r.n
		if imported(r.key) {
			p.Tiles.Imports += r.n
			if !g.in.Imports {
				continue
			}
		}
		p.Tiles.Joined += r.n
	}
	for _, r := range lefts {
		g.left[r.key] += r.n
		p.Tiles.Left += r.n
		switch r.kind {
		case "unsubscribed":
			p.Departures.Unsubscribed += r.n
		case "hard_bounce":
			p.Departures.HardBounced += r.n
		case "complaint":
			p.Departures.Complained += r.n
		case "erased":
			p.Departures.Erased += r.n
		}
	}
	p.Tiles.Net = p.Tiles.Joined - p.Tiles.Left
	base := p.Contacts
	if g.list != uuid.Nil {
		if err := g.s.Pool.QueryRow(ctx, `SELECT count(*) FROM list_memberships WHERE list_id = $1 AND status = 'subscribed'`, g.list).Scan(&base); err != nil {
			return err
		}
	}
	if start := base - p.Tiles.Net; start > 0 {
		p.Tiles.NetPct = float64(p.Tiles.Net) / float64(start)
	}
	// Chart series: the four biggest sources by joins (imports never become
	// bars), then everything else.
	type kv struct {
		key string
		n   int64
	}
	var ranked []kv
	for k, v := range g.joined {
		if !imported(k) && v > 0 {
			ranked = append(ranked, kv{k, v})
		}
	}
	slices.SortFunc(ranked, func(a, b kv) int {
		if a.n != b.n {
			return int(b.n - a.n)
		}
		return strings.Compare(a.key, b.key)
	})
	slot := map[string]int{}
	for i := 0; i < len(ranked) && i < topSeries; i++ {
		src := g.describe(ranked[i].key)
		p.Chart.Series = append(p.Chart.Series, GrowthSeries{Key: ranked[i].key, Source: src})
		slot[ranked[i].key] = i
	}
	other := len(p.Chart.Series)
	p.Chart.Series = append(p.Chart.Series, GrowthSeries{Key: "other", Other: true, Source: GrowthSource{Key: "other", Kind: "other", Hosts: []string{}}})
	days := map[string]*GrowthChartDay{}
	for i := 0; i < w.days; i++ {
		d := w.fromDay.AddDate(0, 0, i).Format("2006-01-02")
		cd := &GrowthChartDay{Day: d, Joined: make([]int64, len(p.Chart.Series)), Left: make([]int64, len(p.Chart.Series))}
		days[d] = cd
		p.Chart.Days = append(p.Chart.Days, GrowthChartDay{})
	}
	idxOf := func(key string) int {
		if i, ok := slot[key]; ok {
			return i
		}
		return other
	}
	for _, r := range joins {
		cd := days[r.day.Format("2006-01-02")]
		if cd == nil {
			continue
		}
		if imported(r.key) {
			if g.in.Imports {
				cd.Imports += r.n
			}
			continue
		}
		i := idxOf(r.key)
		cd.Joined[i] += r.n
		p.Chart.Series[i].Joined += r.n
	}
	for _, r := range lefts {
		cd := days[r.day.Format("2006-01-02")]
		if cd == nil {
			continue
		}
		i := idxOf(r.key)
		cd.Left[i] += r.n
		p.Chart.Series[i].Left += r.n
	}
	for i := 0; i < w.days; i++ {
		p.Chart.Days[i] = *days[w.fromDay.AddDate(0, 0, i).Format("2006-01-02")]
	}
	// Sources table: every source with joins or departures in the period,
	// and every import of the period.
	keys := map[string]bool{}
	for k := range g.joined {
		keys[k] = true
	}
	for k := range g.left {
		keys[k] = true
	}
	for k := range keys {
		src := g.describe(k)
		row := GrowthSourceRow{Source: src, Left: g.left[k], Contacts: g.contacts[k]}
		if g.list != uuid.Nil {
			// With one list picked, a source's rates are about the members
			// it brought to that list in the period: under 50 of them read
			// "Too few to compare".
			row.Contacts = g.joined[k]
		}
		j := g.joined[k]
		row.sortJoined = j
		if src.Imported && !g.in.Imports {
			// counted apart
		} else {
			row.Joined, row.Net = ptr(j), ptr(j-g.left[k])
			p.Total.Joined += j
		}
		p.Total.Left += g.left[k]
		p.Sources = append(p.Sources, row)
	}
	p.Total.Net = p.Total.Joined - p.Total.Left
	sortSources(p.Sources, g.list != uuid.Nil)
	return nil
}

// describe resolves a key with the period's details.
func (g *gather) describe(key string) GrowthSource {
	src := g.n.describe(key)
	if h := g.hosts[key]; len(h) > 0 {
		src.Hosts = h
	}
	if t, ok := g.firstJoin[key]; ok {
		src.FirstJoin = &t
	}
	return src
}

// quality fills each source's rates from the daily rollups.
func (g *gather) quality(ctx context.Context) error {
	p := g.page
	byKey := map[string]rateAgg{}
	var ws rateAgg
	rows, err := g.s.Pool.Query(ctx, `SELECT dim, key, sum(sent), sum(delivered), sum(bounced), sum(complained), sum(human_opened)
		FROM growth_quality_daily WHERE workspace_id = $1 AND day >= $2::date AND day < $3::date GROUP BY 1, 2`,
		g.ws.ID, g.w.fromDay.Format("2006-01-02"), g.w.to.Format("2006-01-02"))
	if err != nil {
		return err
	}
	for rows.Next() {
		var dim, key string
		var a rateAgg
		if err := rows.Scan(&dim, &key, &a.sent, &a.delivered, &a.bounced, &a.complained, &a.opened); err != nil {
			rows.Close()
			return err
		}
		switch dim {
		case "workspace":
			ws = a
		case "source":
			byKey[key] = a
		case "utm":
			byKey["utm|"+key] = a
		}
	}
	rows.Close()
	p.Workspace = GrowthRates{Sent: ws.sent, Delivered: ws.delivered, Bounce: ratio(ws.bounced, ws.sent), Complaint: ratio(ws.complained, ws.delivered), HumanOpen: ratio(ws.opened, ws.delivered)}
	for i := range p.Sources {
		r := &p.Sources[i]
		a := byKey[r.Source.Key]
		r.Sent, r.Delivered = a.sent, a.delivered
		switch {
		case r.Contacts < minContacts:
			r.TooFew = "contacts"
		case a.delivered < minDelivered:
			r.TooFew = "messages"
		default:
			r.BounceRate = ptr(ratio(a.bounced, a.sent))
			r.ComplaintRate = ptr(ratio(a.complained, a.delivered))
			r.HumanOpenRate = ptr(ratio(a.opened, a.delivered))
			if p.Workspace.Bounce > 0 && *r.BounceRate > flagFactor*p.Workspace.Bounce {
				r.BounceFlag = ptr(*r.BounceRate / p.Workspace.Bounce)
			}
			if p.Workspace.Complaint > 0 && *r.ComplaintRate > flagFactor*p.Workspace.Complaint {
				r.ComplaintFlag = ptr(*r.ComplaintRate / p.Workspace.Complaint)
			}
		}
	}
	g.utmRates = byKey
	return nil
}

// utmKey is how a UTM combination is keyed in the rollup.
func utmKey(source, medium, campaign string) string {
	if source == "" && medium == "" && campaign == "" {
		return "none"
	}
	return strings.ToLower(source + "/" + medium + "/" + campaign)
}

func (g *gather) rateOf(key string) *float64 {
	a, ok := g.utmRates["utm|"+key]
	if !ok || a.delivered == 0 {
		return nil
	}
	return ptr(ratio(a.opened, a.delivered))
}

func (g *gather) memberJoin() string {
	if g.list == uuid.Nil {
		return ""
	}
	return " AND EXISTS (SELECT 1 FROM list_memberships lm WHERE lm.contact_id = g.contact_id AND lm.list_id = '" + g.list.String() + "')"
}

func (g *gather) utm(ctx context.Context) error {
	p := g.page
	st, err := g.s.Q.GrowthStateGet(ctx, g.ws.ID)
	if err == nil {
		p.UTM.CaptureStartedAt = st.CaptureStartedAt
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	p.UTM.Rows = []GrowthUTMRow{}
	rows, err := g.s.Pool.Query(ctx, `SELECT g.captured, g.utm_source, g.utm_medium, g.utm_campaign, count(*) FROM contact_growth g
		WHERE g.workspace_id = $1 AND g.joined_at >= $2 AND g.joined_at < $3 AND g.source_kind IN ('form', 'subscribe_page')`+g.memberJoin()+`
		GROUP BY 1, 2, 3, 4 ORDER BY 5 DESC, 2, 3, 4`, g.ws.ID, g.w.from, g.w.to)
	if err != nil {
		return err
	}
	type combo struct {
		row GrowthUTMRow
		key string
	}
	var combos []combo
	for rows.Next() {
		var captured bool
		var r GrowthUTMRow
		if err := rows.Scan(&captured, &r.Source, &r.Medium, &r.Campaign, &r.Joins); err != nil {
			rows.Close()
			return err
		}
		p.UTM.Joins += r.Joins
		if !captured {
			p.UTM.NotCaptured += r.Joins
			continue
		}
		p.UTM.Captured += r.Joins
		k := utmKey(r.Source, r.Medium, r.Campaign)
		if k == "none" {
			p.UTM.None.Joins += r.Joins
			continue
		}
		combos = append(combos, combo{r, k})
	}
	rows.Close()
	p.UTM.None.HumanOpenRate = g.rateOf("none")
	p.UTM.NotCapRate = g.rateOf("not_captured")
	var otherOpened, otherDelivered int64
	for i, c := range combos {
		if i < topRows {
			c.row.HumanOpenRate = g.rateOf(c.key)
			p.UTM.Rows = append(p.UTM.Rows, c.row)
			continue
		}
		p.UTM.Other.Count++
		p.UTM.Other.Joins += c.row.Joins
		if a, ok := g.utmRates["utm|"+c.key]; ok {
			otherOpened += a.opened
			otherDelivered += a.delivered
		}
	}
	if otherDelivered > 0 {
		p.UTM.Other.HumanOpenRate = ptr(ratio(otherOpened, otherDelivered))
	}
	return nil
}

func (g *gather) referrers(ctx context.Context) error {
	p := g.page
	p.Referrers.Rows = []GrowthReferrerRow{}
	rows, err := g.s.Pool.Query(ctx, `SELECT g.referrer, count(*) FROM contact_growth g
		WHERE g.workspace_id = $1 AND g.joined_at >= $2 AND g.joined_at < $3 AND g.source_kind IN ('form', 'subscribe_page') AND g.captured`+g.memberJoin()+`
		GROUP BY 1 ORDER BY 2 DESC, 1`, g.ws.ID, g.w.from, g.w.to)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var r GrowthReferrerRow
		if err := rows.Scan(&r.Page, &r.Joins); err != nil {
			return err
		}
		p.Referrers.Joins += r.Joins
		switch {
		case r.Page == "":
			p.Referrers.None += r.Joins
		case len(p.Referrers.Rows) < topRows:
			p.Referrers.Rows = append(p.Referrers.Rows, r)
		default:
			p.Referrers.Other.Count++
			p.Referrers.Other.Joins += r.Joins
		}
	}
	return rows.Err()
}

// cohorts are the last months' joiners (imports left out), and whether
// they were still subscribed 30 and 90 days after joining.
func (g *gather) cohorts(ctx context.Context) error {
	p := g.page
	now := g.s.now()
	l := now.In(g.w.loc)
	first := time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, g.w.loc).AddDate(0, -4, 0)
	cond, args := "", []any{g.ws.ID, first, now}
	if g.in.CohortSource != "" {
		kind, id := splitKey(g.in.CohortSource)
		cond = " AND g.source_kind = $4 AND g.source_id = $5"
		args = append(args, kind, id)
	}
	rows, err := g.s.Pool.Query(ctx, `SELECT to_char(g.joined_at AT TIME ZONE '`+g.w.loc.String()+`', 'YYYY-MM') AS m, count(*),
			count(*) FILTER (WHERE g.joined_at + interval '30 days' <= $3),
			count(*) FILTER (WHERE g.joined_at + interval '30 days' <= $3 AND (g.left_at IS NULL OR g.left_at >= g.joined_at + interval '30 days')),
			count(*) FILTER (WHERE g.joined_at + interval '90 days' <= $3),
			count(*) FILTER (WHERE g.joined_at + interval '90 days' <= $3 AND (g.left_at IS NULL OR g.left_at >= g.joined_at + interval '90 days')),
			min(g.joined_at)
		FROM contact_growth g
		WHERE g.workspace_id = $1 AND g.joined_at >= $2 AND g.source_kind <> 'import'`+cond+g.memberJoin()+`
		GROUP BY 1 ORDER BY 1`, args...)
	if err != nil {
		return err
	}
	type cohortRow struct {
		month                 string
		n, r30, k30, r90, k90 int64
		firstJoin             time.Time
	}
	var cs []cohortRow
	for rows.Next() {
		var c cohortRow
		if err := rows.Scan(&c.month, &c.n, &c.r30, &c.k30, &c.r90, &c.k90, &c.firstJoin); err != nil {
			rows.Close()
			return err
		}
		cs = append(cs, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	// At most four rows, oldest complete ones first: the current month
	// only shows when fewer months have joiners.
	if len(cs) > 4 {
		cs = cs[len(cs)-5 : len(cs)-1]
	}
	var sourceFirst *time.Time
	if g.in.CohortSource != "" {
		if t, ok := g.firstJoin[g.in.CohortSource]; ok {
			sourceFirst = &t
		}
	}
	mark := func(c cohortRow, days int, reached, kept int64) GrowthRetention {
		month, _ := time.ParseInLocation("2006-01", c.month, g.w.loc)
		end := month.AddDate(0, 1, -1) // last day of the month
		r := GrowthRetention{}
		switch {
		case reached == c.n && c.n > 0:
			r.State, r.Rate = "done", ptr(ratio(kept, c.n))
		case reached == 0:
			r.State = "not_started"
			r.CompleteOn = end.AddDate(0, 0, days).Format("2006-01-02")
			if sourceFirst != nil && !sourceFirst.Before(month) && sourceFirst.Before(month.AddDate(0, 1, 0)) {
				r.From = sourceFirst.In(g.w.loc).AddDate(0, 0, days).Format("2006-01-02")
			}
		default:
			r.State = "in_progress"
			r.CompleteOn = end.AddDate(0, 0, days).Format("2006-01-02")
		}
		return r
	}
	for _, c := range cs {
		p.Cohorts = append(p.Cohorts, GrowthCohort{Month: c.month, Contacts: c.n, After30: mark(c, 30, c.r30, c.k30), After90: mark(c, 90, c.r90, c.k90)})
	}
	// The picker: sources with joins in the cohort months.
	srcRows, err := g.s.Pool.Query(ctx, `SELECT g.source_kind || CASE WHEN g.source_id <> '' THEN ':' || g.source_id ELSE '' END, count(*) FROM contact_growth g
		WHERE g.workspace_id = $1 AND g.joined_at >= $2 AND g.source_kind <> 'import'`+g.memberJoin()+` GROUP BY 1 ORDER BY 2 DESC`, g.ws.ID, first)
	if err != nil {
		return err
	}
	defer srcRows.Close()
	for srcRows.Next() {
		var k string
		var c int64
		if err := srcRows.Scan(&k, &c); err != nil {
			return err
		}
		p.CohortSources = append(p.CohortSources, g.describe(k))
	}
	return srcRows.Err()
}

// imports fills the import tile's items and the only-imports state.
func (g *gather) imports(ctx context.Context) error {
	p := g.page
	p.Tiles.ImportItems = []GrowthImportItem{}
	took := map[string]int64{}
	rows, err := g.s.Pool.Query(ctx, `SELECT g.source_id, count(*), min(g.joined_at) FROM contact_growth g
		WHERE g.workspace_id = $1 AND g.source_kind = 'import' GROUP BY 1`, g.ws.ID)
	if err != nil {
		return err
	}
	type imp struct {
		key string
		n   int64
		at  time.Time
	}
	var all []imp
	for rows.Next() {
		var id string
		var x imp
		if err := rows.Scan(&id, &x.n, &x.at); err != nil {
			rows.Close()
			return err
		}
		x.key = SourceKey(KindImport, id)
		took[x.key] = x.n
		all = append(all, x)
	}
	rows.Close()
	slices.SortFunc(all, func(a, b imp) int { return a.at.Compare(b.at) })
	for _, x := range all {
		src := g.n.describe(x.key)
		item := GrowthImportItem{Key: x.key, Name: src.Name, Day: x.at.In(g.w.loc).Format("2006-01-02"), Count: x.n}
		p.Imports = append(p.Imports, item)
		if g.list == uuid.Nil && !x.at.Before(g.w.from) && x.at.Before(g.w.to) {
			p.Tiles.ImportItems = append(p.Tiles.ImportItems, item)
		}
	}
	for i := range p.Sources {
		if p.Sources[i].Source.Kind == KindImport {
			p.Sources[i].Source.TookImportDay = took[p.Sources[i].Source.Key]
		}
	}
	// Joins only ever came from imports: Growth has nothing to show yet.
	var others int64
	if err := g.s.Pool.QueryRow(ctx, `SELECT count(*) FROM contact_growth WHERE workspace_id = $1 AND source_kind NOT IN ('import', 'carried')`, g.ws.ID).Scan(&others); err != nil {
		return err
	}
	p.OnlyImports = others == 0 && len(all) > 0
	return nil
}

func location(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil {
		return loc
	}
	return time.UTC
}
