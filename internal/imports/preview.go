package imports

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// ImportRiskyGroup is one group of risky addresses with the chosen action.
type ImportRiskyGroup struct {
	Key      string              `json:"key" enum:"role,disposable,typo,stale,trap"`
	Label    string              `json:"label"`
	Rows     int64               `json:"rows"`
	Why      string              `json:"why"`
	Examples string              `json:"examples"`
	Action   string              `json:"action"`
	Actions  []ImportFieldOption `json:"actions"`
}

// ImportDuplicateExample is one address that is already in ReputeCast.
type ImportDuplicateExample struct {
	Email        string `json:"email"`
	InReputeCast string `json:"inReputeCast"`
	// Suppressed rows read as a problem in the "In ReputeCast" column.
	Suppressed bool   `json:"suppressed"`
	InSource   string `json:"inSource"`
	Result     string `json:"result"`
	ResultTone string `json:"resultTone" enum:"ok,bad"`
}

// ImportInvalidGroup is a kind of invalid address and how many rows have it.
type ImportInvalidGroup struct {
	Problem string `json:"problem"`
	Example string `json:"example,omitempty"`
	Rows    int64  `json:"rows"`
}

// ImportPreview is the dry run: what would happen to every row.
type ImportPreview struct {
	Rows        int64 `json:"rows"`
	WillImport  int64 `json:"willImport" doc:"New contacts, subscribed"`
	Merged      int64 `json:"merged" doc:"Already in ReputeCast"`
	Suppressed  int64 `json:"suppressed" doc:"Unsubscribed, bounced or complained in the source"`
	Invalid     int64 `json:"invalid"`
	NotImported int64 `json:"notImported" doc:"Statuses mapped to Skip"`
	Risky       int64 `json:"risky"`
	// ImportCount and SuppressCount label the import button.
	ImportCount   int64                    `json:"importCount"`
	SuppressCount int64                    `json:"suppressCount"`
	RiskyGroups   []ImportRiskyGroup       `json:"riskyGroups"`
	Duplicates    []ImportDuplicateExample `json:"duplicates"`
	InvalidGroups []ImportInvalidGroup     `json:"invalidGroups"`
	OnConflict    string                   `json:"onConflict" enum:"fill,overwrite,keep"`
	ComputedAt    time.Time                `json:"computedAt"`
}

var riskyMeta = []struct {
	key, label, why string
	actions         []ImportFieldOption
}{
	{classRole, "Role addresses", "info@, sales@, support@ are shared inboxes; high complaint rates", []ImportFieldOption{
		{"exclude_broadcasts", "Import, excluded from Broadcasts automatically"}, {"skip", "Skip"}, {"import", "Import normally"}}},
	{classDisposable, "Disposable domains", "temporary inboxes that expire and bounce", []ImportFieldOption{{"skip", "Skip"}, {"import", "Import normally"}}},
	{classTypo, "Likely typos", "domain within one character of a major provider", []ImportFieldOption{
		{"fix", "Fix the domain and import, flagged for review"}, {"skip", "Skip"}, {"import", "Import as is"}}},
	{classStale, "Never engaged, 2+ years old", "", []ImportFieldOption{
		{"pending", "Import as pending, send re-permission"}, {"skip", "Skip"}, {"import", "Import normally"}}},
	{classTrap, "Known spam-trap patterns", "domains and local parts HostRepute associates with trap networks", []ImportFieldOption{
		{"skip", "Skip (recommended)"}, {"import", "Import normally"}}},
}

// mergeChange describes what merging a row into an existing contact does.
type mergeChange struct {
	kind    string // fills, tags, tier, same
	fills   []string
	addTags []string
}

