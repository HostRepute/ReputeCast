package sponsors

import "time"

// The API's shapes. Amounts are in minor units of the slot's currency;
// calendar days are sent as their noon in the workspace's time zone so the
// app formats them on the right day.

// SponsorRef names a sponsor on a row.
type SponsorRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	LogoURL  string `json:"logoUrl,omitempty"`
}

// SponsorCampaignRef is the campaign of an issue.
type SponsorCampaignRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	// ByHand: linked by a person rather than by list and date.
	ByHand bool `json:"byHand"`
	// Author made the campaign; At is when it goes out or went out.
	Author string     `json:"author,omitempty"`
	At     *time.Time `json:"at,omitempty"`
}

// SponsorCreative is a booking's creative.
type SponsorCreative struct {
	Logo     string      `json:"logo,omitempty" maxLength:"1000"`
	Image    string      `json:"image,omitempty" maxLength:"1000"`
	Headline string      `json:"headline,omitempty" maxLength:"120"`
	Text     string      `json:"text,omitempty" maxLength:"1000" doc:"Up to 280 characters."`
	Button   string      `json:"button,omitempty" maxLength:"40"`
	Link     string      `json:"link,omitempty" maxLength:"1000"`
	UTM      SponsorUTMs `json:"utm"`
}

// SponsorUTMs are the UTM parameters added to the creative's link.
type SponsorUTMs struct {
	Source   string `json:"source,omitempty" maxLength:"100"`
	Medium   string `json:"medium,omitempty" maxLength:"100"`
	Campaign string `json:"campaign,omitempty" maxLength:"100"`
}

// SponsorPayment is a booking's payment, recorded by hand.
type SponsorPayment struct {
	// Status: unpaid, paid, or overdue once the due date passed.
	Status     string     `json:"status" enum:"unpaid,paid,overdue"`
	Due        *time.Time `json:"due,omitempty"`
	PaidOn     *time.Time `json:"paidOn,omitempty"`
	PaidCents  *int64     `json:"paidCents,omitempty"`
	Note       string     `json:"note"`
	RecordedBy string     `json:"recordedBy,omitempty"`
	RecordedAt *time.Time `json:"recordedAt,omitempty"`
}

// SponsorBookingBrief is a booking on an issue's row.
type SponsorBookingBrief struct {
	ID string `json:"id"`
	// Status: hold, booked, sent (the issue went out with it).
	Status         string          `json:"status" enum:"hold,booked,sent"`
	HoldUntil      *time.Time      `json:"holdUntil,omitempty"`
	Sponsor        *SponsorRef     `json:"sponsor,omitempty"`
	PriceCents     int64           `json:"priceCents"`
	CreativeStatus string          `json:"creativeStatus" enum:"waiting,ready"`
	CreativeDue    *time.Time      `json:"creativeDue,omitempty"`
	Payment        SponsorPayment  `json:"payment"`
	SentAt         *time.Time      `json:"sentAt,omitempty"`
	ResultsFinalAt *time.Time      `json:"resultsFinalAt,omitempty"`
	Final          bool            `json:"final"`
	SharedAt       *time.Time      `json:"sharedAt,omitempty"`
	ClickRate      *float64        `json:"clickRate,omitempty"`
	Proposal       *SponsorPropose `json:"proposal,omitempty"`
}

// SponsorPropose is an MCP agent's proposal for an open issue.
type SponsorPropose struct {
	ID         string      `json:"id"`
	Agent      string      `json:"agent"`
	Note       string      `json:"note"`
	At         time.Time   `json:"at"`
	Sponsor    *SponsorRef `json:"sponsor,omitempty"`
	PriceCents int64       `json:"priceCents"`
}

// SponsorIssueRow is one issue of a slot: open, on hold, booked or sent.
type SponsorIssueRow struct {
	SlotID   string               `json:"slotId"`
	SlotName string               `json:"slotName"`
	Number   int                  `json:"number"`
	IssueOn  time.Time            `json:"issueOn"`
	Campaign *SponsorCampaignRef  `json:"campaign,omitempty"`
	Booking  *SponsorBookingBrief `json:"booking,omitempty"`
	// Proposal is an agent's proposal for an open issue.
	Proposal *SponsorPropose `json:"proposal,omitempty"`
	Past     bool            `json:"past"`
}

// SponsorSlotItem is a slot with its schedule.
type SponsorSlotItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ListID     string `json:"listId"`
	ListName   string `json:"listName"`
	Placement  string `json:"placement"`
	PriceCents int64  `json:"priceCents"`
	Currency   string `json:"currency"`
	// The schedule: every EveryDays from FirstNumber on FirstOn; Weekday
	// names the day when the issues are weekly ("Tuesday").
	FirstNumber  int       `json:"firstNumber"`
	FirstOn      time.Time `json:"firstOn"`
	EveryDays    int       `json:"everyDays"`
	Weekday      string    `json:"weekday,omitempty"`
	CreativeDays int       `json:"creativeDays"`
	Selling      bool      `json:"selling"`
	Bookings     int       `json:"bookings"`
}

