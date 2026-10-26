package brandkit

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// Synced block kinds.
const (
	KindHeader = "header"
	KindFooter = "footer"
)

// Kinds are the synced blocks, in the order the kit lists them.
var Kinds = []string{KindFooter, KindHeader}

// Markers of the header and footer sections in starters and blocks.
var markers = map[string]string{KindHeader: "<!-- rc:header-logo -->", KindFooter: "<!-- rc:footer -->"}

// attrLogo marks the image in the synced Header that shows the kit's logo.
const attrLogo = "rc-logo"

// SyncedBlock is a synced block and the emails holding it.
type SyncedBlock struct {
	Kind      string    `json:"kind" enum:"header,footer"`
	MJML      string    `json:"mjml" doc:"One section or wrapper, marked rc-synced"`
	Version   int       `json:"version"`
	UpdatedBy string    `json:"updatedBy"`
	UpdatedAt time.Time `json:"updatedAt"`
	// Uses counts the emails holding it: what a change does to them.
	Uses campaigns.SyncedUses `json:"uses"`
}

// SyncedInput changes a synced block.
type SyncedInput struct {
	MJML string `json:"mjml" minLength:"1" maxLength:"200000" doc:"One mj-section or mj-wrapper"`
}

// SyncedSaved is a synced block's change and what it did to the emails.
type SyncedSaved struct {
	Block SyncedBlock `json:"block"`
	// Applied counts the drafts and scheduled emails that got the change,
	// and the held and sent ones that kept theirs.
	Applied campaigns.SyncedUses `json:"applied"`
}

func (s *Service) syncedBlocks(ctx context.Context, ws *store.Workspace) ([]SyncedBlock, error) {
	rows, err := s.Q.SyncedBlocksList(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := make([]SyncedBlock, 0, len(rows))
	for _, kind := range Kinds {
		for _, r := range rows {
			if r.Kind != kind {
				continue
			}
			b := SyncedBlock{Kind: r.Kind, MJML: r.Mjml, Version: int(r.Version), UpdatedBy: r.UpdatedByName, UpdatedAt: r.UpdatedAt}
			if s.Campaigns != nil {
				if b.Uses, err = s.Campaigns.CountSynced(ctx, ws, r.Kind); err != nil {
					return nil, err
				}
			}
			out = append(out, b)
		}
	}
	return out, nil
}

// sectionAfter is the element that follows a marker comment.
func sectionAfter(src, marker string) string {
	at := strings.Index(src, marker)
	if at < 0 {
		return ""
	}
	root, err := mjml.Parse(src)
	if err != nil {
		return ""
	}
	var found *mjml.Node
	root.Walk(func(n *mjml.Node, _ []int) {
		if found == nil && n.Start > at && (n.Tag == "mj-section" || n.Tag == "mj-wrapper") {
			found = n
		}
	})
	if found == nil {
		return ""
	}
	return src[found.Start:found.End]
}

// defaultBlock is a synced block as the kit starts it: the starters' header
// (the logo) or footer (postal address, unsubscribe and preferences links),
// in the kit's style and the saver's language.
func defaultBlock(l i18n.Lang, kind, address string, k BrandKit) string {
	starter := "blank"
	if kind == KindHeader {
		starter = "product-update"
	}
	src, _ := templates.StarterMJMLIn(l, starter, address)
	sec := Apply(sectionAfter(src, markers[kind]), k)
	if kind == KindHeader {
		sec = markLogo(sec)
	}
	out, _ := mjml.MarkSynced(sec, kind)
	return out
}

// markLogo marks the header's first image as the kit's logo.
func markLogo(block string) string {
	doc := mjml.WrapFragment(block)
	root, err := mjml.Parse(doc)
	if err != nil {
		return block
	}
	if img := root.Find("mj-image"); img != nil {
		doc, _ = mjml.SetAttr(doc, img, attrLogo, "light")
	}
	return mjml.UnwrapFragment(doc)
}

// logoImage is the header's image marked as the kit's logo.
func logoImage(doc string) *mjml.Node {
	root, err := mjml.Parse(doc)
	if err != nil {
		return nil
	}
	var img *mjml.Node
	root.Walk(func(n *mjml.Node, _ []int) {
		if img == nil && n.Tag == "mj-image" {
			if _, ok := n.Get(attrLogo); ok {
				img = n
			}
		}
	})
	return img
}

// ensureSynced creates the synced Header and Footer the first time the kit
// is saved.
func (s *Service) ensureSynced(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, k BrandKit) error {
	l := i18n.LangFrom(ctx)
	for _, kind := range Kinds {
		if _, err := q.SyncedBlockGet(ctx, store.SyncedBlockGetParams{WorkspaceID: ws.ID, Kind: kind}); err == nil {
			continue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		block := defaultBlock(l, kind, templates.WorkspaceAddress(ws), k)
		if block == "" {
			continue
		}
		now := s.now()
		if err := q.SyncedBlockInsert(ctx, store.SyncedBlockInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Kind: kind, Mjml: block, UpdatedBy: actor(p), Now: now}); err != nil {
			return err
		}
	}
	return nil
}

// relogo puts the kit's light logo and its alt text in the synced Header's
// logo image. It returns the Header when it changed.
func (s *Service) relogo(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, k BrandKit) (*SyncedBlock, error) {
	r, err := q.SyncedBlockGet(ctx, store.SyncedBlockGetParams{WorkspaceID: ws.ID, Kind: KindHeader})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	doc := mjml.WrapFragment(r.Mjml)
	img := logoImage(doc)
	if img == nil || k.LogoLight == nil {
		return nil, nil
	}
	// The element starts where it did after each change of its opening tag.
	doc, _ = mjml.SetAttrRaw(doc, img, "alt", k.LogoAlt)
	if img = logoImage(doc); img == nil {
		return nil, nil
	}
	doc, _ = mjml.SetAttr(doc, img, "src", k.LogoLight.URL)
	src := mjml.UnwrapFragment(doc)
	if src == r.Mjml {
		return nil, nil
	}
	return s.storeSynced(ctx, q, ws, p, KindHeader, r.Version, src)
}

// storeSynced saves a synced block's next version, audited.
func (s *Service) storeSynced(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, kind string, prev int32, src string) (*SyncedBlock, error) {
	now := s.now()
	if err := q.SyncedBlockUpdate(ctx, store.SyncedBlockUpdateParams{WorkspaceID: ws.ID, Kind: kind, Mjml: src, UpdatedBy: actor(p), Now: now}); err != nil {
		return nil, err
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "synced_block.save", TargetType: "synced_block", TargetID: ws.ID.String() + ":" + kind,
		TargetName: kind, Before: map[string]any{"version": prev}, After: map[string]any{"version": prev + 1, "sizeBytes": len(src)}}); err != nil {
		return nil, err
	}
	name := ""
	if p != nil {
		name = p.Name
	}
	return &SyncedBlock{Kind: kind, MJML: src, Version: int(prev + 1), UpdatedBy: name, UpdatedAt: now}, nil
}

