package texting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// Health of SMS providers (brief §3.1): Healthy, Degraded (receipts silent
// while texts were sent), Down (calls failing) or Credentials rejected.
// The provider health job checks each provider every minute it's in
// trouble, every five otherwise; reads registration statuses every hour
// where the API allows; prices once a day; and opt-out lists every 15
// minutes.

// Health thresholds.
const (
	// ReceiptsSilentAfter: texts sent and no receipt for this long.
	ReceiptsSilentAfter = 2 * time.Hour
	// downAfter failed calls in a row make a provider Down.
	downAfter     = 3
	checkEvery    = 5 * time.Minute
	readRegsEvery = time.Hour
	readPrices    = 24 * time.Hour
	optOutEvery   = 15 * time.Minute
)

// providerHealth is sms_providers.health.
type providerHealth struct {
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	LastError           string     `json:"lastError,omitempty"`
	LastErrorAt         *time.Time `json:"lastErrorAt,omitempty"`
	// Answer is the provider's answer to the last failed call
	// ("401 Unauthorized · invalid credentials").
	Answer string `json:"answer,omitempty"`
	// SilentSince: receipts stopped while texts were sent.
	SilentSince *time.Time `json:"silentSince,omitempty"`
	// LastSuccessAt is the last call that worked.
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
	PricesReadAt  *time.Time `json:"pricesReadAt,omitempty"`
	OptOutsReadAt *time.Time `json:"optOutsReadAt,omitempty"`
	// Notified remembers what people were told, so each problem is told once.
	Notified string `json:"notified,omitempty"`
}

func healthOf(p store.SmsProvider) providerHealth {
	var h providerHealth
	_ = json.Unmarshal(p.Health, &h)
	return h
}

// recordCall notes a call's outcome: rejected credentials and failing calls
// change the provider's status, a success clears them.
func (s *Service) recordCall(ctx context.Context, p store.SmsProvider, callErr error) {
	cur, err := s.Q.SmsProvider(ctx, store.SmsProviderParams{ID: p.ID, WorkspaceID: p.WorkspaceID})
	if err != nil {
		return
	}
	h := healthOf(cur)
	now := s.now()
	status, detail := cur.Status, cur.StatusDetail
	switch {
	case callErr == nil:
		h.ConsecutiveFailures, h.LastSuccessAt = 0, &now
		if status == "down" || status == "credentials_rejected" {
			status, detail = "healthy", ""
			if h.SilentSince != nil {
				status, detail = "degraded", "receipts_silent"
			}
			h.Notified = ""
		}
	case errors.Is(callErr, sms.ErrCredentialsRejected):
		h.ConsecutiveFailures++
		h.LastError, h.LastErrorAt, h.Answer = callErr.Error(), &now, answerOf(callErr)
		status, detail = "credentials_rejected", ""
	case errors.Is(callErr, sms.ErrUnavailable):
		h.ConsecutiveFailures++
		h.LastError, h.LastErrorAt, h.Answer = callErr.Error(), &now, answerOf(callErr)
		if h.ConsecutiveFailures >= downAfter {
			status, detail = "down", ""
		}
	default:
		// The provider refused one text: its health isn't in question.
		return
	}
	if err := s.Q.SmsProviderSetHealth(ctx, store.SmsProviderSetHealthParams{Status: status, StatusDetail: detail, At: &now, Health: jsonOf(h), ID: cur.ID}); err != nil {
		return
	}
	if status != cur.Status {
		s.providerStatusChanged(ctx, cur, status, h)
	}
}

// answerOf writes a provider's answer as the provider card shows it.
func answerOf(err error) string {
	var ce *sms.CallError
	if errors.As(err, &ce) {
		switch ce.Status {
		case 401:
			return "401 Unauthorized · invalid credentials"
		case 403:
			return "403 Forbidden"
		}
		if ce.Status >= 500 {
			return fmt.Sprintf("%d Service Unavailable", ce.Status)
		}
		return fmt.Sprintf("%d %s", ce.Status, ce.Code)
	}
	switch {
	case errors.Is(err, sms.ErrCredentialsRejected):
		return "401 Unauthorized · invalid credentials"
	case errors.Is(err, sms.ErrUnavailable):
		return "no answer"
	}
	return err.Error()
}

