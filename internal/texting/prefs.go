package texting

import (
	"context"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

// The preference center's texts (brief §3.3, §6): a Texts switch per list
// and the number texts go to. Turning one on records SMS consent with the
// same evidence as a form (the exact text, the page, the IP address and
// the time) and texts the opt-in confirmation; turning one off withdraws
// it. A number that replied STOP stays off: only START brings it back.

// PrefTexts is the preference center's texts for a contact, in a language.
type PrefTexts struct {
	// Phone is the number, masked: +1 312 ••• 0123.
	Phone string
	Label string
	Note  string
	// Save is the account page's button.
	Save string
	// On is SMS consent per list (subscribed or waiting for YES).
	On map[uuid.UUID]bool
	// Locked is true when the number replied STOP: the switches are off
	// and stay off here.
	Locked bool
}

// PrefEvidence is the request a switch was changed in, and where:
// preference_center (default) or account_page.
type PrefEvidence struct {
	Source        string
	IP, UserAgent string
}

var (
	mPrefsLabel   = msgs.M("prefs.texts")
	mPrefsNote    = msgs.M("prefs.textsNote")
	mPrefsStopped = msgs.M("prefs.textsStopped")
	mPrefsConsent = msgs.M("prefs.textsConsent")
	mPrefsSave    = msgs.M("prefs.textsSave")
)

// Prefs is the contact's texts; false when the workspace doesn't text or
// the contact has no number.
func (s *Service) Prefs(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, code string) (PrefTexts, bool) {
	c, err := s.contact(ctx, ws, contactID)
	if err != nil || c.Phone == "" {
		return PrefTexts{}, false
	}
	var providers int64
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM sms_providers WHERE workspace_id = $1 AND removed_at IS NULL`, ws.ID).Scan(&providers); err != nil || providers == 0 {
		return PrefTexts{}, false
	}
	lang := pagelabels.Lang(code)
	out := PrefTexts{Phone: maskPhone(c.Phone), Label: mPrefsLabel.Text(lang, nil), Save: mPrefsSave.Text(lang, nil), On: map[uuid.UUID]bool{}}
	rows, err := s.Pool.Query(ctx, `SELECT list_id, status FROM channel_consents WHERE contact_id = $1 AND channel = 'sms'`, contactID)
	if err != nil {
		return PrefTexts{}, false
	}
	for rows.Next() {
		var id uuid.UUID
		var st string
		if rows.Scan(&id, &st) == nil {
			out.On[id] = st == "subscribed" || st == "pending"
		}
	}
	rows.Close()
	if sup, err := s.Q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "sms", Phone: c.Phone}); err == nil && sup.Reason == "opted_out" {
		out.Locked = true
		for id := range out.On {
			out.On[id] = false
		}
		out.Note = mPrefsStopped.Text(lang, i18n.Args{"phone": out.Phone})
		return out, true
	}
	out.Note = mPrefsNote.Text(lang, i18n.Args{"phone": out.Phone})
	return out, true
}

// SavePrefs records the switches of the lists the page showed, in the
// caller's transaction. It returns the lists whose opt-in confirmation is
// to be texted once the transaction commits (SendConfirmation).
func (s *Service) SavePrefs(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID uuid.UUID, shown, on []uuid.UUID, code string, ev PrefEvidence) ([]uuid.UUID, error) {
	c, err := s.contact(ctx, ws, contactID)
	if err != nil || c.Phone == "" || len(shown) == 0 {
		return nil, err
	}
	q := store.New(tx)
	if sup, err := q.ChanSuppressionByPhone(ctx, store.ChanSuppressionByPhoneParams{WorkspaceID: ws.ID, Channel: "sms", Phone: c.Phone}); err == nil && sup.Reason == "opted_out" {
		return nil, nil // STOP stands until START
	}
	current := map[uuid.UUID]string{}
	rows, err := tx.Query(ctx, `SELECT list_id, status FROM channel_consents WHERE contact_id = $1 AND channel = 'sms'`, contactID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id uuid.UUID
		var st string
		if err := rows.Scan(&id, &st); err != nil {
			rows.Close()
			return nil, err
		}
		current[id] = st
	}
	rows.Close()
	lang := pagelabels.Lang(code)
	brand := brandOf(ws)
	// The evidence's page is stored in English and read in each reader's
	// language (evidenceIn).
	source, page := "preference_center", evPreferenceCenter.Source()
	if ev.Source == "account_page" {
		source, page = "account_page", evAccountPage.Source()
	}
	var confirm []uuid.UUID
	for _, id := range shown {
		want := slices.Contains(on, id)
		st := current[id]
		has := st == "subscribed" || st == "pending"
		if want == has {
			continue
		}
		ev := Evidence{Source: source, Page: page, IP: ev.IP, UserAgent: ev.UserAgent}
		status := "not_subscribed"
		if want {
			status = "subscribed"
			var name string
			_ = tx.QueryRow(ctx, `SELECT name FROM lists WHERE id = $1 AND workspace_id = $2`, id, ws.ID).Scan(&name)
			if strings.TrimSpace(name) == "" {
				continue // not a list of this workspace
			}
			ev.ConsentText = mPrefsConsent.Text(lang, i18n.Args{"list": name, "brand": brand})
			if tp, ok := s.PublicTerms(ctx, ws); ok && tp.Version > 0 {
				ev.ConsentVersion = itoa(tp.Version)
			}
		}
		send, err := s.SetConsent(ctx, q, ws, contactID, c.Phone, id, status, ev)
		if err != nil {
			return nil, err
		}
		if send {
			confirm = append(confirm, id)
		}
	}
	return confirm, nil
}
