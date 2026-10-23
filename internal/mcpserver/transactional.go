package mcpserver

import (
	"context"
	"errors"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Transactional email: agents list templates, render previews and read the
// message log without message content. They never send.

type renderTransactionalIn struct {
	Template string         `json:"template" jsonschema:"the template's ID, such as password-reset"`
	Version  string         `json:"version,omitempty" jsonschema:"live (default), draft, or a version number"`
	TestData string         `json:"test_data,omitempty" jsonschema:"the name of one of the template's test data sets"`
	Data     map[string]any `json:"data,omitempty" jsonschema:"variables to render with instead of a test data set; nothing is stored or sent"`
}

type searchTransactionalIn struct {
	Status   string `json:"status,omitempty" jsonschema:"queued, sent, delivered, deferred, bounced, complained, rejected, failed or test (default all)"`
	Template string `json:"template,omitempty" jsonschema:"a template's ID, or raw for mail that arrived by SMTP"`
	Query    string `json:"query,omitempty" jsonschema:"recipient, message ID, idempotency key, tag or metadata value"`
	Range    string `json:"range,omitempty" jsonschema:"1h, 24h (default), 7d or 30d"`
	Limit    int    `json:"limit,omitempty" jsonschema:"at most this many messages (default 25, max 100)"`
}

type listTransactionalIn struct {
	Period string `json:"period,omitempty" jsonschema:"24h, 7d (default) or 30d: the period of the numbers"`
}

// addTransactionalTools registers list_transactional_templates,
// render_transactional_template and search_transactional_messages.
func (s *Service) addTransactionalTools(srv *mcp.Server) {
	tool(s, srv, "list_transactional_templates", authz.TransactionalView, func(ctx context.Context, c *agentCall, in listTransactionalIn) (any, entry, error) {
		e := entry{action: "transactional.templates_listed", targetType: "transactional_template", targetText: keep("target.templates", nil)}
		if s.Transactional == nil {
			return nil, e, errors.New("transactional email isn't available in this process")
		}
		ws := c.Workspace
		list, err := s.Transactional.Templates(ctx, &ws, c.Principal, in.Period, false)
		if err != nil {
			return nil, e, err
		}
		items := make([]map[string]any, 0, len(list.Items))
		for _, t := range list.Items {
			m := map[string]any{"id": t.Key, "name": t.Name, "draft_changes": t.DraftChanges, "archived": t.Archived,
				"sent": t.Stats.Sent, "delivery_rate": t.Stats.DeliveryRate, "bounce_rate": t.Stats.BounceRate, "complaints": t.Stats.Complained}
			if t.Category != nil {
				m["category"], m["category_kind"] = t.Category.Name, t.Category.Kind
			}
			if t.LiveVersion != nil {
				m["live_version"] = *t.LiveVersion
			}
			if t.Stats.MedianMs != nil {
				m["median_delivery_ms"] = *t.Stats.MedianMs
			}
			items = append(items, m)
		}
		e.detail = list.Period
		return map[string]any{"period": list.Period, "templates": items,
			"note": "The send API always sends a template's live version. Agents never send: apps send with a send-only key."}, e, nil
	}, nil)

	tool(s, srv, "render_transactional_template", authz.TransactionalView, func(ctx context.Context, c *agentCall, in renderTransactionalIn) (any, entry, error) {
		e := entry{action: "transactional.template_previewed", targetType: "transactional_template", targetName: in.Template}
		if s.Transactional == nil {
			return nil, e, errors.New("transactional email isn't available in this process")
		}
		version := strings.TrimSpace(in.Version)
		if version == "" {
			version = "live"
		}
		ws := c.Workspace
		out, err := s.Transactional.Preview(ctx, &ws, in.Template, transactional.TransactionalPreviewInput{Version: version, TestData: in.TestData, Data: in.Data})
		if err != nil {
			return nil, e, err
		}
		res := map[string]any{"template": in.Template, "version": out.Version, "subject": out.Subject, "preheader": out.Preheader, "html": out.HTML, "text": out.Text,
			"from": out.FromEmail, "ok": out.OK, "missing": out.Missing, "note": "A preview: nothing was sent or stored."}
		if out.Problem != "" {
			res["problem"] = out.Problem
		}
		e.detailText = keep("detail.version", i18n.Args{"version": out.Version})
		return res, e, nil
	}, nil)

	tool(s, srv, "search_transactional_messages", authz.TransactionalView, func(ctx context.Context, c *agentCall, in searchTransactionalIn) (any, entry, error) {
		e := entry{action: "transactional.log_read", targetType: "transactional_log", targetText: keep("target.messageLog", nil)}
		if s.Transactional == nil {
			return nil, e, errors.New("transactional email isn't available in this process")
		}
		r := in.Range
		if r == "" || r == "custom" {
			r = "24h"
		}
		ws := c.Workspace
		log, err := s.Transactional.MessageLog(ctx, &ws, transactional.LogFilter{Status: in.Status, Template: in.Template, Query: in.Query, Range: r, Limit: in.Limit})
		if err != nil {
			return nil, e, err
		}
		items := make([]map[string]any, 0, len(log.Items))
		for _, it := range log.Items {
			m := map[string]any{"id": it.ID, "at": it.At, "to": it.Email, "subject": it.Subject, "status": it.Status}
			if it.Template != nil {
				m["template"] = it.Template.Key
				if it.Template.Version != nil {
					m["version"] = *it.Template.Version
				}
			} else {
				m["template"] = "raw (SMTP)"
			}
			if it.Provider != nil {
				m["provider"] = it.Provider.Name
			}
			if it.DeliveryMs != nil {
				m["delivery_ms"] = *it.DeliveryMs
			}
			if it.Rejection != nil {
				m["rejected"] = it.Rejection.Reason
			}
			if it.Bounce != nil {
				m["bounce"] = it.Bounce.Kind + ": " + it.Bounce.Diagnostic
			}
			items = append(items, m)
		}
		e.detail = r
		return map[string]any{"messages": items, "counts": log.Counts,
			"note": "Message content isn't available to agents. Messages are kept 90 days."}, e, nil
	}, nil)
}
