package importer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Notifuse reads a self-hosted Notifuse instance's email templates with a
// workspace API key. The key finds its workspace through workspaces.list.
// Templates built in Notifuse's visual editor arrive as MJML; code-mode
// templates bring their MJML source; otherwise the compiled HTML is used.
type Notifuse struct {
	c         *client
	workspace string
	name      string
}

// NewNotifuse builds a client for the instance at base ("https://notifuse.acme.io").
func NewNotifuse(base, key string, hc *http.Client) (*Notifuse, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrNoKey
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil, sourceErr(0, CodeBadResponse, "notifuse.address", nil)
	}
	base = strings.TrimSuffix(base, "/api")
	return &Notifuse{c: newClient("Notifuse", base, hc, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+key) },
		func(status int, _ []byte) i18n.Text {
			return say("auth.notifuse", i18n.Args{"code": strconv.Itoa(status)})
		})}, nil
}

// Dialect implements TemplateSource.
func (n *Notifuse) Dialect() Dialect { return DialectNotifuse }

// NotifuseWorkspace is the workspace an API key is attached to.
type NotifuseWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Workspace finds the workspace the key belongs to.
func (n *Notifuse) Workspace(ctx context.Context) (NotifuseWorkspace, error) {
	if n.workspace != "" {
		return NotifuseWorkspace{ID: n.workspace, Name: n.name}, nil
	}
	var ws []NotifuseWorkspace
	if err := n.c.get(ctx, "/api/workspaces.list", &ws); err != nil {
		return NotifuseWorkspace{}, err
	}
	if len(ws) == 0 {
		return NotifuseWorkspace{}, sourceErr(0, CodeAuthFailed, "notifuse.noWorkspace", nil)
	}
	n.workspace, n.name = ws[0].ID, ws[0].Name
	if n.name == "" {
		n.name = ws[0].ID
	}
	return NotifuseWorkspace{ID: n.workspace, Name: n.name}, nil
}

// Get reads path (with workspace_id added) into out, for the contact
// importer.
func (n *Notifuse) Get(ctx context.Context, path string, q url.Values, out any) error {
	ws, err := n.Workspace(ctx)
	if err != nil {
		return err
	}
	if q == nil {
		q = url.Values{}
	}
	q.Set("workspace_id", ws.ID)
	return n.c.get(ctx, path+"?"+q.Encode(), out)
}

// Connect implements TemplateSource.
func (n *Notifuse) Connect(ctx context.Context) (Account, error) {
	ws, err := n.Workspace(ctx)
	if err != nil {
		return Account{}, err
	}
	return Account{Name: ws.Name, Kind: "workspace"}, nil
}

// NotifuseTemplate is a template as Notifuse's API returns it.
type NotifuseTemplate struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Channel  string         `json:"channel"`
	Category string         `json:"category"`
	Version  int64          `json:"version"`
	TestData map[string]any `json:"test_data"`
	Email    *struct {
		Subject          string          `json:"subject"`
		SubjectPreview   *string         `json:"subject_preview"`
		CompiledPreview  string          `json:"compiled_preview"`
		VisualEditorTree json.RawMessage `json:"visual_editor_tree"`
		Text             *string         `json:"text"`
		EditorMode       string          `json:"editor_mode"`
		MJMLSource       *string         `json:"mjml_source"`
	} `json:"email"`
}

// Templates lists the email templates.
func (n *Notifuse) Templates(ctx context.Context) ([]NotifuseTemplate, error) {
	var res struct {
		Templates []NotifuseTemplate `json:"templates"`
	}
	if err := n.Get(ctx, "/api/templates.list", url.Values{"channel": {"email"}}, &res); err != nil {
		return nil, err
	}
	out := []NotifuseTemplate{}
	for _, t := range res.Templates {
		if t.Channel == "" || t.Channel == "email" {
			out = append(out, t)
		}
	}
	return out, nil
}

// List implements TemplateSource. Transactional templates are picked by
// default; the others (marketing, welcome…) can be picked too.
func (n *Notifuse) List(ctx context.Context) ([]Ref, error) {
	ts, err := n.Templates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Ref, 0, len(ts))
	for _, t := range ts {
		out = append(out, Ref{Ref: t.ID, Alias: t.ID, Name: t.Name, Category: t.Category, Suggested: t.Category == "transactional"})
	}
	return out, nil
}

// Fetch implements TemplateSource.
func (n *Notifuse) Fetch(ctx context.Context, ref string) (Source, error) {
	var res struct {
		Template NotifuseTemplate `json:"template"`
	}
	if err := n.Get(ctx, "/api/templates.get", url.Values{"id": {ref}}, &res); err != nil {
		return Source{}, err
	}
	return NotifuseSource(res.Template), nil
}

// NotifuseSource turns a Notifuse template into a Source: MJML from the
// code editor or the visual editor tree, else the compiled HTML.
func NotifuseSource(t NotifuseTemplate) Source {
	src := Source{Ref: t.ID, Alias: t.ID, Name: t.Name, Category: t.Category, Sample: t.TestData, Notes: []Note{}, Format: FormatHTML}
	e := t.Email
	if e == nil {
		src.Notes = append(src.Notes, noteOf(NoteManual, "template", 0, "", say("note.notifuseEmpty", nil)))
		return src
	}
	src.Subject = e.Subject
	if e.SubjectPreview != nil {
		src.Preheader = *e.SubjectPreview
	}
	if e.Text != nil {
		src.Text = *e.Text
	}
	switch {
	case e.MJMLSource != nil && strings.TrimSpace(*e.MJMLSource) != "" && (e.EditorMode == "code" || len(e.VisualEditorTree) == 0 || string(e.VisualEditorTree) == "null"):
		src.Format, src.Body = FormatMJML, *e.MJMLSource
	case len(e.VisualEditorTree) > 0 && string(e.VisualEditorTree) != "null" && string(e.VisualEditorTree) != "{}":
		body, notes, err := TreeToMJML(e.VisualEditorTree)
		if err == nil {
			src.Format, src.Body = FormatMJML, body
			src.Notes = append(src.Notes, notes...)
			break
		}
		src.Body = e.CompiledPreview
		src.Notes = append(src.Notes, noteOf(NoteChange, "template", 0, "", say("note.notifuseLayout", i18n.Args{"detail": lowerFirst(err.Error())})))
	default:
		src.Body = e.CompiledPreview
	}
	return src
}
