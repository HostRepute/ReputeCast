package imports

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Keila reads contacts through a Keila server's API v1 (read-only). Keila
// has no lists, so everything comes in as one "All contacts" list.
type Keila struct{ c *apiClient }

// KeilaAllContacts is the pseudo-list id of a Keila import.
const KeilaAllContacts = "all"

// NewKeila builds a client for a Keila server (https://app.keila.io or self-hosted).
func NewKeila(baseURL, key string, hc *http.Client) (*Keila, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrNoKey
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "https://app.keila.io"
	}
	if !strings.HasSuffix(base, "/api/v1") {
		base += "/api/v1"
	}
	return &Keila{c: newAPIClient("Keila", base, hc, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+strings.TrimSpace(key)) })}, nil
}

type keilaContact struct {
	Email      string         `json:"email"`
	FirstName  string         `json:"first_name"`
	LastName   string         `json:"last_name"`
	Status     string         `json:"status"`
	Data       map[string]any `json:"data"`
	InsertedAt string         `json:"inserted_at"`
}

type keilaPage struct {
	Data []keilaContact `json:"data"`
	Meta struct {
		Page      int `json:"page"`
		PageCount int `json:"page_count"`
	} `json:"meta"`
}

// Connect implements Source.
func (k *Keila) Connect(ctx context.Context) (Connection, error) {
	var page keilaPage
	if err := k.c.get(ctx, "/contacts?paginate[page]=0&paginate[page_size]=1", &page); err != nil {
		return Connection{}, err
	}
	var segs struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := k.c.get(ctx, "/segments", &segs); err != nil {
		return Connection{}, err
	}
	examples := []string{}
	for _, s := range segs.Data {
		examples = append(examples, s.Name)
	}
	fields, err := k.Fields(ctx, nil)
	if err != nil {
		return Connection{}, err
	}
	return Connection{Account: "Keila project", BaseURL: strings.TrimSuffix(k.c.base, "/api/v1"), CheckedAt: time.Now(), Lists: []SourceList{{
		ID: KeilaAllContacts, Name: "All contacts", Subscribed: int64(page.Meta.PageCount), Fields: len(fields), Tags: len(examples), TagExamples: examples,
	}}}, nil
}

// Fields implements Source: built-in fields plus the custom data keys of
// the first page.
func (k *Keila) Fields(ctx context.Context, _ []string) ([]SourceField, error) {
	var page keilaPage
	if err := k.c.get(ctx, "/contacts?paginate[page]=0&paginate[page_size]=100", &page); err != nil {
		return nil, err
	}
	out := []SourceField{{Key: "email", Label: "email", Type: "email"}, {Key: "first_name", Label: "first_name", Type: "text"}, {Key: "last_name", Label: "last_name", Type: "text"}}
	keys := map[string]bool{}
	for _, c := range page.Data {
		for key := range c.Data {
			keys[key] = true
		}
	}
	names := make([]string, 0, len(keys))
	for key := range keys {
		names = append(names, key)
	}
	sort.Strings(names)
	for _, n := range names {
		out = append(out, SourceField{Key: "data." + n, Label: n, Type: "text"})
	}
	return out, nil
}

// Fetch implements Source.
func (k *Keila) Fetch(ctx context.Context, _ []string, emit func(Row) error) error {
	for p := 0; ; p++ {
		var page keilaPage
		if err := k.c.get(ctx, fmt.Sprintf("/contacts?paginate[page]=%d&paginate[page_size]=500", p), &page); err != nil {
			return err
		}
		for _, c := range page.Data {
			row := Row{Email: c.Email, List: KeilaAllContacts, Status: StatusSubscribed,
				Data: map[string]string{"email": c.Email, "first_name": c.FirstName, "last_name": c.LastName}, Tags: []string{},
				Engagement: Engagement{SignupAt: parseTime(c.InsertedAt)}}
			for key, v := range c.Data {
				row.Data["data."+key] = valueString(v)
			}
			switch c.Status {
			case "unsubscribed":
				row.Status = StatusUnsubscribed
			case "unreachable":
				row.Status = StatusCleaned
			}
			if err := emit(row); err != nil {
				return err
			}
		}
		if len(page.Data) == 0 || p+1 >= page.Meta.PageCount {
			break
		}
	}
	return nil
}
