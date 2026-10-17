package payments

// The merchants of record listed as "Not available yet" (brief 3.1): they
// sell to readers on the publisher's behalf and handle sales tax and VAT.
// A later release adds their adapters; until then they have no Open.
func init() {
	mor := []Capability{CapSubscriptions, CapOneTime, CapTrials, CapCoupons, CapPortal, CapTax, CapMerchantOfRecord, CapTestMode}
	Register(Descriptor{Key: "paddle", Name: "Paddle", MerchantOfRecord: true, Capabilities: mor})
	Register(Descriptor{Key: "lemonsqueezy", Name: "Lemon Squeezy", MerchantOfRecord: true, Capabilities: mor})
}
