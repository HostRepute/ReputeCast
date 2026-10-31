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

// Kinds are the synced blocks every kit has, in the order the kit lists
// them; named blocks (R4.A3) follow, oldest first.
var Kinds = []string{KindFooter, KindHeader}

// namedKind is a named block's kind: "b-" and 12 hex digits, the key the
// emails holding it carry in rc-synced (synced_blocks_kind_check).
var namedKind = regexp.MustCompile(`^b-[0-9a-f]{12}$`)

// Named reports whether kind is a named synced block's (not the Header or
// Footer).
func Named(kind string) bool { return namedKind.MatchString(kind) }

// newNamedKind is a new named block's key, from a UUIDv7's random bits.
func newNamedKind() string {
	hex := strings.ReplaceAll(uuid.Must(uuid.NewV7()).String(), "-", "")
	return "b-" + hex[len(hex)-12:]
}

// Markers of the header and footer sections in starters and blocks.
var markers = map[string]string{KindHeader: "<!-- rc:header-logo -->", KindFooter: "<!-- rc:footer -->"}

// attrLogo marks the image in the synced Header that shows the kit's logo.
const attrLogo = "rc-logo"

// SyncedBlock is a synced block and the emails holding it.
type SyncedBlock struct {
	Kind string `json:"kind" pattern:"^(header|footer|b-[0-9a-f]{12})$" doc:"header, footer, or a named block's key (R4.A3)"`
	// Name is a named block's name; the Header and Footer have none.
	Name      string    `json:"name"`
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
	// Name renames a named block; the Header and Footer keep theirs.
	Name *string `json:"name,omitempty" maxLength:"80" doc:"A named block's new name"`
}

// SyncedCreateInput saves a section as a named synced block (R4.A3).
type SyncedCreateInput struct {
	Name string `json:"name" minLength:"1" maxLength:"80" doc:"What the block is called in the blocks panel and the brand kit"`
	MJML string `json:"mjml" minLength:"1" maxLength:"200000" doc:"One mj-section or mj-wrapper, from the email it is saved from"`
}

// SyncedDeleted is what deleting a named synced block did to the emails
// holding it: drafts and scheduled ones keep its content, unlinked; held
// and sent ones keep theirs as it was.
type SyncedDeleted struct {
	Applied campaigns.SyncedUses `json:"applied"`
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
	add := func(r store.SyncedBlocksListRow) error {
		b := SyncedBlock{Kind: r.Kind, Name: r.Name, MJML: r.Mjml, Version: int(r.Version), UpdatedBy: r.UpdatedByName, UpdatedAt: r.UpdatedAt}
		if s.Campaigns != nil {
			if b.Uses, err = s.Campaigns.CountSynced(ctx, ws, r.Kind); err != nil {
				return err
			}
		}
		out = append(out, b)
		return nil
	}
	for _, kind := range Kinds {
		for _, r := range rows {
			if r.Kind == kind {
				if err := add(r); err != nil {
					return nil, err
				}
			}
		}
	}
	// The named blocks, oldest first (the rows' order).
	for _, r := range rows {
		if Named(r.Kind) {
			if err := add(r); err != nil {
				return nil, err
			}
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
		if err := q.SyncedBlockInsert(ctx, store.SyncedBlockInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Kind: kind, Name: "", Mjml: block, UpdatedBy: actor(p), Now: now}); err != nil {
			return err
		}
	}
	return nil
}

// relogo puts the kit's light logo and its alt text in the synced Header's
// logo image. It returns the Header and its audit entry when it changed.
func (s *Service) relogo(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, k BrandKit) (*SyncedBlock, *audit.Event, error) {
	r, err := q.SyncedBlockGet(ctx, store.SyncedBlockGetParams{WorkspaceID: ws.ID, Kind: KindHeader})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	doc := mjml.WrapFragment(r.Mjml)
	img := logoImage(doc)
	if img == nil || k.LogoLight == nil {
		return nil, nil, nil
	}
	// The element starts where it did after each change of its opening tag.
	doc, _ = mjml.SetAttrRaw(doc, img, "alt", k.LogoAlt)
	if img = logoImage(doc); img == nil {
		return nil, nil, nil
	}
	doc, _ = mjml.SetAttr(doc, img, "src", k.LogoLight.URL)
	src := mjml.UnwrapFragment(doc)
	if src == r.Mjml {
		return nil, nil, nil
	}
	b, e, err := s.storeSynced(ctx, q, ws, p, KindHeader, r.Name, r.Version, src)
	if err != nil {
		return nil, nil, err
	}
	return b, &e, nil
}

