package forms

import (
	"context"
	"errors"
	"html"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/store"
)

// errNoSystemMail means no provider can send System mail yet; the
// confirmation waits and the job retries for three days.
var errNoSystemMail = errors.New("forms: no System provider")

// ConfirmURL is the double opt-in link of a contact (App.URLs.Confirm).
func (s *Service) ConfirmURL(ws *store.Workspace, contactID uuid.UUID) string {
	return s.URLs.Confirm(ws.Slug, s.Tokens.Sign(linktoken.Claims{Kind: linktoken.Confirm, WorkspaceID: ws.ID, ContactID: contactID}))
}

// sendConfirmation emails the confirmation link of one submission on the
// System stream, in the language the form was shown in, and records it on
// the contact's consent history.
func (s *Service) sendConfirmation(ctx context.Context, ws *store.Workspace, set *Settings, submissionID, contactID uuid.UUID, reminder bool, language string) error {
	m := s.mail()
	if !m.SystemAvailable(ctx, &ws.ID) {
		return errNoSystemMail
	}
	pc, err := s.Audience.LoadPublicContact(ctx, s.Q, ws, contactID)
	if errors.Is(err, audience.ErrContactGone) {
		return s.Q.FormSubmissionSettle(ctx, store.FormSubmissionSettleParams{ID: submissionID, At: ptrTime(s.now())})
	}
	if err != nil {
		return err
	}
	lists, err := s.Audience.PublicLists(ctx, s.Q, ws)
	if err != nil {
		return err
	}
	pages, err := s.loadPagesCopy(ctx, s.Q, ws, set)
	if err != nil {
		return err
	}
	// The language the form was shown in; its pages may have gone back to
	// Incomplete since, and then the email follows the pages' fallback.
	code := languages.Base(language)
	if code == "" || !pages.Live(code) {
		code = pages.Fallback()
	}
	var pending []uuid.UUID
	var names []string
	for _, l := range lists {
		if pc.Lists[l.ID] == "pending" {
			pending = append(pending, l.ID)
			names = append(names, pages.ListName(code, l))
		}
	}
	if len(pending) == 0 {
		// Confirmed some other way (a re-permission email, the team).
		return s.Q.FormSubmissionSettle(ctx, store.FormSubmissionSettleParams{ID: submissionID, At: ptrTime(s.now())})
	}
	msg := ConfirmationEmailIn(pages.Labels(code), set.Branding, pc.FirstName, names, s.ConfirmURL(ws, contactID), reminder)
	msg.WorkspaceID, msg.To, msg.ToName = &ws.ID, pc.Email, pc.FirstName
	provider, err := m.SendSystem(ctx, msg)
	if err != nil {
		_ = s.Q.FormSubmissionMarkAttempt(ctx, submissionID)
		if errors.Is(err, mailer.ErrNoSystemProvider) {
			return errNoSystemMail
		}
		return err
	}
	now := s.now()
	note := "confirmation sent"
	if reminder {
		note = "reminder sent"
	}
	if provider != "" {
		note += " via " + provider + " (System stream)"
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if reminder {
			if err := q.FormSubmissionMarkReminded(ctx, store.FormSubmissionMarkRemindedParams{ID: submissionID, At: &now}); err != nil {
				return err
			}
		} else if err := q.FormSubmissionMarkSent(ctx, store.FormSubmissionMarkSentParams{ID: submissionID, At: &now}); err != nil {
			return err
		}
		return s.Audience.RecordOptInSent(ctx, q, contactID, pending, audience.PublicEvidence{Source: "system", Note: note, At: now})
	})
}

// ConfirmationEmail is the double opt-in email in English.
func ConfirmationEmail(b Branding, firstName string, lists []string, link string, reminder bool) mailer.Message {
	return ConfirmationEmailIn(pagelabels.English(), b, firstName, lists, link, reminder)
}

