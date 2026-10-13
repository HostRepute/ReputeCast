package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Field targets a source field can map to.
const (
	TargetEmail    = "email"
	TargetSkip     = "skip"
	TargetEvidence = "consent_evidence"
	TargetStatus   = "status"
	TargetTags     = "tags"
	TargetName     = "name"
)

// builtinTargets are contact columns, in the order the select lists them.
var builtinTargets = []struct{ key, label, typ string }{
	{TargetEmail, "email · required", "email"},
	{"first_name", "first_name", "text"},
	{"last_name", "last_name", "text"},
	{TargetName, "name (first and last)", "text"},
	{"company", "company", "text"},
	{"country", "country", "text"},
	{"timezone", "timezone", "timezone"},
	{"language", "language", "text"},
	{TargetTags, "tags", "tags"},
	{TargetEvidence, "Consent evidence (date, IP)", "consent"},
	{TargetStatus, "Consent status", "status"},
}

// ImportFieldChoice is one source field and where it goes.
type ImportFieldChoice struct {
	Source string `json:"source"`
	// Target: a built-in field, custom:<key>, new:<key> or skip.
	Target string `json:"target" maxLength:"80"`
	// Type of a new field: text, number, date, boolean.
	Type string `json:"type,omitempty" enum:"text,number,date,boolean,"`
}

// FieldMap is imports.field_map.
type FieldMap struct {
	Fields []ImportFieldChoice `json:"fields"`
}

// ImportFieldOption is a choice in a mapping select.
type ImportFieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ImportFieldRow is one row of the mapping table.
type ImportFieldRow struct {
	Source  string              `json:"source"`
	Label   string              `json:"label"`
	Samples []string            `json:"samples"`
	Target  string              `json:"target"`
	Type    string              `json:"type" doc:"What ReputeCast stores: email, text, number, date, timezone, consent, tags, or – when skipped"`
	Status  string              `json:"status" enum:"mapped,new,skipped"`
	Options []ImportFieldOption `json:"options"`
}

// ImportFieldMapping is the "Map fields" step.
type ImportFieldMapping struct {
	Fields  []ImportFieldRow `json:"fields"`
	Mapped  int              `json:"mapped"`
	New     int              `json:"new"`
	Skipped int              `json:"skipped"`
	// Ready is false while rows are still being read from the source.
	Ready bool `json:"ready"`
}

func parseFieldMap(raw json.RawMessage) FieldMap {
	var m FieldMap
	_ = json.Unmarshal(raw, &m)
	if m.Fields == nil {
		m.Fields = []ImportFieldChoice{}
	}
	return m
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func norm(s string) string { return nonAlnum.ReplaceAllString(strings.ToLower(s), "") }

// snakeKey turns "TRIAL_END" or "Trial end" into "trial_end".
func snakeKey(s string) string {
	var sb strings.Builder
	prevUnderscore := true
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) && r < unicode.MaxASCII || unicode.IsDigit(r) {
			sb.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if !prevUnderscore {
			sb.WriteByte('_')
			prevUnderscore = true
		}
	}
	k := strings.Trim(sb.String(), "_")
	if k == "" || !unicode.IsLetter(rune(k[0])) {
		k = "f_" + k
	}
	if len(k) > 63 {
		k = k[:63]
	}
	return k
}

var aliases = map[string]string{
	"email": TargetEmail, "emailaddress": TargetEmail, "mail": TargetEmail,
	"fname": "first_name", "firstname": "first_name", "givenname": "first_name", "vorname": "first_name", "prenom": "first_name",
	"lname": "last_name", "lastname": "last_name", "surname": "last_name", "familyname": "last_name", "nachname": "last_name", "nom": "last_name",
	"name": TargetName, "fullname": TargetName,
	"company": "company", "companyname": "company", "organization": "company", "organisation": "company", "org": "company",
	"country": "country", "countrycode": "country",
	"timezone": "timezone", "tz": "timezone", "language": "language", "lang": "language", "locale": "language",
	"tags": TargetTags, "tag": TargetTags, "labels": TargetTags,
	"optin": TargetEvidence, "status": TargetStatus, "consent": TargetStatus, "subscriptionstatus": TargetStatus,
}

