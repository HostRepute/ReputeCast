package website

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/csvexport"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
	"github.com/hostrepute/reputecast/internal/website/importer"
	"github.com/hostrepute/reputecast/internal/website/render"
)

// ImageFetcher reads an imported post's images from the old site, through
// the outbound guard (feed.Fetcher: every address and redirect checked).
type ImageFetcher interface {
	Get(ctx context.Context, r feed.Request) (*feed.Response, error)
}

// Limits of a run.
const (
	// importImagesAtOnce is how many images of a post are read at once.
	importImagesAtOnce = 4
	// importBatch is how many staged posts a run reads at a time.
	importBatch = 20
	// importReportDays is how long an import's report is kept.
	importReportDays = 30
)

// ImportAnalyseArgs reads an uploaded export into staged posts.
type ImportAnalyseArgs struct {
	ImportID uuid.UUID `json:"import_id"`
}

// Kind implements river.JobArgs.
func (ImportAnalyseArgs) Kind() string { return "website.import_analyse" }

// InsertOpts runs once per import at a time.
func (ImportAnalyseArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

// ImportRunArgs imports an import's posts, from where it got to.
type ImportRunArgs struct {
	ImportID uuid.UUID `json:"import_id"`
}

// Kind implements river.JobArgs.
func (ImportRunArgs) Kind() string { return "website.import_run" }

// InsertOpts runs once per import at a time, retried a few times before
// the import stops.
func (ImportRunArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 5, UniqueOpts: river.UniqueOpts{ByArgs: true}}
}

type importAnalyseWorker struct {
	river.WorkerDefaults[ImportAnalyseArgs]
	s *Service
}

func (w *importAnalyseWorker) Work(ctx context.Context, j *river.Job[ImportAnalyseArgs]) error {
	return w.s.RunImportAnalyse(ctx, j.Args.ImportID)
}

func (w *importAnalyseWorker) Timeout(*river.Job[ImportAnalyseArgs]) time.Duration {
	return 10 * time.Minute
}

type importRunWorker struct {
	river.WorkerDefaults[ImportRunArgs]
	s *Service
}

func (w *importRunWorker) Work(ctx context.Context, j *river.Job[ImportRunArgs]) error {
	err := w.s.RunImport(ctx, j.Args.ImportID)
	if err != nil && j.Attempt >= j.MaxAttempts {
		// Out of retries: the import stops where it got to, and Try again
		// (start) carries on from there.
		return w.s.stopImport(ctx, j.Args.ImportID, err)
	}
	return err
}

// Timeout: a run is resumable (its cursor moves after every post), so a
// run cut short carries on when River retries it.
func (w *importRunWorker) Timeout(*river.Job[ImportRunArgs]) time.Duration { return time.Hour }

// PurgeImports drops reports past their 30 days and the files of ended
// imports the run didn't get to drop.
func (s *Service) PurgeImports(ctx context.Context) error {
	if err := s.Q.WebsiteImportsPurgeReports(ctx, s.now().AddDate(0, 0, -importReportDays)); err != nil {
		return err
	}
	return s.Q.WebsiteImportsPurgeFiles(ctx)
}

