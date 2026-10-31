package brandkit_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The kit needs templates:edit, but changing the emails that hold its
// synced Header needs campaigns:edit, as changing the Header itself does
// (save-synced-block), and an import adds logos to the media library only
// for those who may upload images (upload-media) (review 2026-10). Here
// Daniel's custom role, Content Writer, holds templates:edit without
// campaigns:edit.
func TestKitWithoutCampaignsEdit(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	query := func(sql string, args ...any) (out string) {
		t.Helper()
		if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&out); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return out
	}

	var k kit
	linh.Do("GET", ws+"/brand-kit", nil).MustStatus(t, 200).JSON(t, &k)
	header := k.Synced[1]
	draft := campaign(t, linh, ws, `<mjml><mj-body>`+header.MJML+`<mj-section><mj-column><mj-text>News</mj-text></mj-column></mj-section></mj-body></mjml>`)
	logo := upload(t, linh, ws, "acme-logo-2.png", logoPNG(t))
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE roles SET permissions = array_remove(permissions, 'campaigns:edit') WHERE key = 'content-writer'`); err != nil {
		t.Fatal(err)
	}
	daniel := env.Client()
	daniel.SignInDemo("daniel@acme.io")

	// Daniel's new logo reaches the kit and its synced Header, not the
	// draft holding the Header.
	in := input(logo)
	daniel.Do("PUT", ws+"/brand-kit", in).MustStatus(t, 200).JSON(t, &k)
	if k.LogoLight == nil || k.LogoLight.MediaID != logo || k.Synced[1].Version != header.Version+1 || !strings.Contains(k.Synced[1].MJML, k.LogoLight.URL) {
		t.Fatalf("kit: %+v", k)
	}
	if m := query(`SELECT mjml FROM campaigns WHERE id = $1`, draft); strings.Contains(m, k.LogoLight.URL) {
		t.Fatalf("the draft changed: %s", m)
	}
	if n := query(`SELECT count(*)::text FROM campaign_versions WHERE campaign_id = $1 AND source = 'synced'`, draft); n != "0" {
		t.Fatalf("%s synced versions", n)
	}
	// Linh's next change to the Header brings the draft up to date.
	in["logoAlt"] = "Acme, Inc."
	linh.Do("PUT", ws+"/brand-kit", in).MustStatus(t, 200)
	if m := query(`SELECT mjml FROM campaigns WHERE id = $1`, draft); !strings.Contains(m, k.LogoLight.URL) || !strings.Contains(m, `alt="Acme, Inc."`) {
		t.Fatalf("the draft after Linh's change: %s", m)
	}

	// Daniel's import reads the site's colours and fonts but adds no logo
	// to the library, and says why.
	srv := fakeSite(t, append(logoPNG(t), "another file"...))
	logos := `SELECT count(*)::text FROM media_assets WHERE file_name = 'acme-logo.png'` // the site's logo, and the demo's
	before := query(logos)
	var imp struct {
		LogoLight *struct{ MediaID string }
		Style     struct{ Colors struct{ Button string } }
		Notes     []string
	}
	daniel.Do("POST", ws+"/brand-kit/import", map[string]any{"website": srv.URL}).MustStatus(t, 200).JSON(t, &imp)
	if imp.LogoLight != nil || imp.Style.Colors.Button != "#4338CA" {
		t.Fatalf("import: %+v", imp)
	}
	if !strings.Contains(strings.Join(imp.Notes, "\n"), "wasn't added to the media library: only people who edit campaigns can add images") {
		t.Fatalf("notes: %q", imp.Notes)
	}
	if n := query(logos); n != before {
		t.Fatalf("the site's logo was added to the media library: %s images, were %s", n, before)
	}
}
