package languages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
)

// Owner kinds: what a language version belongs to.
const (
	OwnerCampaign         = "campaign"
	OwnerAutomationEmail  = "automation_email"
	OwnerTransactional    = "transactional_template"
	keepTranslationSaves  = 300
	codeVersionConflict   = "version_conflict"
	maxTranslationVersion = 1 << 30
)

// Owner is an email with language versions: a campaign, a version of an
// automation email or of a transactional template.
type Owner struct {
	Kind string
	ID   uuid.UUID
	// Version is the automation email or template version; 0 for campaigns.
	Version     int32
	WorkspaceID uuid.UUID
	// Name names it in the audit log.
	Name string
}

func (o Owner) key() (string, uuid.UUID, int32) { return o.Kind, o.ID, o.Version }

// auditPrefix is the audit action group of the owner's language versions.
func (o Owner) auditPrefix() string {
	switch o.Kind {
	case OwnerAutomationEmail:
		return "automation_email.language."
	case OwnerTransactional:
		return "transactional_template.language."
	}
	return "campaign.language."
}

// Source is an email's source content as it is now.
type Source struct {
	Subject   string
	Preheader string
	MJML      string
	HTML      string
	// Version is the source's content version: blocks are compared again
	// when it changes.
	Version int
	// By and At are who saved that version and when.
	By string
	At time.Time
}

// sourceState is the source with its keyed blocks, in step with Source.
type sourceState struct {
	Language      string
	Blocks        []Block
	Keys          []SourceBlock
	SubjectHash   string
	PreheaderHash string
	Version       int
	// Changed lists the keys whose text changed in this sync.
	Changed []string
}

// syncSource brings the stored source blocks in step with the source,
// creating them the first time. Call it in a transaction.
func (s *Service) syncSource(ctx context.Context, q *store.Queries, o Owner, src Source, defaultLang string) (sourceState, error) {
	kind, id, ver := o.key()
	blocks := Blocks(src.MJML, src.HTML)
	row, err := q.ContentSourceForUpdate(ctx, store.ContentSourceForUpdateParams{OwnerKind: kind, OwnerID: id, OwnerVersion: ver})
	st := sourceState{Blocks: blocks, SubjectHash: HashText(src.Subject), PreheaderHash: HashText(src.Preheader), Version: src.Version}
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		st.Language = defaultLang
		st.Keys = make([]SourceBlock, len(blocks))
		for i, b := range blocks {
			st.Keys[i] = SourceBlock{Key: fmt.Sprintf("b%d", i+1), Raw: b.Raw, Hash: b.Hash, Text: b.Text}
		}
		return st, s.saveSource(ctx, q, o, st, len(blocks)+1)
	case err != nil:
		return st, err
	}
	st.Language = row.Language
	var stored []SourceBlock
	_ = json.Unmarshal(row.Blocks, &stored)
	if int(row.Version) == src.Version && len(stored) == len(blocks) {
		st.Keys = stored
		// The texts summaries may read differently after an upgrade; keep
		// the stored keys and refresh what describes the blocks.
		for i := range st.Keys {
			st.Keys[i].Text = blocks[i].Text
		}
		return st, nil
	}
	keys, next, changed := SyncSource(stored, blocks, int(row.NextKey), BlockChange{Version: src.Version, By: src.By, At: src.At})
	st.Keys, st.Changed = keys, changed
	return st, s.saveSource(ctx, q, o, st, next)
}

func (s *Service) saveSource(ctx context.Context, q *store.Queries, o Owner, st sourceState, next int) error {
	b, _ := json.Marshal(st.Keys)
	kind, id, ver := o.key()
	return q.ContentSourceUpsert(ctx, store.ContentSourceUpsertParams{WorkspaceID: o.WorkspaceID, OwnerKind: kind, OwnerID: id, OwnerVersion: ver, Language: st.Language,
		Blocks: b, SubjectHash: st.SubjectHash, PreheaderHash: st.PreheaderHash, Version: int32(st.Version), NextKey: int32(next), At: s.now()})
}

// view is a language version compared with its source.
type view struct {
	row      store.ContentTranslation
	by       string // translated by
	updater  string // last saved by
	blocks   []Block
	links    Links
	rows     []Row
	progress Progress
}

