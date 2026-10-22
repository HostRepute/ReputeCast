// Package templates holds the composer's template gallery: the starter
// templates that ship with ReputeCast (MJML files embedded in the binary,
// their placeholder images served at /media/starters/), the workspace's
// saved templates and its saved sections (reusable blocks).
package templates

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed starters
var startersFS embed.FS

// StarterAssets are the placeholder images of the starter templates.
func StarterAssets() fs.FS {
	sub, _ := fs.Sub(startersFS, "starters")
	return sub
}

// StarterPrefix marks the ids of built-in templates ("starter:weekly-digest").
const StarterPrefix = "starter:"

// AddressPlaceholder is replaced by the workspace's postal address.
const AddressPlaceholder = "%%ADDRESS%%"

// DefaultAddress is shown when the workspace has no postal address yet;
// the content check flags it until it is replaced.
const DefaultAddress = "Your company · Street and number · City, postcode"

type starter struct {
	key, name, description string
	thumb                  []mjml.MJMLThumbItem
	// digest starters are offered in the composer's digest mode only (a
	// recurring digest's template), and only there.
	digest bool
}

// The gallery order and wireframes of the starters (ComposerEmpty board).
var starters = []starter{
	{key: "blank", name: "Blank", description: "Just a body and a footer block", thumb: []mjml.MJMLThumbItem{}},
	{key: "weekly-digest", name: "Weekly digest", description: "Hero, two article cards, footer", thumb: []mjml.MJMLThumbItem{{H: 10, W: 40}, {H: 34}, {Row: []mjml.MJMLThumbItem{{H: 28}, {H: 28}}}, {H: 8}}},
	{key: "product-update", name: "Product update", description: "One announcement, one button", thumb: []mjml.MJMLThumbItem{{H: 10, W: 40}, {H: 40}, {H: 8, W: 70}, {H: 14, W: 35, Accent: true}}},
	{key: "plain-text", name: "Plain text", description: "Personal note, no images", thumb: []mjml.MJMLThumbItem{{H: 8, W: 90}, {H: 8, W: 85}, {H: 8, W: 88}, {H: 8, W: 60}}},
	{key: "announcement", name: "Announcement", description: "Full-width hero", thumb: []mjml.MJMLThumbItem{{H: 48}, {H: 10, W: 60}, {H: 8, W: 80}}},
	{key: "product-grid", name: "Product grid", description: "Three product cards per row", thumb: []mjml.MJMLThumbItem{{Row: []mjml.MJMLThumbItem{{H: 30}, {H: 30}, {H: 30}}}, {H: 8, W: 70}, {Row: []mjml.MJMLThumbItem{{H: 30}, {H: 30}, {H: 30}}}}},
	// Recurring digests (design expansion, phase 3).
	{key: "blog-digest", name: "Blog digest", description: "Header, intro, the feed's posts as cards, footer", digest: true,
		thumb: []mjml.MJMLThumbItem{{H: 10, W: 40}, {H: 8, W: 85}, {H: 30}, {H: 8, W: 70}, {H: 30}, {H: 8}}},
}

// BlogDigestStarter is the key of the recurring digest's starter.
const BlogDigestStarter = "blog-digest"

// StarterMJML returns a starter's MJML with the workspace address filled in
// and image URLs pointing at /media/starters/, its sample content in
// English.
func StarterMJML(key, address string) (string, bool) {
	return StarterMJMLIn(i18n.Default, key, address)
}

// StarterMJMLIn is StarterMJML with the sample content written in l, for
// the readers of the email someone starts from it.
func StarterMJMLIn(l i18n.Lang, key, address string) (string, bool) {
	b, err := startersFS.ReadFile("starters/" + key + ".mjml")
	if err != nil {
		return "", false
	}
	if strings.TrimSpace(address) == "" {
		address = DefaultAddressIn(l)
	}
	src := samples(l, string(b))
	src = strings.ReplaceAll(src, AddressPlaceholder, escapeText(address))
	src = strings.ReplaceAll(src, `src="rc-asset:`, `src="/media/starters/`)
	return src, true
}

