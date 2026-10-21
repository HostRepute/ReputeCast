package revenue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/text/currency"
	"golang.org/x/time/rate"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// The conversion API's error codes (Earn › Settings › Attribution lists them).
const (
	CodeMissingIdentity  = "missing_identity"
	CodeInvalidValue     = "invalid_value"
	CodeInvalidCurrency  = "invalid_currency"
	CodeOccurredAtRange  = "occurred_at_out_of_range"
	CodeIdempotencyReuse = "idempotency_key_reused"
)

// MaxAge is how far back a conversion may have happened.
const MaxAge = 30 * day

// ConversionInput is a conversion recorded through the API: a valuable
// action in the publisher's own product, with a value.
type ConversionInput struct {
	// ClickToken is rc_click=… from the link the reader followed, if present.
	ClickToken string `json:"click_token,omitempty" maxLength:"512" doc:"The rc_click token from the link the reader followed, if present"`
	Email      string `json:"email,omitempty" maxLength:"320" doc:"The reader's address; send it, the click token, or both"`
	Name       string `json:"name" minLength:"1" maxLength:"120" doc:"What happened: \"Upgraded to Team plan\""`
	// Value is a decimal string: "29.00", not 29.
	Value      any            `json:"value" doc:"A decimal string: \"29.00\", not 29"`
	Currency   string         `json:"currency" maxLength:"8" doc:"ISO 4217: USD, EUR, GBP…"`
	OccurredAt *time.Time     `json:"occurred_at,omitempty" doc:"When it happened (now when absent): up to 30 days back"`
	Metadata   map[string]any `json:"metadata,omitempty" doc:"Your own fields, kept with the conversion"`
}