func analyzeMerge(r mappedRow, c *existingContact, fm FieldMap, opts ImportOptions) mergeChange {
	var ch mergeChange
	for _, f := range fm.Fields {
		v := strings.TrimSpace(r.Data[f.Source])
		if v == "" {
			continue
		}
		label, cur := "", ""
		switch {
		case f.Target == "first_name":
			label, cur = "first name", c.FirstName
		case f.Target == "last_name":
			label, cur = "last name", c.LastName
		case f.Target == "company":
			label, cur = "company", c.Company
		case f.Target == "country":
			label, cur = "country", c.Country
		case f.Target == "timezone":
			label, cur = "timezone", c.Timezone
		case f.Target == "language":
			label, cur = "language", c.Language
		case strings.HasPrefix(f.Target, "custom:"):
			// Fields this import creates (new:) are empty on every existing
			// contact; filling them is expected and not worth calling out.
			label = f.Target[strings.Index(f.Target, ":")+1:]
			cur = strings.TrimSpace(fmt.Sprint(valueOr(c.Attributes[label], "")))
		default:
			continue
		}
		switch opts.OnConflict {
		case "overwrite":
			if cur != v {
				ch.fills = append(ch.fills, label)
			}
		case "keep":
		default:
			if cur == "" {
				ch.fills = append(ch.fills, label)
			}
		}
	}
	if opts.ImportTags {
		have := map[string]bool{}
		for _, t := range c.Tags {
			have[strings.ToLower(t)] = true
		}
		for _, t := range r.Tags {
			if !have[strings.ToLower(t)] {
				ch.addTags = append(ch.addTags, t)
			}
		}
	}
	switch {
	case len(ch.fills) > 0:
		ch.kind = "fills"
	case len(ch.addTags) > 0:
		ch.kind = "tags"
	case opts.ImportEngagement && (r.Eng.LastOpen != nil || r.Eng.Changed != nil):
		ch.kind = "tier"
	default:
		ch.kind = "same"
	}
	return ch
}

