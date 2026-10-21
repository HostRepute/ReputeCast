package texting

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// testNumbers lists the test numbers with their last test (of a campaign
// when given).
func (s *Service) testNumbers(ctx context.Context, ws *store.Workspace, campaign *uuid.UUID) ([]SmsTestNumber, error) {
	rows, err := s.Q.SmsTestNumbers(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := make([]SmsTestNumber, 0, len(rows))
	for _, r := range rows {
		t := SmsTestNumber{ID: r.ID.String(), Name: r.UserName, Role: r.RoleName, Color: r.UserColor, Phone: r.Phone, Display: displayPhone(r.Phone)}
		if n, err := phone.Parse(r.Phone, ""); err == nil {
			t.Country = n.Country
		}
		if r.UserID != nil {
			t.UserID = r.UserID.String()
		}
		var m store.ChannelMessage
		q := `SELECT * FROM channel_messages WHERE workspace_id = $1 AND kind = 'test' AND recipient = $2`
		args := []any{ws.ID, r.Phone}
		if campaign != nil {
			q += ` AND campaign_id = $3`
			args = append(args, *campaign)
		}
		q += ` ORDER BY created_at DESC LIMIT 1`
		rs, err := s.Pool.Query(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		msgs, err := pgx.CollectRows(rs, pgx.RowToStructByPos[store.ChannelMessage])
		if err != nil {
			return nil, err
		}
		if len(msgs) > 0 {
			m = msgs[0]
			res := &SmsTestResult{MessageID: m.ID.String(), Status: m.Status, At: m.CreatedAt, Sender: senderShown(m.Sender), SenderID: !strings.HasPrefix(m.Sender, "+"),
				Delivered: m.DeliveredAt}
			if m.ProviderID != nil {
				if p, err := s.Q.SmsProviderAny(ctx, *m.ProviderID); err == nil {
					res.Provider = p.Name
				}
			}
			if m.DeliveredAt != nil && m.SentAt != nil {
				sec := m.DeliveredAt.Sub(*m.SentAt).Seconds()
				res.Seconds = &sec
			}
			t.LastTest = res
		}
		out = append(out, t)
	}
	return out, nil
}

func senderShown(v string) string {
	if strings.HasPrefix(v, "+") {
		return displayPhone(v)
	}
	return v
}

// SmsTestInput sends test texts.
type SmsTestInput struct {
	// Numbers are test numbers' ids; empty sends to every test number.
	Numbers []string `json:"numbers,omitempty" maxItems:"20"`
}

// SendTest texts the campaign's text to test numbers, each from the sender
// its country would get. Tests count toward the budget.
func (s *Service) SendTest(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SmsTestInput) ([]SmsTestNumber, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return nil, err
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	numbers, err := s.Q.SmsTestNumbers(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	if len(numbers) == 0 {
		return nil, api.Conflict("Add a test number in Routing & rules first.")
	}
	text := textOf(c.Body, g.rules)
	sent := 0
	for _, n := range numbers {
		if len(in.Numbers) > 0 && !contains(in.Numbers, n.ID.String()) {
			continue
		}
		first := strings.Fields(n.UserName)
		r := Recipient{Phone: n.Phone}
		if len(first) > 0 {
			r.FirstName = first[0]
		}
		if _, err := s.sendOne(ctx, ws, g, oneText{kind: "test", campaignID: &c.ID, recipient: r, text: text, campaignName: c.Name, by: p.UserID}); err != nil {
			return nil, err
		}
		sent++
	}
	if sent == 0 {
		return nil, api.Invalid("body.numbers", "Pick test numbers of this workspace.")
	}
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.test_sent", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
		After: map[string]any{"texts": sent}, Metadata: map[string]any{"channel": "sms"}}); err != nil {
		return nil, err
	}
	return s.testNumbers(ctx, ws, &c.ID)
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// oneText is a text sent at once: a test, a reply, a confirmation.
type oneText struct {
	kind         string
	campaignID   *uuid.UUID
	automationID *uuid.UUID
	contactID    *uuid.UUID
	listID       *uuid.UUID
	recipient    Recipient
	text         Text
	campaignName string
	// sender forces the sender (a reply goes from the number texted).
	sender *store.SmsSendersRow
	by     uuid.UUID
}

// sendOne sends a text now, through the route of the number's country, and
// logs it. Failures are logged on the message; only an unroutable number
// is an error.
func (s *Service) sendOne(ctx context.Context, ws *store.Workspace, g *graph, t oneText) (store.ChannelMessage, error) {
	n, err := phone.Parse(t.recipient.Phone, "")
	if err != nil {
		return store.ChannelMessage{}, api.Invalid("body.phone", "That isn't a phone number ReputeCast can text.")
	}
	sender := t.sender
	var d Decision
	if sender == nil {
		d = g.decide(n.Country)
		sender = d.Sender
		if sender != nil && g.down(sender.ProviderID) && d.Fallback != nil {
			sender = d.Fallback
		}
	} else if pr, ok := g.prices[sender.ProviderID][n.Country]; ok {
		d.Price = pr.Price
	}
	if sender == nil {
		return store.ChannelMessage{}, api.Err(http.StatusConflict, "no_sender", "No sender can text "+countryName(n.Country)+": add a route for it in Routing & rules.").
			WithMeta("country", n.Country)
	}
	if d.Price == 0 {
		if pr, ok := g.prices[sender.ProviderID][n.Country]; ok {
			d.Price = pr.Price
		}
	}
	id := uuid.Must(uuid.NewV7())
	domain := s.shortDomain(g.rules)
	body := t.text.Body
	links := []store.SmsLinkInsertParams{}
	now := s.now()
	if domain != "" && t.kind != "reply" && t.kind != "confirmation" {
		seed := id
		body = withShortLinks(body, domain, func(i int, l Link) string {
			code := LinkCode(seed, uuid.Nil, i)
			links = append(links, store.SmsLinkInsertParams{Code: code, WorkspaceID: ws.ID, MessageID: &id, CampaignID: t.campaignID, AutomationID: t.automationID,
				ContactID: t.contactID, Original: l.URL, Target: target(l.URL, utmCampaign(t.campaignName)), CreatedAt: now, ExpiresAt: now.AddDate(1, 0, 0)})
			return code
		})
	}
	full := Text{Prefix: t.text.Prefix, Body: render(body, t.recipient), OptOut: t.text.OptOut}.Full()
	cnt := sms.CountText(full)
	price := d.Price
	msg := store.ChanMessageInsertParams{ID: id, WorkspaceID: ws.ID, Channel: "sms", Kind: t.kind, CampaignID: t.campaignID, AutomationID: t.automationID,
		ContactID: t.contactID, ListID: t.listID, Recipient: n.E164, Country: n.Country, ProviderID: &sender.ProviderID, SenderID: &sender.ID, Sender: sender.Value,
		Body: &full, Units: int32(max(cnt.Segments, 1)), Encoding: string(cnt.Encoding), Price: &price, Cost: price * float64(max(cnt.Segments, 1)),
		Status: "sending", CreatedAt: now}
	if err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.ChanMessageInsert(ctx, msg); err != nil {
			return err
		}
		for _, l := range links {
			if err := q.SmsLinkInsert(ctx, l); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return store.ChannelMessage{}, err
	}
	prov, ok := g.providers[sender.ProviderID]
	if !ok {
		return store.ChannelMessage{}, errors.New("texting: the sender's provider is gone")
	}
	row, err := s.Q.ChanMessage(ctx, id)
	if err != nil {
		return row, err
	}
	s.deliver(ctx, ws, prov, row)
	return s.Q.ChanMessage(ctx, id)
}

// contextOf names what a number was sent when something happened to it.
func contextOf(kind string) string {
	switch kind {
	case "confirmation":
		return "Opt-in confirmation"
	case "test":
		return "Test text"
	}
	return ""
}

// event logs a message event.
func (s *Service) event(ctx context.Context, wsID, msgID uuid.UUID, kind, code string, detail map[string]any, machine bool, at time.Time) {
	if detail == nil {
		detail = map[string]any{}
	}
	_ = s.Q.ChanMessageEventInsert(ctx, store.ChanMessageEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, MessageID: msgID, Kind: kind,
		ProviderCode: code, Detail: jsonOf(detail), Machine: machine, OccurredAt: at})
}

// SmsPreviewInput asks for the phone preview.
type SmsPreviewInput struct {
	ContactID string `json:"contactId,omitempty" format:"uuid"`
	// Country shows the sender as that country sees it.
	Country string `json:"country,omitempty" pattern:"^[A-Z]{2}$"`
}

// SmsPreviewPart is a piece of the bubble: a locked part, text or a link.
type SmsPreviewPart struct {
	Kind string `json:"kind" enum:"fixed,text,link"`
	Text string `json:"text"`
}

// SmsPreview is the phone preview.
type SmsPreview struct {
	Sender     string           `json:"sender"`
	SenderKind string           `json:"senderKind"`
	Provider   string           `json:"provider"`
	CanReply   bool             `json:"canReply"`
	Country    string           `json:"country"`
	Contact    string           `json:"contact"`
	ContactID  string           `json:"contactId,omitempty"`
	Parts      []SmsPreviewPart `json:"parts"`
	Characters int              `json:"characters"`
	Segments   int              `json:"segments"`
	Encoding   string           `json:"encoding"`
	Price      float64          `json:"price"`
	// Skipped is why this country gets nothing (pending, no_route…).
	Skipped string `json:"skipped,omitempty"`
	// Arrives is when the person gets it (the campaign's schedule in their
	// zone); Zone is their time zone.
	Arrives *time.Time `json:"arrives,omitempty"`
	Zone    string     `json:"zone"`
	// Confirmation is the opt-in confirmation the person got before, for
	// the thread.
	Confirmation *SmsPreviewBubble `json:"confirmation,omitempty"`
}

// SmsPreviewBubble is an earlier text in the thread.
type SmsPreviewBubble struct {
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

// SmsPreviewContact is someone to preview as.
type SmsPreviewContact struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

// Preview renders the text for a recipient and a country.
func (s *Service) Preview(ctx context.Context, ws *store.Workspace, id uuid.UUID, in SmsPreviewInput) (SmsPreview, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return SmsPreview{}, err
	}
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return SmsPreview{}, err
	}
	r := Recipient{}
	if in.ContactID != "" {
		cid, err := uuid.Parse(in.ContactID)
		if err != nil {
			return SmsPreview{}, api.Invalid("body.contactId", "Pick a contact of this workspace.")
		}
		row := s.Pool.QueryRow(ctx, `SELECT id, phone, phone_country, timezone, first_name, last_name, email FROM contacts WHERE id = $1 AND workspace_id = $2`, cid, ws.ID)
		if err := row.Scan(&r.ContactID, &r.Phone, &r.Country, &r.Zone, &r.FirstName, &r.LastName, &r.Email); err != nil {
			return SmsPreview{}, api.MapError(err, "Contact")
		}
	}
	country := in.Country
	if country == "" {
		country = r.Country
	}
	if country == "" {
		country = "US"
	}
	out := SmsPreview{Country: country, Contact: strings.TrimSpace(r.FirstName + " " + r.LastName), Parts: []SmsPreviewPart{}}
	if r.ContactID != uuid.Nil {
		out.ContactID = r.ContactID.String()
	}
	d := g.decide(country)
	sn := d.Sender
	if sn == nil && d.Route != nil {
		if x, ok := g.senders[d.Route.SenderID]; ok {
			sn = &x
		}
		out.Skipped = d.Hold
	} else if sn == nil {
		out.Skipped = d.Hold
	}
	if sn != nil {
		out.Sender, out.SenderKind, out.Provider = senderLabel(*sn), sn.Kind, sn.ProviderName
		out.CanReply = sn.Kind != "sender_id"
	}
	out.Price = d.Price
	text := textOf(c.Body, g.rules)
	domain := s.shortDomain(g.rules)
	// The link codes the person gets: the sent message's, else derived.
	codes := map[int]string{}
	if r.ContactID != uuid.Nil {
		rows, err := s.Pool.Query(ctx, `SELECT l.code FROM sms_links l WHERE l.campaign_id = $1 AND l.contact_id = $2 ORDER BY l.created_at, l.code`, c.ID, r.ContactID)
		if err == nil {
			i := 0
			for rows.Next() {
				var code string
				if rows.Scan(&code) == nil {
					codes[i] = code
					i++
				}
			}
			rows.Close()
		}
	}
	if text.Prefix != "" {
		out.Parts = append(out.Parts, SmsPreviewPart{Kind: "fixed", Text: text.Prefix})
	}
	body := render(text.Body, r)
	links := findLinks(body)
	last := 0
	for i, l := range links {
		if l.Start > last {
			out.Parts = append(out.Parts, SmsPreviewPart{Kind: "text", Text: body[last:l.Start]})
		}
		shown := l.URL
		if domain != "" && !l.Shortener {
			code, ok := codes[i]
			if !ok {
				code = LinkCode(c.ID, r.ContactID, i)
			}
			shown = domain + "/" + code
		}
		out.Parts = append(out.Parts, SmsPreviewPart{Kind: "link", Text: shown})
		last = l.End
	}
	if last < len(body) {
		out.Parts = append(out.Parts, SmsPreviewPart{Kind: "text", Text: body[last:]})
	}
	if text.OptOut != "" {
		out.Parts = append(out.Parts, SmsPreviewPart{Kind: "fixed", Text: text.OptOut})
	}
	full := Text{Prefix: text.Prefix, Body: render(withShortLinks(text.Body, domain, placeholder), r), OptOut: text.OptOut}.Full()
	cnt := sms.CountText(full)
	out.Characters, out.Segments, out.Encoding = cnt.Characters, cnt.Segments, string(cnt.Encoding)
	zone, _ := phone.RecipientZone(r.Zone, r.Phone)
	if zone == "" {
		zone, _ = phone.TimeZone(countryProbe(country))
	}
	out.Zone = zone
	loc := location(zone)
	switch c.ScheduleMode {
	case "local":
		if c.LocalDate != nil && c.LocalMinute != nil {
			t := time.Date(c.LocalDate.Year(), c.LocalDate.Month(), c.LocalDate.Day(), int(*c.LocalMinute)/60, int(*c.LocalMinute)%60, 0, 0, loc)
			out.Arrives = &t
		}
	case "at":
		out.Arrives = c.SendAt
	}
	if c.StartedAt != nil && r.ContactID != uuid.Nil {
		var sentAt *time.Time
		_ = s.Pool.QueryRow(ctx, `SELECT coalesce(sent_at, created_at) FROM channel_messages WHERE campaign_id = $1 AND contact_id = $2 ORDER BY created_at LIMIT 1`,
			c.ID, r.ContactID).Scan(&sentAt)
		if sentAt != nil {
			out.Arrives = sentAt
		}
	}
	if r.ContactID != uuid.Nil {
		var b string
		var at time.Time
		if err := s.Pool.QueryRow(ctx, `SELECT coalesce(body, ''), coalesce(sent_at, created_at) FROM channel_messages WHERE contact_id = $1 AND kind = 'confirmation'
			ORDER BY created_at LIMIT 1`, r.ContactID).Scan(&b, &at); err == nil && b != "" {
			out.Confirmation = &SmsPreviewBubble{Text: b, At: at}
		}
	}
	return out, nil
}

