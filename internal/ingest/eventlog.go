package ingest

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// EventLog serves the provider event log.
type EventLog struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Now   func() time.Time
}

func (e *EventLog) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// ProviderLogEvent is one row of the event log: the raw provider event beside
// the normalized ReputeCast event and the original code.
type ProviderLogEvent struct {
	ID         string    `json:"id"`
	ReceivedAt time.Time `json:"receivedAt"`
	OccurredAt time.Time `json:"occurredAt"`
	ProviderID string    `json:"providerId,omitempty"`
	Provider   string    `json:"provider"`
	// Channel is how the event arrived: webhook, bounce_mailbox or backfill.
	Channel       string `json:"channel" enum:"webhook,bounce_mailbox,backfill,test,internal"`
	RawType       string `json:"rawType"`
	Type          string `json:"type" enum:"delivered,deferred,hard_bounce,soft_bounce,complaint,rejected"`
	Code          string `json:"code"`
	Diagnostic    string `json:"diagnostic"`
	BounceClass   string `json:"bounceClass"`
	Recipient     string `json:"recipient" doc:"Masked"`
	ProviderMsgID string `json:"providerMessageId"`
	Source        string `json:"source"`
	CampaignID    string `json:"campaignId,omitempty"`
}

// ProviderEventLog is one page of the event log.
type ProviderEventLog struct {
	Items         []ProviderLogEvent `json:"items"`
	Total         int64              `json:"total"`
	Page          int                `json:"page"`
	PageSize      int                `json:"pageSize"`
	RetentionDays int                `json:"retentionDays"`
}

// LogFilter filters the event log.
type LogFilter struct {
	Provider string
	Type     string
	Query    string
	Page     int
	PageSize int
}

func (e *EventLog) params(ctx context.Context, ws *store.Workspace, f LogFilter) (store.IngestEventLogParams, error) {
	p := store.IngestEventLogParams{WorkspaceID: ws.ID, Since: e.now().AddDate(0, 0, -RetentionDays), Type: f.Type, Query: f.Query}
	if f.Provider != "" {
		id, err := uuid.Parse(f.Provider)
		if err != nil {
			return p, api.Invalid("query.provider", msgs.M("problem.pickProvider").Text(i18n.LangFrom(ctx), nil))
		}
		p.ProviderID = &id
	}
	return p, nil
}