// storeSynced saves a synced block's next version (and a named block's
// name) and returns its audit entry for the caller to write.
func (s *Service) storeSynced(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, kind, name string, prev int32, src string) (*SyncedBlock, audit.Event, error) {
	now := s.now()
	if err := q.SyncedBlockUpdate(ctx, store.SyncedBlockUpdateParams{WorkspaceID: ws.ID, Kind: kind, Name: name, Mjml: src, UpdatedBy: actor(p), Now: now}); err != nil {
		return nil, audit.Event{}, err
	}
	e := audit.Event{WorkspaceID: &ws.ID, Action: "synced_block.save", TargetType: "synced_block", TargetID: ws.ID.String() + ":" + kind,
		TargetName: targetName(kind, name), Before: map[string]any{"version": prev}, After: map[string]any{"version": prev + 1, "sizeBytes": len(src)}}
	by := ""
	if p != nil {
		by = p.Name
	}
	return &SyncedBlock{Kind: kind, Name: name, MJML: src, Version: int(prev + 1), UpdatedBy: by, UpdatedAt: now}, e, nil
}

// targetName names a synced block in the audit log: the Header and Footer
// by kind, a named block by its name.
func targetName(kind, name string) string {
	if Named(kind) && name != "" {
		return name
	}
	return kind
}

// versionLabel is the label of the versions a synced block's change saves
// in the emails holding it.
func versionLabel(b SyncedBlock) string {
	if Named(b.Kind) {
		return campaigns.SyncedNamedLabel(b.Name)
	}
	return campaigns.SyncedLabel(b.Kind)
}

// propagate puts a synced block's new version into the emails holding it.
func (s *Service) propagate(ctx context.Context, ws *store.Workspace, p *authz.Principal, b SyncedBlock) (campaigns.SyncedUses, error) {
	if s.Campaigns == nil {
		return campaigns.SyncedUses{}, nil
	}
	uses, _, err := s.Campaigns.ApplySynced(ctx, ws, p, b.Kind, b.MJML, versionLabel(b))
	return uses, err
}

var unsubscribeToken = regexp.MustCompile(`\{\{-?\s*unsubscribe_url\s*-?\}\}`)

