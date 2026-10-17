package referrals

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
)

// ReferralImage is a reward's picture, from the media library.
type ReferralImage struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int64  `json:"size"`
}

// ReferralFile is a file for download rewards.
type ReferralFile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	Downloads   int64     `json:"downloads"`
	UploadedBy  string    `json:"uploadedBy"`
	UploadedAt  time.Time `json:"uploadedAt"`
}

// ReferralMilestoneStats are a milestone's rewards: reached, sent, used,
// waiting and failed.
type ReferralMilestoneStats struct {
	Reached      int64 `json:"reached"`
	Sent         int64 `json:"sent"`
	Used         int64 `json:"used"`
	Waiting      int64 `json:"waiting"`
	Failed       int64 `json:"failed"`
	WaitingCodes int64 `json:"waitingCodes"`
	ToFulfill    int64 `json:"toFulfill"`
}

// ReferralMilestoneHealth says whether a reward can go out.
type ReferralMilestoneHealth struct {
	Status string `json:"status" enum:"ok,warn,bad"`
	// Kind: healthy, codes_low, codes_out, webhook_failing, stripe_missing,
	// file_missing.
	Kind string `json:"kind" enum:"healthy,codes_low,codes_out,webhook_failing,stripe_missing,file_missing"`
	// CodesLeft and RunOutDays (the current pace) for code rewards.
	CodesLeft  int64    `json:"codesLeft"`
	RunOutDays *float64 `json:"runOutDays,omitempty"`
	Waiting    int64    `json:"waiting"`
	// LastStatus and Failures describe a failing webhook.
	LastStatus int `json:"lastStatus"`
	Failures   int `json:"failures"`
}

// ReferralMilestoneEmail is a milestone's email.
type ReferralMilestoneEmail struct {
	On       bool       `json:"on"`
	EmailID  string     `json:"emailId"`
	Subject  string     `json:"subject"`
	Versions int64      `json:"versions"`
	SavedAt  *time.Time `json:"savedAt,omitempty"`
}

// ReferralMilestoneView is a milestone with its reward.
type ReferralMilestoneView struct {
	ID                string         `json:"id"`
	Count             int            `json:"count"`
	RewardName        string         `json:"rewardName"`
	RewardDescription string         `json:"rewardDescription"`
	Image             *ReferralImage `json:"image,omitempty"`
	Type              string         `json:"type" enum:"download,codes,manual,coupon,webhook"`
	File              *ReferralFile  `json:"file,omitempty"`
	DownloadURL       string         `json:"downloadUrl"`
	CodesUploaded     int64          `json:"codesUploaded"`
	CodesLeft         int64          `json:"codesLeft"`
	CodesAlertBelow   int            `json:"codesAlertBelow"`
	AskAddress        bool           `json:"askAddress"`
	CouponOfferID     string         `json:"couponOfferId"`
	CouponOfferName   string         `json:"couponOfferName"`
	CouponValidDays   int            `json:"couponValidDays"`
	WebhookURL        string         `json:"webhookUrl"`
	// WebhookSecretSet: a signing secret is stored (it's never returned).
	WebhookSecretSet bool                    `json:"webhookSecretSet"`
	WebhookStatus    *int                    `json:"webhookStatus,omitempty"`
	WebhookCalledAt  *time.Time              `json:"webhookCalledAt,omitempty"`
	Email            ReferralMilestoneEmail  `json:"email"`
	Stats            ReferralMilestoneStats  `json:"stats"`
	Health           ReferralMilestoneHealth `json:"health"`
}

// ReferralCouponOffer is an offer a coupon reward can use.
type ReferralCouponOffer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Discount  string `json:"discount"`
	AppliesTo string `json:"appliesTo"`
	Issued    int64  `json:"issued"`
	Redeemed  int64  `json:"redeemed"`
}

// ReferralStripe says whether coupon rewards can be picked.
type ReferralStripe struct {
	Connected bool                  `json:"connected"`
	Account   string                `json:"account"`
	Offers    []ReferralCouponOffer `json:"offers"`
}

// ReferralFulfillItem is a manual reward waiting to be sent by hand.
type ReferralFulfillItem struct {
	RewardID   string                   `json:"rewardId"`
	ContactID  string                   `json:"contactId"`
	Referrer   string                   `json:"referrer"`
	Email      string                   `json:"email"`
	Milestone  int                      `json:"milestone"`
	RewardName string                   `json:"rewardName"`
	ReachedAt  time.Time                `json:"reachedAt"`
	Address    *ReferralShippingAddress `json:"address,omitempty"`
}

// ReferralShippingAddress is where a manual reward goes. It's deleted 30
// days after the reward is marked fulfilled.
type ReferralShippingAddress struct {
	Name    string `json:"name" maxLength:"200"`
	Street  string `json:"street" maxLength:"300"`
	City    string `json:"city" maxLength:"200"`
	Country string `json:"country" maxLength:"100"`
}