// guessTarget picks where a source field most likely goes.
func guessTarget(f SourceField, custom map[string]store.CustomField) ImportFieldChoice {
	c := ImportFieldChoice{Source: f.Key}
	n := norm(f.Key)
	n = strings.TrimPrefix(strings.TrimPrefix(n, "attribs"), "data")
	switch f.Type {
	case "email":
		c.Target = TargetEmail
		return c
	case "consent":
		c.Target = TargetEvidence
		return c
	case "tags":
		c.Target = TargetTags
		return c
	case "status":
		c.Target = TargetStatus
		return c
	}
	if t, ok := aliases[n]; ok {
		c.Target = t
		return c
	}
	for key := range custom {
		ck := norm(key)
		if ck == n || (len(n) >= 5 && n[:4] == ck[:min(4, len(ck))] && subsequence(n, ck)) {
			c.Target = "custom:" + key
			return c
		}
	}
	if f.Type == "address" || f.Type == "phone" {
		c.Target = TargetSkip
		return c
	}
	c.Target = "new:" + snakeKey(strings.TrimPrefix(strings.TrimPrefix(f.Key, "attribs."), "data."))
	return c
}

// subsequence reports whether a's letters appear in b in order (SIGNUP_SRC → signup_source).
func subsequence(a, b string) bool {
	j := 0
	for i := 0; i < len(b) && j < len(a); i++ {
		if a[j] == b[i] {
			j++
		}
	}
	return j == len(a)
}

func newFieldType(f SourceField, samples []string) string {
	switch f.Type {
	case "number":
		return "number"
	case "date", "birthday":
		return "date"
	case "boolean":
		return "boolean"
	}
	return guessType(samples)
}

// Fields returns the mapping step: each source field with sample values,
// its current or guessed target and the choices.
func (s *Service) Fields(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportFieldMapping, error) {
	return s.fieldMapping(ctx, ws, id, false)
}

// FieldGuesses is the mapping step with every target guessed again
// ("Reset to guesses"); nothing is saved until the mapping is.
func (s *Service) FieldGuesses(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportFieldMapping, error) {
	return s.fieldMapping(ctx, ws, id, true)
}

func (s *Service) fieldMapping(ctx context.Context, ws *store.Workspace, id uuid.UUID, guesses bool) (ImportFieldMapping, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportFieldMapping{}, err
	}
	out := ImportFieldMapping{Fields: []ImportFieldRow{}}
	if imp.StagedAt == nil || imp.Status == "staging" {
		return out, nil
	}
	out.Ready = true
	conn := parseConnection(imp.Connection)
	custom, err := s.customFields(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	samples, err := s.samples(ctx, imp.ID, conn.Fields)
	if err != nil {
		return out, err
	}
	saved := map[string]ImportFieldChoice{}
	if !guesses {
		for _, c := range parseFieldMap(imp.FieldMap).Fields {
			saved[c.Source] = c
		}
	}
	for _, f := range conn.Fields {
		choice, ok := saved[f.Key]
		if !ok {
			choice = guessTarget(f, custom)
		}
		row := ImportFieldRow{Source: f.Key, Label: f.Label, Samples: samples[f.Key], Target: choice.Target, Options: targetOptions(f, custom)}
		if row.Samples == nil {
			row.Samples = []string{}
		}
		switch {
		case choice.Target == TargetSkip:
			row.Status, row.Type = "skipped", "–"
			out.Skipped++
		case strings.HasPrefix(choice.Target, "new:"):
			row.Status = "new"
			row.Type = choice.Type
			if row.Type == "" {
				row.Type = newFieldType(f, row.Samples)
			}
			out.New++
		default:
			row.Status = "mapped"
			row.Type = targetType(choice.Target, custom)
			out.Mapped++
		}
		out.Fields = append(out.Fields, row)
	}
	return out, nil
}

