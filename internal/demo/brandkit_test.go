package demo_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/brandkit"
)

// TestBrandKitDemo reads Acme's brand kit back through the brand kit API
// (BrandKit board): the logos, the colours and their contrast, the fonts,
// the button, and the synced Header and Footer that Weekly Digest #142
// holds (Composer and ComposerWrite boards).
func TestBrandKitDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	type logo struct {
		FileName      string
		URL           string
		Width, Height int
	}
	var k struct {
		Saved               bool
		LogoLight, LogoDark *logo
		LogoAlt, Website    string
		UpdatedBy           string
		Colors              struct{ Button, Text, Background, Link string }
		Heading, Body       struct{ Name, URL, Fallback string }
		Button              struct {
			Shape string
			Size  int
		}
		Synced []struct {
			Kind, MJML, UpdatedBy string
			UpdatedAt             time.Time
			Version               int
			Uses                  struct{ Drafts, Scheduled, Held, Sent int }
		}
	}
	c.Do("GET", "/workspaces/acme-newsletter/brand-kit", nil).MustStatus(t, 200).JSON(t, &k)
	if !k.Saved || k.LogoAlt != "Acme" || k.Website != "acme.io" || k.UpdatedBy != "Linh Nguyen" {
		t.Fatalf("kit: %+v", k)
	}
	if k.LogoLight == nil || k.LogoLight.FileName != "acme-logo-light.png" || k.LogoLight.Width != 240 || k.LogoLight.Height != 64 ||
		k.LogoDark == nil || k.LogoDark.FileName != "acme-logo-white.png" || k.LogoDark.Width != 240 || k.LogoDark.Height != 64 {
		t.Fatalf("logos: %+v %+v", k.LogoLight, k.LogoDark)
	}
	col := k.Colors
	if col.Button != "#4338CA" || col.Text != "#1F2937" || col.Background != "#F5F3FF" || col.Link != "#4338CA" {
		t.Fatalf("colours: %+v", col)
	}
	// The contrast the board shows for each colour where it's used.
	ratio := func(a, b string) string { return fmt.Sprintf("%.1f", brandkit.Contrast(a, b)) }
	if got := []string{ratio("#FFFFFF", col.Button), ratio(col.Text, "#FFFFFF"), ratio(col.Text, col.Background), ratio(col.Link, "#FFFFFF")}; strings.Join(got, " ") != "7.9 14.7 13.4 7.9" {
		t.Fatalf("contrast: %v", got)
	}
	if k.Heading.Name != "Source Serif 4" || k.Heading.Fallback != "Georgia, 'Times New Roman', serif" || !strings.HasPrefix(k.Heading.URL, "https://fonts.googleapis.com/") ||
		k.Body.Name != "Inter" || k.Body.Fallback != "Arial, Helvetica, sans-serif" || k.Button.Shape != "rounded" || k.Button.Size != 15 {
		t.Fatalf("fonts and button: %+v %+v %+v", k.Heading, k.Body, k.Button)
	}

	// Footer and Header, as the board lists them: who edited them last, and
	// Weekly Digest #142 holding both (scheduled and approved: a change
	// waits, so the approved email stays as approved).
	if len(k.Synced) != 2 || k.Synced[0].Kind != "footer" || k.Synced[1].Kind != "header" {
		t.Fatalf("synced: %+v", k.Synced)
	}
	footer, header := k.Synced[0], k.Synced[1]
	if footer.UpdatedBy != "Linh Nguyen" || header.UpdatedBy != "Sofia Rossi" || header.UpdatedAt.AddDate(0, 0, 5).Format("2006-01-02") != footer.UpdatedAt.Format("2006-01-02") {
		t.Fatalf("edited: footer %s %s, header %s %s", footer.UpdatedBy, footer.UpdatedAt, header.UpdatedBy, header.UpdatedAt)
	}
	if !strings.Contains(footer.MJML, `rc-synced="footer"`) || !strings.Contains(footer.MJML, "{{ unsubscribe_url }}") || !strings.Contains(footer.MJML, "500 Market St, Suite 1200") {
		t.Fatalf("footer: %s", footer.MJML)
	}
	if !strings.Contains(header.MJML, `rc-synced="header"`) || !strings.Contains(header.MJML, k.LogoLight.URL) || !strings.Contains(header.MJML, "Changelog") {
		t.Fatalf("header: %s", header.MJML)
	}
	for _, b := range k.Synced {
		if b.Uses.Held != 1 || b.Uses.Drafts != 0 || b.Uses.Scheduled != 0 || b.Uses.Sent != 0 {
			t.Fatalf("%s uses: %+v", b.Kind, b.Uses)
		}
	}
	var mjml, html string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT mjml, html FROM campaigns WHERE name = 'Weekly Digest #142'`).Scan(&mjml, &html); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mjml, `rc-synced="header"`) || !strings.Contains(mjml, `rc-synced="footer"`) || strings.Contains(html, "rc-synced") {
		t.Fatal("Weekly Digest #142 should hold the synced Header and Footer, and its HTML not show the mark")
	}

	// Seeded through the kit's writers, but the audit log keeps the board's
	// events (SettingsAudit).
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('brand_kit.save', 'synced_block.save')`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("audit entries: %d %v", n, err)
	}
}
