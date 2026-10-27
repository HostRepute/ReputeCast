package demo

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedBrandKit = seedBrandKitImpl }

//go:embed assets/brandkit/*.png
var brandKitAssets embed.FS

// The brand kit of the BrandKit board (review 2026-10, CO-11): Acme's logo
// in light and dark versions, the publisher colours (#4338CA buttons and
// links, #1F2937 text, #F5F3FF behind the email), Source Serif 4 and Inter
// from acme.io with their email-safe fallbacks, rounded 15 px buttons, and
// the synced Header (last edited by Sofia Rossi) and Footer (last edited by
// Linh Nguyen) that Weekly Digest #142 holds (composer_content.go). It goes
// through the brand kit service's own writers; like the rest of the demo's
// history, the audit log keeps the board's events only (seed_audit.go).

const (
	brandFontSerif = "https://fonts.googleapis.com/css2?family=Source+Serif+4:wght@400;600;700&display=swap"
	brandFontInter = "https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&display=swap"
)

// brandHeaderMJML is the synced Header: Weekly Digest #142's header with
// the kit's logo, 120 px wide as the board says.
func brandHeaderMJML(logoURL string) string {
	return `<mj-section background-color="#ffffff" padding="20px 24px" border-bottom="1px solid #EEF0F3">
      <mj-column width="40%" vertical-align="middle">
        <mj-image src="` + logoURL + `" alt="Acme" width="120px" align="left" padding="0px" rc-logo="light"></mj-image>
      </mj-column>
      <mj-column width="60%" vertical-align="middle">
        <mj-text align="right" font-family="` + composerPlexFont + `" font-size="12px" color="#5B6472"><a href="https://acme.io/blog" style="color:#5B6472;text-decoration:none">Blog</a> · <a href="https://docs.acme.io" style="color:#5B6472;text-decoration:none">Docs</a> · <a href="https://acme.io/changelog" style="color:#5B6472;text-decoration:none">Changelog</a></mj-text>
      </mj-column>
    </mj-section>`
}

// brandFooterMJML is the synced Footer: Weekly Digest #142's footer (its
// section, without the block's marker comment).
func brandFooterMJML() string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(composerDigestFooter), "<!-- rc:footer -->"))
}

func seedBrandKitImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh, sofia := s.Users["linh@acme.io"], s.Users["sofia@acme.io"]
	if linh.ID == uuid.Nil || sofia.ID == uuid.Nil {
		return nil
	}
	light, err := brandLogo(ctx, s, ws, linh.ID, "acme-logo-light.png")
	if err != nil {
		return err
	}
	dark, err := brandLogo(ctx, s, ws, linh.ID, "acme-logo-white.png")
	if err != nil {
		return err
	}
	principal := func(u store.User) *authz.Principal {
		return &authz.Principal{Type: authz.ActorUser, ID: u.ID, UserID: u.ID, Name: u.Name, Email: u.Email, WorkspaceID: ws.ID}
	}
	at := s.designDay(10, 9, 14)
	svc := &brandkit.Service{Q: s.Q, Now: func() time.Time { return at }}

	// Linh saves the kit (the import from acme.io): the synced Header and
	// Footer start from it, the Header with the logo.
	in := brandkit.BrandKitInput{
		LogoLightID: light.String(), LogoDarkID: dark.String(), LogoAlt: "Acme", Website: "acme.io",
		BrandStyle: brandkit.BrandStyle{
			Colors:  brandkit.BrandColors{Button: "#4338CA", Text: "#1F2937", Background: "#F5F3FF", Link: "#4338CA"},
			Heading: brandkit.BrandFont{Name: "Source Serif 4", URL: brandFontSerif, Fallback: "Georgia, 'Times New Roman', serif"},
			Body:    brandkit.BrandFont{Name: "Inter", URL: brandFontInter, Fallback: "Arial, Helvetica, sans-serif"},
			Button:  brandkit.BrandButtonStyle{Shape: "rounded", Size: 15},
		},
	}
	saved, err := svc.SaveIn(ctx, s.Q, &ws, principal(linh), in)
	if err != nil {
		return fmt.Errorf("brand kit: %w", err)
	}
	if saved.After.LogoLight == nil {
		return fmt.Errorf("brand kit: no light logo")
	}

	// Sofia gives the Header the digest's links; Linh last edits the Footer
	// (BrandKit board: "edited 2 Oct by Sofia Rossi", "edited 7 Oct by Linh Nguyen").
	at = s.designDay(10, 11, 32)
	if _, _, err := svc.SaveSyncedIn(ctx, s.Q, &ws, principal(sofia), brandkit.KindHeader, brandkit.SyncedInput{MJML: brandHeaderMJML(saved.After.LogoLight.URL)}); err != nil {
		return fmt.Errorf("synced header: %w", err)
	}
	at = s.designDay(5, 15, 6)
	if _, _, err := svc.SaveSyncedIn(ctx, s.Q, &ws, principal(linh), brandkit.KindFooter, brandkit.SyncedInput{MJML: brandFooterMJML()}); err != nil {
		return fmt.Errorf("synced footer: %w", err)
	}
	return nil
}

// brandLogo puts a logo in the workspace's image library through the
// library's own writer, uploaded before the composer's sample images so
// the library reads as before.
func brandLogo(ctx context.Context, s *Seeder, ws store.Workspace, by uuid.UUID, name string) (uuid.UUID, error) {
	data, err := brandKitAssets.ReadFile("assets/brandkit/" + name)
	if err != nil {
		return uuid.Nil, err
	}
	ctype, w, h, alpha, err := media.Inspect(data)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s: %w", name, err)
	}
	sum := sha256.Sum256(data)
	row, err := s.Q.InsertMediaAsset(ctx, store.InsertMediaAssetParams{
		ID: ID(), WorkspaceID: ws.ID, FileName: name, ContentType: ctype, SizeBytes: int32(len(data)), Width: int32(w), Height: int32(h), HasAlpha: alpha,
		Sha256: sum[:], Data: data, CreatedBy: &by, CreatedAt: s.Ago(31 * 24 * time.Hour),
	})
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}
