package sponsors

import (
	"context"
	"errors"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// The sponsor's emails (System mail, in the language of the page the
// request was sent from): the request received, approved with the link to
// pay, declined with the reason, expired, booked. They go out from a job,
// queued in the transaction that changed the booking, so a failed send is
// tried again. The status page's token travels sealed in the job; the
// booking keeps only its hash.

// What a sponsor email says.
const (
	MailReceived = "received"
	MailApproved = "approved"
	MailDeclined = "declined"
	MailExpired  = "expired"
	MailBooked   = "booked"
)

// MailArgs is a sponsor email to send.
type MailArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
	BookingID   uuid.UUID `json:"booking_id"`
	Email       string    `json:"email"`
	// Token is the status page's token, sealed (received, approved).
	Token []byte `json:"token,omitempty"`
}

// Kind implements river.JobArgs.
func (MailArgs) Kind() string { return "sponsors.mail" }

// InsertOpts sends from the default queue, retried.
func (MailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 8}
}

type mailWorker struct {
	river.WorkerDefaults[MailArgs]
	S *Service
}

func (w *mailWorker) Work(ctx context.Context, job *river.Job[MailArgs]) error {
	return w.S.SendMail(ctx, job.Args)
}

// queueMail queues a sponsor email in the transaction that changed the
// booking (none without a job client: the OpenAPI command).
func (s *Service) queueMail(ctx context.Context, tx pgx.Tx, a MailArgs) error {
	if s.Jobs == nil || s.Jobs() == nil {
		return nil
	}
	_, err := s.Jobs().InsertTx(ctx, tx, a, nil)
	return err
}

// mailWords is one sponsor email's words.
type mailWords struct {
	subject, heading, lead, button, link, after string
}

