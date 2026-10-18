package revenue

import "time"

// The API's shapes. Amounts are in minor units of the workspace currency
// (MRR is the monthly value; the ledger keeps twelve times it so yearly
// prices stay exact). Rates are percentages.

// RevenueOverview is Earn › Overview once Earn is live.
type RevenueOverview struct {
	Period   string    `json:"period" enum:"30d,90d,12m"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	Currency string    `json:"currency"`
	// Account names the live payment account ("Acme Media").
	Account string `json:"account"`
	// TooEarly: less than 30 days of data (no history before the live
	// connection). The 30-day numbers appear on ShownFrom.
	TooEarly    bool       `json:"tooEarly"`
	DataSince   *time.Time `json:"dataSince,omitempty"`
	ShownFrom   *time.Time `json:"shownFrom,omitempty"`
	ConnectedAt *time.Time `json:"connectedAt,omitempty"`
	// HadHistory: subscriptions older than the connection were found.
	HadHistory    bool                 `json:"hadHistory"`
	Tiles         RevenueTiles         `json:"tiles"`
	Chart         RevenueChart         `json:"chart"`
	Movement      RevenueMovement      `json:"movement"`
	Tiers         []RevenueTierRow     `json:"tiers"`
	Trials        RevenueTrials        `json:"trials"`
	Conversions   RevenueConversions   `json:"conversions"`
	Cancellations RevenueCancellations `json:"cancellations"`
	FreeToPaid    RevenueFreeToPaid    `json:"freeToPaid"`
	Upgrades      RevenueUpgrades      `json:"upgrades"`
	Needs         []RevenueNeed        `json:"needs"`
	Activity      []RevenueActivity    `json:"activity"`
	// Failed names the cards that couldn't be computed: the rest of the
	// page still works (chart, movement, tiers, trials, conversions,
	// cancellations, freeToPaid, upgrades, needs, activity).
	Failed []string `json:"failed"`
	// ReadOnly: the person can see Earn but change none of it; Helpers can.
	ReadOnly bool     `json:"readOnly"`
	Helpers  []string `json:"helpers"`
	// What the person can change, for the read-only card.
	CanEdit     bool `json:"canEdit"`
	CanManage   bool `json:"canManage"`
	CanPayments bool `json:"canPayments"`
	CanSponsors bool `json:"canSponsors"`
}

// RevenueTiles are the numbers at the top.
type RevenueTiles struct {
	MRRCents      int64    `json:"mrrCents"`
	MRRDeltaCents int64    `json:"mrrDeltaCents"`
	MRRDeltaPct   *float64 `json:"mrrDeltaPct,omitempty"`
	ARRCents      int64    `json:"arrCents"`
	Members       int64    `json:"members"`
	MembersDelta  int64    `json:"membersDelta"`
	New           int64    `json:"new"`
	Churned       int64    `json:"churned"`
	Trialing      int64    `json:"trialing"`
	// ChurnRate is churned ÷ paid members at the start.
	ChurnRate *float64 `json:"churnRate,omitempty"`
	ChurnBase int64    `json:"churnBase"`
	// FreeToPaidRate is free subscribers who became paid ÷ (free at the
	// start + joined).
	FreeToPaidRate *float64 `json:"freeToPaidRate,omitempty"`
	FreeToPaid     int64    `json:"freeToPaid"`
	FreeBase       int64    `json:"freeBase"`
	GiftsCents     int64    `json:"giftsCents"`
	SponsorsCents  int64    `json:"sponsorsCents"`
}

// RevenuePoint is the MRR at a moment.
type RevenuePoint struct {
	At       time.Time `json:"at"`
	MRRCents int64     `json:"mrrCents"`
	// History: before the move, read from the provider's subscriptions.
	History bool `json:"history"`
}

// RevenueMark is a numbered event on the MRR line.
type RevenueMark struct {
	N    int       `json:"n"`
	Kind string    `json:"kind" enum:"moved,offer"`
	At   time.Time `json:"at"`
	// Name is the platform moved from ("substack") or the offer's name.
	Name string `json:"name"`
}

// RevenueChart is MRR over the last 12 months, on the same day each month.
type RevenueChart struct {
	Points []RevenuePoint `json:"points"`
	// HistoryUntil is the move: before it the line comes from the
	// provider's subscription history.
	HistoryUntil *time.Time    `json:"historyUntil,omitempty"`
	Marks        []RevenueMark `json:"marks"`
	StartCents   int64         `json:"startCents"`
	EndCents     int64         `json:"endCents"`
	DeltaCents   int64         `json:"deltaCents"`
	DeltaPct     *float64      `json:"deltaPct,omitempty"`
	Since        time.Time     `json:"since"`
}

// RevenueMove is one kind of MRR change: its value and how many paid
// members it took.
type RevenueMove struct {
	Cents int64 `json:"cents"`
	Count int64 `json:"count"`
}

// RevenueMovement is the period's MRR from start to end.
type RevenueMovement struct {
	FromCents   int64       `json:"fromCents"`
	FromMembers int64       `json:"fromMembers"`
	ToCents     int64       `json:"toCents"`
	ToMembers   int64       `json:"toMembers"`
	New         RevenueMove `json:"new"`
	Reactivated RevenueMove `json:"reactivated"`
	Expansion   RevenueMove `json:"expansion"`
	Contraction RevenueMove `json:"contraction"`
	Churned     RevenueMove `json:"churned"`
}

// RevenueTierOffer is paid members of a tier and period on an offer.
type RevenueTierOffer struct {
	Name    string `json:"name"`
	Members int64  `json:"members"`
	// PaidCents is what they pay each period with it.
	PaidCents int64 `json:"paidCents"`
	// Duration: once (the first payment), months or forever.
	Duration string `json:"duration"`
	Months   int    `json:"months,omitempty"`
}

// RevenueTierRow is a tier and period: paying members and MRR today.
type RevenueTierRow struct {
	TierID     string             `json:"tierId"`
	Tier       string             `json:"tier"`
	Period     string             `json:"period"`
	Members    int64              `json:"members"`
	MRRCents   int64              `json:"mrrCents"`
	PriceCents int64              `json:"priceCents"`
	Offers     []RevenueTierOffer `json:"offers"`
	// Includes names the lower tiers it gives access to.
	Includes []string `json:"includes"`
}

// RevenueTrials is the period's trials and what became of them.
type RevenueTrials struct {
	Started    int64 `json:"started"`
	Trialing   int64 `json:"trialing"`
	Ended      int64 `json:"ended"`
	BecamePaid int64 `json:"becamePaid"`
	Canceled   int64 `json:"canceled"`
	EndingSoon int64 `json:"endingSoon"`
	// EndingFrom is the email most of the trials ending soon came from.
	EndingFrom *RevenueCredit `json:"endingFrom,omitempty"`
}

// RevenueCredit is an email upgrades or conversions are credited to.
type RevenueCredit struct {
	Kind  string `json:"kind" enum:"campaign,automation"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// RevenueConversionRow is conversions credited to one email.
type RevenueConversionRow struct {
	Kind       string `json:"kind" enum:"campaign,automation"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	ByToken    int64  `json:"byToken"`
	ByEmail    int64  `json:"byEmail"`
	Count      int64  `json:"count"`
	ValueCents int64  `json:"valueCents"`
}

// RevenueConversionLine is a conversion of the latest ones.
type RevenueConversionLine struct {
	At         time.Time `json:"at"`
	ValueCents int64     `json:"valueCents"`
	Currency   string    `json:"currency"`
	Matched    string    `json:"matched" enum:"click_token,email,none"`
	Credited   string    `json:"credited"`
	CampaignID string    `json:"campaignId,omitempty"`
}

// RevenueConversions is the product conversions recorded through the
// conversion API (not in MRR).
type RevenueConversions struct {
	Count      int64  `json:"count"`
	ValueCents int64  `json:"valueCents"`
	Currency   string `json:"currency"`
	// Since is the first conversion, when it's within the period.
	Since      *time.Time             `json:"since,omitempty"`
	ByEmail    []RevenueConversionRow `json:"byEmail"`
	Uncredited RevenueMove            `json:"uncredited"`
	// Name and ValueCents of a single kind of conversion ("Upgraded to
	// Team plan" at $29), and who sends them.
	Name         string                  `json:"name,omitempty"`
	TypicalCents int64                   `json:"typicalCents,omitempty"`
	Sender       string                  `json:"sender,omitempty"`
	Latest       []RevenueConversionLine `json:"latest"`
	// OtherCurrency counts conversions in another currency: kept, shown in
	// it, and left out of these totals.
	OtherCurrency int64 `json:"otherCurrency"`
	Total         int64 `json:"total"`
}

// RevenueReason is how many cancel-page visitors gave a reason.
type RevenueReason struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

// RevenueCancellations is the cancel page's period.
type RevenueCancellations struct {
	Visits    int64           `json:"visits"`
	Kept      int64           `json:"kept"`
	Continued int64           `json:"continued"`
	Undecided int64           `json:"undecided"`
	KeptOffer string          `json:"keptOffer,omitempty"`
	KeptTier  string          `json:"keptTier,omitempty"`
	Reasons   []RevenueReason `json:"reasons"`
	// OfferID is the live retention offer, for the card's link.
	OfferID string `json:"offerId,omitempty"`
}

// RevenueSourceRow is free subscribers of one source.
type RevenueSourceRow struct {
	Key  string `json:"key"`
	Kind string `json:"kind"`
	// Name is the form, import platform, key or integration; the app words
	// the rest.
	Name     string   `json:"name"`
	FormKind string   `json:"formKind,omitempty"`
	AtStart  int64    `json:"atStart"`
	Joined   int64    `json:"joined"`
	Base     int64    `json:"base"`
	Paid     int64    `json:"paid"`
	Rate     *float64 `json:"rate,omitempty"`
	TooFew   bool     `json:"tooFew"`
	// Others is how many sources an "other" row sums up.
	Others int `json:"others,omitempty"`
}

// RevenueFreeToPaid is free-to-paid conversion by how readers first joined.
type RevenueFreeToPaid struct {
	Rows  []RevenueSourceRow `json:"rows"`
	Total RevenueSourceRow   `json:"total"`
	// Direct counts new contacts who paid straight away; All is every new
	// paid member.
	Direct int64 `json:"direct"`
	All    int64 `json:"all"`
}

// RevenueUpgradeRow is new paid members credited to one email, or to none.
type RevenueUpgradeRow struct {
	Kind     string     `json:"kind" enum:"campaign,automation,none"`
	ID       string     `json:"id,omitempty"`
	Name     string     `json:"name"`
	SentAt   *time.Time `json:"sentAt,omitempty"`
	Members  int64      `json:"members"`
	MRRCents int64      `json:"mrrCents"`
}

// RevenueUpgrades is the period's new paid members by email.
type RevenueUpgrades struct {
	Rows  []RevenueUpgradeRow `json:"rows"`
	Total RevenueMove         `json:"total"`
	// Window is the attribution window in days.
	Window int `json:"window"`
}

// RevenueNeed is something that needs a person.
type RevenueNeed struct {
	// Kind: past_due, offer, creative, sponsor_payment, member_email.
	Kind  string `json:"kind" enum:"past_due,offer,creative,sponsor_payment,member_email"`
	Count int64  `json:"count,omitempty"`
	// Offers: its name, uses, limit and end.
	Name   string     `json:"name,omitempty"`
	Used   int64      `json:"used,omitempty"`
	Limit  int64      `json:"limit,omitempty"`
	EndsAt *time.Time `json:"endsAt,omitempty"`
	// Bookings: the issue, its list, the sponsor and the date due.
	Issue    int        `json:"issue,omitempty"`
	List     string     `json:"list,omitempty"`
	Sponsor  string     `json:"sponsor,omitempty"`
	Due      *time.Time `json:"due,omitempty"`
	Overdue  bool       `json:"overdue,omitempty"`
	TargetID string     `json:"targetId,omitempty"`
	// Member emails: the template's key.
	Template string `json:"template,omitempty"`
}

// RevenueActivity is a recent paid-member event.
type RevenueActivity struct {
	Kind        string         `json:"kind"`
	At          time.Time      `json:"at"`
	ContactID   string         `json:"contactId"`
	Name        string         `json:"name"`
	Tier        string         `json:"tier"`
	Period      string         `json:"period"`
	AmountCents *int64         `json:"amountCents,omitempty"`
	Currency    string         `json:"currency"`
	Data        map[string]any `json:"data"`
}

// EarnCard is the Earn card of the workspace Overview.
type EarnCard struct {
	// State: live (the numbers), invite (no payment provider yet) or none
	// (nothing to show).
	State         string     `json:"state" enum:"live,invite,none"`
	Account       string     `json:"account,omitempty"`
	Mode          string     `json:"mode,omitempty"`
	MRRCents      int64      `json:"mrrCents"`
	MRRDeltaCents int64      `json:"mrrDeltaCents"`
	Members       int64      `json:"members"`
	MembersDelta  int64      `json:"membersDelta"`
	ChurnRate     *float64   `json:"churnRate,omitempty"`
	Trialing      int64      `json:"trialing"`
	Currency      string     `json:"currency"`
	NextSponsor   *CardIssue `json:"nextSponsor,omitempty"`
	// SilentHours: no event from the provider for this long while
	// subscriptions are active (the webhook alert).
	SilentHours int64 `json:"silentHours,omitempty"`
	Active      int64 `json:"active"`
	CanSetUp    bool  `json:"canSetUp"`
}

// CardIssue is the next sponsored issue.
type CardIssue struct {
	BookingID   string     `json:"bookingId"`
	Issue       int        `json:"issue"`
	Sponsor     string     `json:"sponsor"`
	CreativeDue *time.Time `json:"creativeDue,omitempty"`
	Ready       bool       `json:"ready"`
}
