package verification

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/verification/adapter"
)

// SignupTimeout is how long a signup waits for the service: after 3
// seconds the signup goes through and the address is checked later.
const SignupTimeout = 3 * time.Second

// SubscribePage is the form key of the workspace's hosted subscribe page.
const SubscribePage = "subscribe_page"

// SignupCheck is a real-time check's answer for one signup.
type SignupCheck struct {
	// Service answered (empty when the form doesn't check addresses).
	Service string
	Answer  adapter.Answer
	// NoAnswer: the service failed or took more than 3 seconds.
	NoAnswer bool
	FormKey  string
}

// Checked reports whether a service answered.
func (c SignupCheck) Checked() bool { return c.Service != "" && !c.NoAnswer }

// formsService is the workspace's default service for signup forms.
func (s *Service) formsService(ctx context.Context, ws *store.Workspace) (*store.VerificationService, error) {
	rows, err := s.Q.VerifServiceList(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.VerificationService.DefaultForms {
			v := r.VerificationService
			return &v, nil
		}
	}
	return nil, nil
}

// CheckSignup checks a signup's address when the form has real-time checks
// on. It never blocks a signup on the service: a failure or 3 seconds
// without an answer lets it through, and the address is checked once the
// service answers again. Each check is counted by day and form, not
// audited one by one.
func (s *Service) CheckSignup(ctx context.Context, ws *store.Workspace, formKey, email string) SignupCheck {
	out := SignupCheck{FormKey: formKey}
	on, err := s.Q.VerifFormCheckEnabled(ctx, store.VerifFormCheckEnabledParams{WorkspaceID: ws.ID, FormKey: formKey})
	if err != nil || !on {
		return out
	}
	v, err := s.formsService(ctx, ws)
	if err != nil || v == nil {
		return out
	}
	out.Service = v.Service
	a, _, err := s.open(*v)
	if err != nil {
		out.NoAnswer = true
		s.countSignup(ctx, ws, v.Service, formKey, out)
		return out
	}
	cctx, cancel := context.WithTimeout(ctx, SignupTimeout)
	ans, err := a.Verify(cctx, email, SignupTimeout)
	cancel()
	now := s.now()
	if err != nil {
		out.NoAnswer = true
		_ = s.Q.VerifServiceFailed(ctx, store.VerifServiceFailedParams{ID: v.ID, Error: err.Error(), At: &now})
		_ = s.Q.VerifPendingInsert(ctx, store.VerifPendingInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: email, FormKey: formKey, CreatedAt: now})
		s.countSignup(ctx, ws, v.Service, formKey, out)
		return out
	}
	_ = s.Q.VerifServiceAnswered(ctx, store.VerifServiceAnsweredParams{ID: v.ID, At: &now, What: "signup form"})
	out.Answer = ans
	s.countSignup(ctx, ws, v.Service, formKey, out)
	return out
}

func (s *Service) countSignup(ctx context.Context, ws *store.Workspace, service, formKey string, c SignupCheck) {
	p := store.VerifRealtimeCountParams{WorkspaceID: ws.ID, Service: service, FormKey: formKey, Day: dayOf(s.now(), ws)}
	switch {
	case c.NoAnswer:
		p.NoAnswer = 1
	case c.Answer.Result == adapter.Valid:
		p.Valid = 1
	case c.Answer.Result == adapter.Invalid:
		p.Invalid = 1
	case c.Answer.Result == adapter.Risky:
		p.Risky = 1
	default:
		p.Unknown = 1
	}
	if err := s.Q.VerifRealtimeCount(ctx, p); err != nil {
		s.log().WarnContext(ctx, "count real-time check", "err", err)
	}
}

// SignedUp records a checked signup's result on its contact: risky and
// unknown addresses are tagged (invalid ones never got this far).
func (s *Service) SignedUp(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, c SignupCheck) {
	if !c.Checked() || contactID == uuid.Nil {
		return
	}
	a := c.Answer
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.VerifContactChecksUpsert(ctx, store.VerifContactChecksUpsertParams{WorkspaceID: ws.ID, Service: c.Service, CheckedAt: s.now(),
			ContactIds: []uuid.UUID{contactID}, Results: []string{string(a.Result)}, Reasons: []string{a.Reason}, Codes: []string{a.Code}}); err != nil {
			return err
		}
		return q.VerifSetTags(ctx, store.VerifSetTagsParams{WorkspaceID: ws.ID, ContactIds: []uuid.UUID{contactID}, Tags: []string{a.Tag()}})
	})
	if err != nil {
		s.log().WarnContext(ctx, "record real-time check", "err", err)
	}
}

