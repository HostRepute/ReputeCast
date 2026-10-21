package audience

// Web push and WhatsApp in the contacts table (design expansion phase 8,
// X22; brief §6 Contacts): the filters "Push subscribed" (a linked browser
// subscription; anonymous ones aren't contacts) and "WhatsApp consent" (on
// any list).

// PushWhatsAppFilter narrows contacts by push and WhatsApp.
type PushWhatsAppFilter struct {
	PushSubscribed  bool `json:"pushSubscribed,omitempty"`
	WhatsAppConsent bool `json:"whatsAppConsent,omitempty"`
}

// PushWhatsAppQuery is the same filter as query parameters.
type PushWhatsAppQuery struct {
	PushSubscribed  bool `query:"pushSubscribed" doc:"Only contacts with a linked push subscription"`
	WhatsAppConsent bool `query:"whatsAppConsent" doc:"Only contacts with WhatsApp consent on a list"`
}

func (q PushWhatsAppQuery) filter() PushWhatsAppFilter { return PushWhatsAppFilter(q) }

// pushWhatsAppFilter appends the push and WhatsApp conditions of f.
func pushWhatsAppFilter(out *filterSQL, f PushWhatsAppFilter) {
	if f.PushSubscribed {
		out.add(`EXISTS (SELECT 1 FROM push_subscriptions ps JOIN push_sites pst ON pst.id = ps.site_id AND pst.removed_at IS NULL
		WHERE ps.contact_id = c.id AND ps.removed_at IS NULL)`)
	}
	if f.WhatsAppConsent {
		out.add("EXISTS (SELECT 1 FROM channel_consents wc WHERE wc.contact_id = c.id AND wc.channel = 'whatsapp' AND wc.status = 'subscribed')")
	}
}