// ReferralMilestoneList is Referrals › Milestones.
type ReferralMilestoneList struct {
	ProgramID string                  `json:"programId"`
	ListName  string                  `json:"listName"`
	Sender    string                  `json:"sender"`
	Items     []ReferralMilestoneView `json:"items"`
	Files     []ReferralFile          `json:"files"`
	ToFulfill []ReferralFulfillItem   `json:"toFulfill"`
	Stripe    ReferralStripe          `json:"stripe"`
	// ShowAddresses: shipping addresses show to members with subscribers:view.
	ShowAddresses bool           `json:"showAddresses"`
	Helpers       GrowHelperSets `json:"helpers"`
}

// ReferralMilestoneInput creates or changes a milestone.
type ReferralMilestoneInput struct {
	Count             int    `json:"count" minimum:"1" maximum:"100000"`
	RewardName        string `json:"rewardName" maxLength:"120"`
	RewardDescription string `json:"rewardDescription" maxLength:"200"`
	ImageID           string `json:"imageId,omitempty" format:"uuid"`
	Type              string `json:"type" enum:"download,codes,manual,coupon,webhook"`
	FileID            string `json:"fileId,omitempty" format:"uuid"`
	DownloadURL       string `json:"downloadUrl,omitempty" maxLength:"2000"`
	CodesAlertBelow   int    `json:"codesAlertBelow,omitempty" minimum:"0" maximum:"100000"`
	AskAddress        bool   `json:"askAddress,omitempty"`
	CouponOfferID     string `json:"couponOfferId,omitempty" maxLength:"200"`
	CouponValidDays   int    `json:"couponValidDays,omitempty" enum:"0,30,60,90"`
	WebhookURL        string `json:"webhookUrl,omitempty" maxLength:"2000"`
	// WebhookSecret is write-only: set it to replace the stored one.
	WebhookSecret *string `json:"webhookSecret,omitempty" maxLength:"200"`
	EmailOn       bool    `json:"emailOn"`
}

var (
	mCountTaken      = msgs.M("milestone.countTaken")
	mNameRequired    = msgs.M("milestone.nameRequired")
	mDescriptionLong = msgs.M("milestone.descriptionLong")
	mFileOrURL       = msgs.M("milestone.fileOrURL")
	mFileUnknown     = msgs.M("milestone.fileUnknown")
	mURL             = msgs.M("milestone.url")
	mWebhookURL      = msgs.M("milestone.webhookURL")
	mWebhookHTTPS    = msgs.M("milestone.webhookHTTPS")
	mCouponStripe    = msgs.M("milestone.couponNeedsStripe")
	mCouponEarn      = msgs.M("milestone.couponNeedsEarn")
	mCouponEarnAlone = msgs.M("milestone.couponNeedsEarnAlone")
	mOffer           = msgs.M("milestone.offer")
	mImageUnknown    = msgs.M("milestone.imageUnknown")
	mLastMilestone   = msgs.M("milestone.programOnLast")
	mCodesNone       = msgs.M("codes.none")
	mCodesTooMany    = msgs.M("codes.tooMany")
	mCodesLong       = msgs.M("codes.long")
	mCodesNotCodes   = msgs.M("codes.notCodes")
)

// Milestones is Referrals › Milestones: the milestones with their rewards,
// health and emails, the files and the "To fulfill" queue.
func (s *Service) Milestones(ctx context.Context, ws *store.Workspace, programID uuid.UUID) (ReferralMilestoneList, error) {
	cur, err := s.program(ctx, ws, programID)
	if err != nil {
		return ReferralMilestoneList{}, err
	}
	p := api.PrincipalFrom(ctx)
	out := ReferralMilestoneList{ProgramID: programID.String(), ListName: cur.ListName, Items: []ReferralMilestoneView{}, Files: []ReferralFile{},
		ToFulfill: []ReferralFulfillItem{}, Helpers: s.helperSets(ctx, ws.ID), ShowAddresses: p.Can(authz.SubscribersView)}
	if cur.ReferralProgram.SenderID != nil {
		if sd, err := s.Q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: ws.ID, ID: *cur.ReferralProgram.SenderID}); err == nil {
			out.Sender = sd.Name + " <" + sd.Email + ">"
		}
	}
	files, err := s.Q.ReferralFiles(ctx, programID)
	if err != nil {
		return out, err
	}
	for _, f := range files {
		out.Files = append(out.Files, ReferralFile{ID: f.ID.String(), Name: f.FileName, ContentType: f.ContentType, Size: f.SizeBytes,
			Downloads: f.Downloads, UploadedBy: f.UploadedByName, UploadedAt: f.UploadedAt})
	}
	if out.Items, err = s.milestoneViews(ctx, ws, programID); err != nil {
		return out, err
	}
	out.Stripe = s.stripe(ctx, ws)
	todo, err := s.Q.ReferralRewardsToFulfill(ctx, programID)
	if err != nil {
		return out, err
	}
	for _, r := range todo {
		it := ReferralFulfillItem{RewardID: r.ID.String(), ContactID: r.ContactID.String(), Referrer: displayName(r.FirstName, r.LastName, r.Email),
			Email: r.Email, Milestone: int(r.MilestoneCount), RewardName: r.RewardName, ReachedAt: r.ReachedAt}
		if out.ShowAddresses && r.HasAddress {
			it.Address = s.openAddress(r.ID, r.Address)
		}
		if !out.ShowAddresses {
			it.Email = ""
		}
		out.ToFulfill = append(out.ToFulfill, it)
	}
	return out, nil
}