func (s *Service) viewOf(st sourceState, t store.ContentTranslation) view {
	v := view{row: t, blocks: Blocks(t.Mjml, t.Html)}
	_ = json.Unmarshal(t.Links, &v.links)
	v.links = padLinks(v.links, v.blocks)
	v.rows, v.progress = Compare(st.Blocks, st.Keys, v.blocks, v.links, t.Status != storedTranslated, st.SubjectHash, st.PreheaderHash)
	return v
}

// padLinks gives every block a link entry, so the indexes line up.
func padLinks(l Links, blocks []Block) Links {
	for len(l.Blocks) < len(blocks) {
		b := blocks[len(l.Blocks)]
		l.Blocks = append(l.Blocks, BlockLink{Own: b.Hash, OwnRaw: b.Raw})
	}
	if len(l.Blocks) > len(blocks) {
		l.Blocks = l.Blocks[:len(blocks)]
	}
	return l
}

func translationOf(r store.ContentTranslationsRow) store.ContentTranslation {
	return store.ContentTranslation{ID: r.ID, WorkspaceID: r.WorkspaceID, OwnerKind: r.OwnerKind, OwnerID: r.OwnerID, OwnerVersion: r.OwnerVersion, Language: r.Language,
		Status: r.Status, Held: r.Held, Subject: r.Subject, Preheader: r.Preheader, Mjml: r.Mjml, Html: r.Html, PlainText: r.PlainText, AbTest: r.AbTest, Links: r.Links,
		Version: r.Version, ContentHash: r.ContentHash, TranslatedFrom: r.TranslatedFrom, TranslatedBy: r.TranslatedBy, TranslatedAt: r.TranslatedAt, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedBy: r.UpdatedBy, UpdatedAt: r.UpdatedAt, AutosavedAt: r.AutosavedAt}
}

// versions loads an owner's language versions with their state.
func (s *Service) versions(ctx context.Context, q *store.Queries, o Owner, st sourceState) ([]view, error) {
	kind, id, ver := o.key()
	rows, err := q.ContentTranslations(ctx, store.ContentTranslationsParams{OwnerKind: kind, OwnerID: id, OwnerVersion: ver})
	if err != nil {
		return nil, err
	}
	out := make([]view, 0, len(rows))
	for _, r := range rows {
		v := s.viewOf(st, translationOf(r))
		v.by, v.updater = r.TranslatedByName, r.UpdatedByName
		out = append(out, v)
	}
	return out, nil
}

// LanguageStart is what a new language version starts from.
const (
	StartCopy   = "copy"
	StartLayout = "layout"
)

// emptyTexts empties the text of every text block (Add language › "The
// layout only"); buttons, images and their links stay.
func emptyTexts(src string) string {
	root, err := mjml.Parse(src)
	if err != nil {
		return src
	}
	type span struct{ start, end int }
	var cut []span
	root.Walk(func(n *mjml.Node, _ []int) {
		if n.Tag == "mj-text" && strings.TrimSpace(n.Inner) != "" {
			// Inner sits right after the opening tag; find it within the node.
			i := strings.Index(src[n.Start:n.End], n.Inner)
			if i >= 0 {
				cut = append(cut, span{n.Start + i, n.Start + i + len(n.Inner)})
			}
		}
	})
	var b strings.Builder
	last := 0
	for _, c := range cut {
		if c.start < last {
			continue
		}
		b.WriteString(src[last:c.start])
		last = c.end
	}
	b.WriteString(src[last:])
	return b.String()
}

