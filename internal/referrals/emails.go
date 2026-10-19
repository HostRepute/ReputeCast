package referrals

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReferralEmailVersion is a saved version of a milestone email.
type ReferralEmailVersion struct {
	Version int       `json:"version"`
	Subject string    `json:"subject"`
	SavedAt time.Time `json:"savedAt"`
	SavedBy string    `json:"savedBy"`
}

// ReferralMilestoneEmailDetail is a milestone email for the composer's
// milestone mode.
type ReferralMilestoneEmailDetail struct {
	MilestoneID string `json:"milestoneId"`
	ProgramID   string `json:"programId"`
	Count       int    `json:"count"`
	RewardName  string `json:"rewardName"`
	RewardType  string `json:"rewardType"`
	ListName    string `json:"listName"`
	// Sender is the program's ("Acme Newsletter <digest@news.acme.io>");
	// SenderDefault says it's the list's default.
	Sender        string     `json:"sender"`
	SenderDefault bool       `json:"senderDefault"`
	On            bool       `json:"on"`
	Subject       string     `json:"subject"`
	Preheader     string     `json:"preheader"`
	MJML          string     `json:"mjml"`
	HTML          string     `json:"html"`
	Revision      int        `json:"revision"`
	SavedAt       *time.Time `json:"savedAt,omitempty"`
	// Versions are the saves (the email that goes out is the latest).
	Versions []ReferralEmailVersion `json:"versions"`
	// FileName and FileSize describe a download reward ("PDF, 18 MB").
	FileName    string `json:"fileName"`
	FileType    string `json:"fileType"`
	FileSize    int64  `json:"fileSize"`
	Description string `json:"description"`
	UploadedBy  string `json:"uploadedBy"`
}

// ReferralEmailSave is an autosave of a milestone email's draft.
type ReferralEmailSave struct {
	Subject      string `json:"subject" maxLength:"500"`
	Preheader    string `json:"preheader" maxLength:"500"`
	MJML         string `json:"mjml" maxLength:"2000000"`
	HTML         string `json:"html" maxLength:"4000000"`
	BaseRevision int    `json:"baseRevision" minimum:"0"`
}

// ReferralEmailSaved is what a save did.
type ReferralEmailSaved struct {
	Revision int       `json:"revision"`
	SavedAt  time.Time `json:"savedAt"`
	// Version is set when the save made a version (Save, ⌘S).
	Version int `json:"version,omitempty"`
}

func (s *Service) milestoneEmail(ctx context.Context, ws *store.Workspace, id uuid.UUID) (store.ReferralMilestone, store.ReferralProgramGetRow, error) {
	m, err := s.Q.ReferralMilestoneGet(ctx, store.ReferralMilestoneGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return m, store.ReferralProgramGetRow{}, api.MapError(err, "Milestone")
	}
	p, err := s.program(ctx, ws, m.ProgramID)
	return m, p, err
}

// MilestoneEmail is a milestone email for the composer.
func (s *Service) MilestoneEmail(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ReferralMilestoneEmailDetail, error) {
	m, p, err := s.milestoneEmail(ctx, ws, id)
	if err != nil {
		return ReferralMilestoneEmailDetail{}, err
	}
	out := ReferralMilestoneEmailDetail{MilestoneID: m.ID.String(), ProgramID: m.ProgramID.String(), Count: int(m.Count), RewardName: m.RewardName,
		RewardType: m.RewardType, ListName: p.ListName, On: m.EmailOn, Subject: m.EmailSubject, Preheader: m.EmailPreheader, MJML: m.EmailMjml,
		HTML: m.EmailHtml, Revision: int(m.EmailRevision), SavedAt: m.EmailSavedAt, Versions: []ReferralEmailVersion{}, Description: m.RewardDescription}
	if sid := p.ReferralProgram.SenderID; sid != nil {
		if sd, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: *sid}); err == nil {
			out.Sender = sd.Name + " <" + sd.Email + ">"
			def := ws.DefaultAutomationSenderID
			if def == nil {
				def = ws.DefaultSenderID
			}
			out.SenderDefault = def != nil && *def == *sid
		}
	}
	if m.FileID != nil {
		if f, err := s.Q.ReferralFileGet(ctx, *m.FileID); err == nil {
			out.FileName, out.FileType, out.FileSize, out.UploadedBy = f.FileName, f.ContentType, f.SizeBytes, f.UploadedByName
		}
	}
	if m.EmailID != nil {
		vs, err := s.Q.ReferralEmailVersions(ctx, *m.EmailID)
		if err != nil {
			return out, err
		}
		for _, v := range vs {
			out.Versions = append(out.Versions, ReferralEmailVersion{Version: int(v.Version), Subject: v.Subject, SavedAt: v.CreatedAt, SavedBy: v.CreatedByName})
		}
	}
	return out, nil
}