// SaveSynced changes a synced block: drafts and scheduled emails holding it
// get the change as a new version; emails in an approval, sending, paused
// or sent keep theirs.
func (s *Service) SaveSynced(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string, in SyncedInput) (SyncedSaved, error) {
	var saved *SyncedBlock
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		b, events, err := s.SaveSyncedIn(ctx, q, ws, p, kind, in)
		if err != nil {
			return err
		}
		saved = b
		for _, e := range events {
			if err := s.Audit.LogWith(ctx, q, e); err != nil {
				return err
			}
		}
		return nil
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

// SaveSyncedIn is SaveSynced's work in the caller's transaction (q): it
// checks the block (one section or wrapper that compiles; the footer with
// its unsubscribe link) and stores its next version. It leaves the audit
// entries and the emails holding the block to the caller (SaveSynced;
// internal/demo seeds the Acme blocks through it).
func (s *Service) SaveSyncedIn(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, kind string, in SyncedInput) (*SyncedBlock, []audit.Event, error) {
	l := i18n.LangFrom(ctx)
	if kind != KindHeader && kind != KindFooter && !Named(kind) {
		return nil, nil, api.NotFound("Synced block")
	}
	block, err := checkBlock(l, kind, in.MJML)
	if err != nil {
		return nil, nil, err
	}
	r, err := q.SyncedBlockGet(ctx, store.SyncedBlockGetParams{WorkspaceID: ws.ID, Kind: kind})
	if errors.Is(err, pgx.ErrNoRows) {
		if Named(kind) {
			return nil, nil, api.NotFound("Synced block")
		}
		return nil, nil, api.Err(409, "brand_kit_not_saved", mNotSaved.Text(l, nil))
	}
	if err != nil {
		return nil, nil, err
	}
	name := r.Name
	if in.Name != nil && Named(kind) {
		if name = strings.TrimSpace(*in.Name); name == "" {
			return nil, nil, api.Invalid("body.name", mNameRequired.Text(l, nil))
		}
	}
	if r.Mjml == block && r.Name == name {
		return &SyncedBlock{Kind: kind, Name: r.Name, MJML: r.Mjml, Version: int(r.Version), UpdatedBy: r.UpdatedByName, UpdatedAt: r.UpdatedAt}, nil, nil
	}
	saved, e, err := s.storeSynced(ctx, q, ws, p, kind, name, r.Version, block)
	if err != nil {
		return nil, nil, err
	}
	return saved, []audit.Event{e}, nil
}

// checkBlock marks a synced block's MJML with its kind and checks it: one
// section or wrapper that compiles and holds no other synced block, and the
// footer with its unsubscribe link.
func checkBlock(l i18n.Lang, kind, src string) (string, error) {
	block, ok := mjml.MarkSynced(src, kind)
	if !ok {
		return "", api.Invalid("body.mjml", mOneSection.Text(l, nil))
	}
	// A block inside a block would be replaced by the outer one's changes.
	if len(mjml.SyncedKinds(mjml.WrapFragment(block))) > 1 {
		return "", api.Invalid("body.mjml", mHoldsSynced.Text(l, nil))
	}
	if res := mjml.CompileIn(l, "<mjml><mj-body>"+block+"</mj-body></mjml>"); len(res.Errors) > 0 {
		return "", api.Invalid("body.mjml", mCompile.Text(l, i18n.Args{"error": res.Errors[0].Message}))
	}
	if kind == KindFooter && !unsubscribeToken.MatchString(block) {
		return "", api.Invalid("body.mjml", mNeedsUnsubscribe.Text(l, nil))
	}
	return block, nil
}

// CreateSynced saves a section as a named synced block (R4.A3, "Save as
// synced block" in the composer). The block it returns is the section
// marked with its new key: the composer puts it in place of the section,
// so the email it came from holds the block too. Other emails get it from
// the blocks panel.
func (s *Service) CreateSynced(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SyncedCreateInput) (SyncedBlock, error) {
	l := i18n.LangFrom(ctx)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return SyncedBlock{}, api.Invalid("body.name", mNameRequired.Text(l, nil))
	}
	// A section that is (or holds) a synced block already stays that block.
	if len(mjml.SyncedKinds(mjml.WrapFragment(strings.TrimSpace(in.MJML)))) > 0 {
		return SyncedBlock{}, api.Invalid("body.mjml", mAlreadySynced.Text(l, nil))
	}
	kind := newNamedKind()
	block, err := checkBlock(l, kind, in.MJML)
	if err != nil {
		return SyncedBlock{}, err
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SyncedBlockInsert(ctx, store.SyncedBlockInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Kind: kind, Name: name, Mjml: block, UpdatedBy: actor(p), Now: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "synced_block.create", TargetType: "synced_block", TargetID: ws.ID.String() + ":" + kind,
			TargetName: name, After: map[string]any{"kind": kind, "name": name, "version": 1, "sizeBytes": len(block)}})
	})
	if err != nil {
		return SyncedBlock{}, err
	}
	by := ""
	if p != nil {
		by = p.Name
	}
	return SyncedBlock{Kind: kind, Name: name, MJML: block, Version: 1, UpdatedBy: by, UpdatedAt: now}, nil
}

// DeleteSynced deletes a named synced block (R4.A3). The drafts and
// scheduled emails holding it keep its content as ordinary sections, in a
// new version; emails in an approval, sending, paused or sent keep theirs
// (the composer offers to unlink a block that's gone). The Header and
// Footer stay: every kit has them.
func (s *Service) DeleteSynced(ctx context.Context, ws *store.Workspace, p *authz.Principal, kind string) (SyncedDeleted, error) {
	l := i18n.LangFrom(ctx)
	if kind == KindHeader || kind == KindFooter {
		return SyncedDeleted{}, api.Err(409, "synced_block_kept", mKeepHeaderFooter.Text(l, nil))
	}
	if !Named(kind) {
		return SyncedDeleted{}, api.NotFound("Synced block")
	}
	var name string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := q.SyncedBlockGet(ctx, store.SyncedBlockGetParams{WorkspaceID: ws.ID, Kind: kind})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Synced block")
		}
		if err != nil {
			return err
		}
		name = r.Name
		if _, err := q.SyncedBlockDelete(ctx, store.SyncedBlockDeleteParams{WorkspaceID: ws.ID, Kind: kind}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "synced_block.delete", TargetType: "synced_block", TargetID: ws.ID.String() + ":" + kind,
			TargetName: targetName(kind, r.Name), Before: map[string]any{"kind": kind, "name": r.Name, "version": r.Version}})
	})
	if err != nil {
		return SyncedDeleted{}, err
	}
	if s.Campaigns == nil {
		return SyncedDeleted{}, nil
	}
	applied, _, err := s.Campaigns.UnlinkSynced(ctx, ws, p, kind, campaigns.SyncedDeletedLabel(name))
	return SyncedDeleted{Applied: applied}, err
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
		// Named blocks (R4.A3) have no place in a starter.
		if markers[r.Kind] == "" {
			continue
		}
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
