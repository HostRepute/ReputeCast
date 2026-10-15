# ReputeCast — Add-on spec: publish and sell

Phase 7 of `reputecast-design-expansion-plan.md`: the publication website, landing and sales pages, digital products, and Shopify and WooCommerce.

## 0. How to use this document

Read all of it before you start.

- **The prototype from the main spec, the team add-on and the earlier phases already exists.**
  - Add the screens in section 5 as a new canvas page, "P7 Website, pages & commerce".
  - Apply section 6 to the existing screens.
  - Reuse the existing navigation, components and design system.
  - Add the patterns in section 7 to the "9 Design system" page first.
- **This phase builds on:**
  - Phase 1: product downloads are delivered by a transactional template on the Transactional route.
  - Phase 2: Shopify and WooCommerce join the Commerce group of the Integrations hub. WooCommerce needs the ReputeCast WordPress plugin for consent and abandoned checkouts.
  - Phase 3: the Visibility section of the composer's inspector decides which blocks show on the web.
  - Phase 4: growth sources count signups and sales by page, referrer and UTM.
  - Phase 5: the referral hub, invite page, leaderboard and recommendations page join the built-in pages.
  - Phase 6: Stripe (the "Acme Media" account, live), the paid tiers, the upgrade page, paid status on contacts, and the Revenue section of reports.
  - Where an earlier phase's brief names something differently, follow that brief and note the difference.
- **Same rules as before:**
  - Empty, loading, error and success states, plus the states listed for each screen.
  - All copy in English.
  - Use the sample data in section 4, together with the main spec's, the team add-on's and the earlier phases'.
  - Desktop boards at 1440 px. Tablet (834) and phone (390) boards for 5.1, 5.9, 5.12 and 5.15. A phone board for 5.3 and a tablet board for 5.10.
  - Public pages at 720 px, plus phone: 5.6, 5.7, 5.8 and 5.11.
- **Build order:**
  1. 5.1 Website › Posts
  2. 5.3 Publish to web
  3. 5.7 Public post, with the paywall
  4. 5.5 Website › Access
  5. 5.8 Public sign-in and account
  6. 5.6 Public home
  7. 5.4 Website › Home & navigation
  8. 5.2 Turning the website on
  9. 5.9 to 5.11 Pages
  10. 5.12 to 5.14 Products
  11. 5.15 and 5.16 Stores
  12. Then section 6

  If the work needs several passes, keep going until every screen exists.

---

## 1. Principles

1. **Everything is free.** No limits on posts, pages, members, products, orders or stores.
   - The paywall and the prices are the publisher's tools for charging their readers through their own Stripe account.
   - ReputeCast takes no fee and adds no badge. "Sent with ReputeCast" in the footer stays optional.
2. **The website grows out of the archive.**
   - Turning it on keeps every existing link working. Turning it off brings back the archive.
   - Nothing is behind a paywall until someone sets a post's visibility.
3. **Write once.**
   - A campaign becomes a post with one switch. A web-only post can be emailed later.
   - One editor in three modes (email, post, page) and one public theme: the branding from Forms & pages.
4. **Gate on the server.**
   - Text a reader may not see never reaches their browser, the feed or the search index. It isn't hidden with CSS; it isn't sent.
   - Search engines see what an anonymous visitor sees.
5. **Consent stays specific.**
   - Members, buyers and store customers join a list only by a choice they made for that list. The page, time, IP address and exact text are kept, as for signup forms.
   - Buying something or having a store account never subscribes anyone by itself.
6. **The money stays with the publisher.** Payments go through the publisher's Stripe account and their store. ReputeCast never sees card details. Refunds and disputes happen in Stripe and show up here.
7. **Privacy-first numbers.**
   - Views are counted on the server, without cookies.
   - ReputeCast doesn't track what members read, beyond the meter's count of gated posts.
8. **Same access rules as the team add-on.**
   - Every action has a permission key. Publishing can need approval.
   - Every change is audited.
   - MCP agents draft and propose. They never publish, change a price or refund.

---

## 2. Terms

- **Website:** the publisher's public site at their custom domain (news.acme.io). It grows out of the public archive and uses the branding from Forms & pages.
- **Post:** an article on the website.
  - **Post from a campaign:** a campaign published to the web with "Publish to web". It keeps a link to its campaign.
  - **Web-only post:** written in the composer in post mode and never sent. It can be emailed later as a campaign.
- **Visibility:** who can read a post or a page.
  - **Public:** everyone, including search engines.
  - **Members:** anyone signed in.
  - **Paid:** members with a paid tier from Phase 6: every tier, or chosen tiers.
- **Member:** a contact with a confirmed email address who can sign in to the website. Being a member doesn't mean receiving newsletters: a member can be on no list.
- **Sign-in link:** an email with a link and a 6-digit code that signs a member in. No passwords.
- **Free preview marker:** a block placed in a post. Readers without access see the post down to the marker, then the paywall card.
- **Paywall card:** what replaces the rest of a gated post. It offers the tiers, a free sign-up where the meter allows it, and sign-in.
- **Meter:** a number of gated posts a reader can open for free each calendar month.
- **Page:** a web page that isn't a post.
  - **Built-in pages:** the pages ReputeCast provides: subscribe, confirmation, preference center, unsubscribe, sign in, account, archive and shop, plus the public pages of phases 5 and 6 (referral hub, invite page, leaderboard, recommendations, upgrade, gift).
  - **Custom pages:** pages built with blocks: About, landing pages and sales pages.
- **Page template:** a starting layout for a custom page: About, Lead magnet, Launch, Webinar registration, Sales page or Blank.
- **Lead magnet:** a free file offered in exchange for subscribing. It's delivered after the subscription is confirmed.
- **Digital product:** a file plus a price, sold through Stripe Checkout.
- **Order:** one purchase of a digital product, with its buyer, payment and downloads.
- **Download link:** a personal link to a product's files. It expires after a number of days and a number of downloads.
- **Store:** a Shopify or WooCommerce shop connected through the Integrations hub.
- **Store customer:** a customer of a connected store. Only customers who accepted marketing join a list.
- **Abandoned checkout:** a store checkout with an email address that didn't become an order within the set time.
- **Attributed order:** a store order or product sale credited to the email whose link the buyer clicked last, within the attribution window.

---

## 3. Requirements

### 3.1 Website
- **Where:** Forms & pages keeps its name. Its tabs become Signup forms · Website · Pages · Branding & domain (section 6).
- **Turning it on:** Forms & pages › Website › "Turn on the website". Needs website:manage.
  - The root of news.acme.io becomes the home page. /subscribe stays where it is.
  - Every campaign in the archive becomes a public post. Old archive links redirect (301) to the posts.
  - Each list with archived campaigns becomes a tag, for example Weekly Digest and Product Updates.
  - Members can sign in. The header gets Subscribe and Sign in.
  - The archive stays at /archive as the full, searchable list of posts.
- **Turning it off:** a typed confirmation that lists what goes offline: the home page, web-only posts, sign-in, the account page, the paywall, and pages whose visibility is Members or Paid.
  - Paid subscriptions continue, and paid members keep getting their emails.
  - Public pages and the shop stay online, with the logo in place of the site's navigation.
  - The archive comes back. Every post and page is kept for when the website is turned on again.
- **Pages don't need the website.** A workspace can publish landing pages, sales pages and the shop with the website off. Pages with Members or Paid visibility need sign-in, so they need the website.
- **Addresses:**
  - home `/`, posts `/posts/{slug}`, tags `/tag/{slug}`, archive `/archive`, custom pages `/{slug}`, shop `/products`, sign in `/signin`, account `/account`
  - feeds: `/feed` for the site and `/tag/{slug}/feed` per tag. The archive's existing RSS address keeps working.
  - sitemap: `/sitemap.xml`
  - Until the custom domain is verified, the site runs at reputecast.acme.io/p/acme-newsletter, and the screens say so.
- **Posts:**
  - Fields:
    - title: a campaign's subject by default
    - slug: lowercase letters, digits and hyphens, unique. Earlier slugs redirect.
    - excerpt: up to 200 characters, the preheader by default
    - cover image, with alt text required
    - byline: a member or a written name
    - up to 5 tags
    - visibility and the free preview marker (3.3)
    - "Feature on home"
    - SEO and social: SEO title, meta description, social image, canonical URL, "Hide from search engines"
  - Statuses: Draft, Scheduled, Published, Unpublished.
  - **Unpublishing** takes a post off the home page, tags, archive, feeds and sitemap. Its address then shows "This post is no longer available" (410), or redirects to a chosen post or page.
  - **Numbers per post:** views, signups (members who joined from the post) and upgrades to a paid tier, with their top referrers and UTM sources (phase 4). For 30 days and all time.
- **Posts from campaigns ("Publish to web"):**
  - When: with the send (default), or at a set time. A campaign that was already sent can be published now.
  - The web version uses the email's blocks, drawn in the site's theme: its fonts and a 680 px reading column, not the email's tables.
  - Personalization uses each token's fallback. "Hi {{ first_name | default: "there" }}" reads "Hi there".
  - Blocks set to "Email only" in their Visibility section are left out. Footer, unsubscribe, view-online and referral blocks are Email only by default.
  - Conditional blocks (phase 3) are hidden on the web, except blocks for a paid tier: those show to readers who hold that tier, and the paywall card takes their place for everyone else.
  - Approving a campaign (team add-on 3.8) also covers its web settings: slug, visibility and when it's published.