// SaveMilestoneEmail autosaves a milestone email's draft; version true
// also makes it the version that goes out.
func (s *Service) SaveMilestoneEmail(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ReferralEmailSave, version bool) (ReferralEmailSaved, error) {
	m, p, err := s.milestoneEmail(ctx, ws, id)
	if err != nil {
		return ReferralEmailSaved{}, err
	}
	now := s.now()
	by := actorUser(ctx)
	var out ReferralEmailSaved
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		rev, err := q.ReferralMilestoneSaveEmail(ctx, store.ReferralMilestoneSaveEmailParams{ID: id, EmailSubject: in.Subject, EmailPreheader: in.Preheader,
			EmailMjml: in.MJML, EmailHtml: in.HTML, Now: now, By: by, BaseRevision: int32(in.BaseRevision)})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.Err(http.StatusConflict, "version_conflict", s.text(ctx, msgs.M("milestone.emailStale"), nil)).WithMeta("revision", m.EmailRevision)
		}
		if err != nil {
			return err
		}
		out = ReferralEmailSaved{Revision: int(rev), SavedAt: now}
		if !version {
			return nil
		}
		emailID := m.EmailID
		if emailID == nil {
			nid := uuid.Must(uuid.NewV7())
			if err := q.ReferralEmailCreate(ctx, store.ReferralEmailCreateParams{ID: nid, WorkspaceID: ws.ID, Name: emailName(int(m.Count), m.RewardName), CreatedBy: by, Now: now}); err != nil {
				return err
			}
			if err := q.ReferralMilestoneSetEmail(ctx, store.ReferralMilestoneSetEmailParams{ID: id, EmailID: &nid, Now: now}); err != nil {
				return err
			}
			emailID = &nid
		}
		if err := s.setEmailSender(ctx, tx, p.ReferralProgram, *emailID); err != nil {
			return err
		}
		v, err := q.ReferralEmailVersionInsert(ctx, store.ReferralEmailVersionInsertParams{EmailID: *emailID, Subject: in.Subject, Preheader: in.Preheader,
			Mjml: in.MJML, Html: in.HTML, PlainText: contentcheck.PlainText(in.HTML), CreatedBy: by, Now: now})
		if err != nil {
			return err
		}
		out.Version = int(v)
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.milestone_email_saved", TargetType: "referral_milestone",
			TargetID: id.String(), TargetName: m.RewardName, After: map[string]any{"version": v, "subject": in.Subject}, Metadata: map[string]any{"related": m.ProgramID.String()}})
	})
	return out, err
}

// SetMilestoneEmailOn turns a milestone's email on, or off ("No email, I'll
// use an automation").
func (s *Service) SetMilestoneEmailOn(ctx context.Context, ws *store.Workspace, id uuid.UUID, on bool) error {
	m, _, err := s.milestoneEmail(ctx, ws, id)
	if err != nil {
		return err
	}
	if m.EmailOn == on {
		return nil
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralMilestoneEmailOn(ctx, store.ReferralMilestoneEmailOnParams{ID: id, EmailOn: on, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.milestone_updated", TargetType: "referral_milestone",
			TargetID: id.String(), TargetName: m.RewardName, Before: map[string]any{"email": m.EmailOn}, After: map[string]any{"email": on}})
	})
}

