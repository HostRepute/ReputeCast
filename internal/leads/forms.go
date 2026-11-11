package leads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/leads/meta"
	"github.com/hostrepute/reputecast/internal/store"
)

// What a question can be saved as (lead_forms.mapping values); custom
// fields are "field:<key>".
const (
	TargetEmail     = "email"
	TargetFirstName = "first_name"
	TargetLastName  = "last_name"
	TargetFullName  = "full_name"
	TargetCompany   = "company"
	TargetPhone     = "phone"
	fieldPrefix     = "field:"
)

var standardTargets = []string{TargetEmail, TargetFirstName, TargetLastName, TargetFullName, TargetCompany, TargetPhone}

// LeadAdsFormDetail is the mapping dialog: the form's questions and
// consent checkboxes as Meta returns them, how it's mapped, and what it can
// be mapped to.
type LeadAdsFormDetail struct {
	Form       LeadAdsForm       `json:"form"`
	Questions  []LeadAdsQuestion `json:"questions"`
	Checkboxes []LeadAdsCheckbox `json:"checkboxes"`
	Disclaimer string            `json:"disclaimer" doc:"The custom disclaimer's text"`
	// The form's mapping: lists, tags and consent (the questions carry
	// their targets).
	ListIDs     []string `json:"listIds"`
	ConsentKey  string   `json:"consentKey,omitempty"`
	ConsentText string   `json:"consentText,omitempty" doc:"Single opt-in: the checkbox's text the leads agreed to"`
	// Options: the workspace's lists and custom fields.
	Lists  []LeadAdsListOption  `json:"lists"`
	Fields []LeadAdsFieldOption `json:"fields"`
}

// LeadAdsQuestion is one of a form's questions and what it's saved as.
type LeadAdsQuestion struct {
	Key   string `json:"key" example:"full_name"`
	Label string `json:"label" example:"Full name"`
	Type  string `json:"type" example:"FULL_NAME" doc:"Meta's question type"`
	// Target is the mapping ("" when not kept); Suggested what ReputeCast
	// proposes for a form not mapped yet.
	Target    string `json:"target"`
	Suggested string `json:"suggested"`
}

// LeadAdsCheckbox is one of the custom disclaimer's checkboxes.
type LeadAdsCheckbox struct {
	Key      string `json:"key"`
	Text     string `json:"text"`
	Required bool   `json:"required"`
}

// LeadAdsListOption is a list leads can join.
type LeadAdsListOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// DoubleOptIn is the list's default for new signups; a lead form's
	// consent mode decides for its leads.
	DoubleOptIn bool `json:"doubleOptIn"`
}

// LeadAdsFieldOption is a custom field a question can be saved as.
type LeadAdsFieldOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

// LeadAdsMappingInput maps a form.
type LeadAdsMappingInput struct {
	// Mapping is question key → email, first_name, last_name, full_name,
	// company, phone or field:<custom field key>; questions left out aren't
	// kept.
	Mapping     map[string]string `json:"mapping" maxProperties:"60"`
	ListIDs     []string          `json:"listIds" minItems:"1" maxItems:"20"`
	Tags        []string          `json:"tags,omitempty" maxItems:"10"`
	ConsentMode string            `json:"consentMode" enum:"double,single"`
	// ConsentKey is the checkbox that is the consent, for single opt-in.
	ConsentKey string `json:"consentKey,omitempty" maxLength:"200"`
}

// form loads one of the workspace's forms (404 otherwise).
func (s *Service) form(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.LeadFormByIDRow, error) {
	f, err := s.Q.LeadFormByID(ctx, store.LeadFormByIDParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return f, api.Err(http.StatusNotFound, api.CodeNotFound, mFormNotFound.Text(i18n.LangFrom(ctx), nil))
	}
	return f, err
}

// liveForm is a form of a connection that isn't disconnected.
func (s *Service) liveForm(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.LeadFormByIDRow, store.LeadConnection, error) {
	f, err := s.form(ctx, ws, id)
	if err != nil {
		return f, store.LeadConnection{}, err
	}
	c, err := s.live(ctx, s.Q, ws, f.ConnectionID)
	return f, c, err
}

