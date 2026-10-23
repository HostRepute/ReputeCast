package importer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Sealer seals and opens the provider key (internal/secrets).
type Sealer interface {
	Seal(plaintext []byte, ad string) ([]byte, error)
	Open(ciphertext []byte, ad string) ([]byte, error)
}

// Drafts makes the transactional drafts (X02's template service).
type Drafts interface {
	CreateTemplate(ctx context.Context, ws *store.Workspace, p *authz.Principal, in transactional.TransactionalTemplateCreate) (transactional.TransactionalTemplate, error)
}

// Service runs the Import templates dialog (add-on §5.9): connect a
// provider (or upload files), pick templates, read the conversion report,
// and import them as drafts. One row of transactional_template_imports per
// run; the provider key is sealed for the run and dropped when it ends.
type Service struct {
	Pool   *pgxpool.Pool
	Q      *store.Queries
	Audit  *audit.Logger
	Box    Sealer
	Drafts Drafts
	// Saved keeps a Notifuse marketing template as a saved campaign template.
	Saved SavedTemplates
	// HTTP dials the providers through the SSRF guard.
	HTTP  *http.Client
	Guard netguard.Policy
	// Bases replaces the providers' API addresses (tests, the demo's
	// recorded accounts): "postmark", "sendgrid", "mailgun".
	Bases map[string]string
	// Samples accepts the sample Notifuse instance's address without
	// resolving it (development, where HTTP answers it from recordings).
	Samples bool
	Now     func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Sources the dialog offers.
const (
	SourcePostmark = "postmark"
	SourceSendGrid = "sendgrid"
	SourceMailgun  = "mailgun"
	SourceNotifuse = "notifuse"
	SourceFiles    = "files"
)

// SourceName is how the dialog names a source.
func SourceName(source string) string {
	switch source {
	case SourcePostmark:
		return "Postmark"
	case SourceSendGrid:
		return "SendGrid"
	case SourceMailgun:
		return "Mailgun"
	case SourceNotifuse:
		return "Notifuse"
	}
	return "MJML or HTML files"
}

// Session statuses: the dialog's steps.
const (
	statusPick      = "pick"
	statusReview    = "review"
	statusDone      = "done"
	statusCancelled = "cancelled"
)

// TemplateImportFile is an uploaded MJML or HTML file.
type TemplateImportFile struct {
	Name    string `json:"name" minLength:"1" maxLength:"200" example:"password-reset.mjml"`
	Content string `json:"content" maxLength:"2097152"`
}

// TemplateImportStart connects a provider, or converts uploaded files.
type TemplateImportStart struct {
	Source string `json:"source" enum:"postmark,sendgrid,mailgun,notifuse,files"`
	// APIKey is sealed and used only for this import; it is never returned.
	APIKey  string               `json:"apiKey,omitempty" maxLength:"500" doc:"Sealed, used only for this import, never returned"`
	BaseURL string               `json:"baseUrl,omitempty" maxLength:"500" doc:"Notifuse: the instance's address"`
	Region  string               `json:"region,omitempty" enum:"us,eu" doc:"Mailgun: the account's region"`
	Domain  string               `json:"domain,omitempty" maxLength:"253" doc:"Mailgun: one sending domain; empty reads every domain"`
	Files   []TemplateImportFile `json:"files,omitempty" maxItems:"50"`
}

// TemplateImportPick is the templates to convert.
type TemplateImportPick struct {
	Refs []string `json:"refs" minItems:"1" maxItems:"200"`
}

// TemplateImportItem is one row of the conversion report.
type TemplateImportItem struct {
	Ref   string `json:"ref"`
	Alias string `json:"alias"`
	Name  string `json:"name"`
	// Key is the ReputeCast ID the draft gets (password-reset-2 when
	// password-reset is taken).
	Key    string `json:"key"`
	Status Status `json:"status" enum:"converted,notes,manual"`
	// Summary is the report's Notes cell; SummaryLine its line (0 for none).
	Summary     string   `json:"summary"`
	SummaryKind NoteKind `json:"summaryKind" enum:"info,note,manual"`
	SummaryLine int      `json:"summaryLine"`
	Notes       []Note   `json:"notes"`
	Variables   int      `json:"variables"`
	// Failed is a template the provider wouldn't return: it isn't imported.
	Failed bool `json:"failed"`
}

// TemplateImportCreated is a draft an import made.
type TemplateImportCreated struct {
	ID     string `json:"id,omitempty"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Status Status `json:"status" enum:"converted,notes,manual"`
	// Error says why it couldn't be imported.
	Error string `json:"error,omitempty"`
}

// TemplateImportError is the last failure talking to the provider.
type TemplateImportError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TemplateImport is a run of the Import templates dialog.
type TemplateImport struct {
	ID     string `json:"id"`
	Source string `json:"source" enum:"postmark,sendgrid,mailgun,notifuse,files"`
	Status string `json:"status" enum:"connect,pick,review,done,cancelled"`
	// Account is what the key reads ("Acme Cloud production") and AccountKind what that is.
	Account     string `json:"account,omitempty"`
	AccountKind string `json:"accountKind,omitempty" enum:"server,account,workspace,domains"`
	// KeySealed: a key is stored, sealed, until the import ends. It is never returned.
	KeySealed bool                    `json:"keySealed"`
	Available []Ref                   `json:"available"`
	Picked    []string                `json:"picked"`
	Items     []TemplateImportItem    `json:"items"`
	Converted int                     `json:"converted"`
	WithNotes int                     `json:"withNotes"`
	Manual    int                     `json:"manual"`
	Created   []TemplateImportCreated `json:"created"`
	Error     *TemplateImportError    `json:"error,omitempty"`
	CreatedAt time.Time               `json:"createdAt"`
	// FinishedAt is when the drafts were made (or the import cancelled).
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// credentials are what the session seals.
type credentials struct {
	APIKey  string `json:"apiKey"`
	BaseURL string `json:"baseUrl,omitempty"`
	Region  string `json:"region,omitempty"`
	Domain  string `json:"domain,omitempty"`
}

func credentialsAD(id uuid.UUID) string {
	return "transactional_import:" + id.String() + ":credentials"
}

// connection is the session's account, kept to show it again.
type connection struct {
	Account string `json:"account,omitempty"`
	Kind    string `json:"kind,omitempty"`
	BaseURL string `json:"baseUrl,omitempty"`
	Region  string `json:"region,omitempty"`
	Domain  string `json:"domain,omitempty"`
}

// storedItem is a report row with what it converted to. Texts are its
// notes' kept messages (Note.Text), in the notes' order, so the report reads
// in each reader's language.
type storedItem struct {
	TemplateImportItem
	Converted Converted   `json:"converted"`
	Texts     []i18n.Text `json:"texts,omitempty"`
}

// textsOf is the notes' kept messages.
func textsOf(notes []Note) []i18n.Text {
	out := make([]i18n.Text, len(notes))
	for i, n := range notes {
		out[i] = n.Text
	}
	return out
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`null`)
	}
	return b
}

// source builds the client of a provider.
func (s *Service) source(ctx context.Context, kind string, c credentials) (TemplateSource, error) {
	switch kind {
	case SourcePostmark:
		return NewPostmark(c.APIKey, s.Bases[SourcePostmark], s.HTTP)
	case SourceSendGrid:
		return NewSendGrid(c.APIKey, s.Bases[SourceSendGrid], s.HTTP)
	case SourceMailgun:
		return NewMailgun(c.APIKey, c.Region, c.Domain, s.Bases[SourceMailgun], s.HTTP)
	case SourceNotifuse:
		return NewNotifuse(c.BaseURL, c.APIKey, s.HTTP)
	}
	return nil, api.Invalid("body.source", sayIn(ctx, "problem.source", nil))
}

// serverAddress checks a self-hosted instance's address: a full http(s)
// address, no credentials or query, and not on an internal network. The
// problem is in the request's language.
func (s *Service) serverAddress(ctx context.Context, raw string) (string, string) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", sayIn(ctx, "problem.fullAddress", nil)
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery {
		return "", sayIn(ctx, "problem.onlyAddress", nil)
	}
	sample := s.Samples && u.Scheme+"://"+u.Host == RecordedNotifuse
	if _, err := s.Guard.ValidateURL(u.String()); err != nil && !sample {
		return "", sayIn(ctx, "problem.privateAddress", nil)
	}
	return u.Scheme + "://" + u.Host + strings.TrimRight(u.EscapedPath(), "/"), ""
}

// sourceProblem turns a provider failure into the API's answer, in the
// request's language.
func sourceProblem(ctx context.Context, err error) error {
	var se *SourceError
	if errors.As(err, &se) {
		status := http.StatusBadGateway
		switch se.Code {
		case CodeAuthFailed:
			status = http.StatusUnprocessableEntity
		case CodeRateLimited:
			status = http.StatusTooManyRequests
		case CodeNotFound:
			status = http.StatusUnprocessableEntity
		}
		p := api.Err(status, se.Code, errorText(i18n.LangFrom(ctx), se))
		if se.Status > 0 {
			p = p.WithMeta("sourceStatus", se.Status)
		}
		return p
	}
	if errors.Is(err, ErrNoKey) {
		return api.Invalid("body.apiKey", sayIn(ctx, "problem.apiKey", nil))
	}
	return err
}

func (s *Service) keysInUse(ctx context.Context, q *store.Queries, wsID uuid.UUID) (map[string]bool, error) {
	keys, err := q.TxnTemplateKeysInUse(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = true
	}
	return out, nil
}

// item makes a report row: the ID the draft gets, and its Notes cell.
func item(c Converted, src Source, taken map[string]bool) storedItem {
	name := c.Name
	if name == "" {
		name = src.Ref
	}
	want := SuggestID(src.Alias, name)
	key := UniqueID(want, taken)
	if key != want {
		c.Notes = append(c.Notes, noteOf(NoteInfo, "template", 0, "", say("note.idTaken", i18n.Args{"want": want, "key": key})))
		c.Notes = dedupeNotes(c.Notes)
	}
	it := TemplateImportItem{Ref: src.Ref, Alias: src.Alias, Name: name, Key: key, Status: c.Status, Notes: c.Notes, Variables: len(c.Variables)}
	it.Summary, it.SummaryKind, it.SummaryLine = summaryOf(i18n.Default, c)
	return storedItem{TemplateImportItem: it, Converted: c, Texts: textsOf(c.Notes)}
}

// summaryOf is the Notes cell in a reader's language: the first manual fix
// or note, else the ID that changed, else the variables declared.
func summaryOf(l i18n.Lang, c Converted) (string, NoteKind, int) {
	for _, n := range c.Notes {
		if n.Kind == NoteManual || n.Kind == NoteChange {
			return n.In(l).Message, n.Kind, n.Line
		}
	}
	for _, n := range c.Notes {
		if n.Kind == NoteInfo && n.Field == "template" {
			return n.In(l).Message, n.Kind, 0
		}
	}
	return VariablesLineIn(l, c.Variables), NoteInfo, 0
}

// Start connects a provider and lists its templates, or converts uploaded
// files straight to the report.
func (s *Service) Start(ctx context.Context, ws *store.Workspace, p *authz.Principal, in TemplateImportStart) (TemplateImport, error) {
	kind := strings.ToLower(strings.TrimSpace(in.Source))
	if kind == SourceFiles {
		return s.startFiles(ctx, ws, p, in.Files)
	}
	c := credentials{APIKey: strings.TrimSpace(in.APIKey), Region: strings.ToLower(strings.TrimSpace(in.Region)), Domain: strings.TrimSpace(in.Domain)}
	if kind == SourceNotifuse {
		base, problem := s.serverAddress(ctx, in.BaseURL)
		if problem != "" {
			return TemplateImport{}, api.Invalid("body.baseUrl", problem)
		}
		c.BaseURL = base
	}
	src, err := s.source(ctx, kind, c)
	if err != nil {
		return TemplateImport{}, sourceProblem(ctx, err)
	}
	acct, err := src.Connect(ctx)
	if err != nil {
		return TemplateImport{}, sourceProblem(ctx, err)
	}
	refs, err := src.List(ctx)
	if err != nil {
		return TemplateImport{}, sourceProblem(ctx, err)
	}
	id := uuid.Must(uuid.NewV7())
	sealed, err := s.Box.Seal(mustJSON(c), credentialsAD(id))
	if err != nil {
		return TemplateImport{}, err
	}
	picked := []string{}
	for _, r := range refs {
		if r.Suggested {
			picked = append(picked, r.Ref)
		}
	}
	conn := connection{Account: acct.Name, Kind: acct.Kind, BaseURL: c.BaseURL, Region: c.Region, Domain: c.Domain}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.TxnImportInsert(ctx, store.TxnImportInsertParams{ID: id, WorkspaceID: ws.ID, Source: kind, Status: statusPick, CredentialsEnc: sealed,
			Connection: mustJSON(conn), Available: mustJSON(refs), Picked: picked, Files: json.RawMessage(`[]`), Report: json.RawMessage(`[]`), CreatedBy: userRef(p), Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.import_started", TargetType: "transactional_import", TargetID: id.String(),
			TargetName: SourceName(kind), Metadata: map[string]any{"source": kind, "account": acct.Name, "templates": len(refs)}})
	})
	if err != nil {
		return TemplateImport{}, err
	}
	return s.Get(ctx, ws, id.String())
}

func (s *Service) startFiles(ctx context.Context, ws *store.Workspace, p *authz.Principal, files []TemplateImportFile) (TemplateImport, error) {
	if len(files) == 0 {
		return TemplateImport{}, api.Invalid("body.files", sayIn(ctx, "problem.noFiles", nil))
	}
	if len(files) > MaxFiles {
		return TemplateImport{}, api.Invalid("body.files", sayIn(ctx, "problem.tooManyFiles", i18n.Args{"max": MaxFiles}))
	}
	taken, err := s.keysInUse(ctx, s.Q, ws.ID)
	if err != nil {
		return TemplateImport{}, err
	}
	items := make([]storedItem, 0, len(files))
	names := make([]map[string]string, 0, len(files))
	for i, f := range files {
		if len(f.Content) > MaxFileBytes {
			return TemplateImport{}, api.Invalid(fmt.Sprintf("body.files[%d]", i), sayIn(ctx, "problem.fileTooLarge", i18n.Args{"name": f.Name}))
		}
		src, dialect, err := FromFile(f.Name, []byte(f.Content))
		if err != nil {
			msg := err.Error()
			switch {
			case errors.Is(err, ErrFileType):
				msg = sayIn(ctx, "problem.fileType", nil)
			case errors.Is(err, ErrNotText):
				msg = sayIn(ctx, "problem.notText", nil)
			}
			return TemplateImport{}, api.Invalid(fmt.Sprintf("body.files[%d]", i), msg)
		}
		c := Convert(src, Options{Dialect: dialect, Transactional: true})
		items = append(items, item(c, src, taken))
		names = append(names, map[string]string{"name": f.Name})
	}
	id := uuid.Must(uuid.NewV7())
	manual := 0
	for _, it := range items {
		if it.Status == StatusManual {
			manual++
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.TxnImportInsert(ctx, store.TxnImportInsertParams{ID: id, WorkspaceID: ws.ID, Source: SourceFiles, Status: statusReview,
			Connection: json.RawMessage(`{}`), Available: json.RawMessage(`[]`), Picked: []string{}, Files: mustJSON(names), Report: mustJSON(items), CreatedBy: userRef(p), Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.import_started", TargetType: "transactional_import", TargetID: id.String(),
			TargetName: SourceName(SourceFiles), Metadata: map[string]any{"source": SourceFiles, "templates": len(items), "manual": manual}})
	})
	if err != nil {
		return TemplateImport{}, err
	}
	return s.Get(ctx, ws, id.String())
}

func userRef(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

func (s *Service) load(ctx context.Context, q *store.Queries, ws *store.Workspace, ref string, lock bool) (store.TransactionalTemplateImport, error) {
	id, err := uuid.Parse(ref)
	if err != nil {
		return store.TransactionalTemplateImport{}, api.Err(http.StatusNotFound, api.CodeNotFound, sayIn(ctx, "notFound.import", nil))
	}
	var row store.TransactionalTemplateImport
	if lock {
		row, err = q.TxnImportForUpdate(ctx, store.TxnImportForUpdateParams{WorkspaceID: ws.ID, ID: id})
	} else {
		row, err = q.TxnImportGet(ctx, store.TxnImportGetParams{WorkspaceID: ws.ID, ID: id})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return row, api.Err(http.StatusNotFound, api.CodeNotFound, sayIn(ctx, "notFound.import", nil))
	}
	if err != nil {
		return row, api.MapError(err, "Import")
	}
	return row, nil
}

// Get returns a run of the dialog, its report in the request's language.
func (s *Service) Get(ctx context.Context, ws *store.Workspace, ref string) (TemplateImport, error) {
	row, err := s.load(ctx, s.Q, ws, ref, false)
	if err != nil {
		return TemplateImport{}, err
	}
	return view(i18n.LangFrom(ctx), row), nil
}

// localize writes a stored report row in a reader's language: its notes
// from their kept messages, and its Notes cell from them.
func (it storedItem) localize(l i18n.Lang) TemplateImportItem {
	out := it.TemplateImportItem
	if l.Locale == i18n.English || len(it.Texts) == 0 {
		return out
	}
	notes := make([]Note, len(out.Notes))
	for i, n := range out.Notes {
		if i < len(it.Texts) {
			n.Text = it.Texts[i]
		}
		notes[i] = n
	}
	out.Notes = make([]Note, len(notes))
	for i, n := range notes {
		out.Notes[i] = n.In(l)
	}
	if out.Failed {
		if len(out.Notes) > 0 {
			out.Summary = out.Notes[0].Message
		}
		return out
	}
	c := it.Converted
	c.Notes = notes
	out.Summary, out.SummaryKind, out.SummaryLine = summaryOf(l, c)
	return out
}

func view(l i18n.Lang, row store.TransactionalTemplateImport) TemplateImport {
	out := TemplateImport{ID: row.ID.String(), Source: row.Source, Status: row.Status, KeySealed: len(row.CredentialsEnc) > 0,
		Available: []Ref{}, Picked: row.Picked, Items: []TemplateImportItem{}, Created: []TemplateImportCreated{}, CreatedAt: row.CreatedAt, FinishedAt: row.FinishedAt}
	if out.Picked == nil {
		out.Picked = []string{}
	}
	var conn connection
	_ = json.Unmarshal(row.Connection, &conn)
	out.Account, out.AccountKind = conn.Account, conn.Kind
	_ = json.Unmarshal(row.Available, &out.Available)
	if out.Available == nil {
		out.Available = []Ref{}
	}
	var items []storedItem
	_ = json.Unmarshal(row.Report, &items)
	for _, it := range items {
		out.Items = append(out.Items, it.localize(l))
		switch it.Status {
		case StatusConverted:
			out.Converted++
		case StatusNotes:
			out.WithNotes++
		case StatusManual:
			out.Manual++
		}
	}
	_ = json.Unmarshal(row.Result, &out.Created)
	if out.Created == nil {
		out.Created = []TemplateImportCreated{}
	}
	if len(row.Error) > 0 && string(row.Error) != "null" {
		var e TemplateImportError
		if json.Unmarshal(row.Error, &e) == nil && e.Code != "" {
			out.Error = &e
		}
	}
	return out
}

func (s *Service) openCredentials(ctx context.Context, row store.TransactionalTemplateImport) (credentials, error) {
	var c credentials
	if len(row.CredentialsEnc) == 0 {
		return c, api.Conflict(sayIn(ctx, "problem.noKey", nil))
	}
	raw, err := s.Box.Open(row.CredentialsEnc, credentialsAD(row.ID))
	if err != nil {
		return c, api.Conflict(sayIn(ctx, "problem.sealed", nil))
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, err
	}
	return c, nil
}

// Convert reads the picked templates from the provider and converts them.
func (s *Service) Convert(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string, in TemplateImportPick) (TemplateImport, error) {
	row, err := s.load(ctx, s.Q, ws, ref, false)
	if err != nil {
		return TemplateImport{}, err
	}
	if row.Status != statusPick && row.Status != statusReview {
		return TemplateImport{}, api.Conflict(sayIn(ctx, "problem.finished", nil))
	}
	if row.Source == SourceFiles {
		return TemplateImport{}, api.Conflict(sayIn(ctx, "problem.filesConverted", nil))
	}
	c, err := s.openCredentials(ctx, row)
	if err != nil {
		return TemplateImport{}, err
	}
	src, err := s.source(ctx, row.Source, c)
	if err != nil {
		return TemplateImport{}, sourceProblem(ctx, err)
	}
	var available []Ref
	_ = json.Unmarshal(row.Available, &available)
	byRef := map[string]Ref{}
	for _, r := range available {
		byRef[r.Ref] = r
	}
	picked := []string{}
	for _, r := range available {
		if slices.Contains(in.Refs, r.Ref) {
			picked = append(picked, r.Ref)
		}
	}
	if len(picked) == 0 {
		return TemplateImport{}, api.Invalid("body.refs", sayIn(ctx, "problem.pickTemplate", nil))
	}
	taken, err := s.keysInUse(ctx, s.Q, ws.ID)
	if err != nil {
		return TemplateImport{}, err
	}
	items := make([]storedItem, 0, len(picked))
	manual := 0
	for _, r := range picked {
		t, err := src.Fetch(ctx, r)
		if err != nil {
			var se *SourceError
			if errors.As(err, &se) && (se.Code == CodeAuthFailed || se.Code == CodeRateLimited) {
				return TemplateImport{}, sourceProblem(ctx, err)
			}
			msg := err.Error()
			if se != nil {
				msg = se.Message
			}
			note := noteOf(NoteManual, "template", 0, "", say("note.notReturned", i18n.Args{"source": SourceName(row.Source), "reason": lowerFirst(strings.TrimSuffix(msg, "."))}))
			it := TemplateImportItem{Ref: r, Alias: byRef[r].Alias, Name: byRef[r].Name, Key: UniqueID(SuggestID(byRef[r].Alias, byRef[r].Name), taken), Status: StatusManual,
				Notes: []Note{note}, Failed: true}
			it.Summary, it.SummaryKind = it.Notes[0].Message, NoteManual
			items = append(items, storedItem{TemplateImportItem: it, Texts: textsOf(it.Notes)})
			manual++
			continue
		}
		if t.Name == "" {
			t.Name = byRef[r].Name
		}
		if t.Alias == "" {
			t.Alias = byRef[r].Alias
		}
		conv := Convert(t, Options{Dialect: src.Dialect(), Transactional: true})
		it := item(conv, t, taken)
		if it.Status == StatusManual {
			manual++
		}
		items = append(items, it)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.load(ctx, q, ws, ref, true)
		if err != nil {
			return err
		}
		if cur.Status != statusPick && cur.Status != statusReview {
			return api.Conflict(sayIn(ctx, "problem.finished", nil))
		}
		if err := q.TxnImportSave(ctx, store.TxnImportSaveParams{WorkspaceID: ws.ID, ID: cur.ID, Status: statusReview, Picked: picked, Report: mustJSON(items),
			Result: json.RawMessage(`[]`), CredentialsEnc: cur.CredentialsEnc, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.import_converted", TargetType: "transactional_import", TargetID: cur.ID.String(),
			TargetName: SourceName(cur.Source), Metadata: map[string]any{"templates": len(items), "manual": manual}})
	})
	if err != nil {
		return TemplateImport{}, err
	}
	return s.Get(ctx, ws, ref)
}

// createInput is the draft X02's template service makes from a report row.
func createInput(it storedItem, source, importID string) transactional.TransactionalTemplateCreate {
	c := it.Converted
	in := transactional.TransactionalTemplateCreate{Name: it.Name, Key: it.Key, Subject: c.Subject, Preheader: c.Preheader, PlainText: c.Text,
		Variables: draftVariables(c.Variables),
		Origin:    map[string]any{"source": source, "ref": it.Ref, "alias": it.Alias, "import": importID, "status": string(it.Status)}}
	if c.Format == FormatMJML {
		in.EditorMode, in.MJML, in.HTML = "visual", c.Body, c.HTML
		if strings.TrimSpace(c.HTML) == "" {
			// MJML that doesn't compile still arrives: the editor's MJML view shows why.
			in.EditorMode, in.HTML = "mjml", c.Body
		}
	} else {
		in.EditorMode, in.HTML = "html", c.Body
	}
	if len(c.TestData) > 0 {
		in.TestData = []transactional.TransactionalTestData{{Name: "Imported sample", Data: c.TestData}}
	}
	if len(in.Name) > 120 {
		in.Name = in.Name[:120]
	}
	return in
}

// draftVariables turns the converter's declarations into the API's: samples
// and defaults are text, and a sample that doesn't read as its type makes
// the variable text.
func draftVariables(vs []Variable) []transactional.TransactionalVariable {
	out := make([]transactional.TransactionalVariable, 0, len(vs))
	for _, v := range vs {
		if !apiName.MatchString(v.Name) || systemVars[v.Name] || v.Name == "forloop" || v.Name == "tablerowloop" {
			continue
		}
		tv := transactional.TransactionalVariable{Name: v.Name, Type: v.Type, Required: v.Required, Default: v.Default, Sensitive: v.Sensitive, Sample: sampleText(v.Sample)}
		if !fitsType(tv.Type, tv.Sample) || !fitsType(tv.Type, tv.Default) {
			tv.Type = TypeText
		}
		if len(tv.Sample) > 4000 {
			tv.Sample = ""
		}
		out = append(out, tv)
	}
	return out
}

func sampleText(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case json.Number:
		return x.String()
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// fitsType mirrors the template API's check of samples and defaults.
func fitsType(typ, s string) bool {
	if s == "" {
		return true
	}
	switch typ {
	case TypeNumber:
		_, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return err == nil
	case TypeBool:
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "false", "yes", "no", "1", "0":
			return true
		}
		return false
	case TypeList:
		var a []any
		return json.Unmarshal([]byte(s), &a) == nil
	case TypeObject:
		var o map[string]any
		return json.Unmarshal([]byte(s), &o) == nil
	}
	return true
}

// Import makes a draft of every converted template. Nothing is published.
func (s *Service) Import(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (TemplateImport, error) {
	row, err := s.load(ctx, s.Q, ws, ref, false)
	if err != nil {
		return TemplateImport{}, err
	}
	if row.Status != statusReview {
		return TemplateImport{}, api.Conflict(sayIn(ctx, "problem.convertFirst", nil))
	}
	var items []storedItem
	_ = json.Unmarshal(row.Report, &items)
	taken, err := s.keysInUse(ctx, s.Q, ws.ID)
	if err != nil {
		return TemplateImport{}, err
	}
	created := []TemplateImportCreated{}
	manual := 0
	for _, it := range items {
		if it.Failed {
			continue
		}
		in := createInput(it, row.Source, row.ID.String())
		if taken[in.Key] {
			// Someone took the ID since the report.
			in.Key = UniqueID(in.Key, taken)
		}
		taken[in.Key] = true
		t, err := s.Drafts.CreateTemplate(ctx, ws, p, in)
		if err != nil {
			created = append(created, TemplateImportCreated{Key: in.Key, Name: in.Name, Status: it.Status, Error: problemText(err)})
			continue
		}
		if it.Status == StatusManual {
			manual++
		}
		created = append(created, TemplateImportCreated{ID: t.ID, Key: t.Key, Name: t.Name, Status: it.Status})
	}
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.load(ctx, q, ws, ref, true)
		if err != nil {
			return err
		}
		if err := q.TxnImportSave(ctx, store.TxnImportSaveParams{WorkspaceID: ws.ID, ID: cur.ID, Status: statusDone, Picked: cur.Picked, Report: cur.Report,
			Result: mustJSON(created), CredentialsEnc: nil, Now: now, FinishedAt: &now}); err != nil {
			return err
		}
		ok := 0
		for _, c := range created {
			if c.Error == "" {
				ok++
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.templates_imported", TargetType: "transactional_import", TargetID: cur.ID.String(),
			TargetName: SourceName(cur.Source), Metadata: map[string]any{"source": cur.Source, "drafts": ok, "manual": manual, "failed": len(created) - ok}})
	})
	if err != nil {
		return TemplateImport{}, err
	}
	return s.Get(ctx, ws, ref)
}

// problemText is an API error's detail.
func problemText(err error) string {
	var p *api.Problem
	if errors.As(err, &p) && p.Detail != "" {
		return p.Detail
	}
	var tp *transactional.Problem
	if errors.As(err, &tp) && tp.Detail != "" {
		return tp.Detail
	}
	return err.Error()
}

// Cancel ends a run without importing; the key is dropped.
func (s *Service) Cancel(ctx context.Context, ws *store.Workspace, p *authz.Principal, ref string) (TemplateImport, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := s.load(ctx, q, ws, ref, true)
		if err != nil {
			return err
		}
		if cur.Status == statusDone || cur.Status == statusCancelled {
			return nil
		}
		now := s.now()
		if err := q.TxnImportSave(ctx, store.TxnImportSaveParams{WorkspaceID: ws.ID, ID: cur.ID, Status: statusCancelled, Picked: cur.Picked, Report: cur.Report,
			Result: cur.Result, CredentialsEnc: nil, Now: now, FinishedAt: &now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "transactional.import_cancelled", TargetType: "transactional_import", TargetID: cur.ID.String(),
			TargetName: SourceName(cur.Source)})
	})
	if err != nil {
		return TemplateImport{}, err
	}
	return s.Get(ctx, ws, ref)
}
