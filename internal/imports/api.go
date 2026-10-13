package imports

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// ImportFile is a CSV download.
type ImportFile struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

func csvFile(b []byte, name string) *ImportFile {
	return &ImportFile{ContentType: "text/csv; charset=utf-8", ContentDisposition: fmt.Sprintf(`attachment; filename="%s"`, name), CacheControl: "no-store", Body: b}
}

type ImportPathInput struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid"`
}

type importBody[T any] struct {
	ImportPathInput
	Body T
}

type createBody struct {
	api.WorkspaceInput
	Body ImportCreateInput
}

type uploadBody struct {
	ImportPathInput
	FileName string `query:"fileName" maxLength:"200" doc:"The file's name"`
	RawBody  []byte `contentType:"application/octet-stream"`
}

type reportPath struct {
	ImportPathInput
	Kind string `path:"kind" enum:"report,skipped,invalid,unreadable"`
}

// Register adds the importer operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Imports"}
	base := "/workspaces/{workspace}/imports"
	imp := api.Workspace(authz.SubscribersImport)
	id := func(p ImportPathInput) uuid.UUID { return uuid.MustParse(p.ID) }

	api.Get(h, api.Op{ID: "list-imports", Path: base, Summary: "Import history and the draft the wizard resumes", Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[ImportList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-background-jobs", Path: "/workspaces/{workspace}/jobs", Summary: "Imports running in the background and those that just ended (jobs tray)",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[ImportJobList], error) {
			out, err := s.BackgroundJobs(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-import", Path: base, Summary: "Start an import from a source", Tags: tags, Access: imp, Status: http.StatusCreated},
		func(ctx context.Context, in *createBody) (*api.Body[ImportDetail], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-import", Path: base + "/{id}", Summary: "An import with its connection, progress, log and summary", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Body[ImportDetail], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), id(*in))
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-import", Path: base + "/{id}", Summary: "Change a draft's source, options or step", Tags: tags, Access: imp},
		func(ctx context.Context, in *importBody[ImportUpdateInput]) (*api.Body[ImportDetail], error) {
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), id(in.ImportPathInput), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "discard-import", Path: base + "/{id}", Summary: "Discard a draft import", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Empty, error) {
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), id(*in))
		})
	api.Post(h, api.Op{ID: "connect-import", Path: base + "/{id}/connect", Summary: "Check an API key and list the source's audiences",
		Description: "The key is stored encrypted and never returned. Nothing is written to the source.", Tags: tags, Access: imp},
		func(ctx context.Context, in *importBody[ImportConnectInput]) (*api.Body[ImportDetail], error) {
			out, err := s.Connect(ctx, api.WorkspaceFrom(ctx), id(in.ImportPathInput), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "upload-import-file", Path: base + "/{id}/file", Summary: "Upload a CSV or XLSX file (first row: headers)", Tags: tags, Access: imp, MaxBodyBytes: 100 << 20},
		func(ctx context.Context, in *uploadBody) (*api.Body[ImportDetail], error) {
			data := append([]byte(nil), in.RawBody...)
			out, err := s.Upload(ctx, api.WorkspaceFrom(ctx), id(in.ImportPathInput), in.FileName, data)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "skip-import-bad-rows", Path: base + "/{id}/skip-bad-rows", Summary: "Leave out file rows with the wrong number of columns", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Body[ImportDetail], error) {
			out, err := s.SkipBadRows(ctx, api.WorkspaceFrom(ctx), id(*in))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "stage-import", Path: base + "/{id}/stage", Summary: "Read the selected lists' rows from the source", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Body[ImportDetail], error) {
			out, err := s.RequestStage(ctx, api.WorkspaceFrom(ctx), id(*in))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resume-import", Path: base + "/{id}/resume", Summary: "Resume an import paused by the source's rate limit", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Body[ImportDetail], error) {
			out, err := s.Resume(ctx, api.WorkspaceFrom(ctx), id(*in))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-import-fields", Path: base + "/{id}/fields", Summary: "Source fields with sample values and their mapping", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Body[ImportFieldMapping], error) {
			out, err := s.Fields(ctx, api.WorkspaceFrom(ctx), id(*in))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-import-fields", Path: base + "/{id}/fields", Summary: "Save the field mapping", Tags: tags, Access: imp},
		func(ctx context.Context, in *importBody[ImportSaveFieldsInput]) (*api.Body[ImportFieldMapping], error) {
			out, err := s.SaveFields(ctx, api.WorkspaceFrom(ctx), id(in.ImportPathInput), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-import-consent", Path: base + "/{id}/consent", Summary: "Contacts per source status and what each becomes", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Body[ImportConsentMapping], error) {
			out, err := s.Consent(ctx, api.WorkspaceFrom(ctx), id(*in))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-import-consent", Path: base + "/{id}/consent", Summary: "Save the consent mapping", Tags: tags, Access: imp},
		func(ctx context.Context, in *importBody[ImportSaveConsentInput]) (*api.Body[ImportConsentMapping], error) {
			out, err := s.SaveConsent(ctx, api.WorkspaceFrom(ctx), id(in.ImportPathInput), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-import", Path: base + "/{id}/preview", Summary: "Dry run: duplicates, invalid and risky addresses", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Body[ImportPreview], error) {
			out, err := s.Preview(ctx, api.WorkspaceFrom(ctx), id(*in))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-import", Path: base + "/{id}/start", Summary: "Import the staged rows", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Body[ImportDetail], error) {
			out, err := s.Start(ctx, api.WorkspaceFrom(ctx), id(*in))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "cancel-import", Path: base + "/{id}/cancel", Summary: "Stop an import (rows already imported stay)", Tags: tags, Access: imp},
		func(ctx context.Context, in *ImportPathInput) (*api.Body[ImportDetail], error) {
			out, err := s.Cancel(ctx, api.WorkspaceFrom(ctx), id(*in))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "download-import-report", Path: base + "/{id}/files/{kind}", Summary: "Download the full report, skipped rows, invalid rows or unreadable file rows (logged)", Tags: tags, Access: imp},
		func(ctx context.Context, in *reportPath) (*ImportFile, error) {
			ws := api.WorkspaceFrom(ctx)
			b, name, err := s.ReportCSV(ctx, ws, id(in.ImportPathInput), in.Kind)
			if err != nil {
				return nil, err
			}
			if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "import.report_download", TargetType: "import", TargetID: in.ID, TargetName: name}); err != nil {
				return nil, err
			}
			return csvFile(b, name), nil
		})
}

// ReportCSV returns one of the import's CSV files.
func (s *Service) ReportCSV(ctx context.Context, ws *store.Workspace, id uuid.UUID, kind string) ([]byte, string, error) {
	switch kind {
	case "invalid":
		return s.InvalidCSV(ctx, ws, id)
	case "unreadable":
		return s.BadRowsCSV(ctx, ws, id)
	}
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return nil, "", err
	}
	stem := strings.ToLower(SourceName(imp.Source)) + "-import-" + imp.CreatedAt.Format("2006-01-02")
	if kind == "skipped" {
		if len(imp.SkippedCsv) == 0 {
			return nil, "", api.NotFound("Report")
		}
		return imp.SkippedCsv, stem + "-skipped.csv", nil
	}
	if len(imp.ReportCsv) == 0 {
		return nil, "", api.NotFound("Report")
	}
	return imp.ReportCsv, stem + "-report.csv", nil
}

// RegisterJobs adds the staging and import workers.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &StageWorker{Svc: s})
	river.AddWorker(reg.Workers, &RunWorker{Svc: s})
}
