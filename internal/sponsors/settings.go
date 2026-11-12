package sponsors

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Self-serve booking (M65, G-40): the workspace's sponsor page shows a
// media kit (the slot's list's subscribers and rates, an about and an
// audience text) and the self-serve slots with the issues still open.
// Sponsors request an issue with their creative; a person approves the
// request (or declines it with a reason); the sponsor pays through Stripe
// Checkout before a deadline while the issue is held for them. Payments
// are final: ReputeCast never refunds them.

// Limits of the sponsor page's texts.
const (
	maxAbout    = 600
	maxAudience = 300
)

// kitWindow is how far back the media kit's rates look.
const kitWindow = 90 * 24 * time.Hour

// SponsorMediaKit is what the sponsor page shows about the publication:
// which numbers (subscribers rounded to two significant digits, the
// average human open rate, the click rate, of the slot's list over the
// last 90 days) and two texts.
type SponsorMediaKit struct {
	Subscribers bool   `json:"subscribers"`
	OpenRate    bool   `json:"openRate"`
	ClickRate   bool   `json:"clickRate"`
	About       string `json:"about" maxLength:"600"`
	Audience    string `json:"audience" maxLength:"300"`
}

// SponsorKitStats are a list's numbers as the sponsor page shows them.
type SponsorKitStats struct {
	ListID   string `json:"listId"`
	ListName string `json:"listName"`
	// Subscribers is rounded to two significant digits (31,342 → 31,000).
	Subscribers int64 `json:"subscribers"`
	// OpenRate and ClickRate are percentages of delivered over the last
	// 90 days, absent when the list sent nothing then.
	OpenRate  *float64 `json:"openRate,omitempty"`
	ClickRate *float64 `json:"clickRate,omitempty"`
}

// SponsorSelfServe is Earn › Sponsors' "Self-serve booking" card.
type SponsorSelfServe struct {
	SelfServe bool `json:"selfServe"`
	// PageURL is the sponsor page's address (the custom domain's once it
	// is verified).
	PageURL      string          `json:"pageUrl"`
	MediaKit     SponsorMediaKit `json:"mediaKit"`
	ContactEmail string          `json:"contactEmail"`
	// PaymentsConnected: a payment account the sponsor page's checkout
	// uses is connected in Earn (PaymentsTest: in test mode only).
	PaymentsConnected bool `json:"paymentsConnected"`
	PaymentsTest      bool `json:"paymentsTest"`
	// Slots counts the self-serve slots on sale; Stats are their lists'
	// numbers as the page shows them.
	Slots     int               `json:"slots"`
	Stats     []SponsorKitStats `json:"stats"`
	UpdatedAt *time.Time        `json:"updatedAt,omitempty"`
	CanManage bool              `json:"canManage"`
	Helpers   []string          `json:"helpers"`
}

// SponsorSelfServeInput turns the sponsor page on or off and sets what it
// shows.
type SponsorSelfServeInput struct {
	SelfServe    bool            `json:"selfServe"`
	MediaKit     SponsorMediaKit `json:"mediaKit"`
	ContactEmail string          `json:"contactEmail,omitempty" maxLength:"254" doc:"Where sponsors write with questions; Earn's support address when empty."`
}

// settings are a workspace's sponsor page settings (off without a row).
type settings struct {
	on      bool
	kit     SponsorMediaKit
	contact string
	at      *time.Time
}

func (s *Service) settings(ctx context.Context, wsID uuid.UUID) (settings, error) {
	row, err := s.Q.SponsorSettingsGet(ctx, wsID)
	if errors.Is(err, pgx.ErrNoRows) {
		return settings{kit: SponsorMediaKit{Subscribers: true, OpenRate: true, ClickRate: true}}, nil
	}
	if err != nil {
		return settings{}, err
	}
	out := settings{on: row.SelfServe, contact: row.ContactEmail, at: &row.UpdatedAt}
	_ = json.Unmarshal(row.MediaKit, &out.kit)
	return out, nil
}

// selling is the payment connection the sponsor page's checkout uses.
func (s *Service) selling(ctx context.Context, ws *store.Workspace) (connected, test bool, err error) {
	if s.Earn == nil || s.Earn() == nil {
		return false, false, nil
	}
	sel, ok, err := s.Earn().Selling(ctx, ws)
	if err != nil || !ok {
		return false, false, err
	}
	return true, sel.Mode == payments.Test, nil
}

