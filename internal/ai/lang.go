package ai

import (
	"context"
	"embed"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Problems a member reads follow their app language (messages/<locale>.json);
// notifications are written in English when they are created, as the
// notification center stores them; audit entries keep their English.

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
