package demo

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
)

// Web push and WhatsApp (design expansion phase 8, X22; brief §4 "Web push"
// and "WhatsApp"). The demo's push sites and WhatsApp account are marked
// demo: their notifications go to the in-process push service and their
// messages to the in-process Graph API, never to FCM, Mozilla, Apple or
// Meta.
//
// Days are the design's: today is Wednesday 15 October (designDay(0)).
//
//   - Push: keys created 3 Oct by Marcus Chen; news.acme.io (built in,
//     2,946 subscribers) and acme.io (snippet installed 5 Oct, rc-push-sw.js
//     version 1 found at the 15 Oct 08:00 check, 1,318); 4,264 subscribers
//     by browser, 2,118 linked to contacts (612 of them Acme Summit 2026
//     registrants, Sam Rivera among them); the frequency cap of 2 a day;
//     the slide-in bar on news.acme.io (v3, published 8 Oct by Sofia Rossi)
//     and the bell on acme.io with their numbers; the push campaign "New
//     post: Feature flags at scale", sent by Sofia on 14 Oct at 09:05.
//   - WhatsApp: Acme Inc (…4410), +1 312 555 0110, "Acme", quality High,
//     1,000 people a day, connected 8 Oct by Marcus and verified by Meta at
//     09:14; the four templates as synced 15 Oct 08:00; WhatsApp consent of
//     214 registrants; Meta's rate card of 1 Oct 2026; Sofia's draft
//     "Summit: see you tomorrow"; Sam Rivera's conversation, with the
//     customer service window open.
//   - The "Acme Summit reminders" flow gains its Send push and Send
//     WhatsApp steps after the 23 hours 45 minutes wait.

// The demo's WhatsApp Business Account (…4410) and its credentials, which
// only the in-process Graph API (internal/whatsapp/meta/metatest) accepts.
const (
	WhatsAppBusinessID = "102290129344410"
	WhatsAppPhoneID    = "106540352242922"
	WhatsAppToken      = "demo-system-user-token"
	WhatsAppAppSecret  = "demo-app-secret" //nolint:gosec // the demo account's, in the open on purpose
)

//go:embed assets/push
var pushAssets embed.FS

// WhatsAppTemplates are the account's templates as Meta lists them
// (synced 15 Oct 08:00 in the demo).
func WhatsAppTemplates() []meta.Template {
	body := func(text string, examples ...string) meta.TemplateComponent {
		c := meta.TemplateComponent{Type: "BODY", Text: text}
		if len(examples) > 0 {
			c.Example = &meta.TemplateExample{BodyText: [][]string{examples}}
		}
		return c
	}
	return []meta.Template{
		{ID: "1290033512244410", Name: "summit_day_before", Status: "APPROVED", Category: "UTILITY", Language: "en_US", Components: []meta.TemplateComponent{
			body("Hi {{1}}, see you tomorrow at Acme Summit 2026. Doors open at {{2}} at Lakeside Hall, Chicago.", "Sam", "8:30"),
			{Type: "BUTTONS", Buttons: []meta.TemplateButton{{Type: "URL", Text: "View the agenda", URL: "https://news.acme.io/summit"}}},
		}},
		{ID: "1290033512244411", Name: "summit_livestream", Status: "APPROVED", Category: "UTILITY", Language: "en_US", Components: []meta.TemplateComponent{
			body("Acme Summit 2026 starts in 15 minutes. Watch the livestream: {{1}}", "go.acme.io/k3P9xq"),
		}},
		{ID: "1290033512244412", Name: "summit_venue_change", Status: "PENDING", Category: "UTILITY", Language: "en_US", Components: []meta.TemplateComponent{
			body("Acme Summit 2026 has moved to {{1}}. Your registration still stands.", "Harbor Hall, Chicago"),
		}},
		{ID: "1290033512244413", Name: "digest_highlights", Status: "REJECTED", Category: "MARKETING", Language: "en_US", RejectedReason: "INVALID_FORMAT",
			Components: []meta.TemplateComponent{
				body("This week in The Acme newsletter: {{1}}. Read it here: {{2}}"),
				{Type: "BUTTONS", Buttons: []meta.TemplateButton{{Type: "QUICK_REPLY", Text: "Stop promotions"}}},
			}},
	}
}

func init() { seedPushWhatsApp = seedPushWhatsAppImpl }

// pwState carries the seed's ids between its parts.
type pwState struct {
	key            uuid.UUID
	news, acmeSite uuid.UUID
	newsPrompt     uuid.UUID
	bellPrompt     uuid.UUID
	campaign       uuid.UUID
	image          uuid.UUID
	icon           uuid.UUID
	summit         uuid.UUID
	sam            uuid.UUID
	samPhone       string
	account        uuid.UUID
	templates      map[string]uuid.UUID
	waDraft        uuid.UUID
	sent           time.Time
}

// pushDay is a day of the design's October at a New York time: pushDay(14,
// 9, 5) is yesterday 09:05.
func (s *Seeder) pushDay(day, hour, minute int) time.Time {
	return s.designDay(15-day, hour, minute)
}

