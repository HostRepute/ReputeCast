package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The page and form copy of the sample (brief §4, "Page and form copy"):
// Sofia Rossi wrote the Spanish and German pages, labels and the
// Newsletter signup form's copy on 8 Oct, so both are Live; Vietnamese has
// nothing of Acme's own yet (its labels ship with ReputeCast), so it's Not
// set up and Vietnamese visitors see English.

// pageCopyTexts are the pages' own text in Spanish and German, by what each
// string is in English.
//
//nolint:misspell // Spanish and German, not misspelt English
var pageCopyTexts = map[string]map[string]string{
	"es": {
		"subscribe.heading":                "El boletín de Acme",
		"subscribe.description":            "Un correo a la semana con lo que lanzamos, lo que viene y algún análisis a fondo. Lo leen 48.000 personas que construyen con Acme.",
		"subscribe.footnote":               "Primero te enviaremos un enlace de confirmación. Nunca enviamos spam. Puedes cambiar de lista o pausar los correos desde cualquier mensaje.",
		"subscribe.list.Weekly Digest":     "Cada martes por la mañana. {subscribers} suscriptores.",
		"subscribe.list.Product Updates":   "Solo cuando lanzamos algo, unas dos veces al mes.",
		"confirm.list.Weekly Digest":       "Los martes a las 9:00, en tu hora.",
		"confirm.list.Product Updates":     "Solo cuando lanzamos algo, unas dos veces al mes.",
		"confirm.list.Beta Program":        "Invitaciones de acceso anticipado y peticiones de opinión. Pocas veces.",
		"confirm.offer.Product Updates":    "novedades de lanzamientos",
		"confirm.offer.Beta Program":       "invitaciones de acceso anticipado",
		"preferences.list.Weekly Digest":   "Los martes. Lo que lanzamos y lo que viene.",
		"preferences.list.Product Updates": "Solo cuando lanzamos algo, unas dos veces al mes.",
		"preferences.list.Beta Program":    "Invitaciones de acceso anticipado y peticiones de opinión. Pocas veces.",
		"unsubscribe.fewerTitle":           "¿Demasiados correos? Recibe menos.",
		"unsubscribe.fewerText":            "Cambia a un resumen mensual con lo mejor y nada más. La mayoría de quienes lo eligen siguen suscritos.",
		"unsubscribe.fewerButton":          "Enviarme solo lo mejor del mes",
		"list.Weekly Digest.description":   "Cada martes: novedades del producto, guías y el registro de cambios.",
		"list.Product Updates.description": "Lanzamientos y retiradas, unas dos veces al mes.",
		"list.Beta Program.description":    "Acceso anticipado a funciones antes de su lanzamiento.",
		"privacyUrl":                       "https://acme.io/es/privacidad",
	},
	"de": {
		"subscribe.heading":                "Der Acme-Newsletter",
		"subscribe.description":            "Eine E-Mail pro Woche: was erschienen ist, was als Nächstes kommt, und ab und zu ein tiefer Einblick. 48.000 Menschen, die mit Acme entwickeln, lesen mit.",
		"subscribe.footnote":               "Zuerst schicken wir dir einen Bestätigungslink. Niemals Spam. Listen ändern oder E-Mails pausieren kannst du in jeder Nachricht.",
		"subscribe.list.Weekly Digest":     "Jeden Dienstagmorgen. {subscribers} Abonnenten.",
		"subscribe.list.Product Updates":   "Nur wenn etwas erscheint, etwa zweimal im Monat.",
		"confirm.list.Weekly Digest":       "Dienstags um 9:00 Uhr deiner Zeit.",
		"confirm.list.Product Updates":     "Nur wenn etwas erscheint, etwa zweimal im Monat.",
		"confirm.list.Beta Program":        "Einladungen zum frühen Zugang und Bitten um Feedback. Selten.",
		"confirm.offer.Product Updates":    "Neuigkeiten zu Releases",
		"confirm.offer.Beta Program":       "Einladungen zum frühen Zugang",
		"preferences.list.Weekly Digest":   "Dienstags. Was erschienen ist und was kommt.",
		"preferences.list.Product Updates": "Nur wenn etwas erscheint, etwa zweimal im Monat.",
		"preferences.list.Beta Program":    "Einladungen zum frühen Zugang und Bitten um Feedback. Selten.",
		"unsubscribe.fewerTitle":           "Zu viele E-Mails? Bekomm einfach weniger.",
		"unsubscribe.fewerText":            "Wechsle zu einem monatlichen Best-of und sonst nichts. Die meisten, die das wählen, bleiben dabei.",
		"unsubscribe.fewerButton":          "Nur das Beste des Monats schicken",
		"list.Weekly Digest.description":   "Jeden Dienstag: Produktneuigkeiten, Anleitungen und das Changelog.",
		"list.Product Updates.description": "Neue Funktionen und Abkündigungen, etwa zweimal im Monat.",
		"list.Beta Program.description":    "Früher Zugang zu Funktionen vor dem Release.",
		"privacyUrl":                       "https://acme.io/de/datenschutz",
	},
}

