package workspaces

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

const exportReadme = `ReputeCast export of %s
Created %s by %s.

contacts.csv       every contact with status, engagement tier and list memberships
consent.csv        the consent history: subscribe, confirm, unsubscribe, import, erase
lists.csv          lists and their double opt-in setting
suppressions.csv   the workspace suppression list with reasons and provider codes
campaigns.csv      campaigns with subject, status and headline results
campaigns/*.mjml   the MJML source of every campaign
reports.csv        results per campaign, sending provider and mailbox provider
audit-log.json     the workspace audit log%s

Provider credentials, API keys and signing secrets are never exported.
`

// Export writes everything in the workspace as a ZIP archive. It streams
// table by table so large workspaces don't need memory.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, p *authz.Principal, w io.Writer) error {
	zw := zip.NewWriter(w)
	auditNote := ""
	if !p.Can(authz.AuditView) {
		auditNote = " (left out: it needs audit:view)"
	}
	now := s.now().In(loc(ws.Timezone))
	if err := writeFile(zw, "README.txt", fmt.Sprintf(exportReadme, ws.Name, now.Format("Jan 2, 2006 15:04 MST"), p.Name, auditNote)); err != nil {
		return err
	}
	tables := []struct {
		name   string
		header []string
		sql    string
	}{
		{"contacts.csv", []string{"email", "first_name", "last_name", "status", "tier", "source", "country", "timezone", "tags", "lists", "attributes", "created_at", "last_open_at", "last_click_at"},
			`SELECT c.email, c.first_name, c.last_name, c.status, c.tier, c.source, c.country, c.timezone, array_to_string(c.tags, ';'),
			        coalesce((SELECT string_agg(l.name || ' (' || lm.status || ')', '; ' ORDER BY l.name) FROM list_memberships lm JOIN lists l ON l.id = lm.list_id WHERE lm.contact_id = c.id), ''),
			        c.attributes::text, to_char(c.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			        coalesce(to_char(c.last_open_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''), coalesce(to_char(c.last_click_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
			 FROM contacts c WHERE c.workspace_id = $1 AND c.erased_at IS NULL ORDER BY c.email`},
		{"consent.csv", []string{"email", "list", "action", "source", "ip", "user_agent", "detail", "occurred_at"},
			`SELECT c.email, coalesce(l.name, ''), e.action, e.source, coalesce(host(e.ip), ''), e.user_agent, e.detail, to_char(e.occurred_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
			 FROM consent_events e JOIN contacts c ON c.id = e.contact_id LEFT JOIN lists l ON l.id = e.list_id
			 WHERE c.workspace_id = $1 ORDER BY e.occurred_at`},
		{"lists.csv", []string{"name", "description", "double_opt_in", "requires_approval", "created_at"},
			`SELECT name, description, double_opt_in::text, requires_approval::text, to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM lists WHERE workspace_id = $1 AND deleted_at IS NULL ORDER BY created_at`},
		{"suppressions.csv", []string{"email", "reason", "source", "original_code", "detail", "created_at"},
			`SELECT email, reason, source, original_code, detail, to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') FROM suppressions WHERE workspace_id = $1 ORDER BY created_at`},
		{"campaigns.csv", []string{"name", "status", "subject", "preheader", "audience_count", "sent", "delivered", "human_opens", "privacy_protected_opens", "machine_opens", "clicks", "bounces", "complaints", "unsubscribes", "scheduled_at", "started_at", "completed_at"},
			`SELECT c.name, c.status, c.subject, c.preheader, coalesce(c.audience_count::text, ''),
			        coalesce(sum(k.sent), 0)::text, coalesce(sum(k.delivered), 0)::text, coalesce(sum(k.opened_human), 0)::text, coalesce(sum(k.opened_privacy), 0)::text, coalesce(sum(k.opened_machine), 0)::text,
			        coalesce(sum(k.clicked), 0)::text, coalesce(sum(k.hard_bounced + k.soft_bounced), 0)::text, coalesce(sum(k.complained), 0)::text, coalesce(sum(k.unsubscribed), 0)::text,
			        coalesce(to_char(c.scheduled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''), coalesce(to_char(c.started_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
			        coalesce(to_char(c.completed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
			 FROM campaigns c LEFT JOIN campaign_counters k ON k.campaign_id = c.id
			 WHERE c.workspace_id = $1 AND c.deleted_at IS NULL GROUP BY c.id ORDER BY c.created_at`},
		{"reports.csv", []string{"campaign", "sending_provider", "mailbox_provider", "sent", "delivered", "human_opens", "privacy_protected_opens", "machine_opens", "clicks", "hard_bounces", "soft_bounces", "complaints", "unsubscribes"},
			`SELECT c.name, coalesce(p.name, ''), k.mailbox_provider, k.sent::text, k.delivered::text, k.opened_human::text, k.opened_privacy::text, k.opened_machine::text, k.clicked::text,
			        k.hard_bounced::text, k.soft_bounced::text, k.complained::text, k.unsubscribed::text
			 FROM campaign_counters k JOIN campaigns c ON c.id = k.campaign_id LEFT JOIN providers p ON p.id = k.provider_id
			 WHERE c.workspace_id = $1 ORDER BY c.created_at, p.name, k.mailbox_provider`},
	}
	for _, t := range tables {
		if err := s.csvFile(ctx, zw, t.name, t.header, t.sql, ws.ID); err != nil {
			return fmt.Errorf("%s: %w", t.name, err)
		}
	}
	if err := s.mjmlFiles(ctx, zw, ws); err != nil {
		return err
	}
	if p.Can(authz.AuditView) {
		if err := s.auditFile(ctx, zw, ws); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	return s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "workspace.exported", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
		Metadata: audit.KeepSummary(map[string]any{"permission": string(authz.SubscribersExport)}, keep("audit.exported", nil))})
}

