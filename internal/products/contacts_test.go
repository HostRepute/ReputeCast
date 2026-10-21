package products_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/products"
)

// A buyer whose email matches a contact is linked to it, whatever the case
// typed on Stripe's page. A buyer who ticks the list offer joins its list,
// following the list's double opt-in, with the offer's exact text as
// consent and "product_checkout:<the product's address>" as the evidence's
// source. Without the tick, or where the product offers nothing, nobody is
// subscribed: a buyer who isn't a contact stays one only on the order, for
// delivery and support, and enters no automation.
func TestBuyersAndContacts(t *testing.T) {
	s := newShop(t)
	fg := s.liveOf(s.fieldGuide)
	if !fg.OfferOn || fg.OfferListID != s.digest.String() || fg.OfferText == "" {
		t.Fatalf("the demo's Field Guide offers no Weekly Digest: %+v", fg)
	}
	const source = "product_checkout:deliverability-field-guide"
	var double bool
	s.scan(`SELECT double_opt_in FROM lists WHERE id = $1`, []any{s.digest}, &double)
	before := s.count(`SELECT count(*) FROM contacts WHERE workspace_id = $1`, s.ws.ID)
	boughtEvents := func(order uuid.UUID) int {
		t.Helper()
		return s.count(`SELECT count(*) FROM automation_events WHERE workspace_id = $1 AND kind = 'bought_product' AND payload->>'order_id' = $2`, s.ws.ID, order.String())
	}
	buyers := func(product, q string) int64 {
		t.Helper()
		var l struct{ Total int64 }
		s.owner.Do("GET", ws+"/contacts?bought="+product+"&q="+q, nil).MustStatus(t, 200).JSON(t, &l)
		return l.Total
	}

	// segment counts the contacts of a "Bought a product" condition.
	segment := func(op, value string) int64 {
		t.Helper()
		var r struct{ Count int64 }
		s.owner.Do("POST", ws+"/segments/preview", map[string]any{"mode": "builder", "definition": map[string]any{"groups": []map[string]any{
			{"match": "all", "conditions": []map[string]any{{"field": "bought_product", "op": op, "value": value}}}}}}).MustStatus(t, 200).JSON(t, &r)
		return r.Count
	}
	boughtGuide, boughtAny := segment("bought", s.fieldGuide.String()), segment("bought", "any")
	// sales are the product sales credited to a campaign.
	var campaign uuid.UUID
	s.scan(`SELECT id FROM campaigns WHERE workspace_id = $1 ORDER BY created_at DESC LIMIT 1`, []any{s.ws.ID}, &campaign)
	sales := func() products.CampaignProductSales {
		t.Helper()
		var cs products.CampaignProductSales
		s.owner.Do("GET", ws+"/campaigns/"+campaign.String()+"/product-sales", nil).MustStatus(t, 200).JSON(t, &cs)
		return cs
	}
	credited := sales()

	// A contact already, who clicked the campaign two days ago.
	quinn := s.addContact("quinn.reader@example.com", "Quinn")
	msg := uuid.Must(uuid.NewV7())
	s.exec(`INSERT INTO messages (id, workspace_id, stream, campaign_id, contact_id, email, status, queued_at) VALUES ($1, $2, 'broadcast', $3, $4, 'quinn.reader@example.com', 'delivered', $5)`,
		msg, s.ws.ID, campaign, quinn, s.clock.now().Add(-50*time.Hour))
	s.exec(`INSERT INTO tracking_link_clicks (message_id, link, clicked_at) VALUES ($1, 0, $2)`, msg, s.clock.now().Add(-48*time.Hour))
	o, _ := s.buy(s.fieldGuide, buyer{Email: "Quinn.Reader@Example.COM", Name: "Quinn Reader"})
	if o.ContactID == nil || *o.ContactID != quinn || o.Buyer != "contact" || o.BuyerEmail != "quinn.reader@example.com" {
		t.Fatalf("a contact's order: %+v", o)
	}
	if o.CampaignID == nil || *o.CampaignID != campaign || o.AttributedBy != "email" {
		t.Errorf("the sale isn't credited to the campaign Quinn clicked: %v %q", o.CampaignID, o.AttributedBy)
	}
	if cs := sales(); cs.Orders != credited.Orders+1 || cs.RevenueCents != credited.RevenueCents+2900 {
		t.Errorf("campaign product sales: %+v, were %+v", cs, credited)
	}
	if n := s.count(`SELECT count(*) FROM list_memberships WHERE contact_id = $1`, quinn); n != 0 {
		t.Errorf("an unticked purchase joined %d lists", n)
	}
	if n := boughtEvents(o.ID); n != 1 {
		t.Errorf("%d bought_product events for a contact's order", n)
	}
	var purchases products.ContactPurchases
	s.owner.Do("GET", ws+"/contacts/"+quinn.String()+"/purchases", nil).MustStatus(t, 200).JSON(t, &purchases)
	if len(purchases.Items) != 1 || purchases.Items[0].ID != o.ID.String() || purchases.SpentCents != 2900 || purchases.Currency != "usd" || !purchases.CanManage {
		t.Errorf("the contact's purchases: %+v", purchases)
	}
	if n := buyers(s.fieldGuide.String(), "quinn.reader"); n != 1 {
		t.Errorf("Quinn among the Field Guide's buyers: %d", n)
	}

	// A new buyer who ticks the offer.
	o2, _ := s.buy(s.fieldGuide, buyer{Email: "nico.new@example.com", Name: "Nico Newman", Ticked: true})
	nico := s.contactID("nico.new@example.com")
	if nico == uuid.Nil || o2.ContactID == nil || *o2.ContactID != nico || o2.Buyer != "joined" {
		t.Fatalf("a buyer who ticked the offer: contact %s, order %+v", nico, o2)
	}
	var first, last, csource, consent string
	s.scan(`SELECT first_name, last_name, source, consent FROM contacts WHERE id = $1`, []any{nico}, &first, &last, &csource, &consent)
	wantMember, wantConsent := "subscribed", "single"
	if double {
		wantMember, wantConsent = "pending", "pending"
	}
	if first != "Nico" || last != "Newman" || csource != source || consent != wantConsent {
		t.Errorf("new contact: %s %s, source %s, consent %s", first, last, csource, consent)
	}
	var member, msource string
	s.scan(`SELECT status, source FROM list_memberships WHERE list_id = $1 AND contact_id = $2`, []any{s.digest, nico}, &member, &msource)
	if member != wantMember || msource != source {
		t.Errorf("Weekly Digest membership: %s from %s, want %s", member, msource, wantMember)
	}
	var action, esource, text, note, ip string
	s.scan(`SELECT action, source, consent_text, note, coalesce(host(ip), '') FROM consent_events WHERE contact_id = $1 AND list_id = $2 ORDER BY occurred_at LIMIT 1`,
		[]any{nico, s.digest}, &action, &esource, &text, &note, &ip)
	if action != "subscribed" || esource != source || text != fg.OfferText || ip != buyerIP || !strings.Contains(note, fg.OfferText) {
		t.Errorf("consent evidence: %s, %s, %q, %q, %s", action, esource, text, note, ip)
	}
	confirmJobs := s.count(`SELECT count(*) FROM river_job WHERE kind = 'products.confirm' AND args->>'contact_id' = $1`, nico.String())
	if (confirmJobs == 1) != double || confirmJobs > 1 {
		t.Errorf("%d confirmation emails queued (double opt-in %v)", confirmJobs, double)
	}
	if v, ok, err := s.svc.Thanks(s.ctx, &s.ws, s.fieldGuide, o2.SessionID); err != nil || !ok || v.OfferPending != double || v.OfferList != "Weekly Digest" {
		t.Errorf("thank-you page: pending %v, list %q (%v %v)", v.OfferPending, v.OfferList, ok, err)
	}
	var d products.ProductOrderDetail
	s.owner.Do("GET", ws+"/products/orders/"+o2.ID.String(), nil).MustStatus(t, 200).JSON(t, &d)
	if d.Buyer.Kind != "joined" || d.Buyer.ContactID != nico.String() || d.Buyer.List != "Weekly Digest" || d.Buyer.ListStatus != wantMember || d.Buyer.DoubleOptIn != double {
		t.Errorf("order panel buyer: %+v", d.Buyer)
	}
	if !d.Consent.Offered || !d.Consent.Ticked || d.Consent.Text != fg.OfferText || d.Consent.IP != buyerIP || d.Consent.List != "Weekly Digest" ||
		!strings.HasSuffix(d.Consent.Page, "/products/deliverability-field-guide") {
		t.Errorf("order panel consent: %+v", d.Consent)
	}
	if n := boughtEvents(o2.ID); n != 1 {
		t.Errorf("%d bought_product events for a joined buyer", n)
	}

	// A new buyer who leaves it unticked.
	o3, _ := s.buy(s.fieldGuide, buyer{Email: "just.buying@example.com", Name: "Just Buying"})
	if c := s.contactID("just.buying@example.com"); c != uuid.Nil || o3.ContactID != nil || o3.Buyer != "none" || o3.BuyerEmail != "just.buying@example.com" {
		t.Fatalf("a buyer who isn't a contact: contact %s, order %+v", c, o3)
	}
	if n := boughtEvents(o3.ID); n != 0 {
		t.Errorf("%d bought_product events for a buyer who isn't a contact", n)
	}
	if n := s.count(`SELECT count(*) FROM consent_events e JOIN contacts c ON c.id = e.contact_id WHERE c.email = 'just.buying@example.com'`); n != 0 {
		t.Errorf("%d consent events for a buyer who isn't a contact", n)
	}
	var d3 products.ProductOrderDetail
	s.owner.Do("GET", ws+"/products/orders/"+o3.ID.String(), nil).MustStatus(t, 200).JSON(t, &d3)
	if d3.Buyer.Kind != "none" || d3.Buyer.ContactID != "" || d3.Buyer.ContactSince != nil || d3.Buyer.Email != "just.buying@example.com" || d3.Consent.Ticked {
		t.Errorf("order panel buyer: %+v, consent %+v", d3.Buyer, d3.Consent)
	}
	if n := buyers("any", "just.buying"); n != 0 {
		t.Errorf("a buyer who isn't a contact is among the buyers: %d", n)
	}

	// Ticked where the product offers nothing: nothing.
	s.exec(`UPDATE products SET live = jsonb_set(live, '{offerOn}', 'false') WHERE id = $1`, s.launchKit)
	o4, _ := s.buy(s.launchKit, buyer{Email: "no.offer@example.com", Ticked: true, Amount: 600})
	if s.contactID("no.offer@example.com") != uuid.Nil || o4.Buyer != "none" {
		t.Errorf("a tick without an offer made a contact: %+v", o4)
	}

	// A contact who ticks it joins the list and stays a contact who bought.
	o5, _ := s.buy(s.fieldGuide, buyer{Email: "quinn.reader@example.com", Ticked: true})
	if o5.ContactID == nil || *o5.ContactID != quinn || o5.Buyer != "contact" {
		t.Errorf("a contact who ticked: %+v", o5)
	}
	s.scan(`SELECT status FROM list_memberships WHERE list_id = $1 AND contact_id = $2`, []any{s.digest, quinn}, &member)
	if member != wantMember {
		t.Errorf("Quinn's Weekly Digest membership: %s, want %s", member, wantMember)
	}

	// The only contacts made: Quinn (by the test) and the buyer who ticked.
	if n := s.count(`SELECT count(*) FROM contacts WHERE workspace_id = $1`, s.ws.ID); n != before+2 {
		t.Errorf("%d new contacts, want 2", n-before)
	}
	// Segments: the two contacts who bought the Field Guide, nobody else.
	if n := segment("bought", s.fieldGuide.String()); n != boughtGuide+2 {
		t.Errorf("bought the Field Guide: %d contacts, were %d", n, boughtGuide)
	}
	if n := segment("bought", "any"); n != boughtAny+2 {
		t.Errorf("bought any product: %d contacts, were %d", n, boughtAny)
	}
}