// addVersion creates a Draft language version.
func (s *Service) addVersion(ctx context.Context, q *store.Queries, o Owner, p *authz.Principal, st sourceState, src Source, lang, start string, ab json.RawMessage) (store.ContentTranslation, error) {
	mj, html, subject, preheader := src.MJML, src.HTML, src.Subject, src.Preheader
	if start == StartLayout {
		mj = emptyTexts(src.MJML)
		if strings.TrimSpace(mj) != "" {
			res := mjml.Compile(mjml.SetHeadMeta(mj, "", ""))
			if len(res.Errors) == 0 {
				html = res.HTML
			}
		}
		subject, preheader = "", ""
	}
	blocks := Blocks(mj, html)
	links := FreshLinks(st.Blocks, st.Keys, blocks, src.Subject, src.Preheader, st.Version)
	lb, _ := json.Marshal(links)
	now := s.now()
	t := store.ContentTranslation{ID: uuid.Must(uuid.NewV7()), WorkspaceID: o.WorkspaceID, OwnerKind: o.Kind, OwnerID: o.ID, OwnerVersion: o.Version, Language: lang,
		Status: storedDraft, Subject: subject, Preheader: preheader, Mjml: mj, Html: html, PlainText: contentcheck.PlainText(html), AbTest: ab, Links: lb, Version: 1,
		TranslatedFrom: int32(st.Version), CreatedBy: actorID(p), CreatedAt: now, UpdatedBy: actorID(p), UpdatedAt: now}
	t.ContentHash = contentHash(t.Subject, t.Preheader, t.Mjml, t.Html, t.AbTest)
	if err := q.ContentTranslationInsert(ctx, store.ContentTranslationInsertParams{ID: t.ID, WorkspaceID: t.WorkspaceID, OwnerKind: t.OwnerKind, OwnerID: t.OwnerID,
		OwnerVersion: t.OwnerVersion, Language: t.Language, Status: t.Status, Held: false, Subject: t.Subject, Preheader: t.Preheader, Mjml: t.Mjml, Html: t.Html,
		PlainText: t.PlainText, AbTest: t.AbTest, Links: t.Links, Version: 1, ContentHash: t.ContentHash, TranslatedFrom: t.TranslatedFrom, CreatedBy: t.CreatedBy,
		CreatedAt: now, UpdatedBy: t.UpdatedBy, UpdatedAt: now, AutosavedAt: &now}); err != nil {
		return t, err
	}
	label := "copied from " + NameOf(st.Language)
	if start == StartLayout {
		label = "layout of " + NameOf(st.Language)
	}
	if err := q.ContentTranslationVersionInsert(ctx, store.ContentTranslationVersionInsertParams{TranslationID: t.ID, Version: 1, Subject: t.Subject, Preheader: t.Preheader,
		Mjml: t.Mjml, Html: t.Html, PlainText: t.PlainText, AbTest: t.AbTest, Links: t.Links, SavedBy: actorID(p), SavedAt: now, Label: label, Source: "create",
		SizeBytes: int32(len(t.Html))}); err != nil {
		return t, err
	}
	return t, nil
}

// contentHash fingerprints a language version like campaigns.ContentHash.
func contentHash(subject, preheader, mj, html string, ab json.RawMessage) string {
	return HashText(subject + "\x00" + preheader + "\x00" + mj + "\x00" + html + "\x00" + string(ab))
}

// SaveInput is a save of a language version.
type SaveInput struct {
	BaseVersion int
	Subject     string
	Preheader   string
	MJML        string
	HTML        string
	ABTest      json.RawMessage
	Label       string
	Source      string
	Force       bool
	Minify      bool
}

// Saved is the result of a language version save.
type Saved struct {
	Version     int
	SavedAt     time.Time
	ContentHash string
	SizeBytes   int
	PlainText   string
	HTML        string
}