func (s *Service) stripe(ctx context.Context, ws *store.Workspace) ReferralStripe {
	out := ReferralStripe{Offers: []ReferralCouponOffer{}}
	st := s.couponStatus(ctx, ws)
	out.Connected, out.Account = st.Connected, st.Account
	if !st.Connected || s.Coupons == nil || s.Coupons() == nil {
		return out
	}
	offers, err := s.Coupons().Offers(ctx, ws)
	if err != nil {
		return out
	}
	for _, o := range offers {
		out.Offers = append(out.Offers, ReferralCouponOffer{ID: o.ID, Name: o.Name, Discount: o.Discount, AppliesTo: o.AppliesTo, Issued: o.Issued, Redeemed: o.Redeemed})
	}
	return out
}

func (s *Service) milestoneViews(ctx context.Context, ws *store.Workspace, programID uuid.UUID) ([]ReferralMilestoneView, error) {
	ms, err := s.Q.ReferralMilestones(ctx, programID)
	if err != nil {
		return nil, err
	}
	stats, err := s.Q.ReferralMilestoneStats(ctx, programID)
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]store.ReferralMilestoneStatsRow{}
	for _, st := range stats {
		if st.MilestoneID != nil {
			byID[*st.MilestoneID] = st
		}
	}
	stripe := s.couponStatus(ctx, ws)
	out := make([]ReferralMilestoneView, 0, len(ms))
	for _, m := range ms {
		v, err := s.milestoneView(ctx, ws, m, byID[m.ID], stripe)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Service) milestoneView(ctx context.Context, ws *store.Workspace, m store.ReferralMilestone, st store.ReferralMilestoneStatsRow, stripe CouponStatus) (ReferralMilestoneView, error) {
	v := ReferralMilestoneView{
		ID: m.ID.String(), Count: int(m.Count), RewardName: m.RewardName, RewardDescription: m.RewardDescription, Type: m.RewardType,
		DownloadURL: m.DownloadUrl, CodesAlertBelow: int(m.CodesAlertBelow), AskAddress: m.AskAddress, CouponOfferID: m.CouponOfferID,
		CouponOfferName: m.CouponOfferName, CouponValidDays: int(m.CouponValidDays), WebhookURL: m.WebhookUrl, WebhookSecretSet: len(m.WebhookSecret) > 0,
		WebhookCalledAt: m.WebhookCalledAt,
		Stats: ReferralMilestoneStats{Reached: st.Reached, Sent: st.Sent, Used: st.Used, Waiting: st.Waiting, Failed: st.Failed,
			WaitingCodes: st.WaitingCodes, ToFulfill: st.ToFulfill},
		Email:  ReferralMilestoneEmail{On: m.EmailOn, Subject: m.EmailSubject, SavedAt: m.EmailSavedAt},
		Health: ReferralMilestoneHealth{Status: "ok", Kind: "healthy"},
	}
	if m.WebhookStatus != nil {
		code := int(*m.WebhookStatus)
		v.WebhookStatus = &code
	}
	if m.EmailID != nil {
		v.Email.EmailID = m.EmailID.String()
		if n, err := s.Q.ReferralEmailVersions(ctx, *m.EmailID); err == nil {
			v.Email.Versions = int64(len(n))
		}
	}
	if m.ImageID != nil {
		if a, err := s.Q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: *m.ImageID}); err == nil {
			v.Image = &ReferralImage{ID: a.ID.String(), URL: media.URL(a.ID, a.FileName), Name: a.FileName, Width: int(a.Width), Height: int(a.Height), Size: int64(a.SizeBytes)}
		}
	}
	if m.FileID != nil {
		if f, err := s.Q.ReferralFileGet(ctx, *m.FileID); err == nil && f.RemovedAt == nil {
			v.File = &ReferralFile{ID: f.ID.String(), Name: f.FileName, ContentType: f.ContentType, Size: f.SizeBytes, Downloads: f.Downloads,
				UploadedBy: f.UploadedByName, UploadedAt: f.UploadedAt}
		}
	}
	switch m.RewardType {
	case RewardCodes:
		left, err := s.Q.ReferralCodesLeft(ctx, m.ID)
		if err != nil {
			return v, err
		}
		up, err := s.Q.ReferralCodesUploaded(ctx, m.ID)
		if err != nil {
			return v, err
		}
		v.CodesLeft, v.CodesUploaded = left, up
		v.Health.CodesLeft = left
		pace := s.codesPace(ctx, m.ID)
		if pace > 0 && left > 0 {
			days := math.Round(float64(left)/pace*10) / 10
			v.Health.RunOutDays = &days
		}
		switch {
		case left == 0:
			v.Health.Status, v.Health.Kind, v.Health.Waiting = "bad", "codes_out", st.WaitingCodes
		case left < int64(m.CodesAlertBelow) || (v.Health.RunOutDays != nil && *v.Health.RunOutDays < 14):
			v.Health.Status, v.Health.Kind = "warn", "codes_low"
		}
	case RewardWebhook:
		if m.WebhookFailures > 0 && (st.Waiting > 0 || st.Failed > 0) {
			v.Health = ReferralMilestoneHealth{Status: "bad", Kind: "webhook_failing", Waiting: st.Waiting, Failures: int(m.WebhookFailures)}
			if m.WebhookStatus != nil {
				v.Health.LastStatus = int(*m.WebhookStatus)
			}
		}
	case RewardCoupon:
		if !stripe.Connected {
			v.Health = ReferralMilestoneHealth{Status: "warn", Kind: "stripe_missing", Waiting: st.Waiting}
		}
	case RewardDownload:
		if v.File == nil && strings.TrimSpace(m.DownloadUrl) == "" {
			v.Health = ReferralMilestoneHealth{Status: "bad", Kind: "file_missing"}
		}
	}
	return v, nil
}

