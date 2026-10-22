package push

import (
	"context"
	"errors"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/push/webpush"
	"github.com/hostrepute/reputecast/internal/store"
)

// VAPID keys and the guards (brief §3.8): the keys are generated once, the
// public key is shown and the private key sealed; rotating them ends every
// subscription. The frequency cap and quiet hours apply to every
// notification the workspace sends.

// DefaultFrequencyCap is at most 2 notifications a day per subscriber.
const DefaultFrequencyCap = 2

// Guards are the frequency cap and quiet hours as code reads them.
type Guards struct {
	Cap int
	// Quiet is whether push keeps to the window.
	Quiet  bool
	Window channels.Window
}

// guards reads the cap and quiet hours. The window is the SMS rules'
// (09:00–20:00 when the workspace has none).
func (s *Service) guards(ctx context.Context, q *store.Queries, ws *store.Workspace) (Guards, error) {
	g := Guards{Cap: DefaultFrequencyCap, Quiet: true, Window: channels.DefaultWindow}
	set, err := q.PushSettings(ctx, ws.ID)
	switch {
	case err == nil:
		g.Cap, g.Quiet = int(set.FrequencyCap), set.QuietHours
	case !notFound(err):
		return g, err
	}
	w, err := q.PushChannelWindow(ctx, ws.ID)
	switch {
	case err == nil:
		g.Window = channels.Window{From: int(w.QuietFrom), To: int(w.QuietTo)}.Clamp()
		g.Quiet = w.PushSameWindow
	case !notFound(err):
		return g, err
	}
	return g, nil
}

// PushEnableInput turns push on.
type PushEnableInput struct {
	// Contact is the address the push services can reach the workspace at:
	// a mailto: or https: URI. Empty uses the member's email.
	Contact string `json:"contact,omitempty" maxLength:"200"`
}

// checkContact reads a contact for the push services: an email address or
// a mailto: or https: URI (RFC 8292 §2.1).
func checkContact(raw string) (string, bool) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(v), "https://") {
		u, err := url.Parse(v)
		return v, err == nil && u.Host != ""
	}
	addr := strings.TrimPrefix(strings.TrimPrefix(v, "mailto:"), "MAILTO:")
	if a, err := mail.ParseAddress(addr); err != nil || a.Address != addr {
		return "", false
	}
	return "mailto:" + addr, true
}

// newKey makes a key pair and seals the private half.
func (s *Service) newKey() (uuid.UUID, string, []byte, error) {
	k, err := webpush.GenerateVAPIDKey()
	if err != nil {
		return uuid.Nil, "", nil, err
	}
	id := uuid.Must(uuid.NewV7())
	enc, err := s.Box.Seal(k.PrivateBytes(), keyAD(id))
	if err != nil {
		return uuid.Nil, "", nil, err
	}
	return id, k.PublicKeyString(), enc, nil
}

// vapid opens a stored key.
func (s *Service) vapid(k store.PushKey) (*webpush.VAPIDKey, error) {
	raw, err := s.Box.Open(k.PrivateEnc, keyAD(k.ID))
	if err != nil {
		return nil, err
	}
	return webpush.ParseVAPIDKey(raw)
}

// Enable generates the keys and turns push on, with the publication
// website as the first site when the workspace has one: it works at once
// because ReputeCast serves its service worker.
func (s *Service) Enable(ctx context.Context, ws *store.Workspace, p *authz.Principal, in PushEnableInput) error {
	l := i18n.LangFrom(ctx)
	if _, err := s.liveKey(ctx, s.Q, ws); err == nil {
		return api.Conflict(mProblemAlreadyOn.Text(l, nil))
	} else if !errors.Is(err, errNoKeys) {
		return err
	}
	contact := in.Contact
	if contact == "" {
		if u, err := s.Q.GetUser(ctx, p.UserID); err == nil {
			contact = u.Email
		}
	}
	contact, ok := checkContact(contact)
	if !ok {
		return api.Invalid("body.contact", mProblemContact.Text(l, nil))
	}
	id, pub, enc, err := s.newKey()
	if err != nil {
		return err
	}
	now := s.now()
	site := ""
	if s.Website != nil {
		site = s.Website(ctx, ws)
	}
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.PushKeyInsert(ctx, store.PushKeyInsertParams{ID: id, WorkspaceID: ws.ID, PublicKey: pub, PrivateEnc: enc, Contact: contact,
			CreatedBy: uuidPtr(p.UserID), CreatedAt: now}); err != nil {
			if strings.Contains(err.Error(), "push_keys_live_idx") {
				return api.Conflict(mProblemAlreadyOn.Text(l, nil))
			}
			return err
		}
		if err := q.PushSettingsUpsert(ctx, store.PushSettingsUpsertParams{WorkspaceID: ws.ID, FrequencyCap: DefaultFrequencyCap, QuietHours: true,
			UpdatedBy: uuidPtr(p.UserID), UpdatedAt: now}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_keys.created", TargetType: "push_keys", TargetID: id.String(),
			TargetName: "VAPID keys", After: map[string]any{"public_key": pub, "contact": contact}}); err != nil {
			return err
		}
		if site == "" {
			return nil
		}
		if _, err := q.PushSiteByOrigin(ctx, store.PushSiteByOriginParams{WorkspaceID: ws.ID, Origin: site}); err == nil {
			return nil
		}
		_, err := s.insertSite(ctx, q, ws, p, site, "publication", "", now)
		return err
	})
}

