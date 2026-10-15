# ReputeCast — Add-on spec: grow (referrals and recommendations)

Phase 5 of `reputecast-design-expansion-plan.md`.

## 0. How to use this document

Read all of it before you start.

- **The prototype from the main spec, the team add-on and phase 1 already exists.**
  - Add the screens in section 5 as a new canvas page, "P5 Grow".
  - Apply section 6 to the existing screens.
  - Reuse the existing navigation, components and design system.
  - Add the patterns in section 7 to the "9 Design system" page first.
- **Same rules as before:**
  - Empty, loading, error and success states, plus the states listed for each screen.
  - Every screen has its read-only variant for members without the permission. It names the permission and who can help.
  - All copy in English.
  - Use the sample data in section 4, together with the main spec's, the team add-on's and phase 1's.
  - Desktop boards at 1440 px. Tablet (834) and phone (390) boards for 5.1, 5.2 and 5.12, and a phone board for 5.6.
  - Public pages at 720 px, plus phone, for 5.9, 5.10 and 5.14. They use the publisher branding from Forms & pages › Branding and never show ReputeCast's teal.
- **Build order:**
  1. 5.1 Grow overview
  2. 5.3 Set up a referral program
  3. 5.4 Milestones and rewards
  4. 5.8 Composer: Referral block and milestone email
  5. 5.9 Referral hub (public)
  6. 5.10 Invite page (public)
  7. 5.6 Review
  8. 5.2 Referral results
  9. 5.12 Recommendations
  10. 5.13 Adding recommendations and partners
  11. 5.14 Recommendations page (public)
  12. The rest, in order
  13. Then section 6

  If the work needs several passes, keep going until every screen exists.
- **Section 9** lists every board to draw and every existing board to change. When the phase is done, add the new boards to DSCoverage and `docs/DESIGN_INVENTORY.md`.

---

## 1. Principles

1. **Everything is free.** No limits on programs, milestones, rewards, referrers, recommendations or partners. No paywall, upgrade prompt, lock icon or plan badge.
2. **Real readers only.** A referral program must never become a way to fill the list with fake or unwilling addresses, because those hurt sender reputation.
   - Only confirmed subscribers count, after a hold period.
   - Obvious abuse is excluded automatically; anything suspicious waits for a person.
   - Partner signups always go through the partner's own double opt-in.
3. **Clear and fair to readers.**
   - The counting rules are written on the referral hub in plain words.
   - A reader always sees their own progress, and why a referral doesn't count yet.
   - An earned reward never disappears.
4. **Private by default.**
   - A referrer never learns who signed up through their link.
   - Names show on invite pages and the leaderboard only when the reader turns that on.
   - No cookies.
   - A partner never learns whether a reader was already its subscriber, and the recommending instance never learns which of its contacts confirmed.
5. **No middleman.** Recommendations link newsletters directly. Partner links connect two ReputeCast instances with a signed handshake. No central service, directory or tracking network sits between them.
6. **Same access rules as the team add-on.**
   - Every action has a permission key; nobody grants a permission they don't hold.
   - Audit entries for every change people make.
   - Partner secrets are sealed and write-only.
   - MCP agents read and propose. They never decide on referrals, change rewards or connect partners.
7. **One editor, one public theme.** The Referral and Recommendations blocks are custom GrapesJS blocks, milestone emails are edited in the composer, and every public page uses the publisher branding.

---

## 2. Terms

- **Referral program:** rewards for subscribers who bring new subscribers to one list. One program per list.
- **Referrer:** a subscriber of the program's list who shares their link.
- **Referral link:** a referrer's personal link, such as `news.acme.io/join/K7Q2MX`. Every confirmed subscriber of the list has one.
- **Referral:** a signup through a referral link. Its status is one of:
  - **Awaiting confirmation:** signed up, hasn't clicked the double opt-in link yet.
  - **In hold:** confirmed, waiting for the hold period to end.
  - **Held for review:** looks suspicious; waits for a person.
  - **Counted:** final. It counts toward the referrer's milestones.
  - **Not counted:** excluded, with the reason.
- **Hold period:** the days a confirmed referral waits before it counts. 7 by default.
- **Milestone:** a number of counted referrals that earns a reward, such as "3 referrals: the Acme Field Guide".
- **Reward:** what a referrer gets at a milestone. Five types: download link, code from a list, manual fulfillment, Stripe coupon, webhook.
- **Milestone email:** the email a referrer gets when they reach a milestone. One per milestone.
- **Referral hub:** the referrer's personal page: their link, progress, rewards and settings. It opens from a personal link, like the preference center.
- **Invite page:** where a referral link leads: the hosted subscribe page, limited to the program's list, with an invite line.
- **Leaderboard:** an optional public page with the top referrers.
- **Review queue:** referrals held for review, grouped by referrer.
- **Recommendation:** a newsletter you recommend to your readers. Two kinds:
  - **Link:** any newsletter's subscribe page. Readers subscribe there.
  - **Partner:** a newsletter on another ReputeCast instance, connected with a partner link. Readers subscribe with one click and confirm through the partner's double opt-in.
- **Partner link:** a signed connection between two ReputeCast instances, one direction at a time. "Acme recommends Ops Weekly" is one partner link; "Ops Weekly recommends Acme" is another.
- **Partner code:** a single-use link that one instance creates for another to set up a partner link. Valid 7 days, shown once.
- **Subscribe request:** what an instance sends a partner when a reader asks to subscribe there: the reader's email address and the recommending newsletter's name, signed.
- **Growth source:** where a contact came from (phase 4). Referrals and each partner are growth sources.

---

## 3. Requirements

### 3.1 Referral program
- **One program per list.** The settings:
  - the list (fixed once the program has been on)
  - status: Off or On
  - the milestones and their rewards (3.3)
  - the counting rules (3.2)
  - the sender of milestone emails: the list's default sender. They go out on the Automations stream.
  - the copy of the invite page and the hub, the share message, and the leaderboard
- **Turning it on** needs at least one milestone, and for each milestone either a milestone email or "No email, I'll use an automation".
- **Turning it off:**
  - Links lead to the normal subscribe page, and nothing new is attributed.
  - Referral blocks are hidden.
  - The hub says the program is paused, and earned rewards stay available.
  - Counted referrals and sent rewards are kept.
- **Referral links:**
  - Every confirmed subscriber of the list gets a link. Subscribers who were already on the list get theirs when the program first turns on.
  - The code is 6 random characters, without look-alikes (no 0, O, 1, I or L). It says nothing about the person.
  - Links live on the public domain: `news.acme.io/join/K7Q2MX`. Embedded forms and the hosted subscribe page also accept `?ref=K7Q2MX`, so a link to the publisher's own site works too.
  - Share buttons add `?via=x`, `?via=linkedin` and so on, so results show where links were shared.
  - A reviewer can pause a link. A paused link leads to the normal subscribe page, and the page doesn't say why.
  - A referrer who leaves the list keeps their counted referrals and rewards. Their link stops attributing.
- **Attribution:**
  - A referral is recorded when someone signs up through a referral link, or through a form opened from one. There are no cookies, so a signup during a later visit isn't attributed.
  - One referrer per signup: the link they signed up through.
  - Only new subscribers can be referrals. An address that is, or ever was, on the list is "Not counted · already on the list".