// ConfirmationEmailIn is the double opt-in email in a language: plain text
// and a small HTML version in the publisher's colours, no tracking.
func ConfirmationEmailIn(l pagelabels.Set, b Branding, firstName string, lists []string, link string, reminder bool) mailer.Message {
	pub := i18n.Args{"publisher": b.PublisherName}
	greeting := l.Text("email.greetingAnon", nil)
	if name := strings.TrimSpace(firstName); name != "" {
		greeting = l.Text("email.greeting", i18n.Args{"name": name})
	}
	subject := l.Text("email.subject", pub)
	leadKey := "email.lead"
	if reminder {
		subject = l.Text("email.reminderSubject", pub)
		leadKey = "email.reminderLead"
	}
	if len(lists) == 0 {
		leadKey += "Any"
	}
	lead := l.Text(leadKey, i18n.Args{"publisher": b.PublisherName, "lists": l.Join(lists)})
	ignore := l.Text("email.ignore", nil)
	text := greeting + "\n\n" + lead + " " + l.Text("email.open", nil) + "\n\n" + link + "\n\n" + ignore + "\n\n" + b.PublisherName + "\n"
	accent, on := b.Palette.Accent, b.Palette.OnAccent
	esc := html.EscapeString
	body := `<!doctype html><html lang="` + esc(l.Code()) + `" dir="` + pagelabels.Dir(l.Code()) + `"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + esc(subject) + `</title></head>` +
		`<body style="margin:0;padding:0;background:` + esc(b.Palette.Background) + `;font-family:'IBM Plex Sans',Arial,sans-serif;color:#1F2937">` +
		`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:` + esc(b.Palette.Background) + `"><tr><td align="center" style="padding:32px 16px">` +
		`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:520px;background:#ffffff;border:1px solid #E3E6EB;border-radius:14px"><tr><td style="padding:28px">` +
		`<p style="margin:0 0 16px;font-size:18px;font-weight:700;color:#0B1220">` + esc(b.PublisherName) + `</p>` +
		`<p style="margin:0 0 12px;font-size:15px;line-height:1.5">` + esc(greeting) + `</p>` +
		`<p style="margin:0 0 20px;font-size:15px;line-height:1.5">` + esc(lead) + `</p>` +
		`<p style="margin:0 0 20px"><a href="` + esc(link) + `" style="display:inline-block;padding:12px 20px;border-radius:9px;background:` + esc(accent) + `;color:` + esc(on) + `;font-weight:700;font-size:15px;text-decoration:none">` + esc(l.Text("email.button", nil)) + `</a></p>` +
		`<p style="margin:0;font-size:13px;line-height:1.5;color:#5B6472">` + esc(ignore) + `</p>` +
		`</td></tr></table></td></tr></table></body></html>`
	return mailer.Message{Subject: subject, Text: text, HTML: body, Kind: "double_opt_in"}
}