// RunImport imports an import's posts, after the last one it went through:
// images copied (or kept linking), each post made with its old address
// leading to it, published at its original date or kept as a draft. It is
// the website.import_run job. A canceled import stops after the post it is
// on. At the end it writes the report, drops the file, records one audit
// entry and tells whoever started it.
func (s *Service) RunImport(ctx context.Context, id uuid.UUID) error {
	imp, err := s.Q.WebsiteImportByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if imp.FinishedAt != nil || (imp.Status != ImportImporting && imp.Status != ImportCanceled) {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, imp.WorkspaceID)
	if err != nil {
		return err
	}
	opts := decodeOptions(imp.Options)
	st := decodeStats(imp.Stats)
	if st.Progress == nil {
		st.Progress = &WebsiteImportProgress{}
	}
	run := &importRun{s: s, ws: &ws, imp: imp, opts: opts, lang: s.defaultLanguage(ctx, ws.ID)}
	if run.tiers, err = s.tiers(ctx, s.Q, &ws); err != nil {
		return err
	}
	if opts.Images == ImportCopyImages || imp.Source == importer.Ghost {
		// A Ghost zip holds the site's images: posts whose export can't say
		// where the site was find theirs there.
		if data, err := s.Q.WebsiteImportFile(ctx, id); err == nil {
			run.file = data
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	for imp.Status == ImportImporting {
		after := uuid.Nil
		if st.Cursor != "" {
			if after, err = uuid.Parse(st.Cursor); err != nil {
				after = uuid.Nil
			}
		}
		rows, err := s.Q.WebsiteImportPostsToRun(ctx, store.WebsiteImportPostsToRunParams{ImportID: id, After: after, Lim: importBatch})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			if status, err := s.Q.WebsiteImportStatus(ctx, id); err != nil {
				return err
			} else if status != ImportImporting {
				imp.Status = status
				break
			}
			if err := run.post(ctx, row, st.Progress); err != nil {
				return err
			}
			st.Cursor = row.ID.String()
			sb, _ := json.Marshal(st)
			if err := s.Q.WebsiteImportSetStats(ctx, store.WebsiteImportSetStatsParams{ID: id, Stats: sb, At: s.now()}); err != nil {
				return err
			}
		}
	}
	status := ImportDone
	if imp.Status == ImportCanceled {
		status = ImportCanceled
	}
	return run.finish(ctx, status, st)
}

// stopImport ends a run that kept failing: the posts before stay, the
// report says where it got to, and why it stopped is kept for the screen.
func (s *Service) stopImport(ctx context.Context, id uuid.UUID, cause error) error {
	s.log().WarnContext(ctx, "post import stopped", "import", id, "err", cause)
	imp, err := s.Q.WebsiteImportByID(ctx, id)
	if err != nil {
		return err
	}
	if imp.Status != ImportImporting {
		return nil
	}
	ws, err := s.Q.GetWorkspace(ctx, imp.WorkspaceID)
	if err != nil {
		return err
	}
	st := decodeStats(imp.Stats)
	if st.Progress == nil {
		st.Progress = &WebsiteImportProgress{}
	}
	why := msgs.M("import.error.run").Keep(i18n.Args{"done": st.Progress.Done})
	st.ErrorText = &why
	run := &importRun{s: s, ws: &ws, imp: imp, opts: decodeOptions(imp.Options), lang: s.defaultLanguage(ctx, ws.ID)}
	return run.finish(ctx, ImportFailed, st)
}

// importRun is one run of an import.
type importRun struct {
	s     *Service
	ws    *store.Workspace
	imp   store.WebsiteImportByIDRow
	opts  WebsiteImportOptions
	lang  i18n.Lang
	tiers []gate.Tier
	// file is the export, when its images may be in it.
	file []byte
}

// importKept are the problems the analysis found that the run keeps (the
// converter's are found again, with the images as they were copied).
var importKept = map[string]bool{ProblemSlugTaken: true, ProblemExtraTags: true, ProblemAlreadyImported: true}

// post imports one staged post and counts what it did.
func (r *importRun) post(ctx context.Context, row store.WebsiteImportPost, pr *WebsiteImportProgress) error {
	s := r.s
	if row.Decision == ImportDecisionSkip {
		pr.Skipped++
		return nil
	}
	problems := []importer.Problem{}
	for _, p := range decodeProblems(row.Problems) {
		if importKept[p.Code] {
			problems = append(problems, p)
		}
	}
	images := r.images(ctx, row, pr)
	rewrite := func(src string) (string, bool) {
		if out, ok := images[src]; ok {
			return out, out != ""
		}
		if r.opts.Images == ImportLinkImages && servableRemote(src) {
			return src, true
		}
		return src, false
	}
	src, converted := importer.ConvertIn(r.lang, row.Html, rewrite)
	problems = append(problems, converted...)
	if err := validMJML(src); err != nil {
		problems = append(problems, importer.Problem{Code: ProblemConvertFailed, Detail: clip(err.Error(), 200)})
		pr.Done++
		pr.Failed++
		pb, _ := json.Marshal(problems)
		return s.Q.WebsiteImportPostDone(ctx, store.WebsiteImportPostDoneParams{ID: row.ID, PostID: row.PostID, Slug: row.Slug, Problems: pb})
	}
	cover, coverAlt := "", ""
	var seo importSEO
	_ = json.Unmarshal(row.Seo, &seo)
	if row.CoverUrl != "" {
		if c, ok := rewrite(row.CoverUrl); ok {
			cover, coverAlt = c, seo.CoverAlt
		}
	}
	if cover == "" {
		cover, coverAlt = firstImage(src)
	}
	byUser, byName := (*uuid.UUID)(nil), row.Author
	if v := r.opts.Author[row.Author]; v != "" && v != ImportKeepName {
		if uid, err := uuid.Parse(v); err == nil {
			byUser, byName = &uid, ""
		}
	}
	tiers := r.tierIDs(row)
	publish := row.Status == "published" && r.opts.Publish == ImportAsPublished
	outcome := ""
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		source := r.imp.Source
		cur, err := q.WebsiteImportedPost(ctx, store.WebsiteImportedPostParams{WorkspaceID: r.ws.ID, Source: &source, SourceID: &row.SourceID})
		switch {
		case err == nil && row.Decision == ImportDecisionUpdate:
			if err := q.WebsitePostImportUpdate(ctx, store.WebsitePostImportUpdateParams{Title: row.Title, Excerpt: row.Excerpt, CoverUrl: cover,
				CoverAlt: clip(coverAlt, 300), BylineUserID: byUser, BylineName: byName, Visibility: row.Visibility, Tiers: tiers, SeoTitle: seo.Title,
				SeoDescription: seo.Description, CanonicalUrl: seo.CanonicalURL, Mjml: src, UpdatedBy: r.imp.CreatedBy, At: s.now(),
				WorkspaceID: r.ws.ID, ID: cur.ID}); err != nil {
				return err
			}
			if cur, err = q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: r.ws.ID, ID: cur.ID}); err != nil {
				return err
			}
			if cur.Status == PostPublished {
				// The site shows the post as it is now.
				if err := s.snapshot(ctx, q, r.ws, cur); err != nil {
					return err
				}
			}
			outcome = OutcomeUpdated
		case err == nil:
			// Imported by an earlier run that stopped before its cursor
			// moved: the post is there already.
			outcome = OutcomeDraft
			if cur.Status == PostPublished {
				outcome = OutcomePublished
			}
		case errors.Is(err, pgx.ErrNoRows):
			slug := row.Slug
			if taken, err := q.WebsitePostSlugTaken(ctx, store.WebsitePostSlugTakenParams{WorkspaceID: r.ws.ID, Slug: slug, ID: uuid.Nil}); err != nil {
				return err
			} else if taken {
				slug = s.freeSlug(ctx, r.ws, uuid.Nil, slug)
				problems = append(problems, importer.Problem{Code: ProblemSlugTaken, Detail: slug})
			}
			id := newID()
			if err := q.WebsitePostImportInsert(ctx, store.WebsitePostImportInsertParams{ID: id, WorkspaceID: r.ws.ID, Title: row.Title, Slug: slug,
				Excerpt: row.Excerpt, CoverUrl: cover, CoverAlt: clip(coverAlt, 300), BylineUserID: byUser, BylineName: byName, Visibility: row.Visibility,
				Tiers: tiers, SeoTitle: seo.Title, SeoDescription: seo.Description, CanonicalUrl: seo.CanonicalURL, Mjml: src, ImportSource: &source,
				ImportSourceID: &row.SourceID, CreatedBy: r.imp.CreatedBy, At: s.now()}); err != nil {
				return err
			}
			if err := r.tags(ctx, q, id, row.Tags); err != nil {
				return err
			}
			if cur, err = q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: r.ws.ID, ID: id}); err != nil {
				return err
			}
			outcome = OutcomeDraft
			if publish {
				at := s.now()
				if row.PublishedAt != nil && row.PublishedAt.Before(at) {
					at = *row.PublishedAt
				}
				if err := s.publishNow(ctx, tx, r.ws, cur, r.imp.CreatedBy, map[string]any{"at": at, "imported": r.imp.ID.String()}, ""); err != nil {
					return err
				}
				outcome = OutcomePublished
			}
		default:
			return err
		}
		for _, p := range row.LegacyPaths {
			if p == "" {
				continue
			}
			if err := q.WebsiteLegacyPathInsert(ctx, store.WebsiteLegacyPathInsertParams{WorkspaceID: r.ws.ID, Path: p, PostID: cur.ID, At: s.now()}); err != nil {
				return err
			}
		}
		pb, _ := json.Marshal(problems)
		return q.WebsiteImportPostDone(ctx, store.WebsiteImportPostDoneParams{ID: row.ID, PostID: &cur.ID, Slug: cur.Slug, Problems: pb})
	})
	if err != nil {
		return err
	}
	pr.Done++
	switch outcome {
	case OutcomePublished:
		pr.Published++
	case OutcomeUpdated:
		pr.Updated++
	default:
		pr.Drafts++
	}
	return nil
}

