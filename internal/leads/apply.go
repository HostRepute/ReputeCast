package leads

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/leads/meta"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
)

// ApplyWorker applies stored deliveries.
type ApplyWorker struct {
	river.WorkerDefaults[ApplyArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ApplyWorker) Work(ctx context.Context, job *river.Job[ApplyArgs]) error {
	id, err := uuid.Parse(job.Args.EventID)
	if err != nil {
		return river.JobCancel(err)
	}
	return w.S.ApplyEvent(ctx, id)
}

// errHeld is a delivery that waits: the connection's token is refused or
// Meta is limiting calls. It stays pending and is applied once the
// connection works again.
var errHeld = errors.New("leads: held until the connection works again")

// ApplyEvent applies one stored delivery: each lead it names is fetched
// with its Page's token and recorded once. A delivery Meta can't be asked
// about now (rate limit, refused token) stays pending; one that can't be
// read fails for good.
func (s *Service) ApplyEvent(ctx context.Context, id uuid.UUID) error {
	ev, err := s.Q.LeadEventByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if ev.Status != "pending" {
		return nil
	}
	done := func(status, msg string) error {
		return s.Q.LeadEventDone(ctx, store.LeadEventDoneParams{ID: ev.ID, Status: status, Error: clip(msg, 500), At: s.now()})
	}
	c, err := s.Q.LeadConnectionAny(ctx, ev.ConnectionID)
	if err != nil {
		return err
	}
	if c.Status == StatusDisconnected {
		return done("rejected", "the connection was disconnected")
	}
	changes, err := meta.ParseWebhook(ev.Body)
	if err != nil {
		return done("failed", "not a Page webhook: "+err.Error())
	}
	if c.Status == StatusAttention || (c.RateLimitedUntil != nil && c.RateLimitedUntil.After(s.now())) {
		return nil
	}
	cl, err := s.client(c)
	if err != nil {
		return done("failed", "the connection's credentials can't be opened")
	}
	for _, ch := range changes {
		if err := s.applyChange(ctx, c, cl, ch); err != nil {
			if errors.Is(err, errHeld) {
				return nil
			}
			if wait, ok := s.rateLimited(ctx, c, err); ok {
				return river.JobSnooze(wait)
			}
			return err
		}
	}
	return done("applied", "")
}

// rateLimited records Meta's rate limit on the connection.
func (s *Service) rateLimited(ctx context.Context, c store.LeadConnection, err error) (time.Duration, bool) {
	if !meta.RateLimited(err) {
		return 0, false
	}
	until := s.now().Add(RateLimitWait)
	if err := s.Q.LeadConnectionRateLimited(ctx, store.LeadConnectionRateLimitedParams{ID: c.ID, Until: &until}); err != nil {
		s.log().WarnContext(ctx, "lead ads: keep the rate limit", "connection", c.ID, "err", err)
	}
	return RateLimitWait, true
}

// applyChange records the lead one change names. A lead for a Page that
// isn't chosen is skipped; for a form that isn't mapped or is paused, it's
// counted.
func (s *Service) applyChange(ctx context.Context, c store.LeadConnection, cl *meta.Client, ch meta.LeadChange) error {
	sp, err := s.Q.LeadPageByMetaID(ctx, store.LeadPageByMetaIDParams{ConnectionID: c.ID, PageID: ch.PageID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !sp.Chosen {
		return nil
	}
	pg, err := s.pageOf(sp.ID, sp.PageID, sp.PageTokenSealed)
	if err != nil {
		return nil //nolint:nilerr // a Page without a token can't be read; nothing to retry
	}
	f, err := s.Q.LeadFormByMetaID(ctx, store.LeadFormByMetaIDParams{PageID: sp.ID, FormID: ch.FormID})
	if errors.Is(err, pgx.ErrNoRows) {
		// A form made after the last listing: list the Page's forms again.
		if err := s.syncPageForms(ctx, c.WorkspaceID, sp.ID, cl, pg); err != nil {
			return s.metaError(ctx, c, err)
		}
		f, err = s.Q.LeadFormByMetaID(ctx, store.LeadFormByMetaIDParams{PageID: sp.ID, FormID: ch.FormID})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
	}
	if err != nil {
		return err
	}
	if f.MappedAt == nil || f.Status != "active" {
		return s.Q.LeadFormAddWaiting(ctx, f.ID)
	}
	if seen, err := s.Q.LeadEntryExists(ctx, store.LeadEntryExistsParams{FormID: f.ID, LeadgenID: ch.LeadgenID}); err != nil || seen {
		return err
	}
	lead, err := cl.GetLead(ctx, pg, ch.LeadgenID)
	if err != nil {
		if e, ok := meta.AsError(err); ok && e.Code == 100 && e.Subcode == 33 {
			// The lead is gone at Meta (deleted, or a test lead removed).
			return nil
		}
		return s.metaError(ctx, c, err)
	}
	if lead.CreatedTime.IsZero() {
		lead.CreatedTime = ch.CreatedTime
	}
	_, err = s.record(ctx, collectOf(f), lead, SourceWebhook)
	return err
}

// metaError turns a Graph API error into what the caller does: a refused
// token puts the connection in attention and holds the delivery.
func (s *Service) metaError(ctx context.Context, c store.LeadConnection, err error) error {
	if s.noteProblem(ctx, c, err) {
		return errHeld
	}
	return err
}

// collect is how a form is collected.
type collect struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	Name           string
	Mapping        json.RawMessage
	ConsentMode    string
	ConsentKey     string
	ConsentText    string
	ConsentVersion string
	ListIDs        []uuid.UUID
	Tags           []string
	// MetaID is the form's id at Meta.
	MetaID string
}

func collectOf(f store.LeadForm) collect {
	return collect{ID: f.ID, WorkspaceID: f.WorkspaceID, Name: f.Name, Mapping: f.Mapping, ConsentMode: f.ConsentMode, ConsentKey: f.ConsentKey,
		ConsentText: f.ConsentText, ConsentVersion: f.ConsentVersion, ListIDs: f.ListIds, Tags: f.Tags, MetaID: f.FormID}
}

func collectOfRow(f store.LeadFormsCollectingRow) collect {
	return collect{ID: f.ID, WorkspaceID: f.WorkspaceID, Name: f.Name, Mapping: f.Mapping, ConsentMode: f.ConsentMode, ConsentKey: f.ConsentKey,
		ConsentText: f.ConsentText, ConsentVersion: f.ConsentVersion, ListIDs: f.ListIds, Tags: f.Tags, MetaID: f.FormID}
}

// answers are a lead's answers sorted into a signup by the form's mapping.
type answers struct {
	in    audience.SignupInput
	phone string
}

// mapLead sorts a lead's answers by the form's mapping.
func mapLead(f collect, lead meta.Lead) answers {
	var mapping map[string]string
	_ = json.Unmarshal(f.Mapping, &mapping)
	out := answers{in: audience.SignupInput{Fields: map[string]string{}}}
	for _, key := range sortedKeys(mapping) {
		v := lead.Value(key)
		if v == "" {
			continue
		}
		switch target := mapping[key]; {
		case target == TargetEmail:
			out.in.Email = v
		case target == TargetFirstName:
			out.in.FirstName = v
		case target == TargetLastName:
			out.in.LastName = v
		case target == TargetFullName:
			first, last, _ := strings.Cut(strings.Join(strings.Fields(v), " "), " ")
			out.in.FirstName, out.in.LastName = first, last
		case target == TargetCompany:
			out.in.Company = v
		case target == TargetPhone:
			out.phone = v
		case strings.HasPrefix(target, fieldPrefix):
			out.in.Fields[strings.TrimPrefix(target, fieldPrefix)] = clip(v, 500)
		}
	}
	return out
}

// record signs one lead up through its form's mapping, consent and lists,
// and keeps what happened, once per lead: a lead seen before (another
// delivery, a poll, an import) changes nothing. Meta's test leads create
// nobody; with single opt-in, a lead who didn't tick the consent box isn't
// added. The automation trigger starts when the lead joined, or, with
// double opt-in, when they confirm (Confirmed).
func (s *Service) record(ctx context.Context, f collect, lead meta.Lead, source string) (string, error) {
	ws, err := s.Q.GetWorkspace(ctx, f.WorkspaceID)
	if err != nil {
		return "", err
	}
	at := lead.CreatedTime
	if at.IsZero() {
		at = s.now()
	}
	a := mapLead(f, lead)
	evidenceText, version := f.ConsentText, f.ConsentVersion
	note := keptNote(f.Name)
	a.in.Lists = f.ListIDs
	a.in.DoubleOptIn = f.ConsentMode != "single"
	a.in.Evidence = audience.PublicEvidence{Source: ConsentSource(f.ID), ConsentText: evidenceText, ConsentVersion: version,
		Note: note.English, NoteText: note.Text, At: at}
	a.in.FormName = f.Name
	a.in.Tags = f.Tags
	outcome := ""
	var contactID *uuid.UUID
	var res audience.SignupResult
	switch {
	case lead.IsTest():
		outcome = OutcomeTest
	case f.ConsentMode == "single" && !lead.Checked[f.ConsentKey]:
		outcome = OutcomeDeclined
	}
	emitted := false
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if outcome == "" {
			lists, err := q.LeadListsOfWorkspace(ctx, ws.ID)
			if err != nil {
				return err
			}
			a.in.Lists = liveLists(a.in.Lists, lists)
			if len(a.in.Lists) == 0 || strings.TrimSpace(a.in.Email) == "" {
				outcome = OutcomeInvalid
			}
		}
		if outcome == "" {
			r, err := s.signup(ctx, tx, &ws, a.in)
			var fe *audience.FieldError
			switch {
			case errors.As(err, &fe):
				outcome = OutcomeInvalid
			case err != nil:
				return err
			default:
				res = r
				contactID = new(r.ContactID)
				switch {
				case r.Blocked:
					outcome = OutcomeBlocked
				case len(r.Pending) > 0:
					outcome = OutcomePending
				case r.Created:
					outcome = OutcomeCreated
				default:
					outcome = OutcomeUpdated
				}
			}
		}
		if contactID != nil && outcome != OutcomeBlocked && a.phone != "" {
			// Meta sends the number with its country code.
			if n, err := phone.Parse(a.phone, ""); err == nil {
				if err := q.LeadContactFillPhone(ctx, store.LeadContactFillPhoneParams{ID: *contactID, Phone: n.E164, At: at}); err != nil {
					return err
				}
			}
		}
		rows, err := q.LeadEntryInsert(ctx, store.LeadEntryInsertParams{ID: newID(), WorkspaceID: ws.ID, FormID: f.ID, LeadgenID: lead.ID, ContactID: contactID,
			Outcome: outcome, Source: source, LeadAt: at, CreatedAt: s.now()})
		if err != nil {
			return err
		}
		if rows == 0 {
			// Recorded meanwhile by another delivery or a poll: undo this one.
			return errSeen
		}
		if err := q.LeadFormLeadAt(ctx, store.LeadFormLeadAtParams{ID: f.ID, At: at}); err != nil {
			return err
		}
		if outcome == OutcomeCreated || outcome == OutcomeUpdated {
			emitted = true
			return automations.EmitLeadAdSubmitted(ctx, q, ws.ID, *contactID, f.ID)
		}
		return nil
	})
	if errors.Is(err, errSeen) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if res.NeedsConfirmation() && s.Audience != nil {
		if _, err := s.Audience.SendConfirmations(ctx, &ws, []uuid.UUID{res.ContactID}); err != nil {
			s.log().WarnContext(ctx, "lead ads: send the confirmation", "form", f.ID, "err", err)
		}
	}
	if emitted {
		s.kick(ctx)
	}
	return outcome, nil
}

