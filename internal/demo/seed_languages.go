package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedLanguages = seedLanguagesImpl }

// seedLanguagesImpl seeds the content languages of the design expansion
// (brief §4): English, the default and fallback; Spanish and German, added
// by Linh Nguyen on 6 Oct with their page copy Live since Sofia Rossi's work
// on 8 Oct; Vietnamese, added on 13 Oct, its page copy not set up. Contacts
// get their languages (the Weekly Digest's readers as Review & send counts
// them). Weekly Digest #142 goes out in English, Spanish and German, all up
// to date; Weekly Digest #143 has Spanish up to date and German outdated by
// two blocks. Vietnamese has no version in either.
func seedLanguagesImpl(ctx context.Context, s *Seeder) error {
	l := &langSeed{s: s, ws: s.WS(), svc: &languages.Service{Now: func() time.Time { return s.Now }}}
	for _, step := range []func(context.Context) error{l.workspace, l.contacts, l.sampleContacts, l.digest142, l.digest143, l.passwordReset, l.runs} {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

type langSeed struct {
	s   *Seeder
	ws  store.Workspace
	svc *languages.Service
}

// langAt is a time of the sample: today's relative to its 14:20 now, so it
// never lies ahead; earlier days' on the design day.
func (s *Seeder) langAt(daysAgo, hour, minute int) time.Time {
	if daysAgo == 0 {
		return s.Now.Add(-time.Duration(14*60+20-(hour*60+minute)) * time.Minute)
	}
	return s.designDay(daysAgo, hour, minute)
}

func (l *langSeed) workspace(ctx context.Context) error {
	s := l.s
	linh, sofia := s.Users["linh@acme.io"], s.Users["sofia@acme.io"]
	added, pages := s.langAt(9, 10, 5), s.langAt(7, 15, 40)
	rows := []struct {
		code   string
		def    bool
		by     *uuid.UUID
		at     time.Time
		page   string
		pageBy *uuid.UUID
		pageAt *time.Time
	}{
		{code: "en", def: true, at: s.Ago(400 * 24 * time.Hour), page: "live"},
		{code: "es", by: &linh.ID, at: added, page: "live", pageBy: &sofia.ID, pageAt: &pages},
		{code: "de", by: &linh.ID, at: added.Add(40 * time.Second), page: "live", pageBy: &sofia.ID, pageAt: &pages},
		{code: "vi", by: &linh.ID, at: s.langAt(2, 16, 45), page: "not_set_up"},
	}
	for _, r := range rows {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO workspace_languages (workspace_id, code, is_default, is_fallback, added_by, added_at, page_copy_status, page_copy_by, page_copy_at)
			VALUES ($1,$2,$3,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (workspace_id, code) DO UPDATE SET is_default = EXCLUDED.is_default, is_fallback = EXCLUDED.is_fallback, added_by = EXCLUDED.added_by,
				added_at = EXCLUDED.added_at, removed_at = NULL, removed_by = NULL, page_copy_status = EXCLUDED.page_copy_status, page_copy_by = EXCLUDED.page_copy_by,
				page_copy_at = EXCLUDED.page_copy_at`, l.ws.ID, r.code, r.def, r.by, r.at, r.page, r.pageBy, r.pageAt); err != nil {
			return err
		}
		if r.by != nil {
			if err := readinessAudit(ctx, s, r.at, linh, "language.add", "language", r.code, languages.NameOf(r.code), map[string]any{"code": r.code},
				map[string]any{"permission": "workspace:manage"}, "203.0.113.9"); err != nil {
				return err
			}
		}
	}
	return nil
}

// langShare is how many contacts get a language code.
type langShare struct {
	code string
	n    int
}

// The Weekly Digest's final recipients (Review & send: 23,269 English,
// 2,740 Spanish, 1,468 German, 506 Vietnamese, 3,613 with no language and
// 8 with languages Acme doesn't offer), then everyone else; English is the
// rest. In all: 35,500 English, 4,180 Spanish, 2,240 German, 773
// Vietnamese, 5,508 with no language and 12 others (SettingsLanguages).
var (
	langDigestShares = []langShare{{"es", 1320}, {"es-MX", 1000}, {"es-ES", 300}, {"es-AR", 120}, {"de", 1172}, {"de-AT", 200}, {"de-CH", 88}, {"de-LU", 8},
		{"vi", 506}, {"", 3613}, {"pt-BR", 5}, {"fr", 2}, {"it", 1}}
	langOtherShares = []langShare{{"es", 680}, {"es-MX", 500}, {"es-ES", 200}, {"es-AR", 60}, {"de", 616}, {"de-AT", 100}, {"de-CH", 52}, {"de-LU", 4},
		{"vi", 267}, {"", 1895}, {"pt-BR", 2}, {"fr", 1}, {"it", 1}}
)

func (l *langSeed) contacts(ctx context.Context) error {
	s := l.s
	digest, ok := s.Lists["Weekly Digest"]
	if !ok {
		return nil
	}
	var sunset json.RawMessage
	if err := s.Tx.QueryRow(ctx, `SELECT coalesce(sunset_policy, '{}'::jsonb) FROM workspaces WHERE id = $1`, l.ws.ID).Scan(&sunset); err != nil {
		return err
	}
	aud, _ := json.Marshal(map[string]any{"lists": []uuid.UUID{digest}})
	recipients, err := readiness.FinalRecipients(ctx, s.Q, l.ws.ID, aud, sunset, s.Now)
	if err != nil {
		return err
	}
	final := make(map[string]bool, len(recipients))
	for _, r := range recipients {
		final[strings.ToLower(r.Email)] = true
	}
	rows, err := s.Tx.Query(ctx, `SELECT id, lower(email) FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL ORDER BY id`, l.ws.ID)
	if err != nil {
		return err
	}
	var inDigest, others []uuid.UUID
	byEmail := map[string]uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			rows.Close()
			return err
		}
		byEmail[email] = id
		if final[email] {
			inDigest = append(inDigest, id)
		} else {
			others = append(others, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	codes := map[string][]uuid.UUID{}
	// The boards' German contacts keep German.
	pinned := map[uuid.UUID]bool{}
	for _, email := range []string{"hannah@bergconsulting.de", "lena.fischer@hotmail.com"} {
		if id, ok := byEmail[email]; ok {
			pinned[id] = true
			codes["de"] = append(codes["de"], id)
		}
	}
	// The newest contacts get the languages: the boards' named contacts,
	// seeded first, keep English.
	assign := func(all []uuid.UUID, shares []langShare) {
		ids := make([]uuid.UUID, len(all))
		for i, id := range all {
			ids[len(all)-1-i] = id
		}
		quota := map[string]int{}
		for _, sh := range shares {
			quota[sh.code] = sh.n
		}
		for _, id := range ids {
			if pinned[id] {
				quota["de"]-- // a pinned contact counts against German
			}
		}
		i := 0
		for _, sh := range shares {
			for n := 0; n < quota[sh.code] && i < len(ids); i++ {
				if pinned[ids[i]] {
					continue
				}
				codes[sh.code] = append(codes[sh.code], ids[i])
				n++
			}
		}
		for ; i < len(ids); i++ {
			if !pinned[ids[i]] {
				codes["en"] = append(codes["en"], ids[i])
			}
		}
	}
	assign(inDigest, langDigestShares)
	assign(others, langOtherShares)
	for code, ids := range codes {
		if _, err := s.Tx.Exec(ctx, `UPDATE contacts SET language = $2 WHERE id = ANY($1)`, ids, code); err != nil {
			return err
		}
	}
	// Rafael Souza (LanguageStates): a Weekly Digest reader with pt-BR, set
	// by the API on 12 Oct.
	for _, id := range codes["pt-BR"] {
		if !slicesContains(inDigest, id) {
			continue
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE contacts SET first_name = 'Rafael', last_name = 'Souza', email = 'rafael.souza@example.com.br', updated_at = $2
			WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM contacts WHERE workspace_id = $3 AND email = 'rafael.souza@example.com.br')`,
			id, s.langAt(3, 11, 2), l.ws.ID); err != nil {
			return err
		}
		break
	}
	return nil
}

func slicesContains(ids []uuid.UUID, id uuid.UUID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// Translations of the sample emails, longest first, so a phrase is
// translated before the words it contains.
var (
	langDigest142ES = []string{
		"Three launches, one deprecation, and the roadmap for Q4", "Tres lanzamientos, una retirada y la hoja de ruta para el cuarto trimestre",
		"Feature flags are GA, audit log exports, and what we're retiring in November.", "Los feature flags ya están disponibles, las exportaciones del registro de auditoría y lo que retiramos en noviembre.",
		`Hi {{ first_name | default: "there" }}, here's what shipped this week at Acme, what we're retiring in November, and where the next quarter is headed.`,
		`Hola {{ first_name | default: "a todos" }}, esto es lo que lanzamos esta semana en Acme, lo que retiramos en noviembre y hacia dónde va el próximo trimestre.`,
		"What shipped this week, and what we're retiring", "Lo que lanzamos esta semana y lo que retiramos",
		"Target rollouts by plan, region or percentage, with an audit trail for every change.", "Despliega por plan, región o porcentaje, con un registro de cada cambio.",
		"Stream every workspace event to S3 or your SIEM, with 13 months of history.", "Envía cada evento del espacio de trabajo a S3 o a tu SIEM, con 13 meses de historial.",
		"If you still call api.acme.io/v1, the ", "Si todavía usas api.acme.io/v1, la ",
		"migration guide</a> covers every endpoint. Most teams finish in an afternoon.", "guía de migración</a> cubre cada endpoint. La mayoría de los equipos termina en una tarde.",
		"Retiring the v1 API on 30 November", "La API v1 se retira el 30 de noviembre",
		"Feature flags are generally available", "Los feature flags ya están disponibles",
		"Audit log export settings", "Ajustes de exportación del registro de auditoría",
		"Feature flags dashboard", "Panel de feature flags",
		"Q4 roadmap illustration", "Ilustración de la hoja de ruta del cuarto trimestre",
		"Audit log exports", "Exportaciones del registro de auditoría",
		"Email preferences", "Preferencias de correo",
		"View in browser", "Ver en el navegador",
		"See the changelog", "Ver las novedades",
		"Read the docs", "Lee la documentación",
		">Unsubscribe<", ">Darse de baja<",
		">Changelog<", ">Novedades<",
		">Docs<", ">Documentación<",
	}
	langDigest142DE = []string{
		"Three launches, one deprecation, and the roadmap for Q4", "Drei Launches, eine Abkündigung und die Roadmap für Q4",
		"Feature flags are GA, audit log exports, and what we're retiring in November.", "Feature-Flags sind allgemein verfügbar, Audit-Log-Exporte und was wir im November einstellen.",
		`Hi {{ first_name | default: "there" }}, here's what shipped this week at Acme, what we're retiring in November, and where the next quarter is headed.`,
		`Hallo {{ first_name | default: "zusammen" }}, das hat Acme diese Woche veröffentlicht, das stellen wir im November ein, und dahin geht das nächste Quartal.`,
		"What shipped this week, and what we're retiring", "Was diese Woche erschienen ist und was wir einstellen",
		"Target rollouts by plan, region or percentage, with an audit trail for every change.", "Rollouts nach Tarif, Region oder Prozentsatz, mit einem Audit-Trail für jede Änderung.",
		"Stream every workspace event to S3 or your SIEM, with 13 months of history.", "Jedes Workspace-Ereignis nach S3 oder in Ihr SIEM streamen, mit 13 Monaten Verlauf.",
		"If you still call api.acme.io/v1, the ", "Wenn Sie noch api.acme.io/v1 aufrufen: Die ",
		"migration guide</a> covers every endpoint. Most teams finish in an afternoon.", "Migrationsanleitung</a> deckt jeden Endpunkt ab. Die meisten Teams sind an einem Nachmittag fertig.",
		"Retiring the v1 API on 30 November", "Die v1-API wird am 30. November abgeschaltet",
		"Feature flags are generally available", "Feature-Flags sind allgemein verfügbar",
		"Audit log export settings", "Einstellungen für Audit-Log-Exporte",
		"Feature flags dashboard", "Feature-Flag-Dashboard",
		"Q4 roadmap illustration", "Illustration der Roadmap für Q4",
		"Audit log exports", "Audit-Log-Exporte",
		"Email preferences", "E-Mail-Einstellungen",
		"View in browser", "Im Browser ansehen",
		"See the changelog", "Zum Changelog",
		"Read the docs", "Zur Dokumentation",
		">Unsubscribe<", ">Abmelden<",
		">Docs<", ">Doku<",
	}
	langDigest143ES = []string{
		"Scheduled rollouts, faster exports, and 6 weeks left on the v1 API", "Despliegues programados, exportaciones más rápidas y 6 semanas para la API v1",
		"Plan a flag change for 2 a.m. and sleep through it.", "Programa un cambio de flag para las 2 a. m. y sigue durmiendo.",
		`Hi {{ first_name | default: "there" }}, you can now schedule flag changes ahead of time, in any timezone.`,
		`Hola {{ first_name | default: "a todos" }}, ahora puedes programar cambios de flags con antelación, en cualquier zona horaria.`,
		"Click here</a> to see how it works.", "Haz clic aquí</a> para ver cómo funciona.",
		"Pick the minute a flag changes. Every scheduled change shows in the audit log.", "Elige el minuto en que cambia un flag. Cada cambio programado aparece en el registro de auditoría.",
		"Exports of up to a million rows now finish in under a minute.", "Las exportaciones de hasta un millón de filas terminan ahora en menos de un minuto.",
		"The v1 API stops answering on 30 November. Migration guide:", "La API v1 deja de responder el 30 de noviembre. Guía de migración:",
		"The schedule picker in the flag editor", "El selector de horario en el editor de flags",
		"Scheduled rollouts are here", "Ya están aquí los despliegues programados",
		"6 weeks left on the v1 API", "Quedan 6 semanas para la API v1",
		"The CSV export dialog", "El diálogo de exportación CSV",
		"Faster CSV exports", "Exportaciones CSV más rápidas",
		"Scheduled rollouts", "Despliegues programados",
		"Plan your migration", "Planifica tu migración",
		"Acme on LinkedIn", "Acme en LinkedIn",
		"Acme on GitHub", "Acme en GitHub",
		"Acme on X", "Acme en X",
		"Email preferences", "Preferencias de correo",
		"View in browser", "Ver en el navegador",
		"See the changelog", "Ver las novedades",
		"Read the docs", "Lee la documentación",
		">Unsubscribe<", ">Darse de baja<",
		">Changelog<", ">Novedades<",
		">Docs<", ">Documentación<",
	}
	// German was translated from v6, before the intro gained "in any
	// timezone" and the CSV card its new sentence.
	langDigest143DE = []string{
		"Scheduled rollouts, faster exports, and 6 weeks left on the v1 API", "Geplante Rollouts, schnellere Exporte und noch 6 Wochen für die v1-API",
		"Plan a flag change for 2 a.m. and sleep through it.", "Planen Sie eine Flag-Änderung für 2 Uhr nachts und schlafen Sie weiter.",
		`Hi {{ first_name | default: "there" }}, you can now schedule flag changes ahead of time.`,
		`Hallo {{ first_name | default: "zusammen" }}, Sie können Flag-Änderungen jetzt im Voraus planen.`,
		"Click here</a> to see how it works.", "Hier klicken</a>, um zu sehen, wie es funktioniert.",
		"Pick the minute a flag changes. Every scheduled change shows in the audit log.", "Wählen Sie die Minute, in der ein Flag umschaltet. Jede geplante Änderung steht im Audit-Log.",
		"Exports now run up to 3× faster.", "Exporte laufen jetzt bis zu 3× schneller.",
		"The v1 API stops answering on 30 November. Migration guide:", "Die v1-API antwortet ab dem 30. November nicht mehr. Migrationsanleitung:",
		"The schedule picker in the flag editor", "Die Zeitplanauswahl im Flag-Editor",
		"Scheduled rollouts are here", "Geplante Rollouts sind da",
		"6 weeks left on the v1 API", "Noch 6 Wochen für die v1-API",
		"The CSV export dialog", "Der Dialog für CSV-Exporte",
		"Faster CSV exports", "Schnellere CSV-Exporte",
		"Scheduled rollouts", "Geplante Rollouts",
		"Plan your migration", "Migration planen",
		"Acme on LinkedIn", "Acme auf LinkedIn",
		"Acme on GitHub", "Acme auf GitHub",
		"Acme on X", "Acme auf X",
		"Email preferences", "E-Mail-Einstellungen",
		"View in browser", "Im Browser ansehen",
		"See the changelog", "Zum Changelog",
		"Read the docs", "Zur Dokumentation",
		">Unsubscribe<", ">Abmelden<",
		">Docs<", ">Doku<",
	}
)

// langSource is a campaign's source as the languages domain reads it.
func (l *langSeed) langSource(ctx context.Context, id uuid.UUID) (languages.Owner, languages.Source, json.RawMessage, error) {
	var src languages.Source
	var name string
	var ab json.RawMessage
	var version int32
	var by *uuid.UUID
	if err := l.s.Tx.QueryRow(ctx, `SELECT name, subject, preheader, mjml, html, version, updated_by, coalesce(autosaved_at, updated_at), ab_test FROM campaigns WHERE id = $1`, id).
		Scan(&name, &src.Subject, &src.Preheader, &src.MJML, &src.HTML, &version, &by, &src.At, &ab); err != nil {
		return languages.Owner{}, src, nil, err
	}
	src.Version = int(version)
	for _, u := range l.s.Users {
		if by != nil && u.ID == *by {
			src.By = u.Name
		}
	}
	return languages.Owner{Kind: languages.OwnerCampaign, ID: id, WorkspaceID: l.ws.ID, Name: name}, src, ab, nil
}

// translateAB gives the A/B test's subjects in a language.
func translateAB(ab json.RawMessage, r *strings.Replacer) json.RawMessage {
	if len(ab) == 0 || string(ab) == "null" {
		return nil
	}
	return json.RawMessage(r.Replace(string(ab)))
}

func (l *langSeed) digest142(ctx context.Context) error {
	s := l.s
	id, ok := s.Campaigns["Weekly Digest #142"]
	if !ok {
		return nil
	}
	o, src, ab, err := l.langSource(ctx, id)
	if err != nil {
		return err
	}
	daniel, sofia := s.Users["daniel@acme.io"], s.Users["sofia@acme.io"]
	for _, v := range []struct {
		lang  string
		pairs []string
		by    store.User
		at    time.Time
	}{
		{"es", langDigest142ES, daniel, s.langAt(0, 10, 40)},
		{"de", langDigest142DE, sofia, s.langAt(0, 10, 55)},
	} {
		r := strings.NewReplacer(v.pairs...)
		by := v.by.ID
		if err := l.svc.ImportVersion(ctx, s.Q, languages.ImportedVersion{Owner: o, Source: src, SourceLanguage: "en", Language: v.lang, Translated: true,
			Subject: r.Replace(src.Subject), Preheader: r.Replace(src.Preheader), MJML: r.Replace(src.MJML), AB: translateAB(ab, r), By: &by, At: v.at, From: src.Version}); err != nil {
			return fmt.Errorf("weekly digest #142 in %s: %w", v.lang, err)
		}
		if err := readinessAudit(ctx, s, v.at, v.by, "campaign.language.mark_translated", "campaign", id.String(), "Weekly Digest #142",
			map[string]any{"language": v.lang}, map[string]any{"permission": "campaigns:edit"}, "198.51.100.23"); err != nil {
			return err
		}
	}
	return nil
}

// digest143 adds Spanish (up to date, Daniel Park today at 11:20) and
// German (translated by Sofia Rossi from v6 yesterday at 16:20, outdated by
// the intro and the "Faster CSV exports" card) to the accessibility
// sample's Weekly Digest #143, when it is there.
func (l *langSeed) digest143(ctx context.Context) error {
	s := l.s
	id, ok := s.Campaigns["Weekly Digest #143"]
	if !ok {
		return nil
	}
	o, src, ab, err := l.langSource(ctx, id)
	if err != nil {
		return err
	}
	// The source German was translated from, and who changed it since.
	v6 := struct{ mjml, subject, preheader string }{src.MJML, src.Subject, src.Preheader}
	if err := s.Tx.QueryRow(ctx, `SELECT mjml, subject, preheader FROM campaign_versions WHERE campaign_id = $1 AND version = 6`, id).
		Scan(&v6.mjml, &v6.subject, &v6.preheader); err != nil {
		v6.mjml, v6.subject, v6.preheader = src.MJML, src.Subject, src.Preheader
	}
	changes, err := l.changesSince(ctx, id, 6, src.MJML)
	if err != nil {
		return err
	}
	daniel, sofia := s.Users["daniel@acme.io"], s.Users["sofia@acme.io"]
	danielID, sofiaID := daniel.ID, sofia.ID
	esAt, deAt := s.langAt(0, 11, 20), s.langAt(1, 16, 20)
	es := strings.NewReplacer(langDigest143ES...)
	if err := l.svc.ImportVersion(ctx, s.Q, languages.ImportedVersion{Owner: o, Source: src, SourceLanguage: "en", Language: "es", Translated: true,
		Subject: es.Replace(src.Subject), Preheader: es.Replace(src.Preheader), MJML: es.Replace(src.MJML), AB: translateAB(ab, es), By: &danielID, At: esAt,
		From: src.Version, Changes: changes}); err != nil {
		return fmt.Errorf("weekly digest #143 in Spanish: %w", err)
	}
	de := strings.NewReplacer(langDigest143DE...)
	if err := l.svc.ImportVersion(ctx, s.Q, languages.ImportedVersion{Owner: o, Source: src, SourceLanguage: "en", Language: "de", Translated: true,
		Subject: de.Replace(v6.subject), Preheader: de.Replace(v6.preheader), MJML: de.Replace(v6.mjml), AB: translateAB(ab, de), By: &sofiaID, At: deAt,
		From: 6, FromMJML: v6.mjml, FromSubject: v6.subject, FromPreheader: v6.preheader}); err != nil {
		return fmt.Errorf("weekly digest #143 in German: %w", err)
	}
	for _, a := range []struct {
		at     time.Time
		u      store.User
		action string
		lang   string
	}{
		{deAt.Add(-50 * time.Minute), sofia, "campaign.language.add", "de"}, {deAt, sofia, "campaign.language.mark_translated", "de"},
		{esAt.Add(-35 * time.Minute), daniel, "campaign.language.add", "es"}, {esAt, daniel, "campaign.language.mark_translated", "es"},
	} {
		if err := readinessAudit(ctx, s, a.at, a.u, a.action, "campaign", id.String(), "Weekly Digest #143", map[string]any{"language": a.lang},
			map[string]any{"permission": "campaigns:edit"}, "198.51.100.23"); err != nil {
			return err
		}
	}
	return nil
}

// changesSince says, for each block of the source now, the latest version
// after from that changed it, and who saved that version.
func (l *langSeed) changesSince(ctx context.Context, id uuid.UUID, from int, now string) (map[int]languages.BlockChange, error) {
	rows, err := l.s.Tx.Query(ctx, `SELECT v.version, v.mjml, v.saved_at, coalesce(u.name, '') FROM campaign_versions v LEFT JOIN users u ON u.id = v.saved_by
		WHERE v.campaign_id = $1 AND v.version >= $2 ORDER BY v.version`, id, from)
	if err != nil {
		return nil, err
	}
	type ver struct {
		n      int
		src    string
		at     time.Time
		by     string
		blocks []languages.Block
	}
	var vs []ver
	for rows.Next() {
		var v ver
		if err := rows.Scan(&v.n, &v.src, &v.at, &v.by); err != nil {
			rows.Close()
			return nil, err
		}
		v.blocks = languages.Blocks(v.src, "")
		vs = append(vs, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := map[int]languages.BlockChange{}
	if len(vs) < 2 {
		return out, nil
	}
	keys := func(bs []languages.Block) []languages.AlignKey {
		ks := make([]languages.AlignKey, len(bs))
		for i, b := range bs {
			ks[i] = languages.AlignKey{Raw: b.Raw, Hash: b.Hash}
		}
		return ks
	}
	current := languages.Blocks(now, "")
	// pos[i] is block i of the source now in version k, walking back from
	// the latest version: the first change met is the latest.
	last := len(vs) - 1
	pos := languages.Align(keys(vs[last].blocks), keys(current))
	for k := last; k >= 1; k-- {
		back := languages.Align(keys(vs[k-1].blocks), keys(vs[k].blocks))
		for i := range current {
			j := pos[i]
			if j < 0 {
				continue
			}
			bj := back[j]
			if _, seen := out[i]; !seen && (bj < 0 || vs[k-1].blocks[bj].Hash != vs[k].blocks[j].Hash) {
				out[i] = languages.BlockChange{Version: vs[k].n, By: vs[k].by, At: vs[k].at}
			}
			pos[i] = bj
		}
	}
	return out, nil
}

// runs: the sample runs and the approval of #142 and #143 checked the
// emails before their versions existed; they now cover the versions, so
// they aren't stale.
func (l *langSeed) runs(ctx context.Context) error {
	s := l.s
	for _, name := range []string{"Weekly Digest #142", "Weekly Digest #143"} {
		id, ok := s.Campaigns[name]
		if !ok {
			continue
		}
		digest := l.svc.CampaignDigest(ctx, s.Q, l.ws.ID, id)
		if digest == "" {
			continue
		}
		rows, err := s.Tx.Query(ctx, `SELECT id, fingerprint FROM readiness_runs WHERE campaign_id = $1`, id)
		if err != nil {
			return err
		}
		fps := map[uuid.UUID]string{}
		for rows.Next() {
			var rid uuid.UUID
			var fp string
			if err := rows.Scan(&rid, &fp); err != nil {
				rows.Close()
				return err
			}
			fps[rid] = fp
		}
		rows.Close()
		for rid, fp := range fps {
			if _, err := s.Tx.Exec(ctx, `UPDATE readiness_runs SET fingerprint = $2 WHERE id = $1`, rid, readiness.WithLanguages(fp, digest)); err != nil {
				return err
			}
		}
		snaps, err := l.svc.CampaignSnapshot(ctx, s.Q, id)
		if err != nil {
			return err
		}
		b, _ := json.Marshal(snaps)
		if _, err := s.Tx.Exec(ctx, `UPDATE approval_requests SET snapshot = jsonb_set(snapshot, '{languages}', $2::jsonb) WHERE campaign_id = $1`, id, b); err != nil {
			return err
		}
	}
	return nil
}

// langReadinessFacts gives the sample runs of #142 and #143 their Languages
// group and their versions' content checks (SendReadinessAccessibility).
func (r *readinessSeed) langReadinessFacts(f readiness.Facts) readiness.Facts {
	outdated := false
	switch f.Campaign.Name {
	case "Weekly Digest #142":
	case "Weekly Digest #143":
		outdated = true
	default:
		return f
	}
	de := languages.ReadinessLanguage{Language: "de", Name: "German", Status: languages.StatusUpToDate, Offered: true, Readers: 1468, Gets: "de", Blocks: 9, Translated: 9,
		TranslatedFrom: f.Campaign.Version, PageCopy: "live"}
	if outdated {
		de.Status, de.Attention, de.TranslatedFrom = languages.StatusOutdated, 2, 6
		de.Changed = []string{"Intro", "Two cards"}
	}
	f.Languages = languages.ReadinessFacts{Applies: true, Source: "en", SourceVersion: f.Campaign.Version, Fallback: "en", OthersGet: "en", Recipients: f.Audience.Final,
		NoLanguage: 3613, NotOffered: 8, NotOfferedCodes: []string{"pt-BR", "fr", "it"},
		Languages: []languages.ReadinessLanguage{
			{Language: "en", Name: "English", Status: languages.StatusSource, Offered: true, Readers: 23269, Gets: "en", PageCopy: "live"},
			{Language: "es", Name: "Spanish", Status: languages.StatusUpToDate, Offered: true, Readers: 2740, Gets: "es", Blocks: 9, Translated: 9,
				TranslatedFrom: f.Campaign.Version, PageCopy: "live"},
			de,
			{Language: "vi", Name: "Vietnamese", Status: languages.StatusMissing, Offered: true, Readers: 506, Gets: "en", PageCopy: "not_set_up"},
		},
		Versions: []languages.ReadinessVersion{{Language: "es"}, {Language: "de"}}}
	for _, v := range []struct {
		lang string
		kb   int
	}{{"es", 2}, {"de", 5}} {
		rep := f.Content.Report
		rep.SizeBytes += v.kb * 1024
		f.Content.Versions = append(f.Content.Versions, readiness.VersionReport{Language: v.lang, Report: rep})
	}
	return f
}
