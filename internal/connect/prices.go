package connect

import (
	"github.com/hostrepute/reputecast/internal/providers"
)

// "Help me choose" (review PR-13): every card shows what a month costs at
// the volume the person enters. The prices are each adapter's published
// price list in US dollars, reviewed quarterly with the terms; the client
// works out the month from them as the volume is typed.

// ConnectPriceList is an adapter's price list.
type ConnectPriceList struct {
	// Kind: metered (a price per 1,000 emails), plans (monthly plans by
	// volume), contract (set by a contract), self_hosted (no metered price:
	// you pay for your server) or relay (set by your server or relay).
	Kind string `json:"kind" enum:"metered,plans,contract,self_hosted,relay"`
	// PerThousand is the metered price per 1,000 emails.
	PerThousand float64 `json:"perThousand,omitempty"`
	// DataPerMB is a metered price per MB of data on top (Azure).
	DataPerMB float64 `json:"dataPerMb,omitempty"`
	// DedicatedIP is the monthly price of one dedicated IP as an add-on.
	DedicatedIP float64 `json:"dedicatedIp,omitempty"`
	// Plans are the monthly plans, smallest first.
	Plans []ConnectPricePlan `json:"plans,omitempty"`
	// DedicatedPlans replace Plans when a dedicated IP is needed.
	DedicatedPlans []ConnectPricePlan `json:"dedicatedPlans,omitempty"`
	// From is the cheapest paid plan when the list stops before larger
	// volumes ("plans from $17").
	From float64 `json:"from,omitempty"`
}

// ConnectPricePlan is one monthly plan.
type ConnectPricePlan struct {
	// Upto is the most emails a month the plan includes.
	Upto  int64   `json:"upto"`
	Price float64 `json:"price"`
	// Name is the provider's own name for the plan ("Foundation"); empty
	// for plans named by their size.
	Name string `json:"name,omitempty"`
	// UpTo words the plan "up to 50,000" rather than "50,000 included".
	UpTo bool `json:"upTo,omitempty"`
	Free bool `json:"free,omitempty"`
	// PerDay is a free plan's daily limit ("100 a day").
	PerDay int64 `json:"perDay,omitempty"`
	// Dedicated: the plan includes one dedicated IP.
	Dedicated bool `json:"dedicated,omitempty"`
}

// priceLists are the built-in adapters' price lists.
var priceLists = map[string]ConnectPriceList{
	"ses": {Kind: "metered", PerThousand: 0.10, DedicatedIP: 24.95},
	"postmark": {Kind: "plans", Plans: []ConnectPricePlan{{Upto: 10000, Price: 15}, {Upto: 50000, Price: 55}, {Upto: 125000, Price: 115},
		{Upto: 300000, Price: 245}}},
	"mailgun": {Kind: "plans", Plans: []ConnectPricePlan{{Upto: 3000, Free: true, PerDay: 100}, {Upto: 10000, Price: 15, Name: "Basic"},
		{Upto: 50000, Price: 35, Name: "Foundation"}, {Upto: 100000, Price: 90, Name: "Scale"}},
		DedicatedPlans: []ConnectPricePlan{{Upto: 100000, Price: 90, Name: "Scale", Dedicated: true}}},
	"sendgrid": {Kind: "plans", Plans: []ConnectPricePlan{{Upto: 50000, Price: 19.95, Name: "Essentials", UpTo: true},
		{Upto: 100000, Price: 34.95, Name: "Essentials", UpTo: true}}},
	"resend":    {Kind: "plans", Plans: []ConnectPricePlan{{Upto: 3000, Free: true}, {Upto: 50000, Price: 20, Name: "Pro"}, {Upto: 100000, Price: 35, Name: "Pro"}}},
	"sparkpost": {Kind: "contract"},
	"mailjet":   {Kind: "plans", Plans: []ConnectPricePlan{{Upto: 6000, Free: true}}, From: 17},
	"brevo":     {Kind: "plans", Plans: []ConnectPricePlan{{Upto: 9000, Free: true, PerDay: 300}}, From: 15},
	"azureacs":  {Kind: "metered", PerThousand: 0.25, DataPerMB: 0.00012},
	"postal":    {Kind: "self_hosted"},
	"smtp":      {Kind: "relay"},
}

// priceListOf is an adapter's price list, nil when it has none.
func priceListOf(adapter string) *ConnectPriceList {
	if pl, ok := priceLists[adapter]; ok {
		return &pl
	}
	return nil
}

// bulkRank ranks bulk-marketing terms for a recommendation: 2 allowed,
// 1 limited (Broadcast streams only, or your own policy), 0 not allowed
// (the terms badge warns: transactional only).
func bulkRank(d providers.Descriptor, tone string) int {
	switch {
	case tone == "warn" || d.Capabilities.Bulk == providers.BulkForbidden:
		return 0
	case d.Capabilities.Bulk == providers.BulkAllowed:
		return 2
	}
	return 1
}

// capabilityScore counts the capabilities ReputeCast uses that an adapter
// has: event webhooks, domain setup through the API, two-way suppression
// sync, quota discovery and batch sending.
func capabilityScore(d providers.Descriptor) int {
	c := d.Capabilities
	n := 0
	if c.Webhooks != "" && c.Webhooks != "Bounce mailbox" {
		n++
	}
	if c.DomainAPI == "full" {
		n++
	}
	if c.SuppressionSync {
		n++
	}
	if c.QuotaDiscovery {
		n++
	}
	if c.BatchSize > 1 {
		n++
	}
	return n
}

// capabilityMax is every capability capabilityScore counts.
const capabilityMax = 5
