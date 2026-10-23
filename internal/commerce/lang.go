package commerce

import (
	"context"
	"embed"
	"time"

	"github.com/google/uuid"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
)

// Problems and notifications a member reads follow their language
// (messages/<locale>.json). Audit entries and stored reasons keep their
// English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "commerce")

var (
	mAddressMissing     = msgs.M("address.missing")
	mAddressNotShopify  = msgs.M("address.notShopify")
	mAddressNotWoo      = msgs.M("address.notWoo")
	mAddressBlocked     = msgs.M("address.blocked")
	mStoreConnected     = msgs.M("store.connected")
	mStoreNotFound      = msgs.M("store.notFound")
	mStoreNotConnected  = msgs.M("store.notConnected")
	mStoreWrongStep     = msgs.M("store.wrongStep")
	mStoreUnavailable   = msgs.M("store.unavailable")
	mStoreRateLimited   = msgs.M("store.rateLimited")
	mKeysClientID       = msgs.M("keys.clientId")
	mKeysSecretFormat   = msgs.M("keys.secretFormat")
	mKeysMissing        = msgs.M("keys.missing")
	mWooKeyFormat       = msgs.M("woo.keyFormat")
	mWooKeyRefused      = msgs.M("woo.keyRefused")
	mWooReadOnly        = msgs.M("woo.readOnly")
	mWooNotConnected    = msgs.M("woo.notConnected")
	mListExists         = msgs.M("list.exists")
	mListUnknown        = msgs.M("list.unknown")
	mDisconnectConfirm  = msgs.M("disconnect.confirm")
	mBlockStore         = msgs.M("block.store")
	mBlockProducts      = msgs.M("block.products")
	mBlockCollection    = msgs.M("block.collection")
	mFlowPaused         = msgs.M("flow.paused")
	mNotifyDiscTitle    = msgs.M("notify.disconnectedTitle")
	mNotifyDiscBody     = msgs.M("notify.disconnectedBody")
	mNotifyWebhookTitle = msgs.M("notify.webhookTitle")
	mNotifyWebhookBody  = msgs.M("notify.webhookBody")
	mConsentLabel       = msgs.M("consent.label")
	// The store blocks' words, in the email's language.
	mShopNow          = msgs.M("email.shopNow")
	mSoldOut          = msgs.M("email.soldOut")
	mTotal            = msgs.M("email.total")
	mReturnToCheckout = msgs.M("email.returnToCheckout")
	mRemovedFlag      = msgs.M("block.removedFlag")
	mListDescription  = msgs.M("list.description")
	mCheckRemoved     = msgs.M("check.removedTitle")
	mCheckRemovedTag  = msgs.M("check.removedTag")
	mCheckRemovedText = msgs.M("check.removedDetail")
)

// emailLang is the language a workspace writes its emails in by default
// (the store blocks' words, the store list's description): its default
// content language, English when it never set one.
func (s *Service) emailLang(ctx context.Context, wsID uuid.UUID) i18n.Lang {
	code := "en"
	if rows, err := s.Q.LanguagesList(ctx, wsID); err == nil {
		for _, r := range rows {
			if r.IsDefault && r.RemovedAt == nil {
				code = r.Code
			}
		}
	}
	return pagelabels.Lang(code)
}

// standard reports whether a language writes ReputeCast's own formats.
func standard(l i18n.Lang) bool { return l.Region == "" || l.Region == i18n.Standard }

// stampIn is a moment in a language's region format: "14 Oct, 09:05",
// "14 thg 10, 09:05".
func stampIn(l i18n.Lang, t time.Time) string {
	if standard(l) {
		return t.Format("2 Jan, 15:04")
	}
	return l.Region.DayMonth(t) + ", " + l.Region.Time(t, "")
}

// ConsentLabel is a store consent record's source, worded ("Shopify:
// accepted marketing at checkout"), for the audience's consent history.
func ConsentLabel(l i18n.Lang, platform string) string {
	return mConsentLabel.Text(l, i18n.Args{"platform": platformName(platform)})
}
