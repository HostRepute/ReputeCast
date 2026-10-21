package agency

import (
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/store"
)

// MessageChannel reads one channel's usage from the channel message log
// (channel_messages, X21): SMS segments today, and push notifications and
// WhatsApp messages once those channels write there. A message counts once
// it has reached its provider (sent, delivered, undelivered or failed), on
// the UTC day it was sent, with the cost the provider reported.
type MessageChannel struct {
	// Name is sms, push or whatsapp.
	Name string
}

// MessageChannels are the channels the message log carries.
func MessageChannels() []ChannelUsage {
	return []ChannelUsage{MessageChannel{Name: "sms"}, MessageChannel{Name: "push"}, MessageChannel{Name: "whatsapp"}}
}

// Channel implements ChannelUsage.
func (c MessageChannel) Channel() string { return c.Name }

// Day implements ChannelUsage: units and cost per provider and, for SMS,
// segments per destination country.
func (c MessageChannel) Day(ctx context.Context, db store.DBTX, workspaceID uuid.UUID, day time.Time) ([]UsageRow, error) {
	q := store.New(db)
	from := dateOnly(day)
	to := from.AddDate(0, 0, 1)
	provs, err := q.AgencyChannelUsageByProvider(ctx, store.AgencyChannelUsageByProviderParams{WorkspaceID: workspaceID, Channel: c.Name, FromAt: from, ToAt: to})
	if err != nil {
		return nil, err
	}
	out := []UsageRow{}
	for _, p := range provs {
		out = append(out, UsageRow{Dimension: "provider", Key: p.Key, Label: p.Label, Units: p.Units, CostMicros: p.CostMicros})
	}
	if c.Name != "sms" || len(out) == 0 {
		return out, nil
	}
	countries, err := q.AgencyChannelUsageByCountry(ctx, store.AgencyChannelUsageByCountryParams{WorkspaceID: workspaceID, Channel: c.Name, FromAt: from, ToAt: to})
	if err != nil {
		return nil, err
	}
	for _, ct := range countries {
		out = append(out, UsageRow{Dimension: "country", Key: ct.Country, Label: countryName(ct.Country), Units: ct.Units})
	}
	return out, nil
}

// countryName is a country's English name ("CA" → "Canada").
func countryName(code string) string {
	r, err := language.ParseRegion(code)
	if err != nil {
		return code
	}
	if n := display.English.Regions().Name(r); n != "" {
		return n
	}
	return code
}
