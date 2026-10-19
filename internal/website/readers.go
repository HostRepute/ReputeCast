package website

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website/gate"
)

// Cookies of the site: first-party, on the site's own host and path.
const (
	// SessionCookie holds a member's session token.
	SessionCookie = "rc_member"
	// MeterCookie is a visitor's meter key (members-only posts read
	// without signing in), enough for a soft limit.
	MeterCookie = "rc_meter"
	// BrowserCookie names the browser that asked for a sign-in link, so its
	// code is typed where it was asked for.
	BrowserCookie = "rc_signin"
)

// Token makes a random cookie value.
func Token() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// hash is the stored form of a token.
func hash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// BrowserKey is a visitor's meter key: the hash of their meter cookie.
func BrowserKey(cookie string) string {
	if !validToken(cookie) {
		return ""
	}
	sum := sha256.Sum256([]byte("meter:" + cookie))
	return hex.EncodeToString(sum[:16])
}

func validToken(t string) bool {
	if len(t) < 20 || len(t) > 100 {
		return false
	}
	for _, r := range t {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_'
		if !ok {
			return false
		}
	}
	return true
}

// Session is a member's signed-in session.
type Session struct {
	ID        uuid.UUID
	ContactID uuid.UUID
	Email     string
	FirstName string
	LastName  string
	Method    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Name is the member's name, else the part of their address before @.
func (m *Session) Name() string {
	n := strings.TrimSpace(m.FirstName + " " + m.LastName)
	if n != "" {
		return n
	}
	local, _, _ := strings.Cut(m.Email, "@")
	return local
}

// FirstOrName is what the header pill says ("Sam").
func (m *Session) FirstOrName() string {
	if f := strings.TrimSpace(m.FirstName); f != "" {
		return f
	}
	return m.Name()
}

// SessionFor reads a member's session from its cookie (nil when there is
// none, it ended or expired, or the contact is gone). A session is touched
// at most every ten minutes.
func (s *Service) SessionFor(ctx context.Context, ws *store.Workspace, cookie string) (*Session, error) {
	if !validToken(cookie) {
		return nil, nil
	}
	now := s.now()
	row, err := s.Q.WebsiteSessionByToken(ctx, store.WebsiteSessionByTokenParams{TokenHash: hash(cookie), Now: now})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if row.WorkspaceID != ws.ID || row.ErasedAt != nil {
		return nil, nil
	}
	if now.Sub(row.LastSeenAt) > 10*time.Minute {
		if err := s.Q.WebsiteSessionTouch(ctx, store.WebsiteSessionTouchParams{ID: row.ID, At: now}); err != nil {
			s.log().WarnContext(ctx, "touch website session", "err", err)
		}
	}
	return &Session{ID: row.ID, ContactID: row.ContactID, Email: row.Email, FirstName: row.FirstName, LastName: row.LastName,
		Method: row.Method, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt}, nil
}

// Reader is who reads a page: the member of a session (with the paid
// tiers they hold) or a visitor with their meter cookie.
func (s *Service) Reader(ctx context.Context, ws *store.Workspace, sess *Session, meterCookie string) (gate.Reader, error) {
	if sess == nil {
		return gate.Reader{Browser: BrowserKey(meterCookie)}, nil
	}
	m := &gate.Member{ContactID: sess.ContactID, Email: sess.Email, Name: sess.Name(), Tiers: []gate.Tier{}}
	rows, err := s.Q.WebsiteReaderTiers(ctx, store.WebsiteReaderTiersParams{WorkspaceID: ws.ID, ContactID: sess.ContactID})
	if err != nil {
		return gate.Reader{}, err
	}
	for _, t := range rows {
		m.Tiers = append(m.Tiers, gate.Tier{ID: t.ID, Name: t.Name, Label: TierLabel(t.Name), Rank: int(t.AccessRank)})
	}
	return gate.Reader{Member: m}, nil
}

// Rules are the site's access rules and every paid tier, for pages that
// gate themselves (gate.Page).
func (s *Service) Rules(ctx context.Context, ws *store.Workspace) (gate.Rules, []gate.Tier, error) {
	c, err := s.load(ctx, s.Q, ws)
	if err != nil {
		return gate.Rules{}, nil, err
	}
	r, tiers, err := s.rules(ctx, s.Q, ws, c)
	if !c.On() {
		// Without the website nobody signs in: Members and Paid pages
		// can't be read.
		r.SigninOn = false
	}
	return r, tiers, err
}
