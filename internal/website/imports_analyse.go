package website

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/importer"
)

// RunImportAnalyse reads an uploaded export into staged posts, with the
// default decisions — published posts import, drafts and posts imported
// before are skipped, a slug another post holds gets "-2" — and the
// default choices: original dates, images copied, old tiers and authors
// matched by name. It is the website.import_analyse job.
func (s *Service) RunImportAnalyse(ctx context.Context, id uuid.UUID) error {
	imp, err := s.Q.WebsiteImportByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if imp.Status != ImportUploaded && imp.Status != ImportAnalysing {
		return nil
	}
	if _, err := s.Q.WebsiteImportMove(ctx, store.WebsiteImportMoveParams{ID: id, Status: ImportAnalysing, At: s.now(),
		FromStatuses: []string{ImportUploaded, ImportAnalysing}}); err != nil {
		return err
	}
	ws, err := s.Q.GetWorkspace(ctx, imp.WorkspaceID)
	if err != nil {
		return err
	}
	data, err := s.Q.WebsiteImportFile(ctx, id)
	if err != nil {
		return s.failImport(ctx, id, err, msgs.M("import.error.unreadable").Keep(i18n.Args{"source": importer.Name(imp.Source)}))
	}
	posts, err := importer.Parse(imp.Source, data)
	switch {
	case errors.Is(err, importer.ErrTooManyPosts):
		return s.failImport(ctx, id, err, msgs.M("import.error.tooMany").Keep(i18n.Args{"count": importer.MaxPosts}))
	case err != nil:
		return s.failImport(ctx, id, err, msgs.M("import.error.unreadable").Keep(i18n.Args{"source": importer.Name(imp.Source)}))
	}
	// Newest first, drafts last: the staged posts' ids keep that order.
	slices.SortStableFunc(posts, func(a, b importer.Post) int {
		switch {
		case a.Draft != b.Draft:
			if a.Draft {
				return 1
			}
			return -1
		case a.PublishedAt != nil && b.PublishedAt != nil:
			return b.PublishedAt.Compare(*a.PublishedAt)
		case a.PublishedAt != nil:
			return -1
		case b.PublishedAt != nil:
			return 1
		}
		return 0
	})
	ids := make([]string, 0, len(posts))
	for _, p := range posts {
		ids = append(ids, p.SourceID)
	}
	before, err := s.Q.WebsiteImportedPosts(ctx, store.WebsiteImportedPostsParams{WorkspaceID: ws.ID, Source: &imp.Source, SourceIds: ids})
	if err != nil {
		return err
	}
	imported := map[string]uuid.UUID{}
	for _, r := range before {
		imported[r.SourceID] = r.ID
	}
	rows := make([]store.WebsiteImportPostsInsertParams, 0, len(posts))
	taken := map[string]bool{}
	for _, p := range posts {
		row, err := s.stagePost(ctx, &ws, id, p, imported, taken)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	opts := s.guessOptions(ctx, &ws, posts)
	ob, _ := json.Marshal(opts)
	sb, _ := json.Marshal(importStats{})
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WebsiteImportPostsClear(ctx, id); err != nil {
			return err
		}
		if len(rows) > 0 {
			if _, err := q.WebsiteImportPostsInsert(ctx, rows); err != nil {
				return err
			}
		}
		return q.WebsiteImportReady(ctx, store.WebsiteImportReadyParams{ID: id, Options: ob, Stats: sb, At: s.now()})
	})
}

// stagePost is a post of the export as the Review step shows it.
func (s *Service) stagePost(ctx context.Context, ws *store.Workspace, importID uuid.UUID, p importer.Post, imported map[string]uuid.UUID,
	taken map[string]bool) (store.WebsiteImportPostsInsertParams, error) {
	_, problems := importer.Convert(p.HTML, nil)
	excerpt := strings.Join(strings.Fields(p.Excerpt), " ")
	if excerpt == "" {
		excerpt = importer.Summary(p.HTML)
	}
	excerpt = forms.Excerpt(excerpt, 199)
	seo, _ := json.Marshal(importSEO{Title: clip(p.SEOTitle, 300), Description: clip(p.SEODescription, 500), CanonicalURL: safeAddress(p.CanonicalURL),
		CoverAlt: clip(p.CoverAlt, 300), SourceURL: clip(p.SourceURL, 1000)})
	status := "published"
	if p.Draft {
		status = "draft"
	}
	title := clip(strings.TrimSpace(p.Title), 200)
	if title == "" {
		title = tr(ctx, "post.untitled", nil)
	}
	row := store.WebsiteImportPostsInsertParams{ID: newID(), ImportID: importID, SourceID: p.SourceID, Title: title, PublishedAt: p.PublishedAt,
		Status: status, Visibility: p.Visibility, TierRefs: nonNil(p.TierRefs), Tags: nonNil(p.Tags), Author: clip(p.Author, 120), Excerpt: excerpt,
		CoverUrl: clip(p.CoverURL, 1000), Seo: seo, LegacyPaths: nonNil(p.LegacyPaths), Html: p.HTML, Images: int32(len(importer.Images(p.HTML))),
		Decision: ImportDecisionImport}
	if p.Draft {
		row.Decision = ImportDecisionSkip
	}
	if len(p.Tags) > MaxTags {
		problems = append(problems, importer.Problem{Code: ProblemExtraTags, Detail: strings.Join(p.Tags[MaxTags:], ", ")})
	}
	if postID, ok := imported[p.SourceID]; ok {
		// Imported before: skipped, unless Update imports it again.
		row.PostID, row.Decision = &postID, ImportDecisionSkip
		problems = append([]importer.Problem{{Code: ProblemAlreadyImported}}, problems...)
		if cur, err := s.Q.WebsitePostGet(ctx, store.WebsitePostGetParams{WorkspaceID: ws.ID, ID: postID}); err == nil {
			row.Slug = cur.WebsitePost.Slug
		}
	} else {
		slug, moved := s.importSlug(ctx, ws, p, row.Title, taken)
		row.Slug = slug
		if moved {
			problems = append(problems, importer.Problem{Code: ProblemSlugTaken, Detail: slug})
		}
	}
	taken[row.Slug] = true
	row.Problems, _ = json.Marshal(problems)
	return row, nil
}

