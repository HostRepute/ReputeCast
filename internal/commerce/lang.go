package commerce

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
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
)

// ConsentLabel is a store consent record's source, worded ("Shopify:
// accepted marketing at checkout"), for the audience's consent history.
func ConsentLabel(l i18n.Lang, platform string) string {
	return mConsentLabel.Text(l, i18n.Args{"platform": platformName(platform)})
}