// List returns one page of provider events, newest first.
func (e *EventLog) List(ctx context.Context, ws *store.Workspace, f LogFilter) (ProviderEventLog, error) {
	if f.PageSize <= 0 {
		f.PageSize = 25
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	p, err := e.params(ctx, ws, f)
	if err != nil {
		return ProviderEventLog{}, err
	}
	p.Lim, p.Off = int32(f.PageSize), int32((f.Page-1)*f.PageSize)
	rows, err := e.Q.IngestEventLog(ctx, p)
	if err != nil {
		return ProviderEventLog{}, err
	}
	total, err := e.Q.IngestEventLogCount(ctx, store.IngestEventLogCountParams{WorkspaceID: p.WorkspaceID, Since: p.Since, ProviderID: p.ProviderID, Type: p.Type, Query: p.Query})
	if err != nil {
		return ProviderEventLog{}, err
	}
	out := ProviderEventLog{Items: make([]ProviderLogEvent, 0, len(rows)), Total: total, Page: f.Page, PageSize: f.PageSize, RetentionDays: RetentionDays}
	l := i18n.LangFrom(ctx)
	for _, r := range rows {
		out.Items = append(out.Items, eventOf(l, r, true))
	}
	return out, nil
}

// eventOf is one row of the log, in a reader's language (the export's is
// English: its rows are data).
func eventOf(l i18n.Lang, r store.IngestEventLogRow, mask bool) ProviderLogEvent {
	ev := ProviderLogEvent{ID: r.ID.String(), ReceivedAt: r.ReceivedAt, OccurredAt: r.OccurredAt, Provider: r.ProviderName, Channel: r.RawSource,
		RawType: r.OriginalType, Type: r.Type, Code: r.OriginalCode, Diagnostic: r.Diagnostic, BounceClass: r.BounceClass,
		Recipient: r.Recipient, ProviderMsgID: r.ProviderMessageID, Source: sourceIn(l, r)}
	if mask {
		ev.Recipient = manage.MaskEmail(r.Recipient)
	}
	if r.ProviderID != nil {
		ev.ProviderID = r.ProviderID.String()
	}
	if r.CampaignID != nil {
		ev.CampaignID = r.CampaignID.String()
	}
	if ev.Channel == "" {
		ev.Channel = "internal"
	}
	if ev.Channel == "bounce_mailbox" {
		ev.Provider = msgs.M("provider.bounceMailbox").Text(l, nil)
	}
	if ev.Source == "" {
		ev.Source = "–"
	}
	return ev
}

// Export returns the filtered log as CSV (up to 50,000 rows). Recipients
// are shown in full only to members who may export contacts.
func (e *EventLog) Export(ctx context.Context, ws *store.Workspace, f LogFilter) ([]byte, error) {
	p, err := e.params(ctx, ws, f)
	if err != nil {
		return nil, err
	}
	pr := api.PrincipalFrom(ctx)
	full := pr != nil && pr.Can(authz.SubscribersExport)
	p.Lim, p.Off = 50000, 0
	rows, err := e.Q.IngestEventLog(ctx, p)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"received_at", "occurred_at", "sending_provider", "raw_event", "normalized", "original_code", "diagnostic", "bounce_class", "recipient", "provider_message_id", "source"})
	for _, r := range rows {
		ev := eventOf(i18n.Default, r, !full)
		_ = w.Write([]string{ev.ReceivedAt.UTC().Format(time.RFC3339), ev.OccurredAt.UTC().Format(time.RFC3339), ev.Provider, ev.RawType, ev.Type, ev.Code, ev.Diagnostic,
			ev.BounceClass, ev.Recipient, ev.ProviderMsgID, ev.Source})
	}
	w.Flush()
	wsID := ws.ID
	if err := e.Audit.Log(ctx, audit.Event{WorkspaceID: &wsID, Action: "provider_events.export", TargetType: "provider_events", TargetID: ws.ID.String(), TargetName: ws.Name,
		Metadata: map[string]any{"rows": len(rows), "full_recipients": full, "provider": f.Provider, "type": f.Type, "query": f.Query}}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type logInput struct {
	api.WorkspaceInput
	Provider string `query:"provider" doc:"Sending provider id"`
	Type     string `query:"type" enum:"delivered,deferred,hard_bounce,soft_bounce,complaint,rejected" required:"false"`
	Query    string `query:"q" maxLength:"254" doc:"Recipient or provider message id"`
	Page     int    `query:"page" minimum:"1" default:"1"`
	PageSize int    `query:"pageSize" minimum:"1" maximum:"100" default:"25"`
}

type exportOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	Body               []byte
}

// RegisterLog adds the event log operations.
func RegisterLog(h huma.API, e *EventLog) {
	tags := []string{"Sending providers"}
	api.Get(h, api.Op{ID: "list-provider-events", Path: "/workspaces/{workspace}/provider-events", Summary: "Provider events beside their normalized form",
		Description: "Recipients are masked. Events are kept " + strconv.Itoa(RetentionDays) + " days.", Tags: tags, Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, in *logInput) (*api.Body[ProviderEventLog], error) {
			out, err := e.List(ctx, api.WorkspaceFrom(ctx), LogFilter{Provider: in.Provider, Type: in.Type, Query: in.Query, Page: in.Page, PageSize: in.PageSize})
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-provider-events", Path: "/workspaces/{workspace}/provider-events/export", Summary: "Export the provider event log as CSV",
		Description: "Full recipient addresses need subscribers:export; otherwise they are masked. Every export is audited.", Tags: tags, Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, in *logInput) (*exportOutput, error) {
			b, err := e.Export(ctx, api.WorkspaceFrom(ctx), LogFilter{Provider: in.Provider, Type: in.Type, Query: in.Query})
			if err != nil {
				return nil, err
			}
			name := "provider-events-" + time.Now().UTC().Format("2006-01-02") + ".csv"
			return &exportOutput{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, Body: b}, nil
		})
}