func escapeText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// Service holds the templates domain dependencies.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// EmailTemplateSummary is one tile of the gallery.
type EmailTemplateSummary struct {
	ID          string               `json:"id" example:"starter:weekly-digest"`
	Kind        string               `json:"kind" enum:"template,section"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Thumb       []mjml.MJMLThumbItem `json:"thumb"`
	Builtin     bool                 `json:"builtin"`
	UpdatedAt   *time.Time           `json:"updatedAt,omitempty"`
	UpdatedBy   string               `json:"updatedBy,omitempty"`
	SizeBytes   int                  `json:"sizeBytes"`
	// SourceCampaignID is the campaign a template was saved from.
	SourceCampaignID string `json:"sourceCampaignId,omitempty"`
}

// EmailTemplateDetail adds the content.
type EmailTemplateDetail struct {
	EmailTemplateSummary
	MJML string `json:"mjml"`
	HTML string `json:"html"`
}

// EmailTemplateLastCampaign is the most recent sent campaign, offered as "Copy of …".
type EmailTemplateLastCampaign struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	SentAt time.Time            `json:"sentAt"`
	Thumb  []mjml.MJMLThumbItem `json:"thumb"`
}

// EmailTemplateGallery is the empty-canvas gallery and the Templates tab.
type EmailTemplateGallery struct {
	Starters     []EmailTemplateSummary     `json:"starters"`
	Templates    []EmailTemplateSummary     `json:"templates"`
	Sections     []EmailTemplateSummary     `json:"sections"`
	LastCampaign *EmailTemplateLastCampaign `json:"lastCampaign,omitempty"`
}

// Gallery lists starters, the workspace's templates and saved sections, and
// the last sent campaign.
func (s *Service) Gallery(ctx context.Context, ws *store.Workspace) (EmailTemplateGallery, error) {
	return s.GalleryFor(ctx, ws, false)
}

// GalleryFor is the gallery of the composer's regular mode, or of its
// digest mode (digest true), which adds the digest starters.
func (s *Service) GalleryFor(ctx context.Context, ws *store.Workspace, digest bool) (EmailTemplateGallery, error) {
	g := EmailTemplateGallery{Starters: []EmailTemplateSummary{}, Templates: []EmailTemplateSummary{}, Sections: []EmailTemplateSummary{}}
	l := i18n.LangFrom(ctx)
	for _, st := range starters {
		if st.digest && !digest {
			continue
		}
		src, _ := StarterMJMLIn(l, st.key, "")
		name, desc := starterName(l, st)
		g.Starters = append(g.Starters, EmailTemplateSummary{ID: StarterPrefix + st.key, Kind: "template", Name: name, Description: desc, Thumb: st.thumb, Builtin: true, SizeBytes: len(src)})
	}
	rows, err := s.Q.ListEmailTemplates(ctx, store.ListEmailTemplatesParams{WorkspaceID: ws.ID, Kind: ""})
	if err != nil {
		return g, err
	}
	for _, r := range rows {
		t := summary(r.ID, r.Kind, r.Name, r.Description, r.Thumb, r.SizeBytes, r.SourceCampaignID, r.UpdatedAt, r.UpdatedByName)
		if r.Kind == "section" {
			g.Sections = append(g.Sections, t)
		} else {
			g.Templates = append(g.Templates, t)
		}
	}
	last, err := s.Q.ComposerLastSentCampaign(ctx, ws.ID)
	switch {
	case err == nil:
		g.LastCampaign = &EmailTemplateLastCampaign{ID: last.ID.String(), Name: last.Name, SentAt: last.SentAt, Thumb: mjml.Thumb(last.Mjml)}
	case !errors.Is(err, pgx.ErrNoRows):
		return g, err
	}
	return g, nil
}

func summary(id uuid.UUID, kind, name, desc string, thumb json.RawMessage, size int32, source *uuid.UUID, updated time.Time, by string) EmailTemplateSummary {
	t := EmailTemplateSummary{ID: id.String(), Kind: kind, Name: name, Description: desc, Thumb: []mjml.MJMLThumbItem{}, SizeBytes: int(size), UpdatedBy: by}
	_ = json.Unmarshal(thumb, &t.Thumb)
	if t.Thumb == nil {
		t.Thumb = []mjml.MJMLThumbItem{}
	}
	u := updated
	t.UpdatedAt = &u
	if source != nil {
		t.SourceCampaignID = source.String()
	}
	return t
}

// Get returns a template (starter or saved) with its content. address fills
// a starter's footer.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, id, address string) (EmailTemplateDetail, error) {
	if key, ok := strings.CutPrefix(id, StarterPrefix); ok {
		l := i18n.LangFrom(ctx)
		for _, st := range starters {
			if st.key == key {
				src, _ := StarterMJMLIn(l, key, address)
				name, desc := starterName(l, st)
				return EmailTemplateDetail{EmailTemplateSummary: EmailTemplateSummary{ID: id, Kind: "template", Name: name, Description: desc, Thumb: st.thumb, Builtin: true, SizeBytes: len(src)}, MJML: src}, nil
			}
		}
		return EmailTemplateDetail{}, notFound(ctx)
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return EmailTemplateDetail{}, notFound(ctx)
	}
	r, err := s.Q.GetEmailTemplate(ctx, store.GetEmailTemplateParams{WorkspaceID: ws.ID, ID: uid})
	if err != nil {
		return EmailTemplateDetail{}, mapError(ctx, err)
	}
	return EmailTemplateDetail{EmailTemplateSummary: summary(r.ID, r.Kind, r.Name, r.Description, r.Thumb, r.SizeBytes, r.SourceCampaignID, r.UpdatedAt, r.UpdatedByName), MJML: r.Mjml, HTML: r.Html}, nil
}

// EmailTemplateInput is a new or changed template or saved section.
type EmailTemplateInput struct {
	Kind             string `json:"kind,omitempty" enum:"template,section" default:"template"`
	Name             string `json:"name" minLength:"1" maxLength:"120"`
	Description      string `json:"description,omitempty" maxLength:"300"`
	MJML             string `json:"mjml" minLength:"1" maxLength:"2000000"`
	HTML             string `json:"html,omitempty" maxLength:"2000000"`
	SourceCampaignID string `json:"sourceCampaignId,omitempty" format:"uuid"`
}

func (s *Service) validate(ctx context.Context, q *store.Queries, ws *store.Workspace, id uuid.UUID, in *EmailTemplateInput) error {
	l := i18n.LangFrom(ctx)
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return api.Invalid("body.name", mName.Text(l, nil))
	}
	src := in.MJML
	if in.Kind == "section" && !strings.Contains(src, "<mjml") {
		src = "<mjml><mj-body>" + src + "</mj-body></mjml>"
	}
	if _, err := mjml.Parse(src); err != nil {
		return api.Invalid("body.mjml", mMJML.Text(l, i18n.Args{"error": err.Error()}))
	}
	taken, err := q.EmailTemplateNameTaken(ctx, store.EmailTemplateNameTakenParams{WorkspaceID: ws.ID, Kind: in.Kind, Name: in.Name, ID: id})
	if err != nil {
		return err
	}
	if taken {
		return api.Invalid("body.name", mNameTaken.Text(l, i18n.Args{"kind": in.Kind, "name": in.Name}))
	}
	return nil
}

func thumbOf(kind, src string) []mjml.MJMLThumbItem {
	if kind == "section" && !strings.Contains(src, "<mjml") {
		src = "<mjml><mj-body>" + src + "</mj-body></mjml>"
	}
	return mjml.Thumb(src)
}

// Create saves a template or a section.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, actor *uuid.UUID, in EmailTemplateInput) (EmailTemplateDetail, error) {
	if in.Kind == "" {
		in.Kind = "template"
	}
	id := uuid.Must(uuid.NewV7())
	var source *uuid.UUID
	if in.SourceCampaignID != "" {
		if u, err := uuid.Parse(in.SourceCampaignID); err == nil {
			source = &u
		}
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := s.validate(ctx, q, ws, uuid.Nil, &in); err != nil {
			return err
		}
		thumb, _ := json.Marshal(thumbOf(in.Kind, in.MJML))
		if err := q.InsertEmailTemplate(ctx, store.InsertEmailTemplateParams{ID: id, WorkspaceID: ws.ID, Kind: in.Kind, Name: in.Name, Description: in.Description,
			Mjml: in.MJML, Html: in.HTML, Thumb: thumb, SizeBytes: int32(len(in.HTML)), SourceCampaignID: source, CreatedBy: actor, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "template.create", TargetType: "template", TargetID: id.String(), TargetName: in.Name,
			After: map[string]any{"kind": in.Kind, "name": in.Name, "sourceCampaignId": in.SourceCampaignID}})
	})
	if err != nil {
		return EmailTemplateDetail{}, err
	}
	return s.Get(ctx, ws, id.String(), "")
}

// Update renames a template or replaces its content.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, actor *uuid.UUID, id string, in EmailTemplateInput) (EmailTemplateDetail, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return EmailTemplateDetail{}, notFound(ctx)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.GetEmailTemplate(ctx, store.GetEmailTemplateParams{WorkspaceID: ws.ID, ID: uid})
		if err != nil {
			return mapError(ctx, err)
		}
		in.Kind = cur.Kind
		if err := s.validate(ctx, q, ws, uid, &in); err != nil {
			return err
		}
		thumb, _ := json.Marshal(thumbOf(in.Kind, in.MJML))
		if err := q.UpdateEmailTemplate(ctx, store.UpdateEmailTemplateParams{WorkspaceID: ws.ID, ID: uid, Name: in.Name, Description: in.Description, Mjml: in.MJML, Html: in.HTML,
			Thumb: thumb, SizeBytes: int32(len(in.HTML)), UpdatedBy: actor, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "template.update", TargetType: "template", TargetID: id, TargetName: in.Name,
			Before: map[string]any{"name": cur.Name, "description": cur.Description, "sizeBytes": cur.SizeBytes},
			After:  map[string]any{"name": in.Name, "description": in.Description, "sizeBytes": len(in.HTML)}})
	})
	if err != nil {
		return EmailTemplateDetail{}, err
	}
	return s.Get(ctx, ws, id, "")
}

// Delete removes a template or saved section.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, actor *uuid.UUID, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return notFound(ctx)
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.GetEmailTemplate(ctx, store.GetEmailTemplateParams{WorkspaceID: ws.ID, ID: uid})
		if err != nil {
			return mapError(ctx, err)
		}
		now := s.now()
		if _, err := q.SoftDeleteEmailTemplate(ctx, store.SoftDeleteEmailTemplateParams{WorkspaceID: ws.ID, ID: uid, Now: &now, UpdatedBy: actor}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "template.delete", TargetType: "template", TargetID: id, TargetName: cur.Name,
			Before: map[string]any{"kind": cur.Kind, "name": cur.Name}})
	})
}
