package imports

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Notifuse reads contacts and lists from a self-hosted Notifuse instance
// with a workspace API key (read-only use). The key finds its workspace
// through workspaces.list. Templates come in with the importer's template
// step (TemplateImporter): transactional ones as drafts.
type Notifuse struct {
	c         *apiClient
	workspace string
	name      string
	labels    map[string]string
}

// NewNotifuse builds a client for the Notifuse instance at baseURL.
func NewNotifuse(baseURL, key string, hc *http.Client) (*Notifuse, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrNoKey
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, &SourceError{Code: CodeBadResponse, Message: "Enter the address of your Notifuse instance, for example https://notifuse.example.com."}
	}
	base = strings.TrimSuffix(base, "/api")
	return &Notifuse{c: newAPIClient("Notifuse", base, hc, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+strings.TrimSpace(key)) })}, nil
}

func (n *Notifuse) setOnRetry(f func(int, time.Duration, int)) { n.c.OnRetry = f }
func (n *Notifuse) setPace(p int)                              { n.c.setPace(p) }

// workspaceID finds the workspace the key is attached to.
func (n *Notifuse) workspaceID(ctx context.Context) (string, error) {
	if n.workspace != "" {
		return n.workspace, nil
	}
	var ws []struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Settings struct {
			CustomFieldLabels map[string]string `json:"custom_field_labels"`
		} `json:"settings"`
	}
	if err := n.c.get(ctx, "/api/workspaces.list", &ws); err != nil {
		return "", err
	}
	if len(ws) == 0 {
		return "", &SourceError{Status: 403, Code: CodeAuthFailed, Message: "Notifuse rejected the API key: it isn't attached to a workspace."}
	}
	n.workspace, n.name, n.labels = ws[0].ID, ws[0].Name, ws[0].Settings.CustomFieldLabels
	if n.name == "" {
		n.name = ws[0].ID
	}
	return n.workspace, nil
}

func (n *Notifuse) get(ctx context.Context, path string, q url.Values, out any) error {
	ws, err := n.workspaceID(ctx)
	if err != nil {
		return err
	}
	if q == nil {
		q = url.Values{}
	}
	q.Set("workspace_id", ws)
	return n.c.get(ctx, path+"?"+q.Encode(), out)
}

type nfList struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	IsDoubleOptin bool   `json:"is_double_optin"`
	CreatedAt     string `json:"created_at"`
}

// NotifuseTemplateCounts says how many email templates an import brings:
// transactional ones become drafts, marketing, welcome and other templates
// go to the composer's saved templates. Notifuse's own system emails
// (double opt-in, unsubscribe, bounce, blocklist) stay behind.
func notifuseTemplateKind(category string) string {
	switch category {
	case "transactional":
		return "transactional"
	case "marketing", "welcome", "other", "":
		return "saved"
	}
	return "skipped"
}

// Connect implements Source: the workspace, its lists with counts per
// status, and its email templates.
func (n *Notifuse) Connect(ctx context.Context) (ImportConnection, error) {
	var lists struct {
		Lists []nfList `json:"lists"`
	}
	if err := n.get(ctx, "/api/lists.list", nil, &lists); err != nil {
		return ImportConnection{}, err
	}
	out := ImportConnection{Account: n.name, BaseURL: n.c.base, Lists: []ImportSourceList{}, CheckedAt: time.Now()}
	for _, l := range lists.Lists {
		var st struct {
			Stats struct {
				TotalActive       int64 `json:"total_active"`
				TotalPending      int64 `json:"total_pending"`
				TotalUnsubscribed int64 `json:"total_unsubscribed"`
				TotalBounced      int64 `json:"total_bounced"`
				TotalComplained   int64 `json:"total_complained"`
			} `json:"stats"`
		}
		if err := n.get(ctx, "/api/lists.stats", url.Values{"list_id": {l.ID}}, &st); err != nil {
			return ImportConnection{}, err
		}
		sl := ImportSourceList{ID: l.ID, Name: l.Name, DoubleOptIn: l.IsDoubleOptin, TagExamples: []string{},
			Subscribed: st.Stats.TotalActive, Pending: st.Stats.TotalPending, Unsubscribed: st.Stats.TotalUnsubscribed,
			Cleaned: st.Stats.TotalBounced, Complained: st.Stats.TotalComplained}
		if t := parseTime(l.CreatedAt); t != nil {
			sl.Since = strconv.Itoa(t.Year())
		}
		out.Lists = append(out.Lists, sl)
	}
	var tpl struct {
		Templates []struct {
			Channel  string `json:"channel"`
			Category string `json:"category"`
		} `json:"templates"`
	}
	if err := n.get(ctx, "/api/templates.list", url.Values{"channel": {"email"}}, &tpl); err != nil {
		return ImportConnection{}, err
	}
	counts := &ImportTemplateCounts{}
	for _, t := range tpl.Templates {
		if t.Channel != "" && t.Channel != "email" {
			continue
		}
		switch notifuseTemplateKind(t.Category) {
		case "transactional":
			counts.Transactional++
		case "saved":
			counts.Saved++
		default:
			counts.Skipped++
		}
	}
	out.Templates = counts
	return out, nil
}

// nfFields are a Notifuse contact's fields, in the order its contact
// screen shows them.
var nfFields = []struct{ key, typ string }{
	{"email", "email"}, {"first_name", "text"}, {"last_name", "text"}, {"full_name", "text"}, {"external_id", "text"},
	{"phone", "phone"}, {"country", "text"}, {"language", "text"}, {"timezone", "text"}, {"job_title", "text"},
	{"address_line_1", "address"}, {"address_line_2", "address"}, {"postcode", "address"}, {"state", "address"},
}

type nfContact map[string]any