// ReferralEmailRender is a milestone email rendered for a reader.
type ReferralEmailRender struct {
	Subject   string `json:"subject"`
	Preheader string `json:"preheader"`
	HTML      string `json:"html"`
	// For names who the preview shows.
	For string `json:"for"`
}

// ReferralEmailPreviewInput renders a milestone email's draft (or what the
// composer has) as a reader.
type ReferralEmailPreviewInput struct {
	ContactID string `json:"contactId,omitempty" format:"uuid"`
	Subject   string `json:"subject,omitempty" maxLength:"500"`
	Preheader string `json:"preheader,omitempty" maxLength:"500"`
	HTML      string `json:"html,omitempty" maxLength:"4000000"`
}

// PreviewMilestoneEmail renders a milestone email as a reader would get it:
// the contact's own referral and reward, or a sample.
func (s *Service) PreviewMilestoneEmail(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ReferralEmailPreviewInput) (ReferralEmailRender, error) {
	m, _, err := s.milestoneEmail(ctx, ws, id)
	if err != nil {
		return ReferralEmailRender{}, err
	}
	subject, pre, htmlSrc := m.EmailSubject, m.EmailPreheader, m.EmailHtml
	if in.HTML != "" {
		subject, pre, htmlSrc = in.Subject, in.Preheader, in.HTML
	}
	vars, who, err := s.previewVars(ctx, ws, m, in.ContactID)
	if err != nil {
		return ReferralEmailRender{}, err
	}
	out := ReferralEmailRender{For: who}
	if out.Subject, err = liquid.Render(subject, vars); err != nil {
		return out, api.Invalid("body.subject", err.Error())
	}
	if out.Preheader, err = liquid.Render(pre, vars); err != nil {
		return out, api.Invalid("body.preheader", err.Error())
	}
	if out.HTML, err = liquid.Render(htmlSrc, vars); err != nil {
		return out, api.Invalid("body.html", err.Error())
	}
	return out, nil
}

// previewVars are a preview's variables: a contact's (their referral and,
// when they reached this milestone, their reward), or the program's sample.
func (s *Service) previewVars(ctx context.Context, ws *store.Workspace, m store.ReferralMilestone, contactRaw string) (map[string]any, string, error) {
	now := s.now()
	pages, err := s.pages(ctx, ws)
	if err != nil {
		return nil, "", err
	}
	links := map[string]string{"unsubscribe_url": "#unsubscribe", "preferences_url": "#preferences", "view_in_browser_url": "#view-in-browser"}
	if cid, err := uuid.Parse(contactRaw); err == nil {
		var email, first, last string
		if err := s.Pool.QueryRow(ctx, `SELECT email, first_name, last_name FROM contacts WHERE id = $1 AND workspace_id = $2`, cid, ws.ID).Scan(&email, &first, &last); err != nil {
			return nil, "", api.NotFound("Contact")
		}
		vars := liquid.Vars(email, first, last, nil, links)
		refs, err := s.referralVars(ctx, ws, pages, []uuid.UUID{cid}, now)
		if err != nil {
			return nil, "", err
		}
		vars["referral"], vars["referral_programs"] = map[string]any{"show": false}, map[string]any{}
		if rr, ok := refs[cid]; ok {
			vars["referral"], vars["referral_programs"] = rr.def, rr.byProgram
			if v, ok := rr.byProgram[m.ProgramID.String()]; ok {
				vars["referral"] = v
			}
		}
		var rid uuid.UUID
		if err := s.Pool.QueryRow(ctx, `SELECT id FROM referral_rewards WHERE milestone_id = $1 AND contact_id = $2`, m.ID, cid).Scan(&rid); err == nil {
			if rw, err := s.Q.ReferralRewardByID(ctx, rid); err == nil {
				vars["reward"] = s.rewardMap(ctx, ws, pages, m, rw, now)
			}
		}
		if _, ok := vars["reward"]; !ok {
			vars["reward"] = s.sampleReward(ctx, ws, m)
		}
		return vars, displayName(first, last, email), nil
	}
	vars := liquid.Vars("reader@example.com", "Sam", "", nil, links)
	ref, err := s.SampleReferral(ctx, ws, m.ProgramID, int64(m.Count), 0)
	if err != nil {
		return nil, "", err
	}
	vars["referral"] = ref
	vars["reward"] = s.sampleReward(ctx, ws, m)
	return vars, "", nil
}

