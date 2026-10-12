package audience

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// ActivityProvider is the sending provider that delivered an email.
type ActivityProvider struct {
	Name    string `json:"name"`
	Adapter string `json:"adapter"`
}

// ActivityItem is one event on a contact's timeline: a delivery event (with
// the sending provider that handled it), an engagement event or a consent
// change.
type ActivityItem struct {
	ID       string    `json:"id"`
	Category string    `json:"category" enum:"delivery,engagement,consent"`
	Type     string    `json:"type" doc:"delivered, soft_bounce, open, click… or a consent action"`
	At       time.Time `json:"at"`
	// Subject is the email: "Weekly Digest #141", "Welcome series · step 2".
	Subject  string            `json:"subject"`
	Variant  string            `json:"variant,omitempty"`
	Provider *ActivityProvider `json:"provider,omitempty"`
	// Imported names the tool that sent the email before the migration ("Brevo").
	Imported   string `json:"imported,omitempty"`
	Failover   bool   `json:"failover"`
	Code       string `json:"code,omitempty" doc:"Provider code: 250 2.6.0, 452 4.2.2"`
	Diagnostic string `json:"diagnostic,omitempty"`
	RouteNote  string `json:"routeNote,omitempty"`
	Machine    bool   `json:"machine"`
	// Client is the mail client of an open ("Outlook on Windows").
	Client             string     `json:"client,omitempty"`
	AfterDeliveryMin   *int64     `json:"afterDeliveryMin,omitempty"`
	RetriedDeliveredAt *time.Time `json:"retriedDeliveredAt,omitempty"`
	URL                string     `json:"url,omitempty"`
	LinkLabel          string     `json:"linkLabel,omitempty"`
	// Consent events.
	List           string `json:"list,omitempty"`
	Source         string `json:"source,omitempty"`
	SourceLabel    string `json:"sourceLabel,omitempty"`
	Detail         string `json:"detail,omitempty"`
	Note           string `json:"note,omitempty"`
	IP             string `json:"ip,omitempty"`
	Actor          string `json:"actor,omitempty"`
	ConsentVersion string `json:"consentVersion,omitempty"`
}