// providerStatusChanged tells people a provider is down or its
// credentials were rejected, and audits it as ReputeCast's own action.
func (s *Service) providerStatusChanged(ctx context.Context, p store.SmsProvider, status string, h providerHealth) {
	ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
	if err != nil {
		return
	}
	var title, summary, sev string
	switch status {
	case "credentials_rejected":
		title, sev = p.Name+" credentials rejected", "bad"
		summary = fmt.Sprintf("%s rejected the credentials. Texts through %s are held until they're replaced.", p.Name, p.Name)
	case "down":
		title, sev = p.Name+" isn't answering", "bad"
		summary = fmt.Sprintf("%s isn't answering. Texts through %s are held; ReputeCast retries every 2 minutes.", p.Name, p.Name)
	case "healthy":
		title, sev = p.Name+" is back", "ok"
		summary = fmt.Sprintf("%s answers again. Held texts go out inside each recipient's window.", p.Name)
	default:
		return
	}
	link := "/w/" + ws.Slug + "/providers/sms"
	_ = notify.ToMembersWith(ctx, s.Pool, ws.ID, authz.ProvidersView, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSMSProviderDown, Severity: sev,
		Title: title, Summary: summary, Body: summary, Link: link, Data: map[string]any{"provider_id": p.ID.String(), "status": status}})
	_ = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "sms_provider.status_changed", TargetType: "sms_provider", TargetID: p.ID.String(), TargetName: p.Name,
		Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Before: map[string]any{"status": p.Status}, After: map[string]any{"status": status, "answer": h.Answer}})
}

// CheckProvider probes a provider now (its credentials and account).
func (s *Service) CheckProvider(ctx context.Context, p store.SmsProvider) error {
	ad, err := s.open(p)
	if err != nil {
		return err
	}
	cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err = ad.Check(cctx)
	s.recordCall(ctx, p, err)
	_ = s.Pool.QueryRow(ctx, `UPDATE sms_providers SET last_checked_at = $2 WHERE id = $1 RETURNING id`, p.ID, s.now()).Scan(new(uuid.UUID))
	return err
}

// healthTick is the provider health job's SMS pass.
func (s *Service) healthTick(ctx context.Context) error {
	provs, err := s.Q.SmsProvidersAll(ctx)
	if err != nil {
		return err
	}
	now := s.now()
	for _, p := range provs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		h := healthOf(p)
		trouble := p.Status != "healthy"
		if !p.Demo && (p.LastCheckedAt == nil || now.Sub(*p.LastCheckedAt) >= checkEvery || trouble) {
			_ = s.CheckProvider(ctx, p)
		}
		s.checkReceipts(ctx, p, h, now)
		if p.RegistrationsReadAt == nil || now.Sub(*p.RegistrationsReadAt) >= readRegsEvery {
			if err := s.ReadRegistrations(ctx, p); err != nil {
				s.log().WarnContext(ctx, "sms registrations", "provider", p.ID, "err", err)
			}
		}
		if h.PricesReadAt == nil || now.Sub(*h.PricesReadAt) >= readPrices {
			if err := s.ReadPrices(ctx, p); err != nil {
				s.log().WarnContext(ctx, "sms prices", "provider", p.ID, "err", err)
			}
		}
		if h.OptOutsReadAt == nil || now.Sub(*h.OptOutsReadAt) >= optOutEvery {
			if err := s.SyncOptOuts(ctx, p); err != nil {
				s.log().WarnContext(ctx, "sms opt-out sync", "provider", p.ID, "err", err)
			}
		}
	}
	return nil
}