func targetType(target string, custom map[string]store.CustomField) string {
	if key, ok := strings.CutPrefix(target, "custom:"); ok {
		if f, ok := custom[key]; ok {
			if f.Type == "select" {
				return "text"
			}
			return f.Type
		}
	}
	for _, b := range builtinTargets {
		if b.key == target {
			return b.typ
		}
	}
	return "text"
}

// targetOptions are the choices for one source field.
func targetOptions(f SourceField, custom map[string]store.CustomField) []ImportFieldOption {
	if f.Type == "email" {
		return []ImportFieldOption{{TargetEmail, "email · required"}}
	}
	out := []ImportFieldOption{}
	for _, b := range builtinTargets[1:] {
		out = append(out, ImportFieldOption{b.key, b.label})
	}
	for _, c := range sortedCustom(custom) {
		out = append(out, ImportFieldOption{"custom:" + c.Key, c.Key + " · custom, exists"})
	}
	key := snakeKey(strings.TrimPrefix(strings.TrimPrefix(f.Key, "attribs."), "data."))
	if _, exists := custom[key]; !exists {
		out = append(out, ImportFieldOption{"new:" + key, fmt.Sprintf("Create new field %q", key)})
	}
	return append(out, ImportFieldOption{TargetSkip, "Skip"})
}

