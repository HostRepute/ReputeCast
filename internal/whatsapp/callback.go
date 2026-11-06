package whatsapp

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

// The callback URL Meta posts to is /hooks/whatsapp/{workspace}/{token}:
// like every other inbound hook, it carries a token nobody can guess, 256
// random bits. The admin pastes the URL into Meta's app settings by hand
// and may need it again, so the token is kept sealed for the settings
// screen, next to its SHA-256, which is what a request is checked against,
// in constant time, before anything else.
//
// Accounts connected before had /hooks/whatsapp/{workspace}, which anyone
// who knows the workspace's slug can post to. Changing the address in Meta
// is a step someone has to take, so those accounts keep the old address
// until Meta first calls the new one (its verification request, or a
// delivery with a valid signature); from then on the old address refuses
// their deliveries. They get their token the first time the account is
// read, and the settings screen asks the admin to paste the new URL in
// Meta. An account connected now never had the old address.
//
// A new token stops the previous URL at once, and the account waits for
// Meta's verification request at the new one, as when it connected.

// callbackAD names what a sealed callback token belongs to.
func callbackAD(id uuid.UUID) string { return "whatsapp_account:" + id.String() + ":callback_token" }

// newCallback makes an account's callback token: sealed, and its hash.
func (s *Service) newCallback(id uuid.UUID) (hash, sealed []byte, err error) {
	token, hash := authn.NewToken()
	sealed, err = s.Box.SealString(token, callbackAD(id))
	return hash, sealed, err
}

// callbackURL is the address Meta posts to. It carries the account's
// token: only people who can manage providers see it, and it's never
// logged.
func (s *Service) callbackURL(ws *store.Workspace, id uuid.UUID, sealed []byte) (string, error) {
	token, err := s.Box.OpenString(sealed, callbackAD(id))
	if err != nil {
		return "", err
	}
	return strings.TrimRight(s.URLs.Base(), "/") + "/hooks/whatsapp/" + ws.Slug + "/" + token, nil
}

// hookAddress is where a request to Meta's webhook came: an account's
// callback URL (the slug and a token), or the old address (old, no token).
type hookAddress struct {
	slug, token string
	old         bool
}

// hookAccount finds the account a request to Meta's webhook is for, and
// checks the address it came to before anything else: at the callback
// URL, the token; at the old address, that Meta hasn't called the callback
// URL yet. ok is false for an address that isn't, or is no longer, the
// account's; err is pgx.ErrNoRows for a workspace without an account.
func (s *Service) hookAccount(ctx context.Context, at hookAddress) (a store.WhatsappAccount, ok bool, err error) {
	if a, err = s.Q.WhatsAppAccountBySlug(ctx, at.slug); err != nil {
		return a, false, err
	}
	if at.old {
		return a, a.CallbackMovedAt == nil, nil
	}
	return a, callbackMatches(a.CallbackTokenHash, at.token), nil
}

// callbackMatches reports whether token is the one hash was made from. The
// hashes are compared in constant time, so how long it takes says nothing
// about either token. An account without a hash has no callback URL yet.
func callbackMatches(hash []byte, token string) bool {
	return token != "" && len(hash) > 0 && authn.EqualHash(authn.HashToken(token), hash)
}

// wrongAddress counts a delivery to an address that isn't the account's
// (the old one once Meta has moved, or a wrong token) and refuses it
// without reading it, as an unknown address is. Nothing of it is stored.
func (s *Service) wrongAddress(ctx context.Context, a store.WhatsappAccount) int {
	if err := s.Q.WhatsAppAccountWrongAddress(ctx, store.WhatsAppAccountWrongAddressParams{At: s.now(), ID: a.ID}); err != nil {
		s.log().ErrorContext(ctx, "whatsapp webhook wrong address", "account", a.ID, "err", err)
	}
	s.log().WarnContext(ctx, "whatsapp webhook address", "account", a.ID)
	return http.StatusNotFound
}

// moved records, once, that Meta called the account's callback URL: from
// now on the old address refuses the account's deliveries. Meta did it,
// so the audit entry is the system's.
func (s *Service) moved(ctx context.Context, a store.WhatsappAccount, via string) {
	if a.CallbackMovedAt != nil {
		return
	}
	now := s.now()
	err := s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		n, err := q.WhatsAppAccountCallbackMoved(ctx, store.WhatsAppAccountCallbackMovedParams{At: &now, ID: a.ID})
		if err != nil || n == 0 {
			return err // n is 0 when another request recorded it first
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &a.WorkspaceID, Action: "whatsapp.callback_moved", TargetType: "whatsapp_account",
			TargetID: a.ID.String(), TargetName: a.BusinessName, Actor: systemActor, Metadata: map[string]any{"via": via}})
	})
	if err != nil {
		// Meta's next call to the callback URL records it.
		s.log().ErrorContext(ctx, "whatsapp callback moved", "account", a.ID, "err", err)
	}
}

// withCallback gives an account connected before callback tokens its
// token, the first time the account is read, and returns the account as
// stored. Nobody asked for it, so the audit entry is the system's.
func (s *Service) withCallback(ctx context.Context, ws *store.Workspace, a store.WhatsAppAccountRow) (store.WhatsAppAccountRow, error) {
	if len(a.CallbackTokenHash) > 0 {
		return a, nil
	}
	hash, sealed, err := s.newCallback(a.ID)
	if err != nil {
		return a, err
	}
	err = s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		n, err := q.WhatsAppAccountCallbackToken(ctx, store.WhatsAppAccountCallbackTokenParams{CallbackTokenHash: hash, CallbackTokenEnc: sealed, ID: a.ID})
		if err != nil || n == 0 {
			return err // n is 0 when another request gave it one first
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "whatsapp.callback_created", TargetType: "whatsapp_account",
			TargetID: a.ID.String(), TargetName: a.BusinessName, Actor: systemActor})
	})
	if err != nil {
		return a, err
	}
	return s.account(ctx, s.Q, ws)
}

// RotateCallback gives the account a new callback token (logged). The
// current callback URL stops working at once, and the account waits for
// Meta's verification request at the new one, as when it connected; the
// demo's stand-in for Meta knows the new one at once.
func (s *Service) RotateCallback(ctx context.Context, ws *store.Workspace) error {
	a, err := s.account(ctx, s.Q, ws)
	if errors.Is(err, errNoAccount) {
		return api.Conflict(mConnectFirst.Text(langOf(ctx), nil))
	}
	if err != nil {
		return err
	}
	hash, sealed, err := s.newCallback(a.ID)
	if err != nil {
		return err
	}
	now := s.now()
	status := a.Status
	if status == "connected" && !a.Demo {
		status = "waiting"
	}
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.WhatsAppAccountRotateCallback(ctx, store.WhatsAppAccountRotateCallbackParams{CallbackTokenHash: hash, CallbackTokenEnc: sealed, At: now,
			ID: a.ID}); err != nil {
			return err
		}
		if a.Demo {
			if err := q.WhatsAppAccountVerified(ctx, store.WhatsAppAccountVerifiedParams{At: now, ID: a.ID}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "whatsapp.callback_rotated", TargetType: "whatsapp_account",
			TargetID: a.ID.String(), TargetName: a.BusinessName, Before: map[string]any{"status": a.Status}, After: map[string]any{"status": status}})
	})
}