### 3.2 Counting rules
- **Confirmed only.** A referral starts its hold when the referred person confirms their subscription (double opt-in).
- **Hold period:** 0, 3, 7, 14 or 30 days; 7 by default.
  - When the hold ends, the referral counts if the person is still subscribed and hasn't hard-bounced or complained.
  - Once counted, it stays counted, even if the person leaves later.
- **Not counted, automatically,** each with its reason shown:
  - **Self-referral:** the referrer's own mailbox. Plus-tags, and dots in Gmail addresses, are ignored when comparing. This rule can't be turned off.
  - **Same IP address as the referrer:** the address the referrer used, in the last 30 days, to subscribe or to open their hub or invite page.
  - **Disposable domain:** the same list of throwaway domains the signup forms use.
  - **Left during the hold:** unsubscribed, hard-bounced or complained.
  - **Already on the list:** see 3.1.
  - **Never confirmed:** no confirmation within 7 days. The usual confirmation reminder still goes out after 24 hours.
- **Held for review** when a suspicion rule matches. Defaults, each editable and each can be turned off:
  - **Burst:** more than 5 referrals from one referrer confirmed within an hour.
  - **Shared IP:** 3 or more referrals from the same IP address within 24 hours.
  - **Numbered addresses:** addresses that differ only by a number, such as ben.t01@ and ben.t02@.

  A held referral waits in the review queue (3.5) until a person decides. After a "Count" decision, whatever is left of the hold period still applies.
- **Changes apply going forward.** Changing a rule never takes back a counted referral or a sent reward. Before saving, the settings say how many pending referrals the change affects.

### 3.3 Milestones and rewards
- **A milestone has:**
  - the number of counted referrals
  - the reward's name, a one-line description for readers, and an optional image
  - the reward type and its settings
  - the milestone email, or none
- **Rewards are given once per milestone per referrer, in order.** A referrer who passes two milestones at once (after a review decision, for example) gets both.
- **Reward types:**
  1. **Download link:**
     - Upload a file (PDF, ZIP or image, up to 100 MB), or paste a URL.
     - Uploaded files are served through expiring links, valid 7 days. The hub makes a fresh one on demand.
     - Downloads are counted.
  2. **Code from a list:**
     - Upload codes as CSV or text, one per line. Each reward uses one code.
     - A "Low on codes" alert at a threshold: by default fewer than 20 codes, or less than two weeks at the current pace.
     - When the codes run out, new rewards wait as "Waiting for codes" and go out as soon as codes are added. The hub tells the referrer the reward is on its way.
     - Codes are stored encrypted, shown only to members with grow:review, and never written to the audit log.
  3. **Manual fulfillment:**
     - For anything sent by hand, such as printed stickers or a call with the editor.
     - Optionally, the hub asks the referrer for a shipping address (name, address, country).
     - Rewards wait in a "To fulfill" queue until someone marks them fulfilled, with an optional note the referrer sees, such as a tracking number.
  4. **Stripe coupon** (needs phase 6):
     - Pick an offer of the kind "Referral reward" from Earn › Plans & offers.
     - Each reward creates a single-use promotion code in Stripe, and a claim link to the upgrade page with the code applied.
     - Needs Stripe connected and earn:edit, because it gives away paid access.
     - The claim link works for anyone. A referrer who already pays can pass it on, and the hub says so.
  5. **Webhook:**
     - ReputeCast posts `referral.milestone_reached` to an endpoint you choose, signed like outgoing webhooks.
     - If the endpoint answers with `code` or `url` in its JSON, the milestone email and the hub show it.
     - Failed calls are retried like other webhooks, for 6 hours. Then the reward is "Failed", with Retry.
- **Milestone email:**
  - One per milestone, edited in the composer in "Milestone email" mode. It starts from a default with the Reward block and the Referral block (progress to the next milestone).
  - Sent on the Automations stream from the list's default sender, with the usual suppression rules.
  - It can be turned off for a milestone. The "Reached a milestone" automation trigger fires either way.
- **What the referrer sees on the hub:** each reward with its status: available (download, code or claim link), on its way (waiting for codes, to fulfill), fulfilled, or claimed.

### 3.4 What subscribers see
- **Referral block** (composer):
  - Three layouts: a single line; a card with progress; a card with the next rewards.
  - Content: the personal link and a "Share your link" button, share links (email, X, LinkedIn, WhatsApp), the progress to the next milestone, and "See your rewards", which opens the hub.
  - The default copy reads well for 0, 1 and many referrals. It can be edited with the tokens below.
  - The progress bar is email-safe: table cells, no images.
  - Hidden automatically for recipients who aren't on the program's list, when the program is off, and in the public archive.
- **Liquid tokens:**
  - `referral.link`, `referral.code`, `referral.hub_url`
  - `referral.count` (counted), `referral.pending` (in hold, held and awaiting confirmation)
  - `referral.next_milestone.count`, `referral.next_milestone.reward`, `referral.to_next`
  - `referral.rewards`: the rewards earned, each with `name`, `status` and `url`
  - In milestone emails: `reward.name`, `reward.description`, `reward.download_url`, `reward.code`, `reward.claim_url`, `reward.expires_at`, `reward.note`
  - Tokens are empty for contacts outside the program. The composer's checks warn when a token would be empty for some recipients.
- **Referral hub** (public, personal):
  - Opens from `{{ referral.hub_url }}`. Like the preference center: a signed link, noindex and no-store. An expired link offers "Email me my link".
  - The link, with Copy and the share buttons.
  - Progress: the milestone track, with counted referrals, those in hold (and the date each one counts) and those awaiting confirmation.
  - The rewards earned, each with its status and action.
  - The referrals as numbers and dates only, never names or addresses: "Referral 3 · confirmed 12 Oct · counts on 19 Oct".
  - How counting works, in plain words, generated from the program's rules.
  - Two switches, both off by default: "Show my first name on my invite page" and "Show me on the leaderboard as Sam R.".
- **Invite page:** the hosted subscribe page in invite mode, limited to the program's list.
  - "Sam invited you to the Weekly Digest" when Sam allowed it; otherwise "A reader invited you to the Weekly Digest".
  - The same fields, consent text and spam protection as the hosted subscribe page. Double opt-in is always on for referrals.
  - Under the button: "Sam gets credit when you confirm your email. Sam never sees your address."
  - An unknown or paused code, or a program that's off: the normal subscribe page, without the invite line.
- **Leaderboard** (optional, off by default):
  - The top 10 by counted referrals, this month and all time, updated hourly.
  - Readers who turned it on show as "Sam R."; everyone else shows as "A reader".
  - A referrer sees their own rank on the hub.

### 3.5 Review queue
- **Held referrals are grouped by referrer.** For each referral:
  - signup and confirmation times
  - the address
  - the IP address and its network
  - the signals that held it, in plain words
  - engagement since: opened or clicked anything
- **Decisions:** Count · Don't count · Don't count and pause the referrer's link. Per referral or for a whole group, with an optional note.
- **Before deciding,** the panel says what will happen: "Counting these 3 brings Mateo Silva to 5 when their hold ends on 21 Oct. The Field Guide goes out then."
- **A paused link** can be resumed from the referrer's page.
- **Notifications:** "Referrals held for review" reaches members with grow:review when the queue goes from empty to waiting, then at most once a day while it isn't empty.

