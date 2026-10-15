# ReputeCast — Add-on spec: earn (payments, paid subscriptions, offers, gifts, sponsorships and revenue)

Phase 6 of `reputecast-design-expansion-plan.md`.

## 0. How to use this document

Read all of it before you start.

- **The prototype from the main spec, the team add-on, phase 1 and phase 5 already exists.**
  - Add the screens in section 5 as a new canvas page, "P6 Earn".
  - Apply section 6 to the existing screens.
  - Reuse the existing navigation, components and design system.
  - Add the patterns in section 7 to the "9 Design system" page first.
- **Same rules as before:**
  - Empty, loading, error and success states, plus the states listed for each screen.
  - Every screen has its read-only variant for members without the permission. It names the permission and who can help.
  - All copy in English.
  - Use the sample data in section 4, together with the main spec's, the team add-on's, phase 1's and phase 5's.
  - Desktop boards at 1440 px. Tablet (834) and phone (390) boards for 5.1, 5.4, 5.8 and 5.12.
  - Public pages at 720 px, plus phone, for 5.15 to 5.18. They use the publisher branding from Forms & pages › Branding and never show ReputeCast's teal.
- **Build order:**
  1. 5.1 Earn overview
  2. 5.2 Connect a payment provider
  3. 5.4 Plans & offers and 5.5 Tier editor
  4. 5.15 Upgrade page (public)
  5. 5.8 Paid members and 5.9 Member panel
  6. 5.14 Composer: Sponsor and Upgrade blocks, paid-only content
  7. 5.3 Earn settings
  8. 5.6 Offer editor and 5.16 Cancel page (public)
  9. 5.11 Gifts and 5.17 Give a gift (public)
  10. 5.12 Sponsors, 5.13 Booking panel and 5.18 Sponsor report (public)
  11. 5.10 Import paid members
  12. The rest, in order
  13. Then section 6

  If the work needs several passes, keep going until every screen exists.
- **Section 9** lists every board to draw and every existing board to change. When the phase is done, add the new boards to DSCoverage and `docs/DESIGN_INVENTORY.md`.

---

## 1. Principles

1. **Readers pay the publisher, not ReputeCast.**
   - Paid subscriptions, offers, gifts and the upgrade page are tools for publishers to charge their own readers through their own Stripe account.
   - ReputeCast takes no fee and gates nothing. Every feature in this brief is free and unlimited; only Stripe's own fees apply.
   - The publisher's paywall is for their readers. ReputeCast itself still shows no paywall, upgrade prompt, lock icon or plan badge.
2. **Stripe holds the money; ReputeCast mirrors it.**
   - Subscriptions, payments, refunds and disputes live in Stripe. ReputeCast keeps paid status in step from Stripe's events, and links to Stripe for everything about money.
   - ReputeCast never stores card details, not even the last four digits.
   - Nothing sits in between: each workspace connects its own Stripe account with its own restricted key.
3. **Honest numbers.**
   - MRR, churn and conversion each have one written definition (3.13), shown in a tooltip wherever the number appears.
   - Trials, free periods, complimentary and gift access never count as revenue.
   - Movements add up: the start, plus new and expansion, minus contraction and churn, equals the end.
   - Test-mode data never mixes with live numbers.
4. **Fair to readers.**
   - The price, the renewal date and what happens after a trial or a discount are stated before checkout.
   - Cancelling is never more than one decision away. At most one retention offer, on the same page as the cancel button, in a button of the same size.
   - Reminders before a trial ends, before a yearly renewal and before a gift ends.
   - No dark patterns: no pre-checked boxes, no guilt copy, no countdowns.
5. **Sender reputation first.** Member emails go through the Transactional stream, with its own route and suppression rules. Paid-only campaigns go through Send Readiness like any campaign. Sponsor links are checked like every link.
6. **Pluggable payments.** Payment providers follow the sending-provider pattern: a catalog with capability badges, one connection pattern, health status, a write-only credential form. Stripe comes first.
7. **Same access rules as the team add-on.**
   - Money and payment credentials are sensitive permissions (⚑).
   - Audit entries for every change people make.
   - Keys and signing secrets are write-only.
   - MCP agents read and propose. They never touch money.

---

## 2. Terms

- **Payment provider:** the service that takes readers' payments. Stripe now; Paddle and Lemon Squeezy later.
- **Merchant of record:** a payment provider that sells to readers on the publisher's behalf and handles sales tax and VAT. With Stripe, the publisher is the seller.
- **Restricted key:** a Stripe API key limited to the permissions ReputeCast needs. Live keys start with `rk_live_`, test keys with `rk_test_`.
- **Live and test:** a connection's mode. Test payments use Stripe's test cards and are kept apart from everything live.
- **Tier:** what readers pay for, such as "Weekly Digest Insider". It has prices, benefits, an optional trial, the lists its paid members join, an access rank and a visibility.
- **Price:** an amount per period, monthly or yearly. Stripe prices can't be edited, so a new amount is a new price. A price that no longer takes new paid members, but that some still pay, is a **legacy price**.
- **Paid member:** a contact with paid access. In Earn, copy always says "paid member", because "member" means a team member elsewhere in the app.
- **Paid status:**
  - **Trialing:** in a free trial, or in a period made free by a 100% discount (a "free period"). No payment yet.
  - **Active:** paying. "Cancels on 2 Nov" is added when a cancellation is scheduled.
  - **Past due:** the last payment failed and Stripe is retrying.
  - **Canceled:** access has ended.
  - **Complimentary:** access given by the publisher, without payment.
  - **Gift:** access bought by someone else for a fixed period.
- **Lifetime value:** what a paid member has paid, minus refunds, before tax.
- **MRR, ARR, churn rate, free-to-paid conversion:** defined in 3.13.
- **Offer:** a discount. Three kinds:
  - **Signup:** for new paid members, by code or link.
  - **Retention:** shown on the cancel page.
  - **Referral reward:** single-use codes for Grow's Stripe coupon rewards (phase 5).
- **Upgrade page, cancel page, gift page:** hosted public pages on the publisher's domain.
- **Gift:** a fixed period of a tier, bought once by one person for another, and redeemed without a card.
- **Sponsor:** a company that pays for a placement in the newsletter.
- **Slot:** a placement in each issue of a list, such as "Main sponsor".
- **Issue:** one edition of a recurring newsletter, such as Weekly Digest #143 on Tue 21 Oct.
- **Booking:** a sponsor in a slot of one issue, with its price, creative and payment status.
- **Conversion:** a valuable action in the publisher's own product, such as a SaaS upgrade or an order, recorded through the conversion API with a value.
- **Click token:** an opaque token ReputeCast adds to links to the publisher's own domains, so the conversion API can credit the email that brought the click.
- **Attribution window:** how long after an email click an upgrade or a conversion is credited to that email. 7 days by default.

---

## 3. Requirements

### 3.1 Payment providers
- **Catalog,** in Earn › Settings › Payments and in the Integrations hub's Payments group (phase 2):
  - **Stripe:** available.
  - **Paddle** and **Lemon Squeezy:** "Not available yet", each with one line: "A merchant of record: it sells to your readers for you and handles sales tax and VAT."
  - Capability badges: subscriptions, one-time payments, trials, coupons and promotion codes, customer portal, tax calculation, merchant of record, test mode.
- **Connecting Stripe** (5.2):
  - A restricted key, pasted into a write-only field. ReputeCast lists the 13 permissions the key needs (section 4) and checks each with a harmless call, naming any that's missing.
  - Secret keys (`sk_live_`, `sk_test_`) are refused: "Secret keys give full access to your Stripe account, including payouts. Create a restricted key instead."
  - ReputeCast reads the account's name, country and default currency, and whether Stripe Tax is on.
- **The webhook is registered automatically** in Stripe, at `https://reputecast.acme.io/hooks/stripe/{token}`, for the 12 events it needs.
  - The signing secret is sealed.
  - As for sending providers: verify the signature, store the raw event, answer 200 at once, and apply it later in a River job.
  - If the instance's public URL can't be reached from the internet, the step says so and offers a manual webhook: paste the signing secret from Stripe.
- **Customer portal:** ReputeCast sets up Stripe's portal for readers: update the card, see invoices, change the billing email, switch between monthly and yearly, and cancel.
- **Live and test:**
  - A live key and a test key can both be connected, each with its own webhook.
  - Public pages use the live key once it's connected. Before that, they run in test mode with a "Test mode · no real payments" ribbon, so the publisher can try everything first.
  - Team members can always run a test checkout from the upgrade page editor.
  - Test paid members are labelled Test, kept out of every number and every campaign audience, and can be cleared.
- **Currency:** the account's default currency (USD in the sample), fixed once a tier has prices. Readers are charged in that currency.
- **Stripe Tax:**
  - The page shows whether Stripe Tax is on in the account. When it is, a switch adds tax at checkout, and Checkout collects the address it needs.
  - The note: "With Stripe you're the seller: registering for and filing taxes is up to you. Merchants of record handle tax for you."
  - Prices exclude tax by default; a setting makes them include it.
- **Health:**
  - Statuses: Connected; Webhook silent (no event for 24 hours while at least 50 subscriptions are active); Webhook missing (deleted in Stripe); Key rejected (rolled or deleted); Missing permissions.
  - Webhook freshness: the last event and its type, events in the last 24 hours, rejected signatures.
  - Every hour ReputeCast also reads Stripe's list of events and applies any it missed, so a silent webhook never leaves a paid status more than an hour behind.
- **Payments event log:** each Stripe event next to what ReputeCast did with it, such as "invoice.paid · Ben Okoro · $4.00 · applied: renewed". Raw events are kept 90 days, like provider events.
- **Disconnect,** with a typed confirmation:
  - What stops: checkout, the portal links, gift sales, status updates. Upgrade blocks are hidden.
  - What doesn't: "Subscriptions keep billing in Stripe. Disconnecting never cancels anyone."
  - Paid members keep their access until the end of what they've paid for.
- **Engineering:** payment providers implement one provider interface (capabilities, checkout, portal, webhook parsing; optional coupons and tax) and pass a shared contract-test suite, like sending adapters.

### 3.2 Tiers and prices
- **A tier has:**
  - a name and a description
  - benefits: a short list, shown on the upgrade page
  - prices: monthly, yearly or both. The upgrade page shows the yearly saving ("2 months free") and the monthly equivalent.
  - a trial: none, 7, 14 or 30 days, for people who haven't had a trial before. Checkout asks for a card when the trial starts.
  - the lists its paid members join (Weekly Digest in the sample)
  - an access rank: a higher tier includes a lower tier's paid content ("Founding member includes Insider")
  - a visibility: Public (on the upgrade page), Hidden (only through its own link) or Archived (no new paid members; current ones keep paying)