// codesPace is how many rewards a milestone gave per day over the last two
// weeks.
func (s *Service) codesPace(ctx context.Context, milestoneID uuid.UUID) float64 {
	n, err := s.Q.ReferralMilestoneReachedSince(ctx, store.ReferralMilestoneReachedSinceParams{MilestoneID: &milestoneID, Since: s.now().Add(-14 * 24 * time.Hour)})
	if err != nil || n == 0 {
		return 0
	}
	return float64(n) / 14
}

// checkMilestone validates a milestone's input.
func (s *Service) checkMilestone(ctx context.Context, ws *store.Workspace, programID uuid.UUID, except uuid.UUID, in *ReferralMilestoneInput) error {
	l := i18n.LangFrom(ctx)
	in.RewardName = strings.TrimSpace(in.RewardName)
	in.RewardDescription = strings.TrimSpace(in.RewardDescription)
	if in.RewardName == "" {
		return api.Invalid("body.rewardName", mNameRequired.Text(l, nil))
	}
	if utf8.RuneCountInString(in.RewardDescription) > 120 {
		return api.Invalid("body.rewardDescription", mDescriptionLong.Text(l, nil))
	}
	ms, err := s.Q.ReferralMilestones(ctx, programID)
	if err != nil {
		return err
	}
	for _, m := range ms {
		if m.ID != except && int(m.Count) == in.Count {
			return api.Invalid("body.count", mCountTaken.Text(l, i18n.Args{"count": in.Count}))
		}
	}
	if in.ImageID != "" {
		id, err := uuid.Parse(in.ImageID)
		if err != nil {
			return api.Invalid("body.imageId", mImageUnknown.Text(l, nil))
		}
		if _, err := s.Q.GetMediaAsset(ctx, store.GetMediaAssetParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return api.Invalid("body.imageId", mImageUnknown.Text(l, nil))
		}
	}
	switch in.Type {
	case RewardDownload:
		if in.FileID == "" && strings.TrimSpace(in.DownloadURL) == "" {
			return api.Invalid("body.fileId", mFileOrURL.Text(l, nil))
		}
		if in.FileID != "" {
			id, err := uuid.Parse(in.FileID)
			if err != nil {
				return api.Invalid("body.fileId", mFileUnknown.Text(l, nil))
			}
			f, err := s.Q.ReferralFileGet(ctx, id)
			if err != nil || f.ProgramID != programID || f.RemovedAt != nil {
				return api.Invalid("body.fileId", mFileUnknown.Text(l, nil))
			}
			in.DownloadURL = ""
		} else if u := strings.TrimSpace(in.DownloadURL); !strings.HasPrefix(u, "https://") || strings.ContainsAny(u, " \t\n") {
			return api.Invalid("body.downloadUrl", mURL.Text(l, nil))
		}
	case RewardCoupon:
		if !s.couponStatus(ctx, ws).Connected {
			return api.Err(http.StatusConflict, "stripe_not_connected", mCouponStripe.Text(l, nil))
		}
		if p := api.PrincipalFrom(ctx); !p.Can(authz.EarnEdit) {
			helpers := GrowHelpers(ctx, s.Q, ws.ID, authz.EarnEdit)
			msg := mCouponEarnAlone.Text(l, nil)
			if len(helpers) > 0 {
				msg = mCouponEarn.Text(l, i18n.Args{"helpers": i18n.OrList(l.Locale, helpers)})
			}
			return api.Err(http.StatusForbidden, api.CodePermissionDenied, msg).WithMeta("permission", string(authz.EarnEdit), "helpers", helpers)
		}
		if strings.TrimSpace(in.CouponOfferID) == "" {
			return api.Invalid("body.couponOfferId", mOffer.Text(l, nil))
		}
		if in.CouponValidDays == 0 {
			in.CouponValidDays = 60
		}
	case RewardWebhook:
		u := strings.TrimSpace(in.WebhookURL)
		if !strings.HasPrefix(u, "https://") && (!s.allowPrivate() || !strings.HasPrefix(u, "http://")) {
			return api.Invalid("body.webhookUrl", mWebhookHTTPS.Text(l, nil))
		}
		if _, err := (netguard.Policy{AllowPrivate: s.allowPrivate()}).ValidateURL(u); err != nil {
			return api.Invalid("body.webhookUrl", mWebhookURL.Text(l, i18n.Args{"detail": netguard.Describe(ctx, err)}))
		}
		in.WebhookURL = u
	}
	if in.CodesAlertBelow == 0 && in.Type == RewardCodes {
		in.CodesAlertBelow = 20
	}
	if in.CouponValidDays == 0 {
		in.CouponValidDays = 60
	}
	return nil
}