func writeFile(zw *zip.Writer, name, content string) error {
	f, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.WriteString(f, content)
	return err
}

func (s *Service) csvFile(ctx context.Context, zw *zip.Writer, name string, header []string, sql string, wsID any) error {
	f, err := zw.Create(name)
	if err != nil {
		return err
	}
	cw := csv.NewWriter(f)
	if err := cw.Write(header); err != nil {
		return err
	}
	rows, err := s.Pool.Query(ctx, sql, wsID)
	if err != nil {
		return err
	}
	defer rows.Close()
	vals := make([]string, len(header))
	ptrs := make([]any, len(header))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		if err := cw.Write(vals); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

var fileRe = regexp.MustCompile(`[^a-z0-9]+`)

func (s *Service) mjmlFiles(ctx context.Context, zw *zip.Writer, ws *store.Workspace) error {
	rows, err := s.Pool.Query(ctx, `SELECT name, mjml, created_at FROM campaigns WHERE workspace_id = $1 AND deleted_at IS NULL AND mjml <> '' ORDER BY created_at`, ws.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	seen := map[string]int{}
	for rows.Next() {
		var name, mjml string
		var created time.Time
		if err := rows.Scan(&name, &mjml, &created); err != nil {
			return err
		}
		base := strings.Trim(fileRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
		if base == "" {
			base = "campaign"
		}
		seen[base]++
		if seen[base] > 1 {
			base = fmt.Sprintf("%s-%d", base, seen[base])
		}
		if err := writeFile(zw, "campaigns/"+base+".mjml", mjml); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Service) auditFile(ctx context.Context, zw *zip.Writer, ws *store.Workspace) error {
	f, err := zw.Create("audit-log.json")
	if err != nil {
		return err
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name, result, reason,
		coalesce(host(ip), ''), user_agent, request_id, before, after, metadata
		FROM audit_events WHERE workspace_id = $1 ORDER BY occurred_at, id`, ws.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if _, err := io.WriteString(f, "[\n"); err != nil {
		return err
	}
	n := 0
	for rows.Next() {
		var e struct {
			ID         string          `json:"id"`
			OccurredAt time.Time       `json:"occurred_at"`
			ActorType  string          `json:"actor_type"`
			ActorID    *string         `json:"actor_id,omitempty"`
			ActorName  string          `json:"actor_name"`
			Action     string          `json:"action"`
			TargetType string          `json:"target_type"`
			TargetID   string          `json:"target_id"`
			TargetName string          `json:"target_name"`
			Result     string          `json:"result"`
			Reason     string          `json:"reason,omitempty"`
			IP         string          `json:"ip,omitempty"`
			UserAgent  string          `json:"user_agent,omitempty"`
			RequestID  string          `json:"request_id,omitempty"`
			Before     json.RawMessage `json:"before,omitempty"`
			After      json.RawMessage `json:"after,omitempty"`
			Metadata   json.RawMessage `json:"metadata,omitempty"`
		}
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.ActorType, &e.ActorID, &e.ActorName, &e.Action, &e.TargetType, &e.TargetID, &e.TargetName,
			&e.Result, &e.Reason, &e.IP, &e.UserAgent, &e.RequestID, &e.Before, &e.After, &e.Metadata); err != nil {
			return err
		}
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if n > 0 {
			if _, err := io.WriteString(f, ",\n"); err != nil {
				return err
			}
		}
		if _, err := f.Write(b); err != nil {
			return err
		}
		n++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = io.WriteString(f, "\n]\n")
	return err
}

func loc(tz string) *time.Location {
	if l, err := time.LoadLocation(tz); err == nil {
		return l
	}
	return time.UTC
}

var _ = pgx.ErrNoRows