// tierIDs are the tiers a paid post goes to: the mapped ones, or none
// (every paid tier) when the post names none or one maps to any.
func (r *importRun) tierIDs(row store.WebsiteImportPost) []uuid.UUID {
	out := []uuid.UUID{}
	if row.Visibility != gate.Paid {
		return out
	}
	for _, ref := range row.TierRefs {
		v := r.opts.Tiers[ref]
		if v == "" || v == ImportAnyTier {
			return []uuid.UUID{}
		}
		id, err := uuid.Parse(v)
		if err != nil {
			return []uuid.UUID{}
		}
		for _, t := range r.tiers {
			if t.ID == id && !containsID(out, id) {
				out = append(out, id)
			}
		}
	}
	return out
}

func containsID(ids []uuid.UUID, id uuid.UUID) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// tags gives a post its first MaxTags tags by name, making the ones the
// site hasn't got.
func (r *importRun) tags(ctx context.Context, q *store.Queries, post uuid.UUID, names []string) error {
	ids := []string{}
	for _, name := range names {
		if len(ids) == MaxTags {
			break
		}
		name = clip(strings.TrimSpace(name), 60)
		slug := slugify(name)
		if slug == "" {
			continue
		}
		t, err := q.WebsiteTagBySlug(ctx, store.WebsiteTagBySlugParams{WorkspaceID: r.ws.ID, Slug: slug})
		id := t.ID
		if errors.Is(err, pgx.ErrNoRows) {
			id = newID()
			err = q.WebsiteTagInsert(ctx, store.WebsiteTagInsertParams{ID: id, WorkspaceID: r.ws.ID, Name: name, Slug: slug, CreatedBy: r.imp.CreatedBy, At: r.s.now()})
		}
		if err != nil {
			return err
		}
		if s := id.String(); !containsString(ids, s) {
			ids = append(ids, s)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	_, err := r.s.setTags(ctx, q, r.ws, post, ids)
	return err
}

func containsString(vs []string, v string) bool {
	for _, x := range vs {
		if x == v {
			return true
		}
	}
	return false
}

// images copies a post's images (and cover) into the media library, four
// at a time: their new addresses by old address, "" for one that couldn't
// be copied. Kept linking, only images the export holds (a Ghost zip's,
// written as paths of the old site) and data: images are copied, as they
// have no address to link to.
func (r *importRun) images(ctx context.Context, row store.WebsiteImportPost, pr *WebsiteImportProgress) map[string]string {
	srcs := importer.Images(row.Html)
	if row.CoverUrl != "" && !containsString(srcs, row.CoverUrl) {
		srcs = append(srcs, row.CoverUrl)
	}
	out := map[string]string{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, importImagesAtOnce)
	for _, src := range srcs {
		if r.opts.Images == ImportLinkImages && servableRemote(src) {
			continue
		}
		wg.Add(1)
		slots <- struct{}{}
		go func(src string) {
			defer wg.Done()
			defer func() { <-slots }()
			addr, err := r.copyImage(ctx, src)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				r.s.log().InfoContext(ctx, "imported post image", "import", r.imp.ID, "src", clip(src, 200), "err", err)
				pr.ImagesFailed++
				out[src] = ""
				return
			}
			pr.ImagesCopied++
			out[src] = addr
		}(src)
	}
	wg.Wait()
	return out
}

// copyImage stores one image in the media library: read from the export,
// from a data: address, or from the old site through the guard.
func (r *importRun) copyImage(ctx context.Context, src string) (string, error) {
	var data []byte
	switch {
	case strings.HasPrefix(src, "data:"):
		b, err := decodeDataURL(src)
		if err != nil {
			return "", err
		}
		data = b
	case strings.HasPrefix(src, "/") && !strings.HasPrefix(src, "//"):
		u, err := url.Parse(src)
		if err != nil {
			return "", err
		}
		b, ok := importer.File(r.file, u.Path)
		if !ok {
			return "", fmt.Errorf("the export doesn't hold %s", u.Path)
		}
		data = b
	case servableRemote(src):
		f := r.s.imageFetcher()
		if f == nil {
			return "", errors.New("images can't be read from other sites here")
		}
		res, err := f.Get(ctx, feed.Request{URL: src, Accept: "image/png, image/jpeg, image/gif, image/webp, image/*;q=0.8"})
		if err != nil {
			return "", err
		}
		data = res.Body
	default:
		return "", errors.New("not an address an image can be read from")
	}
	prep, err := media.Prepare(ctx, data)
	if err != nil {
		return "", err
	}
	row, err := r.s.Q.InsertMediaAsset(ctx, prep.Params(r.ws.ID, imageName(src), r.imp.CreatedBy, r.s.now()))
	if err != nil {
		return "", err
	}
	return media.URL(row.ID, row.FileName), nil
}

// imageName is the file name an image's address ends with.
func imageName(src string) string {
	if strings.HasPrefix(src, "data:") {
		return "image"
	}
	if u, err := url.Parse(src); err == nil {
		if b := path.Base(u.Path); b != "." && b != "/" {
			return b
		}
	}
	return "image"
}

// decodeDataURL reads a base64 data: image.
func decodeDataURL(src string) ([]byte, error) {
	meta, payload, ok := strings.Cut(strings.TrimPrefix(src, "data:"), ",")
	if !ok || !strings.HasSuffix(meta, ";base64") || !strings.HasPrefix(meta, "image/") {
		return nil, errors.New("not a base64 image")
	}
	if len(payload) > media.MaxUploadBytes*4/3+4 {
		return nil, errors.New("image too large")
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(payload))
}

// servableRemote is an http(s) address on another host.
func servableRemote(src string) bool {
	u, err := url.Parse(src)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (s *Service) imageFetcher() ImageFetcher {
	if s.Images == nil {
		return nil
	}
	return s.Images()
}

// validMJML checks that a converted post parses and draws.
func validMJML(src string) error {
	if _, err := mjml.Parse(src); err != nil {
		return err
	}
	_, err := render.Render(src, render.Options{Links: map[string]string{}})
	return err
}

// finish ends a run: the report, the file dropped and the posts' bodies
// let go, one audit entry, and a notification for whoever started it.
func (r *importRun) finish(ctx context.Context, status string, st importStats) error {
	s := r.s
	report, err := r.report(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	sb, _ := json.Marshal(st)
	pr := st.Progress
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		errText := ""
		if st.ErrorText != nil {
			errText = clip(st.ErrorText.In(i18n.Default), 500)
		}
		if err := q.WebsiteImportFinish(ctx, store.WebsiteImportFinishParams{ID: r.imp.ID, Status: status, Stats: sb, Error: errText, ReportCsv: report, At: &now}); err != nil {
			return err
		}
		if err := q.WebsiteImportFileDelete(ctx, r.imp.ID); err != nil {
			return err
		}
		if err := q.WebsiteImportPostsDropHTML(ctx, r.imp.ID); err != nil {
			return err
		}
		actor := &audit.Actor{Type: authz.ActorSystem, Name: "ReputeCast"}
		if r.imp.CreatedBy != nil {
			name := ""
			if u, err := s.Q.GetUser(ctx, *r.imp.CreatedBy); err == nil {
				name = u.Name
			}
			actor = &audit.Actor{Type: authz.ActorUser, ID: r.imp.CreatedBy, Name: name}
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &r.ws.ID, Action: "website.posts_imported", TargetType: "website_import",
			TargetID: r.imp.ID.String(), TargetName: r.imp.FileName, Actor: actor,
			After: map[string]any{"status": status},
			Metadata: map[string]any{"source": r.imp.Source, "publish": r.opts.Publish, "images": r.opts.Images, "published": pr.Published, "drafts": pr.Drafts,
				"updated": pr.Updated, "failed": pr.Failed, "skipped": pr.Skipped, "images_copied": pr.ImagesCopied, "images_failed": pr.ImagesFailed}}); err != nil {
			return err
		}
		if r.imp.CreatedBy == nil {
			return nil
		}
		args := i18n.Args{"source": importer.Name(r.imp.Source), "published": pr.Published, "drafts": pr.Drafts + pr.Updated, "failed": pr.Failed}
		title := msgs.M("notify.importTitle").Keep(args)
		severity := "ok"
		switch {
		case status == ImportCanceled:
			title = msgs.M("notify.importCanceledTitle").Keep(args)
			severity = "info"
		case status == ImportFailed:
			title = msgs.M("notify.importStoppedTitle").Keep(args)
			severity = "bad"
		case pr.Failed > 0:
			severity = "warn"
		}
		n := notify.Notification{WorkspaceID: &r.ws.ID, Kind: KindPostImportDone, Severity: severity, TitleText: title,
			BodyText: msgs.M("notify.importBody").Keep(args), Link: "/w/" + r.ws.Slug + "/forms/website/import?import=" + r.imp.ID.String(),
			Data: map[string]any{"import": r.imp.ID.String()}}
		return notify.ToUser(ctx, tx, *r.imp.CreatedBy, n)
	})
}