// countryProbe is a sample number of a country, to find its main zone.
func countryProbe(country string) string {
	probes := map[string]string{"US": "+18885550100", "CA": "+14165550100", "GB": "+447700900000", "DE": "+4915123450000", "NL": "+31612345678", "IE": "+353851234567"}
	if p, ok := probes[country]; ok {
		return p
	}
	return ""
}

// PreviewContacts lists people to preview as: the first recipients, one
// per country first, the country with the most recipients leading.
func (s *Service) PreviewContacts(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]SmsPreviewContact, error) {
	c, err := s.campaign(ctx, ws, id)
	if err != nil {
		return nil, err
	}
	f, err := s.funnel(ctx, ws, parseAudience(c.Audience))
	if err != nil {
		return nil, err
	}
	out := []SmsPreviewContact{}
	seen := map[string]bool{}
	add := func(r Recipient) {
		out = append(out, SmsPreviewContact{ID: r.ContactID.String(), Name: strings.TrimSpace(r.FirstName + " " + r.LastName), Country: r.Country})
	}
	per := map[string]int{}
	for _, r := range f.Recipients {
		per[r.Country]++
	}
	firsts := []Recipient{}
	for _, r := range f.Recipients {
		if !seen[r.Country] {
			seen[r.Country] = true
			firsts = append(firsts, r)
		}
	}
	sort.SliceStable(firsts, func(i, j int) bool { return per[firsts[i].Country] > per[firsts[j].Country] })
	for _, r := range firsts {
		add(r)
	}
	for _, r := range f.Recipients {
		if len(out) >= 20 {
			break
		}
		dup := false
		for _, o := range out {
			if o.ID == r.ContactID.String() {
				dup = true
			}
		}
		if !dup {
			add(r)
		}
	}
	return out, nil
}