// Confirm completes double opt-in for a contact from the confirmation
// link: the audience confirms the pending lists; the forms they came from
// count the confirmation on the day of the submission and start their
// "form submitted" automations.
func (s *Service) Confirm(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, ev audience.PublicEvidence) (audience.ConfirmResult, error) {
	var res audience.ConfirmResult
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		res, err = s.Audience.Confirm(ctx, tx, ws, contactID, ev)
		if err != nil || len(res.Confirmed) == 0 {
			return err
		}
		if err := s.signupConfirmed(ctx, tx, ws, ConfirmEvent{ContactID: contactID, Lists: res.Confirmed, IP: ev.IP, At: ev.At}); err != nil {
			return err
		}
		q := store.New(tx)
		rows, err := q.FormSubmissionsConfirm(ctx, store.FormSubmissionsConfirmParams{ContactID: contactID, At: ptrTime(ev.At)})
		if err != nil {
			return err
		}
		loc := location(ws)
		started := map[uuid.UUID]bool{}
		for _, r := range rows {
			if r.FormID == nil {
				continue
			}
			if err := q.FormBumpDaily(ctx, store.FormBumpDailyParams{FormID: *r.FormID, Day: dayOf(r.CreatedAt, loc), Confirmed: 1}); err != nil {
				return err
			}
			if err := q.FormBumpTotals(ctx, store.FormBumpTotalsParams{ID: *r.FormID, Confirmed: 1}); err != nil {
				return err
			}
			if !started[*r.FormID] {
				started[*r.FormID] = true
				if err := automations.EmitFormSubmitted(ctx, q, ws.ID, contactID, *r.FormID); err != nil {
					return err
				}
				if err := emitFormSubmitted(ctx, tx, ws, *r.FormID, contactID, nil, r.CreatedAt, ev.At); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return res, err
}

// ConfirmationsArgs sends double opt-in emails that couldn't go out at
// submission time, reminds contacts who didn't click within 24 hours, and
// drops expired Turnstile holds.
type ConfirmationsArgs struct{}

// Kind implements river.JobArgs.
func (ConfirmationsArgs) Kind() string { return "forms.confirmations" }

// InsertOpts runs on the System queue, one run at a time.
func (ConfirmationsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSystem, UniqueOpts: river.UniqueOpts{ByPeriod: 4 * time.Minute}}
}

type ConfirmationsWorker struct {
	river.WorkerDefaults[ConfirmationsArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ConfirmationsWorker) Work(ctx context.Context, _ *river.Job[ConfirmationsArgs]) error {
	return w.S.RunConfirmations(ctx)
}

// RunConfirmations is one pass of the confirmations job.
func (s *Service) RunConfirmations(ctx context.Context) error {
	now := s.now()
	if n, err := s.Q.FormChallengesPurge(ctx, now); err != nil {
		return err
	} else if n > 0 {
		s.log().DebugContext(ctx, "expired form challenges purged", "count", n)
	}
	type wsState struct {
		ws  *store.Workspace
		set *Settings
		off bool
	}
	cache := map[uuid.UUID]*wsState{}
	state := func(id uuid.UUID) (*wsState, error) {
		if st, ok := cache[id]; ok {
			return st, nil
		}
		w, err := s.Q.GetWorkspace(ctx, id)
		if err != nil {
			return nil, err
		}
		set, err := s.settings(ctx, &w)
		if err != nil {
			return nil, err
		}
		st := &wsState{ws: &w, set: set}
		cache[id] = st
		return st, nil
	}
	unsent, err := s.Q.FormSubmissionsToSend(ctx, store.FormSubmissionsToSendParams{Since: now.Add(-72 * time.Hour), Lim: 200})
	if err != nil {
		return err
	}
	for _, r := range unsent {
		st, err := state(r.WorkspaceID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return err
		}
		if st.off {
			continue
		}
		if err := s.sendConfirmation(ctx, st.ws, st.set, r.ID, r.ContactID, false, r.Language); err != nil {
			if errors.Is(err, errNoSystemMail) {
				st.off = true
				continue
			}
			s.log().WarnContext(ctx, "send confirmation", "submission", r.ID, "err", err)
		}
	}
	due, err := s.Q.FormSubmissionsToRemind(ctx, store.FormSubmissionsToRemindParams{Before: ptrTime(now.Add(-24 * time.Hour)), Since: now.Add(-7 * 24 * time.Hour), Lim: 200})
	if err != nil {
		return err
	}
	for _, r := range due {
		st, err := state(r.WorkspaceID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return err
		}
		if st.off {
			continue
		}
		if err := s.sendConfirmation(ctx, st.ws, st.set, r.ID, r.ContactID, true, r.Language); err != nil {
			if errors.Is(err, errNoSystemMail) {
				st.off = true
				continue
			}
			s.log().WarnContext(ctx, "send confirmation reminder", "submission", r.ID, "err", err)
		}
	}
	return nil
}

func ptrTime(t time.Time) *time.Time { return &t }

// DomainsArgs checks custom domains: pending ones on every run, verified
// ones once a day.
type DomainsArgs struct{}

// Kind implements river.JobArgs.
func (DomainsArgs) Kind() string { return "forms.domains" }

// InsertOpts runs on the maintenance queue.
func (DomainsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 9 * time.Minute}}
}

type DomainsWorker struct {
	river.WorkerDefaults[DomainsArgs]
	S *Service
}

// Work implements river.Worker.
func (w *DomainsWorker) Work(ctx context.Context, _ *river.Job[DomainsArgs]) error {
	rows, err := w.S.Q.PublicSettingsDomainsToCheck(ctx, ptrTime(w.S.now().Add(-24*time.Hour)))
	if err != nil {
		return err
	}
	for _, r := range rows {
		ws, err := w.S.Q.GetWorkspace(ctx, r.WorkspaceID)
		if err != nil {
			continue
		}
		if _, err := w.S.checkDomain(ctx, &ws, false); err != nil {
			w.S.log().WarnContext(ctx, "custom domain check", "domain", r.CustomDomain, "err", err)
		}
	}
	return nil
}