func (n *Notifuse) contacts(ctx context.Context, list, cursor string, limit int) ([]nfContact, string, error) {
	q := url.Values{"list_id": {list}, "with_contact_lists": {"true"}, "limit": {strconv.Itoa(limit)}}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	var page struct {
		Contacts   []nfContact `json:"contacts"`
		NextCursor *string     `json:"next_cursor"`
	}
	if err := n.get(ctx, "/api/contacts.list", q, &page); err != nil {
		return nil, "", err
	}
	next := ""
	if page.NextCursor != nil {
		next = *page.NextCursor
	}
	return page.Contacts, next, nil
}

// customKey names a custom field by its label when the workspace gave it
// one ("Plan" for custom_string_1), else by Notifuse's own name.
func (n *Notifuse) customKey(field string) (key, label string) {
	if l := strings.TrimSpace(n.labels[field]); l != "" {
		return snakeKey(l), l
	}
	return field, field
}

func isNotifuseCustom(k string) bool {
	for _, p := range []string{"custom_string_", "custom_number_", "custom_datetime_", "custom_json_"} {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}

// Fields implements Source: the built-in fields, then the custom fields
// set on the first page of each list.
func (n *Notifuse) Fields(ctx context.Context, lists []string) ([]SourceField, error) {
	out := make([]SourceField, 0, len(nfFields)+4)
	for _, f := range nfFields {
		out = append(out, SourceField{Key: f.key, Label: f.key, Type: f.typ})
	}
	seen := map[string]bool{}
	custom := []SourceField{}
	add := func(k string) {
		if seen[k] {
			return
		}
		seen[k] = true
		key, label := n.customKey(k)
		typ := "text"
		switch {
		case strings.HasPrefix(k, "custom_number_"):
			typ = "number"
		case strings.HasPrefix(k, "custom_datetime_"):
			typ = "date"
		}
		custom = append(custom, SourceField{Key: key, Label: label, Type: typ})
	}
	if _, err := n.workspaceID(ctx); err != nil {
		return nil, err
	}
	// Fields the workspace labelled are in use, wherever their values are.
	for k := range n.labels {
		if isNotifuseCustom(k) {
			add(k)
		}
	}
	for _, l := range lists {
		page, _, err := n.contacts(ctx, l, "", 100)
		if err != nil {
			return nil, err
		}
		for _, c := range page {
			for k, v := range c {
				if isNotifuseCustom(k) && v != nil {
					add(k)
				}
			}
		}
	}
	slices.SortFunc(custom, func(a, b SourceField) int { return strings.Compare(a.Key, b.Key) })
	return append(out, custom...), nil
}

// statusOf maps a Notifuse list subscription status.
func notifuseStatus(s string) string {
	switch s {
	case "pending":
		return StatusPending
	case "unsubscribed":
		return StatusUnsubscribed
	case "bounced":
		return StatusCleaned
	case "complained":
		return StatusComplained
	}
	return StatusSubscribed
}

// Fetch implements Source: every contact of each list, a page of 100 at a
// time, with its status on that list.
func (n *Notifuse) Fetch(ctx context.Context, lists []string, emit func(Row) error) error {
	for _, l := range lists {
		cursor := ""
		for {
			page, next, err := n.contacts(ctx, l, cursor, 100)
			if err != nil {
				return err
			}
			for _, c := range page {
				if err := emit(n.row(l, c)); err != nil {
					return err
				}
			}
			if next == "" || len(page) == 0 || next == cursor {
				break
			}
			cursor = next
		}
	}
	return nil
}

func (n *Notifuse) row(list string, c nfContact) Row {
	email, _ := c["email"].(string)
	row := Row{Email: email, List: list, Status: StatusSubscribed, Data: map[string]string{}, Tags: []string{}}
	for _, f := range nfFields {
		if v, ok := c[f.key]; ok && v != nil {
			row.Data[f.key] = valueString(v)
		}
	}
	for k, v := range c {
		if !isNotifuseCustom(k) || v == nil {
			continue
		}
		key, _ := n.customKey(k)
		if strings.HasPrefix(k, "custom_json_") {
			b, _ := json.Marshal(v)
			row.Data[key] = string(b)
			continue
		}
		row.Data[key] = valueString(v)
	}
	if cls, ok := c["contact_lists"].([]any); ok {
		for _, x := range cls {
			cl, _ := x.(map[string]any)
			if id, _ := cl["list_id"].(string); id != list {
				continue
			}
			st, _ := cl["status"].(string)
			row.Status = notifuseStatus(st)
			if s, _ := cl["created_at"].(string); s != "" {
				row.Engagement.SignupAt = parseTime(s)
				if row.Status == StatusSubscribed {
					row.Engagement.OptinAt = parseTime(s)
				}
			}
			if s, _ := cl["updated_at"].(string); s != "" {
				row.Engagement.Changed = parseTime(s)
			}
		}
	}
	if row.Engagement.SignupAt == nil {
		if s, _ := c["created_at"].(string); s != "" {
			row.Engagement.SignupAt = parseTime(s)
		}
	}
	return row
}

// ImportTemplateCounts are the email templates a source brings along.
type ImportTemplateCounts struct {
	// Transactional templates arrive as transactional drafts.
	Transactional int `json:"transactional"`
	// Saved are marketing and other templates, added to the composer's
	// saved templates.
	Saved int `json:"saved"`
	// Skipped are the source's system emails (double opt-in, unsubscribe,
	// bounce): ReputeCast sends its own.
	Skipped int `json:"skipped"`
}

func (c *ImportTemplateCounts) String() string {
	return fmt.Sprintf("%d transactional, %d saved, %d skipped", c.Transactional, c.Saved, c.Skipped)
}