// newsletterSignupCopy is the Newsletter signup form in Spanish and German
// (FormBuilderLanguages): consent text v1 of each, written for English v3.
var newsletterSignupCopy = map[string]struct{ heading, description, button, consent, after string }{
	"es": {"El boletín de Acme", "Un correo a la semana con lo que lanzamos, lo que viene y algún análisis a fondo. Lo leen 48.000 personas que construyen con Acme.",
		"Suscribirme", "Acepto recibir correos de Acme. Puedo darme de baja cuando quiera con un clic. Consulta la política de privacidad.", "Revisa tu correo para confirmar."},
	"de": {"Der Acme-Newsletter", "Eine E-Mail pro Woche: was erschienen ist, was als Nächstes kommt, und ab und zu ein tiefer Einblick. 48.000 Leserinnen und Leser.",
		"Abonnieren", "Ich möchte E-Mails von Acme erhalten. Abmelden geht jederzeit mit einem Klick. Siehe Datenschutzerklärung.", "Sieh in dein Postfach, um zu bestätigen."},
}

// pageCopy writes the Spanish and German pages and labels.
func (p *pageLangSeed) pageCopy(ctx context.Context) error {
	s := p.s
	sofia := s.Users["sofia@acme.io"]
	// The time X05's sample gives Live page copy: 8 Oct, 15:40.
	labelsAt, pagesAt := s.langAt(7, 11, 5), s.langAt(7, 15, 40)
	lists := map[string]uuid.UUID{}
	rows, err := s.Tx.Query(ctx, `SELECT id, name FROM lists WHERE workspace_id = $1`, p.ws)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return err
		}
		lists[name] = id
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for code, texts := range pageCopyTexts {
		pages := map[string]string{}
		for key, text := range texts {
			k, err := pageCopyKey(key, lists)
			if err != nil {
				return err
			}
			pages[k] = text
		}
		labels := spanishLabels
		if code == "de" {
			labels = germanLabels
		}
		pj, _ := json.Marshal(pages)
		lj, _ := json.Marshal(labels)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO page_copy (workspace_id, language, pages, pages_by, pages_at, labels, labels_by, labels_at)
			VALUES ($1, $2, $3, $4, $5, $6, $4, $7)
			ON CONFLICT (workspace_id, language) DO UPDATE SET pages = EXCLUDED.pages, pages_by = EXCLUDED.pages_by, pages_at = EXCLUDED.pages_at,
				labels = EXCLUDED.labels, labels_by = EXCLUDED.labels_by, labels_at = EXCLUDED.labels_at`,
			p.ws, code, pj, sofia.ID, pagesAt, lj, labelsAt); err != nil {
			return fmt.Errorf("page copy %s: %w", code, err)
		}
	}
	return nil
}

// pageCopyKey turns "confirm.list.Weekly Digest" into the stored key,
// "confirm.list.<the list's id>".
func pageCopyKey(key string, lists map[string]uuid.UUID) (string, error) {
	for _, prefix := range []string{"subscribe.list.", "confirm.list.", "confirm.offer.", "preferences.list."} {
		if name, ok := cutPrefix(key, prefix); ok {
			id, found := lists[name]
			if !found {
				return "", fmt.Errorf("page copy: no list %q", name)
			}
			return prefix + id.String(), nil
		}
	}
	if rest, ok := cutPrefix(key, "list."); ok {
		for name, id := range lists {
			if rest == name+".description" || rest == name+".name" {
				return "list." + id.String() + rest[len(name):], nil
			}
		}
		return "", fmt.Errorf("page copy: no list in %q", key)
	}
	return key, nil
}

func cutPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return "", false
}

// formCopy writes the Newsletter signup form's Spanish and German copy.
func (p *pageLangSeed) formCopy(ctx context.Context) error {
	s := p.s
	sofia := s.Users["sofia@acme.io"]
	at := s.langAt(7, 15, 32)
	var form uuid.UUID
	var version int
	if err := s.Tx.QueryRow(ctx, `SELECT id, consent_version FROM forms WHERE workspace_id = $1 AND name = 'Newsletter signup'`, p.ws).Scan(&form, &version); err != nil {
		return fmt.Errorf("newsletter signup form: %w", err)
	}
	for code, c := range newsletterSignupCopy {
		content, _ := json.Marshal(map[string]any{"heading": c.heading, "description": c.description, "button": c.button, "afterSubmit": c.after})
		if _, err := s.Tx.Exec(ctx, `INSERT INTO form_copy (form_id, language, content, consent_text, consent_version, consent_for, updated_by, updated_at)
			VALUES ($1, $2, $3, $4, 1, $5, $6, $7)
			ON CONFLICT (form_id, language) DO UPDATE SET content = EXCLUDED.content, consent_text = EXCLUDED.consent_text, consent_version = 1,
				consent_for = EXCLUDED.consent_for, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at`,
			form, code, content, c.consent, version, sofia.ID, at); err != nil {
			return fmt.Errorf("form copy %s: %w", code, err)
		}
	}
	return p.luciaConsent(ctx, form)
}

// luciaConsent is Lucía Fernández's signup through the Spanish Newsletter
// signup form on 3 Oct: her consent evidence keeps the Spanish consent text,
// its version (v1) and the language it was shown in.
func (p *pageLangSeed) luciaConsent(ctx context.Context, form uuid.UUID) error {
	s := p.s
	var contact, list uuid.UUID
	err := s.Tx.QueryRow(ctx, `SELECT c.id, l.id FROM contacts c JOIN lists l ON l.workspace_id = c.workspace_id AND l.name = 'Weekly Digest'
		WHERE c.workspace_id = $1 AND c.email = 'lucia.fernandez@example.com'`, p.ws).Scan(&contact, &list)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // another seeder didn't create her
	}
	if err != nil {
		return err
	}
	signed := s.langAt(12, 10, 14)
	confirmed := signed.Add(2*time.Minute + 11*time.Second)
	consent := newsletterSignupCopy["es"].consent
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.6 Mobile/15E148 Safari/604.1"
	if _, err := s.Tx.Exec(ctx, `INSERT INTO consent_events (id, contact_id, list_id, action, source, ip, user_agent, detail, note, consent_version, consent_text, language, occurred_at)
		VALUES ($1, $3, $4, 'subscribed', 'form:newsletter-signup', '189.203.44.17', $5, '', 'waiting for the confirmation click', 'v1', $6, 'es', $7),
		       ($2, $3, $4, 'confirmed', 'confirm_link', '189.203.44.17', $5, '', '', 'v1', $6, 'es', $8)`,
		ID(), ID(), contact, list, ua, consent, signed, confirmed); err != nil {
		return fmt.Errorf("lucía's consent: %w", err)
	}
	_, err = s.Tx.Exec(ctx, `INSERT INTO form_submissions (id, workspace_id, form_id, contact_id, list_ids, double_opt_in, language, created_at, confirm_sent_at, send_attempts, confirmed_at)
		VALUES ($1, $2, $3, $4, ARRAY[$5::uuid], true, 'es', $6, $6, 1, $7)`, ID(), p.ws, form, contact, list, signed, confirmed)
	return err
}
