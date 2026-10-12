package imports

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Brevo reads lists, contact attributes and the blocklist through the
// Brevo API v3 (read-only).
type Brevo struct{ c *apiClient }

// NewBrevo builds a client from an API key; base overrides the API URL.
func NewBrevo(apiKey, base string, hc *http.Client) (*Brevo, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, ErrNoKey
	}
	if base == "" {
		base = "https://api.brevo.com/v3"
	}
	return &Brevo{c: newAPIClient("Brevo", base, hc, func(r *http.Request) { r.Header.Set("api-key", key) })}, nil
}

// Connect implements Source.
func (b *Brevo) Connect(ctx context.Context) (Connection, error) {
	var acct struct {
		CompanyName string `json:"companyName"`
		Email       string `json:"email"`
	}
	if err := b.c.get(ctx, "/account", &acct); err != nil {
		return Connection{}, err
	}
	out := Connection{Account: acct.CompanyName, Lists: []SourceList{}, CheckedAt: time.Now()}
	if out.Account == "" {
		out.Account = acct.Email
	}
	fields, err := b.Fields(ctx, nil)
	if err != nil {
		return Connection{}, err
	}
	for offset := 0; ; offset += 50 {
		var page struct {
			Lists []struct {
				ID                int64 `json:"id"`
				Name              string
				TotalBlacklisted  int64 `json:"totalBlacklisted"`
				TotalSubscribers  int64 `json:"totalSubscribers"`
				UniqueSubscribers int64 `json:"uniqueSubscribers"`
			} `json:"lists"`
			Count int `json:"count"`
		}
		if err := b.c.get(ctx, fmt.Sprintf("/contacts/lists?limit=50&offset=%d", offset), &page); err != nil {
			return Connection{}, err
		}
		for _, l := range page.Lists {
			total := l.UniqueSubscribers
			if total == 0 {
				total = l.TotalSubscribers
			}
			out.Lists = append(out.Lists, SourceList{ID: strconv.FormatInt(l.ID, 10), Name: l.Name, Subscribed: max(total-l.TotalBlacklisted, 0),
				Unsubscribed: l.TotalBlacklisted, Fields: len(fields), TagExamples: []string{}})
		}
		if offset+50 >= page.Count || len(page.Lists) == 0 {
			break
		}
	}
	return out, nil
}

// Fields implements Source: the email plus normal contact attributes.
func (b *Brevo) Fields(ctx context.Context, _ []string) ([]SourceField, error) {
	var attrs struct {
		Attributes []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
			Type     string `json:"type"`
		} `json:"attributes"`
	}
	if err := b.c.get(ctx, "/contacts/attributes", &attrs); err != nil {
		return nil, err
	}
	out := []SourceField{{Key: "EMAIL", Label: "EMAIL", Type: "email"}}
	for _, a := range attrs.Attributes {
		if a.Category != "normal" && a.Category != "" {
			continue
		}
		typ := a.Type
		switch typ {
		case "float", "id":
			typ = "number"
		case "":
			typ = "text"
		}
		out = append(out, SourceField{Key: a.Name, Label: a.Name, Type: typ})
	}
	return out, nil
}

// Fetch implements Source. The blocklist says why an address is blocked:
// hard bounces become cleaned, spam reports complained, the rest unsubscribed.
func (b *Brevo) Fetch(ctx context.Context, lists []string, emit func(Row) error) error {
	blocked := map[string]string{}
	for offset := 0; ; offset += 100 {
		var page struct {
			Contacts []struct {
				Email  string `json:"email"`
				Reason struct {
					Code string `json:"code"`
				} `json:"reason"`
			} `json:"contacts"`
			Count int `json:"count"`
		}
		if err := b.c.get(ctx, fmt.Sprintf("/smtp/blockedContacts?limit=100&offset=%d", offset), &page); err != nil {
			return err
		}
		for _, c := range page.Contacts {
			st := StatusUnsubscribed
			switch c.Reason.Code {
			case "hardBounce":
				st = StatusCleaned
			case "spam", "contactFlaggedAsSpam":
				st = StatusComplained
			}
			blocked[strings.ToLower(c.Email)] = st
		}
		if offset+100 >= page.Count || len(page.Contacts) == 0 {
			break
		}
	}
	for _, l := range lists {
		listID, _ := strconv.ParseInt(l, 10, 64)
		for offset := 0; ; offset += 500 {
			var page struct {
				Contacts []struct {
					Email            string         `json:"email"`
					EmailBlacklisted bool           `json:"emailBlacklisted"`
					CreatedAt        string         `json:"createdAt"`
					ModifiedAt       string         `json:"modifiedAt"`
					ListUnsubscribed []int64        `json:"listUnsubscribed"`
					Attributes       map[string]any `json:"attributes"`
				} `json:"contacts"`
				Count int `json:"count"`
			}
			if err := b.c.get(ctx, fmt.Sprintf("/contacts/lists/%s/contacts?limit=500&offset=%d", l, offset), &page); err != nil {
				return err
			}
			for _, c := range page.Contacts {
				row := Row{Email: c.Email, List: l, Status: StatusSubscribed, Data: map[string]string{"EMAIL": c.Email}, Tags: []string{},
					Engagement: Engagement{SignupAt: parseTime(c.CreatedAt), Changed: parseTime(c.ModifiedAt)}}
				for k, v := range c.Attributes {
					row.Data[k] = valueString(v)
				}
				for _, u := range c.ListUnsubscribed {
					if u == listID {
						row.Status = StatusUnsubscribed
					}
				}
				if st, ok := blocked[strings.ToLower(c.Email)]; ok {
					row.Status = st
				} else if c.EmailBlacklisted {
					row.Status = StatusUnsubscribed
				}
				if err := emit(row); err != nil {
					return err
				}
			}
			if offset+500 >= page.Count || len(page.Contacts) == 0 {
				break
			}
		}
	}
	return nil
}