// SponsorItem is a sponsor of the library, with its bookings.
type SponsorItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Initials     string `json:"initials"`
	Website      string `json:"website"`
	LogoURL      string `json:"logoUrl"`
	ContactName  string `json:"contactName"`
	ContactEmail string `json:"contactEmail"`
	TermsDays    int    `json:"termsDays"`
	Notes        string `json:"notes"`
	// Issues lists its bookings' numbers; Hold its holds.
	Issues      []int  `json:"issues"`
	Hold        []int  `json:"hold"`
	BookedCents int64  `json:"bookedCents"`
	HoldCents   int64  `json:"holdCents"`
	PaidCents   int64  `json:"paidCents"`
	Currency    string `json:"currency"`
	// ClickRate is the latest final report's (unique clickers ÷ delivered)
	// with its issue.
	ClickRate      *float64 `json:"clickRate,omitempty"`
	ClickRateIssue int      `json:"clickRateIssue,omitempty"`
}

// SponsorTiles are the tab's numbers.
type SponsorTiles struct {
	Income30dCents int64 `json:"income30dCents"`
	IncomeIssues   int   `json:"incomeIssues"`
	Paid30dCents   int64 `json:"paid30dCents"`
	Unpaid30dCents int64 `json:"unpaid30dCents"`
	NextCount      int   `json:"nextCount"`
	NextBooked     int   `json:"nextBooked"`
	NextHold       int   `json:"nextHold"`
	NextOpen       int   `json:"nextOpen"`
	NextFrom       int   `json:"nextFrom"`
	NextTo         int   `json:"nextTo"`
	// AvgClickRate is pooled over the final reports of the last 90 days.
	AvgClickRate      *float64 `json:"avgClickRate,omitempty"`
	AvgClickIssues    []int    `json:"avgClickIssues"`
	BookedAheadCents  int64    `json:"bookedAheadCents"`
	BookedAheadIssues []int    `json:"bookedAheadIssues"`
	HoldAheadCents    int64    `json:"holdAheadCents"`
}

// SponsorNotice is something on the tab that needs a person.
type SponsorNotice struct {
	// Kind: creative_late (due today or past, the issue is near),
	// payment_overdue, hold_lapsing (ends tomorrow), no_campaign.
	Kind         string     `json:"kind" enum:"creative_late,payment_overdue,hold_lapsing,no_campaign"`
	BookingID    string     `json:"bookingId"`
	SlotID       string     `json:"slotId"`
	Issue        int        `json:"issue"`
	IssueOn      time.Time  `json:"issueOn"`
	Sponsor      string     `json:"sponsor"`
	ContactName  string     `json:"contactName"`
	ContactEmail string     `json:"contactEmail"`
	AmountCents  int64      `json:"amountCents"`
	Due          *time.Time `json:"due,omitempty"`
	DaysToIssue  int        `json:"daysToIssue"`
	ListName     string     `json:"listName"`
}

// SponsorOverview is Earn › Sponsors.
type SponsorOverview struct {
	Tiles    SponsorTiles      `json:"tiles"`
	Slots    []SponsorSlotItem `json:"slots"`
	SlotID   string            `json:"slotId"`
	Range    string            `json:"range" enum:"around,upcoming,recent,all"`
	Past     []SponsorIssueRow `json:"past"`
	Upcoming []SponsorIssueRow `json:"upcoming"`
	Library  []SponsorItem     `json:"library"`
	Notices  []SponsorNotice   `json:"notices"`
	Currency string            `json:"currency"`
	// CanManage: the person holds sponsors:manage; Helpers can help when not.
	CanManage bool     `json:"canManage"`
	Helpers   []string `json:"helpers"`
}

// SponsorBookingEvent is a line of a booking's activity.
type SponsorBookingEvent struct {
	Kind  string         `json:"kind"`
	At    time.Time      `json:"at"`
	Actor string         `json:"actor,omitempty"`
	Data  map[string]any `json:"data"`
}

// SponsorShareLink is a read-only link to a sponsor report.
type SponsorShareLink struct {
	ID         string     `json:"id"`
	URL        string     `json:"url,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	CreatedBy  string     `json:"createdBy"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	Views      int64      `json:"views"`
	LastViewed *time.Time `json:"lastViewedAt,omitempty"`
	// Every: the link shows every booking of the sponsor.
	Every bool `json:"every"`
}

