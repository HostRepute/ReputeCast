package imports

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Listmonk reads lists and subscribers through a listmonk server's API
// with an API user and token (read-only).
type Listmonk struct{ c *apiClient }

// NewListmonk builds a client for a listmonk server.
func NewListmonk(baseURL, user, token string, hc *http.Client) (*Listmonk, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrNoKey
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, newSourceError(CodeBadResponse, 0, mListmonkAddress, nil)
	}
	if !strings.HasSuffix(base, "/api") {
		base += "/api"
	}
	cred := strings.TrimSpace(user) + ":" + strings.TrimSpace(token)
	return &Listmonk{c: newAPIClient("listmonk", base, hc, func(r *http.Request) { r.Header.Set("Authorization", "token "+cred) })}, nil
}

// Connect implements Source.
func (l *Listmonk) Connect(ctx context.Context) (ImportConnection, error) {
	var res struct {
		Data struct {
			Results []struct {
				ID                 int64            `json:"id"`
				Name               string           `json:"name"`
				Optin              string           `json:"optin"`
				CreatedAt          string           `json:"created_at"`
				SubscriberStatuses map[string]int64 `json:"subscriber_statuses"`
			} `json:"results"`
		} `json:"data"`
	}
	if err := l.c.get(ctx, "/lists?per_page=all", &res); err != nil {
		return ImportConnection{}, err
	}
	out := ImportConnection{Account: "listmonk", BaseURL: strings.TrimSuffix(l.c.base, "/api"), Lists: []ImportSourceList{}, CheckedAt: time.Now()}
	for _, r := range res.Data.Results {
		sl := ImportSourceList{ID: strconv.FormatInt(r.ID, 10), Name: r.Name, DoubleOptIn: r.Optin == "double", TagExamples: []string{},
			Unsubscribed: r.SubscriberStatuses["unsubscribed"]}
		if sl.DoubleOptIn {
			sl.Subscribed, sl.Pending = r.SubscriberStatuses["confirmed"], r.SubscriberStatuses["unconfirmed"]
		} else {
			sl.Subscribed = r.SubscriberStatuses["confirmed"] + r.SubscriberStatuses["unconfirmed"]
		}
		if t := parseTime(r.CreatedAt); t != nil {
			sl.Since = strconv.Itoa(t.Year())
		}
		out.Lists = append(out.Lists, sl)
	}
	return out, nil
}

type lmSubscriber struct {
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Attribs   map[string]any `json:"attribs"`
	CreatedAt string         `json:"created_at"`
	Lists     []struct {
		ID                 int64  `json:"id"`
		SubscriptionStatus string `json:"subscription_status"`
		Optin              string `json:"optin"`
	} `json:"lists"`
}

// Fields implements Source: email, name and the attribute keys found on
// the first page of subscribers.
func (l *Listmonk) Fields(ctx context.Context, lists []string) ([]SourceField, error) {
	out := []SourceField{{Key: "email", Label: "email", Type: "email"}, {Key: "name", Label: "name", Type: "text"}}
	keys := map[string]bool{}
	for _, id := range lists {
		var res struct {
			Data struct {
				Results []lmSubscriber `json:"results"`
			} `json:"data"`
		}
		if err := l.c.get(ctx, fmt.Sprintf("/subscribers?list_id=%s&per_page=100&page=1", id), &res); err != nil {
			return nil, err
		}
		for _, s := range res.Data.Results {
			for k := range s.Attribs {
				keys[k] = true
			}
		}
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		out = append(out, SourceField{Key: "attribs." + k, Label: k, Type: "text"})
	}
	return out, nil
}

// Fetch implements Source.
func (l *Listmonk) Fetch(ctx context.Context, lists []string, emit func(Row) error) error {
	for _, id := range lists {
		listID, _ := strconv.ParseInt(id, 10, 64)
		for page := 1; ; page++ {
			var res struct {
				Data struct {
					Results []lmSubscriber `json:"results"`
					Total   int            `json:"total"`
					PerPage int            `json:"per_page"`
				} `json:"data"`
			}
			if err := l.c.get(ctx, fmt.Sprintf("/subscribers?list_id=%s&per_page=1000&page=%d", id, page), &res); err != nil {
				return err
			}
			for _, s := range res.Data.Results {
				row := Row{Email: s.Email, List: id, Status: StatusSubscribed, Data: map[string]string{"email": s.Email, "name": s.Name}, Tags: []string{},
					Engagement: Engagement{SignupAt: parseTime(s.CreatedAt)}}
				for k, v := range s.Attribs {
					row.Data["attribs."+k] = valueString(v)
				}
				for _, sl := range s.Lists {
					if sl.ID != listID {
						continue
					}
					switch sl.SubscriptionStatus {
					case "unsubscribed":
						row.Status = StatusUnsubscribed
					case "unconfirmed":
						if sl.Optin == "double" {
							row.Status = StatusPending
						}
					}
				}
				if s.Status == "blocklisted" {
					row.Status = StatusUnsubscribed
				}
				if err := emit(row); err != nil {
					return err
				}
			}
			if len(res.Data.Results) == 0 || page*1000 >= res.Data.Total {
				break
			}
		}
	}
	return nil
}