### 3.6 Results
- **Period:** 7 days, 30 days or since launch.
- **Funnel:** link visits, then signups, then confirmed, then counted. Alongside: in hold, held for review, and not counted by reason.
- **Over time:** confirmed referrals per day or week.
- **Where links were shared:** visits and signups by channel: copy link, email, X, LinkedIn, WhatsApp, other.
- **Top referrers.**
- **Rewards per milestone:** reached, sent, used (downloaded, claimed or fulfilled), waiting and failed.
- **Quality of referred subscribers** compared with the whole list: human opens, unsubscribes since joining, bounces. Below 100 referred subscribers the card says there's too little data to compare.
- **Visits are counted without cookies:** one per network and browser per day. Bot visits are left out.

### 3.7 Contacts, segments, automations and webhooks
- **Contact profile:**
  - For referrers: a Referrals card with the link and its status, counted and pending referrals, milestones reached and rewards.
  - For referred contacts: "Referred by Sam Rivera", with the referral's status.
  - Timeline events for joining through a link, a referral counted, a milestone reached and a reward sent.
- **Contacts list:** the source filter gains Referral and Partner. The Source column reads "Referral · Sam Rivera" or "Partner · Ops Weekly".
- **Segment conditions:**
  - Referrals counted: at least, at most, or between
  - Referral milestone: reached or not reached
  - Joined through: a referral, or a partner (any, or one)
- **Automation trigger "Reached a milestone":** pick the program, and any milestone or one. A contact enters once per milestone. The milestone and reward tokens are available in the flow's emails.
- **Webhooks:** a "Referrals" event group: `referral.counted`, `referral.held`, `referral.not_counted`, `referral.milestone_reached`, `referral.reward_sent`, `referral.reward_failed`.
- **API:** a key with grow:view can read a contact's referral link and counts, so an app can show them to its users.