// SendMail sends a sponsor email.
func (s *Service) SendMail(ctx context.Context, a MailArgs) error {
	ws, err := s.Q.GetWorkspace(ctx, a.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	b, err := s.Q.SponsorRequestGet(ctx, store.SponsorRequestGetParams{ID: a.BookingID, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	d, ok := parseRequest(b.Request)
	if !ok || s.Mail == nil || s.Mail() == nil {
		return nil
	}
	token := ""
	if len(a.Token) > 0 && s.Box != nil {
		if token, err = s.Box.OpenString(a.Token, "sponsor_booking:"+b.ID.String()+":status_token"); err != nil {
			return err
		}
	}
	l := requestLang(d)
	brand := Brand{Publisher: ws.Name, Accent: "#0F766E", OnAccent: "#FFFFFF", Background: "#F6F7F9"}
	if s.Brand != nil {
		brand = s.Brand(ctx, &ws)
	}
	st, err := s.settings(ctx, ws.ID)
	if err != nil {
		return err
	}
	t := func(key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }
	loc := location(&ws)
	contact := s.contact(ctx, &ws, st)
	args := i18n.Args{"publisher": brand.Publisher, "name": d.Name, "company": d.Company, "list": b.ListName, "slot": b.SlotName, "issue": b.IssueNumber,
		"date": longDate(l, noonOn(b.IssueOn, &ws).In(loc)), "amount": money(b.PriceCents, b.Currency), "reason": b.DeclinedReason, "contact": contact,
		"host": hostOf(s.pageURL(ctx, &ws, "sponsor"))}
	if b.PaymentDueAt != nil {
		args["due"] = stamp(l, b.PaymentDueAt.In(loc))
	}
	status := ""
	if token != "" {
		status = s.pageURL(ctx, &ws, "sponsor/b/"+token)
	}
	var m mailWords
	switch a.Email {
	case MailReceived:
		m = mailWords{subject: t("mail.received.subject", args), heading: t("mail.received.heading", args), lead: t("mail.received.lead", args),
			button: t("mail.received.button", nil), link: status, after: t("mail.received.after", args)}
	case MailApproved:
		if b.Status != "awaiting_payment" {
			return nil
		}
		m = mailWords{subject: t("mail.approved.subject", args), heading: t("mail.approved.heading", args), lead: t("mail.approved.lead", args),
			button: t("mail.approved.button", args), link: status, after: t("mail.approved.after", args)}
	case MailDeclined:
		m = mailWords{subject: t("mail.declined.subject", args), heading: t("mail.declined.heading", args), lead: t("mail.declined.lead", args),
			after: t("mail.declined.after", args)}
	case MailExpired:
		m = mailWords{subject: t("mail.expired.subject", args), heading: t("mail.expired.heading", args), lead: t("mail.expired.lead", args),
			button: t("mail.expired.button", nil), link: s.pageURL(ctx, &ws, "sponsor"), after: t("mail.expired.after", args)}
	case MailBooked:
		m = mailWords{subject: t("mail.booked.subject", args), heading: t("mail.booked.heading", args), lead: t("mail.booked.lead", args),
			after: t("mail.booked.after", args)}
	default:
		return nil
	}
	if m.link == "" {
		m.button = ""
	}
	if contact != "" {
		m.after = strings.TrimSpace(m.after + " " + t("mail.questions", args))
	}
	foot := t("mail.why", args)
	_, err = s.Mail().SendSystem(ctx, mailer.Message{WorkspaceID: &ws.ID, To: d.Email, ToName: d.Name, Subject: m.subject, Text: mailText(m, foot),
		HTML: mailHTML(l, brand, m, foot), Kind: "sponsor_request", Lang: l})
	if errors.Is(err, mailer.ErrNoSystemProvider) {
		s.log().WarnContext(ctx, "sponsor email: no System provider", "booking", b.ID, "email", a.Email)
		return nil
	}
	return err
}

// longDate is a day with its weekday and year: "Tue 18 Nov 2026".
func longDate(l i18n.Lang, t time.Time) string {
	if l.Region == "" || l.Region == i18n.Standard {
		return t.Format("Mon 2 Jan 2006")
	}
	return l.Region.WeekdayDate(t)
}

// stamp is a day and time: "Fri 17 Oct, 15:00".
func stamp(l i18n.Lang, t time.Time) string {
	if l.Region == "" || l.Region == i18n.Standard {
		return t.Format("Mon 2 Jan, 15:04")
	}
	return l.Region.ShortDate(t) + ", " + l.Region.Time(t, "")
}

func mailText(m mailWords, foot string) string {
	var b strings.Builder
	b.WriteString(m.heading + "\n\n" + m.lead + "\n\n")
	if m.button != "" {
		b.WriteString(m.button + ": " + m.link + "\n\n")
	}
	if m.after != "" {
		b.WriteString(m.after + "\n\n")
	}
	b.WriteString(foot + "\n")
	return b.String()
}

// mailHTML is a small HTML version in the publisher's colours, without
// tracking.
func mailHTML(l i18n.Lang, brand Brand, m mailWords, foot string) string {
	esc := html.EscapeString
	var h strings.Builder
	h.WriteString(`<!doctype html><html lang="` + esc(string(l.Locale)) + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` +
		esc(m.subject) + `</title></head>`)
	h.WriteString(`<body style="margin:0;padding:0;background:` + esc(brand.Background) + `;font-family:'IBM Plex Sans',Arial,sans-serif;color:#1F2937">`)
	h.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + esc(brand.Background) + `"><tr><td align="center" style="padding:32px 16px">`)
	h.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#ffffff;border:1px solid #E3E6EB;border-radius:14px"><tr><td style="padding:28px">`)
	h.WriteString(`<p style="margin:0 0 16px;font-size:13px;font-weight:700;color:#5B6472">` + esc(brand.Publisher) + `</p>`)
	h.WriteString(`<h1 style="margin:0 0 12px;font-size:20px;line-height:1.3;color:#0B1220">` + esc(m.heading) + `</h1>`)
	h.WriteString(`<p style="margin:0 0 20px;font-size:15px;line-height:1.5;white-space:pre-line">` + esc(m.lead) + `</p>`)
	if m.button != "" {
		h.WriteString(`<p style="margin:0 0 20px"><a href="` + esc(m.link) + `" style="display:inline-block;padding:12px 20px;border-radius:9px;background:` + esc(brand.Accent) +
			`;color:` + esc(brand.OnAccent) + `;font-weight:700;font-size:15px;text-decoration:none">` + esc(m.button) + `</a></p>`)
	}
	if m.after != "" {
		h.WriteString(`<p style="margin:0 0 20px;font-size:13px;line-height:1.5;color:#5B6472">` + esc(m.after) + `</p>`)
	}
	h.WriteString(`<p style="margin:0;font-size:12px;line-height:1.5;color:#8A93A3">` + esc(foot) + `</p>`)
	h.WriteString(`</td></tr></table></td></tr></table></body></html>`)
	return h.String()
}

// hostOf is an address without its scheme: "news.acme.io/sponsor".
func hostOf(u string) string {
	return strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
}