// saveVersion stores a new version of a language version's content, with
// the per-language conflict check of autosave.
func (s *Service) saveVersion(ctx context.Context, q *store.Queries, o Owner, p *authz.Principal, lang string, in SaveInput) (Saved, store.ContentTranslation, error) {
	kind, id, ver := o.key()
	cur, err := q.ContentTranslationForUpdate(ctx, store.ContentTranslationForUpdateParams{OwnerKind: kind, OwnerID: id, OwnerVersion: ver, Language: lang})
	if errors.Is(err, pgx.ErrNoRows) {
		return Saved{}, cur, api.NotFound(NameOf(lang) + " version")
	}
	if err != nil {
		return Saved{}, cur, err
	}
	if int(cur.Version) != in.BaseVersion && !in.Force {
		theirs, mine := mjml.ChangedSections(baseMJML(ctx, q, cur.ID, in.BaseVersion), cur.Mjml), []string{}
		by := ""
		if cur.UpdatedBy != nil {
			if u, err := q.GetUser(ctx, *cur.UpdatedBy); err == nil {
				by = u.Name
			}
		}
		at := cur.UpdatedAt
		if cur.AutosavedAt != nil {
			at = *cur.AutosavedAt
		}
		who := by
		if who == "" {
			who = "Someone"
		}
		return Saved{}, cur, api.Err(http.StatusConflict, codeVersionConflict, fmt.Sprintf("%s saved %s at %s while you were editing it.", who, NameOf(lang), at.Format("15:04"))).
			WithMeta("version", cur.Version, "savedBy", by, "savedAt", at, "language", lang, "theirChanges", nonNil(theirs), "yourChanges", mine)
	}
	src, html := in.MJML, in.HTML
	if html == "" && strings.TrimSpace(src) != "" {
		src = mjml.SetHeadMeta(src, in.Subject, in.Preheader)
		res := mjml.Compile(src)
		if len(res.Errors) > 0 {
			return Saved{}, cur, api.Invalid("body.mjml", "The MJML doesn't compile: "+res.Errors[0].Message)
		}
		html = res.HTML
	}
	if in.Minify && html != "" {
		html = contentcheck.Minify(html)
	}
	var old Links
	_ = json.Unmarshal(cur.Links, &old)
	blocks := Blocks(src, html)
	links := Relink(padLinks(old, Blocks(cur.Mjml, cur.Html)), blocks)
	lb, _ := json.Marshal(links)
	plain := contentcheck.PlainText(html)
	hash := contentHash(in.Subject, in.Preheader, src, html, in.ABTest)
	version := cur.Version + 1
	if version >= maxTranslationVersion {
		return Saved{}, cur, api.Conflict("This language version has too many saved versions.")
	}
	now := s.now()
	if err := q.ContentTranslationSave(ctx, store.ContentTranslationSaveParams{ID: cur.ID, Subject: in.Subject, Preheader: in.Preheader, Mjml: src, Html: html,
		PlainText: plain, AbTest: in.ABTest, Links: lb, Version: version, ContentHash: hash, UpdatedBy: actorID(p), At: now}); err != nil {
		return Saved{}, cur, err
	}
	source := in.Source
	if source == "" {
		source = "autosave"
	}
	if err := q.ContentTranslationVersionInsert(ctx, store.ContentTranslationVersionInsertParams{TranslationID: cur.ID, Version: version, Subject: in.Subject,
		Preheader: in.Preheader, Mjml: src, Html: html, PlainText: plain, AbTest: in.ABTest, Links: lb, SavedBy: actorID(p), SavedAt: now,
		Label: strings.TrimSpace(in.Label), Source: source, SizeBytes: int32(len(html))}); err != nil {
		return Saved{}, cur, err
	}
	if _, err := q.ContentTranslationPruneVersions(ctx, store.ContentTranslationPruneVersionsParams{TranslationID: cur.ID, Keep: keepTranslationSaves}); err != nil {
		return Saved{}, cur, err
	}
	before := map[string]any{"version": cur.Version}
	after := map[string]any{"version": version, "sizeBytes": len(html)}
	if cur.Subject != in.Subject {
		before["subject"], after["subject"] = cur.Subject, in.Subject
	}
	if cur.Preheader != in.Preheader {
		before["preheader"], after["preheader"] = cur.Preheader, in.Preheader
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &o.WorkspaceID, Action: o.auditPrefix() + "save", TargetType: o.Kind, TargetID: o.ID.String(),
		TargetName: o.Name, Before: before, After: after, Metadata: map[string]any{"language": lang, "label": in.Label, "source": source, "contentHash": hash}}); err != nil {
		return Saved{}, cur, err
	}
	cur.Subject, cur.Preheader, cur.Mjml, cur.Html, cur.PlainText, cur.AbTest, cur.Links, cur.Version, cur.ContentHash = in.Subject, in.Preheader, src, html, plain, in.ABTest, lb, version, hash
	return Saved{Version: int(version), SavedAt: now, ContentHash: hash, SizeBytes: len(html), PlainText: plain, HTML: html}, cur, nil
}

func baseMJML(ctx context.Context, q *store.Queries, id uuid.UUID, version int) string {
	v, err := q.ContentTranslationVersion(ctx, store.ContentTranslationVersionParams{TranslationID: id, Version: int32(version)})
	if err != nil {
		return ""
	}
	return v.Mjml
}

// lockVersion loads one language version for a change.
func lockVersion(ctx context.Context, q *store.Queries, o Owner, lang string) (store.ContentTranslation, error) {
	kind, id, ver := o.key()
	t, err := q.ContentTranslationForUpdate(ctx, store.ContentTranslationForUpdateParams{OwnerKind: kind, OwnerID: id, OwnerVersion: ver, Language: lang})
	if errors.Is(err, pgx.ErrNoRows) {
		return t, api.NotFound(NameOf(lang) + " version")
	}
	return t, err
}