// ConversionCredit is the email a conversion is credited to.
type ConversionCredit struct {
	Kind string `json:"kind" enum:"campaign,automation"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ConversionResult is the conversion API's answer: how the conversion was
// matched and the email it's credited to.
type ConversionResult struct {
	ID       string            `json:"id"`
	Matched  string            `json:"matched" enum:"click_token,email,none"`
	Credited *ConversionCredit `json:"credited,omitempty"`
	Test     bool              `json:"test,omitempty"`
}

// ConversionItem is a line of the conversions log.
type ConversionItem struct {
	ID         string            `json:"id"`
	ReceivedAt time.Time         `json:"receivedAt"`
	OccurredAt time.Time         `json:"occurredAt"`
	Name       string            `json:"name"`
	Value      string            `json:"value"`
	ValueCents int64             `json:"valueCents"`
	Currency   string            `json:"currency"`
	Matched    string            `json:"matched" enum:"click_token,email,none"`
	Credited   *ConversionCredit `json:"credited,omitempty"`
	KeyName    string            `json:"keyName"`
	Test       bool              `json:"test"`
}

// ConversionList is a page of the conversions log, newest first.
type ConversionList struct {
	Items []ConversionItem `json:"items"`
	// Next is the cursor of the following page.
	Next *time.Time `json:"next,omitempty"`
	// Window is the attribution window in days ("no email click in 7 days").
	Window int `json:"window"`
}

func invalid(ctx context.Context, location, key string, args i18n.Args) *api.Problem {
	return api.Invalid(location, text(ctx, key, args))
}

func invalidAt(ctx context.Context, location string, index int, key string, args i18n.Args) *api.Problem {
	return api.Invalid(fmt.Sprintf("%s[%d]", location, index), text(ctx, key, args))
}

// coded is a 422 for one field with the conversion API's own code.
func coded(ctx context.Context, code, location, key string) *api.Problem {
	p := api.Invalid(location, text(ctx, key, nil))
	p.Code = code
	return p
}

// retryAfter is a 429 that says when to try again.
type retryAfter struct {
	*api.Problem
	seconds int
}

func (e retryAfter) GetHeaders() http.Header {
	h := http.Header{}
	h.Set("Retry-After", strconv.Itoa(e.seconds))
	return h
}

var _ huma.HeadersError = retryAfter{}

// bucket is a key's rate limiter.
type bucket struct {
	lim  *rate.Limiter
	used time.Time
}

// allow takes one conversion from the key's rate, or says how many
// seconds until there's room.
func (s *Service) allow(key uuid.UUID) (bool, int) {
	per := s.PerMinute
	if per <= 0 {
		per = 600
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.buckets == nil {
		s.buckets = map[uuid.UUID]*bucket{}
	}
	b, ok := s.buckets[key]
	if !ok {
		b = &bucket{lim: rate.NewLimiter(rate.Every(time.Minute/time.Duration(per)), max(per/10, 1))}
		s.buckets[key] = b
		if len(s.buckets) > 10000 {
			for k, v := range s.buckets {
				if time.Since(v.used) > time.Hour {
					delete(s.buckets, k)
				}
			}
		}
	}
	b.used = time.Now()
	r := b.lim.Reserve()
	if d := r.Delay(); d > 0 {
		r.Cancel()
		return false, int(math.Ceil(d.Seconds()))
	}
	return true, 0
}

var decimalRe = regexp.MustCompile(`^\d{1,12}(\.\d{1,4})?$`)

// minorUnits turns a decimal string into minor units of the currency
// ("29.00" USD → 2900; "1500" JPY → 1500), refusing more decimals than it has.
func minorUnits(value string, unit currency.Unit) (int64, bool) {
	scale, _ := currency.Standard.Rounding(unit)
	whole, frac, _ := strings.Cut(value, ".")
	if len(frac) > scale {
		if strings.Trim(frac[scale:], "0") != "" {
			return 0, false
		}
		frac = frac[:scale]
	}
	frac += strings.Repeat("0", scale-len(frac))
	n, err := strconv.ParseInt(whole+frac, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// requestHash identifies a request's content, for the idempotency check.
func requestHash(in ConversionInput) string {
	b, _ := json.Marshal(in)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:16])
}

// Record keeps a conversion and credits it to the email the reader clicked
// within the attribution window: by click token, else by address. It's
// logged in the conversions log, not the audit log.
func (s *Service) Record(ctx context.Context, ws *store.Workspace, p *authz.Principal, in ConversionInput, idemKey string, test bool) (ConversionResult, error) {
	in.ClickToken, in.Email, in.Name = strings.TrimSpace(in.ClickToken), strings.ToLower(strings.TrimSpace(in.Email)), strings.TrimSpace(in.Name)
	if in.ClickToken == "" && in.Email == "" {
		return ConversionResult{}, coded(ctx, CodeMissingIdentity, "body", "problem.missingIdentity")
	}
	if in.Name == "" {
		return ConversionResult{}, invalid(ctx, "body.name", "problem.name", nil)
	}
	raw, ok := in.Value.(string)
	raw = strings.TrimSpace(raw)
	if !ok || !decimalRe.MatchString(raw) {
		return ConversionResult{}, coded(ctx, CodeInvalidValue, "body.value", "problem.value")
	}
	code := strings.ToUpper(strings.TrimSpace(in.Currency))
	unit, err := currency.ParseISO(code)
	if err != nil || len(code) != 3 {
		return ConversionResult{}, coded(ctx, CodeInvalidCurrency, "body.currency", "problem.currency")
	}
	cents, ok := minorUnits(raw, unit)
	if !ok {
		return ConversionResult{}, coded(ctx, CodeInvalidValue, "body.value", "problem.valueDecimals")
	}
	now := s.now()
	occurred := now
	if in.OccurredAt != nil {
		occurred = *in.OccurredAt
		if occurred.Before(now.Add(-MaxAge)) || occurred.After(now.Add(5*time.Minute)) {
			return ConversionResult{}, coded(ctx, CodeOccurredAtRange, "body.occurred_at", "problem.occurredAt")
		}
	}
	var keyID *uuid.UUID
	keyName := ""
	if p != nil {
		keyName = p.Name
		if p.Type == authz.ActorAPIKey {
			id := p.ID
			keyID = &id
		}
	}
	if keyID != nil && !test {
		if ok, wait := s.allow(*keyID); !ok {
			pr := api.Err(http.StatusTooManyRequests, api.CodeRateLimited, text(ctx, "problem.rateLimited", map[string]any{"seconds": wait}))
			return ConversionResult{}, retryAfter{Problem: pr, seconds: wait}
		}
	}
	hash := requestHash(in)
	idemKey = strings.TrimSpace(idemKey)
	if len(idemKey) > 200 {
		return ConversionResult{}, invalid(ctx, "header.Idempotency-Key", "problem.idempotencyKey", nil)
	}
	if idemKey != "" && keyID != nil {
		if prev, err := s.Q.RevenueConversionByKey(ctx, store.RevenueConversionByKeyParams{WorkspaceID: ws.ID, ApiKeyID: keyID, IdempotencyKey: idemKey}); err == nil {
			if prev.RequestHash != hash {
				pr := api.Err(http.StatusConflict, CodeIdempotencyReuse, text(ctx, "problem.idempotencyReuse", nil))
				return ConversionResult{}, pr
			}
			return s.resultOf(ctx, prev), nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return ConversionResult{}, err
		}
	}
	window := 7 * day
	if st, err := s.Q.RevenueAttributionSettings(ctx, ws.ID); err == nil {
		window = time.Duration(st.AttributionDays) * day
	}
	row := store.RevenueConversionInsertParams{ID: newID(), WorkspaceID: ws.ID, ApiKeyID: keyID, KeyName: keyName, Name: in.Name, Value: raw, ValueCents: cents,
		Currency: strings.ToLower(code), OccurredAt: occurred, ReceivedAt: now, Matched: "none", Metadata: []byte("{}"), IdempotencyKey: idemKey, RequestHash: hash, Test: test}
	if in.Metadata != nil {
		if b, err := json.Marshal(in.Metadata); err == nil && len(b) <= 8192 {
			row.Metadata = b
		}
	}
	// By click token: the email clicked, and when.
	if in.ClickToken != "" && s.Links != nil {
		if c, err := s.Links.Verify(in.ClickToken, ClickToken); err == nil && c.WorkspaceID == ws.ID {
			clicked := time.Unix(int64(c.Link)*60, 0)
			if !clicked.After(occurred.Add(time.Minute)) && occurred.Sub(clicked) <= window {
				s.credit(ctx, &row, c)
				row.ClickedAt = &clicked
			}
		}
	}
	// By address: the last email the contact clicked within the window.
	if row.Matched == "none" && in.Email != "" {
		if cid, err := s.Q.RevenueContactByEmail(ctx, store.RevenueContactByEmailParams{WorkspaceID: ws.ID, Email: in.Email}); err == nil {
			row.ContactID, row.Matched = &cid, "email"
			if last, err := s.Q.RevenueLastEmailClick(ctx, store.RevenueLastEmailClickParams{ContactID: &cid, At: occurred, Since: occurred.Add(-window)}); err == nil {
				row.MessageID, row.CampaignID, row.AutomationID = last.MessageID, last.CampaignID, last.AutomationID
				at := last.OccurredAt
				row.ClickedAt = &at
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return ConversionResult{}, err
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return ConversionResult{}, err
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.RevenueConversionInsert(ctx, row); err != nil {
			return err
		}
		if test || (row.CampaignID == nil && row.AutomationID == nil) {
			return nil
		}
		return s.emit(ctx, q, ws, "conversion.recorded", conversionEvent(row))
	})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && idemKey != "" {
		// The same key raced itself: answer with the first.
		if prev, err := s.Q.RevenueConversionByKey(ctx, store.RevenueConversionByKeyParams{WorkspaceID: ws.ID, ApiKeyID: keyID, IdempotencyKey: idemKey}); err == nil {
			return s.resultOf(ctx, prev), nil
		}
	}
	if err != nil {
		return ConversionResult{}, err
	}
	return s.resultOf(ctx, store.Conversion{ID: row.ID, Matched: row.Matched, CampaignID: row.CampaignID, AutomationID: row.AutomationID, Test: row.Test}), nil
}

// credit credits a conversion to a click token's email and reader.
func (s *Service) credit(ctx context.Context, row *store.RevenueConversionInsertParams, c linktoken.Claims) {
	row.Matched = "click_token"
	if c.ContactID != uuid.Nil {
		cid := c.ContactID
		row.ContactID = &cid
	}
	if c.MessageID == uuid.Nil {
		return
	}
	mid := c.MessageID
	row.MessageID = &mid
	if m, err := s.Q.RevenueMessageOf(ctx, mid); err == nil && m.WorkspaceID == row.WorkspaceID {
		row.CampaignID, row.AutomationID = m.CampaignID, m.AutomationID
	}
}

// resultOf is the API's answer for a stored conversion.
func (s *Service) resultOf(ctx context.Context, c store.Conversion) ConversionResult {
	out := ConversionResult{ID: c.ID.String(), Matched: c.Matched, Test: c.Test}
	switch {
	case c.CampaignID != nil:
		name, _ := s.Q.EarnCampaignName(ctx, *c.CampaignID)
		out.Credited = &ConversionCredit{Kind: "campaign", ID: c.CampaignID.String(), Name: name}
	case c.AutomationID != nil:
		var name string
		_ = s.Pool.QueryRow(ctx, `SELECT name FROM automations WHERE id = $1`, *c.AutomationID).Scan(&name)
		out.Credited = &ConversionCredit{Kind: "automation", ID: c.AutomationID.String(), Name: name}
	}
	return out
}

// conversionEvent is the conversion.recorded webhook's data.
func conversionEvent(r store.RevenueConversionInsertParams) map[string]any {
	d := map[string]any{"conversion_id": r.ID.String(), "name": r.Name, "value": r.Value, "currency": strings.ToUpper(r.Currency),
		"occurred_at": r.OccurredAt.UTC(), "matched": r.Matched}
	if r.ContactID != nil {
		d["contact_id"] = r.ContactID.String()
	}
	if r.CampaignID != nil {
		d["campaign_id"] = r.CampaignID.String()
	}
	if r.AutomationID != nil {
		d["automation_id"] = r.AutomationID.String()
	}
	if r.MessageID != nil {
		d["message_id"] = r.MessageID.String()
	}
	return d
}

// ConversionsQuery pages the conversions log.
type ConversionsQuery struct {
	Before time.Time `query:"before" doc:"Conversions received before this moment (the previous page's next)"`
	Limit  int       `query:"limit" minimum:"1" maximum:"200" default:"50"`
}

// Conversions is a page of the conversions log, newest first.
func (s *Service) Conversions(ctx context.Context, ws *store.Workspace, in ConversionsQuery) (ConversionList, error) {
	out := ConversionList{Items: []ConversionItem{}, Window: 7}
	if st, err := s.Q.RevenueAttributionSettings(ctx, ws.ID); err == nil {
		out.Window = int(st.AttributionDays)
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	var before *time.Time
	if !in.Before.IsZero() {
		b := in.Before
		before = &b
	}
	rows, err := s.Q.RevenueConversions(ctx, store.RevenueConversionsParams{WorkspaceID: ws.ID, Before: before, Lim: int32(limit + 1)}) //nolint:gosec // at most 201
	if err != nil {
		return out, err
	}
	for i, r := range rows {
		if i == limit {
			next := rows[i-1].ReceivedAt
			out.Next = &next
			break
		}
		it := ConversionItem{ID: r.ID.String(), ReceivedAt: r.ReceivedAt, OccurredAt: r.OccurredAt, Name: r.Name, Value: r.Value, ValueCents: r.ValueCents,
			Currency: r.Currency, Matched: r.Matched, KeyName: r.KeyName, Test: r.Test}
		switch {
		case r.CampaignID != nil:
			it.Credited = &ConversionCredit{Kind: "campaign", ID: r.CampaignID.String(), Name: r.CreditedName}
		case r.AutomationID != nil:
			it.Credited = &ConversionCredit{Kind: "automation", ID: r.AutomationID.String(), Name: r.CreditedName}
		}
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// TestConversion records a test conversion for the person's own address:
// shown in the log, left out of every number.
func (s *Service) TestConversion(ctx context.Context, ws *store.Workspace, p *authz.Principal) (ConversionResult, error) {
	cur := "usd"
	if st, err := s.Q.RevenueAttributionSettings(ctx, ws.ID); err == nil && st.Currency != "" {
		cur = st.Currency
	}
	return s.Record(ctx, ws, p, ConversionInput{Email: p.Email, Name: text(ctx, "test.name", nil), Value: "1.00", Currency: cur}, "", true)
}