// SponsorReportLink is one sponsor link's clicks.
type SponsorReportLink struct {
	Kind   string  `json:"kind" enum:"button,logo,link"`
	Label  string  `json:"label"`
	Clicks int64   `json:"clicks"`
	Share  float64 `json:"share"`
}

// SponsorReport is the report of one booking: who got the block, their
// human opens, unique clickers and clicks on the sponsor's links.
type SponsorReport struct {
	BookingID string     `json:"bookingId"`
	Issue     int        `json:"issue"`
	IssueOn   time.Time  `json:"issueOn"`
	ListName  string     `json:"listName"`
	SlotName  string     `json:"slotName"`
	Placement string     `json:"placement"`
	Sponsor   string     `json:"sponsor"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
	FinalAt   *time.Time `json:"finalAt,omitempty"`
	Final     bool       `json:"final"`
	// Delivered counts only the recipients who got the block.
	Delivered      int64               `json:"delivered"`
	HumanOpens     int64               `json:"humanOpens"`
	HumanOpenRate  float64             `json:"humanOpenRate"`
	UniqueClickers int64               `json:"uniqueClickers"`
	Clicks         int64               `json:"clicks"`
	ClickRate      float64             `json:"clickRate"`
	ClickToOpen    float64             `json:"clickToOpen"`
	Links          []SponsorReportLink `json:"links"`
	// LinkLine is where the links go ("northwind-analytics.example/trial
	// with utm_campaign=141").
	LinkHost     string          `json:"linkHost"`
	LinkCampaign string          `json:"linkCampaign"`
	Visibility   string          `json:"visibility,omitempty"`
	Creative     SponsorCreative `json:"creative"`
	SponsorName  string          `json:"sponsorName"`
	Initials     string          `json:"initials"`
	Label        string          `json:"label"`
}

// SponsorBookingDetail is the booking panel.
type SponsorBookingDetail struct {
	ID          string                `json:"id"`
	SlotID      string                `json:"slotId"`
	SlotName    string                `json:"slotName"`
	ListName    string                `json:"listName"`
	Placement   string                `json:"placement"`
	Issue       int                   `json:"issue"`
	IssueOn     time.Time             `json:"issueOn"`
	Status      string                `json:"status" enum:"proposed,hold,booked,sent,canceled"`
	HoldUntil   *time.Time            `json:"holdUntil,omitempty"`
	Sponsor     *SponsorItem          `json:"sponsor,omitempty"`
	PriceCents  int64                 `json:"priceCents"`
	SlotPrice   int64                 `json:"slotPriceCents"`
	Currency    string                `json:"currency"`
	Campaign    *SponsorCampaignRef   `json:"campaign,omitempty"`
	Creative    SponsorCreative       `json:"creative"`
	CreativeSt  string                `json:"creativeStatus" enum:"waiting,ready"`
	CreativeDue *time.Time            `json:"creativeDue,omitempty"`
	ReadyAt     *time.Time            `json:"readyAt,omitempty"`
	Payment     SponsorPayment        `json:"payment"`
	Notes       string                `json:"notes"`
	Activity    []SponsorBookingEvent `json:"activity"`
	Report      *SponsorReport        `json:"report,omitempty"`
	Shares      []SponsorShareLink    `json:"shares"`
	Proposal    *SponsorPropose       `json:"proposal,omitempty"`
	CanceledAt  *time.Time            `json:"canceledAt,omitempty"`
	// Label is the block's label for this sponsor ("Sponsored by …").
	Label     string   `json:"label"`
	CanManage bool     `json:"canManage"`
	CanShare  bool     `json:"canShare"`
	Helpers   []string `json:"helpers"`
	// Library is the sponsors to book.
	Library []SponsorRef `json:"library"`
}

// SponsorIssueDetail is an open issue's panel: book it or hold it.
type SponsorIssueDetail struct {
	SlotID      string              `json:"slotId"`
	SlotName    string              `json:"slotName"`
	ListName    string              `json:"listName"`
	Placement   string              `json:"placement"`
	Issue       int                 `json:"issue"`
	IssueOn     time.Time           `json:"issueOn"`
	PriceCents  int64               `json:"priceCents"`
	Currency    string              `json:"currency"`
	CreativeDue time.Time           `json:"creativeDue"`
	TermsDays   int                 `json:"termsDays"`
	PaymentDue  time.Time           `json:"paymentDue"`
	Campaign    *SponsorCampaignRef `json:"campaign,omitempty"`
	// BookingID is set when the issue is already booked or held.
	BookingID string           `json:"bookingId,omitempty"`
	Proposals []SponsorPropose `json:"proposals"`
	Library   []SponsorRef     `json:"library"`
	CanManage bool             `json:"canManage"`
	Helpers   []string         `json:"helpers"`
}