// checkReceipts marks a provider Degraded when texts went out and no
// receipt came back for two hours, and clears it when receipts return.
func (s *Service) checkReceipts(ctx context.Context, p store.SmsProvider, h providerHealth, now time.Time) {
	var sentSince int64
	since := now.Add(-ReceiptsSilentAfter)
	if p.LastReceiptAt != nil && p.LastReceiptAt.After(since) {
		since = *p.LastReceiptAt
	}
	_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_messages WHERE provider_id = $1 AND sent_at IS NOT NULL AND sent_at < $2 AND sent_at > $3`,
		p.ID, now.Add(-ReceiptsSilentAfter), since.Add(-time.Hour)).Scan(&sentSince)
	silent := sentSince > 0 && (p.LastReceiptAt == nil || now.Sub(*p.LastReceiptAt) >= ReceiptsSilentAfter)
	switch {
	case silent && h.SilentSince == nil && p.Status == "healthy":
		at := now
		if p.LastReceiptAt != nil {
			at = *p.LastReceiptAt
		}
		h.SilentSince = &at
		_ = s.Q.SmsProviderSetHealth(ctx, store.SmsProviderSetHealthParams{Status: "degraded", StatusDetail: "receipts_silent", At: &now, Health: jsonOf(h), ID: p.ID})
		ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
		if err == nil {
			summary := fmt.Sprintf("No receipts from %s for 2 hours while texts were sent. Check the status callback in %s.", p.Name, p.Name)
			_ = notify.ToMembersWith(ctx, s.Pool, ws.ID, authz.ProvidersView, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSMSProviderDown, Severity: "warn",
				Title: p.Name + " receipts silent", Summary: summary, Body: summary, Link: "/w/" + ws.Slug + "/providers/sms", Data: map[string]any{"provider_id": p.ID.String()}})
		}
	case !silent && h.SilentSince != nil:
		h.SilentSince = nil
		status, detail := p.Status, p.StatusDetail
		if status == "degraded" && detail == "receipts_silent" {
			status, detail = "healthy", ""
		}
		_ = s.Q.SmsProviderSetHealth(ctx, store.SmsProviderSetHealthParams{Status: status, StatusDetail: detail, At: &now, Health: jsonOf(h), ID: p.ID})
	}
}

// ReadRegistrations reads the senders' registration statuses from the
// provider (where its API reports them) and tells people when one was
// approved or rejected.
func (s *Service) ReadRegistrations(ctx context.Context, p store.SmsProvider) error {
	now := s.now()
	defer func() { _ = s.Q.SmsProviderRegistrationsRead(ctx, store.SmsProviderRegistrationsReadParams{At: &now, ID: p.ID}) }()
	ad, err := s.open(p)
	if err != nil {
		return err
	}
	rr, ok := ad.(sms.RegistrationReader)
	if !ok {
		return nil
	}
	senders, err := s.Q.SmsSenders(ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	mine := []sms.Sender{}
	byValue := map[string]store.SmsSendersRow{}
	for _, sn := range senders {
		if sn.ProviderID == p.ID {
			mine = append(mine, sms.Sender{Kind: sn.Kind, Value: sn.Value, Countries: sn.Countries, ProviderRef: sn.ProviderRef})
			byValue[sn.Value] = sn
		}
	}
	if len(mine) == 0 {
		return nil
	}
	regs, err := rr.Registrations(ctx, mine)
	if err != nil {
		return err
	}
	before, err := s.Q.SmsRegistrations(ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	prev := map[string]store.SmsRegistrationsRow{}
	for _, b := range before {
		prev[b.SenderID.String()+"|"+b.Kind+"|"+b.Country] = b
	}
	for _, g := range regs {
		sn, ok := byValue[g.Sender]
		if !ok {
			continue
		}
		var on *time.Time
		if g.Since != nil {
			d := time.Date(g.Since.Year(), g.Since.Month(), g.Since.Day(), 0, 0, 0, 0, time.UTC)
			on = &d
		}
		details := []byte("{}")
		if len(g.Details) > 0 {
			details = jsonOf(g.Details)
		}
		old, had := prev[sn.ID.String()+"|"+g.Kind+"|"+g.Country]
		if had && old.Source == "manual" && old.Status == g.Status {
			continue
		}
		if err := s.Q.SmsRegistrationUpsert(ctx, store.SmsRegistrationUpsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: p.WorkspaceID, SenderID: sn.ID, Kind: g.Kind,
			Country: g.Country, Name: g.Name, Status: g.Status, Reason: g.Reason, StatusOn: on, Source: "api", Details: details, ReadAt: &now, UpdatedAt: now}); err != nil {
			return err
		}
		if had && old.Status != g.Status && (g.Status == "approved" || g.Status == "verified" || g.Status == "rejected") {
			s.registrationChanged(ctx, p, sn, g)
		}
	}
	return nil
}

func (s *Service) registrationChanged(ctx context.Context, p store.SmsProvider, sn store.SmsSendersRow, g sms.Registration) {
	ws, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
	if err != nil {
		return
	}
	what := g.Name
	if what == "" {
		what = senderLabel(sn)
	}
	sev, title, summary := "ok", "Registration approved", fmt.Sprintf("%s approved %s for %s.", p.Name, what, senderLabel(sn))
	if g.Status == "rejected" {
		sev, title = "bad", "Registration rejected"
		summary = fmt.Sprintf("%s rejected %s for %s: %s", p.Name, what, senderLabel(sn), strings.TrimSuffix(g.Reason, ".")+".")
	}
	_ = notify.ToMembersWith(ctx, s.Pool, ws.ID, authz.ProvidersView, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSMSRegistration, Severity: sev,
		Title: title, Summary: summary, Body: summary, Link: "/w/" + ws.Slug + "/providers/sms", Data: map[string]any{"sender_id": sn.ID.String(), "status": g.Status}})
}

// ReadPrices reads the price of a segment for every country the provider's
// senders text.
func (s *Service) ReadPrices(ctx context.Context, p store.SmsProvider) error {
	now := s.now()
	h := healthOf(p)
	h.PricesReadAt = &now
	defer func() { _ = s.Pool.QueryRow(ctx, `UPDATE sms_providers SET health = health || $2::jsonb WHERE id = $1 RETURNING id`, p.ID, jsonOf(map[string]any{"pricesReadAt": now})).Scan(new(uuid.UUID)) }()
	ad, err := s.open(p)
	if err != nil {
		return err
	}
	pr, ok := ad.(sms.PricingReader)
	if !ok {
		return nil
	}
	senders, err := s.Q.SmsSenders(ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	countries := []string{}
	for _, sn := range senders {
		if sn.ProviderID != p.ID {
			continue
		}
		for _, c := range sn.Countries {
			if !slices.Contains(countries, c) {
				countries = append(countries, c)
			}
		}
	}
	if len(countries) == 0 {
		return nil
	}
	prices, err := pr.Prices(ctx, countries)
	if err != nil {
		return err
	}
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for c, v := range prices {
		if err := s.Q.SmsPriceUpsert(ctx, store.SmsPriceUpsertParams{WorkspaceID: p.WorkspaceID, ProviderID: p.ID, Country: c, Price: round4(v), Source: "api", AsOf: day}); err != nil {
			return err
		}
	}
	return nil
}

// SyncOptOuts pulls the provider's opt-out list into the SMS suppression
// list and, for two-way adapters, pushes numbers that opted out here.
func (s *Service) SyncOptOuts(ctx context.Context, p store.SmsProvider) error {
	now := s.now()
	defer func() { _ = s.Pool.QueryRow(ctx, `UPDATE sms_providers SET health = health || $2::jsonb WHERE id = $1 RETURNING id`, p.ID, jsonOf(map[string]any{"optOutsReadAt": now})).Scan(new(uuid.UUID)) }()
	d, ok := sms.Lookup(p.Adapter)
	if !ok {
		return nil
	}
	ad, err := s.open(p)
	if err != nil {
		return err
	}
	mode := map[sms.Support]string{sms.Yes: "two_way", sms.Partial: "read_only", sms.No: "off"}[d.Has(sms.CapOptOutSync)]
	rd, reads := ad.(sms.OptOutReader)
	var pulled, pushed int32
	status, detail := "in_sync", ""
	if reads {
		list, err := rd.OptOuts(ctx)
		if err != nil {
			status, detail = "failing", answerOf(err)
		} else {
			for _, o := range list {
				var n int64
				_ = s.Pool.QueryRow(ctx, `SELECT count(*) FROM channel_suppressions WHERE workspace_id = $1 AND channel = 'sms' AND phone = $2 AND removed_at IS NULL`,
					p.WorkspaceID, o.Phone).Scan(&n)
				if n > 0 {
					continue
				}
				code := o.Keyword
				if code == "" {
					code = p.Name + " opt-out list"
				}
				if err := s.Q.ChanSuppressionInsert(ctx, store.ChanSuppressionInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: p.WorkspaceID, Channel: "sms",
					Phone: o.Phone, Reason: "provider_list", OriginalCode: code, Source: p.Name, SourceProviderID: &p.ID, CreatedAt: now}); err == nil {
					pulled++
				}
			}
		}
	}
	if sy, ok := ad.(sms.OptOutSyncer); ok && status == "in_sync" {
		rows, err := s.Pool.Query(ctx, `SELECT phone FROM channel_suppressions WHERE workspace_id = $1 AND channel = 'sms' AND removed_at IS NULL
			AND reason IN ('opted_out','manual') AND (source_provider_id IS NULL OR source_provider_id <> $2)`, p.WorkspaceID, p.ID)
		if err == nil {
			phones, _ := pgx.CollectRows(rows, pgx.RowTo[string])
			if len(phones) > 0 {
				if err := sy.AddOptOuts(ctx, phones); err != nil {
					status, detail = "failing", answerOf(err)
				} else {
					pushed = int32(len(phones))
				}
			}
		}
	}
	if !reads && mode == "off" {
		status = "in_sync"
	}
	return s.Q.SmsOptOutSyncUpsert(ctx, store.SmsOptOutSyncUpsertParams{ProviderID: p.ID, WorkspaceID: p.WorkspaceID, Mode: mode, Pulled: pulled, Pushed: pushed,
		Status: status, Detail: detail, LastSyncAt: &now})
}
