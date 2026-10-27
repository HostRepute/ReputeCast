package ai

import (
	"context"
	"embed"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/applang"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/notify"
)

// languageIn is a content language's name in the reader's language: the
// catalog's English name in English, CLDR's elsewhere ("Tiếng Tây Ban
// Nha"), as Send Readiness names them. Prompts keep the English name.
func languageIn(l i18n.Lang, code string) string {
	switch l.Locale {
	case i18n.English, i18n.Pseudo, "":
		return languages.NameOf(code)
	}
	tag, err := language.Parse(languages.Base(code))
	if err != nil {
		return languages.NameOf(code)
	}
	if n := display.Tags(l.Locale.Tag()).Name(tag); n != "" {
		return n
	}
	return languages.NameOf(code)
}

// Problems a member reads follow their app language (messages/<locale>.json);
// notifications are written in each recipient's language when they are
// created, as placement tests' are; a version's name follows the member who
// saved it; audit entries and what is sent to a model keep their English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "ai")

var (
	mNotConnected         = msgs.M("problem.notConnected")
	mNotConnectedAsk      = msgs.M("problem.notConnectedAsk")
	mNotConnectedAlone    = msgs.M("problem.notConnectedAlone")
	mFeatureOff           = msgs.M("problem.featureOff")
	mFeatureOffAlone      = msgs.M("problem.featureOffAlone")
	mCapReached           = msgs.M("problem.capReached")
	mCapReachedAlone      = msgs.M("problem.capReachedAlone")
	mTokenCapReached      = msgs.M("problem.tokenCapReached")
	mKeyRejected          = msgs.M("problem.keyRejected")
	mKeyRejectedAlone     = msgs.M("problem.keyRejectedAlone")
	mProviderErrorTwice   = msgs.M("problem.providerErrorTwice")
	mProviderError        = msgs.M("problem.providerError")
	mRateLimited          = msgs.M("problem.rateLimited")
	mRateLimitedNow       = msgs.M("problem.rateLimitedNow")
	mTimeout              = msgs.M("problem.timeout")
	mUnreachable          = msgs.M("problem.unreachable")
	mBlocked              = msgs.M("problem.blocked")
	mBadRequest           = msgs.M("problem.badRequest")
	mUnreadable           = msgs.M("problem.unreadable")
	mNoImageInput         = msgs.M("problem.noImageInput")
	mNoImageInputAlone    = msgs.M("problem.noImageInputAlone")
	mDemoModel            = msgs.M("problem.demoModel")
	mDemoLanguage         = msgs.M("problem.demoLanguage")
	mPrivateAddress       = msgs.M("problem.privateAddress")
	mPrivateHost          = msgs.M("problem.privateHost")
	mBadAddress           = msgs.M("problem.badAddress")
	mKeyNeeded            = msgs.M("problem.keyNeeded")
	mAzureNeeds           = msgs.M("problem.azureNeeds")
	mModelNeeded          = msgs.M("problem.modelNeeded")
	mOneModel             = msgs.M("problem.oneModel")
	mNotConnectedYet      = msgs.M("problem.notConnectedYet")
	mCapRange             = msgs.M("problem.capRange")
	mTokenCapRange        = msgs.M("problem.tokenCapRange")
	mPriceRange           = msgs.M("problem.priceRange")
	mUnknownFeature       = msgs.M("problem.unknownFeature")
	mNothingToRewrite     = msgs.M("problem.nothingToRewrite")
	mTooLong              = msgs.M("problem.tooLong")
	mInstructionNeeded    = msgs.M("problem.instructionNeeded")
	mImageUnreadable      = msgs.M("problem.imageUnreadable")
	mImageMissing         = msgs.M("problem.imageMissing")
	mSuggestionGone       = msgs.M("problem.suggestionGone")
	mNoSourceText         = msgs.M("problem.noSourceText")
	mSameLanguage         = msgs.M("problem.sameLanguage")
	mNotWorkspaceLanguage = msgs.M("problem.notWorkspaceLanguage")
	mSourceChanged        = msgs.M("problem.sourceChanged")
	mNothingAccepted      = msgs.M("problem.nothingAccepted")
	mDescriptionNeeded    = msgs.M("problem.descriptionNeeded")
	mNotSent              = msgs.M("problem.notSent")
	mPickProvider         = msgs.M("problem.pickProvider")
	mCampaignID           = msgs.M("problem.campaignId")
	mNumbersUnmatched     = msgs.M("problem.numbersUnmatched")
	mSuggestionID         = msgs.M("problem.suggestionId")
	mVersionTranslated    = msgs.M("version.translated")

	mImageAddress      = msgs.M("image.address")
	mImageNotInLibrary = msgs.M("image.notInLibrary")
	mImageWhere        = msgs.M("image.where")
	mImageNoAnswer     = msgs.M("image.noAnswer")
	mImageStatus       = msgs.M("image.status")
	mImageTooLarge     = msgs.M("image.tooLarge")
	mImageFormat       = msgs.M("image.format")
	mImageDecode       = msgs.M("image.decode")

	mNewSegment       = msgs.M("activity.newSegment")
	mOnField          = msgs.M("activity.onField")
	mDetailRateLimit  = msgs.M("activity.rateLimited")
	mDetailTimedOut   = msgs.M("activity.timedOut")
	mDetailUnreach    = msgs.M("activity.unreachable")
	mDetailKeyRefused = msgs.M("activity.keyRejected")

	mTestPrompt        = msgs.M("test.prompt")
	mDroppedConditions = msgs.M("segment.dropped")

	mNotifyUsageTitle     = msgs.M("notify.usageTitle")
	mNotifyUsageBody      = msgs.M("notify.usageBody")
	mNotifyCapTitle       = msgs.M("notify.capTitle")
	mNotifyCapBody        = msgs.M("notify.capBody")
	mNotifyTokenUsageBody = msgs.M("notify.tokenUsageBody")
	mNotifyTokenCapBody   = msgs.M("notify.tokenCapBody")
	mNotifyKeyTitle       = msgs.M("notify.keyTitle")
	mNotifyKeyBody        = msgs.M("notify.keyBody")
)