func dayOf(t time.Time, ws *store.Workspace) time.Time {
	loc, err := time.LoadLocation(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

// PendingArgs checks the signups the service didn't answer for in time.
type PendingArgs struct{}

// Kind implements river.JobArgs.
func (PendingArgs) Kind() string { return "verification.pending" }

// PendingWorker runs PendingArgs.
type PendingWorker struct {
	river.WorkerDefaults[PendingArgs]
	Svc *Service
}

// Work implements river.Worker.
func (w *PendingWorker) Work(ctx context.Context, _ *river.Job[PendingArgs]) error {
	return w.Svc.CheckPending(ctx)
}

// CheckPending checks signups that went through without an answer, once
// the forms' service answers again. Invalid ones are suppressed then.
func (s *Service) CheckPending(ctx context.Context) error {
	rows, err := s.Q.VerifPendingDue(ctx, 200)
	if err != nil {
		return err
	}
	byWS := map[uuid.UUID]*store.Workspace{}
	for _, p := range rows {
		ws, ok := byWS[p.WorkspaceID]
		if !ok {
			w, err := s.Q.GetWorkspace(ctx, p.WorkspaceID)
			if err != nil {
				continue
			}
			ws = &w
			byWS[p.WorkspaceID] = ws
		}
		v, err := s.formsService(ctx, ws)
		if err != nil || v == nil {
			_ = s.Q.VerifPendingDelete(ctx, p.ID)
			continue
		}
		a, _, err := s.open(*v)
		if err != nil {
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		ans, err := a.Verify(cctx, p.Email, 10*time.Second)
		cancel()
		if err != nil {
			_ = s.Q.VerifPendingTried(ctx, p.ID)
			if p.Tries > 288 { // a day of tries every 5 minutes
				_ = s.Q.VerifPendingDelete(ctx, p.ID)
			}
			if adapter.Temporary(err) {
				break // the service is still down: try the rest later
			}
			continue
		}
		contacts, err := s.Q.VerifContactsByEmail(ctx, store.VerifContactsByEmailParams{WorkspaceID: ws.ID, Emails: []string{p.Email}})
		if err == nil && len(contacts) == 1 {
			s.recordOne(ctx, ws, *v, contacts[0].ID, p.Email, ans, "Signup form")
		}
		_ = s.Q.VerifPendingDelete(ctx, p.ID)
	}
	return nil
}

// recordOne records one contact's answer outside a job (late real-time
// checks): result, tags and, for an invalid address, the suppression.
func (s *Service) recordOne(ctx context.Context, ws *store.Workspace, v store.VerificationService, contactID uuid.UUID, email string, a adapter.Answer, label string) {
	now := s.now()
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.VerifContactChecksUpsert(ctx, store.VerifContactChecksUpsertParams{WorkspaceID: ws.ID, Service: v.Service, CheckedAt: now,
			ContactIds: []uuid.UUID{contactID}, Results: []string{string(a.Result)}, Reasons: []string{a.Reason}, Codes: []string{a.Code}}); err != nil {
			return err
		}
		if err := q.VerifSetTags(ctx, store.VerifSetTagsParams{WorkspaceID: ws.ID, ContactIds: []uuid.UUID{contactID}, Tags: []string{a.Tag()}}); err != nil {
			return err
		}
		if a.Result != adapter.Invalid {
			return nil
		}
		j := store.VerificationJob{Label: label, Kind: "selection"}
		_, err := suppress(ctx, tx, q, ws, j, ServiceName(v.Service), v.Service,
			[]store.VerifJobPendingRow{{Email: email, ContactID: &contactID}}, []adapter.Answer{a}, []int{0}, now)
		return err
	})
	if err != nil {
		s.log().WarnContext(ctx, "record a late check", "err", err)
	}
}

// VerificationFormSetting is a form's real-time check, for the form
// builder's Spam protection and its 30-day numbers.
type VerificationFormSetting struct {
	FormKey     string  `json:"formKey"`
	Enabled     bool    `json:"enabled"`
	Service     string  `json:"service,omitempty" doc:"The forms default service; empty when there is none"`
	ServiceName string  `json:"serviceName,omitempty"`
	Price       float64 `json:"price"`
	// AskedToFix is the last 30 days' invalid answers: signups asked to
	// fix their address.
	AskedToFix int64 `json:"askedToFix"`
	// Down and Waiting: the service isn't answering, and how many signups
	// went through to be checked when it's back.
	Down    bool  `json:"down"`
	Waiting int64 `json:"waiting"`
	// CanChange: forms:edit and subscribers:verify.
	CanChange bool     `json:"canChange"`
	Helpers   []string `json:"helpers"`
}

// FormSetting returns a form's real-time check.
func (s *Service) FormSetting(ctx context.Context, ws *store.Workspace, formKey string) (VerificationFormSetting, error) {
	p := api.PrincipalFrom(ctx)
	out := VerificationFormSetting{FormKey: formKey, CanChange: p.Can(authz.FormsEdit) && p.Can(authz.SubscribersVerify),
		Helpers: Helpers(ctx, s.Q, ws.ID, authz.SubscribersVerify)}
	on, err := s.Q.VerifFormCheckEnabled(ctx, store.VerifFormCheckEnabledParams{WorkspaceID: ws.ID, FormKey: formKey})
	if err != nil {
		return out, err
	}
	out.Enabled = on
	v, err := s.formsService(ctx, ws)
	if err != nil {
		return out, err
	}
	if v != nil {
		out.Service, out.ServiceName, out.Price = v.Service, ServiceName(v.Service), Money(v.PriceMicros).Dollars()
		out.Down = v.LastErrorSince != nil && v.LastErrorAt != nil && (v.LastSuccessAt == nil || v.LastSuccessAt.Before(*v.LastErrorAt))
	}
	rows, err := s.Q.VerifRealtimeInvalidByForm(ctx, store.VerifRealtimeInvalidByFormParams{WorkspaceID: ws.ID, Since: dayOf(s.now().AddDate(0, 0, -30), ws)})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.FormKey == formKey {
			out.AskedToFix = r.Invalid
		}
	}
	if out.Waiting, err = s.Q.VerifPendingCount(ctx, ws.ID); err != nil {
		return out, err
	}
	return out, nil
}

