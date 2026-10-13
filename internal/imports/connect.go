package imports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// source builds the Source of an import from its stored credentials or file.
func (s *Service) source(imp store.Import) (Source, error) {
	if isFile(imp.Source) {
		opts := parseOptions(imp.Options)
		skip := map[int]bool{}
		for _, l := range opts.SkipRows {
			skip[l] = true
		}
		return &FileSource{Name: imp.FileName, Kind: imp.Source, Data: imp.FileData, SkipRows: skip}, nil
	}
	if len(imp.CredentialsEnc) == 0 {
		return nil, &SourceError{Code: CodeAuthFailed, Message: fmt.Sprintf("Connect %s with an API key first.", SourceName(imp.Source))}
	}
	raw, err := s.Box.Open(imp.CredentialsEnc, credentialsAD(imp.ID))
	if err != nil {
		return nil, errors.New("the stored API key can't be decrypted; paste it again")
	}
	var c credentials
	_ = json.Unmarshal(raw, &c)
	return s.newSource(imp.Source, c)
}

func (s *Service) newSource(src string, c credentials) (Source, error) {
	base := s.BaseURLs[src]
	switch src {
	case "mailchimp":
		return NewMailchimp(c.APIKey, base, s.HTTP)
	case "brevo":
		return NewBrevo(c.APIKey, base, s.HTTP)
	case "mailerlite":
		return NewMailerLite(c.APIKey, base, s.HTTP)
	case "listmonk":
		if base == "" {
			base = c.BaseURL
		}
		return NewListmonk(base, c.User, c.APIKey, s.HTTP)
	case "keila":
		if base == "" {
			base = c.BaseURL
		}
		return NewKeila(base, c.APIKey, s.HTTP)
	}
	return nil, fmt.Errorf("unknown source %q", src)
}

// ImportConnectInput is the API key (and server for self-hosted tools).
type ImportConnectInput struct {
	APIKey  string `json:"apiKey" minLength:"1" maxLength:"500" doc:"Write-only: stored encrypted, never returned"`
	User    string `json:"user,omitempty" maxLength:"200" doc:"listmonk API user"`
	BaseURL string `json:"baseUrl,omitempty" maxLength:"500" format:"uri" doc:"listmonk or Keila server address"`
}

// Connect checks the key against the source, stores it encrypted and
// returns the account and its lists.
func (s *Service) Connect(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ImportConnectInput) (ImportDetail, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportDetail{}, err
	}
	if err := editable(imp); err != nil {
		return ImportDetail{}, err
	}
	if isFile(imp.Source) {
		return ImportDetail{}, api.Invalid("body.apiKey", "File imports don't need an API key: upload the file instead.")
	}
	if in.BaseURL != "" && !strings.HasPrefix(in.BaseURL, "https://") && !strings.HasPrefix(in.BaseURL, "http://") {
		return ImportDetail{}, api.Invalid("body.baseUrl", "Use the full address, like https://lists.example.com.")
	}
	c := credentials{APIKey: strings.TrimSpace(in.APIKey), User: strings.TrimSpace(in.User), BaseURL: strings.TrimSpace(in.BaseURL)}
	src, err := s.newSource(imp.Source, c)
	if err != nil {
		return ImportDetail{}, sourceProblem(err)
	}
	conn, err := src.Connect(ctx)
	if err != nil {
		// Keep the draft usable: remember the failure for the error state.
		_ = s.storeError(ctx, imp, err)
		return ImportDetail{}, sourceProblem(err)
	}
	conn.BaseURL = c.BaseURL
	sealed, err := s.Box.Seal(mustJSON(c), credentialsAD(imp.ID))
	if err != nil {
		return ImportDetail{}, err
	}
	stored := storedConnection{ImportConnection: conn}
	if prev := parseConnection(imp.Connection); len(prev.Fields) > 0 {
		stored.Fields, stored.StagedLists = prev.Fields, prev.StagedLists
	}
	opts := parseOptions(imp.Options)
	if len(opts.Lists) == 0 && len(conn.Lists) > 0 {
		opts.Lists = []ImportListChoice{}
		for _, l := range conn.Lists {
			opts.Lists = append(opts.Lists, ImportListChoice{SourceID: l.ID})
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SetImportConnection(ctx, store.SetImportConnectionParams{ID: imp.ID, Connection: mustJSON(stored), CredentialsEnc: sealed}); err != nil {
			return err
		}
		if err := q.SetImportOptions(ctx, store.SetImportOptionsParams{ID: imp.ID, Options: mustJSON(opts)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.connect", TargetType: "import", TargetID: imp.ID.String(),
			TargetName: SourceName(imp.Source), Metadata: map[string]any{"account": conn.Account, "lists": len(conn.Lists)}})
	})
	if err != nil {
		return ImportDetail{}, err
	}
	return s.Get(ctx, ws, id)
}

// storeError remembers a source failure on the draft.
func (s *Service) storeError(ctx context.Context, imp store.Import, err error) error {
	conn := parseConnection(imp.Connection)
	conn.Error = toImportError(err)
	_, e := s.Pool.Exec(ctx, `UPDATE imports SET connection = $2, updated_at = now() WHERE id = $1`, imp.ID, mustJSON(conn))
	return e
}

func toImportError(err error) *ImportError {
	if se, ok := sourceError(err); ok {
		return &ImportError{Code: se.Code, Message: se.Message, Status: se.Status}
	}
	var fe *FileError
	if errors.As(err, &fe) {
		code := "file_unreadable"
		if len(fe.BadRows) > 0 {
			code = "file_columns"
		}
		return &ImportError{Code: code, Message: fe.Message, BadRows: fe.BadRows, Header: fe.Header}
	}
	return &ImportError{Code: "import_failed", Message: err.Error()}
}