// suggest proposes a target for a question by Meta's type.
func suggest(q meta.Question) string {
	switch strings.ToUpper(q.Type) {
	case "EMAIL", "WORK_EMAIL":
		return TargetEmail
	case "FIRST_NAME":
		return TargetFirstName
	case "LAST_NAME":
		return TargetLastName
	case "FULL_NAME":
		return TargetFullName
	case "COMPANY_NAME":
		return TargetCompany
	case "PHONE", "WORK_PHONE_NUMBER":
		return TargetPhone
	}
	return ""
}

// FormDetail is the mapping dialog. The form's questions and checkboxes
// are read from Meta when it answers, so they are current.
func (s *Service) FormDetail(ctx context.Context, ws *store.Workspace, id uuid.UUID) (LeadAdsFormDetail, error) {
	f, err := s.form(ctx, ws, id)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	if c, err := s.Q.LeadConnectionAny(ctx, f.ConnectionID); err == nil && c.Status != StatusDisconnected && f.Chosen {
		if cl, err := s.client(c); err == nil {
			if pg, err := s.pageOf(f.PageID, f.MetaPageID, f.PageTokenSealed); err == nil {
				if mf, err := cl.Form(ctx, pg, f.FormID); err == nil {
					if up, err := s.Q.LeadFormUpsert(ctx, store.LeadFormUpsertParams{ID: newID(), WorkspaceID: ws.ID, PageID: f.PageID, FormID: f.FormID,
						Name: clip(mf.Name, 200), MetaStatus: mf.Status, Questions: mustJSON(nonNil(mf.Questions)), Checkboxes: mustJSON(nonNil(mf.Checkboxes)),
						Disclaimer: clip(mf.Disclaimer, 4000), At: s.now()}); err == nil {
						f.Name, f.MetaStatus, f.Questions, f.Checkboxes, f.Disclaimer = up.Name, up.MetaStatus, up.Questions, up.Checkboxes, up.Disclaimer
					}
				}
			}
		}
	}
	return s.detail(ctx, ws, f)
}

func (s *Service) detail(ctx context.Context, ws *store.Workspace, f store.LeadFormByIDRow) (LeadAdsFormDetail, error) {
	lists, err := s.Q.LeadListsOfWorkspace(ctx, ws.ID)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	names := map[uuid.UUID]string{}
	out := LeadAdsFormDetail{Questions: []LeadAdsQuestion{}, Checkboxes: []LeadAdsCheckbox{}, Disclaimer: f.Disclaimer, ListIDs: []string{},
		Lists: make([]LeadAdsListOption, 0, len(lists)), Fields: []LeadAdsFieldOption{}}
	for _, l := range lists {
		names[l.ID] = l.Name
		out.Lists = append(out.Lists, LeadAdsListOption{ID: l.ID.String(), Name: l.Name, DoubleOptIn: l.DoubleOptIn})
	}
	leads30d := int64(0)
	if n, err := s.Q.LeadLeads30dOfForm(ctx, store.LeadLeads30dOfFormParams{FormID: f.ID, Since: s.now().AddDate(0, 0, -30)}); err == nil {
		leads30d = n
	}
	out.Form = formView(formByIDRow(f), f.PageName, leads30d, names)
	fields, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	for _, cf := range fields {
		out.Fields = append(out.Fields, LeadAdsFieldOption{Key: cf.Key, Label: cf.Label, Type: cf.Type})
	}
	var qs []meta.Question
	_ = json.Unmarshal(f.Questions, &qs)
	var mapping map[string]string
	_ = json.Unmarshal(f.Mapping, &mapping)
	for _, q := range qs {
		out.Questions = append(out.Questions, LeadAdsQuestion{Key: q.Key, Label: q.Label, Type: q.Type, Target: mapping[q.Key], Suggested: suggest(q)})
	}
	var cbs []meta.Checkbox
	_ = json.Unmarshal(f.Checkboxes, &cbs)
	for _, cb := range cbs {
		out.Checkboxes = append(out.Checkboxes, LeadAdsCheckbox{Key: cb.Key, Text: cb.Text, Required: cb.Required})
	}
	for _, id := range f.ListIds {
		out.ListIDs = append(out.ListIDs, id.String())
	}
	if f.ConsentMode == "single" {
		out.ConsentKey, out.ConsentText = f.ConsentKey, f.ConsentText
	}
	return out, nil
}

