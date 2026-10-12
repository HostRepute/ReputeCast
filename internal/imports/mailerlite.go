package imports

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MailerLite reads groups, fields and subscribers through the MailerLite
// API (connect.mailerlite.com, read-only).
type MailerLite struct{ c *apiClient }

// NewMailerLite builds a client from an API token; base overrides the API URL.
func NewMailerLite(token, base string, hc *http.Client) (*MailerLite, error) {
	key := strings.TrimSpace(token)
	if key == "" {
		return nil, ErrNoKey
	}
	if base == "" {
		base = "https://connect.mailerlite.com/api"
	}
	return &MailerLite{c: newAPIClient("MailerLite", base, hc, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+key) })}, nil
}

// Connect implements Source.
func (m *MailerLite) Connect(ctx context.Context) (Connection, error) {
	out := Connection{Account: "MailerLite account", Lists: []SourceList{}, CheckedAt: time.Now()}
	fields, err := m.Fields(ctx, nil)
	if err != nil {
		return Connection{}, err
	}
	for page := 1; ; page++ {
		var res struct {
			Data []struct {
				ID                any    `json:"id"`
				Name              string `json:"name"`
				ActiveCount       int64  `json:"active_count"`
				UnsubscribedCount int64  `json:"unsubscribed_count"`
				UnconfirmedCount  int64  `json:"unconfirmed_count"`
				BouncedCount      int64  `json:"bounced_count"`
				JunkCount         int64  `json:"junk_count"`
				CreatedAt         string `json:"created_at"`
			} `json:"data"`
			Meta struct {
				CurrentPage int `json:"current_page"`
				LastPage    int `json:"last_page"`
			} `json:"meta"`
		}
		if err := m.c.get(ctx, fmt.Sprintf("/groups?limit=100&page=%d", page), &res); err != nil {
			return Connection{}, err
		}
		for _, g := range res.Data {
			sl := SourceList{ID: valueString(g.ID), Name: g.Name, Subscribed: g.ActiveCount, Unsubscribed: g.UnsubscribedCount,
				Pending: g.UnconfirmedCount, Cleaned: g.BouncedCount, Complained: g.JunkCount, Fields: len(fields), TagExamples: []string{}}
			if t := parseTime(g.CreatedAt); t != nil {
				sl.Since = strconv.Itoa(t.Year())
			}
			out.Lists = append(out.Lists, sl)
		}
		if res.Meta.LastPage == 0 || page >= res.Meta.LastPage {
			break
		}
	}
	return out, nil
}

// Fields implements Source.
func (m *MailerLite) Fields(ctx context.Context, _ []string) ([]SourceField, error) {
	var res struct {
		Data []struct {
			Key  string `json:"key"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"data"`
	}
	if err := m.c.get(ctx, "/fields?limit=100", &res); err != nil {
		return nil, err
	}
	out := []SourceField{{Key: "email", Label: "email", Type: "email"}}
	for _, f := range res.Data {
		out = append(out, SourceField{Key: f.Key, Label: f.Key, Type: f.Type})
	}
	return out, nil
}

// Fetch implements Source.
func (m *MailerLite) Fetch(ctx context.Context, lists []string, emit func(Row) error) error {
	for _, l := range lists {
		cursor := ""
		for {
			path := "/groups/" + url.PathEscape(l) + "/subscribers?limit=1000"
			if cursor != "" {
				path += "&cursor=" + url.QueryEscape(cursor)
			}
			var res struct {
				Data []struct {
					Email        string         `json:"email"`
					Status       string         `json:"status"`
					SubscribedAt string         `json:"subscribed_at"`
					OptedInAt    string         `json:"opted_in_at"`
					OptinIP      string         `json:"optin_ip"`
					Fields       map[string]any `json:"fields"`
					OpensCount   int            `json:"opens_count"`
				} `json:"data"`
				Meta struct {
					NextCursor string `json:"next_cursor"`
				} `json:"meta"`
			}
			if err := m.c.get(ctx, path, &res); err != nil {
				return err
			}
			for _, s := range res.Data {
				row := Row{Email: s.Email, List: l, Data: map[string]string{"email": s.Email}, Tags: []string{},
					Engagement: Engagement{SignupAt: parseTime(s.SubscribedAt), OptinAt: parseTime(s.OptedInAt), OptinIP: s.OptinIP}}
				for k, v := range s.Fields {
					row.Data[k] = valueString(v)
				}
				switch s.Status {
				case "active":
					row.Status = StatusSubscribed
				case "unconfirmed":
					row.Status = StatusPending
				case "bounced":
					row.Status = StatusCleaned
				case "junk":
					row.Status = StatusComplained
				default:
					row.Status = StatusUnsubscribed
				}
				if err := emit(row); err != nil {
					return err
				}
			}
			if res.Meta.NextCursor == "" || len(res.Data) == 0 {
				break
			}
			cursor = res.Meta.NextCursor
		}
	}
	return nil
}