// sourceProblem maps source errors to API problems.
func sourceProblem(err error) error {
	if se, ok := sourceError(err); ok {
		status := http.StatusBadGateway
		switch se.Code {
		case CodeAuthFailed:
			status = http.StatusUnprocessableEntity
		case CodeRateLimited:
			status = http.StatusTooManyRequests
		}
		p := api.Err(status, se.Code, se.Message)
		if se.Status > 0 {
			p = p.WithMeta("sourceStatus", se.Status)
		}
		return p
	}
	if errors.Is(err, ErrNoKey) {
		return api.Invalid("body.apiKey", "Paste an API key.")
	}
	var fe *FileError
	if errors.As(err, &fe) {
		code := "file_unreadable"
		if len(fe.BadRows) > 0 {
			code = "file_columns"
		}
		return api.Err(http.StatusUnprocessableEntity, code, fe.Message)
	}
	return err
}

// Upload stores a CSV or XLSX file on the draft and reads its header.
func (s *Service) Upload(ctx context.Context, ws *store.Workspace, id uuid.UUID, name string, data []byte) (ImportDetail, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportDetail{}, err
	}
	if err := editable(imp); err != nil {
		return ImportDetail{}, err
	}
	if len(data) == 0 {
		return ImportDetail{}, api.Invalid("body", "The file is empty.")
	}
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." {
		name = "contacts.csv"
	}
	kind := "csv"
	if strings.EqualFold(filepath.Ext(name), ".xlsx") || isZip(data) {
		kind = "xlsx"
	}
	f := &FileSource{Name: name, Kind: kind, Data: data}
	conn, cerr := f.Connect(ctx)
	stored := storedConnection{ImportConnection: conn}
	if cerr != nil {
		var fe *FileError
		if !errors.As(cerr, &fe) {
			return ImportDetail{}, cerr
		}
		stored.Error = toImportError(cerr)
		stored.BadRows, stored.Header = fe.BadRows, fe.Header
		stored.Account = name
		if stored.Lists == nil {
			stored.Lists = []ImportSourceList{}
		}
	} else if fields, err := f.Fields(ctx, nil); err == nil {
		stored.Fields = fields
	}
	opts := parseOptions(imp.Options)
	opts.Lists = []ImportListChoice{{SourceID: FileListID, Target: firstTarget(opts.Lists)}}
	opts.SkipRows = []int{}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := tx.Exec(ctx, `UPDATE imports SET source = $2 WHERE id = $1`, imp.ID, kind); err != nil {
			return err
		}
		if err := q.SetImportFile(ctx, store.SetImportFileParams{ID: imp.ID, FileName: name, FileData: data, Connection: mustJSON(stored)}); err != nil {
			return err
		}
		if err := q.SetImportOptions(ctx, store.SetImportOptionsParams{ID: imp.ID, Options: mustJSON(opts)}); err != nil {
			return err
		}
		if err := q.DeleteImportRows(ctx, imp.ID); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.upload", TargetType: "import", TargetID: imp.ID.String(), TargetName: name,
			Metadata: map[string]any{"bytes": len(data), "rows": f.Rows()}})
	})
	if err != nil {
		return ImportDetail{}, err
	}
	return s.Get(ctx, ws, id)
}

func isZip(b []byte) bool { return len(b) > 4 && b[0] == 'P' && b[1] == 'K' && b[2] == 3 && b[3] == 4 }

func firstTarget(ls []ImportListChoice) string {
	for _, l := range ls {
		if l.Target != "" {
			return l.Target
		}
	}
	return ""
}

// SkipBadRows leaves out the file rows whose column count doesn't match the
// header ("Skip the 4 rows and continue").
func (s *Service) SkipBadRows(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ImportDetail, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return ImportDetail{}, err
	}
	conn := parseConnection(imp.Connection)
	if len(conn.BadRows) == 0 {
		return s.detail(ctx, ws, imp)
	}
	opts := parseOptions(imp.Options)
	for _, b := range conn.BadRows {
		opts.SkipRows = append(opts.SkipRows, b.Line)
	}
	f := &FileSource{Name: imp.FileName, Kind: imp.Source, Data: imp.FileData, SkipRows: map[int]bool{}}
	for _, l := range opts.SkipRows {
		f.SkipRows[l] = true
	}
	c, err := f.Connect(ctx)
	if err != nil {
		return ImportDetail{}, sourceProblem(err)
	}
	fields, _ := f.Fields(ctx, nil)
	stored := storedConnection{ImportConnection: c, Fields: fields}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := tx.Exec(ctx, `UPDATE imports SET connection = $2, options = $3, updated_at = now() WHERE id = $1`, imp.ID, mustJSON(stored), mustJSON(opts)); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "import.skip_rows", TargetType: "import", TargetID: imp.ID.String(),
			TargetName: imp.FileName, Metadata: map[string]any{"rows": opts.SkipRows}})
	})
	if err != nil {
		return ImportDetail{}, err
	}
	return s.Get(ctx, ws, id)
}

// BadRowsCSV returns the rows a file import couldn't read, for download.
func (s *Service) BadRowsCSV(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]byte, string, error) {
	imp, err := s.get(ctx, ws, id)
	if err != nil {
		return nil, "", err
	}
	conn := parseConnection(imp.Connection)
	var sb strings.Builder
	sb.WriteString("row,columns,raw\n")
	for _, b := range conn.BadRows {
		fmt.Fprintf(&sb, "%d,%d,%q\n", b.Line, b.Columns, b.Raw)
	}
	return []byte(sb.String()), strings.TrimSuffix(imp.FileName, filepath.Ext(imp.FileName)) + "-unreadable-rows.csv", nil
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}