func valueOr(v any, def any) any {
	if v == nil {
		return def
	}
	return v
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (ch mergeChange) result() string {
	switch ch.kind {
	case "fills":
		return "Merge · fills " + joinAnd(ch.fills)
	case "tags":
		n := len(ch.addTags)
		return fmt.Sprintf("Merge · adds %d %s, keeps consent", n, plural(n, "tag", "tags"))
	case "tier":
		return "Merge · tier unchanged"
	}
	return "Merge · nothing to change"
}

func listSummary(lists []string) string {
	switch {
	case len(lists) == 0:
		return "no lists"
	case len(lists) <= 2:
		return strings.Join(lists, ", ")
	}
	return fmt.Sprintf("%d lists", len(lists))
}

func tierName(t string) string {
	switch t {
	case "active":
		return "Active"
	case "cooling":
		return "Cooling"
	case "at_risk":
		return "At risk"
	case "sunset":
		return "Sunset"
	}
	return t
}

func reasonWord(r string) string {
	switch r {
	case "unsubscribe":
		return "unsubscribed"
	case "hard_bounce":
		return "hard bounced"
	case "complaint":
		return "complained"
	}
	return "suppressed"
}

func sourceShort(src string) string {
	l := segments.SourceLabel(src)
	return strings.TrimSuffix(l, " import")
}

// duplicateExample explains one duplicate in words.
func (s *Service) duplicateExample(r mappedRow, v verdict, ch mergeChange, fm FieldMap, conn storedConnection, loc *time.Location) ImportDuplicateExample {
	day := func(t time.Time) string { return t.In(loc).Format("2 Jan") }
	if v.Supp != nil {
		return ImportDuplicateExample{Email: emailaddr.Mask(r.Email), Suppressed: true,
			InReputeCast: fmt.Sprintf("On suppression list · %s %s", reasonWord(v.Supp.Reason), day(v.Supp.At)),
			InSource:     r.Status, Result: "Stays suppressed", ResultTone: "bad"}
	}
	c := v.Contact
	ex := ImportDuplicateExample{Email: r.Email, Result: ch.result(), ResultTone: "ok"}
	parts := []string{listSummary(c.Lists)}
	srcParts := []string{}
	switch ch.kind {
	case "fills":
		parts = append(parts, ch.fills[0]+" empty")
		srcParts = append(srcParts, fmt.Sprintf("%s %q", ch.fills[0], fieldValue(r, fm, ch.fills[0])))
	case "tags":
		if plan := fmt.Sprint(valueOr(c.Attributes["plan"], "")); plan != "" {
			parts = append(parts, "plan "+plan)
		}
		parts = append(parts, fmt.Sprintf("joined %s via %s", day(c.CreatedAt), sourceShort(c.Source)))
		srcParts = append(srcParts, listName(conn, r.List))
		if plan := fieldValue(r, fm, "plan"); plan != "" {
			srcParts = append(srcParts, "plan "+plan)
		}
		if len(r.Tags) > 0 {
			srcParts = append(srcParts, "tags "+strings.Join(r.Tags, ", "))
		}
	case "tier":
		parts = append(parts, tierName(c.Tier)+" tier")
		srcParts = append(srcParts, r.Status)
		if r.Eng.LastOpen != nil {
			srcParts = append(srcParts, "last open "+strconv.Itoa(r.Eng.LastOpen.Year()))
		} else if r.Eng.Changed != nil {
			srcParts = append(srcParts, "last open "+strconv.Itoa(r.Eng.Changed.Year()))
		}
	default:
		srcParts = append(srcParts, r.Status)
	}
	ex.InReputeCast = strings.Join(parts, " · ")
	ex.InSource = strings.Join(srcParts, " · ")
	return ex
}

// fieldValue is the source value mapped to a target key (company, plan…).
func fieldValue(r mappedRow, fm FieldMap, key string) string {
	for _, f := range fm.Fields {
		t := f.Target
		if i := strings.Index(t, ":"); i >= 0 {
			t = t[i+1:]
		}
		if t == key || (key == "first name" && t == "first_name") || (key == "last name" && t == "last_name") {
			return strings.TrimSpace(r.Data[f.Source])
		}
	}
	return ""
}

func listName(conn storedConnection, id string) string {
	for _, l := range conn.Lists {
		if l.ID == id {
			return l.Name
		}
	}
	return id
}

// Preview runs the dry run over every staged row. Nothing is written
// except the stored preview.
func (s *Service) Preview(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportPreview, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportPreview{}, err
	}
	if imp.StagedAt == nil {
		return ImportPreview{}, api.Conflict("The rows aren't read from the source yet. Wait for the field mapping to load, then try again.")
	}
	custom, err := s.customFields(ctx, ws.ID)
	if err != nil {
		return ImportPreview{}, err
	}
	fm := s.effectiveMap(imp, custom)
	opts := parseOptions(imp.Options)
	cm := parseConsentMap(imp.ConsentMap, opts)
	conn := parseConnection(imp.Connection)
	loc := location(ws)
	now := s.now()
	p := ImportPreview{RiskyGroups: []ImportRiskyGroup{}, Duplicates: []ImportDuplicateExample{}, InvalidGroups: []ImportInvalidGroup{}, OnConflict: opts.OnConflict, ComputedAt: now}
	riskyCount := map[string]int64{}
	riskyExamples := map[string][]string{}
	var staleExample string
	invalidCount := map[string]int64{}
	invalidExample := map[string]string{}
	seenKinds := map[string]bool{}
	var after int32
	for {
		b, err := s.loadBatch(ctx, ws.ID, imp, fm, after, 5000)
		if err != nil {
			return p, err
		}
		if len(b.rows) == 0 {
			break
		}
		for _, r := range b.rows {
			after = r.RowNo
			p.Rows++
			v := classify(r, b, cm, now)
			switch v.Class {
			case classImport:
				p.WillImport++
			case classSuppress:
				p.Suppressed++
			case classSkipStatus:
				p.NotImported++
			case classInvalid:
				p.Invalid++
				invalidCount[v.Problem]++
				if _, ok := invalidExample[v.Problem]; !ok && v.Problem != ProblemNoMX {
					invalidExample[v.Problem] = strings.TrimSpace(r.Raw)
				}
			case classDuplicate, classAlreadySuppressed:
				p.Merged++
				if len(p.Duplicates) < 4 {
					kind := "suppressed"
					var ch mergeChange
					if v.Class == classDuplicate {
						ch = analyzeMerge(r, v.Contact, fm, opts)
						kind = ch.kind
					}
					if !seenKinds[kind] {
						seenKinds[kind] = true
						p.Duplicates = append(p.Duplicates, s.duplicateExample(r, v, ch, fm, conn, loc))
					}
				}
			default:
				p.Risky++
				riskyCount[v.Class]++
				ex := riskyExamples[v.Class]
				switch v.Class {
				case classRole:
					if len(ex) == 0 {
						ex = append(ex, r.Email)
					} else if len(ex) == 1 {
						ex = append(ex, emailaddr.Local(r.Email)+"@…")
					}
				case classDisposable, classTypo:
					d := "…@" + emailaddr.Domain(r.Email)
					limit := 2
					if v.Class == classTypo {
						limit = 3
					}
					if len(ex) < limit && !containsStr(ex, d) {
						ex = append(ex, d)
					}
				case classStale:
					if staleExample == "" {
						staleExample = fmt.Sprintf("member rating %d", r.Eng.Rating)
						if r.Eng.Changed != nil {
							staleExample += fmt.Sprintf(", last changed %d", r.Eng.Changed.Year())
						}
					}
				}
				riskyExamples[v.Class] = ex
			}
		}
	}
	for _, m := range riskyMeta {
		n := riskyCount[m.key]
		if n == 0 {
			continue
		}
		g := ImportRiskyGroup{Key: m.key, Label: m.label, Rows: n, Why: m.why, Actions: m.actions, Action: riskyAction(opts.Risky, m.key)}
		switch m.key {
		case classStale:
			g.Why = fmt.Sprintf("no open or click since %d in %s; most likely to be stale or recycled into spam traps", now.AddDate(-2, 0, 0).Year(), SourceName(imp.Source))
			g.Examples = staleExample
		case classTrap:
			g.Examples = fmt.Sprintf("redacted · %d rows", n)
		default:
			g.Examples = strings.Join(riskyExamples[m.key], " · ")
		}
		p.RiskyGroups = append(p.RiskyGroups, g)
	}
	for _, prob := range []string{emailaddr.ProblemNoDomain, emailaddr.ProblemDoubleDot, emailaddr.ProblemNotAnAddress, emailaddr.ProblemTooLong, ProblemNoMX} {
		if n := invalidCount[prob]; n > 0 {
			p.InvalidGroups = append(p.InvalidGroups, ImportInvalidGroup{Problem: prob, Example: invalidExample[prob], Rows: n})
		}
	}
	p.ImportCount = p.WillImport + p.Merged
	p.SuppressCount = p.Suppressed
	err = pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SetImportPreview(ctx, store.SetImportPreviewParams{ID: imp.ID, Preview: mustJSON(p)}); err != nil {
			return err
		}
		if imp.Step == "consent" || imp.Step == "fields" {
			return q.SetImportStep(ctx, store.SetImportStepParams{ID: imp.ID, Step: "preview"})
		}
		return nil
	})
	return p, err
}