// PushContactInput changes the contact the push services get.
type PushContactInput struct {
	Contact string `json:"contact" minLength:"1" maxLength:"200"`
}

// SetContact changes the contact address.
func (s *Service) SetContact(ctx context.Context, ws *store.Workspace, in PushContactInput) error {
	l := i18n.LangFrom(ctx)
	k, err := s.liveKey(ctx, s.Q, ws)
	if errors.Is(err, errNoKeys) {
		return api.Conflict(mProblemContactNoKeys.Text(l, nil))
	}
	if err != nil {
		return err
	}
	contact, ok := checkContact(in.Contact)
	if !ok {
		return api.Invalid("body.contact", mProblemContact.Text(l, nil))
	}
	if contact == k.Contact {
		return nil
	}
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.PushKeySetContact(ctx, store.PushKeySetContactParams{ID: k.ID, Contact: contact}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_keys.contact_changed", TargetType: "push_keys", TargetID: k.ID.String(),
			TargetName: "VAPID keys", Before: map[string]any{"contact": k.Contact}, After: map[string]any{"contact": contact}})
	})
}

// PushRotation is what rotating the keys would end.
type PushRotation struct {
	Subscriptions int64             `json:"subscriptions"`
	Sites         []PushRotationRow `json:"sites"`
	// Scheduled are the notifications waiting to go that would reach
	// nobody: scheduled campaigns and live automations' push steps.
	Scheduled []string `json:"scheduled"`
	// Confirm is what to type: the number of subscriptions, without
	// separators ("4264").
	Confirm string `json:"confirm"`
}

// PushRotationRow is a site's subscriptions that end.
type PushRotationRow struct {
	Host          string `json:"host"`
	Subscriptions int64  `json:"subscriptions"`
}