- **Free** shows on the upgrade page as a card for the list itself, with its own short description.
- **Stripe:** saving a tier creates or updates its product in Stripe.
  - Changing an amount creates a new price. The old one stops taking new paid members.
  - Current paid members keep their price, shown as "Legacy price" on their row.
- **Imported prices** (3.6) are mapped to tiers and keep working.

### 3.3 Upgrade page and checkout
- **The upgrade page,** at news.acme.io/upgrade:
  - the headline and intro, the Monthly · Yearly switch, a card for each public tier (Free first), the highlighted tier, a short FAQ
  - the line "Payments go straight to Acme through Stripe. Acme never sees your card."
  - With an offer link (`?offer=FALL26`), the offer is applied and stated: "FALL26: $64 for your first year, then $80 a year."
  - From a personal link in an email (`{{ upgrade_url }}`), the page knows the reader ("Upgrading kenji.w@icloud.com · not you?") and credits the email (3.13).
  - A reader who already has paid access sees their tier and "Manage your subscription" instead of the cards.
  - A reader with an unclaimed referral reward (phase 5) sees it first: "You've earned 3 months of Insider free. Use it now." The reward applies to Insider monthly only.
- **Checkout** is Stripe Checkout, opened from a card's button:
  - The price, the trial and the renewal date are stated on the card before it opens.
  - The reader's email is filled in when known.
  - Tax is added when Stripe Tax is on.
  - Offers apply as promotion codes. A reader can also type a code in Checkout.
- **After checkout:**
  - Success: the welcome page (news.acme.io/upgrade/welcome), with what's included, when the next paid issue comes, and "Manage your subscription".
  - Canceled: back to the upgrade page, with "No charge was made."
  - The contact is matched by email (case-insensitive) or created. It joins the tier's lists with consent recorded as "Paid subscription", and gets the "Welcome to Insider" email (3.8).