func sortedCustom(m map[string]store.CustomField) []store.CustomField {
	out := make([]store.CustomField, 0, len(m))
	for _, f := range m {
		out = append(out, f)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && (out[j].Position < out[j-1].Position || (out[j].Position == out[j-1].Position && out[j].Key < out[j-1].Key)); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (s *Service) customFields(ctx context.Context, ws uuid.UUID) (map[string]store.CustomField, error) {
	rows, err := s.Q.ListCustomFields(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[string]store.CustomField, len(rows))
	for _, f := range rows {
		out[f.Key] = f
	}
	return out, nil
}

// samples reads a few values per field from the first staged rows.
func (s *Service) samples(ctx context.Context, id uuid.UUID, fields []SourceField) (map[string][]string, error) {
	rows, err := s.Q.ImportRowsAfter(ctx, store.ImportRowsAfterParams{ImportID: id, After: 0, Lim: 3})
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	add := func(key, v string, limit int) {
		v = strings.TrimSpace(v)
		if v == "" || len(out[key]) >= limit {
			return
		}
		for _, x := range out[key] {
			if x == v {
				return
			}
		}
		out[key] = append(out[key], v)
	}
	for _, r := range rows {
		var data map[string]string
		_ = json.Unmarshal(r.Data, &data)
		for _, f := range fields {
			switch f.Type {
			case "email":
				add(f.Key, firstNonEmpty(data[f.Key], r.Email), 2)
			case "tags":
				for _, t := range r.Tags {
					add(f.Key, t, 3)
				}
			case "consent":
				var e Engagement
				_ = json.Unmarshal(r.Engagement, &e)
				if len(out[f.Key]) == 0 && (data["OPTIN_TIME"] != "" || e.OptinAt != nil) {
					t := data["OPTIN_TIME"]
					if t == "" && e.OptinAt != nil {
						t = e.OptinAt.UTC().Format("2006-01-02 15:04")
					}
					ip := firstNonEmpty(data["OPTIN_IP"], e.OptinIP)
					out[f.Key] = []string{t}
					if ip != "" {
						out[f.Key] = append(out[f.Key], ip)
					}
				}
			default:
				add(f.Key, shorten(data[f.Key]), 3)
			}
		}
	}
	return out, nil
}

// shorten keeps long values readable in the samples column ("500 Market St …").
func shorten(v string) string {
	v = strings.TrimSpace(v)
	if len(v) <= 16 {
		return v
	}
	cut := strings.LastIndex(v[:14], " ")
	if cut <= 0 {
		cut = 13
	}
	return strings.TrimRight(v[:cut], " ,") + " …"
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ImportSaveFieldsInput is the mapping chosen by the person.
type ImportSaveFieldsInput struct {
	Fields []ImportFieldChoice `json:"fields" maxItems:"500"`
}

// SaveFields stores the mapping and moves on to consent.
func (s *Service) SaveFields(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ImportSaveFieldsInput) (ImportFieldMapping, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportFieldMapping{}, err
	}
	if err := editable(imp); err != nil {
		return ImportFieldMapping{}, err
	}
	conn := parseConnection(imp.Connection)
	known := map[string]bool{}
	for _, f := range conn.Fields {
		known[f.Key] = true
	}
	custom, err := s.customFields(ctx, ws.ID)
	if err != nil {
		return ImportFieldMapping{}, err
	}
	emails := 0
	newKeys := map[string]bool{}
	for i, c := range in.Fields {
		loc := fmt.Sprintf("body.fields[%d]", i)
		if !known[c.Source] {
			return ImportFieldMapping{}, api.Invalid(loc+".source", fmt.Sprintf("%s isn't a field of this source.", c.Source))
		}
		switch {
		case c.Target == TargetEmail:
			emails++
		case strings.HasPrefix(c.Target, "custom:"):
			if _, ok := custom[strings.TrimPrefix(c.Target, "custom:")]; !ok {
				return ImportFieldMapping{}, api.Invalid(loc+".target", "That custom field doesn't exist any more.")
			}
		case strings.HasPrefix(c.Target, "new:"):
			key := strings.TrimPrefix(c.Target, "new:")
			if !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(key) {
				return ImportFieldMapping{}, api.Invalid(loc+".target", "New field keys use lowercase letters, digits and _.")
			}
			if _, ok := custom[key]; ok {
				in.Fields[i].Target = "custom:" + key
			}
			if newKeys[key] {
				return ImportFieldMapping{}, api.Invalid(loc+".target", fmt.Sprintf("Two fields would both create %q.", key))
			}
			newKeys[key] = true
			if c.Type == "" {
				in.Fields[i].Type = "text"
			}
		case c.Target == TargetSkip:
		default:
			ok := false
			for _, b := range builtinTargets {
				if b.key == c.Target {
					ok = true
				}
			}
			if !ok {
				return ImportFieldMapping{}, api.Invalid(loc+".target", "Pick a field from the list.")
			}
		}
	}
	if emails != 1 {
		return ImportFieldMapping{}, api.Invalid("body.fields", "Map exactly one field to email.")
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SetImportFieldMap(ctx, store.SetImportFieldMapParams{ID: imp.ID, FieldMap: mustJSON(FieldMap(in))}); err != nil {
			return err
		}
		if err := q.SetImportStep(ctx, store.SetImportStepParams{ID: imp.ID, Step: "consent"}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.map_fields", TargetType: "import", TargetID: imp.ID.String(),
			TargetName: SourceName(imp.Source), After: in})
	})
	if err != nil {
		return ImportFieldMapping{}, err
	}
	return s.Fields(ctx, ws, id)
}

// effectiveMap is the saved mapping, or the guesses when nothing was saved.
func (s *Service) effectiveMap(imp store.Import, custom map[string]store.CustomField) FieldMap {
	m := parseFieldMap(imp.FieldMap)
	if len(m.Fields) > 0 {
		return m
	}
	for _, f := range parseConnection(imp.Connection).Fields {
		c := guessTarget(f, custom)
		if strings.HasPrefix(c.Target, "new:") {
			c.Type = newFieldType(f, nil)
		}
		m.Fields = append(m.Fields, c)
	}
	return m
}
