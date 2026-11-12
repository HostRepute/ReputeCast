package coupons

import (
	"context"
	"embed"
	"net/http"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Problems, checks, notifications and token labels a member reads follow
// their language (messages/<locale>.json); audit entries, stored batch
// errors and message records keep their English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "coupons")

var (
	mPoolNotFound      = msgs.M("pool.notFound")
	mPoolNameMissing   = msgs.M("pool.nameMissing")
	mPoolNameLong      = msgs.M("pool.nameLong")
	mPoolSlugFormat    = msgs.M("pool.slugFormat")
	mPoolSlugTaken     = msgs.M("pool.slugTaken")
	mPoolPrefixFormat  = msgs.M("pool.prefixFormat")
	mPoolLowAt         = msgs.M("pool.lowAt")
	mPoolApplyURL      = msgs.M("pool.applyURL")
	mPoolStoreMissing  = msgs.M("pool.storeMissing")
	mPoolStoreUnknown  = msgs.M("pool.storeUnknown")
	mPoolStorePlatform = msgs.M("pool.storePlatform")
	mPoolUploadStore   = msgs.M("pool.uploadStore")
	mPoolKindMissing   = msgs.M("pool.kindMissing")
	mPoolPercent       = msgs.M("pool.percent")
	mPoolAmount        = msgs.M("pool.amount")
	mPoolMinimum       = msgs.M("pool.minimum")
	mPoolCollections   = msgs.M("pool.collections")
	mPoolEndsDays      = msgs.M("pool.endsDays")
	mPoolLocked        = msgs.M("pool.settingsLocked")
	mPoolArchived      = msgs.M("pool.archived")
	mPoolArchiveLive   = msgs.M("pool.archiveLive")

	mCodesNone       = msgs.M("codes.none")
	mCodesTooMany    = msgs.M("codes.tooMany")
	mCodesStorePool  = msgs.M("codes.storePool")
	mCodesUploadPool = msgs.M("codes.uploadPool")
	mCodesRunning    = msgs.M("codes.running")
	mCodesCount      = msgs.M("codes.count")
	mCodesStoreGone  = msgs.M("codes.storeGone")

	mCheckUnknownTitle  = msgs.M("check.unknownTitle")
	mCheckUnknownDetail = msgs.M("check.unknownDetail")
	mCheckArchivedTitle = msgs.M("check.archivedTitle")
	mCheckArchivedText  = msgs.M("check.archivedDetail")
	mCheckTag           = msgs.M("check.tag")
	mCheckOpenPools     = msgs.M("check.openPools")

	mReadyLabel          = msgs.M("readiness.label")
	mReadyShort          = msgs.M("readiness.short")
	mReadyEnough         = msgs.M("readiness.enough")
	mReadyNotEnough      = msgs.M("readiness.notEnough")
	mReadyNotEnoughFor   = msgs.M("readiness.notEnoughDetail")
	mReadyCreating       = msgs.M("readiness.creating")
	mReadyStoreShort     = msgs.M("readiness.storeShort")
	mReadyCreatingBy     = msgs.M("readiness.creatingBy")
	mReadyCreatingRefill = msgs.M("readiness.creatingRefill")
	mReadyArchived       = msgs.M("readiness.archived")
	mReadyArchivedFor    = msgs.M("readiness.archivedDetail")
	mReadyDisconnected   = msgs.M("readiness.disconnected")
	mReadyDisconnFor     = msgs.M("readiness.disconnectedDetail")
	mReadyAddCodes       = msgs.M("readiness.addCodes")
	mReadyCreateCodes    = msgs.M("readiness.createCodes")
	mReadyOpenEmail      = msgs.M("readiness.openEmail")
	mReadyOpenStore      = msgs.M("readiness.openStore")

	mPublishLow      = msgs.M("publish.low")
	mPublishArchived = msgs.M("publish.archived")
	mPublishUnknown  = msgs.M("publish.unknown")

	mNotifyLowTitle = msgs.M("notify.lowTitle")
	mNotifyLowBody  = msgs.M("notify.lowBody")
	mNotifyOutTitle = msgs.M("notify.outTitle")
	mNotifyOutBody  = msgs.M("notify.outBody")

	mTokenCode = msgs.M("token.code")
	mTokenURL  = msgs.M("token.url")
)

// text is a message in the request's language.
func text(ctx context.Context, m i18n.Msg, args i18n.Args) string {
	return m.Text(i18n.LangFrom(ctx), args)
}

// invalid is a 422 for one field, in the request's language.
func invalid(ctx context.Context, field string, m i18n.Msg, args i18n.Args) error {
	return api.Invalid(field, text(ctx, m, args))
}

// conflict is a 409 with a code, in the request's language.
func conflict(ctx context.Context, code string, m i18n.Msg, args i18n.Args) *api.Problem {
	return api.Err(http.StatusConflict, code, text(ctx, m, args))
}

// notFound is the 404 of a pool not in the workspace.
func notFound(ctx context.Context) error {
	return api.Err(http.StatusNotFound, api.CodeNotFound, text(ctx, mPoolNotFound, nil))
}