// mark marks a version translated (a Draft) or up to date (all blocks, or
// only some), pointing its links at the current source.
func (s *Service) mark(ctx context.Context, q *store.Queries, o Owner, p *authz.Principal, st sourceState, src Source, lang string, blocks []string, action string) error {
	t, err := lockVersion(ctx, q, o, lang)
	if err != nil {
		return err
	}
	v := s.viewOf(st, t)
	links := MarkUpToDate(v.links, st.Blocks, st.Keys, src.Subject, src.Preheader, st.Version, blocks)
	lb, _ := json.Marshal(links)
	at := s.now()
	from := t.TranslatedFrom
	if len(blocks) == 0 {
		from = int32(st.Version)
	}
	if err := q.ContentTranslationMark(ctx, store.ContentTranslationMarkParams{ID: t.ID, Status: storedTranslated, Links: lb, TranslatedFrom: from, TranslatedBy: actorID(p), At: &at}); err != nil {
		return err
	}
	meta := map[string]any{"language": lang, "sourceVersion": st.Version}
	if len(blocks) > 0 {
		meta["blocks"] = blocks
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &o.WorkspaceID, Action: o.auditPrefix() + action, TargetType: o.Kind, TargetID: o.ID.String(), TargetName: o.Name,
		Before: map[string]any{"status": statusWord(v.progress.Status), "outdatedBlocks": v.progress.Attention}, After: map[string]any{"status": "up_to_date"}, Metadata: meta})
}

func statusWord(s string) string { return s }

// removeVersion deletes a language version.
func (s *Service) removeVersion(ctx context.Context, q *store.Queries, o Owner, lang string) error {
	kind, id, ver := o.key()
	n, err := q.ContentTranslationDelete(ctx, store.ContentTranslationDeleteParams{OwnerKind: kind, OwnerID: id, OwnerVersion: ver, Language: lang})
	if err != nil {
		return err
	}
	if n == 0 {
		return api.NotFound(NameOf(lang) + " version")
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &o.WorkspaceID, Action: o.auditPrefix() + "remove", TargetType: o.Kind, TargetID: o.ID.String(), TargetName: o.Name,
		Before: map[string]any{"language": lang}, Metadata: map[string]any{"language": lang}})
}

// hold sends the fallback to a language's readers instead (or stops doing so).
func (s *Service) hold(ctx context.Context, q *store.Queries, o Owner, lang string, held bool) error {
	t, err := lockVersion(ctx, q, o, lang)
	if err != nil {
		return err
	}
	if t.Held == held {
		return nil
	}
	if err := q.ContentTranslationSetHeld(ctx, store.ContentTranslationSetHeldParams{ID: t.ID, Held: held, At: s.now()}); err != nil {
		return err
	}
	action := "hold"
	if !held {
		action = "release"
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &o.WorkspaceID, Action: o.auditPrefix() + action, TargetType: o.Kind, TargetID: o.ID.String(), TargetName: o.Name,
		Before: map[string]any{"held": t.Held}, After: map[string]any{"held": held}, Metadata: map[string]any{"language": lang}})
}

// replaceInBlock swaps a link or image address in the given blocks of a
// document: exact attribute values (href="…", src="…") only.
func replaceInBlock(src string, blocks []Block, indexes []int, from, to string) (string, int) {
	if from == "" || from == to {
		return src, 0
	}
	type span struct{ start, end int }
	spans := []span{}
	for _, i := range indexes {
		if i >= 0 && i < len(blocks) {
			spans = append(spans, span{blocks[i].Start, blocks[i].End})
		}
	}
	n := 0
	var b strings.Builder
	last := 0
	for _, sp := range spans {
		if sp.start < last || sp.end > len(src) {
			continue
		}
		b.WriteString(src[last:sp.start])
		chunk := src[sp.start:sp.end]
		for _, attr := range []string{"href", "src"} {
			for _, q := range []string{`"`, `'`} {
				old := attr + "=" + q + from + q
				if c := strings.Count(chunk, old); c > 0 {
					n += c
					chunk = strings.ReplaceAll(chunk, old, attr+"="+q+to+q)
				}
			}
		}
		b.WriteString(chunk)
		last = sp.end
	}
	b.WriteString(src[last:])
	return b.String(), n
}
