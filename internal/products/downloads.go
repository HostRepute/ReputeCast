package products

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// The download page (/d/{token}): the files, their sizes, the link's
// expiry and the downloads left. Downloads are counted on the server. An
// expired or used-up link offers "Send a new link", which goes only to the
// address on the order; the page shows that address masked.

// Download states.
const (
	DownloadOK       = "ok"
	DownloadExpired  = "expired"
	DownloadUsedUp   = "used_up"
	DownloadRefunded = "refunded"
	DownloadRevoked  = "revoked"
	// DownloadReplaced is a link replaced by a newer one.
	DownloadReplaced = "replaced"
)

// DownloadFileView is one file on the download page.
type DownloadFileView struct {
	ID        uuid.UUID
	Name      string
	KindLabel string
	Pages     int
	Size      int64
}

// DownloadView is the download page.
type DownloadView struct {
	State       string
	OrderID     uuid.UUID
	LinkID      uuid.UUID
	Token       string
	ProductName string
	Files       []DownloadFileView
	ExpiresAt   time.Time
	Used        int32
	Allowed     int32
	LinkDays    int32
	OrderNumber int64
	BoughtAt    time.Time
	EmailMasked string
	RefundedAt  *time.Time
	RevokedAt   *time.Time
	Test        bool
}

// Left is the downloads the link has left.
func (v DownloadView) Left() int32 { return max(v.Allowed-v.Used, 0) }

// ErrNoDownload is a token that isn't a download link of the workspace.
var ErrNoDownload = errors.New("products: no such download link")

// resolve verifies a download token and reads its link and order.
func (s *Service) resolve(ctx context.Context, ws *store.Workspace, token string) (store.ProductOrderLink, store.ProductOrder, error) {
	if s.Links == nil || token == "" || len(token) > 300 {
		return store.ProductOrderLink{}, store.ProductOrder{}, ErrNoDownload
	}
	c, err := s.Links.Verify(token, TokenDownload)
	if err != nil || c.WorkspaceID != ws.ID {
		return store.ProductOrderLink{}, store.ProductOrder{}, ErrNoDownload
	}
	link, err := s.Q.ProductOrderLinkGet(ctx, c.MessageID)
	if err != nil || link.OrderID != c.ContactID {
		return store.ProductOrderLink{}, store.ProductOrder{}, ErrNoDownload
	}
	o, err := s.Q.ProductOrderGet(ctx, store.ProductOrderGetParams{WorkspaceID: ws.ID, ID: link.OrderID})
	if err != nil {
		return store.ProductOrderLink{}, store.ProductOrder{}, ErrNoDownload
	}
	return link, o, nil
}

// Download reads a download page.
func (s *Service) Download(ctx context.Context, ws *store.Workspace, token string) (DownloadView, error) {
	link, o, err := s.resolve(ctx, ws, token)
	if err != nil {
		return DownloadView{}, err
	}
	v := DownloadView{OrderID: o.ID, LinkID: link.ID, Token: token, ProductName: o.ProductName, ExpiresAt: link.ExpiresAt, Used: link.DownloadsUsed,
		Allowed: link.DownloadsAllowed, OrderNumber: o.Number, BoughtAt: o.PaidAt, EmailMasked: MaskEmail(o.BuyerEmail), RefundedAt: o.RefundedAt,
		RevokedAt: o.AccessRevokedAt, Test: o.Mode == string(payments.Test)}
	v.LinkDays = int32(link.ExpiresAt.Sub(link.CreatedAt).Round(24*time.Hour).Hours() / 24)
	if p, err := s.Q.ProductAny(ctx, o.ProductID); err == nil {
		if l, ok := liveOf(p); ok {
			v.ProductName = l.Name
		}
	}
	files, err := s.Q.DownloadFilesOf(ctx, store.DownloadFilesOfParams{OwnerKind: OwnerProduct, OwnerID: o.ProductID})
	if err != nil {
		return v, err
	}
	for _, f := range files {
		fv := DownloadFileView{ID: f.DownloadFile.ID, Name: f.DownloadFile.Name, KindLabel: kindLabel(f.DownloadFile.Kind, f.DownloadFile.Name), Size: f.DownloadFile.SizeBytes}
		if f.DownloadFile.Pages != nil {
			fv.Pages = int(*f.DownloadFile.Pages)
		}
		v.Files = append(v.Files, fv)
	}
	v.State = s.linkState(link, o)
	return v, nil
}

func (s *Service) linkState(link store.ProductOrderLink, o store.ProductOrder) string {
	switch {
	case o.AccessRevokedAt != nil && o.AccessRevokeReason == "refund":
		return DownloadRefunded
	case o.AccessRevokedAt != nil:
		return DownloadRevoked
	case link.SupersededAt != nil:
		return DownloadReplaced
	case link.DownloadsUsed >= link.DownloadsAllowed:
		return DownloadUsedUp
	case !s.now().Before(link.ExpiresAt):
		return DownloadExpired
	}
	return DownloadOK
}