- **Web-only posts:**
  - Written in the composer in post mode: the same blocks and panels, with no audience, subject or route. The canvas shows the web version.
  - "Send as a campaign" creates a campaign draft from the post, with a link back to it.
- **Tags:** name, slug, description, "Show in navigation", and their own page and feed.
- **Home page:**
  - Three layouts: Featured and latest, Latest as a list, Grid.
  - Up to 3 featured posts.
  - Sections to show or hide: latest posts (how many), a row per chosen tag, the about text, a subscribe band, and Phase 6's paid tiers band.
- **Navigation:**
  - Header: up to 6 links to the home page, a tag, a page, the archive or an outside address. On phones they fold into a menu.
  - Footer: archive, RSS, privacy policy, email preferences, and "Sent with ReputeCast" if it's on.
  - Subscribe and Sign in are always in the header.
- **Search engines and feeds:**
  - Site title, tagline, description, social image and favicon.
  - "Let search engines index the site", on by default.
  - Gated posts appear in feeds and the sitemap with their preview only.
  - Gated posts are marked as not free in their structured data, so a search engine doesn't treat the preview as cloaking.
- **Views** are counted on the server without cookies: one view per visitor, per post, per day, from a hash with a salt that changes daily. Nothing that identifies the visitor is stored.
- **Speed and scripts:** pages are rendered on the server and work without JavaScript. Public pages can be cached; pages for a signed-in reader are never cached.

### 3.2 Members and sign-in
- **Who can sign in:** any contact with a confirmed address.
  - A contact who unsubscribed from every list can still sign in. They read on the web and get no newsletters.
  - A contact still pending double opt-in becomes confirmed by clicking a sign-in link.
  - An address suppressed for a hard bounce or a complaint gets no link. The page answers the same way as for everyone else, and the attempt shows on the contact's timeline.
- **Signing in:**
  - The reader enters their email. The page always answers the same way: "If sam.rivera@outlook.com belongs to a member, a link and a code are on their way." It never says whether an address is a member.
  - The email has a button and a 6-digit code. Both work for 30 minutes, once. A newer link replaces older ones.
  - The code is for a link that opens in another browser or device: the reader types it on the page where they asked.
  - Opening the link signs the reader in with a script. Without scripts it shows a Continue button. Link scanners that fetch the address don't use the link up.
  - Limits: 5 links per address per hour, plus per-IP limits. Turnstile shows only when requests look automated, as on signup forms.
  - Sign-in emails go through the System stream, from the workspace's default sender, in the contact's language (phase 2).
- **Signing up on the site:**
  - Where: the paywall card, the Sign in page ("Not a member yet?") and Subscribe.
  - The lists offered, and which are checked by default, are set in Website › Access. Each list's consent text is shown and kept.
  - One email, "Confirm your email and continue reading", confirms the subscription (double opt-in) and signs the reader in.
- **Sessions:** 90 days by default, from 7 to 365. A first-party cookie on the site's domain. The account page has "Sign out" and "Sign out everywhere".
  - People with subscribers:edit can sign a member out everywhere from the contact profile. It's audited.
