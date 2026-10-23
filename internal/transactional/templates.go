package transactional

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// KeepDraftRevisions is how many earlier draft revisions a template keeps.
const KeepDraftRevisions = 100

// TransactionalTemplateAccess says what the caller may do with a template.
type TransactionalTemplateAccess struct {
	CanEdit    bool `json:"canEdit"`
	CanPublish bool `json:"canPublish"`
	// CanSendTestToOthers is transactional:send; without it test emails go
	// to your own address only.
	CanSendTestToOthers bool `json:"canSendTestToOthers"`
	CanViewContent      bool `json:"canViewContent"`
	// EditReason and PublishReason explain a disabled control and name who
	// can help ("Publishing needs transactional:publish. Ask …").
	EditReason     string   `json:"editReason,omitempty"`
	PublishReason  string   `json:"publishReason,omitempty"`
	PublishHelpers []string `json:"publishHelpers"`
}

// TransactionalTemplate is a template with its draft, as the editor loads it.
type TransactionalTemplate struct {
	ID          string                    `json:"id"`
	Key         string                    `json:"key" doc:"The ID the API sends, fixed after the first publish"`
	KeyLocked   bool                      `json:"keyLocked"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Category    *TransactionalCategoryRef `json:"category,omitempty"`
	// Draft content.
	Subject     string                  `json:"subject"`
	Preheader   string                  `json:"preheader"`
	EditorMode  string                  `json:"editorMode" enum:"visual,mjml,html"`
	MJML        string                  `json:"mjml"`
	HTML        string                  `json:"html"`
	PlainText   string                  `json:"plainText"`
	Variables   []TransactionalVariable `json:"variables"`
	TestData    []TransactionalTestData `json:"testData"`
	Sender      *TransactionalSenderRef `json:"sender,omitempty"`
	ReplyTo     string                  `json:"replyTo"`
	TrackOpens  bool                    `json:"trackOpens"`
	TrackClicks bool                    `json:"trackClicks"`
	// DraftRevision is the autosave counter: send it back as baseRevision.
	DraftRevision int       `json:"draftRevision"`
	DraftSavedAt  time.Time `json:"draftSavedAt"`
	DraftSavedBy  string    `json:"draftSavedBy"`
	DraftHash     string    `json:"draftHash"`
	// DraftChanges is true when the draft differs from the live version
	// (always true before the first publish).
	DraftChanges  bool       `json:"draftChanges"`
	LiveVersion   *int       `json:"liveVersion,omitempty"`
	LiveSince     *time.Time `json:"liveSince,omitempty"`
	LiveBy        string     `json:"liveBy,omitempty"`
	VersionsCount int        `json:"versionsCount"`
	Archived      bool       `json:"archived"`
	ArchivedAt    *time.Time `json:"archivedAt,omitempty"`
	// Origin is where an imported template came from ({"source":"postmark"}).
	Origin    map[string]any              `json:"origin"`
	CreatedAt time.Time                   `json:"createdAt"`
	UpdatedAt time.Time                   `json:"updatedAt"`
	Access    TransactionalTemplateAccess `json:"access"`
	// Language is the language of the content returned (?language=) and
	// SourceLanguage the one the template is written in (internal/languages).
	Language       string `json:"language"`
	SourceLanguage string `json:"sourceLanguage"`
}

// draftHash fingerprints what a version would send: content, variables,
// sender and tracking (test data isn't sent, so it isn't part of it).
func draftHash(subject, preheader, mode, mjmlSrc, html, text string, vars []TransactionalVariable, sender *uuid.UUID, replyTo string, opens, clicks bool) string {
	h := sha256.New()
	for _, part := range []string{subject, preheader, mode, mjmlSrc, html, text, string(mustJSON(vars)), idString(sender), replyTo, fmt.Sprint(opens, clicks)} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

func (s *Service) access(ctx context.Context, ws *store.Workspace, p *authz.Principal) TransactionalTemplateAccess {
	a := TransactionalTemplateAccess{CanEdit: p.Can(authz.TransactionalEdit), CanPublish: p.Can(authz.TransactionalPublish),
		CanSendTestToOthers: p.Can(authz.TransactionalSend), CanViewContent: p.Can(authz.TransactionalContent), PublishHelpers: []string{}}
	if !a.CanEdit {
		edit := i18n.Args{"permission": string(authz.TransactionalEdit)}
		if helpers := s.helpers(ctx, ws.ID, authz.TransactionalEdit, 3); len(helpers) > 0 {
			edit["names"] = askNames(ctx, helpers)
			a.EditReason = say(ctx, "denied.editorAsk", edit)
		} else {
			a.EditReason = say(ctx, "denied.editor", edit)
		}
	}
	if !a.CanPublish {
		a.PublishHelpers = s.helpers(ctx, ws.ID, authz.TransactionalPublish, 3)
		publish := i18n.Args{"permission": string(authz.TransactionalPublish), "names": askNames(ctx, a.PublishHelpers)}
		switch {
		case a.CanEdit && len(a.PublishHelpers) > 0:
			a.PublishReason = say(ctx, "denied.publishSavedAsk", publish)
		case a.CanEdit:
			a.PublishReason = say(ctx, "denied.publishSaved", publish)
		default:
			a.PublishReason = deniedMessage(ctx, "publish", authz.TransactionalPublish, a.PublishHelpers)
		}
	}
	return a
}

// resolveTemplate finds a template by its UUID or its key.
func (s *Service) resolveTemplate(ctx context.Context, q *store.Queries, wsID uuid.UUID, ref string) (uuid.UUID, error) {
	if id, err := uuid.Parse(ref); err == nil {
		return id, nil
	}
	id, err := q.TxnTemplateIDByKey(ctx, store.TxnTemplateIDByKeyParams{WorkspaceID: wsID, Key: strings.ToLower(strings.TrimSpace(ref))})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, notFound(ctx, "template")
	}
	return id, err
}

// Template loads a template for the editor.
func (s *Service) Template(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (TransactionalTemplate, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalTemplate{}, err
	}
	r, err := s.Q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return TransactionalTemplate{}, mapTemplateErr(ctx, err)
	}
	t := TransactionalTemplate{
		ID: r.ID.String(), Key: r.Key, KeyLocked: r.FirstPublishedAt != nil, Name: r.Name, Description: r.Description,
		Category: categoryRef(r.CategoryID, r.CategoryName, r.CategoryKind),
		Subject:  r.Subject, Preheader: r.Preheader, EditorMode: r.EditorMode, MJML: r.Mjml, HTML: r.Html, PlainText: r.PlainText,
		Variables: parseVariables(r.Variables), TestData: parseTestData(r.TestData), ReplyTo: r.ReplyTo, TrackOpens: r.TrackOpens, TrackClicks: r.TrackClicks,
		DraftRevision: int(r.DraftRevision), DraftSavedAt: r.DraftSavedAt, DraftSavedBy: r.DraftSavedByName, DraftHash: r.DraftHash,
		DraftChanges: r.LiveHash == "" || r.LiveHash != r.DraftHash, LiveSince: r.LiveSince, LiveBy: r.LiveByName, VersionsCount: int(r.VersionsCount),
		Archived: r.ArchivedAt != nil, ArchivedAt: r.ArchivedAt, Origin: map[string]any{}, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		Access: s.access(ctx, ws, p),
	}
	_ = json.Unmarshal(r.Origin, &t.Origin)
	if t.Origin == nil {
		t.Origin = map[string]any{}
	}
	if r.LiveVersion != nil {
		v := int(*r.LiveVersion)
		t.LiveVersion = &v
	}
	if r.SenderID != nil {
		t.Sender = &TransactionalSenderRef{ID: r.SenderID.String(), Name: r.SenderName, Email: r.SenderEmail}
	}
	return t, nil
}

var keyRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// KeyFromName turns "Password reset" into "password-reset".
func KeyFromName(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	return clip(strings.Trim(b.String(), "-"), 64)
}

func validateKey(ctx context.Context, key string) error {
	if key == "" {
		return api.Invalid("body.key", say(ctx, "template.keyRequired", nil))
	}
	if len(key) > 64 || !keyRe.MatchString(key) {
		return api.Invalid("body.key", say(ctx, "template.keyFormat", nil))
	}
	if _, err := uuid.Parse(key); err == nil {
		return api.Invalid("body.key", say(ctx, "template.keyLooksLikeId", nil))
	}
	return nil
}

// TransactionalTemplateCreate starts a template: blank, from a starter, as
// a copy of another template, or with content (template import).
type TransactionalTemplateCreate struct {
	Name        string `json:"name" minLength:"1" maxLength:"120"`
	Key         string `json:"key,omitempty" maxLength:"64" doc:"Defaults to the name in lowercase with hyphens"`
	Description string `json:"description,omitempty" maxLength:"300"`
	CategoryID  string `json:"categoryId,omitempty" doc:"Defaults to the first Essential category"`
	Starter     string `json:"starter,omitempty" maxLength:"40" doc:"A starter's key (password-reset, magic-link…)"`
	// FromTemplate copies another template's draft.
	FromTemplate string `json:"fromTemplate,omitempty" maxLength:"64"`
	// Content of an imported template (the importer fills these).
	Subject    string                  `json:"subject,omitempty" maxLength:"998"`
	Preheader  string                  `json:"preheader,omitempty" maxLength:"500"`
	EditorMode string                  `json:"editorMode,omitempty" enum:"visual,mjml,html"`
	MJML       string                  `json:"mjml,omitempty" maxLength:"2000000"`
	HTML       string                  `json:"html,omitempty" maxLength:"2500000"`
	PlainText  string                  `json:"plainText,omitempty" maxLength:"500000"`
	Variables  []TransactionalVariable `json:"variables,omitempty" maxItems:"200"`
	TestData   []TransactionalTestData `json:"testData,omitempty" maxItems:"50"`
	Origin     map[string]any          `json:"origin,omitempty"`
}

// CreateTemplate creates a draft template.
func (s *Service) CreateTemplate(ctx context.Context, ws *store.Workspace, p *authz.Principal, in TransactionalTemplateCreate) (TransactionalTemplate, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return TransactionalTemplate{}, api.Invalid("body.name", say(ctx, "template.nameRequired", nil))
	}
	key := strings.ToLower(strings.TrimSpace(in.Key))
	if key == "" {
		key = KeyFromName(name)
	}
	if err := validateKey(ctx, key); err != nil {
		return TransactionalTemplate{}, err
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return TransactionalTemplate{}, err
	}
	c := content{Subject: in.Subject, Preheader: in.Preheader, EditorMode: in.EditorMode, MJML: in.MJML, HTML: in.HTML, Text: in.PlainText, Variables: in.Variables}
	testData := in.TestData
	sender, replyTo := st.DefaultSenderID, st.DefaultReplyTo
	opens, clicks := st.DefaultTrackOpens, st.DefaultTrackClicks
	description := strings.TrimSpace(in.Description)
	categoryKind := ""
	source := "create"
	switch {
	case in.Starter != "":
		sd, ok := starterByKey(in.Starter)
		if !ok {
			return TransactionalTemplate{}, api.Invalid("body.starter", say(ctx, "template.starterUnknown", i18n.Args{"starter": in.Starter}))
		}
		product := s.product(ctx, ws)
		src := sd.mjmlFor(product, s.footer(ctx, ws))
		c = content{Subject: fillProduct(sd.Subject, product), Preheader: fillProduct(sd.Preheader, product), EditorMode: "visual", MJML: src, Variables: sd.Variables}
		testData = sd.TestData
		if description == "" {
			description = sd.Description
		}
		categoryKind = sd.Kind
		source = "starter"
	case in.FromTemplate != "":
		from, err := s.resolveTemplate(ctx, s.Q, ws.ID, in.FromTemplate)
		if err != nil {
			return TransactionalTemplate{}, err
		}
		f, err := s.Q.TxnTemplate(ctx, store.TxnTemplateParams{WorkspaceID: ws.ID, ID: from})
		if err != nil {
			return TransactionalTemplate{}, mapTemplateErr(ctx, err)
		}
		c = content{Subject: f.Subject, Preheader: f.Preheader, EditorMode: f.EditorMode, MJML: f.Mjml, HTML: f.Html, Text: f.PlainText, Variables: parseVariables(f.Variables)}
		testData = parseTestData(f.TestData)
		sender, replyTo, opens, clicks = f.SenderID, f.ReplyTo, f.TrackOpens, f.TrackClicks
		if in.CategoryID == "" && f.CategoryID != nil {
			in.CategoryID = f.CategoryID.String()
		}
		source = "duplicate"
	}
	if c.EditorMode == "" {
		c.EditorMode = "visual"
		if strings.TrimSpace(c.MJML) == "" && strings.TrimSpace(c.HTML) != "" {
			c.EditorMode = "html"
		}
	}
	if c.Variables == nil {
		c.Variables = []TransactionalVariable{}
	}
	if testData == nil {
		testData = []TransactionalTestData{}
	}
	if err := validateVariables(ctx, c.Variables); err != nil {
		return TransactionalTemplate{}, err
	}
	if err := validateTestData(ctx, testData); err != nil {
		return TransactionalTemplate{}, err
	}
	if strings.TrimSpace(c.HTML) == "" && strings.TrimSpace(c.MJML) != "" {
		html, err := c.compiledHTML()
		if err != nil {
			return TransactionalTemplate{}, api.Invalid("body.mjml", problemText(lang(ctx), err))
		}
		c.HTML = html
	}
	// lang, dir and the layout tables' role (accessibility).
	c.HTML = accessibility.FinalizeStored(ctx, s.Q, ws.ID, accessibility.OwnerTransactional, uuid.Nil, 0, c.HTML)
	origin := in.Origin
	if origin == nil {
		origin = map[string]any{}
	}
	id := newID()
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if taken, err := q.TxnTemplateKeyTaken(ctx, store.TxnTemplateKeyTakenParams{WorkspaceID: ws.ID, Key: key, ID: id}); err != nil {
			return err
		} else if taken {
			return api.Invalid("body.key", say(ctx, "template.keyTaken", i18n.Args{"key": key}))
		}
		category, err := s.pickCategory(ctx, q, ws.ID, in.CategoryID, categoryKind)
		if err != nil {
			return err
		}
		hash := draftHash(c.Subject, c.Preheader, c.EditorMode, c.MJML, c.HTML, c.Text, c.Variables, sender, replyTo, opens, clicks)
		if err := q.TxnTemplateInsert(ctx, store.TxnTemplateInsertParams{
			ID: id, WorkspaceID: ws.ID, Key: key, Name: name, Description: description, CategoryID: category,
			Subject: c.Subject, Preheader: c.Preheader, EditorMode: c.EditorMode, Mjml: c.MJML, Html: c.HTML, PlainText: c.Text,
			Variables: mustJSON(c.Variables), TestData: mustJSON(testData), SenderID: sender, ReplyTo: replyTo, TrackOpens: opens, TrackClicks: clicks,
			DraftHash: hash, Now: now, CreatedBy: userRef(p), Origin: mustJSON(origin),
		}); err != nil {
			if isUnique(err) {
				return api.Invalid("body.key", say(ctx, "template.keyTaken", i18n.Args{"key": key}))
			}
			return err
		}
		if err := q.TxnTemplateInsertDraftRevision(ctx, store.TxnTemplateInsertDraftRevisionParams{TemplateID: id, Revision: 1, Subject: c.Subject, Preheader: c.Preheader,
			Mjml: c.MJML, Html: c.HTML, Variables: mustJSON(c.Variables), SavedBy: userRef(p), SavedAt: now, Label: "created", Source: source}); err != nil {
			return err
		}
		meta := map[string]any{"source": source}
		if in.Starter != "" {
			meta["starter"] = in.Starter
		}
		if len(origin) > 0 {
			meta["origin"] = origin
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.template_created", TargetType: "transactional_template", TargetID: id.String(), TargetName: key,
			After: map[string]any{"name": name, "key": key, "category_id": idString(category)}, Metadata: meta})
	})
	if err != nil {
		return TransactionalTemplate{}, err
	}
	return s.Template(ctx, ws, p, id.String())
}

// pickCategory returns the chosen category, or the first one of the wanted
// kind (Essential by default), creating "Account" when there is none.
func (s *Service) pickCategory(ctx context.Context, q *store.Queries, wsID uuid.UUID, ref, kind string) (*uuid.UUID, error) {
	if ref != "" {
		id, err := uuid.Parse(ref)
		if err != nil {
			return nil, api.Invalid("body.categoryId", say(ctx, "template.categoryPick", nil))
		}
		if _, err := q.TxnCategory(ctx, store.TxnCategoryParams{WorkspaceID: wsID, ID: id}); err != nil {
			return nil, api.Invalid("body.categoryId", say(ctx, "template.categoryGone", nil))
		}
		return &id, nil
	}
	if kind == "" {
		kind = KindEssential
	}
	cats, err := q.TxnCategories(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for _, c := range cats {
		if c.Kind == kind {
			id := c.ID
			return &id, nil
		}
	}
	id := newID()
	// The category is the workspace's from now on (members rename it, the
	// SMTP header and the API name it), so it's written once, in the
	// workspace's default language.
	l := workspaceLang(ctx, q, wsID)
	name, desc := texts.M("category.defaultAccount").Text(l, nil), texts.M("category.defaultAccountText").Text(l, nil)
	if kind == KindOptional {
		name, desc = texts.M("category.defaultUpdates").Text(l, nil), texts.M("category.defaultUpdatesText").Text(l, nil)
	}
	if err := q.TxnCategoryInsert(ctx, store.TxnCategoryInsertParams{ID: id, WorkspaceID: wsID, Name: name, Kind: kind, Description: desc, CreatedAt: s.now()}); err != nil {
		return nil, err
	}
	return &id, nil
}

// footer is the transactional footer's address line: the workspace's
// postal address from Settings › Senders.
func (s *Service) footer(_ context.Context, ws *store.Workspace) string {
	var sd struct {
		FooterAddress string `json:"footer_address"`
	}
	_ = json.Unmarshal(ws.SendingDefaults, &sd)
	if a := strings.TrimSpace(sd.FooterAddress); a != "" {
		return a
	}
	return templates.WorkspaceAddress(ws)
}

func validateTestData(ctx context.Context, sets []TransactionalTestData) error {
	seen := map[string]bool{}
	for i, t := range sets {
		name := strings.TrimSpace(t.Name)
		if name == "" {
			return api.Invalid(fmt.Sprintf("body.testData[%d].name", i), say(ctx, "template.testDataName", nil))
		}
		if seen[strings.ToLower(name)] {
			return api.Invalid(fmt.Sprintf("body.testData[%d].name", i), say(ctx, "template.testDataTwice", i18n.Args{"name": name}))
		}
		seen[strings.ToLower(name)] = true
	}
	return nil
}

// TransactionalTemplateUpdate changes what isn't versioned: name, ID
// (before the first publish), description and category.
type TransactionalTemplateUpdate struct {
	Name        *string `json:"name,omitempty" minLength:"1" maxLength:"120"`
	Key         *string `json:"key,omitempty" maxLength:"64"`
	Description *string `json:"description,omitempty" maxLength:"300"`
	CategoryID  *string `json:"categoryId,omitempty"`
}

// UpdateTemplate applies a TransactionalTemplateUpdate.
func (s *Service) UpdateTemplate(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in TransactionalTemplateUpdate) (TransactionalTemplate, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalTemplate{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.TxnTemplateForUpdate(ctx, store.TxnTemplateForUpdateParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapTemplateErr(ctx, err)
		}
		name, key, desc, category := cur.Name, cur.Key, cur.Description, cur.CategoryID
		before, after := map[string]any{}, map[string]any{}
		if in.Name != nil {
			n := strings.TrimSpace(*in.Name)
			if n == "" {
				return api.Invalid("body.name", say(ctx, "template.nameRequired", nil))
			}
			if n != name {
				before["name"], after["name"] = name, n
				name = n
			}
		}
		if in.Key != nil {
			k := strings.ToLower(strings.TrimSpace(*in.Key))
			if k != key {
				if cur.FirstPublishedAt != nil {
					return api.Err(http.StatusConflict, api.CodeConflict, say(ctx, "template.keyFixed", i18n.Args{"key": key}))
				}
				if err := validateKey(ctx, k); err != nil {
					return err
				}
				if taken, err := q.TxnTemplateKeyTaken(ctx, store.TxnTemplateKeyTakenParams{WorkspaceID: ws.ID, Key: k, ID: id}); err != nil {
					return err
				} else if taken {
					return api.Invalid("body.key", say(ctx, "template.keyTaken", i18n.Args{"key": k}))
				}
				before["key"], after["key"] = key, k
				key = k
			}
		}
		if in.Description != nil && strings.TrimSpace(*in.Description) != desc {
			before["description"], after["description"] = desc, strings.TrimSpace(*in.Description)
			desc = strings.TrimSpace(*in.Description)
		}
		if in.CategoryID != nil {
			next, err := s.pickCategory(ctx, q, ws.ID, *in.CategoryID, "")
			if err != nil {
				return err
			}
			if idString(next) != idString(category) {
				before["category_id"], after["category_id"] = idString(category), idString(next)
				category = next
			}
		}
		if len(after) == 0 {
			return nil
		}
		if err := q.TxnTemplateUpdateMeta(ctx, store.TxnTemplateUpdateMetaParams{WorkspaceID: ws.ID, ID: id, Key: key, Name: name, Description: desc, CategoryID: category, UpdatedAt: s.now()}); err != nil {
			if isUnique(err) {
				return api.Invalid("body.key", say(ctx, "template.keyTaken", i18n.Args{"key": key}))
			}
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.template_updated", TargetType: "transactional_template", TargetID: id.String(), TargetName: key,
			Before: before, After: after})
	})
	if err != nil {
		return TransactionalTemplate{}, err
	}
	return s.Template(ctx, ws, p, id.String())
}

// TransactionalDraftSave is one save of the editor (autosave). Omitted
// fields keep their value.
type TransactionalDraftSave struct {
	// BaseRevision is the draft revision the editor started from. Saving on
	// top of a teammate's newer save fails with 409 version_conflict
	// unless Force is set.
	BaseRevision int                      `json:"baseRevision" minimum:"1"`
	Subject      *string                  `json:"subject,omitempty" maxLength:"998"`
	Preheader    *string                  `json:"preheader,omitempty" maxLength:"500"`
	EditorMode   *string                  `json:"editorMode,omitempty" enum:"visual,mjml,html"`
	MJML         *string                  `json:"mjml,omitempty" maxLength:"2000000"`
	HTML         *string                  `json:"html,omitempty" maxLength:"2500000" doc:"The HTML the editor compiled; empty compiles the MJML on the server"`
	PlainText    *string                  `json:"plainText,omitempty" maxLength:"500000"`
	Variables    *[]TransactionalVariable `json:"variables,omitempty" maxItems:"200"`
	TestData     *[]TransactionalTestData `json:"testData,omitempty" maxItems:"50"`
	SenderID     *string                  `json:"senderId,omitempty" doc:"A Transactional sender, or empty for the default"`
	ReplyTo      *string                  `json:"replyTo,omitempty" maxLength:"254"`
	TrackOpens   *bool                    `json:"trackOpens,omitempty"`
	TrackClicks  *bool                    `json:"trackClicks,omitempty"`
	Label        string                   `json:"label,omitempty" maxLength:"200"`
	Source       string                   `json:"source,omitempty" enum:"autosave,manual,import,restore" default:"autosave"`
	Force        bool                     `json:"force,omitempty"`
}

// TransactionalDraftSaved is the result of a save.
type TransactionalDraftSaved struct {
	DraftRevision int       `json:"draftRevision"`
	SavedAt       time.Time `json:"savedAt"`
	DraftHash     string    `json:"draftHash"`
	DraftChanges  bool      `json:"draftChanges"`
	SizeBytes     int       `json:"sizeBytes"`
	HTML          string    `json:"html"`
	PlainText     string    `json:"plainText"`
}

// SaveDraft stores a new draft revision.
func (s *Service) SaveDraft(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in TransactionalDraftSave) (TransactionalDraftSaved, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalDraftSaved{}, err
	}
	if in.Source == "" {
		in.Source = "autosave"
	}
	var out TransactionalDraftSaved
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.TxnTemplateForUpdate(ctx, store.TxnTemplateForUpdateParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapTemplateErr(ctx, err)
		}
		if int(cur.DraftRevision) != in.BaseRevision && !in.Force {
			return s.draftConflict(ctx, q, cur, in)
		}
		c := content{Subject: cur.Subject, Preheader: cur.Preheader, EditorMode: cur.EditorMode, MJML: cur.Mjml, HTML: cur.Html, Text: cur.PlainText, Variables: parseVariables(cur.Variables)}
		testData := parseTestData(cur.TestData)
		sender, replyTo, opens, clicks := cur.SenderID, cur.ReplyTo, cur.TrackOpens, cur.TrackClicks
		contentChanged := false
		if in.Subject != nil {
			if strings.ContainsAny(*in.Subject, "\r\n") {
				return api.Invalid("body.subject", say(ctx, "template.subjectOneLine", nil))
			}
			c.Subject = *in.Subject
			contentChanged = true
		}
		if in.Preheader != nil {
			c.Preheader = *in.Preheader
			contentChanged = true
		}
		if in.EditorMode != nil {
			c.EditorMode = *in.EditorMode
		}
		if in.MJML != nil {
			c.MJML = *in.MJML
			contentChanged = true
		}
		if in.HTML != nil {
			c.HTML = *in.HTML
		} else if in.MJML != nil || in.Subject != nil || in.Preheader != nil {
			if strings.TrimSpace(c.MJML) != "" {
				c.HTML = "" // compiled below
			}
		}
		if in.PlainText != nil {
			c.Text = *in.PlainText
		}
		if in.Variables != nil {
			if err := validateVariables(ctx, *in.Variables); err != nil {
				return err
			}
			c.Variables = *in.Variables
		}
		if in.TestData != nil {
			if err := validateTestData(ctx, *in.TestData); err != nil {
				return err
			}
			testData = *in.TestData
		}
		if in.SenderID != nil {
			if *in.SenderID == "" {
				sender = nil
			} else {
				sid, err := uuid.Parse(*in.SenderID)
				if err != nil {
					return api.Invalid("body.senderId", say(ctx, "sender.pick", nil))
				}
				snd, err := q.TxnSenderByID(ctx, store.TxnSenderByIDParams{WorkspaceID: ws.ID, ID: sid})
				if err != nil {
					return api.Invalid("body.senderId", say(ctx, "sender.gone", nil))
				}
				if !snd.Transactional {
					return api.Invalid("body.senderId", say(ctx, "sender.notTransactionalShort", i18n.Args{"email": snd.Email}))
				}
				sender = &sid
			}
		}
		if in.ReplyTo != nil {
			r := strings.TrimSpace(*in.ReplyTo)
			if r != "" {
				if _, err := mail.ParseAddress(r); err != nil {
					return api.Invalid("body.replyTo", say(ctx, "replyTo.invalid", nil))
				}
			}
			replyTo = r
		}
		if in.TrackOpens != nil {
			opens = *in.TrackOpens
		}
		if in.TrackClicks != nil {
			clicks = *in.TrackClicks
		}
		if strings.TrimSpace(c.HTML) == "" && strings.TrimSpace(c.MJML) != "" {
			html, err := c.compiledHTML()
			if err != nil {
				return api.Invalid("body.mjml", problemText(lang(ctx), err))
			}
			c.HTML = html
		}
		// lang, dir and the layout tables' role (accessibility).
		c.HTML = accessibility.FinalizeStored(ctx, q, ws.ID, accessibility.OwnerTransactional, id, 0, c.HTML)
		_ = contentChanged
		hash := draftHash(c.Subject, c.Preheader, c.EditorMode, c.MJML, c.HTML, c.Text, c.Variables, sender, replyTo, opens, clicks)
		rev := cur.DraftRevision + 1
		now := s.now()
		if err := q.TxnTemplateSaveDraft(ctx, store.TxnTemplateSaveDraftParams{WorkspaceID: ws.ID, ID: id, Subject: c.Subject, Preheader: c.Preheader, EditorMode: c.EditorMode,
			Mjml: c.MJML, Html: c.HTML, PlainText: c.Text, Variables: mustJSON(c.Variables), TestData: mustJSON(testData), SenderID: sender, ReplyTo: replyTo,
			TrackOpens: opens, TrackClicks: clicks, DraftRevision: rev, DraftHash: hash, Now: now, SavedBy: userRef(p)}); err != nil {
			return err
		}
		label := strings.TrimSpace(in.Label)
		if err := q.TxnTemplateInsertDraftRevision(ctx, store.TxnTemplateInsertDraftRevisionParams{TemplateID: id, Revision: rev, Subject: c.Subject, Preheader: c.Preheader,
			Mjml: c.MJML, Html: c.HTML, Variables: mustJSON(c.Variables), SavedBy: userRef(p), SavedAt: now, Label: label, Source: in.Source}); err != nil {
			return err
		}
		if err := q.TxnTemplatePruneDrafts(ctx, store.TxnTemplatePruneDraftsParams{TemplateID: id, Below: rev - KeepDraftRevisions}); err != nil {
			return err
		}
		live := ""
		if cur.LiveVersion != nil {
			if v, err := q.TxnVersion(ctx, store.TxnVersionParams{TemplateID: id, Version: *cur.LiveVersion}); err == nil {
				live = v.ContentHash
			}
		}
		out = TransactionalDraftSaved{DraftRevision: int(rev), SavedAt: now, DraftHash: hash, DraftChanges: live == "" || live != hash, SizeBytes: len(c.HTML), HTML: c.HTML, PlainText: c.Text}
		before := map[string]any{"draft_revision": cur.DraftRevision}
		after := map[string]any{"draft_revision": rev, "size_bytes": len(c.HTML)}
		if cur.Subject != c.Subject {
			before["subject"], after["subject"] = cur.Subject, c.Subject
		}
		if string(cur.Variables) != string(mustJSON(c.Variables)) {
			before["variables"], after["variables"] = variableNames(parseVariables(cur.Variables)), variableNames(c.Variables)
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.template_saved", TargetType: "transactional_template", TargetID: id.String(), TargetName: cur.Key,
			Before: before, After: after, Metadata: map[string]any{"label": label, "source": in.Source}})
	})
	return out, err
}

func variableNames(vars []TransactionalVariable) []string {
	out := make([]string, 0, len(vars))
	for _, v := range vars {
		out = append(out, v.Name)
	}
	return out
}

// draftConflict is the 409 sent when a teammate saved a newer revision: who,
// when, and which parts each side changed.
func (s *Service) draftConflict(ctx context.Context, q *store.Queries, cur store.TransactionalTemplate, in TransactionalDraftSave) error {
	theirs, mine := []string{}, []string{}
	if base, err := q.TxnTemplateDraftRevision(ctx, store.TxnTemplateDraftRevisionParams{TemplateID: cur.ID, Revision: int32(in.BaseRevision)}); err == nil {
		theirs = append(theirs, mjml.ChangedSections(base.Mjml, cur.Mjml)...)
		if base.Subject != cur.Subject {
			theirs = append(theirs, "subject")
		}
		if string(base.Variables) != string(cur.Variables) {
			theirs = append(theirs, "variables")
		}
		if in.MJML != nil {
			mine = append(mine, mjml.ChangedSections(base.Mjml, *in.MJML)...)
		}
		if in.Subject != nil && *in.Subject != base.Subject {
			mine = append(mine, "subject")
		}
		if in.Variables != nil && string(mustJSON(*in.Variables)) != string(base.Variables) {
			mine = append(mine, "variables")
		}
	}
	who := say(ctx, "template.someone", nil)
	if cur.DraftSavedBy != nil {
		if u, err := q.GetUser(ctx, *cur.DraftSavedBy); err == nil && u.Name != "" {
			who = u.Name
		}
	}
	return api.Err(http.StatusConflict, CodeVersionConflict, say(ctx, "template.draftConflict", i18n.Args{"who": who, "time": clockIn(ctx, cur.DraftSavedAt)})).
		WithMeta("draftRevision", cur.DraftRevision, "savedBy", who, "savedAt", cur.DraftSavedAt, "theirChanges", theirs, "yourChanges", mine)
}

// Archive takes a template out of use: the API answers template_not_found
// for it until it is restored.
func (s *Service) Archive(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, archive bool) (TransactionalTemplate, error) {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return TransactionalTemplate{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.TxnTemplateForUpdate(ctx, store.TxnTemplateForUpdateParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapTemplateErr(ctx, err)
		}
		if (cur.ArchivedAt != nil) == archive {
			return nil
		}
		now := s.now()
		var at *time.Time
		var by *uuid.UUID
		action := "transactional.template_restored"
		if archive {
			at, by = &now, userRef(p)
			action = "transactional.template_archived"
		}
		if err := q.TxnTemplateArchive(ctx, store.TxnTemplateArchiveParams{WorkspaceID: ws.ID, ID: id, ArchivedAt: at, ArchivedBy: by, Now: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "transactional_template", TargetID: id.String(), TargetName: cur.Key,
			Before: map[string]any{"archived": cur.ArchivedAt != nil}, After: map[string]any{"archived": archive}})
	})
	if err != nil {
		return TransactionalTemplate{}, err
	}
	return s.Template(ctx, ws, p, id.String())
}

// DeleteTemplate removes a template; its messages stay in the log.
func (s *Service) DeleteTemplate(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) error {
	id, err := s.resolveTemplate(ctx, s.Q, ws.ID, ref)
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.TxnTemplateForUpdate(ctx, store.TxnTemplateForUpdateParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return mapTemplateErr(ctx, err)
		}
		if cur.LiveVersion != nil && cur.ArchivedAt == nil {
			return api.Err(http.StatusConflict, api.CodeConflict, say(ctx, "template.deleteLive", i18n.Args{"key": cur.Key}))
		}
		if err := q.TxnTemplateDelete(ctx, store.TxnTemplateDeleteParams{WorkspaceID: ws.ID, ID: id, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.template_deleted", TargetType: "transactional_template", TargetID: id.String(), TargetName: cur.Key,
			Before: map[string]any{"name": cur.Name, "key": cur.Key, "live_version": cur.LiveVersion}})
	})
}

// plainText is the text part of an HTML body.
func plainText(html string) string { return contentcheck.PlainText(html) }
