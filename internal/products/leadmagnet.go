package products

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// Lead magnets (design expansion phase 7): a signup form's "After
// confirmation: offer a download". The confirmation page offers the file
// with a link of its own (signed from its id, working for the form's
// number of days); the file lives on the data volume like products'.

// FormDownload is a form's download, for the form builder.
type FormDownload struct {
	File *ProductFile `json:"file,omitempty"`
	Days int32        `json:"days"`
	// Signups, Links and Downloaded are the confirmation page's numbers.
	Links      int64 `json:"links"`
	Downloaded int64 `json:"downloaded"`
}

// FormDownloadOf reads a form's download.
func (s *Service) FormDownloadOf(ctx context.Context, ws *store.Workspace, formID uuid.UUID) (FormDownload, error) {
	d, err := s.Q.FormDownloadGet(ctx, store.FormDownloadGetParams{ID: formID, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return FormDownload{}, api.NotFound("Form")
	}
	if err != nil {
		return FormDownload{}, err
	}
	out := FormDownload{Days: d.DownloadDays}
	if d.DownloadFileID != nil {
		f, err := s.Q.DownloadFileGet(ctx, store.DownloadFileGetParams{WorkspaceID: ws.ID, ID: *d.DownloadFileID})
		if err == nil && f.RemovedAt == nil {
			pf := fileOf(f, "")
			out.File = &pf
		}
	}
	st, err := s.Q.LeadMagnetStats(ctx, formID)
	if err == nil {
		out.Links, out.Downloaded = st.Links, st.Downloaded
	}
	return out, nil
}

// FormDownloadDays sets how long a form's download links work.
func (s *Service) FormDownloadDays(ctx context.Context, ws *store.Workspace, p *authz.Principal, formID uuid.UUID, days int32) (FormDownload, error) {
	if days < 1 || days > 30 {
		return FormDownload{}, api.Invalid("body.days", "Links work for 1 to 30 days.")
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.FormDownloadGet(ctx, store.FormDownloadGetParams{ID: formID, WorkspaceID: ws.ID})
		if err != nil {
			return api.MapError(err, "Form")
		}
		if cur.DownloadDays == days {
			return nil
		}
		if err := q.FormDownloadSet(ctx, store.FormDownloadSetParams{ID: formID, WorkspaceID: ws.ID, FileID: cur.DownloadFileID, Days: days}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.download_updated", TargetType: "form", TargetID: formID.String(), TargetName: cur.Name,
			Before: map[string]any{"days": cur.DownloadDays}, After: map[string]any{"days": days}})
	})
	if err != nil {
		return FormDownload{}, err
	}
	return s.FormDownloadOf(ctx, ws, formID)
}

// LeadMagnetOffer is the confirmation page's download.
type LeadMagnetOffer struct {
	Name      string
	Kind      string
	Pages     int
	Size      int64
	URL       string
	Days      int32
	ExpiresAt time.Time
}

// LeadMagnetFor is the download a contact who confirmed through a form
// gets: a link of their own, made once and reused while it works.
func (s *Service) LeadMagnetFor(ctx context.Context, ws *store.Workspace, base string, formID, contactID uuid.UUID) (LeadMagnetOffer, bool, error) {
	d, err := s.Q.FormDownloadGet(ctx, store.FormDownloadGetParams{ID: formID, WorkspaceID: ws.ID})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return LeadMagnetOffer{}, false, err
	}
	if errors.Is(err, pgx.ErrNoRows) || d.DownloadFileID == nil {
		return LeadMagnetOffer{}, false, nil
	}
	f, err := s.Q.DownloadFileGet(ctx, store.DownloadFileGetParams{WorkspaceID: ws.ID, ID: *d.DownloadFileID})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return LeadMagnetOffer{}, false, err
	}
	if errors.Is(err, pgx.ErrNoRows) || f.RemovedAt != nil {
		return LeadMagnetOffer{}, false, nil
	}
	now := s.now()
	cid := contactID
	link, err := s.Q.LeadMagnetLinkFor(ctx, store.LeadMagnetLinkForParams{FormID: formID, ContactID: &cid})
	if err != nil || link.FileID != f.ID || !now.Before(link.ExpiresAt) {
		link = store.LeadMagnetLink{ID: newID(), WorkspaceID: ws.ID, FormID: formID, FileID: f.ID, ContactID: &cid, CreatedAt: now,
			ExpiresAt: now.Add(time.Duration(d.DownloadDays) * 24 * time.Hour)}
		if err := s.Q.LeadMagnetLinkInsert(ctx, store.LeadMagnetLinkInsertParams{ID: link.ID, WorkspaceID: ws.ID, FormID: formID, FileID: f.ID,
			ContactID: &cid, At: now, ExpiresAt: link.ExpiresAt}); err != nil {
			return LeadMagnetOffer{}, false, err
		}
	}
	out := LeadMagnetOffer{Name: f.Name, Kind: kindLabel(f.Kind, f.Name), Size: f.SizeBytes, Days: d.DownloadDays, ExpiresAt: link.ExpiresAt}
	if f.Pages != nil {
		out.Pages = int(*f.Pages)
	}
	if s.Links != nil {
		out.URL = base + "/l/" + s.Links.Sign(linktoken.Claims{Kind: TokenLeadMagnet, WorkspaceID: ws.ID, MessageID: link.ID, ContactID: cid})
	}
	return out, true, nil
}

// ErrLinkExpired is a lead magnet link past its days.
var ErrLinkExpired = errors.New("products: the download link expired")

// OpenLeadMagnet counts a lead magnet's download and opens its file.
func (s *Service) OpenLeadMagnet(ctx context.Context, ws *store.Workspace, token string, count bool) (store.DownloadFile, *os.File, error) {
	if s.Links == nil {
		return store.DownloadFile{}, nil, ErrNoDownload
	}
	c, err := s.Links.Verify(token, TokenLeadMagnet)
	if err != nil || c.WorkspaceID != ws.ID {
		return store.DownloadFile{}, nil, ErrNoDownload
	}
	link, err := s.Q.LeadMagnetLinkGet(ctx, c.MessageID)
	if err != nil || link.WorkspaceID != ws.ID {
		return store.DownloadFile{}, nil, ErrNoDownload
	}
	if !s.now().Before(link.ExpiresAt) {
		return store.DownloadFile{}, nil, ErrLinkExpired
	}
	f, err := s.Q.DownloadFileGet(ctx, store.DownloadFileGetParams{WorkspaceID: ws.ID, ID: link.FileID})
	if err != nil || f.RemovedAt != nil {
		return store.DownloadFile{}, nil, ErrNoDownload
	}
	if count {
		if err := s.Q.LeadMagnetLinkUse(ctx, link.ID); err != nil {
			return f, nil, err
		}
	}
	file, err := s.Files.Open(f.StorageKey)
	if err != nil {
		return f, nil, err
	}
	return f, file, nil
}