// SelfServeSettings is the "Self-serve booking" card.
func (s *Service) SelfServeSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal) (SponsorSelfServe, error) {
	st, err := s.settings(ctx, ws.ID)
	if err != nil {
		return SponsorSelfServe{}, err
	}
	out := SponsorSelfServe{SelfServe: st.on, PageURL: s.pageURL(ctx, ws, "sponsor"), MediaKit: st.kit, ContactEmail: st.contact, UpdatedAt: st.at,
		Stats: []SponsorKitStats{}, CanManage: p.Can(authz.SponsorsManage), Helpers: []string{}}
	if !out.CanManage {
		out.Helpers = s.helpers(ctx, ws.ID, authz.SponsorsManage, p.UserID, 3)
	}
	if out.PaymentsConnected, out.PaymentsTest, err = s.selling(ctx, ws); err != nil {
		return out, err
	}
	slots, err := s.Q.SponsorSelfServeSlots(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Slots = len(slots)
	seen := map[uuid.UUID]bool{}
	for _, sl := range slots {
		if seen[sl.ListID] {
			continue
		}
		seen[sl.ListID] = true
		k, err := s.kitStats(ctx, ws, sl.ListID, sl.ListName)
		if err != nil {
			return out, err
		}
		out.Stats = append(out.Stats, k)
	}
	return out, nil
}

// SaveSelfServe turns the sponsor page on or off and saves its media kit.
// Turning it on needs a payment connection: sponsors pay through it.
func (s *Service) SaveSelfServe(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SponsorSelfServeInput) (SponsorSelfServe, error) {
	kit := in.MediaKit
	kit.About = strings.TrimSpace(strings.ReplaceAll(kit.About, "\r\n", "\n"))
	kit.Audience = strings.TrimSpace(strings.ReplaceAll(kit.Audience, "\r\n", "\n"))
	if utf8.RuneCountInString(kit.About) > maxAbout {
		return SponsorSelfServe{}, invalid(ctx, "body.mediaKit.about", "problem.aboutLength", map[string]any{"max": maxAbout})
	}
	if utf8.RuneCountInString(kit.Audience) > maxAudience {
		return SponsorSelfServe{}, invalid(ctx, "body.mediaKit.audience", "problem.audienceLength", map[string]any{"max": maxAudience})
	}
	contact := emailaddr.Normalize(in.ContactEmail)
	if contact != "" && !emailaddr.Valid(contact) {
		return SponsorSelfServe{}, invalid(ctx, "body.contactEmail", "problem.email", nil)
	}
	old, err := s.settings(ctx, ws.ID)
	if err != nil {
		return SponsorSelfServe{}, err
	}
	if in.SelfServe {
		connected, _, err := s.selling(ctx, ws)
		if err != nil {
			return SponsorSelfServe{}, err
		}
		if !connected {
			return SponsorSelfServe{}, problem(ctx, 409, api.CodeConflict, "problem.connectStripe", nil)
		}
	}
	raw, _ := json.Marshal(kit)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SponsorSettingsSave(ctx, store.SponsorSettingsSaveParams{WorkspaceID: ws.ID, SelfServe: in.SelfServe, MediaKit: raw, ContactEmail: contact,
			UpdatedBy: userID(p), At: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sponsor_settings.updated", TargetType: "sponsor_settings", TargetID: ws.ID.String(),
			TargetName: text(ctx, "audit.settingsName", nil), Before: settingsAudit(old.on, old.kit, old.contact), After: settingsAudit(in.SelfServe, kit, contact)})
	})
	if err != nil {
		return SponsorSelfServe{}, err
	}
	return s.SelfServeSettings(ctx, ws, p)
}

func settingsAudit(on bool, kit SponsorMediaKit, contact string) map[string]any {
	return map[string]any{"selfServe": on, "subscribers": kit.Subscribers, "openRate": kit.OpenRate, "clickRate": kit.ClickRate, "about": kit.About,
		"audience": kit.Audience, "contactEmail": contact}
}

// pageURL is a public page's address on the workspace's preferred host.
func (s *Service) pageURL(ctx context.Context, ws *store.Workspace, path string) string {
	if s.Earn != nil && s.Earn() != nil {
		return s.Earn().PageURL(ctx, ws, path)
	}
	return s.URLs.Base() + "/p/" + ws.Slug + "/" + path
}

// twoSignificant rounds a count to two significant digits, as a media kit
// says it: 31,342 → 31,000; 987 → 990; 12 → 12.
func twoSignificant(n int64) int64 {
	if n < 100 {
		return n
	}
	p := math.Pow10(int(math.Floor(math.Log10(float64(n)))) - 1)
	return int64(math.Round(float64(n)/p) * p)
}

// kitCache keeps the media kit's numbers for ten minutes: the sponsor page
// is public, and its numbers move slowly.
type kitCache struct {
	mu      sync.Mutex
	entries map[uuid.UUID]kitEntry
}

type kitEntry struct {
	stats SponsorKitStats
	at    time.Time
}

const kitTTL = 10 * time.Minute

// kitStats are a list's numbers: its subscribers now, rounded, and its
// sends' human open and click rates over the last 90 days.
func (s *Service) kitStats(ctx context.Context, ws *store.Workspace, listID uuid.UUID, listName string) (SponsorKitStats, error) {
	now := s.now()
	s.kitOnce.Do(func() { s.kits = &kitCache{entries: map[uuid.UUID]kitEntry{}} })
	s.kits.mu.Lock()
	if e, ok := s.kits.entries[listID]; ok && now.Sub(e.at) < kitTTL && now.After(e.at) {
		s.kits.mu.Unlock()
		e.stats.ListName = listName
		return e.stats, nil
	}
	s.kits.mu.Unlock()
	subs, err := s.Q.SponsorListSubscribers(ctx, listID)
	if err != nil {
		return SponsorKitStats{}, err
	}
	eng, err := s.Q.SponsorListEngagement(ctx, store.SponsorListEngagementParams{WorkspaceID: ws.ID, Since: now.Add(-kitWindow), ListID: listID.String()})
	if err != nil {
		return SponsorKitStats{}, err
	}
	out := SponsorKitStats{ListID: listID.String(), ListName: listName, Subscribers: twoSignificant(subs)}
	if eng.Delivered > 0 {
		open := float64(eng.HumanOpens) / float64(eng.Delivered) * 100
		click := float64(eng.Clicked) / float64(eng.Delivered) * 100
		out.OpenRate, out.ClickRate = &open, &click
	}
	s.kits.mu.Lock()
	if len(s.kits.entries) > 1000 {
		s.kits.entries = map[uuid.UUID]kitEntry{}
	}
	s.kits.entries[listID] = kitEntry{stats: out, at: now}
	s.kits.mu.Unlock()
	return out, nil
}
