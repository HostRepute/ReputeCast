package audience

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/phone"
)

// Texts in the contacts table (design expansion phase 8, X21): the phone
// column, and the filters "has a phone", "phone country" and "SMS consent".

// ChannelFilter narrows contacts by their phone number and SMS consent.
type ChannelFilter struct {
	// Phone is "yes" for contacts with a number, "no" for those without.
	Phone string `json:"phone,omitempty" enum:"yes,no"`
	// PhoneCountry is the number's country (GB).
	PhoneCountry string `json:"phoneCountry,omitempty" maxLength:"2"`
	// SMSConsent is "any" (SMS consent on some list), "none", or a list id.
	SMSConsent string `json:"smsConsent,omitempty" maxLength:"40"`
}

// ChannelQuery is the same filter as query parameters.
type ChannelQuery struct {
	Phone        string `query:"phone" enum:"yes,no," doc:"yes: contacts with a phone number; no: without one"`
	PhoneCountry string `query:"phoneCountry" maxLength:"2" doc:"The phone number's country (GB)"`
	SMSConsent   string `query:"smsConsent" maxLength:"40" doc:"any: SMS consent on some list; none: on none; or a list id"`
}

func (q ChannelQuery) filter() ChannelFilter { return ChannelFilter(q) }

// channelFilter appends the phone and SMS consent conditions of f.
func channelFilter(ctx context.Context, out *filterSQL, f ChannelFilter) error {
	switch f.Phone {
	case "yes":
		out.add("c.phone <> ''")
	case "no":
		out.add("c.phone = ''")
	}
	if cc := strings.ToUpper(strings.TrimSpace(f.PhoneCountry)); cc != "" {
		out.add("c.phone_country = ?", cc)
	}
	switch v := strings.TrimSpace(f.SMSConsent); v {
	case "":
	case "any":
		out.add("EXISTS (SELECT 1 FROM channel_consents cc WHERE cc.contact_id = c.id AND cc.channel = 'sms' AND cc.status = 'subscribed')")
	case "none":
		out.add("NOT EXISTS (SELECT 1 FROM channel_consents cc WHERE cc.contact_id = c.id AND cc.channel = 'sms' AND cc.status = 'subscribed')")
	default:
		id, err := uuid.Parse(v)
		if err != nil {
			return api.Invalid("smsConsent", say(ctx, mSMSConsentFilter, nil))
		}
		out.add("EXISTS (SELECT 1 FROM channel_consents cc WHERE cc.contact_id = c.id AND cc.channel = 'sms' AND cc.status = 'subscribed' AND cc.list_id = ?)", id)
	}
	return nil
}

// fillPhones puts each contact's number on its row, as people read it.
func (s *Service) fillPhones(ctx context.Context, items []ContactListItem, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT id, phone FROM contacts WHERE id = ANY($1) AND phone <> ''`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	byID := map[string]string{}
	for rows.Next() {
		var id uuid.UUID
		var e164 string
		if err := rows.Scan(&id, &e164); err != nil {
			return err
		}
		byID[id.String()] = phone.Display(e164)
	}
	for i := range items {
		items[i].Phone = byID[items[i].ID]
	}
	return rows.Err()
}