// SaveMapping maps a form: each question to a field, its lists and tags,
// and its consent. Single opt-in needs one of the form's consent
// checkboxes: its text, as Meta returns it, becomes the consent text and
// its hash the version. A form mapped for the first time starts
// collecting.
func (s *Service) SaveMapping(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in LeadAdsMappingInput) (LeadAdsFormDetail, error) {
	l := i18n.LangFrom(ctx)
	f, _, err := s.liveForm(ctx, ws, id)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	var qs []meta.Question
	_ = json.Unmarshal(f.Questions, &qs)
	keys := map[string]bool{}
	for _, q := range qs {
		keys[q.Key] = true
	}
	fields, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	fieldKeys := map[string]bool{}
	for _, cf := range fields {
		fieldKeys[cf.Key] = true
	}
	mapping := map[string]string{}
	used := map[string]string{}
	email := false
	for _, k := range sortedKeys(in.Mapping) {
		target := strings.TrimSpace(in.Mapping[k])
		if target == "" {
			continue
		}
		if !keys[k] {
			return LeadAdsFormDetail{}, api.Invalid("body.mapping."+k, mMappingQuestion.Text(l, i18n.Args{"key": k}))
		}
		switch {
		case slices.Contains(standardTargets, target):
		case strings.HasPrefix(target, fieldPrefix) && fieldKeys[strings.TrimPrefix(target, fieldPrefix)]:
		default:
			return LeadAdsFormDetail{}, api.Invalid("body.mapping."+k, mMappingTarget.Text(l, nil))
		}
		if prev, ok := used[target]; ok {
			return LeadAdsFormDetail{}, api.Invalid("body.mapping."+k, mMappingTwice.Text(l, i18n.Args{"first": prev, "second": k}))
		}
		used[target] = k
		if target == TargetEmail {
			email = true
		}
		mapping[k] = target
	}
	if !email {
		return LeadAdsFormDetail{}, api.Invalid("body.mapping", mMappingEmail.Text(l, nil))
	}
	if _, ok := used[TargetFullName]; ok {
		_, first := used[TargetFirstName]
		_, last := used[TargetLastName]
		if first || last {
			return LeadAdsFormDetail{}, api.Invalid("body.mapping", mMappingNames.Text(l, nil))
		}
	}
	lists, err := s.Q.LeadListsOfWorkspace(ctx, ws.ID)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	listNames := map[uuid.UUID]string{}
	for _, li := range lists {
		listNames[li.ID] = li.Name
	}
	listIDs := []uuid.UUID{}
	seenList := map[uuid.UUID]bool{}
	for _, raw := range in.ListIDs {
		lid, err := uuid.Parse(raw)
		if err != nil || listNames[lid] == "" {
			return LeadAdsFormDetail{}, api.Invalid("body.listIds", mMappingList.Text(l, nil))
		}
		if p != nil && !p.Scope.AllowsList(lid) {
			return LeadAdsFormDetail{}, api.Invalid("body.listIds", mMappingListScope.Text(l, i18n.Args{"list": listNames[lid]}))
		}
		if !seenList[lid] {
			seenList[lid] = true
			listIDs = append(listIDs, lid)
		}
	}
	tags := cleanTags(in.Tags)
	mode := in.ConsentMode
	consentKey, consentText := "", ""
	if mode == "single" {
		var cbs []meta.Checkbox
		_ = json.Unmarshal(f.Checkboxes, &cbs)
		if len(cbs) == 0 {
			return LeadAdsFormDetail{}, api.Invalid("body.consentMode", mConsentNoCheckbox.Text(l, nil))
		}
		key := in.ConsentKey
		if key == "" && len(cbs) == 1 {
			key = cbs[0].Key
		}
		for _, cb := range cbs {
			if cb.Key == key {
				consentKey, consentText = cb.Key, cb.Text
			}
		}
		if consentKey == "" || strings.TrimSpace(consentText) == "" {
			return LeadAdsFormDetail{}, api.Invalid("body.consentKey", mConsentPick.Text(l, nil))
		}
	} else {
		consentText = f.Disclaimer
	}
	before := mappingSnapshot(f.MappedAt != nil, f.Mapping, f.ListIds, f.Tags, f.ConsentMode, f.ConsentKey, listNames)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		row, err := q.LeadFormSaveMapping(ctx, store.LeadFormSaveMappingParams{ID: f.ID, Mapping: mustJSON(mapping), ConsentMode: mode, ConsentKey: consentKey,
			ConsentText: clip(consentText, 4000), ConsentVersion: consentVersion(consentText), ListIds: listIDs, Tags: tags, At: s.now()})
		if err != nil {
			return err
		}
		after := mappingSnapshot(true, row.Mapping, row.ListIds, row.Tags, row.ConsentMode, row.ConsentKey, listNames)
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "lead_form.mapped", TargetType: "lead_form", TargetID: f.ID.String(),
			TargetName: f.Name, Before: before, After: after, Metadata: map[string]any{"page": f.PageName}})
	})
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	f, err = s.form(ctx, ws, id)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	return s.detail(ctx, ws, f)
}