// OpenFile counts a download and opens the file to send. The caller
// closes it. A download resumed from the middle (a Range request) doesn't
// count again: count is false for it.
func (s *Service) OpenFile(ctx context.Context, ws *store.Workspace, token string, fileID uuid.UUID, ip string, count bool) (store.DownloadFile, *os.File, string, error) {
	link, o, err := s.resolve(ctx, ws, token)
	if err != nil {
		return store.DownloadFile{}, nil, "", err
	}
	f, err := s.Q.DownloadFileGet(ctx, store.DownloadFileGetParams{WorkspaceID: ws.ID, ID: fileID})
	if err != nil || f.OwnerKind != OwnerProduct || f.OwnerID != o.ProductID || f.RemovedAt != nil {
		return store.DownloadFile{}, nil, "", ErrNoDownload
	}
	// The file first: a download that can't be sent isn't counted.
	file, err := s.Files.Open(f.StorageKey)
	if err != nil {
		return f, nil, "", err
	}
	state := DownloadOK
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		l, err := q.ProductOrderLinkLock(ctx, link.ID)
		if err != nil {
			return err
		}
		state = s.linkState(l, o)
		if !count && state == DownloadUsedUp {
			// The rest of a download already counted.
			state = DownloadOK
			return nil
		}
		if state != DownloadOK {
			return nil
		}
		if !count {
			return nil
		}
		if err := q.ProductOrderLinkUse(ctx, l.ID); err != nil {
			return err
		}
		h := sha256.Sum256([]byte(o.ID.String() + "|" + ip))
		return q.ProductOrderDownloadInsert(ctx, store.ProductOrderDownloadInsertParams{ID: newID(), OrderID: o.ID, LinkID: l.ID, FileID: f.ID, IpHash: h[:], At: s.now()})
	})
	if err != nil || state != DownloadOK {
		_ = file.Close()
		return f, nil, state, err
	}
	return f, file, DownloadOK, nil
}

// ErrTooManyLinks is a buyer asking for new links too often.
var ErrTooManyLinks = errors.New("products: too many new links")

// BuyerNewLink sends a new link to the order's address, from an expired or
// used-up link's page. It answers the masked address the link went to.
func (s *Service) BuyerNewLink(ctx context.Context, ws *store.Workspace, token string) (string, error) {
	_, o, err := s.resolve(ctx, ws, token)
	if err != nil {
		return "", err
	}
	if o.AccessRevokedAt != nil {
		return "", ErrNoDownload
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.ProductOrderLock(ctx, store.ProductOrderLockParams{WorkspaceID: ws.ID, ID: o.ID})
		if err != nil {
			return err
		}
		links, err := q.ProductOrderLinks(ctx, cur.ID)
		if err != nil {
			return err
		}
		recent := 0
		for _, l := range links {
			if l.ProductOrderLink.Reason == "buyer" && s.now().Sub(l.ProductOrderLink.CreatedAt) < 24*time.Hour {
				recent++
			}
		}
		if recent >= 3 {
			return ErrTooManyLinks
		}
		// A new link replaces the one the buyer has, even one that still
		// works: what they asked for is an email with a link.
		if _, err := s.newLink(ctx, tx, ws, cur, "buyer", nil); err != nil {
			return err
		}
		return s.event(ctx, q, cur.ID, "buyer_asked", s.now(), nil, map[string]any{"email": MaskEmail(cur.BuyerEmail)})
	})
	if err != nil {
		return "", err
	}
	return MaskEmail(o.BuyerEmail), nil
}

// AccountNewLink sends a new link for an order of a signed-in member (the
// account page's "Send me a new link"), to the address on the order.
func (s *Service) AccountNewLink(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, _ string, orderID uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.ProductOrderLock(ctx, store.ProductOrderLockParams{WorkspaceID: ws.ID, ID: orderID})
		if err != nil {
			return ErrNoDownload
		}
		if cur.ContactID == nil || *cur.ContactID != contactID || cur.AccessRevokedAt != nil {
			return ErrNoDownload
		}
		links, err := q.ProductOrderLinks(ctx, cur.ID)
		if err != nil {
			return err
		}
		recent := 0
		for _, l := range links {
			if (l.ProductOrderLink.Reason == "buyer" || l.ProductOrderLink.Reason == "account") && s.now().Sub(l.ProductOrderLink.CreatedAt) < 24*time.Hour {
				recent++
			}
		}
		if recent >= 3 {
			return ErrTooManyLinks
		}
		if _, err := s.newLink(ctx, tx, ws, cur, "account", nil); err != nil {
			return err
		}
		return s.event(ctx, q, cur.ID, "buyer_asked", s.now(), nil, map[string]any{"email": MaskEmail(cur.BuyerEmail), "from": "account"})
	})
}