// VerificationFormInput turns a form's real-time check on or off.
type VerificationFormInput struct {
	Enabled bool `json:"enabled"`
}

// SetFormCheck turns real-time checks on or off for a form (audited). It
// needs forms:edit on top of subscribers:verify.
func (s *Service) SetFormCheck(ctx context.Context, ws *store.Workspace, formKey, formName string, in VerificationFormInput) (VerificationFormSetting, error) {
	p := api.PrincipalFrom(ctx)
	l := i18n.LangFrom(ctx)
	if !p.Can(authz.FormsEdit) {
		h := Helpers(ctx, s.Q, ws.ID, authz.FormsEdit)
		msg := mFormsEditPlain.Text(l, nil)
		if len(h) > 0 {
			msg = mFormsEdit.Text(l, i18n.Args{"helpers": i18n.OrList(l.Locale, h)})
		}
		p := api.Forbidden(authz.FormsEdit, h)
		p.Detail = msg
		return VerificationFormSetting{}, p
	}
	v, err := s.formsService(ctx, ws)
	if err != nil {
		return VerificationFormSetting{}, err
	}
	if in.Enabled && v == nil {
		return VerificationFormSetting{}, api.Conflict(mNoFormsService.Text(l, nil))
	}
	before, err := s.Q.VerifFormCheckEnabled(ctx, store.VerifFormCheckEnabledParams{WorkspaceID: ws.ID, FormKey: formKey})
	if err != nil {
		return VerificationFormSetting{}, err
	}
	service := ""
	if v != nil {
		service = v.Service
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.VerifFormCheckSet(ctx, store.VerifFormCheckSetParams{WorkspaceID: ws.ID, FormKey: formKey, Enabled: in.Enabled, UpdatedBy: actor(ctx), UpdatedAt: s.now()}); err != nil {
			return err
		}
		action := "verification.realtime_off"
		if in.Enabled {
			action = "verification.realtime_on"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "form", TargetID: formKey, TargetName: formName,
			Before: map[string]any{"realtimeCheck": before}, After: map[string]any{"realtimeCheck": in.Enabled},
			Metadata: map[string]any{"service": service}})
	})
	if err != nil {
		return VerificationFormSetting{}, err
	}
	return s.FormSetting(ctx, ws, formKey)
}

// VerificationRealtimeDay is one day's real-time checks.
type VerificationRealtimeDay struct {
	Day    string `json:"day" doc:"YYYY-MM-DD"`
	Checks int64  `json:"checks"`
}

// VerificationRealtime is a service's real-time checks by form and by day
// (the forms default's page).
type VerificationRealtime struct {
	Service    string                    `json:"service"`
	Since      *time.Time                `json:"since,omitempty"`
	Checks     int64                     `json:"checks"`
	Spent      float64                   `json:"spent"`
	AskedToFix int64                     `json:"askedToFix"`
	Forms      []VerificationFormCount   `json:"forms"`
	Total      VerificationFormCount     `json:"total"`
	Days       []VerificationRealtimeDay `json:"days"`
	Waiting    int64                     `json:"waiting"`
}