// propagate puts a synced block's new version into the emails holding it.
func (s *Service) propagate(ctx context.Context, ws *store.Workspace, p *authz.Principal, b SyncedBlock) (campaigns.SyncedUses, error) {
	if s.Campaigns == nil {
		return campaigns.SyncedUses{}, nil
	}
	uses, _, err := s.Campaigns.ApplySynced(ctx, ws, p, b.Kind, b.MJML, campaigns.SyncedLabel(b.Kind))
	return uses, err
}

var unsubscribeToken = regexp.MustCompile(`\{\{-?\s*unsubscribe_url\s*-?\}\}`)

// SaveSynced changes a synced block: drafts and scheduled emails holding it
// get the change as a new version; emails in an approval, sending, paused
// or sent keep theirs.
func (s *Service) SaveSynced(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, in SyncedInput) (SyncedSaved, error) {
	l := i18n.LangFrom(ctx)
	if kind != KindHeader && kind != KindFooter {
		return SyncedSaved{}, api.NotFound("Synced block")
	}
	block, ok := mjml.MarkSynced(in.MJML, kind)
	if !ok {
		return SyncedSaved{}, api.Invalid("body.mjml", mOneSection.Text(l, nil))
	}
	if res := mjml.CompileIn(l, "<mjml><mj-body>"+block+"</mj-body></mjml>"); len(res.Errors) > 0 {
		return SyncedSaved{}, api.Invalid("body.mjml", mCompile.Text(l, i18n.Args{"error": res.Errors[0].Message}))
	}
	if kind == KindFooter && !unsubscribeToken.MatchString(block) {
		return SyncedSaved{}, api.Invalid("body.mjml", mNeedsUnsubscribe.Text(l, nil))
	}
	var saved *SyncedBlock
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.SyncedBlockGet(ctx, store.SyncedBlockGetParams{WorkspaceID: ws.ID, Kind: kind})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Err(409, "brand_kit_not_saved", mNotSaved.Text(l, nil))
		}
		if err != nil {
			return err
		}
		if r.Mjml == block {
			saved = &SyncedBlock{Kind: kind, MJML: r.Mjml, Version: int(r.Version), UpdatedBy: r.UpdatedByName, UpdatedAt: r.UpdatedAt}
			return nil
		}
		saved, err = s.storeSynced(ctx, q, ws, p, kind, r.Version, block)
		return err
	})
	if err != nil {
		return SyncedSaved{}, err
	}
	applied, err := s.propagate(ctx, ws, p, *saved)
	if err != nil {
		return SyncedSaved{}, err
	}
	if s.Campaigns != nil {
		if saved.Uses, err = s.Campaigns.CountSynced(ctx, ws, kind); err != nil {
			return SyncedSaved{}, err
		}
	}
	return SyncedSaved{Block: *saved, Applied: applied}, nil
}

// Starter puts the workspace's kit on a starter template: its style and
// logo, and its synced Header and Footer in place of the starter's own.
func (s *Service) Starter(ctx context.Context, ws *store.Workspace, src string) string {
	k, err := s.Kit(ctx, ws)
	if err != nil || !k.Saved {
		return src
	}
	src = Apply(src, k)
	rows, err := s.Q.SyncedBlocksList(ctx, ws.ID)
	if err != nil {
		return src
	}
	for _, r := range rows {
		old := sectionAfter(src, markers[r.Kind])
		if old == "" {
			continue
		}
		at := strings.Index(src, markers[r.Kind])
		if i := strings.Index(src[at:], old); i >= 0 {
			src = src[:at+i] + r.Mjml + src[at+i+len(old):]
		}
	}
	return src
}