// ActivityPage is one page of the timeline, newest first.
type ActivityPage struct {
	Items      []ActivityItem `json:"items"`
	Total      int64          `json:"total" doc:"Events matching the filter"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

var deliveryTypes = []string{"delivered", "deferred", "hard_bounce", "soft_bounce", "complaint", "rejected"}
var engagementTypes = []string{"open", "click", "unsubscribe"}

// Activity returns a contact's timeline. filter is all, deliveries,
// engagement or consent.
func (s *Service) Activity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, filter, cursor string, limit int) (ActivityPage, error) {
	if _, err := s.getContact(ctx, s.Q, ws, contactID); err != nil {
		return ActivityPage{}, err
	}
	if limit <= 0 || limit > 100 {
		limit = 8
	}
	types := append(append([]string{}, deliveryTypes...), engagementTypes...)
	withConsent := true
	switch filter {
	case "deliveries":
		types, withConsent = deliveryTypes, false
	case "engagement":
		types, withConsent = engagementTypes, false
	case "consent":
		types = []string{}
	}
	args := []any{contactID, types, withConsent}
	cur := ""
	if cursor != "" {
		at, id, err := decodeCursor(cursor)
		if err != nil {
			return ActivityPage{}, api.Invalid("cursor", "This page link expired. Reload the timeline.")
		}
		args = append(args, at, id)
		cur = " WHERE (ev.occurred_at, ev.id) < ($4, $5)"
	}
	union := `SELECT de.id, 'event'::text AS src, de.type AS kind, de.occurred_at,
		       coalesce(m.variant, '') AS variant, coalesce(m.failover, false) AS failover, coalesce(m.route_note, '') AS route_note,
		       m.delivered_at, coalesce(cp.name, '') AS campaign, coalesce(cp.origin, '') AS origin,
		       coalesce(a.name, '') AS automation, coalesce(m.automation_node, '') AS node,
		       coalesce(p.name, '') AS provider, coalesce(p.adapter, '') AS adapter,
		       de.original_code, de.diagnostic, de.machine, de.url, de.user_agent,
		       coalesce((SELECT l.label FROM campaign_links l WHERE l.campaign_id = cp.id AND l.url = de.url ORDER BY l.position LIMIT 1), '') AS link_label,
		       ''::text AS list, ''::text AS source, ''::text AS detail, ''::text AS note, NULL::inet AS ip, ''::text AS actor, ''::text AS version
		FROM delivery_events de
		LEFT JOIN messages m ON m.id = de.message_id
		LEFT JOIN campaigns cp ON cp.id = coalesce(de.campaign_id, m.campaign_id)
		LEFT JOIN automations a ON a.id = m.automation_id
		LEFT JOIN providers p ON p.id = coalesce(de.provider_id, m.provider_id)
		WHERE de.contact_id = $1 AND de.type = ANY ($2)
		UNION ALL
		SELECT ce.id, 'consent', ce.action, ce.occurred_at, '', false, '', NULL, '', '', '', '', '', '', '', '', false, '', ce.user_agent, '',
		       coalesce(l.name, ''), ce.source, ce.detail, ce.note, ce.ip, coalesce(u.name, ''), ce.consent_version
		FROM consent_events ce
		LEFT JOIN lists l ON l.id = ce.list_id
		LEFT JOIN users u ON u.id = ce.actor_id
		WHERE ce.contact_id = $1 AND $3`
	var out ActivityPage
	out.Items = []ActivityItem{}
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM ("+union+") ev", contactID, types, withConsent).Scan(&out.Total); err != nil {
		return out, err
	}
	args = append(args, limit+1)
	rows, err := s.Pool.Query(ctx, "SELECT * FROM ("+union+") ev"+cur+fmt.Sprintf(" ORDER BY ev.occurred_at DESC, ev.id DESC LIMIT $%d", len(args)), args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var it ActivityItem
		var id uuid.UUID
		var src, campaign, origin, automation, node, provider, adapter, userAgent, url string
		var delivered *time.Time
		var ip *netip.Addr
		if err := rows.Scan(&id, &src, &it.Type, &it.At, &it.Variant, &it.Failover, &it.RouteNote, &delivered, &campaign, &origin,
			&automation, &node, &provider, &adapter, &it.Code, &it.Diagnostic, &it.Machine, &url, &userAgent, &it.LinkLabel,
			&it.List, &it.Source, &it.Detail, &it.Note, &ip, &it.Actor, &it.ConsentVersion); err != nil {
			return out, err
		}
		it.ID = id.String()
		if src == "consent" {
			it.Category = "consent"
			it.SourceLabel = consentSourceLabel(it.Source)
			if strings.HasPrefix(it.Source, "import:") {
				it.SourceLabel = importerLabel(strings.TrimPrefix(it.Source, "import:"))
			}
			if ip != nil {
				it.IP = ip.String()
			}
		} else {
			it.Category = "delivery"
			for _, t := range engagementTypes {
				if t == it.Type {
					it.Category = "engagement"
				}
			}
			switch {
			case campaign != "":
				it.Subject = campaign
			case automation != "":
				it.Subject = automation
				if node != "" {
					it.Subject += " · " + node
				}
			default:
				it.Subject = "an email"
			}
			if provider != "" {
				it.Provider = &ActivityProvider{Name: provider, Adapter: adapter}
			} else if origin != "" {
				it.Imported = importerLabel(origin)
			}
			if it.Type == "open" {
				it.Client = MailClient(userAgent)
				if delivered != nil && it.At.After(*delivered) {
					m := int64(it.At.Sub(*delivered).Minutes())
					it.AfterDeliveryMin = &m
				}
			}
			if it.Type == "soft_bounce" && delivered != nil && delivered.After(it.At) {
				it.RetriedDeliveredAt = delivered
			}
			if url != "" {
				it.URL = strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
			}
		}
		out.Items = append(out.Items, it)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Items) > limit {
		last := out.Items[limit-1]
		out.Items = out.Items[:limit]
		out.NextCursor = encodeCursor(last.At, last.ID)
	}
	return out, nil
}

func encodeCursor(at time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(c string) (time.Time, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	ts, idStr, ok := strings.Cut(string(b), "|")
	if !ok {
		return time.Time{}, uuid.Nil, fmt.Errorf("bad cursor")
	}
	at, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(idStr)
	return at, id, err
}

// MailClient names the mail client behind an open's user agent.
func MailClient(ua string) string {
	u := strings.ToLower(ua)
	switch {
	case u == "":
		return ""
	case strings.Contains(u, "googleimageproxy"):
		return "Gmail"
	case strings.Contains(u, "outlook") || strings.Contains(u, "microsoft office"):
		switch {
		case strings.Contains(u, "windows"):
			return "Outlook on Windows"
		case strings.Contains(u, "mac"):
			return "Outlook on Mac"
		case strings.Contains(u, "iphone") || strings.Contains(u, "android"):
			return "Outlook mobile"
		}
		return "Outlook"
	case strings.Contains(u, "yahoomailproxy"):
		return "Yahoo Mail"
	case strings.Contains(u, "thunderbird"):
		return "Thunderbird"
	case strings.Contains(u, "iphone") || strings.Contains(u, "ipad"):
		return "Apple Mail on iPhone"
	case strings.Contains(u, "macintosh"):
		return "Apple Mail on Mac"
	case strings.Contains(u, "android"):
		return "Android"
	case strings.Contains(u, "windows"):
		return "Browser on Windows"
	}
	return "Other client"
}
