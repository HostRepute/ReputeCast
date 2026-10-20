package texting

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
)

// Audience is an SMS campaign's audience: lists and segments, stored like
// an email campaign's.
type Audience struct {
	Lists    []string `json:"lists,omitempty"`
	Segments []string `json:"segments,omitempty"`
}

// parseAudience reads a stored audience.
func parseAudience(raw json.RawMessage) Audience {
	var a Audience
	_ = json.Unmarshal(raw, &a)
	return a
}

func uuids(ss []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ss))
	for _, s := range ss {
		if id, err := uuid.Parse(s); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// Recipient is one person a text can go to.
type Recipient struct {
	ContactID  uuid.UUID
	ListID     uuid.UUID
	Phone      string
	Country    string
	FirstName  string
	LastName   string
	Email      string
	Attributes map[string]any
	Zone       string
	ZoneSource phone.ZoneSource
}

// Funnel is how an audience narrows to the people who can receive texts.
type Funnel struct {
	// Base is everyone on the audience's lists and segments.
	Base int64
	// NoConsent have no number, or no SMS consent on a list of the
	// audience (any list for a segment): email sign-ups never count.
	NoConsent int64
	// Suppressed have consent but their number is on the SMS suppression
	// list, by reason.
	Suppressed         int64
	SuppressedByReason map[string]int64
	// Shared are a second contact with a number already in the audience:
	// a phone gets one text.
	Shared int64
	// CanReceive are the recipients.
	CanReceive int64
	Recipients []Recipient
	// Assumed is how many recipients' time zone comes from their number.
	Assumed       int64
	AssumedByArea int64
}

// funnel resolves an audience.
func (s *Service) funnel(ctx context.Context, ws *store.Workspace, a Audience) (Funnel, error) {
	f := Funnel{SuppressedByReason: map[string]int64{}, Recipients: []Recipient{}}
	lists := uuids(a.Lists)
	segContacts := []uuid.UUID{}
	for _, sid := range uuids(a.Segments) {
		if s.Segments == nil {
			break
		}
		ids, err := s.Segments.ContactIDs(ctx, ws.ID, sid)
		if err != nil {
			return f, err
		}
		segContacts = append(segContacts, ids...)
	}
	if len(lists) == 0 && len(segContacts) == 0 {
		return f, nil
	}
	rows, err := s.Pool.Query(ctx, `
		WITH live AS (SELECT id FROM lists WHERE workspace_id = $1 AND deleted_at IS NULL AND id = ANY($2::uuid[])),
		base AS (
			SELECT lm.contact_id FROM list_memberships lm JOIN live ON live.id = lm.list_id
			UNION SELECT cc.contact_id FROM channel_consents cc JOIN live ON live.id = cc.list_id WHERE cc.channel = 'sms'
			UNION SELECT unnest($3::uuid[])
		)
		SELECT c.id, c.phone, c.phone_country, c.timezone, c.first_name, c.last_name, c.email, c.attributes,
			(SELECT cc.list_id FROM channel_consents cc
			 WHERE cc.contact_id = c.id AND cc.channel = 'sms' AND cc.status = 'subscribed'
			   AND (cc.list_id IN (SELECT id FROM live) OR cardinality($3::uuid[]) > 0)
			 ORDER BY cc.changed_at LIMIT 1) AS consent_list
		FROM base b JOIN contacts c ON c.id = b.contact_id AND c.workspace_id = $1 AND c.erased_at IS NULL
		ORDER BY c.created_at, c.id`, ws.ID, lists, segContacts)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	candidates := []Recipient{}
	for rows.Next() {
		var r Recipient
		var attrs []byte
		var list *uuid.UUID
		if err := rows.Scan(&r.ContactID, &r.Phone, &r.Country, &r.Zone, &r.FirstName, &r.LastName, &r.Email, &attrs, &list); err != nil {
			return f, err
		}
		f.Base++
		if list == nil || r.Phone == "" {
			f.NoConsent++
			continue
		}
		r.ListID = *list
		_ = json.Unmarshal(attrs, &r.Attributes)
		candidates = append(candidates, r)
	}
	if err := rows.Err(); err != nil {
		return f, err
	}
	phones := make([]string, 0, len(candidates))
	for _, r := range candidates {
		phones = append(phones, r.Phone)
	}
	sup, err := s.Q.ChanSuppressedPhones(ctx, store.ChanSuppressedPhonesParams{WorkspaceID: ws.ID, Channel: "sms", Phones: phones})
	if err != nil {
		return f, err
	}
	suppressed := map[string]string{}
	for _, x := range sup {
		suppressed[x.Phone] = x.Reason
	}
	seen := map[string]bool{}
	for _, r := range candidates {
		if reason, ok := suppressed[r.Phone]; ok {
			f.Suppressed++
			f.SuppressedByReason[reason]++
			continue
		}
		if seen[r.Phone] {
			f.Shared++
			continue
		}
		seen[r.Phone] = true
		if r.Country == "" {
			if n, err := phone.Parse(r.Phone, ""); err == nil {
				r.Country = n.Country
			}
		}
		r.Zone, r.ZoneSource = phone.RecipientZone(r.Zone, r.Phone)
		if r.ZoneSource != phone.ZoneContact {
			f.Assumed++
			if r.ZoneSource == phone.ZoneAreaCode {
				f.AssumedByArea++
			}
		}
		f.Recipients = append(f.Recipients, r)
	}
	f.CanReceive = int64(len(f.Recipients))
	return f, nil
}

// smsListsCounts counts, per country, the people who can receive texts on
// any list (the routes table's "On SMS lists").
func (s *Service) smsListsCounts(ctx context.Context, ws *store.Workspace) (map[string]int64, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT c.phone_country, count(DISTINCT c.phone)::bigint
		FROM contacts c
		WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.phone <> ''
		  AND EXISTS (SELECT 1 FROM channel_consents cc JOIN lists l ON l.id = cc.list_id AND l.deleted_at IS NULL
		              WHERE cc.contact_id = c.id AND cc.channel = 'sms' AND cc.status = 'subscribed')
		  AND NOT EXISTS (SELECT 1 FROM channel_suppressions s WHERE s.workspace_id = c.workspace_id AND s.channel = 'sms'
		                  AND s.phone = c.phone AND s.removed_at IS NULL)
		GROUP BY c.phone_country`, ws.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var c string
		var n int64
		if err := rows.Scan(&c, &n); err != nil {
			return nil, err
		}
		out[c] = n
	}
	return out, rows.Err()
}

// assumedZones counts the people on SMS lists whose time zone would come
// from their number (no time zone on the contact).
func (s *Service) assumedZones(ctx context.Context, ws *store.Workspace) (int64, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `
		SELECT count(DISTINCT c.id)::bigint FROM contacts c
		WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.phone <> '' AND c.timezone = ''
		  AND EXISTS (SELECT 1 FROM channel_consents cc WHERE cc.contact_id = c.id AND cc.channel = 'sms' AND cc.status = 'subscribed')`, ws.ID).Scan(&n)
	return n, err
}

// listNames names lists by id.
func (s *Service) listNames(ctx context.Context, ws *store.Workspace, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name FROM lists WHERE workspace_id = $1 AND id = ANY($2::uuid[])`, ws.ID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var n string
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// audienceNames names an audience's lists and segments, in order.
func (s *Service) audienceNames(ctx context.Context, ws *store.Workspace, a Audience) ([]string, error) {
	out := []string{}
	if ids := uuids(a.Lists); len(ids) > 0 {
		names, err := s.listNames(ctx, ws, ids)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if n, ok := names[id]; ok {
				out = append(out, n)
			}
		}
	}
	if ids := uuids(a.Segments); len(ids) > 0 {
		rows, err := s.Pool.Query(ctx, `SELECT id, name FROM segments WHERE workspace_id = $1 AND id = ANY($2::uuid[])`, ws.ID, ids)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		names := map[uuid.UUID]string{}
		for rows.Next() {
			var id uuid.UUID
			var n string
			if err := rows.Scan(&id, &n); err != nil {
				return nil, err
			}
			names[id] = n
		}
		for _, id := range ids {
			if n, ok := names[id]; ok && !slices.Contains(out, n) {
				out = append(out, n)
			}
		}
	}
	return out, nil
}