- **Manage subscription:** the preference center gets a "Your paid subscription" section (section 6), with "Manage billing" (Stripe's portal) and "Cancel subscription" (the cancel page, 3.10).

### 3.4 Paid status on contacts
- **Every contact with paid access shows:**
  - tier, price and period, and paid status
  - member since, and the renewal or end date
  - any discount, and when it ends
  - lifetime value
  - source: the email or page that led to the upgrade, an import, a gift, or a grant
  - links to the Stripe customer and subscription
- **Mapping from Stripe:**
  - trialing → Trialing
  - active → Active, or Trialing while a 100% discount makes it free
  - past_due → Past due
  - canceled, unpaid, incomplete_expired → Canceled
  - incomplete (checkout not finished) → not shown
- **One paid subscription per contact.** If Stripe has two active subscriptions for one person, the contact shows the higher tier and a sync warning: "Stripe has 2 active subscriptions for this person. Refund and cancel one in Stripe."
- **The billing email can differ** from the contact's email. A change in Stripe's portal updates the billing email shown, never the contact's email.

### 3.5 Managing paid members
- **Complimentary access:**
  - Give a contact, or a new email address, a tier until a date or with no end, with a reason (team, contributor, press, partner, giveaway, other) and an optional note.
  - Optionally, send the "Welcome to Insider" email.
  - A new email address joins the tier's lists through the usual double opt-in.
  - Complimentary access lives in ReputeCast only. Nothing is created in Stripe.
  - Change the end date, or remove the access.
- **Cancel a subscription** for a reader who asks by email: at the end of the period, or now. Refunds are made in Stripe.
- **Open in Stripe:** the customer and the subscription, for refunds, invoices and disputes.
- **Resend a member email,** such as "Payment failed".
- **Export** paid members as CSV. Needs subscribers:export ⚑, and is audited.

### 3.6 Importing paid members
- **Sources:** exports from Substack, Ghost, beehiiv and Kit, or any CSV with an email column and, ideally, a Stripe customer ID column.
- **Matched with Stripe:** the import reads the connected account's customers and subscriptions, read-only, and matches each row by Stripe customer ID, then by email.
- **Results, by group:**
  - matched to an active subscription: imported as paid members
  - matched to a subscription that has ended: imported as free subscribers
  - no subscription in Stripe (gifted or complimentary on the old platform, or billed through another account): complimentary access until the paid-through date in the export (the default), import as free subscribers, or skip
  - conflicts (one email, several Stripe customers): pick one
- **Prices:** every Stripe price found is mapped to a tier, or kept as a legacy price of a tier.
- **Lists and consent:** the tier's lists. Existing contacts are merged. Consent is recorded as "Imported from Substack (paid member)".
- **Emails:** none by default. Optionally, the "Welcome to Insider" email.
- **Nothing changes in Stripe.** Subscriptions keep billing as before. The last step has a checklist: "Stop your old platform from charging or emailing these paid members, without canceling their subscriptions. Check with it how."
- Needs earn:manage ⚑ and subscribers:import. Audited, with the counts.

### 3.7 Paid-only campaigns and blocks
- **Paid-only campaigns:**
  - The audience picker gets "Who on this list: Everyone · Paid members only · Free subscribers only", with a tier filter: any paid tier, Insider and above, or Founding member.
  - "Paid members only" includes trialing, complimentary and gift access, and past due while Stripe retries (3.8).
  - The campaign gets a "Paid only" badge.
  - Unsubscribing works as for any campaign. A paid member who unsubscribes from the list keeps their access, but paid-only emails stop.
- **Paid-only blocks:** the Visibility setting (phase 3) gains "Paid access": any paid tier, a tier and above, or free subscribers only.
- **Upgrade block** (the paywall card): a headline, one line, the price and trial, and a button to `{{ upgrade_url }}`, with an optional offer. Shown to free subscribers only by default.
- **Preview as:** a free subscriber, a paid member of each tier, or any contact.
- **Send Readiness** renders each version (phase 3) and states the paid audience: "Weekly Digest · paid members only: 1,231 recipients."
- **Public archive:** paid-only campaigns are left out. Other issues appear in their free version: paid-only blocks removed, Upgrade blocks kept. Paid posts on the web come with phase 7.
- **Tokens:** `paid.tier`, `paid.status`, `paid.price`, `paid.renews_at`, `paid.ends_at`, `upgrade_url`, `manage_subscription_url`.

### 3.8 Member emails and access rules
- **Lifecycle emails are transactional templates** (phase 1), in a "Memberships" category, which is Essential.
  - They go through the Transactional route with its suppression rules, so they reach paid members who unsubscribed from marketing.
  - The eight: Welcome to Insider, Trial ends soon (3 days before), Payment failed, Renewal reminder (7 days before a yearly renewal), Membership canceled, Gift receipt, Your gift (on the delivery date), Gift ending (7 days before).
  - When Stripe is first connected, the 8 templates are created as drafts from starters. Each must be published (transactional:publish) before it's sent. Until then, the event is logged as "not sent: no published version".
  - Earn › Settings lists them with their template, version and status, and links to the editor.
- **Stripe's own emails:** "Stripe sends receipts, with the tax details. Keep them on. Turn off Stripe's emails about failed payments and upcoming renewals, so paid members don't get two. ReputeCast can't change those settings for you."
- **Access rules,** in Earn › Settings:
  - When a payment fails: keep access while Stripe retries (default), or end access at once.
  - A canceled subscription keeps access until the end of the paid period.
  - A refund doesn't end access by itself. Cancel the subscription too, here or in Stripe.
  - A trial that ends without payment ends access.

### 3.9 Offers
- **An offer has:**
  - a name and a kind (signup, retention, referral reward)
  - a discount: a percentage or a fixed amount
  - a duration: once, a number of months, or forever ("the first year" for yearly prices)
  - the tiers and periods it applies to
  - how readers get it: a code, a link (`news.acme.io/upgrade?offer=FALL26`) or both. Retention offers appear only on the cancel page; referral rewards are single-use codes.
  - a redemption limit and an end date, both optional
  - "New paid members only" (signup offers)
- **Stripe:** saving creates a coupon and promotion codes in Stripe. Changing the discount creates a new coupon; codes already used keep theirs.
- **Results:** redemptions, new paid members, and MRR from the offer, after the discount.
- **Ending:** an offer ends at its date or its limit. The upgrade page then says "This offer has ended" and shows the normal prices.

### 3.10 Cancel page and the retention offer
- **The cancel page** (news.acme.io/cancel, a personal link from the preference center):
  - what happens: "You'll keep Insider until 2 Nov. After that you'll still get the free Weekly Digest."
  - an optional reason
  - the retention offer, if the paid member is eligible
  - two buttons of the same size: "Keep Insider for $4/month" and "Continue to cancel"
- **Continue to cancel** opens Stripe's portal directly on that subscription's cancel step, where the reader confirms. Back on the publisher's domain, the page says "Your Insider membership ends on 2 Nov."
- **Eligibility for the retention offer** is set on the offer: tiers and periods, paid for at least N months, no retention offer in the last 12 months, not past due. One offer at most.
- **Accepting** applies the discount at once, and the page shows the new price and when it ends.
- **Reasons** go on the paid member's timeline and are summed on the Earn overview.

### 3.11 Gift subscriptions
- **Gift options:** a tier and a period with a one-time price, such as "Insider for 3 months, $24" and "Insider for a year, $80". The gift page can be turned on or off.
- **The gift page** (news.acme.io/gift):
  - pick a gift
  - the recipient's name and email, or "I'll give it in person" (the gift link goes to the buyer)
  - the delivery date: now, or a date and time in the buyer's time zone
  - a message, up to 300 characters
  - the buyer's name and email
  - one payment through Stripe Checkout. Nothing renews.
- **Delivery:** the "Your gift" email on the delivery date, with the redeem link. The buyer gets the "Gift receipt" at once.
- **Redeeming** (news.acme.io/gift/redeem/{code}), with no card:
  - The recipient confirms the address the gift is for. Redeeming adds them to the tier's lists, with the page's text as consent evidence, so the paid issues reach them.
  - Access starts at once and ends after the gift's period.
  - A gift link works once, for 12 months. Until it's used, it can be passed to someone else.
  - A recipient who already has paid access is told so, and can pass the gift on.
- **Reminders:** to a recipient who hasn't redeemed after 7 days; and "Gift ending", 7 days before the end, with "Keep Insider", which opens the upgrade page.
- **Earn › Gifts** lists gifts with buyer, recipient, option, amount, delivery date and status: Scheduled, Delivered, Redeemed, Not redeemed, Expired or Refunded.
  - Change the delivery date of a scheduled gift.
  - Resend the gift email.
  - Copy the redeem link (audited).
  - Refunds are made in Stripe.

### 3.12 Sponsorships
- **Slots,** per list: a name, where it sits in the issue, a default price, one booking per issue.
  - Issues come from the list's schedule: "every Tuesday, numbered from #142 on 14 Oct 2026".
  - An issue links to its campaign by list and date, or by hand.
- **Sponsor library:** name, logo, website, contact person and email, payment terms (due N days after the issue), notes. Per sponsor: bookings, booked and paid amounts, average click rate.
- **A booking has:**
  - the issue and slot, the sponsor and the price
  - a status: Open, Hold (until a date; then the issue opens again), Booked, Sent or Canceled
  - the creative: logo or image, headline, text (up to 280 characters), button text and link. ReputeCast adds UTM parameters to the link, and they can be changed.
  - the creative's status: Waiting (with a due date, 4 days before the issue by default) or Ready
  - the payment status, recorded by hand: Unpaid, Paid (date, amount, note) or Overdue once the due date passes. No invoicing.
  - notes
- **The Sponsor block** fills itself from the campaign's booking:
  - Always labelled "Sponsored", or "Sponsored by Northwind Analytics". The label can't be removed.
  - Without a booking for the issue, the block is hidden, and the composer says so.
  - While the creative is waiting, the composer shows a placeholder, and Send Readiness won't let the issue go out until the creative is ready or the block is removed.
  - Click tracking works as usual. The click token (3.13) is never added to sponsor links.
  - Like any block, it can be shown to free subscribers only. The sponsor report then counts only the recipients who got it.
- **Sponsor report,** for one booking or every booking of a sponsor:
  - delivered (recipients who got the block), human opens and their rate, unique clickers on sponsor links, clicks by link, click rate (unique clickers ÷ delivered), click-to-open (unique clickers ÷ human opens)
  - machine opens and bot clicks are left out, and the report says so
  - final 72 hours after the send; until then, "Results still coming in"
- **Sharing:** a read-only link like the campaign report's: optional expiry, revocable, audited, needs reports:share. The page uses the publisher branding and shows only the sponsor's numbers and the creative as sent, never contact data.
- **Sponsorships work without Stripe.**

### 3.13 Revenue and attribution
- **Definitions,** shown in tooltips:
  - **MRR:** the monthly value of every Active and Past due subscription, after discounts, before tax, in the workspace currency. A yearly price counts as a twelfth. Trials, free periods, complimentary and gift access, gift sales, sponsors, conversions and test mode are left out.
  - **ARR:** MRR × 12.
  - **Paid members:** Active and Past due subscriptions, the paying ones.
  - **MRR movement** over a period:
    - new: first payments
    - expansion: a higher tier or price
    - contraction: a lower price, a discount starting, monthly to yearly
    - churned: access ended
    - reactivated: paid again within 90 days of ending
  - **Churn rate:** paid members whose access ended in the period ÷ paid members at the start. A scheduled cancellation counts when access ends.
  - **Free-to-paid conversion:** free subscribers who became paid members in the period ÷ (free subscribers at the start + those who joined during it). New contacts who paid straight away are counted apart.
- **History:** when Stripe is connected, ReputeCast reads the account's subscription history, so MRR before the switch is filled in.
- **Attribution:**
  - An upgrade is credited to the last campaign or automation email the reader clicked within the attribution window before checkout. Otherwise it's credited to how they reached the upgrade page: an offer link, the website, or directly.
  - The window is 7 days by default, from 1 to 30.
  - Conversions use the same window.
- **Conversion API,** for SaaS founders and shops:
  - `POST /api/v1/workspaces/{workspace}/conversions`
  - Body: `email` or `click_token` (at least one), `name` (such as "Upgraded to Team plan"), `value` (a decimal string, such as "29.00"), `currency` (ISO 4217), `occurred_at` (optional, up to 30 days back), `metadata`. Header: `Idempotency-Key`.
  - Answer: `202 Accepted`, with the conversion ID and how it was matched: by click token, by email, or not credited.
  - Errors follow RFC 9457, each with a fix: `missing_identity`, `invalid_value`, `invalid_currency`, `occurred_at_out_of_range`, `rate_limited`.
  - A conversion in another currency is kept and shown in that currency, and left out of totals.
  - Keys need conversions:create. The API keys screen has a "Conversions key" preset.
- **Click token:** off by default. For each domain listed (app.acme.io in the sample), the click redirect adds `rc_click=…` to links to that domain. The token is opaque and signed, expires with the window, and is never added to links to any other domain.
- **Where revenue shows:** the Earn overview (5.1), the campaign report's Revenue section, and the Earn card on the Overview (section 6).

### 3.14 Permissions
New permission group, "Earn":
- view revenue, paid members, gifts and sponsors (earn:view)
- edit tiers, prices, offers, gift options and the upgrade, cancel and gift pages (earn:edit) ⚑
- manage paid members: complimentary access, imports, cancellations and gift changes (earn:manage) ⚑
- connect and manage payment providers, their keys and webhooks (payments:manage) ⚑
- manage sponsors, bookings and sponsor payment status (sponsors:manage)
- record conversions through the API (conversions:create)

Who gets what:
- **Built-in roles:**
  - Owner and Admin: all six.
  - Editor: earn:view and sponsors:manage.
  - Viewer: earn:view.
- **Content Writer**, the sample's custom role: none. Daniel Park can still add the Sponsor and Upgrade blocks and paid-only sections to a draft, which needs only campaigns:edit. The composer shows a booking's creative but not its price.
- **Combined with other permissions:**
  - Paid members' names and addresses also need subscribers:view. Exporting them needs subscribers:export ⚑.
  - Importing paid members also needs subscribers:import.
  - Sharing a sponsor report needs reports:share, like any read-only link.
  - The campaign report's Revenue section needs earn:view. Without it, the section says so.
  - A Grow reward with a Stripe coupon needs earn:edit.
- **API keys:** the "Conversions key" preset holds conversions:create only. A key can be given earn:view to read revenue and paid statuses.
- **MCP agents:** can read revenue numbers, paid-member counts and sponsor bookings, and can propose a booking for a person to confirm. They never see payment credentials, change prices or offers, grant access, cancel subscriptions or record payments.

### 3.15 Audit
- **Audited:**
  - a payment provider connected or disconnected; a key replaced; the webhook registered again; the test key added or removed; test data cleared
  - Stripe Tax at checkout, the currency, the access rules or the attribution settings changed
  - a tier created, changed, archived or deleted; a price added or replaced, with the old and new amounts
  - an offer created, changed, ended or deleted
  - the upgrade, cancel or gift page changed; gift options changed
  - complimentary access granted, changed or removed, with the reason
  - a subscription canceled by a team member
  - paid members imported, with the counts; paid members exported
  - a gift's delivery date changed, its email resent, or its redeem link copied
  - a sponsor, slot or booking created, changed, canceled or deleted; a payment recorded or changed
  - a sponsor report link created or revoked
  - a conversions key created, rotated or revoked, as for every key
- **Not in the audit log:** Stripe events (payments, renewals, failures, refunds, and the changes readers make in Checkout or the portal) and each conversion API call. The payments event log (90 days), the conversions log and each paid member's timeline are their record. Refunds are made in Stripe, whose own logs show who made them. See section 8.

### 3.16 Privacy and data
- **No card details, ever:** no card numbers, no brand, no last four digits, no expiry dates. No bank details, no billing addresses. Readers change them in Stripe's portal; the team sees them in Stripe.
- **Stored:** Stripe customer and subscription IDs, tier, price, status, dates, discounts, the amounts of payments and refunds (for lifetime value), the billing email.
- Keys and webhook signing secrets are sealed and write-only.
- Gift messages are kept until the gift ends, plus 30 days.
- Cancel reasons stay on the paid member's timeline.
- Erasing a contact (subscribers:delete) removes their paid status and timeline from ReputeCast. The dialog says Stripe keeps its own records: "Delete the customer in Stripe too if you need to."
- Nothing in Earn is sent to HostRepute.

### 3.17 Out of scope for this phase
- Paddle and Lemon Squeezy (a later release).
- Invoices for sponsors, self-serve sponsor booking and payment, and a sponsor marketplace.
- Member sign-in, paywalled web posts and digital products (phase 7).
- Pausing subscriptions; moving current paid members to a new price; several currencies in one workspace; group or team subscriptions.
- Refunds from inside ReputeCast. They're made in Stripe.
- Tax filing.

---

## 4. Sample data

"Now" is Wednesday 15 Oct 2026, 10:00 America/New_York, as in phases 1 and 5. Weekly Digest #142 went out on Tue 14 Oct.

### Story
- Acme ran its paid "Insider" tier on Substack, which billed through Acme's own Stripe account, "Acme Media". The free Weekly Digest came from Brevo.
- The newsletter moved to ReputeCast on Thu 18 Sep 2026:
  - Linh Nguyen connected Stripe the day before, Wed 17 Sep.
  - She imported the paid members from Substack's export on 18 Sep, matched to Acme Media's subscriptions.
  - The upgrade page and the gift page went live the same day.
- Marcus Chen started the FALL26 offer on 1 Oct.

### Stripe connection
- **Account:** Acme Media, United States, default currency USD.
- **Live:** restricted key `rk_live_…9fQ2`, connected 17 Sep 2026 by Linh Nguyen. 13 of 13 permissions, last checked 15 Oct, 09:50.
- **Test:** restricted key `rk_test_…3mZ8`, connected the same day, for test checkouts.
- **Webhook:** `https://reputecast.acme.io/hooks/stripe/…`, 12 events, registered 17 Sep.
  - Last event 4 min ago (invoice.paid).
  - 71 events in the last 24 hours, 0 rejected signatures.
  - The hourly gap check last ran at 09:00 and found nothing missed.
- **Stripe Tax:** on in Acme Media. Checkout adds tax; prices exclude tax.
- **Customer portal:** set up by ReputeCast.
- **Webhook events:** checkout.session.completed; customer.subscription.created, updated, deleted and trial_will_end; invoice.paid, payment_failed and upcoming; charge.refunded; charge.dispute.created; customer.updated and deleted.

### Restricted key permissions
| Stripe resource | Access | Why |
|---|---|---|
| Customers | Write | Create and match customers at checkout |
| Checkout Sessions | Write | Open checkout for tiers and gifts |
| Customer portal | Write | Manage billing, and the cancel step |
| Products | Write | One product per tier |
| Prices | Write | Tier and gift prices |
| Coupons | Write | Offers |
| Promotion codes | Write | Offer codes and referral rewards |
| Subscriptions | Write | Read status; cancel when a team member asks |
| Invoices | Read | Payments, failures and lifetime value |
| Charges and refunds | Read | Refunds and disputes |
| Webhook endpoints | Write | Register ReputeCast's webhook |
| Events | Read | Fill gaps when a webhook is missed |
| Tax settings | Read | Show whether Stripe Tax is on |

### Tiers
| Tier | Prices | Trial | Visibility | Paid members | MRR |
|---|---|---|---|---|---|
| Free (the Weekly Digest list) | – | – | Shown on the upgrade page | – | – |
| Weekly Digest Insider | $8/month · $80/year | 14 days, for people who haven't had one | Public, "Most popular" | 1,124: 431 monthly, 693 yearly | $7,960 |
| Founding member | $200/year | none | Public | 60 | $1,000 |
| **Total** | | | | **1,184** | **$8,960** |

- **Insider monthly:** 431. 419 at $8 ($3,352) and 12 on "Stay for half price" at $4 ($48). MRR $3,400.
- **Insider yearly:** 693. 648 at $80 ($4,320) and 45 with FALL26 at $64 a year ($240). MRR $4,560.
- **Founding member:** 60 at $200 a year. MRR $1,000.
- **ARR:** $107,520.
- **Insider benefits:** "A members-only deep dive every Tuesday" · "The full archive of deep dives" · "Quarterly roadmap calls with the Acme team" · "Reply to any issue and an editor answers".
- **Founding member benefits:** "Everything in Insider" · "Your name in the annual thank-you issue" · "Early access to Acme betas". It includes Insider's paid content.
- **Lists:** paid members of both tiers join Weekly Digest.
- **Stripe:** the prices are the ones Substack created in Acme Media, mapped to these tiers at the import.

### Offers
| Offer | Kind | Discount | For | How | Used | Ends |
|---|---|---|---|---|---|---|
| FALL26 | Signup | 20% off the first year: $64 instead of $80 | Insider yearly, new paid members | Code FALL26 and the link news.acme.io/upgrade?offer=FALL26 | 45 of 100 | 31 Oct 2026 |
| Stay for half price | Retention | 50% off for 3 months: $4/month | Insider monthly, paid at least 2 months, no retention offer in 12 months | Cancel page | Shown 34 times, accepted 12 (35%) | – |
| Referral reward · 3 months of Insider free | Referral reward | 100% off Insider monthly for 3 months | New subscriptions | Single-use codes from Grow's 10-referral milestone | 2 issued, 0 redeemed | Codes valid 60 days |

### Paid members today
- **Paying:** 1,184. 1,170 Active (26 of them cancel at the end of their period) and 14 Past due.
- **Trialing:** 37.
- **Complimentary:** 23. 17 came from the import, each until their paid-through date; 6 were granted by hand.
- **Gift:** 8.
- **With access:** 1,252.

### Last 30 days (15 Sep – 14 Oct)
- **MRR movement:**
  - start: $8,542, 1,119 paid members
  - new: +$692, 96 paid members: 39 Insider monthly after a trial ($312), 6 Insider yearly after a trial ($40), 45 Insider yearly with FALL26 ($240), 6 Founding members ($100)
  - expansion: +$26, 3 moved from Insider monthly to Founding member
  - contraction: −$60, 9 moved from monthly to yearly (−$12) and 12 took the retention offer (−$48)
  - churned: −$240, 31 paid members: 25 Insider monthly (−$200) and 6 Insider yearly (−$40)
  - end: $8,960, 1,184 paid members. +$418 (+4.9%).
- **Churn rate:** 2.8% (31 of 1,119).
- **Trials:** 112 started; 45 became paid, 30 canceled, 37 still trialing. 60% of ended trials became paid (45 of 75). 29 end in the next 7 days, 16 of them from Weekly Digest #141.
- **Upgrade page:** 4,180 visits, 312 checkouts started, 163 completed (112 trials, and 51 paid at once).
- **Cancel page:** 43 visits. 12 kept Insider with the offer, 26 went on to cancel, 5 left without deciding. Reasons: not reading enough 14, too expensive 11, content isn't for me 6, switching to another newsletter 3, other 4, no answer 5.
- **Refunds:** 1, $80 (Marta Kowalski, 13 Oct). No disputes.
- **Gifts:** 11 sold, $656.
- **Sponsors:** $2,250 from 3 issues (#140, #141, #142): $1,500 paid, $750 unpaid.

### Free-to-paid by source (last 30 days)
| Source | Free subscribers in the period | Became paid | Rate |
|---|---|---|---|
| Newsletter signup form | 13,400 (12,840 at the start, 560 joined) | 38 | 0.28% |
| Brevo import | 13,722 | 27 | 0.20% |
| Hosted subscribe page | 2,334 (2,214 + 120) | 9 | 0.39% |
| Other forms, API and CSV | 2,001 (1,950 + 51) | 6 | 0.30% |
| Referral (phase 5) | 171 (all joined) | 3 | Too few to compare |
| Partner · Ops Weekly (phase 5) | 102 (all joined) | 1 | Too few to compare |
| **Total** | **31,730** (30,726 at the start, 1,004 joined) | **84** | **0.26%** |

12 more paid members were new contacts who paid straight away: 96 in all.

### Upgrades by email (last 30 days)
Each is credited to the last email clicked within 7 days before checkout.

| Email | New paid members | New MRR |
|---|---|---|
| Weekly Digest #139 · 23 Sep | 19 | $140 |
| Weekly Digest #140 · 30 Sep | 16 | $116 |
| Weekly Digest #141 · 7 Oct | 9 | $82 |
| Welcome to news.acme.io · 18 Sep | 11 | $88 |
| Welcome series (automation) | 12 | $72 |
| No email click: upgrade page, offer links, website | 29 | $194 |
| **Total** | **96** | **$692** |

### Weekly Digest #141's Revenue section
- 386 human clicks on upgrade links; 58 checkouts started.
- 25 upgrades: 9 paid at once (6 Insider yearly with FALL26, 3 Founding members) and 16 trials (13 monthly, 3 yearly) that end on 21 Oct.
- $984 collected. New MRR $82 now, up to $206 if every trial becomes paid.
- 0.08% of delivered recipients upgraded (25 of 31,299).
- Sponsor: Northwind Analytics, $750, paid 9 Oct. 412 unique clickers (1.3%).

### Paid members (sample rows)
| Paid member | Tier and price | Status | Since | Renews or ends | Lifetime value | Source |
|---|---|---|---|---|---|---|
| Sam Rivera · sam.rivera@outlook.com | Insider · $64 for the first year with FALL26, then $80/year | Active | 7 Oct 2026 | Renews 7 Oct 2027 | $64 | Weekly Digest #141 |
| Priya Natarajan · priya.n@gmail.com | Founding member · $200/year | Active | 14 Apr 2025 | Renews 14 Apr 2027 | $400 | Imported from Substack |
| Ben Okoro · ben@okoro.example | Insider · $4/month until 12 Jan 2027 (Stay for half price), then $8 | Active | 12 Mar 2026 | Renews 12 Nov | $60 | Imported from Substack |
| Olga Bennett · olga.bennett@gmail.com | Insider · $8/month | Active · cancels 2 Nov | 2 Jul 2026 | Ends 2 Nov | $32 | Imported from Substack |
| Grace Okafor · g.okafor@outlook.com | Insider · $8/month | Past due | 13 Feb 2026 | Stripe tries again 16 Oct | $64 | Imported from Substack |
| Diego Alvarez · d.alvarez@yahoo.com | Insider · $8/month | Trialing | 7 Oct 2026 | Trial ends 21 Oct | $0 | Weekly Digest #141 |
| Ravi Menon · ravi@menon.example | Insider | Complimentary | 3 Oct 2026 | No end date | $0 | Granted by Marcus Chen · contributor |
| Lena Fischer · lena.fischer@hotmail.com | Insider | Gift | 11 Oct 2026 | Ends 11 Oct 2027 | $0 | Gift from Priya Natarajan |

- **Grace Okafor:** Stripe couldn't charge her card on 13 Oct ("Your card has insufficient funds") and tries again on 16 Oct. She keeps Insider meanwhile. member-payment-failed went out on 13 Oct.
- **Olga Bennett** canceled on 12 Oct on the cancel page ("Not reading enough"), after seeing the retention offer.
- **Ben Okoro** accepted "Stay for half price" on 12 Oct.
- **Marta Kowalski** (Ended): Insider yearly since 10 Oct 2025. She renewed on 10 Oct 2026 and asked for a refund. Marcus Chen refunded the $80 in Stripe on 13 Oct and canceled the subscription at once. Lifetime value $80.
- **Lena Fischer's** Weekly Digest consent was pending since the Mailchimp import. Redeeming the gift confirmed it, with the redeem page's text as evidence.

### Import from Substack (18 Sep 2026, Linh Nguyen)
- **File:** substack-acme-insider-export.zip, with 1,150 paid and complimentary subscribers.
- **Matched in Stripe (Acme Media):**
  - 1,121 to active subscriptions (1,110 by Stripe customer ID, 11 by email), imported as paid members
  - 12 to subscriptions that had ended, imported as free subscribers
  - 17 with no subscription in Stripe (complimentary on Substack), given complimentary access until the paid-through date in the export, between 31 Oct 2026 and 30 Jun 2027
- **Prices:** 3 found and mapped: $8/month and $80/year to Insider, $200/year to Founding member.
- **Contacts:** 1,083 already existed and were merged; 67 were new. All on Weekly Digest, with consent "Imported from Substack (paid member)".
- **Emails:** none sent.

### Member emails
| Email | Sent when | Template | Version |
|---|---|---|---|
| Welcome to Insider | a trial or paid subscription starts | member-welcome | v2 live |
| Trial ends soon | 3 days before a trial ends | member-trial-ending | v1 live |
| Payment failed | Stripe couldn't charge the card | member-payment-failed | v1 live |
| Renewal reminder | 7 days before a yearly renewal | member-renewal-reminder | v1 live |
| Membership canceled | a cancellation is confirmed | member-canceled | v1 live |
| Gift receipt | a gift is bought | gift-purchased | v1 live |
| Your gift | on the gift's delivery date | gift-received | v1 live |
| Gift ending | 7 days before a gift ends | gift-ending | v1 draft, never published |

- Category "Memberships" (Essential). From Acme Newsletter <members@notify.acme.io>, reply-to hello@acme.io. Transactional route: Postmark, with failover to Amazon SES.
- gift-ending has never been published, so Earn warns: "Gift ending reminders aren't sent until gift-ending is published. The first one is due on 3 Jan 2027."

### Gifts
| Bought | From | To | Gift | Paid | Delivery | Status |
|---|---|---|---|---|---|---|
| 14 Oct | Omar Haddad | Karim Haddad, karim.haddad@example.com | Insider · 1 year | $80 | 24 Dec, 08:00 | Scheduled |
| 13 Oct | Hannah Berg | jonas.berg@example.de | Insider · 3 months | $24 | 13 Oct | Not redeemed yet; reminder on 20 Oct |
| 12 Oct | Kenji Watanabe | yuki.watanabe@example.jp | Insider · 3 months | $24 | 1 Nov, 09:00 | Scheduled |
| 11 Oct | Priya Natarajan | Lena Fischer | Insider · 1 year | $80 | 11 Oct | Redeemed 11 Oct; ends 11 Oct 2027 |
| 9 Oct | Diego Alvarez | In person | Insider · 3 months | $24 | To Diego, 9 Oct | Redeemed 10 Oct by m.alvarez@yahoo.com; ends 10 Jan 2027 |

- The other 6 gifts of the 30 days: 5 for a year and 1 for 3 months, all redeemed.
- In all: 11 sold ($656: 7 × $80 and 4 × $24), 9 delivered, 8 redeemed, 2 scheduled.

### Sponsors
- **Slot:** Weekly Digest · "Main sponsor" · after the first story · $750 an issue · one booking per issue. Issues every Tuesday.
- **Payment terms:** due 14 days after the issue.

| Sponsor | Contact | Bookings | Booked | Paid | Click rate |
|---|---|---|---|---|---|
| Northwind Analytics | Maya Lindqvist, maya@northwind-analytics.example | #141, #143, #146 | $2,250 | $750 | 1.3% (#141) |
| Contoso Cloud | Felix Osei, felix@contoso.example | #140, #142, and #145 on hold | $1,500, plus $750 on hold | $750 | 1.1% (#140) |

### Bookings
| Issue | Date | Sponsor | Price | Booking | Creative | Payment |
|---|---|---|---|---|---|---|
| #140 | Tue 30 Sep | Contoso Cloud | $750 | Sent | – | Paid 3 Oct |
| #141 | Tue 7 Oct | Northwind Analytics | $750 | Sent · report shared 9 Oct | – | Paid 9 Oct |
| #142 | Tue 14 Oct | Contoso Cloud | $750 | Sent · results final on 17 Oct | – | Unpaid · due 28 Oct |
| #143 | Tue 21 Oct | Northwind Analytics | $750 | Booked | Waiting · due Fri 17 Oct | Unpaid · due 4 Nov |
| #144 | Tue 28 Oct | – | – | Open | – | – |
| #145 | Tue 4 Nov | Contoso Cloud | $750 | Hold until Mon 20 Oct | – | – |
| #146 | Tue 11 Nov | Northwind Analytics | $750 | Booked | Waiting · due Fri 7 Nov | Unpaid · due 25 Nov |
| #147 | Tue 18 Nov | – | – | Open | – | – |

- **Northwind Analytics' creative (#141):** headline "Find the dashboards nobody opens"; text "Northwind shows which reports your team actually uses, and which ones you can retire. Free for 14 days."; button "Try Northwind free"; link northwind-analytics.example/trial with `utm_source=acme-weekly-digest&utm_medium=email&utm_campaign=141`.
- **Contoso Cloud's creative (#142):** "Postgres that scales to zero" · "Managed Postgres with a branch for every pull request." · "Start building", to contoso.example/postgres.
- **Sponsor report, #141 for Northwind Analytics:** delivered 31,299; human opens 14,022 (44.8%); unique clickers 412; clicks 498 (button 431, logo 67); click rate 1.3%; click-to-open 2.9%. Sofia Rossi shared it on 9 Oct as `reputecast.acme.io/r/Qm7vT2pXk9LbN4sW8cYd1A`, expiring 7 Jan 2027.
- **#140 for Contoso Cloud:** delivered 31,126; 342 unique clickers (1.1%).

### Conversions
- **Key:** "Acme Cloud · conversions" (conversions:create), created by Marcus Chen on 6 Oct 2026, last used 12 min ago from 198.51.100.24.
- **Click token:** on for app.acme.io.
- **Since 6 Oct:** 64 conversions "Upgraded to Team plan" at $29, $1,856 in all.
  - 23 ($667) credited to October Product Update: 19 by click token, 4 by email.
  - 41 ($1,189) not credited to an email.

### Paid-only campaign
- "Insider: the Q4 roadmap call, annotated", a draft by Sofia Rossi, scheduled for Thu 16 Oct, 09:00.
- Audience: Weekly Digest · paid members only: 1,231 recipients. 1,252 have paid access; 14 of them unsubscribed from Weekly Digest's emails, 4 are suppressed and 3 paused their email.

### Weekly Digest #143 (draft, Tue 21 Oct)
- Northwind Analytics' Sponsor block after the first story, waiting for its creative.
- A paid-only section, "Insider deep dive: connection pooling without tears", for Insider and above.
- An Upgrade block for free subscribers: "Get the full deep dive with Insider" · "Paid members get a deep dive every Tuesday, the archive, and quarterly roadmap calls." · "$8/month or $80/year · 14 days free" · "Start your free trial".
- The Referral block near the end (phase 5).
- Two versions: free (30,373 recipients) and paid (1,231).

### People
- **Linh Nguyen** (Owner) connected Stripe and ran the import.
- **Marcus Chen** (Admin) runs tiers, offers and complimentary access, created the conversions key, and canceled Marta Kowalski's subscription.
- **Sofia Rossi** (Editor) runs sponsors and bookings, shares sponsor reports and builds the issues. She sees Plans & offers and Paid members read-only: she holds neither earn:edit nor earn:manage.
- **Daniel Park** (Content Writer) adds Sponsor and Upgrade blocks and paid-only sections to drafts. He has no Earn tab, and the composer hides prices from him.
- **Aisha Bello** (Viewer) sees all of Earn, read-only.

---

## 5. New screens

The new sidebar item "Earn" sits after Grow, with six tabs: Overview · Plans & offers · Paid members · Gifts · Sponsors · Settings. A chip in the top bar shows the payment connection on every Earn screen: "Stripe · Acme Media · Live".

### 5.1 Earn overview (Overview tab)
- **Period:** 30 days, 90 days or 12 months, and "Open upgrade page".
- **Numbers:**
  - MRR $8,960, ▲ $418 (+4.9%)
  - ARR $107,520
  - paid members 1,184, ▲ 65 (96 new, 31 churned)
  - churn rate 2.8%
  - free-to-paid 0.26%
  - other income: gifts $656, sponsors $2,250
- **MRR over 12 months,** with the markers "Moved from Substack · 18 Sep" and "FALL26 · 1 Oct". The months before 18 Sep come from Stripe's history.
- **MRR movement:** the bridge from $8,542 to $8,960, with counts.
- **By tier:** paid members and MRR per tier and period.
- **Trials:** 37 trialing; 60% of ended trials became paid; 29 end in the next 7 days.
- **Free-to-paid by source** and **upgrades by email:** the tables from section 4, each row linking to its source or report.
- **Cancellations:** cancel page visits, offers accepted, and the reasons.
- **Product conversions:** 64 since 6 Oct, $1,856; 23 credited to October Product Update.
- **Needs you:**
  - "14 payments failed. Stripe is retrying, and these paid members keep access meanwhile." → Paid members, filtered
  - "FALL26: 45 of 100 used, ends 31 Oct." → Offer
  - "Weekly Digest #143: Northwind Analytics' creative is due Fri 17 Oct." → Booking
  - "Contoso Cloud hasn't paid for #142 yet. Due 28 Oct." → Booking
  - "gift-ending isn't published, so gift reminders won't go out." → Template
- **Recent activity** from Stripe's events: "Sam Rivera started Insider yearly with FALL26 · $64.00 · from Weekly Digest #141 · 7 Oct", a failed payment, a refund, a cancellation, a gift redeemed.

States to show:
- the sample
- first run, the launch checklist:
  - "Earn from your newsletter: paid subscriptions, gifts and sponsorships. Readers pay you through your own Stripe account. ReputeCast takes no fee; Stripe's fees apply."
  - Steps: connect Stripe (a test key is fine); create a tier; publish the 8 member emails; review the upgrade page; try a test checkout; connect the live key and share the upgrade page.
  - Side cards: "Already charging on Substack, Ghost, beehiiv or Kit? Import paid members" and "Sponsorships work without Stripe".
- test only: a purple "Test mode" banner across Earn: "Only test payments so far. Checkout uses Stripe's test cards and nothing is charged. Test paid members are labelled Test and left out of every number." → Connect your live key
- conversions only: no payment provider, but conversions recorded. The overview shows conversions and sponsors, and invites to connect Stripe.
- too early: "MRR history needs 30 days of data."
- Aisha Bello: read-only
- Daniel Park (no earn:view): the in-shell 403: "Earn needs earn:view. Ask Marcus Chen or Linh Nguyen.", with "Request access"
- loading, and an error in one card

### 5.2 Connect a payment provider
A page with steps, like the connect wizard. Its progress is kept, so a reload resumes the same step.
1. **Provider:** the catalog (3.1).
2. **Restricted key:**
   - the 13 permissions as a table, with Copy and "Open Stripe ↗"
   - "Start with a test key" as the recommended path
   - paste the key, then "Check key"
3. **Check:** the account (Acme Media, United States, USD), the mode (Live or Test), Stripe Tax (on), and each permission with a tick or what's missing.
4. **Webhook:** progress for the endpoint created at `…/hooks/stripe/…`, the 12 events subscribed, the signing secret stored, then "Waiting for the first event. It arrives with the next payment, renewal or test checkout."
5. **Settings:** the currency (fixed once tiers have prices), Stripe Tax at checkout, prices with or without tax, and what the portal lets readers do.
6. **Done:** "Stripe is connected: Acme Media · Live", and the next step: create a tier, or import paid members.

States to show:
- the catalog
- a missing permission: "Webhook endpoints: Write is missing. Edit the key in Stripe, then check again."
- a secret key pasted: refused, with the reason
- the key rejected by Stripe: "Stripe says this key isn't valid."
- the public URL not reachable: "Stripe can't reach reputecast.acme.io. Set REPUTECAST_PUBLIC_URL to an address on the internet, or add the webhook by hand." → the manual path (paste the signing secret)
- waiting for the first event; the first event received
- the test key connected; the live key connected
- disconnect, with the typed confirmation and the list from 3.1
- Sofia Rossi (no payments:manage): "Connecting a payment provider needs payments:manage. Ask Marcus Chen or Linh Nguyen."

### 5.3 Earn settings (Settings tab)
Pill tabs: Payments · Member emails · Access · Attribution · Pages.
- **Payments:**
  - the provider card: Stripe · Live · Acme Media, the masked key with Replace, the permission check, "Connected 17 Sep by Linh Nguyen"
  - the test key under it, with "Show test data" and "Clear test data"
  - currency, Stripe Tax at checkout, prices with or without tax
  - webhook freshness: the last event, events in 24 hours, rejected signatures, the hourly gap check, and "Register again"
  - the payments event log: time, event, paid member, amount, and what ReputeCast did
  - "Add a payment provider" (the catalog), and Disconnect
- **Member emails:** the 8 emails from section 4, with template, version and status; the sender and route; the note about Stripe's own emails.
- **Access:** the rules from 3.8.
- **Attribution:**
  - the window and the click-token domains
  - the conversion API: the endpoint, the keys holding conversions:create, code samples in cURL, Node.js, Python, Go and PHP, and "Send a test conversion"
  - the conversions log
- **Pages:** the terms-of-sale link and the support address shown on the upgrade, cancel and gift pages.

States to show:
- the sample
- webhook silent (warn): "No event from Stripe for 26 hours, while 1,184 subscriptions are active. ReputeCast reads missed events every hour, so statuses are at most an hour behind."
- webhook missing (bad): "The webhook was deleted in Stripe on 14 Oct, 18:02. Paid statuses may be up to an hour behind." → Register again
- key rejected: "Stripe rejected the restricted key on 15 Oct, 07:40. It was rolled or deleted. Checkout and the portal links stop working until you replace it. Subscriptions keep billing in Stripe." → Replace key
- an unmapped price (the sync error): "3 subscriptions use a price ReputeCast doesn't know: $6.00/month (price_1Q8…k2). Map it to a tier, or keep it as a legacy price." → Map price
- a member email never published (gift-ending)
- no Transactional route: "Member emails need a Transactional route. Choose one in Sending providers › Routing." It also blocks the launch checklist.
- saved, and a validation error
- read-only (Sofia Rossi, Aisha Bello)

### 5.4 Plans & offers (Plans & offers tab)
- **Upgrade page card:** news.acme.io/upgrade, Live, 4,180 visits and 163 completed checkouts in 30 days, with "Edit page" (5.7), "Copy link" and "Preview with test checkout".
- **Tiers:** a card each for Free, Weekly Digest Insider and Founding member, with prices, trial, visibility, paid members, MRR and Edit. "New tier".
- **Offers:** the table from section 4, with kind, discount, who it's for, how readers get it, use and status. "New offer".
- **Gift options:** one line, linking to Gifts.

States to show:
- the sample
- no tiers yet: "Create your first tier. Readers see it on your upgrade page." Tiers can be drafted before Stripe is connected; publishing them needs Stripe.
- an offer at its limit; an offer that has ended
- Sofia Rossi, read-only: "Changing plans and offers needs earn:edit. Ask Marcus Chen or Linh Nguyen."
- loading, error

### 5.5 Tier editor
- name, description, and benefits (add, reorder)
- prices, monthly and yearly, with the saving and the monthly equivalent as readers will see them
- trial, lists, access rank, visibility
- the Stripe product and prices, with "Open in Stripe ↗"
- legacy prices still in use, with their paid members
- Save: "Saving updates the product in Stripe."

States to show:
- Weekly Digest Insider (the sample)
- changing a price: "Stripe prices can't be edited. ReputeCast creates a $9/month price for new paid members. The 431 monthly paid members keep paying $8, and their rows show Legacy price."
- archiving Founding member: "No new paid members. The 60 current ones keep paying, and the upgrade page stops showing it."
- creating the product in Stripe (progress), and a Stripe error with Retry
- a yearly price higher than 12 monthly payments: a warning, not a block

### 5.6 Offer editor
- the fields from 3.9, a preview of how the offer reads on the upgrade page or the cancel page, and its results

States to show:
- FALL26 (the sample)
- "Stay for half price": its eligibility, and a preview of the cancel page with the offer
- "Referral reward · 3 months of Insider free": single-use codes created by Grow, and a link to Grow's milestone
- an offer deleted in Stripe: "This offer's coupon no longer exists in Stripe, so readers can't use FALL26." → Create it again, or end the offer
- ending an offer early
- read-only

### 5.7 Upgrade page editor
Options on the left and the page on the right, like the Forms & pages editors.
- headline and intro, which tiers show and in what order, the default period (yearly), the highlighted tier and its label, the free card's text, the FAQ, the welcome page's text
- preview on desktop or mobile: as an anonymous visitor, as a free reader from an email, as a paid member, or with an offer link
- "Preview with test checkout"

States to show:
- the sample
- no public tier: "Readers will see only the Free card. Make a tier public first."
- unsaved changes

### 5.8 Paid members (Paid members tab)
- **Status chips:** With access 1,252 · Paying 1,184 · Trialing 37 · Past due 14 · Cancels soon 26 · Complimentary 23 · Gift 8 · Ended.
- **Search and filters:** tier, period, status, source, offer, member since.
- **Table:** paid member, tier and price, status, since, renews or ends, lifetime value, source. The rows from section 4.
- **Actions:** "Give complimentary access", "Import paid members", "Export CSV" (needs subscribers:export, audited).
- **Summary line:** "1,184 paying · $8,960 MRR · 14 past due, Stripe retrying".
- A row opens the member panel (5.9).

States to show:
- the sample
- filtered to Past due
- Ended, with Marta Kowalski
- "Show test data" on: test rows with the Test badge
- no paid members yet: "Paid members appear here when readers subscribe on your upgrade page." → Copy upgrade page link, Import paid members
- search with no results
- Aisha Bello: no actions; Export says "Exporting needs subscribers:export. Ask Marcus Chen or Linh Nguyen."
- loading, error

### 5.9 Member panel and actions
A side panel from the table, with its own link and "Open contact".
- **Summary:** tier, price and period, the status explained in words, since, renews or ends, the discount and its end, lifetime value, source, and the billing email if it differs.
- **Stripe:** the customer and subscription IDs, "Open in Stripe ↗".
- **Timeline:** checkout, payments, failures with Stripe's reason and the next try, renewals, plan changes, offers, emails sent (with their transactional status), cancellations with the reason, refunds.
- **Actions:** give complimentary access, cancel the subscription, resend a member email, open in Stripe.

States to show:
- Sam Rivera (the sample): "Active · Insider yearly · $64 for the first year with FALL26, then $80 a year · renews 7 Oct 2027"
- Grace Okafor, past due: "Stripe couldn't charge the card on 13 Oct (insufficient funds). Next try on 16 Oct. Grace keeps Insider while Stripe retries."
- Marta Kowalski, refunded and ended: "Refunded $80.00 on 13 Oct in Stripe. Marcus Chen canceled the subscription at once."
- two subscriptions for one person (the sync warning)
- Give complimentary access (dialog): Ravi Menon, Insider, no end date, "Contributor", "Send the welcome email"
- Cancel subscription (dialog) for Ben Okoro: "At the end of the period (12 Nov 2026)" or "Now. Make any refund in Stripe.", "Email Ben", and a note
- Sofia Rossi: the actions disabled, with "Managing paid members needs earn:manage. Ask Marcus Chen or Linh Nguyen."

### 5.10 Import paid members
A dialog in five steps:
1. **Source:** Substack, Ghost, beehiiv, Kit or another CSV, each with where to find its export.
2. **Upload:** the file, the columns found (email, Stripe customer ID, plan, paid-through date), and any problem.
3. **Match with Stripe:** progress, then the groups from 3.6 with counts and examples, and the choice for members without a subscription.
4. **Prices and lists:** each Stripe price found, mapped to a tier or kept as a legacy price; the tier's lists; whether to send the welcome email (off).
5. **Review and import,** then the summary and the checklist about the old platform.

States to show:
- the Substack sample at step 3 (the board)
- Stripe not connected: "Connect Stripe first. The import matches paid members with their Stripe subscriptions."
- a file without an email column: "No email column found. Pick the column that holds addresses."
- an unknown price: "$6.00/month (price_1Q8…k2) isn't mapped to a tier yet."
- a conflict: one email, two Stripe customers
- importing, in the jobs tray
- done

### 5.11 Gifts (Gifts tab)
- **Numbers (30 days):** 11 gifts sold, $656; 8 of 9 delivered gifts redeemed; 2 scheduled.
- **Gift options:** cards for "Insider · 3 months · $24" and "Insider · 1 year · $80", with Edit and "New option"; the gift page switch and link (news.acme.io/gift); the redemption window (12 months).
- **Gifts:** the table from section 4. A menu per row: change the delivery date (scheduled gifts), resend the gift email, copy the redeem link (audited), open the payment in Stripe.

States to show:
- the sample
- no gift options: "Let readers give Insider as a gift. They pay once; nothing renews."
- a gift not redeemed after 7 days: "Reminder sent on 20 Oct"
- a refunded gift: "Refunded in Stripe on 14 Oct. The redeem link no longer works."
- the gift page off
- Stripe not connected
- read-only

### 5.12 Sponsors (Sponsors tab)
- **Numbers:** income in 30 days ($2,250, $750 unpaid), the next 5 issues (2 booked, 1 on hold, 2 open), the average click rate.
- **Pill tabs:** Bookings · Sponsors · Slots.
  - **Bookings:** a row per issue and slot, from the table in section 4: issue and date, slot, booking badge, sponsor, price, creative badge, payment badge. Past issues link to their sponsor report; open issues have "Book".
  - **Sponsors:** the sponsor library, with "New sponsor".
  - **Slots:** Weekly Digest's "Main sponsor", with Edit and "New slot".

States to show:
- the sample
- no slots yet: "Sell a placement in your issues. Add a slot, then book sponsors into it. Sponsorships work without Stripe."
- a creative late: "#143 goes out in 4 days, and Northwind Analytics' creative hasn't arrived."
- a payment overdue: "Contoso Cloud: $750 for #142 was due on 28 Oct."
- a hold about to lapse: "Contoso Cloud's hold on #145 ends Mon 20 Oct."
- an issue with no campaign yet
- Aisha Bello: read-only (Daniel Park has no Earn tab, 5.1)
- loading, error

### 5.13 Booking panel and sponsor reports
A side panel from a booking, with its own link.
- the issue and slot, the campaign (linked), the sponsor and the price
- the status, and the hold date
- the creative: its fields, a preview of the Sponsor block, the due date, and "Mark as ready"
- payment: Unpaid, Paid or Overdue, with "Record payment" (date, amount, note)
- notes
- after the send: the sponsor report, with "Share read-only link"

States to show:
- #143 for Northwind Analytics, waiting for its creative (the board)
- #141, sent: the report, paid on 9 Oct, the shared link with its expiry and Revoke
- Record payment (dialog): "$750 · 9 Oct · Wire transfer, ref 5523"
- Share read-only link (dialog): the expiry, what the sponsor will see, Copy
- canceling a booking: "The Sponsor block in #145 will be hidden."
- booking #144
- read-only

### 5.14 Composer: Sponsor and Upgrade blocks, paid-only content and campaigns
- **Weekly Digest #143, Sponsor block selected:**
  - The canvas shows the placeholder "Sponsored · Northwind Analytics. The creative is due Fri 17 Oct. Until it arrives, this issue can't go out with the block."
  - The inspector: the booking (issue, slot, sponsor), layout, the "Sponsored" label (always shown, optionally "by Northwind Analytics"), the link with its UTM parameters, Visibility.
- **The same issue, paid content:**
  - The paid-only section with a "Paid · Insider and above" badge, and the Upgrade block with "Free subscribers only".
  - "Preview as: Free subscriber" shows the Upgrade block; "Insider member" shows the deep dive.
  - The summary: "2 versions: free (30,373) and paid (1,231)".
- **The paid-only campaign:** "Insider: the Q4 roadmap call, annotated", with the audience popover open on Weekly Digest · Paid members only · Any paid tier · 1,231 recipients. The header shows "Paid only".

States to show:
- the three frames above
- the Sponsor block filled, with #141's creative
- no booking for the issue: "No sponsor is booked for #144. This block will be hidden."
- Daniel Park: the booking shows its creative but not its price
- an Upgrade block with FALL26: "$64 for your first year"
- Stripe not connected: "The Upgrade block is hidden until a payment provider is connected."

### 5.15 Public: upgrade page and checkout return
The upgrade page at 720 px and on a phone, for an anonymous visitor arriving from the archive, with Yearly selected.
- "Get more from the Weekly Digest", and "The free Weekly Digest stays free. Insider adds a deep dive every Tuesday, and the people behind it."
- Monthly · Yearly (2 months free)
- the cards:
  - Free: "Subscribe free" (or "You're reading it" when the reader is known)
  - Weekly Digest Insider: "Most popular", $80 a year ($6.67 a month), 14 days free, the benefits, "Start your free trial"
  - Founding member: $200 a year, the benefits, "Become a founding member"
- "Payments go straight to Acme through Stripe. Acme never sees your card. Cancel any time from your email preferences; you keep access until the end of what you've paid for."
- the FAQ: when you're charged, switching between monthly and yearly, keeping the free issues
- the footer: Terms of sale · Privacy · "Sent with ReputeCast" (if on)

States to show (on the public states board):
- from an email: "Upgrading kenji.w@icloud.com · not you?"
- a referral reward waiting (Kenji Watanabe): "You've earned 3 months of Insider free. Use it now." The page opens on Insider monthly: "Free for 3 months, then $8 a month."
- FALL26 applied; FALL26 ended
- already a paid member (Sam Rivera): his tier, renewal date, and "Manage your subscription"
- the test-mode ribbon
- checkout canceled: "No charge was made."
- the welcome page after checkout: "Welcome to Insider, Sam.", what's included, the next paid issue (Tue 21 Oct), and "Manage your subscription"
- Stripe unreachable: "Payments aren't available right now. Nothing was charged. Please try again in a few minutes."

### 5.16 Public: cancel page
Olga Bennett's cancel page at 720 px and on a phone, before she decides.
- "Cancel your Insider membership?"
- "You'll keep Insider until 2 Nov, the end of the month you've paid for. After that you'll still get the free Weekly Digest."
- "Why are you leaving? (optional)": not reading enough, too expensive, content isn't for me, switching to another newsletter, other
- the offer card: "Stay for $4 a month for the next 3 months. Then it's $8 a month again, and you can cancel any time."
- two buttons of the same size: "Keep Insider for $4/month" and "Continue to cancel"
- "Continue to cancel opens Stripe to confirm. Nothing changes until you confirm there."

States to show (on the public states board):
- offer accepted (Ben Okoro): "Done. You pay $4 a month until 12 Jan 2027, then $8."
- no offer (Sam Rivera, yearly): only "Continue to cancel"
- canceled: "Your Insider membership ends on 2 Nov. You'll keep getting the free Weekly Digest."
- past due (Grace Okafor): "Your last payment didn't go through, so canceling ends Insider today and Stripe stops trying."
- the personal link expired: "Email me my link"

### 5.17 Public: give a gift and redeem it
Two pages, each at 720 px and on a phone.
- **Give a gift** (news.acme.io/gift), for an anonymous buyer:
  - "Give the Weekly Digest Insider"
  - the options: 3 months for $24, a year for $80 (selected)
  - "Who's it for?": name and email, or "I'll give it in person"
  - "When should it arrive?": now, or a date and time
  - a message, with "300 characters left"
  - your name and email
  - "Continue to payment", and "One payment through Stripe. Nothing renews. The gift link works for 12 months."
- **Redeem** (Lena Fischer's link):
  - "Priya Natarajan gave you a year of Weekly Digest Insider", with Priya's message
  - what's included
  - "Start my Insider year", for lena.fischer@hotmail.com
  - "No card needed. Insider ends on 11 Oct 2027 unless you choose to continue. Redeeming adds you to the Weekly Digest, so Insider issues reach you. You can unsubscribe any time."

States to show (on the public states board):
- gift scheduled: "Your gift to Karim arrives on 24 Dec at 08:00. We've emailed your receipt."
- redeemed: "You're an Insider until 11 Oct 2027."
- already a paid member: "You already have Insider. Pass this gift on: the link works for anyone until it's used."
- the link already used, or expired
- refunded: "This gift was refunded, so the link no longer works."

### 5.18 Public: sponsor report
Northwind Analytics' report for Weekly Digest #141, at 720 px and on a phone.
- the publisher logo, "Sponsor report", "Weekly Digest #141 · Tue 7 Oct 2026 · sponsored by Northwind Analytics"
- the numbers from section 4, and the creative as sent
- "How we count: human opens leave out Apple Mail Privacy Protection prefetches and security scanners. Clicks by bots are left out. Results are final 72 hours after sending."
- "Shared by Acme Newsletter · this link expires on 7 Jan 2027"

States to show (on the public states board):
- results still coming in (#142 for Contoso Cloud)
- every booking of a sponsor in one report (Northwind Analytics: #141 so far; #143 and #146 to come)
- the link revoked or expired

---

## 6. Changes to existing screens

- **App shell:**
  - The sidebar gets "Earn" after Grow, with a coins icon.
  - The tablet rail gets the same item. With Earn, the rail holds 11 items, its limit.
  - The phone More sheet lists Earn after Grow.
- **Settings › General:** an "Earn" switch in Sidebar sections, on by default. Turning it off only hides the item; payments, pages and blocks keep working.
- **Overview:** an Earn card, for members with earn:view:
  - MRR $8,960 ▲ $418, 1,184 paid members ▲ 65, churn 2.8%, 37 trials
  - the next sponsor: "#143 · Northwind Analytics · creative due Fri 17 Oct"
  - the webhook alert, when there is one
  - Without a payment provider, the card is an invitation that can be dismissed.
- **Settings › Integrations (the phase 2 catalog):** the Payments group: Stripe ("Connected · Acme Media · Live", which opens Earn › Settings › Payments), Paddle and Lemon Squeezy ("Not available yet").
- **Campaigns:** a "Paid only" badge, a "Sponsored · Northwind Analytics" chip on issues with a booking, and a "Paid only" filter.
- **Composer:**
  - The Blocks panel gets an "Earn" group: Sponsor and Upgrade.
  - The audience chip reads "Weekly Digest · Paid members only · 1,231".
  - Visibility (phase 3) gains "Paid access".
  - "Preview as" gains a free subscriber and each tier.
  - The token menu gets a "Paid" group.
- **Review & send (Send Readiness):**
  - Audience quality: "Paid members only: 1,231 recipients. Trials, complimentary and gift access are included, and 14 paid members whose payment failed while Stripe retries." (pass)
  - Content: "Sponsor block: Northwind Analytics' creative is missing." (block, with "Remove the block" and "Open booking"); "Sponsored content is labelled" (pass); "2 versions render: free and paid" (pass).
  - Links: sponsor links are checked like every link.
- **Campaign report:**
  - A Revenue section: upgrades (paid at once and trials), collected, new MRR now and if trials convert, upgrade clicks and checkouts, and conversions from the API. Weekly Digest #141 shows the numbers in section 4; October Product Update shows its 23 conversions ($667).
  - A Sponsor section: the sponsor, price, payment status, unique clickers and click rate, and "Share sponsor report".
  - States: "Results still coming in: 16 trials end on 21 Oct"; and without earn:view, "Revenue needs earn:view. Ask Marcus Chen or Linh Nguyen."
- **Contacts:** a "Paid status" filter (tier and status), a tier chip next to paid members' names, and a saved view "Paid members · 1,252".
- **Contact profile:** a "Paid subscription" card (Sam Rivera: Insider yearly, Active, renews 7 Oct 2027, lifetime value $64, from Weekly Digest #141, Open in Stripe), its timeline events, and the actions from 5.9.
- **Segment builder:** "Paid subscription" conditions: tier, status, period, member since, ended in the last N days, lifetime value.
- **Public preference center:**
  - For paid members, a "Your paid subscription" section at the top: tier, price, renewal date, "Manage billing" (Stripe's portal) and "Cancel subscription" (5.16).
  - Complimentary: "Complimentary Insider access from Acme, no end date". Gift: "A gift from Priya Natarajan, until 11 Oct 2027", with "Keep Insider after your gift".
  - For free readers, when the upgrade page is public: "Get more with Insider · from $8 a month", which opens the upgrade page.
- **Public unsubscribe page:** for a paid member: "You're an Insider member. Unsubscribing stops all emails, including Insider issues, but doesn't cancel your subscription." → Cancel subscription
- **Public archive:** paid-only campaigns are left out, and issues show their free version.
- **Roles & permissions:** the Earn group (3.14) in the role editor and the compare matrix.
- **Audit log:** the actions from 3.15, and an "Earn" filter category.
- **Notifications:**
  - New kinds: payment webhook missing or silent, key rejected, and an unmapped price (payments:manage); a paid-members import finished (earn:manage); a sponsor creative due tomorrow or late, and a sponsor payment overdue (sponsors:manage); a daily summary of failed payments (earn:view, off by default).
  - An "Earn" filter tab.
- **Webhooks:** a "Paid members" event group: `subscription.trial_started`, `subscription.started`, `subscription.renewed`, `subscription.payment_failed`, `subscription.cancel_scheduled`, `subscription.ended`, `subscription.changed`, `access.complimentary_granted`, `access.complimentary_ended`, `gift.purchased`, `gift.redeemed`, `conversion.recorded`.
- **API keys:** a "Conversions key" preset (conversions:create), and a "Conversions" badge on its rows.
- **MCP server:** new tools in the agents' permission matrix: get_revenue_overview, get_paid_member_counts, list_sponsor_bookings, propose_sponsor_booking.
- **Migration importer › Source:** a line under the sources: "Moving paid members from Substack, Ghost, beehiiv or Kit? Import them in Earn, matched with your Stripe subscriptions."
- **Transactional (phase 1 boards):** the "Memberships" category (Essential), with the 8 starters, in Templates and in Settings › Categories.

---

## 7. New design-system patterns

Add these to the "9 Design system" page before the screens. Use the existing tokens and badge colors.

- **Money:**
  - The currency symbol before the amount, tabular figures, thousands separators.
  - Totals and MRR in whole dollars ($8,960); single payments with cents ($64.00).
  - Prices with their period: "$8/month", "$80/year". Negative amounts with a minus sign: "−$60".
  - Deltas with an arrow and the change: "▲ $418 (+4.9%)" in `ok-fg`, "▼" in `bad-fg`. Never colour alone.
- **MRR movement bridge:** start and end bars with the new, expansion, contraction and churned segments between them, each labelled with its amount and count. It uses the chart palette, not status colours, and has a table view.
- **Paid status badges:**
  - Trialing: info, with "free until" for a free period
  - Active: ok, followed by a neutral "Cancels 2 Nov" chip when a cancellation is scheduled
  - Past due: warn
  - Canceled: neutral
  - Complimentary: purple
  - Gift: purple, with a gift icon
  - Test: purple, as in phase 1
- **Payment mode badge:** Live: ok · Test: purple.
- **Booking badges:** Open: neutral outline · Hold: info, with its end date · Booked: ok · Waiting for creative: warn · Ready: ok · Sent: neutral · Canceled: neutral.
- **Payment badges:** Paid: ok · Unpaid: neutral · Overdue: warn.
- **Public tier cards:** name, price and period, the monthly equivalent of a yearly price, the trial line, benefits with check icons, one button. The highlighted card has a `publisher-accent` border and its label. With the Monthly · Yearly switch. No lock icons.
- **Paywall card** (the Upgrade block): headline, one line, price and trial, button. An email-safe version and a public version.
- **Gift cards:** the gift option card (period and price), and the message card the recipient sees (from, message, what's included).
- **Payment provider catalog card:** the phase 2 integration card, with capability badges and "Not available yet".
- **Issue × slot rows:** issue and date, slot, sponsor, and the booking, creative and payment badges.

---

## 8. Conflicts to note

1. **Earn's tabs.** The plan's navigation lists Overview · Plans & offers · Paid members · Gifts · Sponsors (and Products in phase 7), while 6.1 puts payments in "Earn › Settings › Payments". This brief adds a Settings tab to Earn. Workspace Settings still gets no new tab, and payment providers also appear in the Integrations hub.
2. **Statuses.** The plan lists trialing, active, past due, canceled and complimentary. This brief adds Gift as a kind of access, "Cancels on" as a note on Active, and counts a period made free by a 100% discount as Trialing.
3. **Definitions.** MRR leaves out trials, free periods, complimentary and gift access, and includes past due while Stripe retries. Churn counts when access ends. The owner needs to confirm these.
4. **Cancel flow.** The plan shows the retention offer "on a hosted cancel page, shown before Stripe's portal". Here the hosted page shows the offer, then opens Stripe's portal directly on the subscription's cancel step. A paid member who opens the portal from "Manage billing" can still cancel there without seeing the offer; ReputeCast doesn't hide that.
5. **Member emails.** The plan names payment-failed and renewal emails. This brief uses the same mechanism for 8 lifecycle emails, all in an Essential "Memberships" category. Receipts stay with Stripe.
6. **Transactional dependency.** Member emails need a Transactional route and published templates (phase 1). Hiding Transactional in the sidebar doesn't stop them, and Earn links to the templates directly.
7. **Audit log vs payments event log.** As in phase 1, automatic events (Stripe's payments, renewals, failures and refunds, readers' own changes, and conversion API calls) stay in the payments event log, the conversions log and timelines, not the audit log. Everything a team member does is audited. The owner needs to confirm this.
8. **No card details at all.** The plan asks to show payment data only as far as the publisher needs. This brief stores none: no brand, no last four digits. Support goes through "Open in Stripe".
9. **Restricted keys only.** Secret keys are refused, though Stripe would accept them.
10. **No Stripe Connect platform.** Connecting through OAuth would need a platform account run by someone, which is a middleman. Each workspace uses its own restricted key instead.
11. **Archive.** Paid-only campaigns stay out of the public archive until phase 7's website adds paid posts.
12. **"Not available yet."** Paddle and Lemon Squeezy appear in the catalog with that label. It states availability; it isn't a plan badge.
13. **"Paid member" vs "member".** The team add-on uses "member" for team members, so Earn's copy always says "paid member".
14. **Sample story.** The other boards say the newsletter came from Brevo. In this brief the paid tier ran separately on Substack, billing through Acme's own Stripe account, and moved to ReputeCast on 18 Sep 2026, the same day as the newsletter. "Now" is Wed 15 Oct 2026, as in phases 1 and 5; the existing boards keep their dates.
15. **Attribution window.** 7 days by default; the owner may prefer another.
16. **Sponsor reports and Visibility.** When the Sponsor block shows to free subscribers only, the report's "Delivered" counts only those recipients.
17. **Retention offers.** The plan doesn't say who gets one. This brief allows one offer at most per paid member per 12 months, with the eligibility set on the offer.

---

## 9. Boards

All new boards go on the canvas page "P6 Earn". Desktop boards are 1440 px wide; public boards 720 px. Tablet and phone boards are wrappers that render the same markup at 834 and 390 px, like the existing ones.

### New boards
| Board | Canvas title | Implements | Tablet and phone |
|---|---|---|---|
| `EarnOverview.dc.html` | Earn · overview | 5.1 | `EarnOverviewTablet`, `EarnOverviewPhone` |
| `EarnStates.dc.html` | Earn · launch checklist, test only, conversions only, no access, read-only, loading, error | 5.1 states | – |
| `EarnConnect.dc.html` | Earn · connect Stripe | 5.2 | – |
| `EarnSettings.dc.html` | Earn · settings, with the webhook, key, price and template states | 5.3 | – |
| `EarnPlans.dc.html` | Earn · plans & offers | 5.4 | `EarnPlansTablet`, `EarnPlansPhone` |
| `EarnPlanEditors.dc.html` | Earn · tier, offer and upgrade page editors | 5.5–5.7 | – |
| `EarnMembers.dc.html` | Earn · paid members | 5.8 | `EarnMembersTablet`, `EarnMembersPhone` |
| `EarnMemberPanel.dc.html` | Earn · member panel, past due, refund, complimentary and cancel dialogs | 5.9 | – |
| `EarnImport.dc.html` | Earn · import paid members | 5.10 | – |
| `EarnGifts.dc.html` | Earn · gifts | 5.11 | – |
| `EarnSponsors.dc.html` | Earn · sponsors | 5.12 | `EarnSponsorsTablet`, `EarnSponsorsPhone` |
| `EarnBooking.dc.html` | Earn · booking panel, payment and report sharing | 5.13 | – |
| `ComposerEarnBlocks.dc.html` | Composer · Sponsor and Upgrade blocks, paid-only content and campaign | 5.14 | – |
| `PublicUpgrade.dc.html` | Public · upgrade page | 5.15 | `PublicUpgradePhone` |
| `PublicCancel.dc.html` | Public · cancel page | 5.16 | `PublicCancelPhone` |
| `PublicGift.dc.html` | Public · give a gift and redeem it | 5.17 | `PublicGiftPhone` |
| `PublicSponsorReport.dc.html` | Public · sponsor report | 5.18 | `PublicSponsorReportPhone` |
| `PublicEarnStates.dc.html` | Public · checkout return, offers, already paid, cancel, gift and report states | 5.15–5.18 states | – |

13 app boards and 5 public boards.

### Design-system boards
| Board | Change |
|---|---|
| `DSPatterns.dc.html` | The patterns in section 7 |
| `DSComponents.dc.html` | Money, the new badges, tier cards, the paywall card and gift cards |
| `DSCharts.dc.html` | The MRR movement bridge, and the MRR line with markers |
| `DSCoverage.dc.html` | P6 screens × states |
| `DSAccessibility.dc.html` | The P6 boards in the audit |

### Existing boards to change (section 6)
| Board | Change | Also update |
|---|---|---|
| `Main.dc.html` | Sidebar: Earn after Grow (on every board's sidebar; design it here). Overview: the Earn card | `MainTablet` (rail), `MainPhone` (More sheet) |
| `WorkspaceSettings.dc.html` | Sidebar sections: the Earn switch | `WorkspaceSettingsPhone` |
| `SettingsIntegrations.dc.html` (as reshaped by phase 2) | The Payments group | – |
| `Campaigns.dc.html` | "Paid only" badge and filter, sponsor chip | `CampaignsTablet`, `CampaignsPhone` |
| `Composer.dc.html` | The Earn blocks group, the paid audience chip, "Paid access" visibility, Preview as, the Paid tokens | `ComposerTablet`, `ComposerPhone` |
| `SendReadiness.dc.html` | The paid audience row and the sponsor checks | `SendReadinessTablet`, `SendReadinessPhone` |
| `Report.dc.html` | The Revenue and Sponsor sections | `ReportTablet`, `ReportPhone` |
| `ReportStates.dc.html` | "Results still coming in" for revenue; Revenue without earn:view | – |
| `Contacts.dc.html` | The paid status filter, tier chip and saved view | `ContactsTablet`, `ContactsPhone` |
| `ContactProfile.dc.html` | The Paid subscription card, timeline and actions | `ContactProfilePhone` |
| `SegmentBuilder.dc.html` | The paid subscription conditions | `SegmentBuilderTablet` |
| `PublicPreferences.dc.html` | "Your paid subscription", and the upgrade line for free readers | `PublicPreferencesPhone` |
| `PublicUnsubscribe.dc.html` | The paid member note | – |
| `PublicArchive.dc.html` | Paid-only campaigns left out; issues in their free version | – |
| `Roles.dc.html`, `RoleEditor.dc.html` | The Earn permission group | `RolesTablet`, `RolesPhone` |
| `SettingsAudit.dc.html` | Earn actions and the Earn filter | – |
| `Notifications.dc.html` | Earn kinds and filter tab | `NotificationsTablet`, `NotificationsPhone` |
| `SettingsAPI.dc.html` | The "Paid members" event group | `SettingsAPIPhone` |
| `SettingsKeys.dc.html`, `SettingsKeysStates.dc.html` | The "Conversions key" preset and badge | `SettingsKeysTablet`, `SettingsKeysPhone` |
| `SettingsMCP.dc.html` | Earn tools in the agents' matrix | – |
| `MigrationSource.dc.html` | The line pointing to the paid-members import | `MigrationSourceTablet`, `MigrationSourcePhone` |
| Transactional templates and settings (phase 1 boards) | The "Memberships" category and its 8 starters | their tablet and phone boards |
