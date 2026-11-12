package website

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
	"github.com/hostrepute/reputecast/internal/website/importer"
)

// Importing posts from Substack, Ghost and WordPress (M61, G-26): an
// export is uploaded (its format told from its content), read in the
// background into staged posts (website.import_analyse), reviewed —
// publish with the original dates or as drafts, copy images or keep
// linking, the old tiers and authors mapped, a decision per post — and
// imported in the background (website.import_run), resumable: a post
// remembers its platform's id, so a second run finds it. The old site's
// addresses lead to the imported posts (publicpages). Imports never send
// email; one audit entry (website.posts_imported) records the run.

// MaxImportFile is the largest export taken (100 MB).
const MaxImportFile = 100 << 20

// Import statuses.
const (
	ImportUploaded  = "uploaded"
	ImportAnalysing = "analysing"
	ImportReady     = "ready"
	ImportImporting = "importing"
	ImportDone      = "done"
	ImportFailed    = "failed"
	ImportCanceled  = "canceled"
)

// Choices.
const (
	ImportAsPublished = "as_published"
	ImportAsDrafts    = "as_drafts"
	ImportCopyImages  = "copy"
	ImportLinkImages  = "link"
	// ImportAnyTier maps an old tier to every paid tier; ImportKeepName
	// keeps an author's name as the byline.
	ImportAnyTier  = "any"
	ImportKeepName = "keep"
)

// Decisions on a staged post.
const (
	ImportDecisionImport = "import"
	ImportDecisionSkip   = "skip"
	ImportDecisionUpdate = "update"
)

// Problems the import adds to the converter's (importer.Problem codes).
const (
	ProblemSlugTaken       = "slug_taken"
	ProblemExtraTags       = "extra_tags"
	ProblemAlreadyImported = "already_imported"
	ProblemConvertFailed   = "convert_failed"
)

// KindPostImportDone tells whoever started an import that it ended.
const KindPostImportDone = notify.KindPostImportDone