func seedPushWhatsAppImpl(ctx context.Context, s *Seeder) error {
	st := &pwState{templates: map[string]uuid.UUID{}, summit: s.Lists["Acme Summit 2026"]}
	if st.summit == uuid.Nil {
		return fmt.Errorf("the Acme Summit 2026 list isn't seeded")
	}
	if err := pwAssets(ctx, s, st); err != nil {
		return fmt.Errorf("assets: %w", err)
	}
	if err := pushSetup(ctx, s, st); err != nil {
		return fmt.Errorf("push setup: %w", err)
	}
	if err := pushPeople(ctx, s, st); err != nil {
		return fmt.Errorf("push subscribers: %w", err)
	}
	if err := whatsAppAccount(ctx, s, st); err != nil {
		return fmt.Errorf("whatsapp: %w", err)
	}
	if err := whatsAppPeople(ctx, s, st); err != nil {
		return fmt.Errorf("whatsapp people: %w", err)
	}
	if err := pwAutomation(ctx, s, st); err != nil {
		return fmt.Errorf("automation: %w", err)
	}
	return pwAudit(ctx, s, st)
}

// pwAssets puts the notification icon and the campaign's image in the
// media library.
func pwAssets(ctx context.Context, s *Seeder, st *pwState) error {
	ws := s.WS()
	for _, a := range []struct {
		file string
		id   *uuid.UUID
		by   string
		at   time.Time
	}{{"acme-icon-192.png", &st.icon, "marcus@acme.io", s.pushDay(3, 10, 20)}, {"feature-flags.png", &st.image, "sofia@acme.io", s.pushDay(14, 8, 42)}} {
		data, err := pushAssets.ReadFile("assets/push/" + a.file)
		if err != nil {
			return err
		}
		ctype, wd, ht, alpha, err := media.Inspect(data)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if err := s.Tx.QueryRow(ctx, `INSERT INTO media_assets (id, workspace_id, file_name, content_type, size_bytes, width, height, has_alpha, sha256, data, created_by, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (workspace_id, sha256) DO UPDATE SET file_name = EXCLUDED.file_name RETURNING id`,
			ID(), ws.ID, a.file, ctype, len(data), wd, ht, alpha, sum[:], data, s.Users[a.by].ID, a.at).Scan(a.id); err != nil {
			return err
		}
	}
	return nil
}

// pwAutomation adds the Send push and Send WhatsApp steps to "Acme Summit
// reminders", after the wait to 22 Oct 08:45 Chicago time (brief §4 steps
// 6 and 7).
func pwAutomation(ctx context.Context, s *Seeder, st *pwState) error {
	ws := s.WS()
	var id uuid.UUID
	var version int
	var raw []byte
	if err := s.Tx.QueryRow(ctx, `SELECT a.id, v.version, v.graph FROM automations a JOIN automation_versions v ON v.automation_id = a.id AND v.version = a.live_version
		WHERE a.workspace_id = $1 AND a.name = 'Acme Summit reminders'`, ws.ID).Scan(&id, &version, &raw); err != nil {
		return err
	}
	var g automations.AutomationGraph
	if err := json.Unmarshal(raw, &g); err != nil {
		return err
	}
	summit := st.summit.String()
	push := autoNode("p1", automations.NodePush, 120, 700, autoFlowData{Label: "15 minutes before", Push: &automations.AutomationPushStep{
		Title: "Acme Summit starts in 15 minutes", Message: "Doors are open at Lakeside Hall, and the livestream starts at 09:00 Central.",
		Link: "https://news.acme.io/summit/live", Button: "Watch the livestream", ExpiryMinutes: 30, Outside: "skip"}})
	wa := autoNode("w1", automations.NodeWhatsApp, 120, 860, autoFlowData{Label: "Livestream link", WhatsApp: &automations.AutomationWhatsAppStep{
		Mode: "template", Template: "summit_livestream", Language: "en_US", ConsentListID: summit, Outside: "wait",
		Variables: []automations.AutomationWhatsAppVar{{Key: "1", Source: "text", Text: "go.acme.io/k3P9xq"}}}})
	nodes := []autoFlowNode{}
	for _, n := range g.Nodes {
		if n.ID == "x1" {
			n.Position.Y = 1000
		}
		nodes = append(nodes, n)
		if n.ID == "d1" {
			nodes = append(nodes, push, wa)
		}
	}
	edges := []automations.AutomationEdge{}
	for _, e := range g.Edges {
		if e.Source == "d1" && e.Target == "x1" {
			edges = append(edges, autoEdge("d1", "p1", ""), autoEdge("p1", "w1", ""), autoEdge("w1", "x1", ""))
			continue
		}
		edges = append(edges, e)
	}
	g.Nodes, g.Edges = nodes, edges
	_, err := s.Tx.Exec(ctx, `UPDATE automation_versions SET graph = $3 WHERE automation_id = $1 AND version = $2`, id, version, g.MustJSON())
	return err
}
