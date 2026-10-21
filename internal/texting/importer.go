package texting

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
)

// Numbers and SMS consent from imports (brief §3.3): SMS consent comes in
// only with its evidence, its date, source and text, recorded as
// "imported"; without them the number comes in and SMS consent stays Not
// subscribed. Imports never text anyone.

// ImportTexts is texting's side of imports (imports.Texts).
type ImportTexts struct{ S *Service }

var _ imports.Texts = ImportTexts{}

// parseNumber reads a number as typed, in a country when it has no code.
func parseNumber(raw, country string) (phone.Number, bool) {
	n, err := phone.Parse(raw, country)
	return n, err == nil
}

// DefaultCountry implements imports.Texts: the first country the
// workspace routes texts to.
func (t ImportTexts) DefaultCountry(ctx context.Context, ws *store.Workspace) string {
	g, err := t.S.graph(ctx, t.S.Q, ws)
	if err != nil {
		return ""
	}
	for _, r := range g.routes {
		if len(r.Countries) > 0 {
			return r.Countries[0]
		}
	}
	return ""
}

// Import implements imports.Texts. A contact's number is kept unless the
// conflict rule overwrites it; SMS consent goes with the contact's number
// only.
func (t ImportTexts) Import(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID, listID uuid.UUID, row imports.TextRow) (imports.TextOutcome, error) {
	// A value that isn't a phone number is counted, not an error.
	n, ok := parseNumber(row.Phone, row.Country)
	if !ok {
		return imports.TextInvalid, nil
	}
	q := store.New(tx)
	var cur string
	if err := tx.QueryRow(ctx, `SELECT phone FROM contacts WHERE id = $1 AND workspace_id = $2`, contactID, ws.ID).Scan(&cur); err != nil {
		return "", err
	}
	if cur == "" || (row.Overwrite && cur != n.E164) {
		at := t.S.now()
		if err := q.ChanSetPhone(ctx, store.ChanSetPhoneParams{Phone: n.E164, PhoneCountry: n.Country, PhoneLineType: string(n.Type), PhoneLineSource: "number_plan",
			At: &at, ID: contactID, WorkspaceID: ws.ID}); err != nil {
			return "", err
		}
		cur = n.E164
	}
	if !row.Consent || cur != n.E164 {
		return imports.TextNumber, nil
	}
	if row.At == nil || strings.TrimSpace(row.Source) == "" || strings.TrimSpace(row.Text) == "" {
		return imports.TextNoEvidence, nil
	}
	importID := row.ImportID
	if _, err := t.S.SetConsent(ctx, q, ws, contactID, n.E164, listID, "subscribed", Evidence{Source: "import:" + row.File, Page: strings.TrimSpace(row.Source),
		ConsentText: strings.TrimSpace(row.Text), ImportID: &importID, At: *row.At}); err != nil {
		return "", err
	}
	return imports.TextConsent, nil
}