// report is the run's CSV: each staged post's title, old address, outcome
// and problems, in the language of whoever started it.
func (r *importRun) report(ctx context.Context) ([]byte, error) {
	s := r.s
	l := i18n.Default
	if r.imp.CreatedBy != nil {
		l = applang.ForUserID(ctx, s.Q, *r.imp.CreatedBy)
	}
	imp, err := s.Q.WebsiteImportByID(ctx, r.imp.ID)
	if err != nil {
		return nil, err
	}
	st := decodeStats(imp.Stats)
	var buf bytes.Buffer
	w := csvexport.NewWriter(&buf)
	if err := w.Write([]string{msgs.M("import.report.title").Text(l, nil), msgs.M("import.report.sourceUrl").Text(l, nil),
		msgs.M("import.report.outcome").Text(l, nil), msgs.M("import.report.problems").Text(l, nil)}); err != nil {
		return nil, err
	}
	after := uuid.Nil
	for {
		rows, err := s.Q.WebsiteImportPostList(ctx, store.WebsiteImportPostListParams{ImportID: r.imp.ID, After: after, Lim: 500})
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			after = row.ID
			problems := decodeProblems(row.Problems)
			outcome := importOutcome(st, row.ID, row.Decision, row.PostID, row.PostStatus, problems)
			if outcome == "" {
				outcome = OutcomeSkipped
			}
			texts := make([]string, 0, len(problems))
			for _, p := range problems {
				texts = append(texts, problemLine(l, p))
			}
			if err := w.Write([]string{row.Title, row.SourceUrl, msgs.M("import.outcome."+outcome).Text(l, nil), strings.Join(texts, "; ")}); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