// importSlug is the address a post gets: its old one when it's free, else
// the first free "<slug>-2", "-3"… moved reports the change.
func (s *Service) importSlug(ctx context.Context, ws *store.Workspace, p importer.Post, title string, staged map[string]bool) (string, bool) {
	base := p.Slug
	if !slugRe.MatchString(base) || len(base) > 120 {
		base = slugify(cmp.Or(base, title))
	}
	if base == "" {
		base = slugify(title)
	}
	if base == "" {
		base = "post"
	}
	free := func(slug string) bool {
		if staged[slug] {
			return false
		}
		taken, err := s.Q.WebsitePostSlugTaken(ctx, store.WebsitePostSlugTakenParams{WorkspaceID: ws.ID, Slug: slug, ID: uuid.Nil})
		return err == nil && !taken
	}
	if free(base) {
		return base, false
	}
	for i := 2; i < 1000; i++ {
		suffix := fmt.Sprintf("-%d", i)
		slug := strings.Trim(base[:min(len(base), 120-len(suffix))], "-") + suffix
		if free(slug) {
			return slug, true
		}
	}
	return base + "-" + newID().String()[:8], true
}

// guessOptions are an import's first choices: the original dates, images
// copied, old tiers and authors matched with the workspace's by name.
func (s *Service) guessOptions(ctx context.Context, ws *store.Workspace, posts []importer.Post) WebsiteImportOptions {
	opts := WebsiteImportOptions{Publish: ImportAsPublished, Images: ImportCopyImages, Tiers: map[string]string{}, Author: map[string]string{}}
	tiers, _ := s.tiers(ctx, s.Q, ws)
	members, _ := s.Q.ApprovalMembers(ctx, ws.ID)
	for _, p := range posts {
		for _, ref := range p.TierRefs {
			if _, ok := opts.Tiers[ref]; ok {
				continue
			}
			opts.Tiers[ref] = ImportAnyTier
			for _, t := range tiers {
				if sameName(t.Name, ref) {
					opts.Tiers[ref] = t.ID.String()
					break
				}
			}
		}
		if p.Author == "" {
			continue
		}
		if _, ok := opts.Author[p.Author]; ok {
			continue
		}
		opts.Author[p.Author] = ImportKeepName
		for _, m := range members {
			if strings.EqualFold(strings.TrimSpace(m.Name), strings.TrimSpace(p.Author)) {
				opts.Author[p.Author] = m.ID.String()
				break
			}
		}
	}
	return opts
}

// sameName compares names of tiers as people write them: case, spaces and
// a plural "s" don't count ("Founding members" is "Founding member").
func sameName(a, b string) bool {
	norm := func(s string) string {
		words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
		for i, w := range words {
			if len(w) > 3 {
				words[i] = strings.TrimSuffix(w, "s")
			}
		}
		return strings.Join(words, " ")
	}
	return norm(a) != "" && norm(a) == norm(b)
}

// safeAddress keeps an http(s) address, else nothing.
func safeAddress(u string) string {
	u = strings.TrimSpace(u)
	if (strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://")) && len(u) <= 1000 {
		return u
	}
	return ""
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// failImport ends an import that couldn't go on: why, in English and as a
// message for its readers.
func (s *Service) failImport(ctx context.Context, id uuid.UUID, cause error, why i18n.Text) error {
	s.log().WarnContext(ctx, "post import failed", "import", id, "err", cause)
	imp, err := s.Q.WebsiteImportByID(ctx, id)
	if err != nil {
		return err
	}
	st := decodeStats(imp.Stats)
	st.ErrorText = &why
	sb, _ := json.Marshal(st)
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WebsiteImportFinish(ctx, store.WebsiteImportFinishParams{ID: id, Status: ImportFailed, Stats: sb, Error: clip(why.In(i18n.Default), 500), At: &now}); err != nil {
			return err
		}
		return q.WebsiteImportFileDelete(ctx, id)
	})
}
