package imports

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Mailchimp reads audiences, merge fields, tags and members through the
// Marketing API v3 (read-only: nothing is written to Mailchimp).
type Mailchimp struct {
	c  *apiClient
	DC string
}

// mailchimpDC is a Mailchimp data center ("us21"): the key's suffix becomes
// part of the API host, so nothing else may pass as one.
var mailchimpDC = regexp.MustCompile(`^[a-z]{2}[0-9]{1,2}$`)

// NewMailchimp builds a client from an API key ("…-us21"); base overrides
// the API URL (tests).
func NewMailchimp(apiKey, base string, hc *http.Client) (*Mailchimp, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, ErrNoKey
	}
	_, dc, ok := strings.Cut(key, "-")
	if !ok || !mailchimpDC.MatchString(dc) {
		return nil, &SourceError{Code: CodeAuthFailed, Message: "Mailchimp keys end with the data center, like xxxxxxxx-us21. Copy the whole key from Account › Extras › API keys."}
	}
	if base == "" {
		base = "https://" + dc + ".api.mailchimp.com/3.0"
	}
	m := &Mailchimp{DC: dc}
	m.c = newAPIClient("Mailchimp", base, hc, func(r *http.Request) { r.SetBasicAuth("reputecast", key) })
	return m, nil
}

// Connect implements Source.
func (m *Mailchimp) Connect(ctx context.Context) (ImportConnection, error) {
	var acct struct {
		AccountName string `json:"account_name"`
	}
	if err := m.c.get(ctx, "/", &acct); err != nil {
		if se := (*SourceError)(nil); errors.As(err, &se) && se.Code == CodeAuthFailed {
			said := ""
			if se.Reason != "" {
				said = fmt.Sprintf(": %q", se.Reason)
			}
			se.Message = fmt.Sprintf(`Mailchimp rejected the API key%s from the %s data center. Keys look like xxxxxxxx-%s; the suffix must match the data center the account lives in.`, said, m.DC, m.DC)
		}
		return ImportConnection{}, err
	}
	var lists struct {
		Lists []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DoubleOptin bool   `json:"double_optin"`
			DateCreated string `json:"date_created"`
			Stats       struct {
				MemberCount      int64 `json:"member_count"`
				UnsubscribeCount int64 `json:"unsubscribe_count"`
				CleanedCount     int64 `json:"cleaned_count"`
				MergeFieldCount  int   `json:"merge_field_count"`
			} `json:"stats"`
		} `json:"lists"`
	}
	if err := m.c.get(ctx, "/lists?count=1000&fields=lists.id,lists.name,lists.stats,lists.double_optin,lists.date_created", &lists); err != nil {
		return ImportConnection{}, err
	}
	out := ImportConnection{Account: acct.AccountName, DataCenter: m.DC, Lists: []ImportSourceList{}, CheckedAt: time.Now()}
	for _, l := range lists.Lists {
		sl := ImportSourceList{ID: l.ID, Name: l.Name, Subscribed: l.Stats.MemberCount, Unsubscribed: l.Stats.UnsubscribeCount,
			Cleaned: l.Stats.CleanedCount, DoubleOptIn: l.DoubleOptin, TagExamples: []string{}}
		if t := parseTime(l.DateCreated); t != nil {
			sl.Since = strconv.Itoa(t.Year())
		}
		for _, st := range []string{"pending", "transactional"} {
			var n struct {
				TotalItems int64 `json:"total_items"`
			}
			if err := m.c.get(ctx, "/lists/"+url.PathEscape(l.ID)+"/members?status="+st+"&count=1&fields=total_items", &n); err != nil {
				return ImportConnection{}, err
			}
			if st == "pending" {
				sl.Pending = n.TotalItems
			} else {
				sl.Transactional = n.TotalItems
			}
		}
		var abuse struct {
			TotalItems int64 `json:"total_items"`
		}
		if err := m.c.get(ctx, "/lists/"+url.PathEscape(l.ID)+"/abuse-reports?count=1&fields=total_items", &abuse); err != nil {
			return ImportConnection{}, err
		}
		sl.Complained = abuse.TotalItems
		fields, err := m.Fields(ctx, []string{l.ID})
		if err != nil {
			return ImportConnection{}, err
		}
		sl.Fields = len(fields)
		tags, err := m.tags(ctx, l.ID)
		if err != nil {
			return ImportConnection{}, err
		}
		sl.Tags = len(tags)
		sl.TagExamples = tags
		out.Lists = append(out.Lists, sl)
	}
	return out, nil
}

func (m *Mailchimp) tags(ctx context.Context, list string) ([]string, error) {
	var segs struct {
		Segments []struct {
			Name        string `json:"name"`
			MemberCount int64  `json:"member_count"`
		} `json:"segments"`
	}
	if err := m.c.get(ctx, "/lists/"+url.PathEscape(list)+"/segments?type=static&count=1000&fields=segments.name,segments.member_count", &segs); err != nil {
		return nil, err
	}
	sort.SliceStable(segs.Segments, func(i, j int) bool { return segs.Segments[i].MemberCount > segs.Segments[j].MemberCount })
	out := make([]string, 0, len(segs.Segments))
	for _, s := range segs.Segments {
		out = append(out, s.Name)
	}
	return out, nil
}

