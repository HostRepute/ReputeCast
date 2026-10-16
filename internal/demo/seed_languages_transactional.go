package demo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/store"
)

// The content languages of the transactional sample (brief §4) and the
// contacts the composer's "Preview as" names (ComposerLanguages).

// langResetES and langResetDE translate password-reset (its subject,
// preheader, heading, text, button and footer).
var (
	langResetES = []string{
		"Reset your Acme Cloud password", "Restablece tu contraseña de Acme Cloud",
		"This link works for {{ expires_in }}.", "Este enlace funciona durante {{ expires_in }}.",
		"Reset your password", "Restablece tu contraseña",
		`Hi {{ first_name | default: "there" }},`, `Hola, {{ first_name | default: "hola" }}:`,
		"We got a request to reset the password for your Acme Cloud account. The link below works for {{ expires_in }}.",
		"Recibimos una solicitud para restablecer la contraseña de tu cuenta de Acme Cloud. El enlace de abajo funciona durante {{ expires_in }}.",
		"Reset password", "Restablecer contraseña",
		"If you didn't ask for this, you can ignore this email. Your password stays the same.",
		"Si no lo pediste, puedes ignorar este correo. Tu contraseña no cambia.",
		"You get this email because of your Acme Cloud account. Account emails can't be turned off.",
		"Recibes este correo por tu cuenta de Acme Cloud. Los correos de la cuenta no se pueden desactivar.",
	}
	langResetDE = []string{
		"Reset your Acme Cloud password", "Setze dein Acme-Cloud-Passwort zurück",
		"This link works for {{ expires_in }}.", "Dieser Link gilt {{ expires_in }} lang.",
		"Reset your password", "Passwort zurücksetzen",
		`Hi {{ first_name | default: "there" }},`, `Hallo {{ first_name | default: "zusammen" }},`,
		"We got a request to reset the password for your Acme Cloud account. The link below works for {{ expires_in }}.",
		"Wir haben eine Anfrage erhalten, das Passwort deines Acme-Cloud-Kontos zurückzusetzen. Der Link unten gilt {{ expires_in }} lang.",
		"Reset password", "Passwort zurücksetzen",
		"If you didn't ask for this, you can ignore this email. Your password stays the same.",
		"Falls du das nicht angefordert hast, kannst du diese E-Mail ignorieren. Dein Passwort bleibt gleich.",
		"You get this email because of your Acme Cloud account. Account emails can't be turned off.",
		"Du erhältst diese E-Mail wegen deines Acme-Cloud-Kontos. Konto-E-Mails lassen sich nicht abbestellen.",
	}
)

// sampleContacts names the first Weekly Digest reader of each language the
// way the boards' "Preview as" does: Lucía Fernández (Spanish), Jonas
// Weber (German) and Phạm Minh Anh (Vietnamese).
func (l *langSeed) sampleContacts(ctx context.Context) error {
	s := l.s
	digest, ok := s.Lists["Weekly Digest"]
	if !ok {
		return nil
	}
	for _, c := range []struct{ base, first, last, email string }{
		{"es", "Lucía", "Fernández", "lucia.fernandez@example.com"},
		{"de", "Jonas", "Weber", "jonas.weber@example.de"},
		{"vi", "Phạm", "Minh Anh", "minhanh.pham@example.vn"},
	} {
		if _, err := s.Tx.Exec(ctx, `UPDATE contacts SET first_name = $3, last_name = $4, email = $5
			WHERE id = (SELECT c.id FROM contacts c JOIN list_memberships m ON m.contact_id = c.id AND m.status = 'subscribed' AND m.list_id = $2
				WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.status = 'active' AND lower(split_part(replace(c.language, '_', '-'), '-', 1)) = $6
				ORDER BY (c.first_name = '') ASC, c.last_open_at DESC NULLS LAST, c.email LIMIT 1)
			AND NOT EXISTS (SELECT 1 FROM contacts WHERE workspace_id = $1 AND email = $5)`,
			l.ws.ID, digest, c.first, c.last, c.email, c.base); err != nil {
			return fmt.Errorf("name %s %s: %w", c.first, c.last, err)
		}
	}
	return nil
}

// passwordReset gives the transactional sample's password-reset its
// Spanish and German versions, translated by Sofia Rossi today at 12:30 and
// published with its live version: Spanish contacts get Spanish, German
// contacts German, everyone else English.
func (l *langSeed) passwordReset(ctx context.Context) error {
	s := l.s
	t, err := s.Q.LanguagesTemplateSource(ctx, store.LanguagesTemplateSourceParams{WorkspaceID: l.ws.ID, Key: "password-reset"})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	o := languages.Owner{Kind: languages.OwnerTransactional, ID: t.ID, WorkspaceID: l.ws.ID, Name: t.Key}
	src := languages.Source{Subject: t.Subject, Preheader: t.Preheader, MJML: t.Mjml, HTML: t.Html, Version: int(t.DraftRevision), By: t.SavedByName, At: t.DraftSavedAt}
	sofia := s.Users["sofia@acme.io"]
	at := s.langAt(0, 12, 30)
	for _, v := range []struct {
		lang  string
		pairs []string
	}{{"es", langResetES}, {"de", langResetDE}} {
		r := strings.NewReplacer(v.pairs...)
		by := sofia.ID
		if err := l.svc.ImportVersion(ctx, s.Q, languages.ImportedVersion{Owner: o, Source: src, SourceLanguage: "en", Language: v.lang, Translated: true,
			Subject: r.Replace(src.Subject), Preheader: r.Replace(src.Preheader), MJML: r.Replace(src.MJML), By: &by, At: at.Add(-5 * time.Minute), From: src.Version}); err != nil {
			return fmt.Errorf("password-reset in %s: %w", v.lang, err)
		}
		if err := readinessAudit(ctx, s, at.Add(-5*time.Minute), sofia, "transactional_template.language.mark_translated", "transactional_template", t.ID.String(), t.Key,
			map[string]any{"language": v.lang}, map[string]any{"permission": "transactional:edit"}, "198.51.100.23"); err != nil {
			return err
		}
	}
	if t.LiveVersion == nil {
		return nil
	}
	// Published with the live version: its sends pick them.
	return languages.Transactional{S: l.svc}.Publish(ctx, s.Q, &l.ws, t.ID, int(*t.LiveVersion))
}