- **Account page** (`/account`):
  - Email (changed by a confirmation link sent to the new address) and name.
  - Newsletters: the preference center's lists, frequency and pause.
  - Plan (phase 6): the tier, price and renewal date, "Manage billing" (Stripe's customer portal), and Upgrade or Change plan.
  - Gifts (phase 6): gifts given and received.
  - Purchases: the digital products bought, with "Download" or "Send me a new link".
  - Referrals (phase 5): the member's link and progress, opening their referral hub.
  - The preference center stays for links in emails. When the reader is signed in, it links to the account page.

### 3.3 Visibility, free preview and meter
- **Visibility per post:** Public, Members, or Paid (every tier, or chosen tiers). Pages take the same values.
- **Defaults:**
  - Web-only posts and pages: Public.
  - Posts from campaigns follow the audience: a paid-only campaign (phase 6) becomes Paid.
- **Free preview marker:**
  - At most one per post. In the composer it's a block, "Free preview ends here", shown as a dashed line with its label.
  - Readers without access see the post down to the marker, then the paywall card.
  - Without a marker, they see the title, cover and excerpt.
  - In the email the marker does nothing: the audience and paid-only blocks decide who gets what.
- **What each reader sees on a gated post:**
  - A visitor who isn't signed in: the preview and the paywall card, with Sign in.
  - A free member: the preview and the paywall card with the tiers. Where the meter has a post left, the whole post with a note.
  - A member with the right tier: the whole post. A quiet label names the tier, for example "Insider".
- **Meter:** two rules, both off by default.
  - "Visitors who aren't signed in can read [N] members-only posts a month." Counted per browser with a first-party cookie, which is enough for a soft limit.
  - "Free members can read [N] paid posts a month." Counted per member.
  - Months are calendar months. A post counts once; reading it again is free.
  - A post can be excluded from the meter ("Never free"), for example a members' AMA.
  - The note on a metered post: "This is your free paid post for October. Upgrade to read them all."
- **Paywall card:** heading, text, the tiers to show with their prices, button labels, and a free sign-up row when the meter allows it.
  - Edited in Website › Access, with a preview for each kind of reader.
  - No lock icon. Lock icons are kept out of the product.
- **Search engines get no special access.**

### 3.4 Pages
- **Pages tab** (Forms & pages › Pages) lists two groups:
  - **Your pages:** name, address, template, status, visibility, views, and the result: signups, registrations or sales.
  - **Built-in pages:** subscribe, confirmation, preference center, unsubscribe, sign in, account, archive and shop, plus the public pages of phases 5 and 6 (referral hub, invite page, leaderboard, recommendations, upgrade, gift). Each opens the editor it already has, with options on the left and a live preview on the right. Sign in and account appear while the website is on.
- **Page builder:** the composer's interface in page mode.
  - The same panels, autosave, versions and publish dialog as the composer.
  - The blocks are web sections on the publisher's theme. Styles are limited to the theme's colors, fonts and spacing, so pages stay on brand and pass contrast.
  - Device preview: desktop, tablet, phone.
  - Page settings: title, slug, SEO title and description, social image, visibility, "Hide the site header and navigation" (for focused landing pages), "Show in navigation", and a publish time.
  - Statuses: Draft, Scheduled, Published, Unpublished. Unpublished pages answer like unpublished posts.
- **Blocks:**
  - Layout: section, 2 or 3 columns, divider, spacer.
  - Content: hero (heading, text, image, button), text, image (alt text required), button, video, features list, testimonials, FAQ, author bio, recent posts (from a tag).
    - Video blocks (YouTube or Vimeo) load nothing from the video site until the reader clicks. The block names the site it will load from.
    - The FAQ opens and closes without scripts.
  - Conversion:
    - signup form: one of the workspace's forms
    - countdown
    - event details
    - product: one digital product with its Buy button
    - product grid: digital or store products
    - store product: one Shopify or WooCommerce product (3.6)
    - pricing tiers: Phase 6's tier cards
- **Countdown:** to a date and time in a time zone. When it ends, it hides, shows a message, or reveals another block (for example the Buy button).
- **Event details:** date, start and end time in the event's time zone, also shown in the reader's own time zone. Online (the joining link goes out after registering) or a venue address. "Add to calendar" for Google, Outlook and Apple (.ics), after registering.
- **Templates:**
  - **About:** hero, text, author bios, subscribe band.
  - **Lead magnet:** hero, benefits, a signup form whose form offers a download after confirmation (section 6, Form builder), testimonials, FAQ.
  - **Launch:** hero with a countdown, a waitlist form, features, FAQ. When the countdown ends, the Buy button replaces the waitlist form.
  - **Webinar registration:** hero, event details, speakers, agenda, a registration form.
    - The form adds registrants to the event's list, with consent text about that event only, and writes the event's date and time to a date field.
    - The template offers to create the reminder automation: a date-field trigger 1 day and 1 hour before the start, with the joining link.
  - **Sales page:** hero with the product and a Buy button, what's inside, author bio, testimonials, FAQ, a second Buy button. On phones a Buy bar stays at the bottom of the screen.
  - **Blank.**
- **Shop** (`/products`): a built-in page listing every published product. Each card leads to the product's sales page, or to its default product page when it has none.
- **Numbers per page:** views, visitors, signups (confirmed), registrations, sales and revenue, and the conversion rate, by referrer and UTM (phase 4).
- **Publish checks:**
  - Blocks:
    - a slug already taken
    - a signup form block whose form is paused or deleted
    - a product block for a product that isn't published
    - a Buy button while Stripe isn't connected
  - Warnings:
    - missing alt text
    - a countdown or event in the past
    - a broken link
    - no heading at the top of the page

### 3.5 Digital products
- **Where:** Earn › Products, a new tab after Sponsors (expansion plan, section 4).
- **A product has:**
  - name, slug, a short description for cards, a long description for the default product page, a cover image with alt text
  - one or more files: up to 500 MB each and 2 GB per product. The file type is checked from the file's content, not only its name.
  - a price:
    - **One-time:** for example $29.
    - **Pay what you want:** a minimum of at least $1 and a suggested price.
    - The currency is the Stripe account's. Stripe Tax follows the Phase 6 payments setting.
  - selling: the sales page (one of the workspace's pages, or the default product page) and "Show in the shop"
  - an optional list offer at checkout: "Also send me the Weekly Digest". Unchecked by default; the exact text is kept as consent.
  - delivery: the link's validity (1 to 30 days, default 7), downloads per link (1 to 20, default 5), and the delivery template
  - "Revoke access when a payment is refunded or disputed", on by default
  - status: Draft, Published, Archived. An archived product can't be bought; its buyers keep their downloads.
- **Publish checks:**
  - Stripe is connected. In test mode, the product and its orders carry a Test badge.
  - At least one file finished uploading.
  - A price is set.
  - The delivery template is published and the Transactional route exists.
  - The cover has alt text.
  - The chosen sales page is published.
- **Buying:**
  - A Buy button creates a Stripe Checkout session for the product: a one-time payment, or a customer-chosen amount with the minimum and suggested price.
  - For a signed-in member, the email is filled in.
  - The list offer's checkbox sits on ReputeCast's page, next to the Buy button, because Stripe's page can't carry it. Its state, the page, the time and the IP address travel with the session.
  - When Stripe confirms the payment (checkout.session.completed), the order becomes Paid and the delivery email goes out.
  - Back from Stripe, the thank-you page has "Download now" and says where the link was emailed.
- **Delivery:**
  - A new Phase 1 starter template, "Product delivery" (`product-delivery`), in a new Essential category, "Purchases".
  - It goes through the Transactional route. Its `download_url` is a sensitive variable, so it's never stored.
  - The download page (`/d/{token}`) shows the files, their sizes, the link's expiry and the downloads left.
  - An expired or used-up link offers "Send a new link". The new link goes only to the address on the order, and the page shows that address masked.
- **Buyers and contacts:**
  - A buyer whose email matches a contact is linked to that contact.
  - A buyer who ticks the list offer becomes a contact on that list, following the list's double opt-in.
  - Any other buyer is not a contact. The order keeps their email for delivery and support only.
- **Orders:**
  - Statuses: Paid, Refunded, Partially refunded, Disputed (open, won or lost), Test.
  - Each order has its number, time, buyer, product, amount, a link to the payment in Stripe's dashboard, the delivery email's status (linked to Transactional › Messages), downloads used, the link's expiry and the consent given at checkout.
  - Actions:
    - "Send a new link", to the buyer's address only
    - "Revoke access" and "Restore access"
    - "Refund in Stripe", which opens the payment in Stripe's dashboard
  - Refunds and disputes arrive from Stripe's webhooks. The status updates, access is revoked if the setting says so, the contact's timeline records it, and a dispute sends a notification.
  - Exporting orders needs subscribers:export, because they hold addresses, and is audited.
- **"Bought a product" trigger:** for any product or a chosen one. Only contacts enter. Its data is available in Liquid (`{{ order.product_name }}`, `{{ order.amount }}`), and a refund makes the contact leave the flow.
- **Stripe's fees are Stripe's. ReputeCast takes nothing.**

### 3.6 Shopify and WooCommerce
- **Where:** the Integrations hub's Commerce group (Settings › Integrations), with capability badges:
  - Orders, Products, Customers with consent, Abandoned checkouts, Consent write-back, Attribution.
  - A workspace can connect several stores. Each has its own detail page and settings.
- **Connecting Shopify** (a custom app in the merchant's own Shopify account):
  1. The store's address: shop.acme.io or acme-merch.myshopify.com.
  2. Create the app in Shopify's Dev Dashboard, with values to copy:
     - the app URL and redirect URL, both on the instance's public URL
     - the access it needs: read customers, orders and products; write customers is optional, to pass unsubscribes back
     - protected customer data: name and email
  3. Paste the app's client ID and client secret. The secret is write-only.
  4. "Install on acme-merch.myshopify.com" opens Shopify's own approval page, which returns to ReputeCast.
  5. Choose what to sync (below), then "Start sync".
- **Connecting WooCommerce:**
  1. The store's address.
  2. A REST API key with Read/Write access, created in WooCommerce › Settings › Advanced › REST API. The key is sealed and never shown again.
  3. Webhooks are registered with a secret, and their signatures are checked.
  4. The ReputeCast WordPress plugin (phase 2):
     - WooCommerce has no marketing consent of its own. The plugin adds the checkout checkbox, with its exact text, and records the consent.
     - It also reports abandoned checkouts and keeps the click token on the order.
     - Without the plugin, ReputeCast syncs products and orders only, and the screen says what is missing.
- **What syncs:**
  - Products: title, address, images, price, compare-at price, availability, collections.
  - Customers who accepted marketing: email, name, country, the consent and its date, and the opt-in level.
  - Orders: number, date, items, totals, currency, discount codes, refunds, and the landing page with its UTM parameters.
  - Abandoned checkouts: items, total, and the store's recovery link.
  - The first import covers the last 3, 6 or 12 months of orders (default 12). Then webhooks, plus a nightly reconciliation that fetches anything missed.
  - Never stored: addresses, phone numbers, payment details, order notes.
- **Consent:**
  - Customers who accepted marketing join a chosen list: a new list named after the store, or an existing one.
    - The screen reminds: "They agreed to marketing from Acme Merch. Add them only to lists that consent covers."
    - The consent record says "Shopify: accepted marketing at checkout", with its date and opt-in level.
  - Customers who didn't accept marketing are never added to a list.
    - An existing contact keeps the consent they already have, and their orders show on their profile.
    - For anyone else, the order counts in the store's totals and attribution, without their email.
  - A customer who unsubscribes in Shopify leaves the store's list.
  - "Tell Shopify when someone unsubscribes here" (on by default) needs the optional write access.
- **Triggers:**
  - Order placed: filters by product, collection, total and "first order only".
  - First order.
  - Abandoned checkout: after 1 to 24 hours without an order (default 1 hour). Only contacts who accepted marketing enter, and that filter can't be removed. Placing the order exits the flow.
  - "Placed an order" is also a goal.
- **Blocks in the composer:**
  - Product card: the composer's existing block, filled from the store's catalog or from a digital product (3.5). Its manual mode stays.
  - Product grid: chosen products, or the newest from a collection.
  - Checkout items: the items, quantities and prices of the abandoned checkout or order that started the flow, with "Return to checkout". Only in automation emails started by a store trigger.
  - Prices and availability are read when each email is sent. Sold-out products are hidden or labelled "Sold out", per block. A product removed from the store is flagged by the composer's checks and Send Readiness.
- **Links to the store** get UTM parameters (source acme-newsletter, medium email, campaign slug) and the click token `rc`.
- **Attribution:**
  - An order is credited to the email whose link the buyer clicked last, within the window: 1 to 30 days, default 5.
  - It's matched by the click token in the order's landing page, otherwise by the order's email belonging to a contact who clicked within the window.
  - One order is credited to one email: a campaign or an automation step.
  - Past orders are matched to past clicks during the first import.
  - Attributed orders show in the campaign report's Revenue section (phase 6), in automation stats, in Earn › Overview and on the store's page.
- **Abandoned-cart starter automation:**
  1. Trigger: abandoned checkout, after 1 hour.
  2. Email "You left something behind", with the Checkout items block.
  3. Wait 23 hours.
  4. Goal: placed an order, which exits the flow.
  5. Email "Still deciding?", with a product grid.
  6. Exit.
  - Skip rule: the contact got an abandoned-cart email in the last 7 days.
- **Disconnecting** removes ReputeCast's webhooks from the store.
  - Synced products and orders are kept, unless "Delete store data" is chosen with a typed confirmation.
  - Contacts the store added stay, with their consent.
  - Automations with store triggers pause and say why.
- **Health:**
  - A missing webhook: ReputeCast polls every 15 minutes until it's registered again.
  - A revoked token or an uninstalled app: Disconnected, with Reconnect. Orders placed in the meantime are fetched on reconnect.
  - Rate limits: the sync slows down and says so.

### 3.7 Permissions
New permission group, "Website and pages":
- view posts, pages and website numbers (website:view)
- edit posts and pages (website:edit)
- publish, schedule and unpublish posts and pages (website:publish) ⚑
- manage the website: turning it on or off, home, navigation, tags, members, meter and paywall (website:manage) ⚑

New permission group, "Products and orders":
- view products and orders (products:view)
- edit products (products:edit)
- publish, archive and change the price of products (products:publish) ⚑
- manage orders: new links, revoking and restoring access (orders:manage)

Existing keys that apply:
- A store's page: integrations:view. Connecting, configuring and disconnecting a store: integrations:manage ⚑, people only (phase 2).
- Store orders on a contact's profile: subscribers:view. Revenue in reports: reports:view.
- Exporting orders: subscribers:export ⚑.
- Built-in pages and forms: forms:edit.

Who gets what:
- **Built-in roles:**
  - Owner, Admin and Editor: all eight.
  - Viewer: website:view and products:view.
- **Content Writer**, the sample's custom role: website:view, website:edit, products:view and products:edit.
- **API keys:** any of the eight, never more than their creator holds.
- **MCP agents:** can list and read posts, pages, products and their numbers, and draft posts. Publishing becomes a proposal for a person. They never publish, change a price, refund or revoke access.

**Approvals:** Settings › Approvals gets an optional rule, "Publishing a post or page needs approval", off by default.
- The existing rule "The author lacks the send permission" also covers website:publish. With it on, an author without website:publish gets "Request approval" instead of Publish.
- Approvers are the workspace's approvers (campaigns:approve).
- The approval covers the content, slug, visibility and publish time.

### 3.8 Audit
- **Audited:**
  - the website turned on or off
  - a post published, scheduled, unpublished or deleted, its visibility changed, or its slug changed (with the redirect)
  - home, navigation, tags, access, meter and paywall settings changed
  - a page published, unpublished or deleted
  - a product created, published, archived or deleted, its price changed, or its file replaced
  - a new download link sent by a person, and access revoked or restored
  - a refund or dispute recorded from Stripe, by the system
  - a store connected, reconnected or disconnected; its sync, list or attribution settings changed; store data deleted
  - orders exported
  - a member signed out everywhere by a person
- **Not in the audit log:**
  - page views
  - member sign-ins, which are on the contact's timeline
  - each order placed, since the orders list is the record
  - each store webhook, which is in the store's activity log

  See section 8.

### 3.9 HostRepute
- The website uses news.acme.io, which is already monitored. This phase adds no new touchpoint.
- Nothing about members, buyers, orders, stores or page content is ever sent to HostRepute.

### 3.10 Out of scope for this phase
- Comments, community and member directories.
- Uploading custom themes, or custom code in the page head.
- Podcasts and video hosting.
- Physical products, inventory and shipping (that's the store's job), and changing store data from ReputeCast, except unsubscribes.
- Subscriptions as products (Phase 6 tiers cover recurring payments), license keys, product discount codes and affiliates.
- Invoices (Stripe's receipts and Stripe Tax cover them).
- Store platforms other than Shopify and WooCommerce, and abandoned-browse triggers.
- Custom domains per page, and A/B tests of pages.

---

## 4. Sample data

### Website
- **Address:** news.acme.io.
  - The custom domain was verified on 18 Sep 2026. Let's Encrypt certificate, renews 9 Dec 2026.
  - Turned on by Linh Nguyen on 29 Sep 2026. Until then, news.acme.io showed the subscribe page and the archive.
- **Title:** "The Acme newsletter". **Tagline:** "What shipped, what's next and the occasional deep dive."
- **Home:** layout Featured and latest. Featured: "What shipped in October, and what we're retiring" and "Inside our Postgres migration: the runbook".
- **Header:** Weekly Digest · Product Updates · Engineering · Field Guide · About, then Subscribe and Sign in.
- **Footer:** Archive · RSS · Privacy · Email preferences · Sent with ReputeCast.
- **Last 30 days** (16 Sep–15 Oct), across posts, home, archive and tag pages:
  - 61,420 views from 23,910 visitors
  - 1,312 signups and 41 upgrades to a paid tier
  - 2,804 members have signed in since 29 Sep

### Posts
- 190 published: 186 from campaigns and 4 web-only. Plus 1 scheduled, 1 draft and 1 unpublished.

| Post | From | Visibility | Status | Views, 30 d | Signups | Upgrades |
|---|---|---|---|---|---|---|
| Founding members' AMA: recording and notes | Web only · Linh Nguyen | Paid · Founding member · never free | Scheduled 16 Oct, 17:00 America/New_York | – | – | – |
| Feature flags at scale: what 10,000 flags taught us | Weekly Digest #142 | Public | Published with the send, 14 Oct 09:00 | 4,206 | 88 | 3 |
| Inside our Postgres migration: the runbook | Web only · Sofia Rossi | Paid · all tiers · preview of 3 paragraphs | Published 13 Oct 08:00 | 3,912 | 46 | 17 |
| Ask Acme: your deliverability questions, answered | Web only · Marcus Chen | Members | Published 10 Oct | 2,288 | 131 | 2 |
| What shipped in October, and what we're retiring | October Product Update | Public · featured | Published with the send, 9 Oct 10:00 | 9,874 | 212 | 6 |
| The case for boring infrastructure | Weekly Digest #141 | Public | Published with the send, 7 Oct 09:00 | 7,140 | 164 | 4 |
| Q4 roadmap deep dive | Web only · Daniel Park | Members | Draft · waiting for approval from Marcus Chen or Linh Nguyen | – | – | – |
| Feature flags preview | Beta Program campaign, 2 Oct | – | Unpublished 10 Oct by Sofia Rossi: "Beta content, not for the public." 412 views before | – | – | – |

- The other two web-only posts: "Welcome to the new news.acme.io" (29 Sep, Linh Nguyen) and "How we write the Weekly Digest" (1 Oct, Sofia Rossi).
- **Weekly Digest #142, as a post:**
  - Its greeting "Hi {{ first_name | default: "there" }}" reads "Hi there".
  - Hidden on the web: the footer, the view-online link, and a block for Beta Program readers.

### Tags
| Tag | Posts | In navigation |
|---|---|---|
| Weekly Digest | 143 | Yes |
| Product Updates | 34 | Yes |
| Engineering | 61 | Yes |
| Deliverability | 19 | No |
| Insider (paid posts) | 9 | No |
| Announcements | 3 | No |
| Ask Acme | 3 | No |
| Beta Program | 0: its only post is unpublished | No |

### Members and access
- **Sign-in:** on. Links and codes work for 30 minutes. Members stay signed in for 90 days.
- **Sign-up on the site:** on. It offers Weekly Digest (checked by default) and Product Updates.
- **Default visibility:** web-only posts Public; posts from campaigns follow the audience.
- **Meter:**
  - Visitors who aren't signed in can read 2 members-only posts a month.
  - Free members can read 1 paid post a month.
  - "Founding members' AMA" is never free.
- **Paywall card:**
  - Heading: "This post is for Insider members"
  - Text: "Deep dives, runbooks and the monthly AMA, from the people who build Acme."
  - Tiers (phase 6): Weekly Digest Insider, $8 a month or $80 a year; Founding member, $200 a year.
  - Free row: "Not ready? Sign up free and read 1 paid post a month."
- **Members:** 2,804 signed in since 29 Sep, 1,126 in the last 7 days.
- **Paid members:** 1,184 (phase 6): 1,091 Weekly Digest Insider and 93 Founding member.

### Readers
- **A visitor:** not signed in, first visit this month.
- **Sam Rivera** (sam.rivera@outlook.com), a free member on Weekly Digest and Product Updates:
  - signed in on 12 Oct
  - read the runbook as his free paid post for October on 13 Oct
  - bought The Deliverability Field Guide on 3 Oct
  - bought a hoodie from Acme Merch on 13 Oct (see "Shopify")
- **Hannah Weiss**, Weekly Digest Insider, yearly ($80), member since 2 Feb 2026, renews 2 Feb 2027.

### Pages
| Page | Address | Template | Status | Views, 30 d | Result |
|---|---|---|---|---|---|
| About | /about | About | Published 29 Sep | 2,104 | 38 signups |
| The Deliverability Field Guide | /field-guide | Sales page | Published 1 Oct | 9,840 | 189 sales · $5,481 · 1.9% |
| DNS records checklist | /dns-checklist | Lead magnet | Published 22 Sep | 6,412 | 1,148 signups · 17.9% |
| Live: Fix your DMARC setup in 30 minutes | /dmarc-live | Webinar registration | Published 8 Oct | 2,930 | 486 registrations · 16.6% |
| Newsletter Launch Kit | /launch-kit | Launch | Published 29 Sep · countdown ended 6 Oct 09:00 | 3,618 | 81 sales · $1,071 |
| Sponsor the Weekly Digest | /sponsor | Blank | Draft · Daniel Park | – | – |
| Shop | /products | Built-in | Published | 3,105 | 41 sales · $942 |

- **Built-in pages:** Subscribe, Confirmation, Preference center, Unsubscribe, Sign in, Account, Archive and Shop; Phase 5's Referral hub, Invite page and Recommendations; Phase 6's Upgrade and Give a gift.

### Lead magnet: DNS records checklist
- A PDF, 2 pages, 380 KB, attached to the form "DNS checklist".
- The form is embedded on /dns-checklist, uses double opt-in and adds people to Weekly Digest.
- Consent text: "Send me the DNS records checklist and the Weekly Digest. Unsubscribe any time."
- The confirmation email says "Confirm your subscription to get the checklist". After the click, the confirmation page offers the download. The link works for 7 days.
- Since 22 Sep: 6,412 views, 1,148 signups, 1,032 confirmed (89.9%), 1,019 downloads.

### Webinar: Live: Fix your DMARC setup in 30 minutes
- 29 Oct 2026, 17:00–17:30 UTC (13:00 in New York, 17:00 in London), on Zoom. Host: Marcus Chen.
- 486 registrants on the list "DMARC live · 29 Oct": 301 were already contacts and 185 are new.
- Consent text: "Send me the joining link and reminders for this session."
- The form writes the date field `dmarc_live_at`.
- The automation "DMARC live reminders" sends the joining link on registration, 1 day before and 1 hour before.

### Launch: Newsletter Launch Kit
- A countdown to 6 Oct 2026, 09:00 America/New_York. Before it ended, 1,406 people joined the waitlist.
- Since then the page shows the Buy button: 81 sales.

### Products
| Product | Files | Price | Status | Orders | Gross | Refunded |
|---|---|---|---|---|---|---|
| The Deliverability Field Guide | PDF, 86 pages, 4.2 MB | $29 | Published 1 Oct by Sofia Rossi | 214 | $6,206 | 3 · $87 |
| Newsletter Launch Kit | ZIP, 38 MB: 12 MJML templates and a checklist | Pay what you want, from $5, suggested $15 | Published 6 Oct | 97 | $1,288 | 0 |
| Warm-up planner | XLSX, 210 KB | $12 | Draft · Daniel Park | – | – | – |

- **Totals:** 311 orders, $7,494 gross, $87 refunded, $7,407 net before Stripe's fees.
- **Where the sales came from:**
  - Field Guide: 189 from its sales page and 25 from the shop.
  - Launch Kit: 81 from its page ($1,071) and 16 from the shop ($217).
- **Stripe:** the "Acme Media" account, live mode, USD.
- **Delivery:** `product-delivery` v1 in the Purchases category. The Transactional route is Postmark, then Amazon SES. Links work for 7 days and 5 downloads.
- **List offer:** "Also send me the Weekly Digest". 61 buyers ticked it: 44 Field Guide buyers and 17 Launch Kit buyers.
- **Field Guide buyers:** 131 were already contacts, 44 joined at checkout and 39 aren't contacts.
- **Automation "Field Guide follow-up":** "Bought a product" trigger, live. 172 contacts entered: 175 contacts minus 3 refunded.

### product-delivery v1
- **Subject:** `Your download: {{ product_name }}`
- **Variables:**
  - `first_name`: optional
  - `product_name`: required
  - `download_url`: URL, required, sensitive
  - `expires_on`: required
  - `downloads_allowed`: required
  - `order_number`: required
  - `amount`: required
- **Body:** "Thanks for buying The Deliverability Field Guide. [Download the PDF, 4.2 MB] This link works until 22 Oct 2026, for 5 downloads. Need it again later? Sign in at news.acme.io/account. Order 1311 · $29.00. Stripe sends your receipt separately."

### Orders (latest)
| Order | Time | Buyer | Product | Amount | Status | Download |
|---|---|---|---|---|---|---|
| 1311 | 15 Oct 10:12 | ana.ruiz@example.com | Field Guide | $29.00 | Paid | Delivered via Postmark · 1 of 5 downloads |
| 1310 | 15 Oct 09:47 | marco.bianchi@example.it | Launch Kit | $20.00, pay what you want | Paid | Link sent 09:47 · 0 of 5 |
| 1309 | 15 Oct 08:30 | lee.wong@example.com | Field Guide | $29.00 | Refunded 15 Oct 09:05, in Stripe | Access revoked |
| 1302 | 14 Oct 19:02 | d.okoro@example.org | Field Guide | $29.00 | Paid | Link expired · new link sent 15 Oct 08:00 by Sofia Rossi |
| 1288 | 12 Oct 21:15 | j.kowalski@example.pl | Field Guide | $29.00 | Disputed 14 Oct: "fraudulent" | Access revoked automatically |
| 1107 | 3 Oct 11:20 | sam.rivera@outlook.com | Field Guide | $29.00 | Paid | 2 of 5 · link expired 10 Oct |

### Shopify: Acme Merch
- **Store:** shop.acme.io (acme-merch.myshopify.com), USD.
  - Connected on 6 Oct 2026 by Marcus Chen, through the custom app "ReputeCast", API version 2026-10.
  - 11 webhooks: orders created, paid, updated; refunds; customers created and updated; checkouts created and updated; products created, updated and deleted. Last event 2 min ago.
- **First import:**
  - 38 products in 4 collections: Apparel, Desk, Stickers, New arrivals.
  - 6,240 customers checked:
    - 2,318 accepted marketing and joined the new list "Acme Merch": 1,947 were already contacts and 371 are new.
    - 3,922 didn't accept marketing and were not added.
  - 7,880 orders from the last 12 months.
- **"Tell Shopify when someone unsubscribes here":** on.
- **Workspace contacts:** 48,769 on 15 Oct. That's 48,213, plus 371 from Acme Merch and 185 webinar registrants. Weekly Digest, Product Updates and Beta Program keep 31,988, 14,110 and 2,115.
- **Attribution:** 5 days after a click. 89 orders were matched by click token, 7 by email.
- **Last 30 days:** 412 orders, $21,836. 96 orders and $5,184 (23.7%) are attributed to email.

| Attributed to | Orders | Revenue |
|---|---|---|
| October Product Update | 31 | $1,612 |
| Weekly Digest #141 | 18 | $902 |
| Welcome series | 11 | $598 |
| Abandoned cart (automation) | 8 | $472 |
| Weekly Digest #142 | 6 | $318 |
| 9 other campaigns | 22 | $1,282 |
| **Total** | **96** | **$5,184** |

- **Product blocks sent:**
  - October Product Update had a product grid from New arrivals: Acme Logo Hoodie $58, Ship It mug $18, Feature Flag sticker pack $9.
  - Weekly Digest #141 had one product card from the store: Boring Infrastructure tee $28.
- **Abandoned checkouts since the flow went live on 7 Oct:**
  - 104 abandoned, 78 with an email.
  - 44 contacts who accepted marketing entered the flow. 34 weren't emailed, because they didn't accept marketing.
  - 8 recovered orders, $472 (18.2% of 44). Email 1 sent 44, email 2 sent 30.
- **Grace Liu:**
  - Left a checkout on 14 Oct at 20:12: Acme Logo Hoodie (M) $58 and Ship It mug $18, $76 in all.
  - Got "You left something behind" at 21:12 and ordered on 15 Oct at 08:40. Credited to Abandoned cart, email 1.
- **Sam Rivera:**
  - Ordered a hoodie on 13 Oct (#4409, $58) without ticking Shopify's marketing box.
  - He isn't added to Acme Merch. He's already a Weekly Digest subscriber, so the order shows on his profile.
  - It's credited to October Product Update, whose link he clicked on 9 Oct.
- **theo.martin@example.com:** ordered on 15 Oct at 09:58 (#4421, $64) without accepting marketing. He isn't a contact, so the order counts in the totals without his email.
- **Incident (history):**
  - On 13 Oct at 14:20 someone deleted the "checkouts updated" webhook in Shopify.
  - ReputeCast polled every 15 minutes until Marcus Chen registered it again at 16:05. No checkout was missed.

### WooCommerce
- Not connected. It's shown in the catalog, and its connect dialog uses placeholders.

### People
- **Linh Nguyen** (Owner) turned the website on and owns the Stripe account.
- **Marcus Chen** (Admin) connected Shopify and set the attribution window.
- **Sofia Rossi** (Editor) publishes posts, pages and products, and handles orders.
- **Daniel Park** (Content Writer) drafts "Q4 roadmap deep dive", the "Sponsor the Weekly Digest" page and the Warm-up planner. He can't publish any of them; for the post he requests approval.
- **Aisha Bello** (Viewer) sees posts, pages, products and orders, read-only.

---

## 5. New screens

Forms & pages has four tabs: Signup forms · Website · Pages · Branding & domain.
- The Website tab has three views, as pill tabs: Posts · Home & navigation · Access.
- Earn gets a Products tab.

### 5.1 Website › Posts
- **Header:**
  - status "Live · news.acme.io · certificate valid", and "View site"
  - the numbers for 30 days: views, visitors, signups, upgrades
  - "New post", and "Publish past campaigns" (pick sent campaigns that aren't posts yet)
- **Table:**
  - title with its excerpt
  - source: "Weekly Digest #142" (links to the campaign) or "Web only", with the byline
  - visibility badge, and "featured"
  - status badge with its time
  - tags
  - views, signups and upgrades for 30 days
  - row menu: Edit, View, Copy link, Unpublish, Send as a campaign (web-only posts), Delete
- **Filters:** status, visibility, tag, source. Search by title.

States to show:
- the sample
- Daniel Park's draft waiting for approval, with who can approve
- the unpublished post, with its reason
- a scheduled post
- read-only (Aisha Bello): actions hidden, with the reason
- loading, error, and a filter with no results
- the website off: the archive's settings with "Turn on the website" (5.2)

### 5.2 Turning the website on
A page with four short sections and one button.
1. **Address:** news.acme.io is verified and its certificate is valid. If not, a link to Branding & domain.
2. **What changes:** the list from 3.1, with numbers. On 29 Sep: "182 archived campaigns become public posts. Their old links redirect."
3. **Home layout:** three thumbnails.
4. **Members:** sign-in on, and the lists offered on sign-up.

"Turn on the website" ends with "Your website is live at news.acme.io" and three next steps: set visibility for paid posts, check the navigation, write an About page.

States to show:
- ready
- domain pending: the site would run at reputecast.acme.io/p/acme-newsletter until DNS is verified, and the screen says what to add
- certificate failing: "Let's Encrypt couldn't issue a certificate for news.acme.io: the CNAME points elsewhere"
- turning on (progress: posts created, redirects written)
- done
- turning off: the typed confirmation with what goes offline ("The home page, 4 web-only posts, sign-in and the paywall go offline. 1,184 paid members keep their subscriptions and emails. Your 5 public pages and the shop stay online.")
- no website:manage: read-only, with who can help

### 5.3 Publish to web, and web-only posts
**In the campaign composer:**
- The header gets a Web chip: "Not on the web", "Publishes with the send", "Scheduled for the web" or "Published".
- The chip opens a Web tab in the right panel, next to Style and Settings:
  - a "Publish to news.acme.io" switch
  - slug, with the full address
  - title, excerpt, cover and alt text, tags, byline
  - visibility, with the tiers when Paid
  - free preview: "Ends after block 3 · Move" or "Add a marker"
  - when: with the send, at a time, or now (sent campaigns)
  - "Feature on home"
  - SEO and social, folded
  - web checks:
    - the slug is free
    - the cover has alt text
    - tokens that will show their fallback ("Hi there")
    - blocks hidden on the web, listed and linked to the canvas
- **Preview on web:** the post as it will look, with a switch: Visitor · Free member · Paid member.

**Web-only posts:** the composer in post mode.
- The header has the breadcrumb Website › Posts, the title, status and visibility chip, Preview, and Publish (or Schedule, or Request approval).
- The canvas shows the web version in the 680 px column. The same Web tab holds the post's settings.

States to show:
- Weekly Digest #142 set to publish with the send
- Inside our Postgres migration (web-only, Paid) with the marker after the third paragraph, the Web tab open, and the preview as a visitor
- the same post previewed as a free member and as a paid member
- slug taken: "/posts/feature-flags is used by a post from 2025. Use feature-flags-at-scale?"
- Daniel Park: "Request approval" in place of Publish, with the matched rule
- published: a toast with "View post" and "Undo", which unpublishes
- phone: the Web tab as a sheet

### 5.4 Website › Home & navigation
Two columns: settings on the left, the live home page on the right (desktop and phone).
- **Layout:** three thumbnails.
- **Featured posts:** pick up to 3, with drag to order.
- **Home sections:** latest posts (how many), a row per chosen tag, the about text, a subscribe band, a paid tiers band. Each can be shown or hidden.
- **Navigation:** header links (drag to order, up to 6), footer links.
- **Tags:** name, slug, description, posts, "Show in navigation", feed address.
- **Site identity and search:**
  - title, tagline, description, social image, favicon
  - "Let search engines index the site"
  - sitemap and RSS addresses
- **Turn off the website:** in a danger zone at the bottom.

States to show:
- the sample
- a seventh header link: "The header holds 6 links. Move one to the footer."
- a tag with no posts: Beta Program, whose only post is unpublished
- unsaved changes, saved
- read-only

### 5.5 Website › Access
- **Sign-in:** link and code lifetime, session length, the sender (the workspace default, System stream).
- **Sign-up on the site:** on or off, the lists offered and which are checked.
- **Default visibility:** for web-only posts and pages, and for posts from campaigns.
- **Meter:** the two rules, each with a switch and a number, and the posts excluded from the meter.
- **Paywall card:** heading, text, tiers, button labels and the free row, with a live preview: Visitor · Free member · Free member, meter used up.
- **Members:**
  - 2,804 signed in since 29 Sep, 1,126 in the last 7 days
  - 1,184 paid members, linked to Earn › Paid members
  - "Show signed-in members", which opens Contacts with that filter

States to show:
- the sample
- Stripe not connected: Paid visibility and the tiers row are unavailable, linked to Earn › Settings › Payments; Public and Members still work
- no paid tiers yet (phase 6 set up but empty)
- sign-in turned off: members-only and paid posts become unreadable on the web, and the screen says how many
- a validation error (meter of 0 with the rule on)
- read-only

### 5.6 Public: home and tag pages
- **Header:** logo and title, navigation, Subscribe and Sign in (or the member's initials).
- **Hero band:** title, tagline and an inline subscribe form.
- **Featured:** two large cards: cover, tag, title, excerpt, and a visibility label ("Insider" for paid, "Members" for members-only).
- **Latest posts:** a list with date, tag, title, excerpt and label.
- **Then:** a tag row (Engineering), the about text, the paid tiers band, the footer.
- **Tag page:** the tag's name, description and feed link, then its posts in a list with paging.

States to show (in PublicSiteStates):
- the home page as a signed-in member: initials in the header, and the subscribe band replaced by "You're subscribed to Weekly Digest"
- the Engineering tag page
- the archive with a search ("postgres", 6 posts)

### 5.7 Public: post
- **The post:**
  - title, byline, date, reading time, tags, cover with alt text, the text in the 680 px column
  - for a gated post: the preview, then the paywall card
- **Main board:** "Inside our Postgres migration: the runbook", seen by a visitor.
  - three paragraphs, then the paywall card:
    - heading
    - the two tiers with prices and Upgrade buttons
    - the free row with an email field
    - "Already a member? Sign in"
- **Below every post:** "More from Engineering" (3 posts) and the subscribe band.

States to show (in PublicSiteStates):
- Sam Rivera on 13 Oct: the whole runbook, with the note "This is your free paid post for October"
- Sam Rivera on 15 Oct, opening another paid post: the meter is used up ("You've read your free paid post for October. Upgrade to read them all, or come back on 1 November.")
- Hannah Weiss: the whole runbook, with the Insider label
- "Ask Acme" (Members) as a visitor: "This is 1 of 2 members-only posts you can read this month without signing in"
- the same post after the visitor's 2 posts: "Sign up free to keep reading", with the email field
- the unpublished post: "This post is no longer available", with links to the latest posts and the archive
- a post page that fails to load: the site's error page

### 5.8 Public: sign in and account
- **Sign in** (`/signin`): email field, Continue, "Not a member yet? Subscribe free".
- **Link sent:** "If sam.rivera@outlook.com belongs to a member, a link and a code are on their way", a 6-digit code field, "Send again" (after 60 seconds), "Use a different email".
- **Sign-in email:** subject "Your sign-in link for The Acme newsletter", a Sign in button, the code 482 913, "Works for 30 minutes".
- **Account** (`/account`), main board for Sam Rivera:
  - "Your account", sam.rivera@outlook.com, "Free member since 18 Sep 2026"
  - Plan: Free, with the Insider tier card and Upgrade
  - Newsletters: Weekly Digest (weekly roundup) and Product Updates on, Beta Program off, pause
  - Purchases: The Deliverability Field Guide, 3 Oct, $29, "Your link expired on 10 Oct · Send me a new link"
  - Gifts: none, "Give a gift" (phase 6)
  - Sign out · Sign out everywhere

States to show (in PublicSiteStates):
- link sent
- wrong code: "That code doesn't match. 2 tries left."
- link expired: "This link has expired. Links work for 30 minutes."
- link already used: "This link was used to sign in on another device. Send a new one here."
- too many requests: "Too many links for this address. Try again in 40 minutes."
- signed in: a toast on the post the reader came from
- Hannah Weiss's account: plan "Weekly Digest Insider · $80 a year · renews 2 Feb 2027", "Manage billing" and "Change plan"
- the new-link email sent from the account page

### 5.9 Pages
- **Header:** "New page", and the 30-day numbers: views, signups, registrations, sales.
- **Your pages:** name, address, template, status, visibility, views, result, last edited by. Filters by template and status.
- **Built-in pages:** name, address, on or off, and "Edit", which opens that page's existing editor.
- **New page:** a template picker, each template with a thumbnail and one line on what it's for.

States to show:
- the sample
- the template picker
- Daniel Park's draft, with Publish unavailable and who can help
- an empty list: "No pages yet. Start with About or a template."
- read-only
- loading, error

### 5.10 Page builder
The composer's interface in page mode, on "The Deliverability Field Guide".
- **Header:** breadcrumb Pages › The Deliverability Field Guide, "Published · unpublished changes", device switch, Preview, "Publish changes".
- **Left panel:** blocks grouped as Layout, Content, Conversion, and From your site (recent posts, author bio). Then layers.
- **Canvas:** the page on the publisher's theme, with the selected block outlined and its toolbar.
- **Right panel:**
  - **Block:** the selected block's settings, here the product block: product, button text, "Show price", "Show the list offer".
  - **Page:** slug, title, SEO, social image, visibility, "Hide the site header and navigation", "Show in navigation", publish time.
  - **Checks:** the page's publish checks.
- **Publish dialog:** the diff since the published version, the checks, "Publish".

States to show:
- the sample, with the product block selected
- the Launch template's countdown, with "When it ends: reveal the Buy button"
- the Webinar template's event details, with the offer to create the reminder automation
- a blocking check: "The signup form 'Beta apply' is paused. Resume it or pick another form."
- Stripe not connected: Buy buttons marked "Not available until Stripe is connected"
- autosave saved, and a conflict
- tablet: panels stacked under the canvas

### 5.11 Public: landing pages, sales pages and the shop
- **Main board:** the Field Guide's sales page.
  - hero with the cover and "Buy for $29"
  - the list offer checkbox, unchecked
  - what's inside (8 chapters), the author bio (Sofia Rossi), 3 testimonials, FAQ, a second Buy button
  - on phones, a Buy bar at the bottom
- **Other pages (in PublicPagesStates):**
  - **DNS records checklist (lead magnet):** hero, the form, "We'll email a confirmation link. The checklist is on the page that opens."
  - **Its confirmation page:** "You're in. Download the checklist (PDF, 380 KB)".
  - **DMARC live (webinar):** event details in UTC and the reader's own time ("13:00 your time, New York"), registration form. After registering: "You're registered", with Add to calendar.
  - **Launch Kit:** before the countdown ended (waitlist form, "5 days 3 hours"), and after (the Buy button, pay what you want with a suggested $15).
  - **Shop:** the two published products as cards.

### 5.12 Earn › Products
- **Header:** the 30-day numbers: orders, gross, refunded, net. "New product".
- **Products table:** cover, name, price, status, orders, gross, refunds, and the sales page.
- **Orders:** below the products, with search (order number or email), filters (product, status, date) and "Export CSV".
- **Order panel:** a side panel with:
  - the order's number, status and amount
  - the buyer, linked to their contact if any, and the consent given at checkout
  - the payment, with "Open in Stripe"
  - delivery: the email's status, linked to Transactional › Messages
  - the download link's expiry and the downloads used
  - actions: "Send a new link", "Revoke access", "Refund in Stripe"

States to show:
- the sample, with order 1311 open
- the refunded order 1309
- the disputed order 1288, with the dispute's status from Stripe
- d.okoro's expired link, and the new link sent
- Stripe in test mode: a Test badge on products and orders, and a banner
- Stripe not connected: products can be drafted and nothing can be bought, linked to Earn › Settings › Payments
- empty: "No products yet. Sell a guide, templates or recordings, through your own Stripe account."
- read-only (Aisha Bello): no actions, with the reason
- loading, error

### 5.13 Product editor
- **Left column:**
  - **Details:** name, slug, short and long description, cover with alt text.
  - **Files:** drop zone, the list with sizes and upload progress.
  - **Price:** one-time, or pay what you want with minimum and suggested price; the currency and the Stripe Tax setting, read from Phase 6.
  - **Selling:** sales page, "Show in the shop", the list offer (list and exact text).
  - **Delivery:** the template, with its preview and a link to edit it; link validity; downloads per link; "Revoke access on refund or dispute".
- **Right column:** the product card as it appears in the shop, the default product page, the publish checks, and, once published, the product's numbers.
- **Header:** status, "Save", "Publish" (or "Publish changes").

States to show:
- the Field Guide, published
- the Warm-up planner as Daniel Park: Publish unavailable, with who can help
- an upload in progress, and a failed upload: "warmup-planner.xlsx stopped at 64%. Try again."
- a file whose content doesn't match its name: "This file says .pdf but contains a ZIP archive"
- publish blocked: "product-delivery has no published version" and "No Transactional route", each with its fix
- pay what you want below Stripe's minimum: "The minimum must be at least $1"
- changing the price of a published product: a confirmation that existing buyers keep their access

### 5.14 Buying and downloads (public)
All in PublicPagesStates:
- **The default product page:** for a product without a sales page.
- **"Taking you to Stripe…":** a short handoff that names the publisher's Stripe account.
- **Thank-you page:** "Thanks, your download is ready", "Download now", "We also emailed the link to a•••@example.com".
- **The delivery email:** product-delivery v1 with the sample values.
- **Download page:** the files, sizes, "Link works until 22 Oct · 4 of 5 downloads left".
- **Link expired:** "This link has expired. Send a new link to s•••@outlook.com".
- **Downloads used up.**
- **Access revoked after a refund:** "This purchase was refunded, so the download is no longer available."
- **Checkout cancelled:** back on the sales page, "No payment was taken".
- **Test mode:** a banner on the thank-you page, "Test purchase: no money was taken".

### 5.15 Shopify and WooCommerce
**Shopify's page** in the Integrations hub (Acme Merch), in the phase 2 detail-page layout:
- **Header:** the store's name and address, status "Connected · last event 2 min ago", "Open store", "Sync now".
- **What it does:**
  - triggers: order placed, first order, abandoned checkout
  - blocks: product card from the catalog, product grid, checkout items
  - the data it syncs
- **Sync:** products 38, customers who accepted marketing 2,318, customers not added 3,922, orders imported 7,880, last reconciliation.
- **Customers and consent:**
  - the list (Acme Merch) and the reminder about what the consent covers
  - "Tell Shopify when someone unsubscribes here"
  - what is stored and what never is
- **Attribution:** the window, the matching, UTM tagging, and the last 30 days (orders, revenue, share from email, the table by email).
- **Webhooks:** each topic with its status and last event, and "Register again".
- **Activity:** recent events and errors.
- **Disconnect:** with the choice to keep or delete store data.

**Connect dialogs** (in CommerceStates):
- Shopify: the five steps of 3.6, with the values to copy.
- WooCommerce: the four steps, with the plugin check.

States to show:
- the sample
- first import running: "Importing orders · 5,120 of 7,880"
- the 13 Oct incident: a webhook missing, "Polling every 15 minutes", "Register again"
- disconnected: "Shopify revoked access on 14 Oct 18:05 (the app was uninstalled). Abandoned cart is paused. Orders placed since then are fetched when you reconnect."
- a customer who didn't accept marketing, as seen on the activity log
- WooCommerce without the plugin: "Customers can't join lists and abandoned checkouts aren't reported until the ReputeCast plugin is active on store.example.com"
- an invalid client secret, and a store address that isn't a Shopify store
- read-only (Aisha Bello)
- tablet and phone: the cards stacked

### 5.16 Store data in emails and automations
All in CommerceStates:
- **Product picker in the composer:**
  - search the catalog, collection filter, product cards with price and availability, "Insert"
  - the product grid's settings: products or collection, how many, sold-out behavior
- **The October Product Update grid** on the canvas, with its three products.
- **A product removed from the store**, flagged: "Feature Flag sticker pack is no longer in Acme Merch. Remove it or pick another product."
- **The abandoned-cart starter** on the flow canvas, with its numbers (44 entered, 8 orders, $472), and the trigger's side panel showing the locked filter "Accepted marketing in Acme Merch".
- **Grace Liu's first email**, with her two checkout items and "Return to checkout".
- **The October Product Update report's Revenue section:** "Acme Merch: 31 orders · $1,612" next to Phase 6's subscription revenue.
- **Sam Rivera's profile:** the Field Guide purchase and the Acme Merch order, with consent "Not added to Acme Merch: didn't accept marketing in Shopify".

---

## 6. Changes to existing screens

- **Forms & pages tabs** (FormBuilder, FormsStates and every page tab):
  - The tabs become Signup forms · Website · Pages · Branding & domain.
  - The Archive tab becomes Website. With the website off, Website shows today's archive settings and the "Turn on the website" card.
  - The Subscribe page, Preference center and Unsubscribe page tabs move to Pages › Built-in pages. Their editors don't change.
- **Branding & domain:**
  - The card says what it applies to: the website, pages and every public page.
  - The domain card shows what news.acme.io serves: "Website · 190 posts".
- **Form builder:**
  - "After confirmation: offer a download", with a file, for lead magnets.
  - "Used on pages", listing the pages whose signup form block uses the form.
  - Pages on the site are allowed origins automatically.
- **Public archive:**
  - With the website on, it uses the site's header and navigation, and its issues link to their posts.
  - Search covers titles, excerpts and public text.
- **Public subscribe, confirmation, preference center and unsubscribe pages:**
  - With the website on, they use the site's header.
  - The confirmation page offers the lead magnet's download when the form has one.
  - The preference center links to the account page when the reader is signed in.
- **Composer:**
  - The Web chip and the Web tab (5.3).
  - The Email / Web switch in the preview.
  - New blocks: "Free preview ends here", Product grid and Checkout items.
  - The existing Product card can be filled from a digital product or the store's catalog.
  - The Visibility section (phase 3) gains "Show on: email and web / email only / web only".
- **Review & send:**
  - A Web line in the summary: "Also publishes news.acme.io/posts/feature-flags-at-scale, public, with the send."
  - The web checks in the content group.
- **Campaigns list:** a Web column: Published, Scheduled, or –.
- **Campaign report:**
  - A Web section: views, signups and upgrades from the post.
  - The Revenue section (phase 6) gains rows for product sales and store orders.
- **Automations:**
  - Triggers: Bought a product; Order placed; First order; Abandoned checkout.
  - Goals: Bought a product; Placed an order.
  - The Checkout items block in emails of store-triggered flows.
  - Starters: Abandoned cart.
  - The list's 30-day columns add attributed revenue.
- **Contacts:** filters for "Signed in to the website", "Bought a product", "Store customer" and "Total spent".
- **Contact profile:**
  - a Website card: last sign-in, "Sign out everywhere"
  - a Purchases card: products, with the download status and "Send a new link"
  - a Store orders card
  - timeline events: signed in, bought, refunded, order placed, abandoned checkout
  - consent history with the new sources: "Shopify: accepted marketing at checkout" and "Product checkout: ticked 'Also send me the Weekly Digest'"
- **Segment builder:** conditions for signed in since, bought a product, store orders (count, total spent, last order date, product or collection bought) and accepted marketing in a store.
- **Earn** (phase 6):
  - Tabs add Products.
  - Overview gains a "Product sales" card (311 orders, $7,494 gross, $87 refunded) and a "Store revenue from email" card ($5,184 of $21,836, 23.7%).
- **Transactional** (phase 1):
  - The "Product delivery" starter and the Purchases category (Essential).
  - The template's row says "Used by 2 products".
- **Settings › Integrations** (the phase 2 hub):
  - The Commerce group with Shopify (Connected) and WooCommerce.
  - Shopify's page follows the phase 2 detail-page layout (5.15).
  - WordPress sites: a site running WooCommerce shows "WooCommerce detected", with the plugin's checkout consent and abandoned-checkout reporting, linked to the store's page.
- **Settings › Approvals:** the rule for posts and pages, and the note that "The author lacks the send permission" covers website:publish.
- **Approval inbox and review:** posts and pages, with a web preview in place of the email preview.
- **Roles & permissions:** the two new groups (3.7) in the role editor and the compare matrix.
- **API keys:** the new keys in the permission picker.
- **Webhooks:** event groups "Website" (post.published, post.unpublished) and "Products and orders" (order.paid, order.refunded, order.disputed).
- **Notifications:** new kinds:
  - store disconnected, and store webhooks missing
  - product delivery failed
  - dispute opened
  - website domain or certificate problem
- **Audit log:** the actions from 3.8, and filter categories Website, Products and Commerce.
- **Settings › MCP server:** the new tools in the agents' permission matrix: list and read posts, pages and products; draft a post; propose publishing.
- **DSCoverage and `docs/DESIGN_INVENTORY.md`:** the new boards, and the conflicts in section 8.

---

## 7. New design-system patterns

Add these to the "9 Design system" page before the screens. Use the existing tokens, badge colors and the publisher-accent rules for public pages.

- **Public site templates:**
  - home, post, paywalled post, page, member account
  - the header (logo, navigation, Subscribe, Sign in or the member's initials) and the footer
  - the 680 px reading column, inside the 720 px public frame
- **Visibility labels:**
  - Public: a globe icon
  - Members: a person icon
  - Paid: a star icon and the tier name ("Insider")
  - No lock icons.
- **Post and page status badges:**
  - Draft: neutral
  - Scheduled: info
  - Published: ok
  - Unpublished: neutral, with the reason
- **Free preview marker:** a dashed line labelled "Free preview ends here" in the editors. On public pages the paywall card takes its place, with no fade over readable text.
- **Paywall card**, extending Phase 6's: the tiers, the free row with the meter, sign-in, and the meter note on a metered post.
- **View-as switch:** a segmented control, Visitor · Free member · Paid member.
- **Page blocks:** palette icons for each block, the countdown in tabular figures with an accessible sentence ("Starts in 5 days, 3 hours"), event details with both time zones, and the Add to calendar menu.
- **Product card (public):** cover, title, price ("$29" or "Pay what you want · from $5"), "Buy for $29".
- **Order status badges:**
  - Paid: ok
  - Refunded and Partially refunded: neutral
  - Disputed: bad
  - Test: purple
- **Download chip:** "Link works until 22 Oct · 4 of 5 downloads left".
- **Store sync row:** the provider-health pattern applied to a store: "Connected · last event 2 min ago", or "Webhook missing · polling every 15 min".

---

## 8. Conflicts to note

1. **Forms & pages tabs.**
   - Today's boards have six tabs: Signup forms, Subscribe page, Preference center, Unsubscribe page, Archive, Branding.
   - The expansion plan lists Forms · Website · Pages · Branding & domain. This brief follows the plan and keeps the wording "Signup forms": the archive becomes Website, and the three page editors move to Pages › Built-in pages, unchanged.
   - "No renames" in the plan is about sidebar items; these are tab changes, including Branding becoming "Branding & domain".
2. **Audit log vs the site's own records.** Page views, member sign-ins, orders and store webhooks are kept in their own logs, not the audit log, as Phase 1 did for transactional messages. The owner needs to confirm this.
3. **Pages aren't MJML.**
   - CLAUDE.md describes MJML compiled in the browser, with both the source and the HTML stored. That stays true for emails and for posts, which keep their MJML so they can be emailed.
   - Pages are web pages: page mode uses GrapesJS with web components and stores the GrapesJS project and the rendered HTML.
   - Posts also need a new renderer from MJML blocks to the site's web layout. Engineering should confirm both.
4. **Contact totals change.** The Acme Merch sync (371) and the webinar (185) add contacts: 48,213 becomes 48,769 on this phase's boards. The three original lists keep their numbers.
5. **Phase 6 names.** Tiers, the upgrade and gift pages, paid statuses and "Earn › Settings › Payments" are taken from the expansion plan. If the Phase 6 brief names them differently, follow it. If it defines an Earn permission group, products:* joins that group.
6. **File storage.** Product files can be large, and the stack names no object storage. Files go to the instance's data volume, like composer images, with the size limits in 3.5. S3-compatible storage would be a new dependency.
7. **Sign-in emails use the System stream**, like opt-in confirmations, not the Transactional stream. They are ReputeCast's own mail for its public pages.
8. **Approvers for posts** are the workspace's approvers (campaigns:approve). Sofia Rossi, who publishes posts, can't approve Daniel Park's post unless her role gets campaigns:approve.
9. **Lead magnets are delivered by forms, not as free products.** A product needs a price of at least $1. A free file is offered after the reader confirms a subscription, so the list stays confirmed.
10. **WooCommerce consent depends on the phase 2 WordPress plugin.** Without it, WooCommerce customers never join lists.
11. **Archive size.** PublicArchive says "142 issues"; the demo computes 44. This brief's 186 posts from campaigns is sample data for the boards. The app computes the real number.

---

## 9. Boards

New boards for the canvas page "P7 Website, pages & commerce". Public pages are 720 px wide; the rest are 1440 px. Tablet boards are 834 px and phone boards 390 px, named `<Board>Tablet.dc.html` and `<Board>Phone.dc.html`.

| # | Board | Canvas title | Implements | Width | Tablet | Phone |
|---|---|---|---|---|---|---|
| 1 | `WebsitePosts.dc.html` | Website · posts | 5.1 | 1440 | Yes | Yes |
| 2 | `ComposerPublishToWeb.dc.html` | Composer · publish to web, view as each reader | 5.3 | 1440 | – | Yes |
| 3 | `WebsiteHomeNav.dc.html` | Website · home, navigation, tags and site settings | 5.4 | 1440 | – | – |
| 4 | `WebsiteAccess.dc.html` | Website · members, meter and paywall | 5.5 | 1440 | – | – |
| 5 | `WebsiteStates.dc.html` | Website and pages · turn on, domain pending, approval, empty, loading, error, read-only | 5.1–5.5, 5.9, 5.10 | 1440 | – | – |
| 6 | `PublicHome.dc.html` | Public · website home | 5.6 | 720 | – | Yes |
| 7 | `PublicPost.dc.html` | Public · paid post at the paywall (visitor) | 5.7 | 720 | – | Yes |
| 8 | `PublicAccount.dc.html` | Public · member account (Sam Rivera) | 5.8 | 720 | – | Yes |
| 9 | `PublicSiteStates.dc.html` | Public · each reader, meter, sign-in, tag and archive, unpublished | 5.6–5.8 | 1440 | – | – |
| 10 | `PagesList.dc.html` | Pages · your pages and built-in pages | 5.9 | 1440 | Yes | Yes |
| 11 | `PageBuilder.dc.html` | Page builder · Field Guide sales page | 5.10 | 1440 | Yes | – |
| 12 | `PublicSalesPage.dc.html` | Public · sales page (The Deliverability Field Guide) | 5.11 | 720 | – | Yes |
| 13 | `PublicPagesStates.dc.html` | Public · lead magnet, webinar, launch, shop, checkout and download | 5.11, 5.14 | 1440 | – | – |
| 14 | `Products.dc.html` | Earn · products and orders | 5.12 | 1440 | Yes | Yes |
| 15 | `ProductEditor.dc.html` | Product editor · The Deliverability Field Guide | 5.13 | 1440 | – | – |
| 16 | `IntegrationsShopify.dc.html` | Integrations · Shopify (Acme Merch) | 5.15 | 1440 | Yes | Yes |
| 17 | `CommerceStates.dc.html` | Products and stores · connect, product blocks, abandoned-cart starter, states | 5.12–5.16 | 1440 | – | – |

That's 17 new boards: 13 at 1440 px and 4 public pages at 720 px. Plus 5 tablet and 9 phone boards. The two public states boards are 1440 px wide, holding 720 px frames side by side.

**Existing boards to update (section 6):**

| Board | Change |
|---|---|
| `FormBuilder.dc.html` (+ tablet, phone) | New tab set; "Offer a download after confirmation"; "Used on pages" |
| `FormsStates.dc.html` | New tab set; "Branding & domain" and what it applies to |
| `PublicArchive.dc.html` | Site header and navigation; issues link to posts |
| `PublicSubscribe.dc.html` (+ phone), `PublicConfirm.dc.html`, `PublicPreferences.dc.html` (+ phone), `PublicUnsubscribe.dc.html` | Site header when the website is on; the lead magnet's download on the confirmation page; the account link |
| `Composer.dc.html` (+ tablet, phone), `ComposerPreview.dc.html` | Web chip; Email / Web preview; new blocks; "Show on" in Visibility |
| `SendReadiness.dc.html` (+ tablet, phone) | Web line and web checks |
| `Campaigns.dc.html` (+ tablet, phone) | Web column |
| `Report.dc.html` (+ tablet, phone) | Web section; product and store rows in Revenue |
| `AutomationFlow.dc.html` (+ tablet, phone), `AutomationsList.dc.html` (+ tablet, phone), `AutomationsStates.dc.html` | New triggers and goals; revenue column; Abandoned cart starter |
| `Contacts.dc.html` (+ tablet, phone), `ContactProfile.dc.html` (+ phone) | New filters; Website, Purchases and Store orders cards; timeline events; consent sources |
| `SegmentBuilder.dc.html` (+ tablet) | Member, product and store conditions |
| `Roles.dc.html` (+ tablet, phone), `RoleEditor.dc.html` | Website and pages, Products and orders groups |
| `ApprovalPolicy.dc.html`, `ApprovalInbox.dc.html` (+ phone), `ApprovalReview.dc.html` (+ tablet, phone) | Post and page rule; posts in the inbox; web preview in the review |
| Phase 2 boards: `IntegrationsCatalog.dc.html` (+ tablet, phone), `IntegrationsWordPress.dc.html` | Commerce group: Shopify (Connected), WooCommerce; WooCommerce detected on a WordPress site |
| `SettingsKeys.dc.html` (+ tablet, phone), `SettingsAPI.dc.html` (+ phone), `SettingsMCP.dc.html` | New keys; webhook event groups; MCP tools |
| `Notifications.dc.html` (+ tablet, phone) | New notification kinds |
| `SettingsAudit.dc.html`, `AuditInstance.dc.html` | New actions and filter categories |
| Phase 1 boards: Transactional templates and settings | Product delivery starter; Purchases category |
| Phase 6 boards: Earn overview and tabs | Products tab; Product sales and Store revenue cards |
| `DSCoverage.dc.html`, `DSPatterns.dc.html`, `DSComponents.dc.html` | New boards in the coverage matrix; section 7 patterns |