// Fields implements Source: EMAIL, the merge fields, and the member
// properties ReputeCast can use (timezone, opt-in evidence, tags).
func (m *Mailchimp) Fields(ctx context.Context, lists []string) ([]SourceField, error) {
	out := []SourceField{{Key: "EMAIL", Label: "EMAIL", Type: "email"}}
	seen := map[string]bool{"EMAIL": true}
	for _, l := range lists {
		var mf struct {
			MergeFields []struct {
				Tag  string `json:"tag"`
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"merge_fields"`
		}
		if err := m.c.get(ctx, "/lists/"+url.PathEscape(l)+"/merge-fields?count=100", &mf); err != nil {
			return nil, err
		}
		for _, f := range mf.MergeFields {
			if seen[f.Tag] {
				continue
			}
			seen[f.Tag] = true
			typ := f.Type
			switch typ {
			case "dropdown", "radio", "zip", "imageurl":
				typ = "text"
			}
			out = append(out, SourceField{Key: f.Tag, Label: f.Tag, Type: typ})
		}
	}
	// Address and phone fields go last, as they are skipped by default.
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i].Type) < rank(out[j].Type) })
	extra := []SourceField{{Key: "TIMEZONE", Label: "TIMEZONE", Type: "timezone"}, {Key: "OPTIN", Label: "OPTIN_TIME · OPTIN_IP", Type: "consent"}, {Key: "TAGS", Label: "TAGS", Type: "tags"}}
	n := len(out)
	for i, f := range out {
		if f.Type == "address" || f.Type == "phone" {
			n = i
			break
		}
	}
	return append(append(append([]SourceField{}, out[:n]...), extra...), out[n:]...), nil
}

func rank(t string) int {
	switch t {
	case "email":
		return 0
	case "address", "phone":
		return 2
	}
	return 1
}

type mcMember struct {
	EmailAddress    string                  `json:"email_address"`
	Status          string                  `json:"status"`
	MergeFields     map[string]any          `json:"merge_fields"`
	Tags            []struct{ Name string } `json:"tags"`
	TimestampOpt    string                  `json:"timestamp_opt"`
	IPOpt           string                  `json:"ip_opt"`
	TimestampSignup string                  `json:"timestamp_signup"`
	MemberRating    int                     `json:"member_rating"`
	LastChanged     string                  `json:"last_changed"`
	Location        struct {
		Timezone string `json:"timezone"`
	} `json:"location"`
}

// Fetch implements Source. Members who reported abuse come from the abuse
// reports and are marked complained.
func (m *Mailchimp) Fetch(ctx context.Context, lists []string, emit func(Row) error) error {
	for _, l := range lists {
		complained := map[string]bool{}
		for offset := 0; ; offset += 1000 {
			var ab struct {
				Reports []struct {
					EmailAddress string `json:"email_address"`
				} `json:"abuse_reports"`
				TotalItems int `json:"total_items"`
			}
			if err := m.c.get(ctx, fmt.Sprintf("/lists/%s/abuse-reports?count=1000&offset=%d", url.PathEscape(l), offset), &ab); err != nil {
				return err
			}
			for _, r := range ab.Reports {
				complained[strings.ToLower(r.EmailAddress)] = true
			}
			if offset+1000 >= ab.TotalItems || len(ab.Reports) == 0 {
				break
			}
		}
		for offset := 0; ; offset += 1000 {
			var page struct {
				Members    []mcMember `json:"members"`
				TotalItems int        `json:"total_items"`
			}
			path := fmt.Sprintf("/lists/%s/members?count=1000&offset=%d&fields=members.email_address,members.status,members.merge_fields,members.tags,members.timestamp_opt,members.ip_opt,members.timestamp_signup,members.member_rating,members.last_changed,members.location,total_items", url.PathEscape(l), offset)
			if err := m.c.get(ctx, path, &page); err != nil {
				return err
			}
			for _, mem := range page.Members {
				row := Row{Email: mem.EmailAddress, List: l, Data: map[string]string{"EMAIL": mem.EmailAddress}, Tags: []string{}}
				for k, v := range mem.MergeFields {
					row.Data[k] = valueString(v)
				}
				for _, t := range mem.Tags {
					row.Tags = append(row.Tags, t.Name)
				}
				if mem.Location.Timezone != "" {
					row.Data["TIMEZONE"] = mem.Location.Timezone
				}
				row.Engagement = Engagement{OptinAt: parseTime(mem.TimestampOpt), OptinIP: mem.IPOpt, SignupAt: parseTime(mem.TimestampSignup),
					Changed: parseTime(mem.LastChanged), Rating: mem.MemberRating}
				if row.Engagement.OptinAt != nil {
					row.Data["OPTIN_TIME"] = row.Engagement.OptinAt.UTC().Format("2006-01-02 15:04")
				}
				if mem.IPOpt != "" {
					row.Data["OPTIN_IP"] = mem.IPOpt
				}
				switch mem.Status {
				case "subscribed":
					row.Status = StatusSubscribed
				case "unsubscribed":
					row.Status = StatusUnsubscribed
				case "cleaned":
					row.Status = StatusCleaned
				case "pending":
					row.Status = StatusPending
				case "transactional":
					row.Status = StatusTransactional
				default:
					row.Status = StatusUnsubscribed
				}
				if complained[strings.ToLower(mem.EmailAddress)] {
					row.Status = StatusComplained
				}
				if err := emit(row); err != nil {
					return err
				}
			}
			if offset+1000 >= page.TotalItems || len(page.Members) == 0 {
				break
			}
		}
	}
	return nil
}

// valueString flattens a merge field value; addresses become one line.
func valueString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	case map[string]any:
		parts := []string{}
		for _, k := range []string{"addr1", "addr2", "city", "state", "zip", "country"} {
			if s, ok := x[k].(string); ok && strings.TrimSpace(s) != "" {
				parts = append(parts, strings.TrimSpace(s))
			}
		}
		return strings.Join(parts, ", ")
	}
	return fmt.Sprint(v)
}
