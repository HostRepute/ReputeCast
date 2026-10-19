package commerce

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/secrets"
)

// What the demo's seeder (internal/demo) writes the way the service does:
// the store's sealed credentials, its activity entries and the Checkout
// items of an abandoned-cart email.

// ActivityFacts are an activity entry's facts (store_activity.data).
type ActivityFacts = activity

// CheckoutLine is a line of an order or a checkout, as stored.
type CheckoutLine = orderItem

// SealCredentials seals a Shopify app's client secret and access token as
// a store keeps them.
func SealCredentials(box *secrets.Box, storeID uuid.UUID, clientSecret, accessToken string) ([]byte, error) {
	raw, err := json.Marshal(credentials{ClientSecret: clientSecret, AccessToken: accessToken}) //nolint:gosec // G117: sealed on the next line, never stored or returned in clear
	if err != nil {
		return nil, err
	}
	return box.Seal(raw, credentialsAD(storeID))
}