func (s *Service) allowPrivate() bool { return s.AllowPrivate != nil && s.AllowPrivate() }

func optUUID(raw string) *uuid.UUID {
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}

func milestoneSnapshot(m store.ReferralMilestone) map[string]any {
	return map[string]any{
		"count": m.Count, "reward": m.RewardName, "description": m.RewardDescription, "type": m.RewardType, "file": m.FileID,
		"download_url": m.DownloadUrl, "codes_alert_below": m.CodesAlertBelow, "ask_address": m.AskAddress, "offer": m.CouponOfferName,
		"coupon_valid_days": m.CouponValidDays, "webhook_url": m.WebhookUrl, "email": m.EmailOn, "image": m.ImageID,
	}
}

// CreateMilestone adds a milestone with its reward and a milestone email
// that starts from the default (the Reward block, then the Referral block).
func (s *Service) CreateMilestone(ctx context.Context, ws *store.Workspace, programID uuid.UUID, in ReferralMilestoneInput) (ReferralMilestoneView, error) {
	cur, err := s.program(ctx, ws, programID)
	if err != nil {
		return ReferralMilestoneView{}, err
	}
	if err := s.checkMilestone(ctx, ws, programID, uuid.Nil, &in); err != nil {
		return ReferralMilestoneView{}, err
	}
	var secret []byte
	id := uuid.Must(uuid.NewV7())
	if in.Type == RewardWebhook && in.WebhookSecret != nil && strings.TrimSpace(*in.WebhookSecret) != "" {
		if secret, err = s.Box.SealString(strings.TrimSpace(*in.WebhookSecret), secretAD(id)); err != nil {
			return ReferralMilestoneView{}, err
		}
	}
	offerName := s.offerName(ctx, ws, in.CouponOfferID)
	subject, preheader, mjmlSrc := defaultMilestoneEmail(cur.ListName, in.RewardName, in.Type)
	compiled := mjml.Compile(mjmlSrc)
	now := s.now()
	by := actorUser(ctx)
	emailID := uuid.Must(uuid.NewV7())
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralEmailCreate(ctx, store.ReferralEmailCreateParams{ID: emailID, WorkspaceID: ws.ID, Name: emailName(in.Count, in.RewardName), CreatedBy: by, Now: now}); err != nil {
			return err
		}
		if _, err := q.ReferralEmailVersionInsert(ctx, store.ReferralEmailVersionInsertParams{EmailID: emailID, Subject: subject, Preheader: preheader,
			Mjml: mjmlSrc, Html: compiled.HTML, PlainText: "", CreatedBy: by, Now: now}); err != nil {
			return err
		}
		if err := s.setEmailSender(ctx, tx, cur.ReferralProgram, emailID); err != nil {
			return err
		}
		if err := q.ReferralMilestoneInsert(ctx, store.ReferralMilestoneInsertParams{ID: id, WorkspaceID: ws.ID, ProgramID: programID, Count: int32(in.Count),
			RewardName: in.RewardName, RewardDescription: in.RewardDescription, ImageID: optUUID(in.ImageID), RewardType: in.Type, FileID: optUUID(in.FileID),
			DownloadUrl: in.DownloadURL, CodesAlertBelow: int32(in.CodesAlertBelow), AskAddress: in.AskAddress, CouponOfferID: in.CouponOfferID,
			CouponOfferName: offerName, CouponValidDays: int32(in.CouponValidDays), WebhookUrl: in.WebhookURL, WebhookSecret: secret,
			EmailOn: in.EmailOn, EmailSubject: subject, EmailPreheader: preheader, EmailMjml: mjmlSrc, EmailHtml: compiled.HTML, CreatedBy: by, Now: now}); err != nil {
			return err
		}
		if err := q.ReferralMilestoneSetEmail(ctx, store.ReferralMilestoneSetEmailParams{ID: id, EmailID: &emailID, Now: now}); err != nil {
			return err
		}
		m, err := q.ReferralMilestoneByID(ctx, id)
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.milestone_created", TargetType: "referral_milestone",
			TargetID: id.String(), TargetName: in.RewardName, After: milestoneSnapshot(m), Metadata: map[string]any{"related": programID.String()}})
	})
	if err != nil {
		return ReferralMilestoneView{}, err
	}
	return s.MilestoneView(ctx, ws, id)
}