func (s *Service) sampleReward(ctx context.Context, ws *store.Workspace, m store.ReferralMilestone) map[string]any {
	v := map[string]any{"name": m.RewardName, "description": m.RewardDescription, "note": "", "download_url": "", "code": "", "claim_url": "",
		"expires_at": "", "url": "#reward", "image_url": ""}
	if m.ImageID != nil {
		if a, err := s.Q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: *m.ImageID}); err == nil {
			v["image_url"] = s.mediaURL(a.ID, a.FileName)
		}
	}
	switch m.RewardType {
	case RewardDownload:
		v["download_url"] = "#download"
	case RewardCodes:
		v["code"] = "SAMPLE-CODE"
	case RewardCoupon:
		v["claim_url"], v["code"] = "#claim", "SAMPLE"
	}
	return dropEmpty(v)
}

// ReferralEmailTest sends a milestone email's draft to the member, or to
// teammates (campaigns:send).
type ReferralEmailTest struct {
	To        []string `json:"to" minItems:"1" maxItems:"5"`
	ContactID string   `json:"contactId,omitempty" format:"uuid"`
	Subject   string   `json:"subject,omitempty" maxLength:"500"`
	Preheader string   `json:"preheader,omitempty" maxLength:"500"`
	HTML      string   `json:"html,omitempty" maxLength:"4000000"`
}

// ReferralEmailTestResult says where the test went.
type ReferralEmailTestResult struct {
	Sent     []string `json:"sent"`
	Provider string   `json:"provider"`
}

// TestMilestoneEmail sends a test of a milestone email through the System
// route ("Send me a test").
func (s *Service) TestMilestoneEmail(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ReferralEmailTest) (ReferralEmailTestResult, error) {
	p := api.PrincipalFrom(ctx)
	out := ReferralEmailTestResult{Sent: []string{}}
	var to []string
	for _, raw := range in.To {
		a, err := mail.ParseAddress(strings.TrimSpace(raw))
		if err != nil {
			return out, api.Invalid("body.to", "“"+raw+"” isn't an email address.")
		}
		e := strings.ToLower(a.Address)
		if !strings.EqualFold(e, p.Email) && !p.Can(authz.CampaignsSend) {
			return out, api.Forbidden(authz.CampaignsSend, GrowHelpers(ctx, s.Q, ws.ID, authz.CampaignsSend)).WithMeta("reason", "Without campaigns:send, test emails can only go to your own address.")
		}
		to = append(to, e)
	}
	r, err := s.PreviewMilestoneEmail(ctx, ws, id, ReferralEmailPreviewInput{ContactID: in.ContactID, Subject: in.Subject, Preheader: in.Preheader, HTML: in.HTML})
	if err != nil {
		return out, err
	}
	if s.Mail == nil || s.Mail() == nil || !s.Mail().SystemAvailable(ctx, &ws.ID) {
		return out, api.Err(http.StatusConflict, api.CodeProviderUnavailable,
			"Test emails go out through the System route, and no sending provider can deliver them yet. Connect one in Sending providers, or check the email in Preview.")
	}
	for _, addr := range to {
		name, err := s.Mail().SendSystem(ctx, mailer.Message{WorkspaceID: &ws.ID, To: addr, Subject: "[Test] " + r.Subject, HTML: r.HTML,
			Text: contentcheck.PlainText(r.HTML), Kind: "milestone_email_test"})
		if err != nil {
			return out, err
		}
		out.Provider = name
		out.Sent = append(out.Sent, addr)
	}
	return out, nil
}