// mappingSnapshot is a mapping as the audit log shows it.
func mappingSnapshot(mapped bool, raw json.RawMessage, lists []uuid.UUID, tags []string, mode, key string, names map[uuid.UUID]string) map[string]any {
	if !mapped {
		return map[string]any{"mapped": false}
	}
	var m map[string]string
	_ = json.Unmarshal(raw, &m)
	ln := []string{}
	for _, id := range lists {
		ln = append(ln, names[id])
	}
	out := map[string]any{"mapped": true, "questions": m, "lists": ln, "tags": nonNil(tags), "consent": mode}
	if key != "" {
		out["consentCheckbox"] = key
	}
	return out
}

// cleanTags trims, lower-cases and dedupes tags, as the profile does.
func cleanTags(in []string) []string {
	out := []string{}
	for _, t := range in {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || len(t) > 60 || slices.Contains(out, t) {
			continue
		}
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

// SetFormStatus pauses or resumes a form.
func (s *Service) SetFormStatus(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, status string) (LeadAdsFormDetail, error) {
	f, _, err := s.liveForm(ctx, ws, id)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	if status == "active" && f.MappedAt == nil {
		return LeadAdsFormDetail{}, api.Conflict(mFormNotMapped.Text(i18n.LangFrom(ctx), i18n.Args{"form": f.Name}))
	}
	if f.Status != status {
		action := "lead_form.paused"
		if status == "active" {
			action = "lead_form.resumed"
		}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := q.LeadFormSetStatus(ctx, store.LeadFormSetStatusParams{ID: f.ID, Status: status, At: s.now()}); err != nil {
				return err
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "lead_form", TargetID: f.ID.String(), TargetName: f.Name,
				Before: map[string]any{"status": f.Status}, After: map[string]any{"status": status}, Metadata: map[string]any{"page": f.PageName}})
		})
		if err != nil {
			return LeadAdsFormDetail{}, err
		}
	}
	f, err = s.form(ctx, ws, id)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	return s.detail(ctx, ws, f)
}

// StartImport queues Import past leads: the form's leads of the last 90
// days, each once.
func (s *Service) StartImport(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (LeadAdsFormDetail, error) {
	l := i18n.LangFrom(ctx)
	f, _, err := s.liveForm(ctx, ws, id)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	switch {
	case f.MappedAt == nil:
		return LeadAdsFormDetail{}, api.Conflict(mFormNotMapped.Text(l, i18n.Args{"form": f.Name}))
	case f.Status != "active":
		return LeadAdsFormDetail{}, api.Conflict(mImportPaused.Text(l, i18n.Args{"form": f.Name}))
	case f.ImportState == "running":
		return LeadAdsFormDetail{}, api.Conflict(mImportRunning.Text(l, i18n.Args{"form": f.Name}))
	case !f.Chosen:
		return LeadAdsFormDetail{}, api.Conflict(mPageNotChosen.Text(l, i18n.Args{"page": f.PageName}))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.LeadFormSetImport(ctx, store.LeadFormSetImportParams{ID: f.ID, ImportState: "running", ImportCount: 0, ImportedAt: f.ImportedAt}); err != nil {
			return err
		}
		if jc := s.jobs(); jc != nil {
			if _, err := jc.InsertTx(ctx, tx, ImportArgs{FormID: f.ID.String()}, nil); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "lead_form.imported", TargetType: "lead_form", TargetID: f.ID.String(),
			TargetName: f.Name, Metadata: map[string]any{"page": f.PageName, "days": int(ImportWindow.Hours() / 24)}})
	})
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	f, err = s.form(ctx, ws, id)
	if err != nil {
		return LeadAdsFormDetail{}, err
	}
	return s.detail(ctx, ws, f)
}
