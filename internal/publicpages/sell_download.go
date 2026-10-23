package publicpages

import (
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// The download page (/d/{token}): the files, their sizes, the link's expiry
// and the downloads left; an expired or used-up link offers "Send a new
// link", which goes only to the address on the order. A lead magnet's file
// is /l/{token}.

// downloadFileView is a file of the download page.
type downloadFileView struct {
	Name, Meta, Href string
}

// downloadBody is the download page in any of its states.
type downloadBody struct {
	State                    string
	Product                  string
	Badge                    string
	Files                    []downloadFileView
	Heading, Text, Small     string
	NewLinkAction, NewLinkTx string
	Notice                   template.HTML
	Note                     string
}

// downloadPage is /d/{token}.
func (h *Handler) downloadPage(w http.ResponseWriter, r *http.Request) {
	h.serveDownload(w, r, "")
}

func (h *Handler) serveDownload(w http.ResponseWriter, r *http.Request, notice string) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Products == nil {
		h.notFound(w, r, s)
		return
	}
	ctx := r.Context()
	token := chi.URLParam(r, "token")
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	h.sellView(v, s)
	noStore(w, v)
	v.Robots = "noindex, nofollow"
	dv, err := h.Products.Download(ctx, s.ws, token)
	if err != nil {
		v.Title = v.T("site.title", "page", v.S("dl.notFoundTitle"), "site", v.Frame.Title)
		v.Body = downloadBody{State: "missing", Heading: v.S("dl.notFoundTitle"), Text: v.S("dl.notFoundText")}
		h.renderSite(w, r, s, "sell-download", http.StatusNotFound, v)
		return
	}
	loc := location(s.ws)
	b := downloadBody{State: dv.State, Product: dv.ProductName, NewLinkAction: s.path("d", token, "new-link"),
		NewLinkTx: v.S("dl.newLink", "email", dv.EmailMasked), Note: v.S("dl.newLinkNote")}
	if notice != "" {
		b.Notice = template.HTML(template.HTMLEscapeString(notice)) //nolint:gosec // escaped
	}
	account := ""
	if v.Frame.On && v.Frame.SigninOn {
		account = strings.TrimPrefix(strings.TrimPrefix(s.path("account"), "https://"), "http://")
	}
	switch dv.State {
	case products.DownloadOK:
		v.Title = v.T("site.title", "page", dv.ProductName, "site", v.Frame.Title)
		b.Badge = v.S("dl.works", "date", v.shortDate(dv.ExpiresAt.In(loc)), "left", dv.Left(), "total", dv.Allowed)
		for _, f := range dv.Files {
			meta := v.S("dl.file", "kind", f.KindLabel, "size", v.size(f.Size))
			if f.Pages > 0 {
				meta = v.S("dl.filePages", "kind", f.KindLabel, "count", f.Pages, "size", v.size(f.Size))
			}
			b.Files = append(b.Files, downloadFileView{Name: f.Name, Meta: meta, Href: s.path("d", token, f.ID.String())})
		}
		key := "dl.bought"
		if dv.Test {
			key = "dl.boughtTest"
		}
		b.Small = v.S(key, "date", v.fullDate(dv.BoughtAt.In(loc)), "order", dv.OrderNumber)
		if account != "" {
			b.Small += " " + v.S("dl.later", "date", v.shortDate(dv.ExpiresAt.In(loc)), "account", account)
		} else {
			b.Small += " " + v.S("dl.laterNoSite", "date", v.shortDate(dv.ExpiresAt.In(loc)))
		}
	case products.DownloadExpired:
		b.Heading = v.S("dl.expiredTitle")
		b.Text = v.S("dl.expiredText", "days", dv.LinkDays, "product", dv.ProductName, "date", v.shortDate(dv.ExpiresAt.In(loc)))
	case products.DownloadUsedUp:
		b.Heading = v.S("dl.usedTitle", "count", dv.Allowed)
		b.Text = v.S("dl.usedText", "product", dv.ProductName, "count", dv.Allowed)
	case products.DownloadReplaced:
		b.Heading = v.S("dl.replacedTitle")
		b.Text = v.S("dl.replacedText", "product", dv.ProductName)
	case products.DownloadRefunded, products.DownloadRevoked:
		b.Heading = v.S("dl.revokedTitle")
		support := h.supportEmail(r, s)
		if dv.State == products.DownloadRefunded {
			b.Text = v.S("dl.refundedText")
			if dv.RefundedAt != nil {
				b.Small = v.S("dl.refundedOn", "date", v.fullDate(dv.RefundedAt.In(loc)), "email", support)
			}
		} else {
			b.Text = v.S("dl.revokedText")
			if dv.RevokedAt != nil {
				b.Small = v.S("dl.revokedOn", "date", v.fullDate(dv.RevokedAt.In(loc)), "email", support)
			}
		}
	}
	if b.Heading != "" {
		v.Title = v.T("site.title", "page", b.Heading, "site", v.Frame.Title)
	}
	v.Body = b
	h.renderSite(w, r, s, "sell-download", http.StatusOK, v)
}