### 3.8 Recommendations
- **Grow › Recommendations** lists the newsletters you recommend, in the order readers see them. Each has:
  - a logo, a name and a one-line description (up to 120 characters)
  - a link (any newsletter's subscribe page) or a partner link (3.9)
  - where it shows, and Pause
- **Where they show:**
  - **The opt-in confirmation page:** up to 3, under "While you're here". On by default once a recommendation exists.
  - **The Recommendations block in emails,** usually in the welcome email. The block links to a hosted recommendations page where the reader confirms with a button. Link scanners open links but don't press buttons, so they can't subscribe anyone.
  - Each recommendation can show on the confirmation page, in the block, or both.
- **Consent line** under every Subscribe button: "Subscribing sends your email address to Ops Weekly. They'll email you to confirm." Only the partner's name changes.
- **Numbers per recommendation:**
  - Shown: confirmation pages and delivered emails that showed it.
  - For links: clicks. Signups happen on the other site, so they aren't known.
  - For partners: subscribe requests and confirmations.
- **Recommending you:** the partners that recommend your list, with requests received, confirmations, confirmation rate and the daily limit you gave them. Pause, change the limit, or end.
- **Your partner profile** is what partners see and show: name, logo, description and the list offered. Partner signups always use double opt-in.

### 3.9 Partner links between ReputeCast instances
- **Connecting (once per direction):**
  1. The recommended newsletter (Ops Weekly) creates a partner code in its Grow › Recommendations › "Let a newsletter recommend you". It picks the list offered, a daily limit of subscribe requests (50 by default) and a label for whom it's for. The code is a single-use link valid 7 days, shown once, such as `https://reputecast.opsweekly.example/p/H3RT-6QWN-8ZKB`.
  2. The recommending newsletter (Acme) pastes it in "Add recommendation › Partner on ReputeCast". Acme's instance fetches the partner's profile (name, logo, description, list, daily limit) and shows it before connecting.
  3. **Connect:** the two instances agree a shared signing secret over HTTPS. It's sealed on both sides and never shown. Acme sees the partner under "You recommend", and Ops Weekly sees Acme under "Recommending you".

  A mutual partnership is two codes, one in each direction.
- **A subscribe request:**
  - When a reader presses Subscribe on Acme's confirmation page or recommendations page, Acme's server sends the partner's server a signed request: the email address, Acme's newsletter name as the source, a request ID and a timestamp. Nothing goes in a URL.
  - The partner checks the signature (HMAC-SHA256), the timestamp (within 5 minutes), that the request ID is new, and its daily limit. Then it runs its own double opt-in: a confirmation email from Ops Weekly on its System stream, with the line "You asked for this on the Acme Weekly Digest's page."
  - The partner answers the same way whether or not the address is already its subscriber, so Acme never learns who reads Ops Weekly. An existing subscriber gets nothing.
  - When the reader confirms, the partner sends Acme a signed confirmation for that request ID. Acme keeps only the count per recommendation, never which contact confirmed.
  - On the partner's side, the new contact's growth source is "Partner · Acme Weekly Digest", with the consent evidence of its own double opt-in.
- **Limits and failures:**
  - **Daily limit reached:** the partner answers "limit reached". Acme hides that recommendation until the partner's day resets at 00:00 UTC, and says so in its list.
  - **Partner unreachable:** requests are retried with back-off for 24 hours, then marked failed. The reader is told "We've passed your request to Ops Weekly. Their confirmation email can take a few minutes."
  - **Signature rejected, or the partner ended the partnership:** the link turns Invalid, the recommendation is hidden everywhere, and members with grow:connect are notified. The fix is a new partner code.
  - **Pause or end:** either side can pause (requests stop and the recommendation is hidden) or end (both sides are told over the signed channel). Ending is final; reconnecting needs a new code.
  - **Rotating the secret:** either side can rotate it. The other side receives the new one over the signed channel, and both secrets work for 24 hours.
- **Network:** both instances must reach each other over HTTPS at their public URLs. Partner URLs go through the SSRF guard, so private and loopback addresses are refused.

### 3.10 Growth sources
- Referrals and each partner are growth sources in Audience › Growth (phase 4): "Referral" and "Partner · Ops Weekly", with joins, departures, cohort retention and quality like any other source.
- The Grow overview compares Grow's sources with the list's other sources.

### 3.11 Permissions
New permission group, "Grow":
- view referral and recommendation results (grow:view)
- edit the referral program, its milestones and rewards, and recommendations (grow:edit)
- review held referrals, pause a referrer's link, resend rewards and mark them fulfilled (grow:review)
- connect, pause and end partner links with other instances (grow:connect) ⚑

Who gets what:
- **Built-in roles:**
  - Owner and Admin: all four.
  - Editor: view, edit and review.
  - Viewer: view only.
- **Content Writer**, the sample's custom role: view only. Adding the Referral or Recommendations block to an email needs only campaigns:edit.
- **Combined with other permissions:**
  - Seeing referrers' and referred people's addresses also needs subscribers:view, which every built-in role has.
  - A Stripe coupon reward also needs earn:edit (phase 6).
  - Exporting the referrers list or the "To fulfill" queue needs subscribers:export ⚑ and is audited.
- **API keys:** grow:view lets a key read results, referrers and a contact's referral link.
- **MCP agents:** can read results, referrers' counts and the review queue (with IP addresses masked), and can propose a review decision for a person. They never decide, change rewards or touch partner links.

### 3.12 Audit
- **Audited:**
  - a program created, turned on or off, or its settings changed
  - a milestone or reward created, changed or deleted
  - codes uploaded or removed (the count only, never the codes)
  - a review decision, with the referrer and the referrals it covered
  - a referrer's link paused or resumed
  - a reward resent, marked fulfilled, or its claim link copied
  - the leaderboard turned on or off
  - a recommendation added, changed, paused, resumed or removed
  - a partner code created or revoked
  - a partner link connected, paused, resumed or ended; its secret rotated; its daily limit changed
  - the referrers list or the "To fulfill" queue exported
- **Not in the audit log:** each referral signup, confirmation and automatic count, and each partner subscribe request. The referral log (kept with the program) and the partner request log (90 days) are their record, as the message log is for transactional mail. See section 8.

### 3.13 Privacy and retention
- The hub shows a referrer counts and dates, never who signed up.
- Names show on invite pages and the leaderboard only when the reader turns that on.
- Link visits are counted per day without cookies; the daily key is discarded after 24 hours.
- IP addresses on referrals are kept 90 days for review, then reduced to their network.
- Shipping addresses for manual rewards are deleted 30 days after the reward is marked fulfilled.
- A subscribe request carries only the email address and the source name.
- Erasing a contact (subscribers:delete) removes their link, hub and rewards. Referrals they made stay counted. A referred contact who is erased stays counted for their referrer.
- Nothing in Grow is sent to HostRepute.

### 3.14 Out of scope for this phase
- A central marketplace or directory of newsletters. It would be a hosted service, not part of the open-source app.
- Paying partners per signup.
- Cash rewards and payouts to referrers.
- Shipping integrations (manual fulfillment only).
- Referral programs for transactional mail, SMS or push.

---

## 4. Sample data

"Now" is Wednesday 15 Oct 2026, 10:00 America/New_York, as in phase 1. Weekly Digest #142 went out on Tue 14 Oct.

### Referral program: Weekly Digest
- **Status:** on since Tue 23 Sep 2026, announced in Weekly Digest #139. Sofia Rossi set it up and turned it on.
- **List:** Weekly Digest, 31,988 subscribers, each with a link.
- **Milestone emails:** Automations stream, from Acme Newsletter <digest@news.acme.io>.
- **Counting:** confirmed only, 7-day hold, the automatic exclusions, the suspicion rules at their defaults.
- **Leaderboard:** on since 1 Oct (Sofia Rossi). Names are opt-in.
- **Share message:** "I read the Acme Weekly Digest every Tuesday. Here's my link if you'd like it too: {link}"

### Milestones
| Milestone | Reward | Type | Reached | Sent | Used |
|---|---|---|---|---|---|
| 1 referral | Sticker pack (PDF, 2.4 MB) | Download link | 61 | 61 | 54 downloaded |
| 3 referrals | The Acme Field Guide (PDF, 18 MB) | Download link | 14 | 14 | 13 downloaded |
| 10 referrals | Insider for 3 months | Stripe coupon: offer "Referral reward · 3 months of Insider free" (phase 6) | 2 | 2 | 0 claimed; codes valid 60 days |

### Results since launch (23 Sep – 15 Oct 2026)
- **Link visits:** 2,940. By channel: copy link 1,412, X 604, LinkedIn 512, WhatsApp 233, email 179.
- **Signups:** 214 (7.3% of visits).
- **Confirmed:** 171 (80% of signups). The other 43 never confirmed or are still pending.
- **Of the 171 confirmed:**
  - 139 counted
  - 18 in hold
  - 9 held for review
  - 5 not counted: 3 from the referrer's IP address, 1 disposable domain, 1 unsubscribed during the hold
- **Referrers:** 61 with at least one counted referral, 14 with 3 or more, 2 with 10 or more.
- **Rewards sent:** 77.
- **Quality:** human opens on Weekly Digest #141 were 51.6% for referred subscribers and 44.8% for the whole list. 4 of the 171 have unsubscribed since joining (2.3%).

### Weekly Digest's new confirmed subscribers, last 30 days
| Week | Forms and other sources | Referrals | Partners |
|---|---|---|---|
| 15–21 Sep | 74 | – | – |
| 22–28 Sep | 198 | 31 | 24 |
| 29 Sep – 5 Oct | 186 | 52 | 29 |
| 6–12 Oct | 192 | 58 | 31 |
| 13–15 Oct | 81 | 30 | 18 |
| **Total** | **731** | **171** | **102** |

273 of 1,004 new subscribers (27%) came through Grow.

### Top referrers
| Referrer | Counted | In hold | Held | Milestones | Last referral |
|---|---|---|---|---|---|
| Omar Haddad | 23 | 1 | – | 1 · 3 · 10 (Insider reward not claimed yet) | 13 Oct |
| Kenji Watanabe | 17 | 2 | – | 1 · 3 · 10 (not claimed yet) | 14 Oct |
| Priya Natarajan | 9 | – | – | 1 · 3 | 11 Oct |
| Diego Alvarez | 6 | 1 | – | 1 · 3 | 12 Oct |
| Sam Rivera | 2 | 1 | – | 1 | 12 Oct |
| Mateo Silva | 2 | – | 3 | 1 | 14 Oct |
| Alex Moreau | 1 | – | 6 | 1 | 14 Oct |

- Omar's coupon was issued on 9 Oct and expires on 8 Dec; Kenji's was issued on 13 Oct and expires on 12 Dec.
- **Leaderboard (all time):** Omar H. 23 · Kenji W. 17 · A reader 9 · Diego A. 6, then readers with 4 and 3. Priya Natarajan didn't turn her name on.

### Sam Rivera, referrer
- **Link:** news.acme.io/join/K7Q2MX, created on 23 Sep 2026 when the program turned on. Sam had confirmed his Weekly Digest subscription on 19 Sep, after the Brevo import.
- **Settings:** name on the invite page off; leaderboard off.
- **61 visits and 4 signups:**
  1. nina.patel@gmail.com: signed up and confirmed 26 Sep, counted 3 Oct.
  2. leo.martin@yahoo.com: signed up 1 Oct, confirmed 2 Oct, counted 9 Oct.
  3. ivy.walker@icloud.com: signed up and confirmed 12 Oct, in hold, counts on 19 Oct.
  4. owen.price@outlook.com: signed up 14 Oct, awaiting confirmation; reminder sent 15 Oct.

  The addresses show only to the team. Sam's hub shows "Referral 1" to "Referral 4" with dates.
- **Rewards:** Sticker pack, sent 3 Oct, downloaded 3 Oct.
- **Next:** 1 more counted referral for the Field Guide. Referral 3 brings it on 19 Oct if that subscriber stays.
- **Already a member:** Sam pays for Insider (phase 6 sample). His hub notes that the 10-referral reward can be passed on.

### Review queue: 9 held
- **Alex Moreau** (alex.moreau@gmail.com, Weekly Digest since 2 Feb 2026, 1 counted): 6 referrals signed up on 14 Oct between 21:10 and 21:24.
  - Signals: 6 confirmed within 14 minutes; 4 from 203.0.113.77; numbered addresses ben.t01@outlook.com to ben.t06@outlook.com.
  - Engagement since: none.
  - Suggested decision: "Don't count and pause the link".
- **Mateo Silva** (mateo.silva@gmail.com, 2 counted): 3 referrals signed up on 13 and 14 Oct.
  - Signal: all 3 from 198.51.100.12.
  - Evidence against fraud: different names (a.adeyemi@, t.okoro@ and f.nwosu@ at a company domain), spread over 2 days, and 2 of the 3 opened Weekly Digest #142.
  - They confirmed on 13 and 14 Oct, so their hold ends on 20 and 21 Oct. Counting them brings Mateo to 5 on 21 Oct, and the Field Guide goes out then.

### Recommendations: Acme recommends
| Newsletter | Kind | Shows on | Status | Shown (30 d) | Requests or clicks | Confirmed |
|---|---|---|---|---|---|---|
| Ops Weekly | Partner · reputecast.opsweekly.example | Confirmation page, Welcome email | Connected | 2,322 | 168 requests | 119 (71%) |
| The Deliverability Desk | Partner · mail.deliverabilitydesk.example | Confirmation page | Link invalid since 13 Oct 08:12, hidden | 958 | 97 requests | 64 (66%) |
| Fabrikam Engineering Letter | Link · fabrikam.example/newsletter | Confirmation page, Welcome email | Showing | 2,322 | 96 clicks | – |

- **Shown:** 1,046 confirmation pages and 1,276 delivered welcome emails (Welcome series step 1) in 30 days. The Deliverability Desk shows only on the confirmation page, and only until 13 Oct.
- **Descriptions:**
  - Ops Weekly: "Incidents, on-call and the tools behind them. Every Thursday."
  - The Deliverability Desk: "One deliverability question, answered properly, every other Monday."
  - Fabrikam Engineering Letter: "How Fabrikam's platform team ships. Monthly."
- **The Deliverability Desk's problem:** since 13 Oct 08:12 its instance rejects Acme's signature. They moved to a new server and lost the partnership's secret.

### Recommending you
| Partner | Since | Requests (30 d) | Confirmed | Daily limit | Status |
|---|---|---|---|---|---|
| Ops Weekly | 22 Sep 2026 | 143 | 102 (71%) | 50 | Connected |

- **Acme's partner profile:** "Acme Weekly Digest". "What shipped at Acme, what's next, and a deep dive every Tuesday." List offered: Weekly Digest.
- **History:** Marcus Chen created Acme's partner code for Ops Weekly and pasted Ops Weekly's code on 22 Sep, so the partnership is mutual. He connected The Deliverability Desk on 25 Sep.

### Codes running out (state example only)
The sample program has no code reward. The "codes running out" state shows the same program with its 3-referral reward set to a code list: "Acme store voucher", 200 codes uploaded, 9 left, running out in about 4 days at the current pace.

### People
- **Sofia Rossi** (Editor) runs the program: milestones, rewards, the blocks and the review queue.
- **Marcus Chen** (Admin) connects partners, which needs grow:connect.
- **Daniel Park** (Content Writer) adds the Referral block to Weekly Digest #143. He sees results but can't change the program.
- **Aisha Bello** (Viewer) sees everything read-only.
- **Linh Nguyen** (Owner) can do everything.

---

## 5. New screens

The new sidebar item "Grow" sits after Audience, with three tabs: Overview · Referrals · Recommendations. The Referrals tab has pill tabs of its own: Results · Milestones · Referrers · Review · Settings.

### 5.1 Grow overview (Overview tab)
- **Numbers for the last 30 days:**
  - new subscribers from Grow (273), and their share of the list's new subscribers (27%)
  - referral conversion: visits to confirmed (5.8%)
  - partner signups (102)
  - rewards sent (77)
- **Needs you:** a short list, each item with its action.
  - "9 referrals held for review: Alex Moreau (6), Mateo Silva (3)" → Review
  - "The Deliverability Desk: partner link invalid since 13 Oct 08:12. It's hidden from your pages." → Fix
  - "2 Insider rewards not claimed yet (Omar Haddad, Kenji Watanabe). Codes expire 8 Dec and 12 Dec." → Resend
- **Chart:** Weekly Digest's new confirmed subscribers per week, stacked by forms and other sources, referrals and partners, with a marker at "Referral program started 23 Sep". It links to Audience › Growth.
- **Referral program card:** status and list, the milestone track with how many referrers reached each milestone, the top 3 referrers, and "Open referrals".
- **Recommendations card:** what you recommend and their status, who recommends you, and "Open recommendations".

States to show:
- the sample
- first run, nothing set up: "Grow your list with the people who already read it", with two cards, "Start a referral program" (to 5.3) and "Recommend newsletters you like" (to 5.13). One line: "Partner links connect directly to another ReputeCast instance. There's no marketplace in between."
- only recommendations set up (no program)
- nothing needs you
- read-only (Aisha Bello): no actions; the Needs you items link to read-only views
- loading, and an error in one card while the other loads

### 5.2 Referral results (Referrals › Results)
- **Header:** the program's list and status, with "Turn off" and the period (7 days, 30 days, since launch).
- **Funnel:** visits 2,940 → signups 214 → confirmed 171 → counted 139. In hold (18), held (9) and not counted (5, by reason) branch off.
- **Over time:** confirmed referrals per week.
- **Where links were shared:** visits and signups by channel.
- **Top referrers:** the table from section 4, linking to 5.5.
- **Rewards:** per milestone, reached, sent, used, waiting and failed.
- **Quality:** referred subscribers against the whole list.

States to show:
- the sample
- too little data: "Results show after 20 confirmed referrals"
- program off (the banner from 5.7)
- read-only
- loading, error

### 5.3 Set up a referral program
A guide in five steps. Its progress is kept, so a reload resumes the same step, as in the connect wizard.
1. **List:** pick the list. It shows the subscriber count and "Each confirmed subscriber gets a personal link".
2. **Milestones and rewards:** add milestones, with the suggestion 1, 3 and 10. Each opens the reward editor (5.4).
3. **Counting rules:** the defaults from 3.2, each explained in one line, with the hold period to pick.
4. **Emails and pages:**
   - the milestone email for each milestone, opened in the composer, with "Send me a test"
   - "Add the Referral block to your next issue", which opens the next Weekly Digest draft
   - previews of the invite page and the hub
   - the leaderboard switch (off)
5. **Turn on:** a summary of steps 1 to 4, the checks (a milestone exists, every milestone has an email or "No email", files uploaded, Stripe connected for a coupon reward), and "Turn on the program".

The guide ends with "Your referral program is on", and links to Results and to the next issue.

States to show:
- step 5 with every check passing (the board)
- a check failing: "Insider for 3 months needs Stripe. Connect it in Earn › Settings › Payments, or pick another reward."
- saved and resumed at step 3

### 5.4 Milestones and rewards (Referrals › Milestones)
- **Milestones:** a row per milestone with the count, the reward name and image, the type, reached, sent and used, the reward's health ("6 codes left", "2 failed") and its milestone email (On, Off, or "Uses an automation").
- **Add milestone,** and Edit on each row.
- **Reward editor** (dialog):
  - count, name, description for readers, image
  - type: Download link · Code from a list · Manual fulfillment · Stripe coupon · Webhook, each with one line about when to use it
  - the type's settings:
    - Download link: the file (name, size, uploaded by) or URL
    - Code from a list: upload codes, the count left, the alert threshold
    - Manual fulfillment: whether to ask for a shipping address, and the note to readers
    - Stripe coupon: the offer, its discount and duration, how long codes stay valid
    - Webhook: the endpoint, the signing secret (write-only), "Send a test event", and the answer it gave
  - the milestone email: Edit in composer, Send me a test, or "No email, I'll use an automation"
  - Delete milestone. Rewards already sent stay with their referrers.
- **"To fulfill" queue** (Manual fulfillment): referrer, milestone, date reached, shipping address, "Mark fulfilled" with a note. Export CSV needs subscribers:export.

States to show:
- the sample, with the reward editor open on "Insider for 3 months" (Stripe coupon)
- the type picker
- a code reward running low: "9 codes left. At the current pace they run out in about 4 days." → Upload more codes
- codes run out: "0 codes left · 3 rewards waiting. They go out as soon as you add codes."
- a webhook reward failing: "https://shop.acme.io/hooks/referral answered 502 to the last 8 calls. 2 rewards are waiting." → Retry, Edit endpoint
- the "To fulfill" queue with 3 rewards
- Stripe not connected: the Stripe coupon type is disabled, with "Connect Stripe in Earn › Settings › Payments first"
- Sofia Rossi without earn:edit: "Coupon rewards need earn:edit. Ask Marcus Chen or Linh Nguyen."
- read-only

### 5.5 Referrers (Referrals › Referrers)
- **Table:** referrer and email, counted, in hold, held, not counted, milestones reached, rewards, link status, last referral.
- **Search, and filters:** milestone reached, has held referrals, link paused.
- **Export CSV** needs subscribers:export and is audited.
- **Referrer panel** (side panel, with its own link):
  - the link, its visits and signups, Copy, Pause or Resume
  - progress on the milestone track
  - their referrals, with address, dates and status. Not-counted ones show the reason.
  - their rewards, each with Resend, and "Copy claim link" for coupons (audited)
  - "Open contact"

States to show:
- Sam Rivera's panel (the sample)
- Alex Moreau's panel after the review: "Link paused by Sofia Rossi on 15 Oct. 6 referrals not counted: burst from one IP address." → Resume link
- a reward resent (toast)
- search with no results
- read-only (Daniel Park): no Pause, Resend or Copy claim link; each says why

### 5.6 Review (Referrals › Review)
- **Header:** "9 referrals held for review · the oldest since 13 Oct", filters by signal and referrer, and a link to the suspicion rules.
- **A group per referrer:** the referrer (name, email, on the list since, counted so far), the held referrals with their evidence (3.5), signal chips explained in words, and the group's actions.
- **Decision bar:** Count · Don't count · Don't count and pause the link, an optional note, and the consequence in one sentence.

States to show:
- the sample: Alex Moreau's burst and Mateo Silva's shared office network
- deciding a group, with the consequence: "Counting these 3 brings Mateo Silva to 5 when their hold ends on 21 Oct. The Field Guide goes out then."
- a decision made, with Undo on the toast (for 10 seconds, before any reward goes out)
- empty: "Nothing to review. Suspicious referrals wait here instead of counting."
- Daniel Park (no grow:review): read-only, "Deciding needs grow:review. Ask Sofia Rossi, Marcus Chen or Linh Nguyen."
- phone: groups as stacked cards, with the decision bar fixed at the bottom

### 5.7 Referral settings (Referrals › Settings)
- **Program:** list (fixed), status switch, started on and by.
- **Counting:** hold period; the automatic exclusions, each with one line (self-referral can't be turned off); the suspicion rules, each with its threshold and a switch; the impact line before saving: "18 referrals in hold would count up to 7 days later."
- **Links and pages:** the link format, the invite page's headline and text, the hub's intro, the share message, which share buttons show.
- **Leaderboard:** off or on, periods (this month, all time), how many rows.
- **Milestone emails:** sender and reply-to. The stream is always Automations.
- **Danger zone:** turn the program off; delete it (only when off). Deleting stops the links; counted referrals and sent rewards stay in contacts' timelines.

States to show:
- the sample
- program off: "The referral program is off since 15 Oct, 09:12 (Sofia Rossi). Links lead to the normal subscribe page and Referral blocks are hidden. Earned rewards stay on readers' hubs." → Turn on
- a validation error
- saved
- read-only

### 5.8 Composer: Referral block, milestone emails and the Recommendations block
- **Referral block in Weekly Digest #143** (draft, Tue 21 Oct):
  - The canvas shows the block for "Preview as" Sam Rivera: "You've referred 2 friends. 1 more and you get the Acme Field Guide." with the progress bar, the link and the share links.
  - The inspector: program, layout, which share links show, button text, "Show progress", and a note: "Hidden for recipients who aren't on Weekly Digest, and in the archive."
  - The token menu has a Referral group.
- **Milestone email** for "3 referrals · The Acme Field Guide":
  - The header shows Grow › Referrals › Milestone 3, the sender, "Automations stream", Send test and Save.
  - Subject: "You unlocked the Acme Field Guide". Preheader: `Thanks for sharing the Weekly Digest, {{ first_name | default: "there" }}.`
  - Body: a thank-you line, the Reward block ("Download the Field Guide (PDF, 18 MB)", "The link works for 7 days. You can get a fresh one from your rewards page."), then the Referral block with the progress to 10.
- **Recommendations block in Welcome series step 1:** the cards for Ops Weekly and Fabrikam Engineering Letter, each with a "Subscribe" or "Visit" button that opens the hosted recommendations page (5.14). The inspector picks which recommendations show (all showing, or chosen ones) and the layout.

States to show:
- the three frames above
- the program off: the Referral block shows "The referral program is off. This block will be hidden." on the canvas
- "Preview as" a Product Updates subscriber: the block is hidden, with the reason
- a check: "referral.link is empty for 16,225 recipients who aren't on Weekly Digest"
- every recommendation hidden: "No recommendation can show right now. The block will be hidden."
- Daniel Park adding the block (campaigns:edit only): the block works; "Edit milestones" in the inspector is disabled with the reason

### 5.9 Public: referral hub
Sam Rivera's hub at 720 px and on a phone.
- **Header:** the publisher logo, "Your referrals", "for sam.rivera@outlook.com · not you?".
- **Share card:** "Invite friends to the Weekly Digest", the link field with Copy, and the share buttons.
- **Progress card:** the milestone track (1 · Sticker pack ✓, 3 · Field Guide, 10 · Insider for 3 months), "2 counted · 1 counts on 19 Oct · 1 waiting for confirmation".
- **Rewards card:** "Sticker pack · Download", "Field Guide · 1 more referral", "Insider for 3 months · 8 more. You already have Insider, so you can give this one to a friend."
- **Your referrals:** Referral 1 to 4, with dates and status, and no names.
- **How it works:** "A referral counts when your friend confirms their email and is still subscribed 7 days later. Signups from your own address, from your network, or from throwaway addresses don't count."
- **Privacy:** the two switches, and "Your friends' addresses are never shown to you, and yours is never shown to them."
- **Footer:** Privacy · Email preferences · "Sent with ReputeCast" (if on).

### 5.10 Public: invite page
Sam's link, news.acme.io/join/K7Q2MX, at 720 px and on a phone.
- "A reader invited you to the Acme Weekly Digest" (Sam's name is off).
- The publisher's description of the list, then email, first name (optional), the consent text and "Subscribe".
- Under the button: "We'll send a confirmation link first. The reader who invited you gets credit when you confirm, and never sees your address."

### 5.11 Public: leaderboard
- "Top referrers": This month · All time.
- Rank, name ("Omar H." or "A reader") and counted referrals. Ties share a rank.
- "Names show only for readers who chose to. Updated hourly."
- A reader who arrives from their hub sees their own row highlighted, even outside the top 10.

### 5.12 Recommendations (Recommendations tab)
- **You recommend · 3:** a row per recommendation: logo, name and description, kind (Partner on ReputeCast, with the instance, or Link), where it shows, status, shown, requests or clicks, confirmed. Drag to reorder. A menu: Edit, Pause, Remove.
- **Recommending you · 1:** Ops Weekly, with since, requests, confirmed, rate, daily limit and status. A menu: Change limit, Pause, End partnership.
- **Where recommendations show:** the confirmation page switch (up to 3), and the welcome emails that contain the block (Welcome series step 1).
- **Your partner profile:** name, logo, description and the list offered, with Edit.
- **Actions:** "Add recommendation" and "Let a newsletter recommend you".

States to show:
- the sample, including The Deliverability Desk's invalid link: "Their instance has rejected our signature since 13 Oct 08:12. They may have moved servers or ended the partnership. Hidden from your pages until it's fixed." → "Paste a new partner code"
- read-only (Aisha Bello)
- Sofia Rossi (no grow:connect): she can add and edit link recommendations; partner actions are disabled with "Partner links need grow:connect. Ask Marcus Chen or Linh Nguyen."

### 5.13 Adding recommendations and partners (dialogs)
- **Add recommendation:**
  1. "Is this newsletter on ReputeCast?" → "Yes, I have a partner code" or "No, add a link".
  2. **Link:** name, logo, description, subscribe URL (https), where it shows. ReputeCast can prefill name, description and logo from the page's metadata; that fetch goes through the SSRF guard.
  3. **Partner:** paste the code, then "Checking…", then the partner's profile as it will show (logo, name, description, list, "Double opt-in: always", "Daily limit for your readers: 50", the instance), where it shows, and Connect.
- **Let a newsletter recommend you:** a label (for whom), the list offered, the daily limit. Then the code, shown once, with Copy and "Copied, close". It's valid 7 days and works once.
- **End partnership:** what stops (requests, both lists of partners, the numbers stop growing), that it can't be undone, and that reconnecting needs a new code.

States to show:
- each step above
- partner code expired, already used, not a partner code, or the partner unreachable ("reputecast.opsweekly.example didn't answer over HTTPS")
- the partner's instance doesn't support partner links yet
- connected (toast): "Connected. Ops Weekly now lists Acme Weekly Digest under Recommending you."
- the code shown once
- a recommendation paused
- a partner's daily limit reached: "Ops Weekly isn't taking more partner signups today. Hidden until 00:00 UTC."
- a partner unreachable: "Requests to Ops Weekly have failed for 40 minutes. 6 are waiting and will be retried for 24 hours."
- none yet: "Recommend newsletters you like. They show on your confirmation page and in your welcome email." → Add recommendation
- loading, error

### 5.14 Public: recommendations page
The page the Recommendations block opens, at 720 px and on a phone: news.acme.io/recommends, with Sam's personal link.
- "Newsletters Acme recommends", then a card per showing recommendation: logo, name, description, and Subscribe (partner) or Visit (link).
- The consent line under each Subscribe.
- After Subscribe, the card reads: "Ops Weekly will email sam.rivera@outlook.com to confirm, unless you already get it."

States to show (on the public states board):
- subscribed, as above
- the personal link expired: the page asks for an email address first
- the partner unreachable: "We've passed your request to Ops Weekly. Their confirmation email can take a few minutes."
- no recommendation showing: "Acme isn't recommending other newsletters right now."

---

## 6. Changes to existing screens

- **App shell:**
  - The sidebar gets "Grow" after Audience (and before Earn, phase 6), with a sprout icon.
  - The tablet rail gets the same item. The rail fits 11 items.
  - The phone More sheet (phase 1) lists Grow after Transactional.
- **Settings › General:** a "Grow" switch in Sidebar sections, on by default. Turning it off only hides the item. Programs, links, blocks and partner links keep working.
- **Composer:**
  - The Blocks panel gets a "Grow" group with Referral and Recommendations.
  - The token menu gets a "Referral" group.
  - Inline checks for empty referral tokens, a Referral block while the program is off, and a Recommendations block with nothing to show.
- **Review & send (Send Readiness):**
  - Content: "Referral block: shown to 31,604 of 31,604 recipients" (pass), or "The referral program is off. The Referral block will be hidden." (warn).
  - Content: "Recommendations block: 2 recommendations show. The Deliverability Desk is hidden (partner link invalid)." (pass, with the note).
- **Automations flow editor:** the trigger "Reached a milestone" in the palette, and its side panel (program, milestone, "once per contact per milestone", stats).
- **Contacts:** the source filter gains Referral and Partner; the Source column reads "Referral · Sam Rivera" or "Partner · Ops Weekly".
- **Contact profile:**
  - Sam Rivera gets a Referrals card: the link, 2 counted, 1 in hold, 1 awaiting, the Sticker pack sent, and "Open in Grow".
  - A referred contact shows "Referred by Sam Rivera · counts on 19 Oct".
  - Timeline events: "Joined through Sam Rivera's referral link", "Referral counted (Sam Rivera now has 2)", "Reached milestone 1 · Sticker pack sent", "Asked to subscribe to Ops Weekly (partner)".
- **Segment builder:** the three conditions from 3.7.
- **Public confirmation page:** under the lists, "While you're here: newsletters Acme recommends", with up to 3 cards (Ops Weekly and Fabrikam Engineering Letter in the sample), the consent line and the after-Subscribe state.
- **Public preference center:** an "Invite friends" card for contacts on the program's list while the program is on: "You've referred 2 friends" and "Your link and rewards", which opens the hub.
- **Signup forms:** under the embed code, one line: "Referral links can point here. ?ref= codes are recorded."
- **Campaign report:** the click map groups the Referral block's personal links as "Referral links (personal)" and names the Recommendations block's links by newsletter.
- **Roles & permissions:** the Grow group (3.11) in the role editor and the compare matrix.
- **Audit log:** the actions from 3.12, and a "Grow" filter category.
- **Notifications:**
  - New kinds: referrals held for review (grow:review); reward codes running low or out, and a reward webhook failing (grow:edit); a partner link invalid or unreachable (grow:connect).
  - A "Grow" filter tab.
- **Webhooks:** the "Referrals" event group. An endpoint used by a webhook reward shows "Used by a referral reward".
- **MCP server:** new tools in the agents' permission matrix: get_referral_results, list_referrers, list_held_referrals (IP addresses masked), propose_referral_decision, list_recommendations.
- **Audience › Growth (phase 4):** Referral and each partner are sources.

---

## 7. New design-system patterns

Add these to the "9 Design system" page before the screens. Use the existing tokens and badge colors.

- **Milestone track:** a horizontal track with a dot per milestone and the reward's name under it, filled up to the counted referrals, with "counting soon" referrals in a lighter fill and a "you are here" marker.
  - In the app it uses `accent`; on public pages `publisher-accent`.
  - The email version is table cells, with no images.
- **Referral status badges:**
  - Awaiting confirmation: neutral
  - In hold: info, with the date it counts
  - Held for review: warn
  - Counted: ok
  - Not counted: neutral, with the reason
- **Reward status badges:** Sent: ok · Used (downloaded, claimed or fulfilled): ok · To fulfill: info · Waiting for codes: warn · Failed: bad · Expired: neutral.
- **Signal chips** for the review queue: a short label ("Burst · 6 in 14 min", "Same IP ×4", "Numbered addresses") whose full sentence is on hover and focus. Neutral; the group's suggested decision sits in a warn note.
- **Share row** (public pages): the link field with Copy, and labelled share buttons. Plain share links: no third-party scripts, no trackers.
- **Recommendation card:** logo, name, one-line description, the Subscribe or Visit button and the consent line.
  - Public version (publisher accent) and email-safe version.
  - States: default, requested, hidden.
- **Partner status badges:** Connected: ok · Paused: neutral · Daily limit reached: warn · Unreachable: warn · Link invalid: bad.
- **Leaderboard row** (public): rank, name or "A reader", count. The viewer's own row is highlighted.

---

## 8. Conflicts to note

1. **Grow's tabs.** The plan lists Referrals · Recommendations. This brief adds an Overview tab as Grow's landing page.
2. **Audit log vs referral log.** As in phase 1, automatic events (each signup, confirmation, automatic count and partner request) stay in the referral log and the partner request log, not the audit log. Every change a person makes is audited. The owner needs to confirm this.
3. **Counting rules.** The plan names confirmed only, a hold period, the same IP and disposable domains. This brief adds self-referral, already on the list, left during the hold and never confirmed as automatic exclusions, and three suspicion rules that hold referrals for review.
4. **Disposable domains.** Signup forms already refuse them, so this rule mostly catches signups through the API and domains added to the list after a signup.
5. **Phase order.** The Stripe coupon reward needs phase 6, which comes after this phase. Until payments exist, the reward type shows "Connect Stripe" and can't be picked. Engineering may ship it with the first phase 6 milestone.
6. **Recommendations page.** The plan puts recommendations on the confirmation page and in a welcome-email block. This brief adds a hosted recommendations page behind the block, so link scanners can't subscribe readers to partners.
7. **Partner privacy.** The plan says the new subscriber is attributed to a growth source. That happens on the partner's side. The recommending instance keeps only counts, never which contact confirmed.
8. **Partner codes.** The plan says two instances "exchange a signed cross-subscribe link". This brief sets that up with a single-use partner code and a shared signing secret, one per direction.
9. **Leaderboard.** Optional in the plan; here it's off by default and names are opt-in.
10. **Sample date.** The sample's "now" is Wed 15 Oct 2026, as in phase 1, so Weekly Digest #142 has gone out. The existing boards keep their dates.
11. **"Shown" for recommendations.** It counts confirmation pages and delivered welcome emails. Opens aren't used, because machine opens would inflate them.

---

## 9. Boards

All new boards go on the canvas page "P5 Grow". Desktop boards are 1440 px wide; public boards 720 px. Tablet and phone boards are wrappers that render the same markup at 834 and 390 px, like the existing ones.

### New boards
| Board | Canvas title | Implements | Tablet and phone |
|---|---|---|---|
| `GrowOverview.dc.html` | Grow · overview | 5.1 | `GrowOverviewTablet`, `GrowOverviewPhone` |
| `GrowStates.dc.html` | Grow · first run, recommendations only, read-only, loading, error | 5.1 states | – |
| `Referrals.dc.html` | Referrals · results | 5.2 | `ReferralsTablet`, `ReferralsPhone` |
| `ReferralSetup.dc.html` | Referrals · set up a program | 5.3 | – |
| `ReferralMilestones.dc.html` | Referrals · milestones and reward editor | 5.4 | – |
| `ReferralReferrers.dc.html` | Referrals · referrers and Sam Rivera's panel | 5.5 | – |
| `ReferralReview.dc.html` | Referrals · review queue | 5.6 | `ReferralReviewPhone` |
| `ReferralSettings.dc.html` | Referrals · settings | 5.7 | – |
| `ReferralStates.dc.html` | Referrals · program off, codes low and out, webhook failing, to fulfill, Stripe not connected, link paused, read-only | 5.2–5.7 states | – |
| `ComposerGrowBlocks.dc.html` | Composer · Referral block, milestone email, Recommendations block | 5.8 | – |
| `Recommendations.dc.html` | Recommendations · you recommend and recommending you | 5.12 | `RecommendationsTablet`, `RecommendationsPhone` |
| `RecommendationStates.dc.html` | Recommendations · add, partner code, end, paused, limit, unreachable, empty | 5.12–5.13 dialogs and states | – |
| `PublicReferralHub.dc.html` | Public · referral hub | 5.9 | `PublicReferralHubPhone` |
| `PublicReferralInvite.dc.html` | Public · invite page | 5.10 | `PublicReferralInvitePhone` |
| `PublicRecommendations.dc.html` | Public · recommendations page | 5.14 | `PublicRecommendationsPhone` |
| `PublicGrowStates.dc.html` | Public · leaderboard, hub states, invite fallback, recommendation states, emails as received | 5.9–5.11, 5.14 states | – |

12 app boards and 4 public boards. PublicGrowStates also shows the milestone email and Ops Weekly's confirmation email as a reader receives them.

### Design-system boards
| Board | Change |
|---|---|
| `DSPatterns.dc.html` | The patterns in section 7 |
| `DSComponents.dc.html` | The new badges, signal chips and the leaderboard row |
| `DSCoverage.dc.html` | P5 screens × states |
| `DSAccessibility.dc.html` | The P5 boards in the audit |

### Existing boards to change (section 6)
| Board | Change | Also update |
|---|---|---|
| `Main.dc.html` | Sidebar: Grow after Audience (the item appears on every board's sidebar; design it here) | `MainTablet` (rail), `MainPhone` (More sheet) |
| `WorkspaceSettings.dc.html` | Sidebar sections: the Grow switch | `WorkspaceSettingsPhone` |
| `Composer.dc.html` | Blocks panel: the Grow group; token menu: Referral | `ComposerTablet`, `ComposerPhone` |
| `SendReadiness.dc.html` | Content rows for the Referral and Recommendations blocks | `SendReadinessTablet`, `SendReadinessPhone` |
| `AutomationFlow.dc.html` | Trigger "Reached a milestone" and its panel | `AutomationFlowTablet`, `AutomationFlowPhone` |
| `Contacts.dc.html` | Source filter and Source column | `ContactsTablet`, `ContactsPhone` |
| `ContactProfile.dc.html` | Referrals card, "Referred by", timeline events | `ContactProfilePhone` |
| `SegmentBuilder.dc.html` | The referral conditions | `SegmentBuilderTablet` |
| `PublicConfirm.dc.html` | "While you're here" recommendations | – |
| `PublicPreferences.dc.html` | "Invite friends" card | `PublicPreferencesPhone` |
| `FormBuilder.dc.html` | The `?ref=` line under the embed code | `FormBuilderTablet`, `FormBuilderPhone` |
| `Report.dc.html` | Click map: referral and recommendation links | `ReportTablet`, `ReportPhone` |
| `Roles.dc.html`, `RoleEditor.dc.html` | The Grow permission group | `RolesTablet`, `RolesPhone` |
| `SettingsAudit.dc.html` | Grow actions and the Grow filter | – |
| `Notifications.dc.html` | Grow kinds and filter tab | `NotificationsTablet`, `NotificationsPhone` |
| `SettingsAPI.dc.html` | The Referrals event group | `SettingsAPIPhone` |
| `SettingsMCP.dc.html` | Grow tools in the agents' matrix | – |
| Audience › Growth (phase 4 board) | Referral and partner sources | its tablet and phone boards |