// RegisterJobs adds the forms workers and their schedules.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &ConfirmationsWorker{S: s})
	river.AddWorker(reg.Workers, &DomainsWorker{S: s})
	reg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return ConfirmationsArgs{}, nil }, true)
	reg.Every(10*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return DomainsArgs{}, nil }, true)
}

// SendPreferenceLink emails a contact the link to their preference center
// ("Send me a link" on the public preferences page), in English.
func (s *Service) SendPreferenceLink(ctx context.Context, ws *store.Workspace, set *Settings, contactID uuid.UUID) error {
	return s.SendPreferenceLinkIn(ctx, ws, set, contactID, "")
}

// SendPreferenceLinkIn emails the link in the language the page asking for
// it was shown in, when its pages are Live (else their fallback).
func (s *Service) SendPreferenceLinkIn(ctx context.Context, ws *store.Workspace, set *Settings, contactID uuid.UUID, language string) error {
	m := s.mail()
	if !m.SystemAvailable(ctx, &ws.ID) {
		return errNoSystemMail
	}
	pc, err := s.Audience.LoadPublicContact(ctx, s.Q, ws, contactID)
	if err != nil {
		return err
	}
	link := s.URLs.Preferences(ws.Slug, s.Tokens.Sign(linktoken.Claims{Kind: linktoken.Preferences, WorkspaceID: ws.ID, ContactID: contactID}))
	l := pagelabels.English()
	if code := languages.Base(language); code != "" {
		if pages, err := s.loadPagesCopy(ctx, s.Q, ws, set); err == nil && pages.Live(code) {
			l = pages.Labels(code)
		}
	}
	b := set.Branding
	greeting := l.Text("email.greetingAnon", nil)
	if name := strings.TrimSpace(pc.FirstName); name != "" {
		greeting = l.Text("email.greeting", i18n.Args{"name": name})
	}
	pub := i18n.Args{"publisher": b.PublisherName}
	subject := l.Text("email.prefs.subject", pub)
	ignore := l.Text("email.prefs.ignore", nil)
	text := greeting + "\n\n" + l.Text("email.prefs.text", pub) + "\n\n" + link + "\n\n" + ignore + "\n\n" + b.PublisherName + "\n"
	esc := html.EscapeString
	body := `<!doctype html><html lang="` + esc(l.Code()) + `" dir="` + pagelabels.Dir(l.Code()) + `"><head><meta charset="utf-8"><title>` + esc(subject) + `</title></head>` +
		`<body style="margin:0;padding:24px 16px;background:` + esc(b.Palette.Background) + `;font-family:'IBM Plex Sans',Arial,sans-serif;color:#1F2937">` +
		`<div style="max-width:520px;margin:0 auto;background:#ffffff;border:1px solid #E3E6EB;border-radius:14px;padding:28px">` +
		`<p style="margin:0 0 16px;font-size:18px;font-weight:700;color:#0B1220">` + esc(b.PublisherName) + `</p>` +
		`<p style="margin:0 0 12px;font-size:15px;line-height:1.5">` + esc(greeting) + `</p>` +
		`<p style="margin:0 0 20px;font-size:15px;line-height:1.5">` + esc(l.Text("email.prefs.lead", nil)) + `</p>` +
		`<p style="margin:0 0 20px"><a href="` + esc(link) + `" style="display:inline-block;padding:12px 20px;border-radius:9px;background:` + esc(b.Palette.Accent) + `;color:` + esc(b.Palette.OnAccent) + `;font-weight:700;font-size:15px;text-decoration:none">` + esc(l.Text("email.prefs.button", nil)) + `</a></p>` +
		`<p style="margin:0;font-size:13px;line-height:1.5;color:#5B6472">` + esc(ignore) + `</p></div></body></html>`
	msg := mailer.Message{WorkspaceID: &ws.ID, To: pc.Email, ToName: pc.FirstName, Subject: subject, Text: text, HTML: body, Kind: "preference_link"}
	_, err = m.SendSystem(ctx, msg)
	return err
}

// SystemMailAvailable reports whether System mail can go out for ws.
func (s *Service) SystemMailAvailable(ctx context.Context, ws *store.Workspace) bool {
	return s.mail().SystemAvailable(ctx, &ws.ID)
}

// IsNoSystemMail reports that System mail can't go out yet.
func IsNoSystemMail(err error) bool { return errors.Is(err, errNoSystemMail) }