// setEmailSender makes a milestone email come from the program's sender.
func (s *Service) setEmailSender(ctx context.Context, tx pgx.Tx, p store.ReferralProgram, emailID uuid.UUID) error {
	_, err := tx.Exec(ctx, `UPDATE automation_emails SET sender_id = $2, reply_to = $3 WHERE id = $1`, emailID, p.SenderID, p.ReplyTo)
	return err
}

func emailName(count int, reward string) string {
	return "Milestone email · " + formatInt(int64(count)) + " · " + reward
}

func (s *Service) offerName(ctx context.Context, ws *store.Workspace, offerID string) string {
	if offerID == "" || s.Coupons == nil || s.Coupons() == nil {
		return ""
	}
	offers, err := s.Coupons().Offers(ctx, ws)
	if err != nil {
		return ""
	}
	for _, o := range offers {
		if o.ID == offerID {
			return o.Name
		}
	}
	return ""
}

func secretAD(milestoneID uuid.UUID) string {
	return "referral_milestone:" + milestoneID.String() + ":webhook_secret"
}

// MilestoneView is one milestone.
func (s *Service) MilestoneView(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ReferralMilestoneView, error) {
	m, err := s.Q.ReferralMilestoneGet(ctx, store.ReferralMilestoneGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ReferralMilestoneView{}, api.MapError(err, "Milestone")
	}
	stats, err := s.Q.ReferralMilestoneStats(ctx, m.ProgramID)
	if err != nil {
		return ReferralMilestoneView{}, err
	}
	var st store.ReferralMilestoneStatsRow
	for _, r := range stats {
		if r.MilestoneID != nil && *r.MilestoneID == id {
			st = r
		}
	}
	return s.milestoneView(ctx, ws, m, st, s.couponStatus(ctx, ws))
}

// UpdateMilestone changes a milestone. Rewards already given stay as they
// were; changes apply to rewards given from now on.
func (s *Service) UpdateMilestone(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ReferralMilestoneInput) (ReferralMilestoneView, error) {
	m, err := s.Q.ReferralMilestoneGet(ctx, store.ReferralMilestoneGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ReferralMilestoneView{}, api.MapError(err, "Milestone")
	}
	// A coupon reward that stays a coupon with the same offer doesn't need
	// Stripe again to change its name or email.
	keepCoupon := m.RewardType == RewardCoupon && in.Type == RewardCoupon && in.CouponOfferID == m.CouponOfferID
	if keepCoupon {
		in.Type = "keep-coupon"
	}
	if err := s.checkMilestone(ctx, ws, m.ProgramID, id, &in); err != nil {
		return ReferralMilestoneView{}, err
	}
	offerName := m.CouponOfferName
	if keepCoupon {
		in.Type = RewardCoupon
	} else if in.Type == RewardCoupon {
		offerName = s.offerName(ctx, ws, in.CouponOfferID)
	}
	if in.CouponValidDays == 0 {
		in.CouponValidDays = int(m.CouponValidDays)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralMilestoneUpdate(ctx, store.ReferralMilestoneUpdateParams{ID: id, Count: int32(in.Count), RewardName: in.RewardName,
			RewardDescription: in.RewardDescription, ImageID: optUUID(in.ImageID), RewardType: in.Type, FileID: optUUID(in.FileID), DownloadUrl: in.DownloadURL,
			CodesAlertBelow: int32(in.CodesAlertBelow), AskAddress: in.AskAddress, CouponOfferID: in.CouponOfferID, CouponOfferName: offerName,
			CouponValidDays: int32(in.CouponValidDays), WebhookUrl: in.WebhookURL, EmailOn: in.EmailOn, Now: s.now()}); err != nil {
			return err
		}
		secretChanged := false
		if in.WebhookSecret != nil {
			var sealed []byte
			if v := strings.TrimSpace(*in.WebhookSecret); v != "" {
				var err error
				if sealed, err = s.Box.SealString(v, secretAD(id)); err != nil {
					return err
				}
			}
			if err := q.ReferralMilestoneSetSecret(ctx, store.ReferralMilestoneSetSecretParams{ID: id, WebhookSecret: sealed, Now: s.now()}); err != nil {
				return err
			}
			secretChanged = true
		}
		if m.EmailID != nil {
			if err := q.ReferralEmailRename(ctx, store.ReferralEmailRenameParams{ID: *m.EmailID, Name: emailName(in.Count, in.RewardName), Now: s.now()}); err != nil {
				return err
			}
		}
		after, err := q.ReferralMilestoneByID(ctx, id)
		if err != nil {
			return err
		}
		meta := map[string]any{"related": m.ProgramID.String()}
		if secretChanged {
			meta["what"] = "webhook signing secret replaced"
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.milestone_updated", TargetType: "referral_milestone",
			TargetID: id.String(), TargetName: in.RewardName, Before: milestoneSnapshot(m), After: milestoneSnapshot(after), Metadata: meta})
	})
	if err != nil {
		return ReferralMilestoneView{}, err
	}
	return s.MilestoneView(ctx, ws, id)
}

