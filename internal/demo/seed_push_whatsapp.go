package demo

import (
	"context"

	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
)

// Web push and WhatsApp (design expansion phase 8, X22; brief §4 "Web push"
// and "WhatsApp"). The demo's push sites and WhatsApp account are marked
// demo: their notifications go to the in-process push service and their
// messages to the in-process Graph API, never to FCM, Mozilla, Apple or
// Meta.

// The demo's WhatsApp Business Account (…4410) and its credentials, which
// only the in-process Graph API (internal/whatsapp/meta/metatest) accepts.
const (
	WhatsAppBusinessID = "102290129344410"
	WhatsAppPhoneID    = "106540352242922"
	WhatsAppToken      = "demo-system-user-token"
	WhatsAppAppSecret  = "demo-app-secret"
)

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
			{Type: "BUTTONS", Buttons: []meta.TemplateButton{{Type: "URL", Text: "View the agenda", URL: "https://news.acme.io/summit/agenda"}}},
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

func init() {
	seedPushWhatsApp = func(ctx context.Context, s *Seeder) error {
		return nil
	}
}
