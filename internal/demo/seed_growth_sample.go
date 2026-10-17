package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Growth sources (audience-insights brief §4, GrowthSources and
// GrowthStates): how each contact first joined and when it left, the first
// touch of the last 30 days' signups, and the daily send quality per
// source and UTM combination.

// sgGrowth is a contact's growth row.
type sgGrowth struct {
	src      string // a sample source (sgNL…)
	joined   time.Time
	left     *time.Time
	leftKind string
	utm      [3]string
	page     string
	ref      string
	captured bool
}

// The sample's sources.
const (
	sgNL     = "nl"     // Newsletter signup
	sgFooter = "footer" // Footer mini form
	sgSub    = "sub"    // the subscribe page
	sgBeta   = "beta"   // Beta apply
	sgAPI    = "api"    // API key "Website signup form"
	sgZapier = "zapier" // Zapier · Beta application (Typeform)
	sgPerson = "person" // added by a person
	sgBrevo  = "brevo"  // took the Brevo import's day
	sgMC     = "mc"     // the Mailchimp import
	sgCSV    = "csv"    // carried: the CSV import of Feb 2024
	sgDocs   = "docs"   // carried: Brevo's "Signup form · docs"
)

// key is a source's key in contact_growth and the rollups.
func (w *sgWorld) key(src string) (kind, id string) {
	switch src {
	case sgNL:
		return "form", w.forms["Newsletter signup"].String()
	case sgFooter:
		return "form", w.forms["Footer mini form"].String()
	case sgBeta:
		return "form", w.forms["Beta apply"].String()
	case sgSub:
		return "subscribe_page", ""
	case sgAPI:
		return "api_key", w.apiKey.String()
	case sgZapier:
		return "integration", w.zapier.String() + ":" + sgWorkflowKey
	case sgPerson:
		return "person", ""
	case sgBrevo:
		return "import", w.brevoImport.String()
	case sgMC:
		return "import", w.mcImport.String()
	case sgCSV:
		return "carried", "import:csv"
	case sgDocs:
		return "carried", "form:docs"
	}
	return "unknown", ""
}

func (w *sgWorld) keyString(src string) string {
	k, id := w.key(src)
	if id == "" {
		return k
	}
	return k + ":" + id
}

// The last 30 days' joins by source (the board's table), and the contacts
// each source has in all.
var (
	sgPeriodJoins = map[string]int{sgNL: 731, sgFooter: 612, sgSub: 104, sgBeta: 43, sgAPI: 214, sgZapier: 38, sgPerson: 14}
	sgPreNatives  = map[string]int{sgNL: 2300, sgFooter: 1500, sgSub: 450, sgBeta: 125, sgAPI: 562}
	sgCarried     = map[string]int{sgNL: 10123, sgFooter: 7388, sgSub: 1546, sgAPI: 4024}
	sgSourceOrder = []string{sgNL, sgFooter, sgSub, sgBeta, sgAPI, sgZapier, sgPerson}
)

// Contacts carried over with Brevo's own "Signup form · docs", and the
// contacts that took the import's day (Brevo had no date or source).
const (
	sgDocsCarried = 3100
	sgBrevoDay    = 7391
)

// sgCohorts are the four months before this one: joiners (imports left
// out) and how many left within 30 days (and, for the oldest, within 90).
var sgCohorts = []struct{ joined, left30, left90 int }{{1402, 57, 123}, {1288, 48, 0}, {1351, 66, 0}, {1612, 0, 0}}

func (w *sgWorld) growth(ctx context.Context) error {
	if err := w.growthSources(); err != nil {
		return err
	}
	if err := w.growthDepartures(); err != nil {
		return err
	}
	w.growthTouches()
	if err := w.writeGrowth(ctx); err != nil {
		return err
	}
	if err := w.growthQuality(ctx); err != nil {
		return err
	}
	if err := w.betaMemberships(ctx); err != nil {
		return err
	}
	return w.growthView(ctx)
}

// growthSources decides every contact's source and join date.
func (w *sgWorld) growthSources() error {
	set := func(c *sgContact, src string, at time.Time) {
		c.g.src, c.g.joined = src, at
	}
	// Imports: Mailchimp, the CSV import Brevo carried (Grace's date), and
	// the contacts that took the Brevo import's day.
	csvAt := w.named("grace").created
	for _, c := range w.contacts {
		switch c.source {
		case "import:mailchimp":
			set(c, sgMC, c.created)
			w.used[c] = sgMC
		case "import:csv":
			set(c, sgCSV, csvAt)
			w.used[c] = sgCSV
		case "import:brevo":
			set(c, sgBrevo, w.brevoAt)
			w.used[c] = sgBrevo
		}
	}
	// The board's contacts: their own dates, under the sample's sources.
	named := map[string]string{"priya": sgDocs, "diego": sgNL, "hannah": sgAPI, "kenji": sgFooter, "omar": sgNL, "tomasz": sgNL, "aisha": sgBeta}
	for key, src := range named {
		c := w.named(key)
		set(c, src, c.created)
		w.used[c] = "named"
	}
	brevo := 0
	for _, c := range w.contacts {
		if c.g.src == sgBrevo {
			brevo++
		}
	}
	others := w.shuffled(func(c *sgContact) bool {
		return c.importID != nil && !c.pending && c.board == "" && c.leftList == 0 && (c.source == "form:pricing" || c.source == "form:blog" || c.source == "form:popup")
	})
	extra, err := w.take(others, sgBrevoDay-brevo, sgBrevo)
	if err != nil {
		return err
	}
	for _, c := range extra {
		set(c, sgBrevo, w.brevoAt)
	}

	// The last 30 days' joins, by day.
	beta := func(c *sgContact) bool {
		return c.lists&audInBeta != 0 && !c.pending && c.board == "" && c.leftList == 0
	}
	natives := func(c *sgContact) bool { return c.importID == nil && !c.pending && c.board == "" }
	byPref := func(keep func(*sgContact) bool) []*sgContact {
		pool := w.shuffled(keep)
		sort.SliceStable(pool, func(i, j int) bool { return natives(pool[i]) && !natives(pool[j]) })
		return pool
	}
	open := func(c *sgContact) bool {
		return !c.pending && c.board == "" && c.leftList == 0 && c.source != "import:mailchimp" && c.source != "import:csv" && c.source != "import:brevo"
	}
	roles := []struct {
		src  string
		n    int
		pool []*sgContact
	}{
		{sgZapier, sgPeriodJoins[sgZapier], byPref(beta)},
		{sgBeta, sgPeriodJoins[sgBeta] - 1, byPref(beta)}, // Aisha is the 43rd
		{"nl-beta", 7, byPref(beta)},
		{"person-beta", 6, byPref(beta)},
		{sgPerson, sgPeriodJoins[sgPerson] - 6, byPref(open)},
		{sgAPI, sgPeriodJoins[sgAPI], byPref(func(c *sgContact) bool { return open(c) && c.source == "api" })},
		{sgSub, sgPeriodJoins[sgSub], byPref(open)},
		{sgFooter, sgPeriodJoins[sgFooter], byPref(func(c *sgContact) bool { return open(c) && c.source == "form:footer" })},
		{sgNL, sgPeriodJoins[sgNL] - 7, byPref(open)},
	}
	period := map[string][]*sgContact{}
	for _, r := range roles {
		cs, err := w.take(r.pool, r.n, r.src)
		if err != nil {
			return err
		}
		src := r.src
		switch r.src {
		case "nl-beta":
			src = sgNL
		case "person-beta":
			src = sgPerson
		}
		for _, c := range cs {
			c.g.src = src
		}
		period[src] = append(period[src], cs...)
	}
	period[sgBeta] = append(period[sgBeta], w.named("aisha"))
	for _, src := range sgSourceOrder {
		times := w.periodJoinTimes(src, len(period[src]))
		cs := period[src]
		for i, c := range cs {
			if c.board == "aisha" {
				continue // her own date, 23 days ago
			}
			c.g.joined = times[i]
		}
	}

	// Earlier joins: the cohort months, then everything before them.
	l := w.now.In(w.loc)
	month := time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, w.loc)
	cohortStart := month.AddDate(0, -4, 0)
	inCohort := make([]int, 4)
	monthOf := func(t time.Time) int {
		t = t.In(w.loc)
		for i := 0; i < 4; i++ {
			from := cohortStart.AddDate(0, i, 0)
			if !t.Before(from) && t.Before(from.AddDate(0, 1, 0)) {
				return i
			}
		}
		return -1
	}
	for _, c := range w.contacts {
		if c.g.src == "" || c.g.src == sgBrevo || c.g.src == sgMC || c.g.src == sgCSV {
			continue
		}
		if i := monthOf(c.g.joined); i >= 0 {
			inCohort[i]++
		}
	}
	// Natives first joined through a form once it existed.
	formSince := map[string]time.Time{sgNL: w.now.AddDate(0, 0, -210), sgFooter: w.now.AddDate(0, 0, -160), sgBeta: w.now.AddDate(0, 0, -63),
		sgSub: w.now.AddDate(0, 0, -150), sgAPI: w.now.AddDate(0, 0, -200)}
	natLeft := map[string]int{}
	for src, n := range sgPreNatives {
		natLeft[src] = n
	}
	natLeft[sgAPI]-- // Hannah
	natLeft[sgNL]--  // Tomasz
	carLeft := map[string]int{}
	for src, n := range sgCarried {
		carLeft[src] = n
	}
	carLeft[sgNL] -= 2 // Diego and Omar
	carLeft[sgFooter]--
	type slot struct {
		src     string
		from    time.Time
		to      time.Time
		carried bool
	}
	var slots []slot
	for i, co := range sgCohorts {
		from := cohortStart.AddDate(0, i, 0)
		to := from.AddDate(0, 1, 0)
		if to.After(w.periodStart) {
			to = w.periodStart
		}
		need := co.joined - inCohort[i]
		if need <= 0 || !to.After(from) {
			continue
		}
		nat := int(math.Round(float64(need) * 0.6))
		// Natives by source: Beta apply's (a form since August) all in the
		// months it existed, then the rest in proportion to what each
		// still has, among the forms that existed.
		alloc := map[string]int{}
		if formSince[sgBeta].Before(to) {
			alloc[sgBeta] = min(natLeft[sgBeta], nat/2)
		}
		rest := nat - alloc[sgBeta]
		var srcs []string
		var weights []float64
		for _, src := range []string{sgNL, sgFooter, sgSub, sgAPI} {
			if natLeft[src] > 0 && formSince[src].Before(to) {
				srcs = append(srcs, src)
				weights = append(weights, float64(natLeft[src]))
			}
		}
		for j, n := range Split(int64(rest), weights) {
			alloc[srcs[j]] += int(n)
		}
		for _, src := range []string{sgBeta, sgNL, sgFooter, sgSub, sgAPI} {
			n := min(alloc[src], natLeft[src])
			natLeft[src] -= n
			f := from
			if formSince[src].After(f) {
				f = formSince[src]
			}
			for k := 0; k < n; k++ {
				slots = append(slots, slot{src, f, to, false})
			}
		}
		car := need - nat
		srcs, weights = nil, nil
		for _, src := range []string{sgNL, sgFooter, sgSub, sgAPI} {
			srcs = append(srcs, src)
			weights = append(weights, float64(carLeft[src]))
		}
		for j, n := range Split(int64(car), weights) {
			carLeft[srcs[j]] -= int(n)
			for k := int64(0); k < n; k++ {
				slots = append(slots, slot{srcs[j], from, to, true})
			}
		}
	}
	// Natives left over joined between their form's launch and the cohorts.
	for _, src := range []string{sgNL, sgFooter, sgSub, sgBeta, sgAPI} {
		for k := 0; k < natLeft[src]; k++ {
			from := formSince[src]
			to := cohortStart
			if !to.After(from) {
				to = w.periodStart
			}
			slots = append(slots, slot{src, from, to, false})
		}
		natLeft[src] = 0
	}
	// Contacts carried with an older date, from three years back.
	oldest := w.now.AddDate(-3, 0, 0)
	for _, src := range []string{sgNL, sgFooter, sgSub, sgAPI} {
		for k := 0; k < carLeft[src]; k++ {
			slots = append(slots, slot{src, oldest, cohortStart, true})
		}
	}
	docs := sgDocsCarried - 1 // Priya
	for k := 0; k < docs; k++ {
		slots = append(slots, slot{sgDocs, oldest, cohortStart.AddDate(0, -1, 0), true})
	}
	// The contacts that fill them: W02's sources where they fit.
	prefer := map[string]string{sgFooter: "form:footer", sgAPI: "api", sgDocs: "form:docs"}
	rest := w.shuffled(func(c *sgContact) bool { _, taken := w.used[c]; return !taken })
	bySource := map[string][]*sgContact{}
	for _, c := range rest {
		bySource[c.source] = append(bySource[c.source], c)
	}
	pick := func(src string) *sgContact {
		if s, ok := prefer[src]; ok && len(bySource[s]) > 0 {
			c := bySource[s][0]
			bySource[s] = bySource[s][1:]
			return c
		}
		for _, s := range []string{"form:docs", "form:blog", "form:pricing", "form:popup", "form:footer", "api"} {
			for len(bySource[s]) > 0 {
				c := bySource[s][0]
				bySource[s] = bySource[s][1:]
				if _, taken := w.used[c]; !taken {
					return c
				}
			}
		}
		return nil
	}
	// The scarcer preferences first, so docs and footer get their own.
	sort.SliceStable(slots, func(i, j int) bool { return slots[i].src == sgDocs && slots[j].src != sgDocs })
	for _, sl := range slots {
		c := pick(sl.src)
		if c == nil {
			return fmt.Errorf("growth: no contact left for %s", sl.src)
		}
		w.used[c] = "joined:" + sl.src
		span := sl.to.Sub(sl.from)
		c.g.src = sl.src
		c.g.joined = sl.from.Add(time.Duration(w.rng.Int64N(int64(span)))).Truncate(time.Second)
	}
	for _, c := range w.contacts {
		if c.g.src == "" {
			return fmt.Errorf("growth: %s has no source", c.email)
		}
	}
	return nil
}