var errSeen = errors.New("leads: lead recorded already")

// signup signs a lead up; a custom field value its field refuses (a word
// in a number field) is dropped rather than losing the lead.
func (s *Service) signup(ctx context.Context, tx pgx.Tx, ws *store.Workspace, in audience.SignupInput) (audience.SignupResult, error) {
	for range len(in.Fields) + 1 {
		res, err := s.Audience.Signup(ctx, tx, ws, in)
		var fe *audience.FieldError
		if errors.As(err, &fe) {
			if _, ok := in.Fields[fe.Field]; ok {
				delete(in.Fields, fe.Field)
				continue
			}
		}
		return res, err
	}
	return s.Audience.Signup(ctx, tx, ws, in)
}

// liveLists keeps the form's lists that still exist.
func liveLists(ids []uuid.UUID, lists []store.LeadListsOfWorkspaceRow) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		for _, l := range lists {
			if l.ID == id {
				out = append(out, id)
				break
			}
		}
	}
	return out
}

// Confirmed starts "Submitted a lead ad form" for a contact's double
// opt-in leads once their confirmation joined one of the form's lists
// (inside the confirmation's transaction).
func (s *Service) Confirmed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, contactID uuid.UUID, lists []uuid.UUID, at time.Time) error {
	q := store.New(tx)
	rows, err := q.LeadEntriesWaitingOf(ctx, &contactID)
	if err != nil {
		return err
	}
	started := map[uuid.UUID]bool{}
	for _, r := range rows {
		joined := false
		for _, l := range r.ListIds {
			for _, c := range lists {
				if l == c {
					joined = true
				}
			}
		}
		if !joined {
			continue
		}
		if at.IsZero() {
			at = s.now()
		}
		if err := q.LeadEntryConfirmed(ctx, store.LeadEntryConfirmedParams{ID: r.ID, At: at}); err != nil {
			return err
		}
		if started[r.FormID] {
			continue
		}
		started[r.FormID] = true
		if err := automations.EmitLeadAdSubmitted(ctx, q, ws.ID, contactID, r.FormID); err != nil {
			return err
		}
	}
	return nil
}