// WebsiteImport is an import of posts: its file, how far it got, what the
// export holds, the choices and a page of its posts.
type WebsiteImport struct {
	ID       string `json:"id"`
	Source   string `json:"source" enum:"substack,ghost,wordpress"`
	Status   string `json:"status" enum:"uploaded,analysing,ready,importing,done,failed,canceled"`
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	// CreatedBy names who uploaded it.
	CreatedBy  string     `json:"createdBy"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	// Error says why it failed.
	Error  string              `json:"error,omitempty"`
	Counts WebsiteImportCounts `json:"counts"`
	// Progress is the run's, once it started.
	Progress *WebsiteImportProgress `json:"progress,omitempty"`
	Options  WebsiteImportOptions   `json:"options"`
	// Tiers and Authors are the old site's, as the posts name them, with
	// what each maps to.
	Tiers   []WebsiteImportTier   `json:"tiers"`
	Authors []WebsiteImportAuthor `json:"authors"`
	// PaidTiers and Team are what they can map to.
	PaidTiers []WebsiteImportChoice `json:"paidTiers"`
	Team      []WebsiteImportChoice `json:"team"`
	// Posts is a page of the staged posts (newest first, drafts last).
	Posts      []WebsiteImportPost `json:"posts"`
	NextCursor string              `json:"nextCursor,omitempty" doc:"Pass as ?cursor= for the next page of posts"`
	// ReportURL downloads the report once the import ended (kept 30 days).
	ReportURL string `json:"reportUrl,omitempty"`
}

// WebsiteImportSummary is an import in the list of a workspace's imports.
type WebsiteImportSummary struct {
	ID         string     `json:"id"`
	Source     string     `json:"source" enum:"substack,ghost,wordpress"`
	Status     string     `json:"status" enum:"uploaded,analysing,ready,importing,done,failed,canceled"`
	FileName   string     `json:"fileName"`
	FileSize   int64      `json:"fileSize"`
	CreatedBy  string     `json:"createdBy"`
	CreatedAt  time.Time  `json:"createdAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// WebsiteImportList is a workspace's latest 50 imports, newest first.
type WebsiteImportList struct {
	Items []WebsiteImportSummary `json:"items"`
}

// WebsiteImportCounts are the Review step's numbers.
type WebsiteImportCounts struct {
	Found           int64 `json:"found"`
	ToImport        int64 `json:"toImport"`
	ToUpdate        int64 `json:"toUpdate"`
	Skipped         int64 `json:"skipped"`
	Drafts          int64 `json:"drafts"`
	DraftsSkipped   int64 `json:"draftsSkipped"`
	AlreadyImported int64 `json:"alreadyImported"`
	Paid            int64 `json:"paid"`
	Members         int64 `json:"members"`
	Images          int64 `json:"images"`
	WithProblems    int64 `json:"withProblems"`
}

// WebsiteImportProgress is how far a run got.
type WebsiteImportProgress struct {
	// Done counts the posts gone through of Total (the ones to import or
	// update).
	Done         int64      `json:"done"`
	Total        int64      `json:"total"`
	Published    int64      `json:"published"`
	Drafts       int64      `json:"drafts"`
	Updated      int64      `json:"updated"`
	Failed       int64      `json:"failed"`
	Skipped      int64      `json:"skipped"`
	ImagesCopied int64      `json:"imagesCopied"`
	ImagesFailed int64      `json:"imagesFailed"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
}

// WebsiteImportOptions are an import's choices.
type WebsiteImportOptions struct {
	// Publish: as_published keeps the original dates and publishes;
	// as_drafts imports every post as a draft.
	Publish string `json:"publish" enum:"as_published,as_drafts"`
	// Images: copy them to ReputeCast, or keep linking to the old site.
	Images string `json:"images" enum:"copy,link"`
	// Tiers maps an old tier to a tier's id, or "any" (every paid tier).
	Tiers map[string]string `json:"tiers"`
	// Author maps an old author to a team member's id, or "keep" (the
	// name as the byline).
	Author map[string]string `json:"author"`
}

// WebsiteImportOptionsInput changes an import's choices and decisions
// before it starts.
type WebsiteImportOptionsInput struct {
	Publish   string                  `json:"publish,omitempty" enum:"as_published,as_drafts"`
	Images    string                  `json:"images,omitempty" enum:"copy,link"`
	Tiers     map[string]string       `json:"tiers,omitempty" doc:"Old tier → tier id or any"`
	Author    map[string]string       `json:"author,omitempty" doc:"Old author → team member id or keep"`
	Decisions []WebsiteImportDecision `json:"decisions,omitempty" maxItems:"10000"`
}

// WebsiteImportDecision is the decision on one staged post.
type WebsiteImportDecision struct {
	ID       string `json:"id" format:"uuid"`
	Decision string `json:"decision" enum:"import,skip,update" doc:"update: import again over the post imported before"`
}

// WebsiteImportTier is an old tier the posts name.
type WebsiteImportTier struct {
	Ref   string `json:"ref"`
	Posts int64  `json:"posts"`
	// TierID is a tier's id, or "any".
	TierID string `json:"tierId"`
}

// WebsiteImportAuthor is an old author the posts name.
type WebsiteImportAuthor struct {
	Name  string `json:"name"`
	Posts int64  `json:"posts"`
	// UserID is a team member's id, or "keep".
	UserID string `json:"userId"`
}

// WebsiteImportChoice is a tier or a team member a mapping can pick.
type WebsiteImportChoice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// WebsiteImportPost is a staged post.
type WebsiteImportPost struct {
	ID          string     `json:"id"`
	SourceID    string     `json:"sourceId"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	// Status is as it was on the old site.
	Status     string                 `json:"status" enum:"published,draft"`
	Visibility string                 `json:"visibility" enum:"public,members,paid"`
	TierRefs   []string               `json:"tierRefs"`
	Tags       []string               `json:"tags"`
	Author     string                 `json:"author"`
	Images     int                    `json:"images"`
	Problems   []WebsiteImportProblem `json:"problems"`
	Decision   string                 `json:"decision" enum:"import,skip,update"`
	// AlreadyImported: an earlier import made this post (PostID).
	AlreadyImported bool `json:"alreadyImported"`
	// PostID is the post it became, or the one imported before.
	PostID string `json:"postId,omitempty"`
	// SourceURL is its address on the old site (a path when the export
	// doesn't say where the site was).
	SourceURL string `json:"sourceUrl"`
	// Outcome is what the run did with it, once it ran.
	Outcome string `json:"outcome,omitempty" enum:"published,draft,updated,skipped,failed,"`
}

// WebsiteImportProblem is something about a post the import couldn't keep
// as it was, in the reader's words.
type WebsiteImportProblem struct {
	Code   string `json:"code" enum:"embed_link,image_alt,image_failed,shortcode,slug_taken,extra_tags,already_imported,convert_failed"`
	Detail string `json:"detail,omitempty"`
	// Text names the problem ("Slug taken"); Note says more when there is
	// more to say ("imported as how-we-test-email-rendering-2").
	Text string `json:"text"`
	Note string `json:"note,omitempty"`
}

// importStats is what website_imports.stats keeps: the run's progress, its
// cursor over the staged posts, and why it failed.
type importStats struct {
	Progress  *WebsiteImportProgress `json:"progress,omitempty"`
	Cursor    string                 `json:"cursor,omitempty"`
	ErrorText *i18n.Text             `json:"errorText,omitempty"`
}

// importSEO is website_import_posts.seo.
type importSEO struct {
	Title        string `json:"title,omitempty"`
	Description  string `json:"description,omitempty"`
	CanonicalURL string `json:"canonicalUrl,omitempty"`
	CoverAlt     string `json:"coverAlt,omitempty"`
	SourceURL    string `json:"sourceUrl,omitempty"`
}

// WebsiteImportPath is an import's path.
type WebsiteImportPath struct {
	api.WorkspaceInput
	Import string `path:"import" format:"uuid" doc:"The import's id"`
}

func (p WebsiteImportPath) id(ctx context.Context) (uuid.UUID, error) {
	return parseID(ctx, p.Import, "import")
}

type importUpload struct {
	api.WorkspaceInput
	FileName string `query:"fileName" maxLength:"200" doc:"The file's name, as it was chosen"`
	RawBody  []byte `contentType:"application/octet-stream" doc:"The export: Substack's zip, Ghost's JSON (or a zip of it) or WordPress's XML, up to 100 MB"`
}

type importGet struct {
	WebsiteImportPath
	Cursor string `query:"cursor" doc:"The posts' page: nextCursor of the previous one"`
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"500"`
}

type importBody struct {
	WebsiteImportPath
	Body WebsiteImportOptionsInput
}

// WebsiteImportReportFile is the report, as CSV.
type WebsiteImportReportFile struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

// registerImports adds the import's operations: website:publish, people
// only.
func registerImports(h huma.API, s *Service) {
	tags := []string{"Website"}
	base := "/workspaces/{workspace}/website/imports"
	access := api.WorkspaceHuman(authz.WebsitePublish)
	api.Post(h, api.Op{ID: "create-website-import", Path: base, Summary: "Upload a Substack, Ghost or WordPress export to import its posts",
		Description: "The format is told from the file's content. The posts are read in the background; nothing is imported until the import is started.",
		Tags:        tags, Access: access, Status: http.StatusCreated, MaxBodyBytes: MaxImportFile},
		func(ctx context.Context, in *importUpload) (*api.Body[WebsiteImport], error) {
			out, err := s.CreateImport(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.FileName, in.RawBody)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-website-imports", Path: base, Summary: "A workspace's latest imports of posts, newest first", Tags: tags, Access: access},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[WebsiteImportList], error) {
			out, err := s.Imports(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-website-import", Path: base + "/{import}", Summary: "An import of posts: how far it got, what the export holds, the choices and its posts",
		Tags: tags, Access: access},
		func(ctx context.Context, in *importGet) (*api.Body[WebsiteImport], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.Import(ctx, api.WorkspaceFrom(ctx), id, in.Cursor, in.Limit)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-website-import", Path: base + "/{import}", Summary: "Change an import's choices, mappings and decisions before it starts",
		Tags: tags, Access: access},
		func(ctx context.Context, in *importBody) (*api.Body[WebsiteImport], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateImport(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-website-import", Path: base + "/{import}/start", Summary: "Start importing the posts in the background",
		Description: "Posts kept as published go live at their original dates, without emails, webhooks or an audit entry each; one entry records the import.",
		Tags:        tags, Access: access},
		func(ctx context.Context, in *WebsiteImportPath) (*api.Body[WebsiteImport], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.StartImport(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "cancel-website-import", Path: base + "/{import}/cancel", Summary: "Cancel an import, or stop it after the post it is on",
		Description: "Posts imported so far stay.", Tags: tags, Access: access},
		func(ctx context.Context, in *WebsiteImportPath) (*api.Body[WebsiteImport], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.CancelImport(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "download-website-import-report", Path: base + "/{import}/report.csv", Summary: "An ended import's report: each post's title, old address, outcome and problems",
		Tags: tags, Access: access},
		func(ctx context.Context, in *WebsiteImportPath) (*WebsiteImportReportFile, error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			return s.ImportReport(ctx, api.WorkspaceFrom(ctx), id)
		})
}

// importSiteOn refuses imports while the site is off: their posts would
// have nowhere to go.
func (s *Service) importSiteOn(ctx context.Context, q *store.Queries, ws *store.Workspace) error {
	c, err := s.load(ctx, q, ws)
	if err != nil {
		return err
	}
	if !c.On() {
		return api.Conflict(tr(ctx, "site.off", nil))
	}
	return nil
}

// CreateImport keeps an uploaded export and reads its posts in the
// background.
func (s *Service) CreateImport(ctx context.Context, ws *store.Workspace, p *authz.Principal, fileName string, data []byte) (WebsiteImport, error) {
	if err := s.importSiteOn(ctx, s.Q, ws); err != nil {
		return WebsiteImport{}, err
	}
	if len(data) == 0 {
		return WebsiteImport{}, api.Invalid("body", tr(ctx, "import.empty", nil))
	}
	if len(data) > MaxImportFile {
		return WebsiteImport{}, api.Invalid("body", tr(ctx, "import.tooLarge", i18n.Args{"mb": MaxImportFile >> 20}))
	}
	source, err := importer.Detect(data)
	if err != nil {
		return WebsiteImport{}, api.Invalid("body", tr(ctx, "import.notExport", nil))
	}
	name := strings.TrimSpace(path.Base(strings.ReplaceAll(fileName, "\\", "/")))
	if name == "." || name == "/" {
		name = ""
	}
	id, now := newID(), s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.WebsiteImportInsert(ctx, store.WebsiteImportInsertParams{ID: id, WorkspaceID: ws.ID, Source: source, FileName: clip(name, 200),
			FileSize: int64(len(data)), CreatedBy: userID(p), At: now}); err != nil {
			return err
		}
		if err := q.WebsiteImportFileInsert(ctx, store.WebsiteImportFileInsertParams{ImportID: id, Data: data}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "website.import_uploaded", TargetType: "website_import", TargetID: id.String(),
			TargetName: name, After: map[string]any{"source": source, "file_name": name, "file_size": len(data)}})
	})
	if err != nil {
		return WebsiteImport{}, err
	}
	if j := s.jobs(); j != nil {
		if _, err := j.Insert(ctx, ImportAnalyseArgs{ImportID: id}, nil); err != nil {
			return WebsiteImport{}, err
		}
	} else if err := s.RunImportAnalyse(ctx, id); err != nil {
		return WebsiteImport{}, err
	}
	return s.Import(ctx, ws, id, "", 50)
}

// Imports are a workspace's latest imports.
func (s *Service) Imports(ctx context.Context, ws *store.Workspace) (WebsiteImportList, error) {
	rows, err := s.Q.WebsiteImportList(ctx, ws.ID)
	if err != nil {
		return WebsiteImportList{}, err
	}
	out := WebsiteImportList{Items: make([]WebsiteImportSummary, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, WebsiteImportSummary{ID: r.ID.String(), Source: r.Source, Status: r.Status, FileName: r.FileName, FileSize: r.FileSize,
			CreatedBy: r.CreatedByName, CreatedAt: r.CreatedAt, FinishedAt: r.FinishedAt})
	}
	return out, nil
}

// Import is an import with a page of its posts.
func (s *Service) Import(ctx context.Context, ws *store.Workspace, id uuid.UUID, cursor string, limit int) (WebsiteImport, error) {
	r, err := s.Q.WebsiteImportGet(ctx, store.WebsiteImportGetParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return WebsiteImport{}, notFound(ctx, "import")
	}
	if err != nil {
		return WebsiteImport{}, err
	}
	l := lang(ctx)
	opts := decodeOptions(r.Options)
	st := decodeStats(r.Stats)
	out := WebsiteImport{ID: r.ID.String(), Source: r.Source, Status: r.Status, FileName: r.FileName, FileSize: r.FileSize, CreatedBy: r.CreatedByName,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, FinishedAt: r.FinishedAt, Options: opts, Progress: st.Progress,
		Tiers: []WebsiteImportTier{}, Authors: []WebsiteImportAuthor{}, PaidTiers: []WebsiteImportChoice{}, Team: []WebsiteImportChoice{}, Posts: []WebsiteImportPost{}}
	if r.Status == ImportFailed {
		out.Error = r.Error
		if st.ErrorText != nil {
			if t := st.ErrorText.In(l); t != "" {
				out.Error = t
			}
		}
	}
	if r.HasReport {
		out.ReportURL = api.Version + "/workspaces/" + ws.Slug + "/website/imports/" + r.ID.String() + "/report.csv"
	}
	counts, err := s.Q.WebsiteImportCounts(ctx, id)
	if err != nil {
		return WebsiteImport{}, err
	}
	out.Counts = WebsiteImportCounts{Found: counts.Found, ToImport: counts.ToImport, ToUpdate: counts.ToUpdate, Skipped: counts.ToSkip, Drafts: counts.Drafts,
		DraftsSkipped: counts.DraftsSkipped, AlreadyImported: counts.AlreadyImported, Paid: counts.Paid, Members: counts.Members, Images: counts.Images,
		WithProblems: counts.WithProblems}
	refs, err := s.Q.WebsiteImportTierRefs(ctx, id)
	if err != nil {
		return WebsiteImport{}, err
	}
	for _, t := range refs {
		out.Tiers = append(out.Tiers, WebsiteImportTier{Ref: t.Ref, Posts: t.Posts, TierID: cmp.Or(opts.Tiers[t.Ref], ImportAnyTier)})
	}
	authors, err := s.Q.WebsiteImportAuthors(ctx, id)
	if err != nil {
		return WebsiteImport{}, err
	}
	for _, a := range authors {
		out.Authors = append(out.Authors, WebsiteImportAuthor{Name: a.Author, Posts: a.Posts, UserID: cmp.Or(opts.Author[a.Author], ImportKeepName)})
	}
	tiers, err := s.tiers(ctx, s.Q, ws)
	if err != nil {
		return WebsiteImport{}, err
	}
	for _, t := range tiers {
		out.PaidTiers = append(out.PaidTiers, WebsiteImportChoice{ID: t.ID.String(), Name: t.Name})
	}
	members, err := s.Q.ApprovalMembers(ctx, ws.ID)
	if err != nil {
		return WebsiteImport{}, err
	}
	for _, m := range members {
		out.Team = append(out.Team, WebsiteImportChoice{ID: m.ID.String(), Name: m.Name})
	}
	after := uuid.Nil
	if cursor != "" {
		if after, err = uuid.Parse(cursor); err != nil {
			return WebsiteImport{}, api.Invalid("query.cursor", tr(ctx, "import.badCursor", nil))
		}
	}
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.Q.WebsiteImportPostList(ctx, store.WebsiteImportPostListParams{ImportID: id, After: after, Lim: int32(limit + 1)})
	if err != nil {
		return WebsiteImport{}, err
	}
	if len(rows) > limit {
		rows = rows[:limit]
		out.NextCursor = rows[len(rows)-1].ID.String()
	}
	ran := r.Status == ImportDone || r.Status == ImportFailed || r.Status == ImportCanceled || r.Status == ImportImporting
	for _, row := range rows {
		problems := decodeProblems(row.Problems)
		item := WebsiteImportPost{ID: row.ID.String(), SourceID: row.SourceID, Title: row.Title, Slug: row.Slug, PublishedAt: row.PublishedAt, Status: row.Status,
			Visibility: row.Visibility, TierRefs: row.TierRefs, Tags: row.Tags, Author: row.Author, Images: int(row.Images), Decision: row.Decision,
			SourceURL: row.SourceUrl, Problems: problemViews(l, problems), AlreadyImported: hasProblem(problems, ProblemAlreadyImported)}
		if row.PostID != nil {
			item.PostID = row.PostID.String()
		}
		if ran {
			item.Outcome = importOutcome(st, row.ID, row.Decision, row.PostID, row.PostStatus, problems)
		}
		out.Posts = append(out.Posts, item)
	}
	return out, nil
}

// UpdateImport changes an import's choices, mappings and decisions while
// it waits to start.
func (s *Service) UpdateImport(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in WebsiteImportOptionsInput) (WebsiteImport, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsiteImportLock(ctx, store.WebsiteImportLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(ctx, "import")
		}
		if err != nil {
			return err
		}
		if cur.Status != ImportReady {
			return api.Err(http.StatusConflict, "import_not_ready", tr(ctx, "import.notEditable", nil))
		}
		if err := s.importSiteOn(ctx, q, ws); err != nil {
			return err
		}
		before := decodeOptions(cur.Options)
		next := before
		next.Tiers, next.Author = maps.Clone(before.Tiers), maps.Clone(before.Author)
		if in.Publish != "" {
			next.Publish = in.Publish
		}
		if in.Images != "" {
			next.Images = in.Images
		}
		if len(in.Tiers) > 0 {
			tiers, err := s.tiers(ctx, q, ws)
			if err != nil {
				return err
			}
			for ref, v := range in.Tiers {
				if v != ImportAnyTier {
					tid, err := uuid.Parse(v)
					if err != nil || !slices.ContainsFunc(tiers, func(t gate.Tier) bool { return t.ID == tid }) {
						return api.Invalid("body.tiers", tr(ctx, "import.unknownTier", i18n.Args{"tier": ref}))
					}
				}
				next.Tiers[ref] = v
			}
		}
		if len(in.Author) > 0 {
			members, err := q.ApprovalMembers(ctx, ws.ID)
			if err != nil {
				return err
			}
			for name, v := range in.Author {
				if v != ImportKeepName {
					uid, err := uuid.Parse(v)
					if err != nil || !slices.ContainsFunc(members, func(m store.ApprovalMembersRow) bool { return m.ID == uid }) {
						return api.Invalid("body.author", tr(ctx, "import.unknownMember", i18n.Args{"name": name}))
					}
				}
				next.Author[name] = v
			}
		}
		changed := 0
		if len(in.Decisions) > 0 {
			ids := make([]uuid.UUID, 0, len(in.Decisions))
			for _, d := range in.Decisions {
				pid, err := uuid.Parse(d.ID)
				if err != nil {
					return api.Invalid("body.decisions", tr(ctx, "import.unknownPost", nil))
				}
				ids = append(ids, pid)
			}
			refs, err := q.WebsiteImportPostRefs(ctx, store.WebsiteImportPostRefsParams{ImportID: id, Ids: ids})
			if err != nil {
				return err
			}
			byID := map[uuid.UUID]store.WebsiteImportPostRefsRow{}
			for _, r := range refs {
				byID[r.ID] = r
			}
			for i, d := range in.Decisions {
				r, ok := byID[ids[i]]
				if !ok {
					return api.Invalid("body.decisions", tr(ctx, "import.unknownPost", nil))
				}
				if d.Decision == ImportDecisionUpdate && r.PostID == nil {
					return api.Invalid("body.decisions", tr(ctx, "import.updateNeedsPost", nil))
				}
				if d.Decision == ImportDecisionImport && r.PostID != nil {
					// Imported before: only Update imports it again.
					return api.Invalid("body.decisions", tr(ctx, "import.importedBefore", nil))
				}
				if d.Decision == r.Decision {
					continue
				}
				if err := q.WebsiteImportPostSetDecision(ctx, store.WebsiteImportPostSetDecisionParams{ImportID: id, ID: r.ID, Decision: d.Decision}); err != nil {
					return err
				}
				changed++
			}
		}
		b, _ := json.Marshal(next)
		if err := q.WebsiteImportSetOptions(ctx, store.WebsiteImportSetOptionsParams{ID: id, Options: b, At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "website.import_updated", TargetType: "website_import", TargetID: id.String(),
			TargetName: cur.FileName, Before: optionsAudit(before), After: optionsAudit(next), Metadata: map[string]any{"decisions_changed": changed}})
	})
	if err != nil {
		return WebsiteImport{}, err
	}
	return s.Import(ctx, ws, id, "", 50)
}

// StartImport starts a ready import in the background. Keeping posts
// published needs no approval: where publishing posts needs one, the
// import goes as drafts or not at all.
func (s *Service) StartImport(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (WebsiteImport, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsiteImportLock(ctx, store.WebsiteImportLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(ctx, "import")
		}
		if err != nil {
			return err
		}
		st := decodeStats(cur.Stats)
		if cur.Status == ImportFailed && st.Progress != nil {
			// Try again: the run carries on after the last post it went
			// through.
			if err := s.importSiteOn(ctx, q, ws); err != nil {
				return err
			}
			st.ErrorText = nil
			sb, _ := json.Marshal(st)
			if n, err := q.WebsiteImportResume(ctx, store.WebsiteImportResumeParams{ID: id, Stats: sb, At: s.now()}); err != nil {
				return err
			} else if n == 0 {
				return api.Err(http.StatusConflict, "import_not_ready", tr(ctx, "import.started", nil))
			}
			return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "website.import_updated", TargetType: "website_import", TargetID: id.String(),
				TargetName: cur.FileName, Before: map[string]any{"status": ImportFailed}, After: map[string]any{"status": ImportImporting, "resumed_after": st.Progress.Done}})
		}
		if cur.Status != ImportReady {
			key := "import.notReady"
			if cur.Status != ImportUploaded && cur.Status != ImportAnalysing {
				key = "import.started"
			}
			return api.Err(http.StatusConflict, "import_not_ready", tr(ctx, key, nil))
		}
		if err := s.importSiteOn(ctx, q, ws); err != nil {
			return err
		}
		opts := decodeOptions(cur.Options)
		if opts.Publish == ImportAsPublished {
			if a := s.approvals(); a != nil {
				d, err := a.PostDecision(ctx, ws, p, PostApprovalRequest{Kind: KindWeb, When: WhenNow, Visibility: gate.Public})
				if err != nil {
					return err
				}
				if d.NeedsApproval {
					return api.Err(http.StatusConflict, "approval_required", tr(ctx, "import.needsApproval", nil)).WithMeta("rule", d.Rule, "approvers", d.Approvers)
				}
			}
		}
		counts, err := q.WebsiteImportCounts(ctx, id)
		if err != nil {
			return err
		}
		if counts.ToImport+counts.ToUpdate == 0 {
			return api.Err(http.StatusConflict, "nothing_to_import", tr(ctx, "import.nothing", nil))
		}
		now := s.now()
		st.Progress = &WebsiteImportProgress{Total: counts.ToImport + counts.ToUpdate, StartedAt: &now}
		st.Cursor = ""
		sb, _ := json.Marshal(st)
		if err := q.WebsiteImportSetStats(ctx, store.WebsiteImportSetStatsParams{ID: id, Stats: sb, At: now}); err != nil {
			return err
		}
		if n, err := q.WebsiteImportMove(ctx, store.WebsiteImportMoveParams{ID: id, Status: ImportImporting, At: now, FromStatuses: []string{ImportReady}}); err != nil {
			return err
		} else if n == 0 {
			return api.Err(http.StatusConflict, "import_not_ready", tr(ctx, "import.started", nil))
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "website.import_updated", TargetType: "website_import", TargetID: id.String(),
			TargetName: cur.FileName, Before: map[string]any{"status": ImportReady}, After: map[string]any{"status": ImportImporting, "publish": opts.Publish,
				"images": opts.Images, "to_import": counts.ToImport, "to_update": counts.ToUpdate}})
	})
	if err != nil {
		return WebsiteImport{}, err
	}
	if j := s.jobs(); j != nil {
		if _, err := j.Insert(ctx, ImportRunArgs{ImportID: id}, nil); err != nil {
			return WebsiteImport{}, err
		}
	} else if err := s.RunImport(ctx, id); err != nil {
		return WebsiteImport{}, err
	}
	return s.Import(ctx, ws, id, "", 50)
}

// CancelImport drops an import that hasn't started, with its file, or
// stops a running one after the post it is on (the run writes its report).
func (s *Service) CancelImport(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (WebsiteImport, error) {
	running := false
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.WebsiteImportLock(ctx, store.WebsiteImportLockParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(ctx, "import")
		}
		if err != nil {
			return err
		}
		now := s.now()
		switch cur.Status {
		case ImportUploaded, ImportAnalysing, ImportReady:
			if err := q.WebsiteImportFinish(ctx, store.WebsiteImportFinishParams{ID: id, Status: ImportCanceled, Stats: cur.Stats, At: &now}); err != nil {
				return err
			}
			if err := q.WebsiteImportFileDelete(ctx, id); err != nil {
				return err
			}
		case ImportImporting:
			running = true
			if _, err := q.WebsiteImportMove(ctx, store.WebsiteImportMoveParams{ID: id, Status: ImportCanceled, At: now, FromStatuses: []string{ImportImporting}}); err != nil {
				return err
			}
		default:
			return api.Err(http.StatusConflict, "import_ended", tr(ctx, "import.ended", nil))
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "website.import_canceled", TargetType: "website_import", TargetID: id.String(),
			TargetName: cur.FileName, Before: map[string]any{"status": cur.Status}, After: map[string]any{"status": ImportCanceled}})
	})
	if err != nil {
		return WebsiteImport{}, err
	}
	if running {
		// The run stops after its post and writes the report; queued again
		// in case it isn't running any more.
		if j := s.jobs(); j != nil {
			if _, err := j.Insert(ctx, ImportRunArgs{ImportID: id}, nil); err != nil {
				return WebsiteImport{}, err
			}
		} else if err := s.RunImport(ctx, id); err != nil {
			return WebsiteImport{}, err
		}
	}
	return s.Import(ctx, ws, id, "", 50)
}

// ImportReport is an ended import's report.
func (s *Service) ImportReport(ctx context.Context, ws *store.Workspace, id uuid.UUID) (*WebsiteImportReportFile, error) {
	r, err := s.Q.WebsiteImportReport(ctx, store.WebsiteImportReportParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, notFound(ctx, "importReport")
	}
	if err != nil {
		return nil, err
	}
	day := s.now()
	if r.FinishedAt != nil {
		day = *r.FinishedAt
	}
	name := fmt.Sprintf("post-import-%s-%s.csv", ws.Slug, day.In(location(ws)).Format("2006-01-02"))
	return &WebsiteImportReportFile{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, CacheControl: "no-store",
		Body: r.ReportCsv}, nil
}

// decodeOptions reads an import's choices, with the defaults.
func decodeOptions(raw json.RawMessage) WebsiteImportOptions {
	o := WebsiteImportOptions{}
	_ = json.Unmarshal(raw, &o)
	if o.Publish != ImportAsDrafts {
		o.Publish = ImportAsPublished
	}
	if o.Images != ImportLinkImages {
		o.Images = ImportCopyImages
	}
	if o.Tiers == nil {
		o.Tiers = map[string]string{}
	}
	if o.Author == nil {
		o.Author = map[string]string{}
	}
	return o
}

func decodeStats(raw json.RawMessage) importStats {
	var st importStats
	_ = json.Unmarshal(raw, &st)
	return st
}

func decodeProblems(raw json.RawMessage) []importer.Problem {
	out := []importer.Problem{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func hasProblem(ps []importer.Problem, code string) bool {
	return slices.ContainsFunc(ps, func(p importer.Problem) bool { return p.Code == code })
}

// problemViews are a post's problems in a reader's words.
func problemViews(l i18n.Lang, ps []importer.Problem) []WebsiteImportProblem {
	out := make([]WebsiteImportProblem, 0, len(ps))
	for _, p := range ps {
		out = append(out, WebsiteImportProblem{Code: p.Code, Detail: p.Detail, Text: problemText(l, p), Note: problemNote(l, p)})
	}
	return out
}

// problemText names a problem in a reader's words.
func problemText(l i18n.Lang, p importer.Problem) string {
	m, err := msgs.Lookup("import.problem." + p.Code)
	if err != nil {
		return p.Code
	}
	return m.Text(l, nil)
}

// problemNote is what more a problem has to say ("imported as …"), when
// it has a detail worth reading.
func problemNote(l i18n.Lang, p importer.Problem) string {
	if p.Detail == "" {
		return ""
	}
	m, err := msgs.Lookup("import.note." + p.Code)
	if err != nil {
		return ""
	}
	return m.Text(l, i18n.Args{"detail": p.Detail})
}

// problemLine is a problem as the report writes it: its name, and its
// note after a colon.
func problemLine(l i18n.Lang, p importer.Problem) string {
	if n := problemNote(l, p); n != "" {
		return problemText(l, p) + ": " + n
	}
	return problemText(l, p)
}

// optionsAudit is what the audit log keeps of an import's choices.
func optionsAudit(o WebsiteImportOptions) map[string]any {
	return map[string]any{"publish": o.Publish, "images": o.Images, "tiers": o.Tiers, "author": o.Author}
}

// Outcomes of a staged post's run.
const (
	OutcomePublished = "published"
	OutcomeDraft     = "draft"
	OutcomeUpdated   = "updated"
	OutcomeSkipped   = "skipped"
	OutcomeFailed    = "failed"
)

// importOutcome is what a run did with a staged post: "" while the run
// hasn't reached it. The run's cursor is the last post it went through.
func importOutcome(st importStats, id uuid.UUID, decision string, postID *uuid.UUID, postStatus string, problems []importer.Problem) string {
	switch {
	case decision == ImportDecisionSkip:
		return OutcomeSkipped
	case st.Cursor == "" || id.String() > st.Cursor:
		return ""
	case postID == nil || hasProblem(problems, ProblemConvertFailed):
		return OutcomeFailed
	case decision == ImportDecisionUpdate:
		return OutcomeUpdated
	case postStatus == PostPublished:
		return OutcomePublished
	}
	return OutcomeDraft
}

// LegacyPost is the slug of the post an old address of the site leads to
// (key as importer.LegacyKey makes it): "" when none does.
func (s *Service) LegacyPost(ctx context.Context, ws *store.Workspace, key string) (string, error) {
	if key == "" {
		return "", nil
	}
	slug, err := s.Q.WebsiteLegacyPathPost(ctx, store.WebsiteLegacyPathPostParams{WorkspaceID: ws.ID, Path: key})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return slug, err
}