// periodJoinTimes spreads a source's joins over the last 30 days: fewer
// at weekends, Hacker News on day 12, the LinkedIn campaign the week after.
func (w *sgWorld) periodJoinTimes(src string, n int) []time.Time {
	weights := make([]float64, 30)
	for d := 29; d >= 0; d-- {
		day := w.day(d)
		x := w.sgWeekday(day)
		switch src {
		case sgNL:
			switch {
			case d == 12:
				x *= 2.3
			case d >= 5 && d <= 8:
				x *= 1.55
			case d <= 2:
				x *= 1.45
			}
		case sgFooter:
			if d == 12 {
				x *= 1.9
			}
		case sgSub:
			if d == 12 {
				x *= 9
			}
		case sgZapier:
			if d > 21 {
				x = 0
			}
		}
		if d == 0 {
			// Today counts until now.
			l := w.now.In(w.loc)
			x *= math.Max(0.05, float64(l.Hour()*60+l.Minute())/(24*60))
		}
		weights[29-d] = x
	}
	counts := Split(int64(n), weights)
	var out []time.Time
	for i, k := range counts {
		d := 29 - i
		for j := int64(0); j < k; j++ {
			out = append(out, w.sgAt(w.day(d), 7, 22))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	if src == sgZapier && len(out) > 0 {
		out[0] = w.s.hubDay(21, 10, 19) // the Zap's first call
	}
	w.rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// sgDepartures are the last 30 days' departures by source and kind.
var sgDepartures = []struct {
	src                            string
	unsub, hard, complaint, erased int
}{
	{sgBrevo, 262, 35, 4, 0},
	{sgMC, 14, 30, 20, 0},
	{sgAPI, 9, 20, 0, 2},
	{sgNL, 26, 9, 1, 3},
	{sgFooter, 20, 7, 1, 3},
	{sgSub, 1, 2, 0, 0},
	{sgCSV, 1, 1, 1, 0},
}

// growthDepartures marks who left: the cohorts' leavers before the 30
// days, then the 30 days' 472 (eight of them erased since).
func (w *sgWorld) growthDepartures() error {
	l := w.now.In(w.loc)
	cohortStart := time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, w.loc).AddDate(0, -4, 0)
	before := w.periodStart.Add(-2 * time.Hour)
	leave := func(c *sgContact, at time.Time, kind string) {
		c.g.left, c.g.leftKind = &at, kind
	}
	kindOf := func(i int) string {
		switch i % 20 {
		case 0, 7, 13:
			return "hard_bounce"
		case 10:
			return "complaint"
		}
		return "unsubscribed"
	}
	for i, co := range sgCohorts {
		from := cohortStart.AddDate(0, i, 0)
		to := from.AddDate(0, 1, 0)
		pool := w.shuffled(func(c *sgContact) bool {
			return c.board == "" && c.leftList == 0 && c.g.left == nil && c.g.src != sgBrevo && c.g.src != sgMC && c.g.src != sgCSV &&
				!c.g.joined.Before(from) && c.g.joined.Before(to) && c.g.joined.Before(before.AddDate(0, 0, -2)) &&
				w.used[c] != sgZapier && w.used[c] != sgBeta && w.used[c] != sgPerson && w.used[c] != "nl-beta" && w.used[c] != "person-beta"
		})
		k := 0
		mark := func(n, minDays, maxDays int) error {
			for done := 0; done < n; k++ {
				if k >= len(pool) {
					return fmt.Errorf("cohort %d: %d leavers short", i, n-done)
				}
				c := pool[k]
				at := c.g.joined.Add(time.Duration(minDays)*24*time.Hour + time.Duration(w.rng.Int64N(int64(time.Duration(maxDays-minDays)*24*time.Hour))))
				if !at.Before(before) {
					continue
				}
				leave(c, at.Truncate(time.Second), kindOf(done))
				done++
			}
			return nil
		}
		if err := mark(co.left30, 1, 29); err != nil {
			return err
		}
		if co.left90 > co.left30 {
			if err := mark(co.left90-co.left30, 31, 89); err != nil {
				return err
			}
		}
	}

	// The last 30 days: after the Brevo import, mostly after a send.
	sends := map[int]float64{}
	for _, x := range audSends(w.s) {
		d := int(w.day(0).Sub(x.at.In(w.loc)).Hours()/24) + 1
		if x.at.After(w.day(0)) {
			d = 0
		}
		if d >= 0 && d <= 24 {
			sends[d] += 4
			if d > 0 {
				sends[d-1] += 1.5
			}
		}
	}
	importDay := int(w.day(0).Sub(w.brevoAt.In(w.loc)).Hours()/24) + 1
	dayWeights := func(src string) []float64 {
		ws := make([]float64, 30)
		for d := 29; d >= 0; d-- {
			x := 0.0
			if d <= importDay {
				x = 0.6 + sends[d]
				if src == sgBrevo || src == sgMC {
					switch importDay - d {
					case 0:
						x += 12
					case 1:
						x += 8
					case 2:
						x += 3
					}
				}
			}
			ws[29-d] = x
		}
		return ws
	}
	departAt := func(src string, n int) []time.Time {
		var out []time.Time
		for i, k := range Split(int64(n), dayWeights(src)) {
			d := 29 - i
			for j := int64(0); j < k; j++ {
				t := w.sgAt(w.day(d), 8, 23)
				if d == importDay && t.Before(w.brevoAt.Add(70*time.Minute)) {
					t = w.brevoAt.Add(time.Duration(70+w.rng.IntN(300)) * time.Minute)
					if limit := w.now.Add(-10 * time.Minute); t.After(limit) {
						t = limit
					}
				}
				out = append(out, t)
			}
		}
		w.rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
		return out
	}
	for _, d := range sgDepartures {
		total := d.unsub + d.hard + d.complaint + d.erased
		times := departAt(d.src, total)
		pool := w.shuffled(func(c *sgContact) bool {
			if c.g.src != d.src || c.board != "" || c.g.left != nil || c.leftList != 0 {
				return false
			}
			switch w.used[c] {
			case sgZapier, sgBeta, sgPerson, "nl-beta", "person-beta":
				return false
			}
			// Leave the cohorts' 30- and 90-day numbers as they are.
			return c.g.joined.Before(cohortStart) || !c.g.joined.Before(w.periodStart) || d.src == sgBrevo || d.src == sgMC
		})
		k := 0
		kinds := []struct {
			kind string
			n    int
		}{{"unsubscribed", d.unsub}, {"hard_bounce", d.hard}, {"complaint", d.complaint}}
		ti := 0
		for _, kd := range kinds {
			for j := 0; j < kd.n; j++ {
				// The next contact who had joined an hour before.
				x := k
				for x < len(pool) && (!pool[x].g.joined.Before(times[ti].Add(-time.Hour)) || (kd.kind != "unsubscribed" && pool[x].lists&audInBeta != 0)) {
					x++
				}
				if x >= len(pool) {
					return fmt.Errorf("departures from %s: pool too small", d.src)
				}
				pool[k], pool[x] = pool[x], pool[k]
				leave(pool[k], times[ti], kd.kind)
				k++
				ti++
			}
		}
		// Erased since: their rows outlive the contact.
		for j := 0; j < d.erased; j++ {
			at := times[ti]
			ti++
			joined := w.brevoAt
			switch d.src {
			case sgMC:
				joined = w.mcAt
			case sgNL, sgFooter, sgAPI:
				joined = cohortStart.AddDate(0, -2-j, 3+j)
			}
			c := &sgContact{id: ID(), email: fmt.Sprintf("erased-%s-%d", d.src, j), board: "erased"}
			c.g = sgGrowth{src: d.src, joined: joined, left: &at, leftKind: "erased"}
			w.contacts = append(w.contacts, c)
		}
	}
	return nil
}

// sgUTM are the first touches of the last 30 days' 1,490 form and
// subscribe-page joins (the board's UTM and referring-page tables).
type sgTouch struct {
	utm   [3]string
	count map[string]int // by source
	refs  []sgRef
}

type sgRef struct {
	page string
	n    int
}

// sgOtherPages are the 52 referring pages the board folds into "Other".
var sgOtherPages = []string{
	"acme.io/blog", "acme.io/customers/northwind", "acme.io/changelog", "acme.io/docs", "acme.io/features/feature-flags-ga",
	"acme.io/about", "acme.io/careers", "acme.io/security", "acme.io/compare/launchdarkly", "acme.io/integrations",
	"docs.acme.io/getting-started", "docs.acme.io/environments", "docs.acme.io/audit-log", "docs.acme.io/api", "docs.acme.io/sdk/go",
	"docs.acme.io/sdk/python", "docs.acme.io/migrate-v2", "blog.acme.io/2026/09/boring-infrastructure", "blog.acme.io/2026/09/feature-flags-ga",
	"blog.acme.io/2026/08/postgres-vacuum", "blog.acme.io/2026/10/audit-exports", "blog.acme.io/tags/postgres", "dev.to/acme/feature-flags-ga",
	"dev.to/acme/postgres-partitioning", "reddit.com/r/postgresql", "reddit.com/r/devops", "lobste.rs/s/boring-infra", "bsky.app/profile/acme.io",
	"mastodon.social/@acme", "linkedin.com/feed", "x.com/acme", "youtube.com/watch", "producthunt.com/posts/acme-flags",
	"stackoverflow.com/questions/tagged/feature-flags", "github.com/acme/flags-go", "github.com/acme/flags-node", "github.com/acme/awesome-flags",
	"duckduckgo.com", "bing.com/search", "google.com", "news.google.com", "slack.com/archives", "discord.com/channels",
	"northwindlabs.com/blog", "tidewater.io/engineering", "kestrel-analytics.com/blog", "changelog.com/podcast/542",
	"pragmaticengineer.com", "infoq.com/news/2026/10/feature-flags", "thenewstack.io/feature-flags", "hn.algolia.com", "substack.com/@acme",
}

func sgTouches() []sgTouch {
	return []sgTouch{
		{utm: [3]string{"linkedin", "social", "flags-ga"}, count: map[string]int{sgNL: 120, sgFooter: 70, sgSub: 14, sgBeta: 10},
			refs: []sgRef{{"docs.acme.io/flags", 140}, {"acme.io/pricing", 50}, {"", -24}}},
		{utm: [3]string{"google", "cpc", "brand-newsletter"}, count: map[string]int{sgNL: 80, sgFooter: 70, sgSub: 18},
			refs: []sgRef{{"acme.io/pricing", 71}, {"blog.acme.io/2026/10/postgres-partitioning", 73}, {"", -24}}},
		{utm: [3]string{"github", "readme", ""}, count: map[string]int{sgNL: 70, sgFooter: 40, sgSub: 11},
			refs: []sgRef{{"github.com/acme/flags", 121}}},
		{utm: [3]string{"hackernews", "social", "boring-infra"}, count: map[string]int{sgNL: 44, sgFooter: 32, sgSub: 21},
			refs: []sgRef{{"news.ycombinator.com", 97}}},
		{utm: [3]string{"devto", "social", "flags-ga"}, count: map[string]int{sgNL: 30, sgFooter: 20, sgBeta: 8},
			refs: []sgRef{{"docs.acme.io/flags", 58}}},
		// "(none)": typed, bookmarked or from Acme's own pages.
		{count: map[string]int{sgNL: 332, sgFooter: 335, sgSub: 30, sgBeta: 19},
			refs: []sgRef{{"none", 493}, {"acme.io/pricing", 120}, {"blog.acme.io/2026/10/postgres-partitioning", 103}}},
	}
}

// sgOtherCombos are the 14 smaller combinations ("Other, 14 combinations").
var sgOtherCombos = []struct {
	utm [3]string
	n   int
}{
	{[3]string{"reddit", "social", "postgres-partitioning"}, 16}, {[3]string{"bluesky", "social", "boring-infra"}, 14},
	{[3]string{"newsletter", "email", "weekly-digest-141"}, 12}, {[3]string{"youtube", "video", "flags-walkthrough"}, 11},
	{[3]string{"mastodon", "social", "boring-infra"}, 10}, {[3]string{"podcast", "audio", "changelog-542"}, 9},
	{[3]string{"partner", "referral", "northwind"}, 8}, {[3]string{"bing", "cpc", "brand-newsletter"}, 7},
	{[3]string{"duckduckgo", "organic", ""}, 7}, {[3]string{"lobsters", "social", "postgres-partitioning"}, 6},
	{[3]string{"slack", "community", ""}, 5}, {[3]string{"discord", "community", ""}, 4},
	{[3]string{"stackoverflow", "referral", ""}, 4}, {[3]string{"producthunt", "launch", "feature-flags"}, 3},
}

// growthTouches gives the last 30 days' form and subscribe-page joins
// their first touch: the form's page, UTM parameters and referrer.
func (w *sgWorld) growthTouches() {
	pools := map[string][]*sgContact{}
	for _, c := range w.contacts {
		switch c.g.src {
		case sgNL, sgFooter, sgSub, sgBeta:
			if !c.g.joined.Before(w.periodStart) && c.board != "erased" {
				pools[c.g.src] = append(pools[c.g.src], c)
			}
		}
	}
	for _, p := range pools {
		sort.Slice(p, func(i, j int) bool { return p[i].g.joined.Before(p[j].g.joined) })
	}
	// Hacker News joins land on day 12; LinkedIn's the week after.
	hnDay := w.day(12)
	takeFrom := func(src string, n int, want func(*sgContact) bool) []*sgContact {
		var out []*sgContact
		rest := pools[src][:0:0]
		for _, c := range pools[src] {
			if len(out) < n && (want == nil || want(c)) {
				out = append(out, c)
				continue
			}
			rest = append(rest, c)
		}
		// Not enough on the wanted days: take any.
		for len(out) < n && len(rest) > 0 {
			out, rest = append(out, rest[0]), rest[1:]
		}
		pools[src] = rest
		return out
	}
	other := 0
	otherRefs := 0
	nextOtherRef := func() string {
		p := sgOtherPages[otherRefs%len(sgOtherPages)]
		otherRefs++
		return p
	}
	for _, t := range sgTouches() {
		var cs []*sgContact
		for _, src := range []string{sgSub, sgBeta, sgFooter, sgNL} {
			n := t.count[src]
			var want func(*sgContact) bool
			if t.utm[0] == "hackernews" {
				want = func(c *sgContact) bool {
					return !c.g.joined.Before(hnDay) && c.g.joined.Before(hnDay.AddDate(0, 0, 2))
				}
			}
			cs = append(cs, takeFrom(src, n, want)...)
		}
		w.rng.Shuffle(len(cs), func(i, j int) { cs[i], cs[j] = cs[j], cs[i] })
		i := 0
		for _, r := range t.refs {
			n := r.n
			if n < 0 {
				n = -n
			}
			for k := 0; k < n && i < len(cs); k++ {
				c := cs[i]
				i++
				c.g.utm = t.utm
				switch r.page {
				case "none":
					c.g.ref = ""
				case "":
					c.g.ref = nextOtherRef()
				default:
					c.g.ref = r.page
				}
			}
		}
		other += len(cs) - i
	}
	// The 14 smaller combinations, from what is left.
	var rest []*sgContact
	for _, src := range []string{sgNL, sgFooter, sgSub, sgBeta} {
		rest = append(rest, pools[src]...)
	}
	w.rng.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	i := 0
	for _, oc := range sgOtherCombos {
		for k := 0; k < oc.n && i < len(rest); k++ {
			rest[i].g.utm = oc.utm
			rest[i].g.ref = nextOtherRef()
			i++
		}
	}
	// Every 30-day form join was captured; the form's own page.
	blogPosts := []string{"blog.acme.io/2026/10/postgres-partitioning", "blog.acme.io/2026/09/boring-infrastructure", "blog.acme.io/2026/10/audit-exports", "blog.acme.io"}
	sitePages := []string{"acme.io", "acme.io/newsletter", "acme.io/pricing", "acme.io/features/feature-flags-ga", "acme.io/customers/northwind", "acme.io/changelog"}
	for _, c := range w.contacts {
		switch c.g.src {
		case sgNL, sgFooter, sgSub, sgBeta:
		default:
			continue
		}
		if c.g.joined.Before(w.periodStart) || c.board == "erased" {
			continue
		}
		c.g.captured = true
		switch c.g.src {
		case sgNL:
			if w.rng.IntN(100) < 62 {
				c.g.page = sitePages[w.rng.IntN(2)]
			} else {
				c.g.page = blogPosts[w.rng.IntN(len(blogPosts))]
			}
		case sgFooter:
			c.g.page = sitePages[w.rng.IntN(len(sitePages))]
		case sgSub:
			c.g.page = "news.acme.io/subscribe"
		case sgBeta:
			c.g.page = "blog.acme.io/2026/10/feature-flags-beta"
		}
	}
	_ = other
}

func (w *sgWorld) writeGrowth(ctx context.Context) error {
	var rows [][]any
	for _, c := range w.contacts {
		kind, id := w.key(c.g.src)
		var left any
		var leftKind any
		if c.g.left != nil {
			left, leftKind = *c.g.left, c.g.leftKind
		}
		rows = append(rows, []any{c.id, w.ws.ID, kind, id, c.g.joined, left, leftKind, c.g.utm[0], c.g.utm[1], c.g.utm[2], "", "",
			c.g.page, c.g.ref, sgDomain(c.g.ref), c.g.captured, w.now})
	}
	if err := sgCopy(ctx, w.s.Tx, "contact_growth", []string{"contact_id", "workspace_id", "source_kind", "source_id", "joined_at", "left_at", "left_kind",
		"utm_source", "utm_medium", "utm_campaign", "utm_content", "utm_term", "page", "referrer", "referrer_domain", "captured", "updated_at"}, rows); err != nil {
		return err
	}
	l := w.now.In(w.loc)
	yesterday := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	_, err := w.s.Tx.Exec(ctx, `INSERT INTO growth_state (workspace_id, capture_started_at, attributed_through, rolled_up_through, updated_at)
		VALUES ($1, $2, $3, $4, $3)`, w.ws.ID, w.periodStart.AddDate(0, 0, -1), w.now, yesterday)
	return err
}

func sgDomain(ref string) string {
	host := ref
	for i, r := range ref {
		if r == '/' {
			host = ref[:i]
			break
		}
	}
	return host
}

// sgQuality is one source's sends over the last 30 days: the board's
// rates (sent, bounced, complained, human opens of the delivered).
type sgQuality struct {
	key                               string
	sent, bounced, complained, opened int64
}

// growthQuality writes the daily send quality since the Brevo import, the
// day ReputeCast started sending: per source, per UTM combination and for
// the workspace. The sources' 30 days come to the board's rates; the
// contacts Brevo carried with its own "Signup form · docs" have none in
// the table but count in the workspace's 0.6% and 0.04%.
func (w *sgWorld) growthQuality(ctx context.Context) error {
	src := []sgQuality{
		{w.keyString(sgNL), 67000, 201, 10, 35003},
		{w.keyString(sgFooter), 50000, 200, 12, 23456},
		{w.keyString(sgSub), 9800, 29, 0, 4964},
		{w.keyString(sgBeta), 1350, 3, 0, 855},
		{w.keyString(sgAPI), 22000, 462, 7, 8529},
		{w.keyString(sgZapier), 70, 0, 0, 42},
		{w.keyString(sgPerson), 30, 0, 0, 16},
		{w.keyString(sgBrevo), 28500, 171, 12, 11615},
		{w.keyString(sgMC), 2265, 43, 4, 413},
		{w.keyString(sgCSV), 12000, 108, 6, 2890},
		{w.keyString(sgDocs), 12500, 19, 30, 5866},
	}
	// UTM combinations: the 30 days' signups, and the earlier form joins
	// (not captured).
	utm := []sgQuality{
		utmRow("none", 1418, 0.498), utmRow("linkedin/social/flags-ga", 431, 0.512), utmRow("google/cpc/brand-newsletter", 338, 0.44),
		utmRow("github/readme/", 243, 0.583), utmRow("hackernews/social/boring-infra", 193, 0.614), utmRow("devto/social/flags-ga", 116, 0.475),
		utmRow("not_captured", 96300, 0.502),
	}
	// The smaller combinations come to 46.9% together.
	var od, oo int64
	for i, oc := range sgOtherCombos {
		sent := int64(oc.n * 2)
		key := oc.utm[0] + "/" + oc.utm[1] + "/" + oc.utm[2]
		opened := int64(math.Round(float64(sent) * 0.469))
		if i == len(sgOtherCombos)-1 {
			opened = int64(math.Round(float64(od+sent)*0.469)) - oo
		}
		od += sent
		oo += opened
		utm = append(utm, sgQuality{key, sent, 0, 0, opened})
	}
	var ws sgQuality
	for _, q := range src {
		ws.sent += q.sent
		ws.bounced += q.bounced
		ws.complained += q.complained
		ws.opened += q.opened
	}
	// Days since the import, weighted by what went out: the digests, the
	// October Product Update, and the automations every day.
	importDay := int(w.day(0).Sub(w.brevoAt.In(w.loc)).Hours()/24) + 1
	weights := map[int]float64{}
	for d := importDay; d >= 1; d-- {
		weights[d] = 0.15
	}
	for _, x := range audSends(w.s) {
		d := int(w.day(0).Sub(x.at.In(w.loc)).Hours()/24) + 1
		if d >= 1 && d <= importDay {
			weights[d] += float64(x.lists&audInDigest)*0.7 + 0.3
		}
	}
	var days []int
	var ws_ []float64
	for d := importDay; d >= 1; d-- {
		days = append(days, d)
		ws_ = append(ws_, weights[d])
	}
	var rows [][]any
	add := func(dim string, q sgQuality) {
		sent, bounced, complained, opened := Split(q.sent, ws_), Split(q.bounced, ws_), Split(q.complained, ws_), Split(q.opened, ws_)
		for i, d := range days {
			if sent[i] == 0 && bounced[i] == 0 {
				continue
			}
			day := w.day(d).In(time.UTC)
			date := time.Date(w.day(d).Year(), w.day(d).Month(), w.day(d).Day(), 0, 0, 0, 0, time.UTC)
			_ = day
			b := min(bounced[i], sent[i])
			rows = append(rows, []any{w.ws.ID, date, dim, q.key, sent[i], sent[i] - b, b, complained[i], opened[i]})
		}
	}
	add("workspace", sgQuality{key: "", sent: ws.sent, bounced: ws.bounced, complained: ws.complained, opened: ws.opened})
	for _, q := range src {
		add("source", q)
	}
	for _, q := range utm {
		add("utm", q)
	}
	return sgCopy(ctx, w.s.Tx, "growth_quality_daily", []string{"workspace_id", "day", "dim", "key", "sent", "delivered", "bounced", "complained", "human_opened"}, rows)
}

// utmRow is about sent sends of a UTM combination whose human-open rate
// reads rate to one decimal (sent moves up until it does).
func utmRow(key string, sent int64, rate float64) sgQuality {
	for ; ; sent++ {
		bounced := int64(math.Round(float64(sent) * 0.003))
		delivered := sent - bounced
		opened := int64(math.Round(float64(delivered) * rate))
		if math.Round(float64(opened)/float64(delivered)*1000) == math.Round(rate*1000) {
			return sgQuality{key: key, sent: sent, bounced: bounced, opened: opened}
		}
	}
}

// betaMemberships reshapes Beta Program's last 30 days for the one-list
// view (108 joined, 27 left, +81 as List health shows): the members who
// joined through Beta apply, Zapier, the preference center, the newsletter
// form and a person, and the ones who left while staying on another list.
// The list's size and net growth don't change.
func (w *sgWorld) betaMemberships(ctx context.Context) error {
	tx := w.s.Tx
	beta := w.s.Lists["Beta Program"]
	// Every other member joined before the 30 days.
	if _, err := tx.Exec(ctx, `UPDATE list_memberships SET subscribed_at = $2 - (interval '1 day' * (31 + abs(hashtext(contact_id::text)) % 300))
		WHERE list_id = $1 AND status = 'subscribed' AND subscribed_at >= $2`, beta, w.periodStart); err != nil {
		return err
	}
	join := func(cs []*sgContact, source string, at func(*sgContact) time.Time) error {
		for _, c := range cs {
			if _, err := tx.Exec(ctx, `UPDATE list_memberships SET subscribed_at = $3, source = $4 WHERE list_id = $1 AND contact_id = $2`,
				beta, c.id, at(c), source); err != nil {
				return err
			}
		}
		return nil
	}
	byRole := func(role string) []*sgContact {
		var out []*sgContact
		for _, c := range w.contacts {
			if w.used[c] == role {
				out = append(out, c)
			}
		}
		return out
	}
	joined := func(c *sgContact) time.Time { return c.g.joined }
	if err := join(append(byRole(sgBeta), w.named("aisha")), "form:beta-apply", joined); err != nil {
		return err
	}
	if err := join(byRole(sgZapier), "api", joined); err != nil {
		return err
	}
	if err := join(byRole("nl-beta"), "form:newsletter-signup", joined); err != nil {
		return err
	}
	if err := join(byRole("person-beta"), "team", joined); err != nil {
		return err
	}
	// Fourteen members added Beta Program in the preference center.
	prefs, err := w.takeFree(w.shuffled(func(c *sgContact) bool {
		return c.lists&audInBeta != 0 && c.board == "" && c.g.left == nil && c.g.joined.Before(w.periodStart)
	}), 14, "preference-center")
	if err != nil {
		return err
	}
	if err := join(prefs, "preference_center", func(*sgContact) time.Time {
		return w.sgAt(w.day(1+w.rng.IntN(27)), 8, 22)
	}); err != nil {
		return err
	}
	// The 27 who left Beta Program and kept another list: 1.0's nine (six
	// came through the newsletter form, three with the CSV import), and
	// eighteen more.
	setSource := func(c *sgContact, src string) error {
		c.g.src = src
		kind, id := w.key(src)
		_, err := tx.Exec(ctx, `UPDATE contact_growth SET source_kind = $2, source_id = $3 WHERE contact_id = $1`, c.id, kind, id)
		return err
	}
	var old []*sgContact
	for _, c := range w.contacts {
		if c.leftList&audInBeta != 0 {
			old = append(old, c)
		}
	}
	for i, c := range old {
		src := sgNL
		if i >= 6 {
			src = sgCSV
		}
		if c.g.src != src && c.g.joined.Before(w.periodStart) {
			if err := setSource(c, src); err != nil {
				return err
			}
		}
	}
	others := func(c *sgContact) bool {
		return c.lists&audInBeta == 0 && c.leftList == 0 && c.lists != 0 && c.board == "" && c.g.left == nil
	}
	csv := 8 - max(0, len(old)-6)
	leavers := []struct {
		n      int
		source string
		keep   func(*sgContact) bool
	}{
		{4, "form:beta-apply", func(c *sgContact) bool { return others(c) && c.g.src == sgBeta && c.g.joined.Before(w.periodStart) }},
		{9, "preference_center", func(c *sgContact) bool { return others(c) && c.g.joined.Before(w.periodStart) }},
		{6 - min(6, len(old)), "form:newsletter-signup", func(c *sgContact) bool { return others(c) && c.g.src == sgNL && c.g.joined.Before(w.periodStart) }},
		{csv, "import:csv", func(c *sgContact) bool { return others(c) && c.g.src == sgCSV }},
	}
	var rows [][]any
	for _, lv := range leavers {
		cs, err := w.takeFree(w.shuffled(lv.keep), lv.n, "left-beta:"+lv.source)
		if err != nil {
			// The Beta apply leavers may not exist yet: make them.
			if lv.source != "form:beta-apply" {
				return err
			}
			more, err2 := w.takeFree(w.shuffled(func(c *sgContact) bool { return others(c) && c.g.src == sgNL && c.g.joined.Before(w.periodStart) }), lv.n-len(cs), "left-beta:"+lv.source)
			if err2 != nil {
				return err2
			}
			cs = append(cs, more...)
		}
		for _, c := range cs {
			if lv.source == "form:beta-apply" && c.g.src != sgBeta {
				// They joined through the beta form before the 30 days.
				if err := setSource(c, sgBeta); err != nil {
					return err
				}
			}
			from := c.g.joined
			if from.Before(w.now.AddDate(-1, 0, 0)) {
				from = w.now.AddDate(0, -8, 0)
			}
			left := w.sgAt(w.day(1+w.rng.IntN(26)), 8, 22)
			rows = append(rows, []any{beta, c.id, "unsubscribed", lv.source, from, left, from})
		}
	}
	return sgCopy(ctx, tx, "list_memberships", []string{"list_id", "contact_id", "status", "source", "subscribed_at", "unsubscribed_at", "created_at"}, rows)
}

// growthView is the saved view of the Zapier contacts (GrowthStates).
func (w *sgWorld) growthView(ctx context.Context) error {
	f, _ := json.Marshal(map[string]string{"growthSource": w.keyString(sgZapier)})
	_, err := w.s.Tx.Exec(ctx, `INSERT INTO saved_views (id, workspace_id, user_id, name, filters, columns, position, created_at, show_in_health)
		VALUES ($1,$2,NULL,'Beta applicants from Typeform',$3,'{}',7,$4,false)`, ID(), w.ws.ID, f, w.s.designDay(2, 11, 20))
	return err
}

// sgEngineeringBlog gives the second workspace its only source so far: a
// CSV import of 2 Oct (3,240 contacts took its day; the rest kept the date
// and source of the old platform).
func sgEngineeringBlog(ctx context.Context, s *Seeder) error {
	eng := s.Workspaces["acme-engineering-blog"]
	if eng.ID == uuid.Nil {
		return nil
	}
	at := s.designDay(13, 15, 20)
	imp := ID()
	stats, _ := json.Marshal(map[string]any{"imported": 6420, "updated": 0, "skipped": 0, "suppressed": 0})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO imports (id, workspace_id, source, status, file_name, stats, created_by, created_at, started_at, finished_at, step, updated_at)
		VALUES ($1,$2,'csv','done','engineering-blog-subscribers.csv',$3,$4,$5,$5,$6,'done',$6)`,
		imp, eng.ID, stats, s.Users["marcus@acme.io"].ID, at, at.Add(4*time.Minute)); err != nil {
		return err
	}
	rows, err := s.Tx.Query(ctx, `SELECT id, created_at FROM contacts WHERE workspace_id = $1 ORDER BY email`, eng.ID)
	if err != nil {
		return err
	}
	type ec struct {
		id      uuid.UUID
		created time.Time
	}
	var cs []ec
	for rows.Next() {
		var c ec
		if err := rows.Scan(&c.id, &c.created); err != nil {
			rows.Close()
			return err
		}
		cs = append(cs, c)
	}
	rows.Close()
	var out [][]any
	for i, c := range cs {
		if i < 3240 {
			out = append(out, []any{c.id, eng.ID, "import", imp.String(), at.Add(4 * time.Minute), false, s.Now})
		} else {
			out = append(out, []any{c.id, eng.ID, "carried", "form:blog", c.created, false, s.Now})
		}
	}
	if err := sgCopy(ctx, s.Tx, "contact_growth", []string{"contact_id", "workspace_id", "source_kind", "source_id", "joined_at", "captured", "updated_at"}, out); err != nil {
		return err
	}
	_, err = s.Tx.Exec(ctx, `INSERT INTO growth_state (workspace_id, capture_started_at, attributed_through, updated_at) VALUES ($1, $2, $2, $2)`, eng.ID, s.Now)
	return err
}