func containsStr(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func location(ws *store.Workspace) *time.Location {
	if loc, err := time.LoadLocation(ws.Timezone); err == nil {
		return loc
	}
	return time.UTC
}

// InvalidCSV lists the invalid rows of the preview ("Download these rows").
func (s *Service) InvalidCSV(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]byte, string, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return nil, "", err
	}
	custom, err := s.customFields(ctx, ws.ID)
	if err != nil {
		return nil, "", err
	}
	fm := s.effectiveMap(imp, custom)
	cm := parseConsentMap(imp.ConsentMap, parseOptions(imp.Options))
	var sb strings.Builder
	sb.WriteString("row,value,problem\n")
	var after int32
	for {
		b, err := s.loadBatch(ctx, ws.ID, imp, fm, after, 5000)
		if err != nil {
			return nil, "", err
		}
		if len(b.rows) == 0 {
			break
		}
		for _, r := range b.rows {
			after = r.RowNo
			if v := classify(r, b, cm, s.now()); v.Class == classInvalid {
				fmt.Fprintf(&sb, "%d,%s,%s\n", r.RowNo, csvField(r.Raw), csvField(v.Problem))
			}
		}
	}
	return []byte(sb.String()), "invalid-rows.csv", nil
}

func csvField(s string) string {
	s = formulaSafe(s)
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// formulaSafe neutralizes a value a spreadsheet would run as a formula; a
// csv.Writer quotes the rest.
func formulaSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