// Error codes of the assistant's problems.
const (
	codeNotConnected  = "ai_not_configured"
	codeFeatureOff    = "ai_feature_off"
	codeCapReached    = "ai_cap_reached"
	codeKeyRejected   = "ai_key_rejected"
	codeProviderError = "ai_provider_error"
	codeRateLimited   = "ai_rate_limited"
	codeNoImageInput  = "ai_no_image_input"
	codeDemoModel     = "ai_demo_model"
	codePrivate       = "ai_private_address"
	codeOneModel      = "ai_one_model"
)

// text writes a message in the request's language.
func text(ctx context.Context, m i18n.Msg, args i18n.Args) string {
	return m.Text(i18n.LangFrom(ctx), args)
}

// problem is an RFC 9457 problem with a message of this package.
func problem(ctx context.Context, status int, code string, m i18n.Msg, args i18n.Args) *api.Problem {
	return api.Err(status, code, text(ctx, m, args))
}

// invalid is a validation problem at a field, in the reader's language.
func invalid(ctx context.Context, location string, m i18n.Msg, args i18n.Args) *api.Problem {
	return api.Invalid(location, text(ctx, m, args))
}

// orList joins names as "Marcus Chen or Linh Nguyen" in the reader's language.
func orList(ctx context.Context, names []string) string {
	return i18n.OrList(i18n.LangFrom(ctx).Locale, names)
}

// withHelpers picks the sentence that names who can help, or the one
// without names.
func withHelpers(ctx context.Context, status int, code string, named, alone i18n.Msg, args i18n.Args, helpers []string) *api.Problem {
	if args == nil {
		args = i18n.Args{}
	}
	if len(helpers) == 0 {
		return problem(ctx, status, code, alone, args)
	}
	args["helpers"] = orList(ctx, helpers)
	p := problem(ctx, status, code, named, args)
	p.Helpers = helpers
	return p
}

// notifyMembers tells the workspace's members who hold perm, each in their
// own language.
func (s *Service) notifyMembers(ctx context.Context, workspaceID uuid.UUID, perm authz.Permission, notification func(l i18n.Lang) notify.Notification) error {
	ids, err := notify.MembersWith(ctx, s.Pool, workspaceID, perm)
	if err != nil {
		return err
	}
	for _, id := range ids {
		n := notification(applang.ForUserID(ctx, s.Q, id))
		if n.WorkspaceID == nil {
			n.WorkspaceID = &workspaceID
		}
		if err := notify.ToUser(ctx, s.Pool, id, n); err != nil {
			return err
		}
	}
	return nil
}

// What the activity log keeps in English, written in each reader's
// language when it is read.
const (
	// newSegmentLabel is what a segment draft is on before the segment has a name.
	newSegmentLabel   = "New segment"
	detailRateLimited = "rate limited"
	detailTimedOut    = "timed out"
	detailUnreachable = "unreachable"
	// detailKeyRejected follows the status: "401 key rejected".
	detailKeyRejected = " key rejected"
)

// onIn is what a request was on, for a reader: a campaign's or segment's
// name stays as it is; a segment without one yet is named in their words,
// and so is the field a rewrite was on ("Weekly Digest #143 subject").
func onIn(l i18n.Lang, feature, kind string, target *uuid.UUID, label string) string {
	if kind == "segment" && target == nil && label == newSegmentLabel {
		return mNewSegment.Text(l, nil)
	}
	if feature == FeatureRewrite && kind == "campaign" {
		for _, field := range []string{"subject", "preheader"} {
			if name, ok := strings.CutSuffix(label, " "+field); ok && name != "" {
				return mOnField.Text(l, i18n.Args{"name": name, "field": field})
			}
		}
	}
	return label
}

// detailIn is why a request failed, for a reader: a provider's own error
// type ("529 overloaded") stays as it is.
func detailIn(l i18n.Lang, detail string) string {
	switch detail {
	case detailRateLimited:
		return mDetailRateLimit.Text(l, nil)
	case detailTimedOut:
		return mDetailTimedOut.Text(l, nil)
	case detailUnreachable:
		return mDetailUnreach.Text(l, nil)
	}
	if status, ok := strings.CutSuffix(detail, detailKeyRejected); ok {
		if _, err := strconv.Atoi(status); err == nil {
			return mDetailKeyRefused.Text(l, i18n.Args{"status": status})
		}
	}
	return detail
}
