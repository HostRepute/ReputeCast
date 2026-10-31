package reports

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
)

// "Email me when it's final" (ReportStates): someone reading a report whose
// results are still coming in asks to be told when the 72 hours after the
// send are over. A maintenance job tells them once, in the app and by email
// as their notification preferences say, then forgets the request.

// ReportFinalAlert says whether the member asked to be emailed when the
// report's results are final.
type ReportFinalAlert struct {
	On      bool       `json:"on"`
	FinalAt *time.Time `json:"finalAt,omitempty"`
}

// FinalAlert reads whether the member asked.
func (s *Service) FinalAlert(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (ReportFinalAlert, error) {
	c, err := s.Q.ReportCampaign(ctx, store.ReportCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return ReportFinalAlert{}, api.MapError(err, "Campaign")
	}
	out := ReportFinalAlert{FinalAt: finalAt(c)}
	// An API key or agent never asked; it doesn't read what the person
	// who made it asked.
	p := api.PrincipalFrom(ctx)
	if !p.IsHuman() || p.UserID == uuid.Nil {
		return out, nil
	}
	out.On, err = s.Q.ReportFinalAlertIs(ctx, store.ReportFinalAlertIsParams{CampaignID: c.ID, UserID: p.UserID})
	return out, err
}

func finalAt(c store.ReportCampaignRow) *time.Time {
	if c.CompletedAt == nil {
		return nil
	}
	at := c.CompletedAt.Add(FinalAfter)
	return &at
}

// SetFinalAlert asks (on) or stops asking to be emailed when the results
// are final. Only a person can ask, for a campaign whose results are still
// coming in.
func (s *Service) SetFinalAlert(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID, on bool) (ReportFinalAlert, error) {
	l := i18n.LangFrom(ctx)
	c, err := s.Q.ReportCampaign(ctx, store.ReportCampaignParams{WorkspaceID: ws.ID, ID: campaignID})
	if err != nil {
		return ReportFinalAlert{}, api.MapError(err, "Campaign")
	}
	// A key's UserID is the person who made it: they didn't ask.
	p := api.PrincipalFrom(ctx)
	if !p.IsHuman() || p.UserID == uuid.Nil {
		return ReportFinalAlert{}, api.Err(http.StatusForbidden, api.CodeHumansOnly, mFinalAlertPeople.Text(l, nil))
	}
	out := ReportFinalAlert{On: on, FinalAt: finalAt(c)}
	if on {
		if unsent[c.Status] || c.StartedAt == nil {
			return ReportFinalAlert{}, api.Conflict(mFinalAlertNotSent.Text(l, nil))
		}
		if out.FinalAt != nil && !s.now().Before(*out.FinalAt) {
			return ReportFinalAlert{}, api.Conflict(mFinalAlertFinal.Text(l, nil))
		}
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		action := "report_final_alert.set"
		if on {
			if err := q.ReportFinalAlertSet(ctx, store.ReportFinalAlertSetParams{CampaignID: c.ID, UserID: p.UserID, WorkspaceID: ws.ID}); err != nil {
				return err
			}
		} else {
			n, err := q.ReportFinalAlertRemove(ctx, store.ReportFinalAlertRemoveParams{CampaignID: c.ID, UserID: p.UserID})
			if err != nil || n == 0 {
				return err
			}
			action = "report_final_alert.remove"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: action, TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Metadata: map[string]any{"finalAt": out.FinalAt}})
	})
	return out, err
}

// SendFinalAlerts tells everyone who asked about a report whose results
// became final, if they can still read it, and returns how many were told.
func (s *Service) SendFinalAlerts(ctx context.Context) (int, error) {
	due, err := s.Q.ReportFinalAlertsDue(ctx, s.now().Add(-FinalAfter))
	if err != nil || len(due) == 0 {
		return 0, err
	}
	told := 0
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		readers := map[uuid.UUID][]uuid.UUID{}
		for _, a := range due {
			ids, ok := readers[a.WorkspaceID]
			if !ok {
				if ids, err = notify.MembersWith(ctx, tx, a.WorkspaceID, authz.ReportsView); err != nil {
					return err
				}
				readers[a.WorkspaceID] = ids
			}
			if slices.Contains(ids, a.UserID) {
				wsID := a.WorkspaceID
				args := i18n.Args{"campaign": a.CampaignName}
				if err := notify.ToUser(ctx, tx, a.UserID, notify.Notification{WorkspaceID: &wsID, Kind: notify.KindResultsFinal, Severity: "info",
					TitleText: mFinalAlertTitle.Keep(args), BodyText: mFinalAlertBody.Keep(args),
					Link: "/w/" + a.WorkspaceSlug + "/campaigns/" + a.CampaignID.String() + "/report",
					Data: map[string]any{"campaignId": a.CampaignID.String()}}); err != nil {
					return err
				}
				told++
			}
			if _, err := q.ReportFinalAlertRemove(ctx, store.ReportFinalAlertRemoveParams{CampaignID: a.CampaignID, UserID: a.UserID}); err != nil {
				return err
			}
		}
		return nil
	})
	return told, err
}

// FinalAlertsArgs sends the alerts that are due (periodic).
type FinalAlertsArgs struct{}

// Kind implements river.JobArgs.
func (FinalAlertsArgs) Kind() string { return "reports.final_alerts" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (FinalAlertsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 5 * time.Minute}}
}

// FinalAlertsWorker runs FinalAlertsArgs.
type FinalAlertsWorker struct {
	river.WorkerDefaults[FinalAlertsArgs]
	S *Service
}

// Work implements river.Worker.
func (w *FinalAlertsWorker) Work(ctx context.Context, _ *river.Job[FinalAlertsArgs]) error {
	_, err := w.S.SendFinalAlerts(ctx)
	return err
}
