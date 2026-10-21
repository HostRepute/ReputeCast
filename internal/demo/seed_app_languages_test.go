package demo

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/testdb"
)

// The app languages of the connect-and-reach sample (brief §4).
func TestSeedAppLanguages(t *testing.T) {
	if testing.Short() {
		t.Skip("seeds the whole demo")
	}
	pool := testdb.New(t)
	cfg := config.Defaults()
	cfg.Files.Dir = t.TempDir() // the sample product files (X19)
	box, err := secrets.New(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := Seed(ctx, pool, &cfg, box, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var lang, region string
	if err := pool.QueryRow(ctx, `SELECT default_language, default_region_format FROM instance_settings`).Scan(&lang, &region); err != nil || lang != "en" || region != "standard" {
		t.Fatalf("instance default: %q %q %v", lang, region, err)
	}
	var marcusLang, marcusRegion *string
	if err := pool.QueryRow(ctx, `SELECT language, region_format FROM users WHERE email = 'marcus@acme.io'`).Scan(&marcusLang, &marcusRegion); err != nil {
		t.Fatal(err)
	}
	if marcusLang == nil || *marcusLang != "en" || marcusRegion == nil || *marcusRegion != "en-US" {
		t.Fatalf("Marcus reads English with the US region format: %v %v", marcusLang, marcusRegion)
	}
	var chosen int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email <> 'marcus@acme.io' AND (language IS NOT NULL OR region_format IS NOT NULL)`).Scan(&chosen); err != nil || chosen != 0 {
		t.Fatalf("everyone else follows the instance default: %d %v", chosen, err)
	}
	var inviteLang *string
	if err := pool.QueryRow(ctx, `SELECT language FROM invitations WHERE email = 'jo@contractor.dev'`).Scan(&inviteLang); err != nil || inviteLang == nil || *inviteLang != "en" {
		t.Fatalf("jo@contractor.dev's invitation is in English: %v %v", inviteLang, err)
	}
}