// DeleteMilestone removes a milestone. Rewards already sent stay with their
// referrers. A program that is on keeps at least one milestone.
func (s *Service) DeleteMilestone(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	m, err := s.Q.ReferralMilestoneGet(ctx, store.ReferralMilestoneGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return api.MapError(err, "Milestone")
	}
	prog, err := s.Q.ReferralProgramByID(ctx, m.ProgramID)
	if err != nil {
		return err
	}
	if prog.Status == "on" {
		ms, err := s.Q.ReferralMilestones(ctx, m.ProgramID)
		if err != nil {
			return err
		}
		if len(ms) <= 1 {
			return api.Err(http.StatusConflict, "last_milestone", s.text(ctx, mLastMilestone, nil))
		}
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ReferralMilestoneDelete(ctx, store.ReferralMilestoneDeleteParams{ID: id, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.milestone_deleted", TargetType: "referral_milestone",
			TargetID: id.String(), TargetName: m.RewardName, Before: milestoneSnapshot(m), Metadata: map[string]any{"related": m.ProgramID.String()}})
	})
}

// --- Codes -------------------------------------------------------------------

// ReferralCodesInput uploads codes: CSV or text, one per line.
type ReferralCodesInput struct {
	Text string `json:"text" maxLength:"5000000" doc:"The codes, one per line; for CSV the first column"`
}

// ReferralCodesResult is what an upload did.
type ReferralCodesResult struct {
	Added      int64 `json:"added"`
	Duplicates int64 `json:"duplicates"`
	Left       int64 `json:"left"`
	// Released: rewards that were waiting for codes and can go out now.
	Released int64 `json:"released"`
}

func (s *Service) codeHash(milestone uuid.UUID, code string) []byte {
	m := hmac.New(sha256.New, []byte("reputecast-referral-codes:"+milestone.String()))
	m.Write([]byte(code))
	return m.Sum(nil)
}

// parseCodes reads codes from text or CSV: one per line, the first column
// of a CSV, a header row named "code" skipped.
func parseCodes(text string) []string {
	var out []string
	seen := map[string]bool{}
	for i, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if c := strings.IndexAny(line, ",;\t"); c >= 0 {
			line = strings.TrimSpace(line[:c])
		}
		line = strings.Trim(line, `"`)
		if line == "" || (i == 0 && strings.EqualFold(line, "code")) || seen[line] {
			continue
		}
		seen[line] = true
		out = append(out, line)
	}
	return out
}

// UploadCodes adds codes to a code reward. Codes are sealed and only their
// count goes in the audit log. Rewards waiting for codes go out at once.
func (s *Service) UploadCodes(ctx context.Context, ws *store.Workspace, milestoneID uuid.UUID, in ReferralCodesInput) (ReferralCodesResult, error) {
	m, err := s.Q.ReferralMilestoneGet(ctx, store.ReferralMilestoneGetParams{WorkspaceID: ws.ID, ID: milestoneID})
	if err != nil {
		return ReferralCodesResult{}, api.MapError(err, "Milestone")
	}
	if m.RewardType != RewardCodes {
		return ReferralCodesResult{}, api.Err(http.StatusConflict, "not_codes", s.text(ctx, mCodesNotCodes, nil))
	}
	codes := parseCodes(in.Text)
	if len(codes) == 0 {
		return ReferralCodesResult{}, api.Invalid("body.text", s.text(ctx, mCodesNone, nil))
	}
	if len(codes) > 100000 {
		return ReferralCodesResult{}, api.Invalid("body.text", s.text(ctx, mCodesTooMany, nil))
	}
	for _, c := range codes {
		if len(c) > 200 {
			return ReferralCodesResult{}, api.Invalid("body.text", s.text(ctx, mCodesLong, nil))
		}
	}
	out := ReferralCodesResult{}
	batch := uuid.Must(uuid.NewV7())
	now := s.now()
	by := actorUser(ctx)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, c := range codes {
			sealed, err := s.Box.SealString(c, codeAD(milestoneID))
			if err != nil {
				return err
			}
			if _, err := q.ReferralCodeInsert(ctx, store.ReferralCodeInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, MilestoneID: milestoneID,
				Code: sealed, CodeHash: s.codeHash(milestoneID, c), BatchID: batch, UploadedBy: by, Now: now}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					out.Duplicates++
					continue
				}
				return err
			}
			out.Added++
		}
		if err := q.ReferralMilestoneCodesAlerted(ctx, store.ReferralMilestoneCodesAlertedParams{ID: milestoneID, CodesAlerted: ""}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.codes_uploaded", TargetType: "referral_milestone",
			TargetID: milestoneID.String(), TargetName: m.RewardName,
			Metadata: map[string]any{"summary": formatInt(out.Added) + " codes uploaded", "added": out.Added, "duplicates": out.Duplicates}})
	})
	if err != nil {
		return out, err
	}
	released, err := s.releaseWaitingCodes(ctx, milestoneID)
	if err != nil {
		s.log().WarnContext(ctx, "referral codes: release waiting rewards", "milestone", milestoneID, "err", err)
	}
	out.Released = released
	out.Left, _ = s.Q.ReferralCodesLeft(ctx, milestoneID)
	return out, nil
}