// Realtime returns a service's real-time checks since it was connected.
func (s *Service) Realtime(ctx context.Context, ws *store.Workspace, service string) (VerificationRealtime, error) {
	out := VerificationRealtime{Service: service, Forms: []VerificationFormCount{}, Days: []VerificationRealtimeDay{}}
	v, err := s.connected(ctx, ws, service)
	if err != nil {
		return out, err
	}
	since := dayOf(v.ConnectedAt, ws)
	rows, err := s.Q.VerifRealtimeDaily(ctx, store.VerifRealtimeDailyParams{WorkspaceID: ws.ID, Service: service, Since: since})
	if err != nil {
		return out, err
	}
	names, err := s.formNames(ctx, ws)
	if err != nil {
		return out, err
	}
	byForm := map[string]*VerificationFormCount{}
	order := []string{}
	byDay := map[string]int64{}
	days := []string{}
	for _, r := range rows {
		f, ok := byForm[r.FormKey]
		if !ok {
			f = &VerificationFormCount{FormKey: r.FormKey, Name: names[r.FormKey]}
			byForm[r.FormKey] = f
			order = append(order, r.FormKey)
		}
		f.Checks += int64(r.Checks)
		f.Valid += int64(r.Valid)
		f.Invalid += int64(r.Invalid)
		f.Risky += int64(r.Risky)
		f.Unknown += int64(r.Unknown)
		f.NoAnswer += int64(r.NoAnswer)
		d := r.Day.Format("2006-01-02")
		if _, ok := byDay[d]; !ok {
			days = append(days, d)
		}
		byDay[d] += int64(r.Checks)
		if out.Since == nil || r.Day.Before(*out.Since) {
			t := r.Day
			out.Since = &t
		}
	}
	for _, k := range order {
		f := *byForm[k]
		out.Forms = append(out.Forms, f)
		out.Total.Checks += f.Checks
		out.Total.Valid += f.Valid
		out.Total.Invalid += f.Invalid
		out.Total.Risky += f.Risky
		out.Total.Unknown += f.Unknown
		out.Total.NoAnswer += f.NoAnswer
	}
	// The forms first in the order the boards list them: biggest first,
	// the subscribe page last.
	sortForms(out.Forms)
	for _, d := range days {
		out.Days = append(out.Days, VerificationRealtimeDay{Day: d, Checks: byDay[d]})
	}
	out.Checks = out.Total.Checks
	out.Spent = cost(out.Checks, v.PriceMicros)
	out.AskedToFix = out.Total.Invalid
	if out.Waiting, err = s.Q.VerifPendingCount(ctx, ws.ID); err != nil {
		return out, err
	}
	return out, nil
}

func sortForms(fs []VerificationFormCount) {
	for i := 1; i < len(fs); i++ {
		for j := i; j > 0; j-- {
			a, b := fs[j-1], fs[j]
			if a.FormKey == SubscribePage && b.FormKey != SubscribePage || a.FormKey != SubscribePage && b.FormKey != SubscribePage && a.Checks < b.Checks {
				fs[j-1], fs[j] = b, a
			}
		}
	}
}

// formNames maps form keys to names ("subscribe page" for the hosted page).
func (s *Service) formNames(ctx context.Context, ws *store.Workspace) (map[string]string, error) {
	out := map[string]string{SubscribePage: "Subscribe page"}
	rows, err := s.Pool.Query(ctx, `SELECT id::text, name FROM forms WHERE workspace_id = $1`, ws.ID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return out, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

// realtimeUse is the forms default's real-time checks as a use row.
func (s *Service) realtimeUse(ctx context.Context, ws *store.Workspace, v store.VerificationService, _ time.Time) (*VerificationUse, error) {
	rt, err := s.Realtime(ctx, ws, v.Service)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && rt.Checks == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	at := v.ConnectedAt
	if rt.Since != nil {
		at = *rt.Since
	}
	return &VerificationUse{ID: "realtime-" + v.Service, Kind: "realtime", Service: v.Service, ServiceName: ServiceName(v.Service), At: at,
		By: "", Label: "", Status: "running", Addresses: rt.Checks, Cost: rt.Spent, Valid: rt.Total.Valid, Invalid: rt.Total.Invalid,
		Risky: rt.Total.Risky, Unknown: rt.Total.Unknown, NoAnswer: rt.Total.NoAnswer, Forms: rt.Forms}, nil
}