// supportEmail is where buyers write: the workspace's reply-to address.
func (h *Handler) supportEmail(r *http.Request, s *site) string {
	var email string
	if err := h.Pool.QueryRow(r.Context(), `SELECT coalesce(nullif(sn.reply_to, ''), sn.email) FROM workspaces w JOIN senders sn ON sn.id = w.default_sender_id WHERE w.id = $1`, s.ws.ID).Scan(&email); err != nil {
		return s.set.Branding.PublisherName
	}
	return email
}

// downloadFile sends one of an order's files and counts the download.
func (h *Handler) downloadFile(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Products == nil {
		h.notFound(w, r, s)
		return
	}
	token := chi.URLParam(r, "token")
	fileID, err := uuid.Parse(chi.URLParam(r, "file"))
	if err != nil {
		h.notFound(w, r, s)
		return
	}
	ip := ""
	if a := clientIP(r); a != nil {
		ip = a.String()
	}
	count := r.Method == http.MethodGet && rangeFromStart(r.Header.Get("Range"))
	f, file, state, err := h.Products.OpenFile(r.Context(), s.ws, token, fileID, ip, count)
	switch {
	case errors.Is(err, products.ErrNoDownload):
		h.notFound(w, r, s)
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
	case state != products.DownloadOK:
		// The page says why (expired, used up, refunded).
		http.Redirect(w, r, s.path("d", token), http.StatusSeeOther) //nolint:gosec // G710: the download page of this link
		return
	}
	defer file.Close() //nolint:errcheck // read-only
	serveDownloadFile(w, r, f, file)
}

// rangeFromStart reports whether a request downloads a file from its
// start (a resumed download counts once).
func rangeFromStart(rng string) bool {
	rng = strings.TrimSpace(rng)
	return rng == "" || strings.HasPrefix(rng, "bytes=0-")
}

// serveDownloadFile sends a file as an attachment, resumable, with the
// time a big file needs.
func serveDownloadFile(w http.ResponseWriter, r *http.Request, f store.DownloadFile, file interface {
	Read([]byte) (int, error)
	Seek(int64, int) (int64, error)
}) {
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(3 * time.Hour))
	hd := w.Header()
	hd.Set("Content-Type", f.ContentType)
	hd.Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(f.Name, `"`, "")+`"; filename*=UTF-8''`+url.PathEscape(f.Name))
	hd.Set("Cache-Control", "private, no-store")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("X-Robots-Tag", "noindex, nofollow")
	hd.Set("Referrer-Policy", "no-referrer")
	http.ServeContent(w, r, "", f.UploadedAt, file)
}

// downloadNewLink sends a new link to the order's address.
func (h *Handler) downloadNewLink(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Products == nil {
		h.notFound(w, r, s)
		return
	}
	ctx := r.Context()
	token := chi.URLParam(r, "token")
	masked, err := h.Products.BuyerNewLink(ctx, s.ws, token)
	pl := h.visitorLang(nil, r, s, nil)
	v := &siteView{words: siteWordsOf(pl)}
	switch {
	case errors.Is(err, products.ErrNoDownload):
		h.notFound(w, r, s)
		return
	case errors.Is(err, products.ErrTooManyLinks):
		h.serveDownload(w, r, v.S("dl.tooMany"))
		return
	case err != nil:
		h.serverError(w, r, s, err)
		return
	}
	h.serveDownload(w, r, v.S("dl.sent", "email", masked))
}

// leadMagnetFile sends a form's download (the confirmation page's link).
func (h *Handler) leadMagnetFile(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Products == nil {
		h.notFound(w, r, s)
		return
	}
	count := r.Method == http.MethodGet && rangeFromStart(r.Header.Get("Range"))
	f, file, err := h.Products.OpenLeadMagnet(r.Context(), s.ws, chi.URLParam(r, "token"), count)
	switch {
	case errors.Is(err, products.ErrLinkExpired):
		pl := h.visitorLang(w, r, s, nil)
		v := &siteView{words: siteWordsOf(pl)}
		h.messageIn(w, r, s, pl, http.StatusGone, s.set.Branding.PublisherName, messageBody{Heading: v.S("dl.expiredTitle"), Text: v.S("lm.expired"),
			Actions: []action{{Label: pl.Labels.Text("page.subscribePage", nil), Href: s.links.Subscribe("")}}}, true)
		return
	case err != nil:
		h.notFound(w, r, s)
		return
	}
	defer file.Close() //nolint:errcheck // read-only
	serveDownloadFile(w, r, f, file)
}

// siteWordsOf are the site's words in a page language.
func siteWordsOf(pl pageLang) website.Words { return website.WordsIn(pl.Labels.Lang()) }