func codeAD(milestone uuid.UUID) string { return "referral_milestone:" + milestone.String() + ":code" }

// RemoveCodes removes a code reward's codes nobody has received yet (the
// count only goes in the audit log).
func (s *Service) RemoveCodes(ctx context.Context, ws *store.Workspace, milestoneID uuid.UUID) (int64, error) {
	m, err := s.Q.ReferralMilestoneGet(ctx, store.ReferralMilestoneGetParams{WorkspaceID: ws.ID, ID: milestoneID})
	if err != nil {
		return 0, api.MapError(err, "Milestone")
	}
	var n int64
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		if n, err = q.ReferralCodesRemoveFree(ctx, store.ReferralCodesRemoveFreeParams{MilestoneID: milestoneID, Now: s.now()}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "referral.codes_removed", TargetType: "referral_milestone",
			TargetID: milestoneID.String(), TargetName: m.RewardName, Metadata: map[string]any{"summary": formatInt(n) + " unused codes removed", "removed": n}})
	})
	return n, err
}

// --- Defaults ------------------------------------------------------------------

// defaultMilestoneEmail is a new milestone's email: a heading, a thank-you
// line, the Reward block, then the Referral block with the progress to the
// next milestone, and the footer.
func defaultMilestoneEmail(list, reward, rewardType string) (subject, preheader, src string) {
	subject = "You unlocked " + articled(reward)
	preheader = `Thanks for sharing the ` + list + `, {{ first_name | default: "there" }}.`
	button := "Get your reward"
	switch rewardType {
	case RewardDownload:
		button = "Download " + strings.TrimPrefix(strings.TrimPrefix(reward, "The "), "the ")
	case RewardCoupon:
		button = "Claim your reward"
	case RewardCodes:
		button = "Use your code"
	}
	reward = strings.ReplaceAll(reward, "<", "&lt;")
	src = `<mjml>
  <mj-head>
    <mj-attributes>
      <mj-all font-family="IBM Plex Sans, Arial, sans-serif" />
      <mj-text font-size="15px" line-height="1.55" color="#0B1220" />
    </mj-attributes>
  </mj-head>
  <mj-body background-color="#F3F4F6" width="600px">
    <mj-section background-color="#FFFFFF" padding="22px 24px 8px">
      <mj-column>
        <mj-text font-size="26px" line-height="1.2" font-weight="700" padding="0 0 10px">You unlocked ` + articled(reward) + `</mj-text>
        <mj-text padding="0">Thanks for sharing the ` + list + `, {{ first_name | default: "there" }}. Your friends now read it because of you, so this one is yours.</mj-text>
      </mj-column>
    </mj-section>
    <mj-raw><!-- rc:reward {"button":"` + jsonText(button) + `","image":true,"description":true,"expiry":true,"hub":true} --></mj-raw>
    <mj-raw><!-- rc:referral {"layout":"progress","share":["email","x","linkedin","whatsapp"],"button":"Share your link","progress":true} --></mj-raw>
    <mj-section background-color="#F3F4F6" padding="18px 24px">
      <mj-column>
        <mj-text align="center" font-size="12px" color="#5B6472" line-height="1.6">You get this because you're in the ` + list + ` referral program.<br/><a href="{{ unsubscribe_url }}" style="color:#0F766E">Unsubscribe</a> · <a href="{{ preferences_url }}" style="color:#0F766E">Email preferences</a></mj-text>
      </mj-column>
    </mj-section>
  </mj-body>
</mjml>`
	return subject, preheader, src
}

// articled is "the Acme Field Guide" for "The Acme Field Guide", and a
// reward name as written otherwise.
func articled(reward string) string {
	if strings.HasPrefix(reward, "The ") {
		return "the " + reward[4:]
	}
	return reward
}

func jsonText(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "--", `--`, "<", `<`, ">", `>`)
	return r.Replace(s)
}