// Rotation previews a rotation.
func (s *Service) Rotation(ctx context.Context, ws *store.Workspace) (PushRotation, error) {
	out := PushRotation{Sites: []PushRotationRow{}, Scheduled: []string{}}
	sites, err := s.Q.PushSites(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	counts, err := s.Q.PushSubscriptionsBySite(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	bySite := map[uuid.UUID]int64{}
	for _, c := range counts {
		bySite[c.SiteID] = c.N
		out.Subscriptions += c.N
	}
	for _, st := range sites {
		out.Sites = append(out.Sites, PushRotationRow{Host: st.Host, Subscriptions: bySite[st.ID]})
	}
	rows, err := s.Pool.Query(ctx, `SELECT c.name FROM campaigns c WHERE c.workspace_id = $1 AND c.channel = 'push' AND c.deleted_at IS NULL
		AND c.status IN ('scheduled','approved','sending') ORDER BY c.scheduled_at NULLS LAST`, ws.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil {
			out.Scheduled = append(out.Scheduled, n)
		}
	}
	rows.Close()
	steps, err := s.liveSteps(ctx, ws)
	if err != nil {
		return out, err
	}
	for _, st := range steps {
		out.Scheduled = append(out.Scheduled, st.Title)
	}
	out.Confirm = strconv.FormatInt(out.Subscriptions, 10)
	return out, nil
}

// PushRotateInput confirms a rotation by typing the number of
// subscriptions that end.
type PushRotateInput struct {
	Confirm string `json:"confirm" maxLength:"20" doc:"The number of subscriptions that end, as Rotation's confirm says."`
}

// Rotate replaces the keys. Every subscription made with the old keys ends:
// readers have to allow notifications again.
func (s *Service) Rotate(ctx context.Context, ws *store.Workspace, p *authz.Principal, in PushRotateInput) (PushRotation, error) {
	l := i18n.LangFrom(ctx)
	old, err := s.liveKey(ctx, s.Q, ws)
	if errors.Is(err, errNoKeys) {
		return PushRotation{}, api.Conflict(mProblemNoKeysToRotate.Text(l, nil))
	}
	if err != nil {
		return PushRotation{}, err
	}
	pre, err := s.Rotation(ctx, ws)
	if err != nil {
		return pre, err
	}
	typed := strings.NewReplacer(",", "", " ", "", ".", "", "\u00a0", "", "\u202f", "").Replace(strings.TrimSpace(in.Confirm))
	if typed != pre.Confirm {
		return pre, api.Invalid("body.confirm", mProblemRotateConfirm.Text(l, i18n.Args{"n": pre.Confirm}))
	}
	id, pub, enc, err := s.newKey()
	if err != nil {
		return pre, err
	}
	now := s.now()
	err = s.tx(ctx, func(q *store.Queries, tx pgx.Tx) error {
		if err := q.PushKeyRotate(ctx, store.PushKeyRotateParams{ID: old.ID, At: &now, RotatedBy: uuidPtr(p.UserID)}); err != nil {
			return err
		}
		if err := q.PushKeyInsert(ctx, store.PushKeyInsertParams{ID: id, WorkspaceID: ws.ID, PublicKey: pub, PrivateEnc: enc, Contact: old.Contact,
			CreatedBy: uuidPtr(p.UserID), CreatedAt: now}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE push_subscriptions SET removed_at = $2, removed_reason = 'rotated', endpoint = '', p256dh = NULL, auth = NULL
			WHERE workspace_id = $1 AND removed_at IS NULL`, ws.ID, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM push_test_subscriptions WHERE workspace_id = $1`, ws.ID); err != nil {
			return err
		}
		// Notifications waiting to go can't reach anyone any more.
		if _, err := tx.Exec(ctx, `UPDATE channel_messages SET status = 'cancelled', reason = 'keys_rotated', updated_at = $2
			WHERE workspace_id = $1 AND channel = 'push' AND status IN ('queued','held')`, ws.ID, now); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_keys.rotated", TargetType: "push_keys", TargetID: id.String(),
			TargetName: "VAPID keys", Before: map[string]any{"public_key": old.PublicKey}, After: map[string]any{"public_key": pub},
			Metadata: map[string]any{"subscriptions_ended": pre.Subscriptions}})
	})
	return pre, err
}

// PushSettingsInput is the frequency cap and quiet hours.
type PushSettingsInput struct {
	// FrequencyCap is the most notifications a subscriber gets a day.
	FrequencyCap int `json:"frequencyCap" minimum:"1" maximum:"24"`
	// QuietHours keeps notifications to the window texts use.
	QuietHours bool `json:"quietHours"`
}

// SaveSettings changes the frequency cap and quiet hours.
func (s *Service) SaveSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, in PushSettingsInput) error {
	before, err := s.guards(ctx, s.Q, ws)
	if err != nil {
		return err
	}
	if before.Cap == in.FrequencyCap && before.Quiet == in.QuietHours {
		return nil
	}
	return s.tx(ctx, func(q *store.Queries, _ pgx.Tx) error {
		if err := q.PushSettingsUpsert(ctx, store.PushSettingsUpsertParams{WorkspaceID: ws.ID, FrequencyCap: int32(in.FrequencyCap), QuietHours: in.QuietHours,
			UpdatedBy: uuidPtr(p.UserID), UpdatedAt: s.now()}); err != nil {
			return err
		}
		if _, err := q.PushSetSameWindow(ctx, store.PushSetSameWindowParams{WorkspaceID: ws.ID, SameWindow: in.QuietHours}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "push_settings.updated", TargetType: "workspace", TargetID: ws.ID.String(),
			TargetName: ws.Name, Before: map[string]any{"frequency_cap": before.Cap, "quiet_hours": before.Quiet},
			After: map[string]any{"frequency_cap": in.FrequencyCap, "quiet_hours": in.QuietHours}})
	})
}

// helpers names who can help with a permission.
func (s *Service) helpers(ctx context.Context, ws *store.Workspace, perm authz.Permission) []string {
	return access.Helpers(ctx, s.Q, ws.ID, perm)
}

// monthStart is the first instant of t's month in loc.
func monthStart(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, loc)
}
