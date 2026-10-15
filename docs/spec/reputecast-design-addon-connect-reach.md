# ReputeCast — Add-on spec: integrations, languages and accessibility

Phase 2, "Connect and reach", of `reputecast-design-expansion-plan.md`.

## 0. How to use this document

Read all of it before you start.

- **The prototype from the main spec, the team add-on and phase 1 already exists.**
  - Add the screens in section 5 as a new canvas page, "P2 Integrations, languages & accessibility".
  - Apply section 6 to the existing screens, including phase 1's.
  - Reuse the existing navigation, components and design system. No new Settings tabs, no renames.
  - Add the patterns in section 7 to the "9 Design system" page first.
- **Same rules as before:**
  - Empty, loading, error and success states, plus the states listed for each screen.
  - A read-only view for members without the permission, which names the permission and who can help.
  - All copy in English. The exceptions are the Vietnamese and pseudo-locale boards of 5.10 and the sample translations in section 4.
  - Use the sample data in section 4, together with the main spec's, the team add-on's and phase 1's.
  - Desktop boards at 1440 px. Tablet (834) and phone (390) boards for 5.1, 5.2, 5.5, 5.6 and 5.11. Phone boards for 5.9, 5.13 and the pseudo-locale boards of 5.10. Section 9 lists every board.
- **Build order:**
  1. 5.1 Integrations catalog
  2. 5.2 Integration detail (Zapier)
  3. 5.3 Connect dialog
  4. 5.5 Settings › General: Languages and Accessibility
  5. 5.6 Composer: languages
  6. 5.7 Compare view
  7. 5.11 Composer: Accessibility panel
  8. 5.13 Review & send: Languages and Accessibility
  9. 5.12 Color picker with contrast
  10. 5.9 Language & region
  11. 5.10 Stress tests
  12. The rest, in order
  13. Then section 6

  If the work needs several passes, keep going until every screen exists.

---

## 1. Principles

1. **Everything is free.** No limits on integrations, sites, languages, keys or checks. No lock icon, plan badge or upgrade prompt. Zapier, Make, n8n and WordPress run on the workspace's own accounts.
2. **One key per integration, with the least it needs.**
   - Each integration gets its own key with only the permissions its triggers and actions need, and the key works only for those operations.
   - The key is shown once.
   - Disconnecting revokes the key and removes the webhooks it created.
3. **Say what leaves.** Every integration states what it receives and what it can change, as Connect HostRepute does.
4. **One language per reader, never a mix.** A contact gets the version in their language when it's ready, otherwise the fallback, whole. Half-translated emails and pages are never sent.
5. **Translations know when they're stale.** When the source changes, the translations that depend on it say so, block by block, until someone updates them or confirms they're still right.
6. **Fix it, don't just flag it.**
   - Every accessibility issue points to its element, and most have a one-click fix.
   - What a machine can add safely (`lang`, `dir`, `role="presentation"`) is added automatically.
7. **Warn by default.** Accessibility issues never block a send unless the workspace says so.
8. **Same access rules as the team add-on.**
   - Permission keys, the escalation guard, write-only secrets, audit entries.
   - MCP agents never connect integrations and never send.
9. **The app speaks the member's language; emails speak the reader's.** The app language is personal. Content languages belong to the workspace. One never changes the other.

---

## 2. Terms

- **Integration:** a connection between the workspace and an outside tool: Zapier, Make, n8n, a WordPress site or HostRepute. The REST API, webhooks and the MCP server are listed with them as developer tools.
- **Integration key:** an API key created for one integration, with a preset of permissions.
  - It works only for the operations that integration uses.
  - Keys & MCP shows it as "Managed by Zapier" (or n8n, WordPress…).
- **Trigger:** an event in ReputeCast that starts a workflow in another tool, such as "Contact subscribed". Delivered through a managed webhook.
- **Action:** something another tool does in ReputeCast, such as "Create or update a contact". A call made with the integration key.
- **Managed webhook:** a webhook endpoint an integration created through the API when a workflow with a trigger was turned on.
  - It can't be edited by hand.
  - It's removed when the workflow is turned off or the integration is disconnected.
- **Workflow:** a Zap in Zapier, a scenario in Make, a workflow in n8n. The apps send its name with each call, so the activity can say which one called.
- **Site:** a WordPress site connected through the ReputeCast plugin. Each site has its own key.
- **Workspace languages:** the languages the workspace writes emails and public pages in, set in Settings › General › Languages.
- **Default language:** the language new emails are written in, and the first language of the public pages.
- **Fallback language:** what a contact gets when an email has no ready version in their language, or when the contact has no language. English in the sample.
- **Contact language:** the existing contact field, holding a language code such as es-MX. A regional code matches its base language: es-MX gets Spanish.
- **Source:** the version an email was written in. The other versions are translated from it.
- **Language version:** the email in one language, with its own subject, preheader and content. The plan calls it a variant. The UI names it by its language ("Spanish"), because "variant" already means A/B variants in the composer.
- **Translation status** of a version:
  - **Up to date:** translated from the current source.
  - **Outdated:** the source changed after it was translated. The changed blocks are listed.
  - **Draft:** being translated, and not sent yet. Its readers get the fallback.
  - **Missing:** there's no version. Its readers get the fallback.
- **Page copy:** the text of the public pages, forms and confirmation email in one language. It has two parts: the publisher's own text, and ReputeCast's built-in labels.
- **App language:** the language of ReputeCast's own interface and of the emails it sends to members.
- **Region format:** how dates, times and numbers are written.
- **Pseudo-locale:** a test language. Every letter is accented, the text is about 40% longer, and each string is wrapped in ⟦ ⟧. Development only.
- **Accessibility issue:** a failure a machine can find in the compiled email.
  - **Serious** issues can block sending when the workspace says so.
  - **Moderate** issues only warn.
- **Decorative image:** an image marked as adding nothing to the text. It gets empty alt text, so screen readers skip it.
- **Contrast ratio:** the WCAG 2.2 contrast between text and its background.
  - AA needs 4.5:1, or 3:1 for large text (24 px, or 18.66 px bold).
  - AAA needs 7:1, or 4.5:1 for large text.
  - Ratios are rounded down to one decimal, so 4.49 never shows as 4.5.

---

## 3. Requirements

### 3.1 Integrations hub
- **Where:** Settings › Integrations becomes the catalog. No new tab.
- **Groups, in this order:**
  - Automation platforms: Zapier, Make, n8n.
  - Websites: WordPress.
  - Deliverability: HostRepute. It keeps its own flow and its own permission, integrations:hostrepute.
  - Developer: REST API, Webhooks, MCP server. These cards lead to the API reference, Webhooks and Keys & MCP.
  - Later phases add AI models (3), Verification (4), Payments (6) and Commerce (7). A group with nothing in it isn't shown.
- **Catalog card:**
  - logo, name and maintainer ("App by ReputeCast", "Plugin by ReputeCast", "By HostRepute")
  - one line on what it does
  - capability badges: Instant triggers, Actions, Signup forms, Post drafts…
  - a status badge
  - one fact line, for example "3 Zaps · last call 2 min ago"
  - Connect or Manage
- **Statuses:**
  - **Not connected.**
  - **Waiting for first call:** the key exists, but nothing has called with it yet.
  - **Connected.**
  - **Needs attention**, with the reason on the card:
    - calls rejected because the key was revoked or expired, or its owner left
    - the key expires within 7 days
    - webhook deliveries failing for 30 minutes
    - more than half of the calls failing for an hour
    - a site not heard from for 24 hours
  - **Update available:** a site's plugin is older than the latest release. Information, not a problem.
- **Detail page** per integration (5.2): what it can do, its workflows, calls and errors, managed webhooks, the connection, what it receives, and Disconnect.
- **Connecting** follows one pattern for every integration (5.3):
  1. **Choose what it may do.** Every trigger and action is listed with the permission it needs. All are on, except actions that need a ⚑ permission, which start off. Optional list scope and expiry.
  2. **Paste the instance URL and the key** into the other tool. The key is shown once, with "I copied the key".
  3. **Wait for the first call.** The dialog shows it arriving; the integration becomes Connected.
- **Integration keys:**
  - Named "{Integration} · {label}", for example "Zapier · Acme ops".
  - The escalation guard applies. Permissions you don't hold are locked, with the reason, and the triggers and actions that need them are listed as unavailable.
  - The key works only for the operations of its triggers and actions, even where a permission would allow more. Its webhooks:manage covers only the endpoints it created.
  - The key knows its workspace, so the other tool needs only the URL and the key.
  - **Change what it can do:** updates the key's permissions in place. Nothing to paste again.
  - **Rotate:** a new key, shown once. The old one keeps working for 24 hours so it can be replaced without failed runs. "Stop the old key now" is offered.
  - **Take over:** another member with integrations:manage becomes the key's owner, for example when the person who connected it leaves. The key keeps its value, and from then on it's limited by the new owner's permissions. Anything the new owner doesn't hold is listed before confirming.
  - **Revoke** stays possible from Keys & MCP, with a warning that names the workflows that stop.
- **Reachability:**
  - Zapier and Make call the instance from the internet. The connect dialog shows the public URL check from Instance admin (DNS, TLS, reachable).
  - When the instance isn't reachable from the internet, Zapier and Make can't connect, and the dialog says so. n8n can, when it runs on the same network.
  - Managed webhooks go through the SSRF guard. When a workflow tool's URL resolves to a private address, the dialog says an instance admin has to allow private addresses for webhooks.
- **Activity:**
  - **Calls:** the last 7 days of calls with the key: time, workflow, action, target, result and duration. Each error carries its fix.
  - **Deliveries:** the managed webhooks' deliveries, from the webhook log.
  - Request and response bodies follow the message log's rules: sensitive variables masked.
  - Contacts are kept by ID. Members with subscribers:view see the address; others see "a contact". Erased contacts show as "Erased contact".
  - The call log is kept for 7 days. It isn't the audit log: state-changing calls are also audited, with the key as actor, as today.
- **Disconnect:**
  - revokes the key at once
  - deletes the managed webhooks, with their queued and dead-letter deliveries
  - lists the workflows that stop, from the last 30 days of calls
  - keeps the contacts, events and messages created through it
- **Notification:** "Integration needs attention", to members with integrations:manage.

### 3.2 Zapier, Make and n8n
- **Published by the ReputeCast project:** a Zapier app, a Make app and an n8n community node (`n8n-nodes-reputecast`). They use only the public API, so anyone can build another integration the same way.
- **Connection fields:** instance URL and key. "Test connection" checks both.
- **Triggers**, instant, through managed webhooks:

  | Trigger | Event | Needs |
  |---|---|---|
  | Contact subscribed | `contact.subscribed` | subscribers:view |
  | Contact unsubscribed | `contact.unsubscribed` | subscribers:view |
  | Contact updated | `contact.updated` | subscribers:view |
  | Form submitted | `form.submitted` (new) | subscribers:view |
  | Campaign sent | `campaign.completed` | campaigns:view |
  | Transactional message bounced | `message.bounced` (phase 1) | transactional:view |

  - Turning on a workflow that uses a trigger creates its webhook. Turning it off removes it.
  - When the tool answers 410 Gone, ReputeCast removes that webhook, as Zapier's REST Hooks ask. The removal is audited with the system as actor.
  - Payloads are the Webhooks page's payloads, signed the same way. A managed webhook's signing secret goes once to the integration that created it, and is never shown in ReputeCast.
- **Actions:**

  | Action | What it does | Needs |
  |---|---|---|
  | Create or update a contact | Matches by email. Sets names, fields, language and tags, and adds to lists with single or double opt-in. | subscribers:create, subscribers:edit |
  | Add to a list | Single or double opt-in | subscribers:edit |
  | Add a tag | — | subscribers:edit |
  | Record an event | An API event with properties. Starts the automations that listen for it. | subscribers:edit |
  | Send a transactional email | A published template, a recipient, the data and an optional language | transactional:send ⚑ |

  - **Consent is kept.** An action never adds someone back to a list they left, and never adds a suppressed address. It answers 409 with the date and the reason, and the activity shows it.
  - **Transactional email** follows phase 1: published templates only, idempotency keys, the Transactional route.
- **Workflow names:** each app sends the workflow's name and ID with every call (`X-ReputeCast-Workflow`), when the platform provides them.
- **Out of scope**, as in the plan: native CRM sync and Slack notifications. A Zap can forward ReputeCast's webhooks to Slack.

### 3.3 WordPress plugin
- **Install:**
  - "ReputeCast" from the WordPress plugin directory, on any WordPress that allows plugins.
  - In WordPress › Settings › ReputeCast, paste the instance URL and the site's key, then Connect.
- **One key per site.** A site connects to one workspace.
- **Signup forms:**
  - a "ReputeCast signup form" block, and the shortcode `[reputecast_form id="frm_8c2e1a"]`
  - the block lists the workspace's forms
  - forms load through each form's public script, as embeds do today. They keep working after the site is disconnected.
  - the site must be one of the form's allowed domains. The block warns the WordPress editor when it isn't.
  - plugin 1.3.0: the form's language follows the page's language (WPML or Polylang)
- **Post drafts:**
  - A rule per site, set in ReputeCast: posts published in some categories become a campaign draft, with a chosen audience, template and sender. The rule names who is notified.
  - The draft takes the post's title as its subject and its excerpt as its preheader. The template's post section gets the featured image, title, excerpt and a "Read more" button.
  - Drafts only. Nobody gets anything until a person reviews and sends it, through Send Readiness and approvals as usual.
  - Plugin 1.3.0: a post's translations (WPML or Polylang) become the draft's language versions, marked Up to date.
- **Feed:** the plugin reports the site's feed URL. Recurring digests (phase 3) will list it as a source.
- **Health:**
  - The plugin reports in every 12 hours and with every post: plugin, WordPress and PHP versions, the feed URL, and where forms are used.
  - "Not heard from" after 24 hours.
  - "Check now" calls the site's status endpoint through the SSRF guard. ReputeCast never polls a site on a schedule.
  - "Update available" when a newer plugin exists. The latest version comes with the daily update check (Instance admin › Instance & updates). With the check off, the screen compares with the plugin version this release of ReputeCast knows.
- **Key permissions:**
  - subscribers:view, only to list the signup forms. The key can't read contacts.
  - campaigns:edit, only to create drafts from posts. The key can't open, change or send any other campaign.

### 3.4 Workspace languages
- **Settings › General › Languages:**
  - the workspace's languages, each with its contacts and the status of its page copy
  - the default language and the fallback language
  - Add language and Remove
  - how contacts get a language, and how many have none
- **Codes:** BCP 47. A workspace language is a base language (es). Regional codes match it (es-MX, es-ES).
- **Right-to-left languages** (Arabic, Hebrew, Persian, Urdu and others) aren't supported yet. They're listed but can't be added, and the dialog says why. Their contacts get the fallback.
- **A contact's language** uses the existing field.
  - **Signup forms** set it from the browser's first language, when the contact has none. A form never overwrites it.
  - **Imports** map a column to it. Values are normalized: "Spanish", "es_MX" and "ES" become es and es-MX. Unknown values are listed in the preview and left empty.
  - **The API, integrations and the preference center** set it. **Team members** edit it on the profile.
  - **A language the workspace doesn't offer** is kept (pt-BR), so adding Portuguese later works at once. Until then, those contacts get the fallback.
- **Which version a contact gets:**
  1. the version in their language, when it's Up to date or Outdated
  2. otherwise, the fallback language's version, when it's ready
  3. otherwise, the source
- **Removing a language:** its versions are kept but no longer sent, and come back if the language is added again. Its contacts keep their language and get the fallback. The dialog lists what has a version in it.
- **Changing the fallback** shows how many contacts it moves.
- **Contacts and segments:** Contacts gets a Language filter. Segments already have the language field; they show language names ("Language is Spanish").

### 3.5 Language versions
- **Applies to** campaigns, automation emails and transactional templates. Saved templates and sections stay in one language; inserting one into a version copies it into that version.
- **Source:** the default language, or another workspace language chosen when the email is created ("Written in German"). It can change while the email has only one version.
- **Add language:**
  - creates a Draft. It copies the source's layout and text, or the layout only, with empty text blocks.
  - "Translate with AI" is shown, disabled: "Not available yet". AI arrives in phase 3, with the workspace's own model key and never with contact data.
- **Each version has its own** subject, preheader, content, links and images. Personalization tokens are shared; their fallback text (`default: "there"`) is per version.
- **Tracking, block by block:**
  - Each block of a version remembers which source block and which source version it was translated from.
  - Editing a source block marks that block Outdated in every version translated from it. A toast offers "Keep them up to date", for edits that need no translating, such as a typo.
  - Changing a link or an image in the source offers "Apply to all languages".
  - A block added to the source shows in the versions as "New in English". A removed one shows as "Removed from English".
  - In a Draft, blocks still identical to the source show "Still in English".
- **Marking:**
  - A Draft is sent only after someone marks it translated.
  - An Outdated version is sent as it is, until someone updates it or marks it up to date.
- **A/B tests:** each language has its own A and B. The test runs across all languages and picks one winner for everyone.
- **Previews and tests:** "Preview as" a contact shows their version. "Send test" sends one language, or one of each.
- **Checks run on every version:** content, links and accessibility. A result that differs between versions names the language. Subject spam heuristics look for English phrases; other languages get the language-neutral ones (capitals, punctuation, length).
- **Approvals cover every version.** Changing any version invalidates an approval like any other edit, and the diff names the language.
- **Automations:** adding or changing a version in a live flow becomes part of the flow's next version, as today. The publish dialog lists Outdated and Draft versions.
- **Transactional templates:**
  - Publishing publishes every version together.
  - The API takes `language`. Without it, the linked contact's language is used, then the fallback.
  - The message log and message detail show the language sent and why.
- **Send Readiness › Languages**, a new group:
  - **Recipients per language** (Pass): who gets which version. Who gets the fallback, and why: no language, a language not offered, or no ready version. A Missing version doesn't warn.
  - **Outdated version** (Warn): Compare, "Mark as up to date", or "Send English to them instead".
  - **Draft version** (Warn): "506 Vietnamese readers get English unless Vietnamese is marked translated."
  - **Public pages** (Pass): a note when a language's readers will see the preference center and unsubscribe page in the fallback.
- **Reports:**
  - A "Results per language" section, when a campaign went out in more than one language: recipients, delivered, human opens, clicks, unsubscribes and complaints per version.
  - Readers who got the fallback have their own row.
  - The click map gets a language picker.

### 3.6 Public pages and forms
- **Page copy per language**, for every workspace language:
  - **the publisher's own text:** page headings and descriptions, list names and descriptions, consent text, the fewer-emails offer, the form's copy and after-submit message, the privacy policy link, and the confirmation email's intro
  - **ReputeCast's built-in labels:** buttons, field labels, messages, the confirmation email
- **Built-in labels ship** in the languages ReputeCast is translated into: English and Vietnamese at first, more as the community adds them. For other languages, the publisher writes them once (about 60 short strings), starting from "Copy English".
- **Status per language:** Live, Incomplete (n strings missing) or Not set up.
  - A language's pages and forms are used only when it's Live.
  - Until then, its visitors see the fallback language's pages, whole.
- **Which language a visitor sees:**
  - pages reached from an email (preference center, unsubscribe, view online): the contact's language
  - other pages: the language picked in the page's language menu, then the browser's languages, then the default
  - the language menu lists Live languages in their own names ("English · Español · Deutsch") and works without JavaScript
- **Preference center:** an "Email language" choice listing every workspace language, even one whose pages aren't Live yet. Changing it updates the contact's language.
- **Forms:**
  - copy per language, with the same Live rule
  - **Form language:** match the page it's on (the page's `lang`), follow the browser, or always one language. Embedded and popup forms match the page by default; the hosted page follows the browser.
  - The consent text has its own version per language, and the consent evidence keeps the language shown.
  - When the default language's consent text changes, the other languages' consent texts are flagged "Review the translation".
- **Confirmation emails** go out in the language the form was shown in.
- **Archive:** an issue page shows the version in the visitor's language, with "Also in: …" links and `hreflang` alternates. The archive lists issues in the visitor's language where a version exists.
- **Every page sets `lang` and `dir`** from the language shown.

### 3.7 App languages
- **Account & security › Language & region**, a new section after Profile:
  - **App language:** "Instance default (English)", English or Tiếng Việt. Each language is named in its own language.
  - **Region format:** how dates, times and numbers look. By default it follows the app language: ReputeCast standard for English, Tiếng Việt (Việt Nam) for Vietnamese.
    - "ReputeCast standard" keeps today's formats: Tue 14 Oct 2026 · 09:00 · 48,213 · 99.1%.
    - Others come from Unicode CLDR: English (United States), Tiếng Việt (Việt Nam), Deutsch (Deutschland) and more.
  - **Time** (24-hour or 12-hour) and **week starts on** (Monday or Sunday). Both follow the region format unless changed.
  - A live preview line.
  - Times stay in the workspace's timezone, as today, and the section says so. The personal timezone stays in Profile, where it sets quiet hours.
  - When the browser prefers a language the app has, and the member hasn't chosen one, the app offers it once, in a banner they can dismiss.
- **Instance default:** Instance admin › Instance & updates › Instance gets "Default language" and "Default region format". They apply to:
  - the sign-in page
  - accounts that haven't chosen
  - emails to people without an account, such as invitations
- **Sign-in, invitation and password pages** follow the browser when the app has its language, with a language menu at the bottom.
- **Emails to members** use their app language: invitations, password reset, email change, two-factor reset, notifications, approval requests. The invite dialog gets "Invitation language", which defaults to the instance default.
- **English and Vietnamese first,** left-to-right only.
  - A language appears in the picker once its translation is complete for that release.
  - "Help translate ReputeCast" links to the translation guide on GitHub.
- **Never translated:** workspace content (campaign, list, template and form names), permission keys, audit action names, API error codes, provider names and codes, DNS records, code samples, CLI output.
- **Messages from the server** (errors with their fix, "who can help", notifications) follow the member's app language. API clients get English unless they send Accept-Language.
- **Rules for the build:**
  - messages in ICU format, with plurals
  - no sentence assembled from pieces
  - sorting follows the language's rules
  - IBM Plex Sans's Vietnamese subset, already self-hosted
  - `<html lang>` set to the app language
- **Pseudo-locale:** "Pseudo (+40%)" appears in the picker in development, or when the server runs with the pseudo-locale turned on. Strings that aren't translated show without ⟦ ⟧, so hard-coded text stands out.
- **Stress tests** (5.10) settle the truncation and wrapping rules. They go into the design system (section 7).

### 3.8 Accessibility checks
- **What's checked:** the compiled HTML of every language version.
- **When:**
  - in the composer, as you edit
  - in Send Readiness
  - when publishing an automation or a transactional template
  - for every template in the gallery
- **Issues:**

  | Issue | Severity | Fix |
  |---|---|---|
  | Image without alt text | Serious | Write alt text, or mark it "Decorative image". A linked image can't be decorative: its alt text is the link's text. |
  | Link or button with no text | Serious | Add text, or alt text to the linked image. A linked image without alt text is one issue, not two. |
  | Text contrast below AA | Serious | One click: use a suggested color |
  | Link text that says nothing: "click here", "here", "read more", a bare URL | Moderate | Use the linked page's title, which the link check already reads, when the page is in the version's language. Or write your own. |
  | A skipped heading level, such as Heading 3 after Heading 1 | Moderate | One click: change the level. The text keeps its look. |
  | Text below the minimum size (12 px by default) | Moderate | One click: set it to the minimum |

- **Phrases that say nothing** are listed for English, Spanish, German, French, Portuguese and Vietnamese at first. Other languages get the bare-URL and no-text checks only.
- **Added automatically** when the email is compiled, and shown as passing checks:
  - `lang` and `dir` from the version's language. `dir` is always `ltr` for now.
  - `role="presentation"` on layout tables. MJML adds it to most; ReputeCast adds it to any it missed, such as tables inside Raw HTML blocks. Data tables (the Table block, or tables with header cells) keep their meaning.
- **Headings:** Text blocks get a "Text role" setting: Paragraph, Heading 1, Heading 2 or Heading 3. The Hero block's headline is Heading 1 and the Article card's title is Heading 2, by default.
- **Contrast** compares a text color with the nearest background color behind it: the element's, then the column's, section's, wrapper's and body's.
  - Over a background image it can't check. It says "Check it by eye" and doesn't count an issue.
  - Picking a background compares it with every text color inside, worst first.
- **One issue in several versions** is listed once, with its languages.
  - Fixes that involve no words (heading level, size, color) apply to every version.
  - Alt text and link text are fixed in each language.
- **Alt text moves** from the Content checks to Accessibility, in the composer and in Send Readiness. Under the default policy it's still a warning. The image-to-text ratio stays in Content.
- **Workspace policy,** in Settings › General › Accessibility (workspace:manage):
  - When serious issues remain: Warn (default) or Block.
  - Block applies to sends (Send Readiness turns Blocked), to publishing an automation and to publishing a transactional template. Live automations and published templates keep sending; their next version needs the fix.
  - Minimum text size: 12 px by default, adjustable from 12 to 16 px.
  - Changing the policy shows its impact first: scheduled campaigns that would be held at their send-time recheck, and drafts and flows with serious issues.
- **Send Readiness › Accessibility,** a new group:
  - one row per serious issue
  - the moderate issues together in one row. "Fix in one click" applies the fixes that need no words (heading levels, text size) to every version; link text is fixed in the composer, language by language.
  - the automatic additions as passing rows
  - one-click fixes are audited and re-run the checks, like the existing fixes
- **Template gallery:** each template shows "All clear" or its issues ("2 issues · 1 serious"). The built-in templates and starters are all clear.

### 3.9 Permissions
New permission group, "Integrations":
- see the integrations hub, each integration's health, activity and managed webhooks (integrations:view)
- connect, change, take over and disconnect integrations, which creates, changes and revokes their keys and webhooks (integrations:manage) ⚑

Who gets what:
- **Built-in roles:**
  - Owner and Admin: both.
  - Editor and Viewer: integrations:view.
- **Content Writer**, the sample's custom role: integrations:view.
- **People only:** connecting, changing, rotating, taking over and disconnecting need a signed-in member. API keys and MCP agents can't hold integrations:manage.
- **HostRepute** keeps integrations:hostrepute.
- **Existing permissions cover the rest:**
  - workspace languages, the fallback and the accessibility policy: workspace:manage
  - page and form copy per language: forms:edit
  - language versions and accessibility fixes: the permission that edits that email (campaigns:edit, automations:edit or transactional:edit)
  - the instance default language: instance admins
  - your own app language and region format: every member
- **MCP agents:**
  - get_send_readiness returns the two new groups
  - update_campaign_content takes a language
  - new read tools: get_campaign_languages and get_accessibility_issues
  - agents never connect integrations

### 3.10 Audit
- **Audited:**
  - an integration connected, its permissions changed, its key rotated, taken over or revoked, the integration disconnected
  - a managed webhook created or removed. The actor is the integration's key, or the system after a 410 Gone.
  - a WordPress site connected or disconnected, its post-draft rule changed, a draft created from a post (actor: the site's key)
  - a workspace language added or removed; the default or the fallback changed
  - page or form copy changed, per language
  - a language version added, removed, marked translated or marked up to date
  - an accessibility fix applied from Send Readiness; the accessibility policy changed
  - a member's app language or region format changed; the instance default language changed
- **Not in the audit log:** reads by integration keys. The 7-day call log (3.1) shows them, for troubleshooting.

### 3.11 API and webhooks
- **New webhook event** `form.submitted`, in a new Forms group. It fires when a signup is confirmed (at once for single opt-in), with the form and the answers: the same moment as the "Form submitted" automation trigger.
- **Contacts:** create or update by email. `language` is accepted and normalized; an invalid value answers 422 with the closest match.
- **Content:** the campaign, automation-email and transactional template endpoints take `language`. A languages resource lists each email's versions and statuses.
- **Transactional:** `language` on send, batch and render (phase 1's API).
- **Reports:** results per language.
- **Integrations:** list, detail and activity (integrations:view); connect, change, rotate, take over and disconnect (integrations:manage, people only).
- **Webhooks API:** endpoints an integration key created are marked managed. Other keys can't see them.

### 3.12 HostRepute
- HostRepute's card moves into the catalog's Deliverability group. Its detail page is today's connected view.
- Nothing else changes: same flow, same permission, same data boundary.
- Nothing in this phase sends contacts, content or statistics to HostRepute. Accessibility checks are one-off checks on an email. Integration health is passive: sites report in, and "Check now" is on demand. There's no new monitoring.

### 3.13 Out of scope for this phase
- Native CRM sync and Slack notifications, as the plan says.
- Right-to-left languages, for emails, pages and the app.
- AI translation, alt text and fixes (phase 3). Their buttons show, disabled.
- Exporting and importing translations for outside translators.
- A different send time per language.
- App languages beyond English and Vietnamese, until the community adds them.
- Accessibility checks that need a person: reading order, whether alt text is accurate, color as the only signal.

---

## 4. Sample data

### Sample time
Wednesday 15 Oct 2026, 14:20, America/New_York (the workspace timezone). Phase 1's samples end at 10:42 the same day.

### Integrations (Acme Newsletter)
| Integration | Status | Connected | Facts |
|---|---|---|---|
| Zapier | Connected | Marcus Chen, 24 Sep 2026 | key "Zapier · Acme ops" (rc_live_5d1e…7b20) · 3 Zaps · 169 calls and 3 errors in 7 days · last call 14:18 |
| Make | Not connected | — | — |
| n8n | Connected | Linh Nguyen, 29 Sep 2026 | key "n8n · Acme automations" (rc_live_9a40…e1c3) · 3 workflows · 1 managed webhook · last call 14:14 |
| WordPress | Needs attention | Marcus Chen | 2 sites, one key each · staging.acme.io not heard from since 13 Oct · update available for blog.acme.io |
| HostRepute | Connected | Linh Nguyen, 20 Aug 2026 | Acme Inc · synced 4 min ago, as today |
| Webhooks | 4 endpoints · 1 failing | — | the three on the Webhooks board, plus n8n's managed one |
| MCP server | On · 2 agents | — | as today |

"Zapier · forms", the plain API key Sofia Rossi made before the hub existed (TeamDialogs), isn't part of the Zapier integration. It stays an ordinary key.

### Zapier
- **Permissions:** subscribers:create, subscribers:edit, campaigns:view, transactional:view, transactional:send ⚑, webhooks:manage (its own endpoints). All lists. No expiry.
- **Zaps** seen in the last 30 days:

  | Zap | Uses | Calls · 7 days | Errors | Last run |
  |---|---|---|---|---|
  | Beta application (Typeform) → ReputeCast contact | Create or update a contact: Beta Program, tag beta-applicant, double opt-in | 128 | 1 | 14:18 |
  | HubSpot deal won → tag customer | Add a tag: customer | 37 | 0 | 12:51 |
  | Stripe payment failed → payment-failed email | Send a transactional email: payment-failed | 4 | 2 | 13:55 |

- **Errors:**
  - 15 Oct 11:52 and 12:06 · Stripe payment failed → payment-failed email · 422 missing_variables. "amount_due is required. In the Zap's ReputeCast step, map Stripe's Amount Due to it." Marcus Chen fixed the Zap at 12:15; the 12:40 and 13:55 runs were delivered.
  - 14 Oct 16:12 · Beta application (Typeform) → ReputeCast contact · 409 unsubscribed. "rob@example.net left Beta Program on 2 Oct. ReputeCast doesn't add people back to a list they left."
- **Managed webhooks:** none. All three Zaps use actions only.
- **What Zapier receives:** nothing until a Zap uses a trigger.

### payment-failed (transactional template)
- Billing category (Essential). v1, published 15 Oct 11:30 by Sofia Rossi.
- Variables: `first_name` (text, optional), `amount_due` (text, required), `update_url` (URL, required, sensitive).
- Sent only by the Stripe Zap.

### n8n
- **Server:** n8n.acme.io, self-hosted by Acme, on a public address.
- **Permissions:** subscribers:create, subscribers:edit, webhooks:manage (its own endpoints).
- **Workflows:**

  | Workflow | Uses | Last 7 days |
  |---|---|---|
  | Acme Cloud: workspace created | Record an event `workspace.created`, which starts the Onboarding automation | 61 calls |
  | Acme Cloud: teammate invited | Record an event `teammate.invited`, the Onboarding goal | 23 calls |
  | Unsubscribes → Acme Cloud CRM | Trigger: Contact unsubscribed | 14 deliveries |

- **Managed webhook:** `https://n8n.acme.io/webhook/7c1e…/reputecast` · contact.unsubscribed · created 29 Sep · Healthy.
- **Failing (a state):**
  - Since 13:05, n8n.acme.io answers 502 Bad Gateway. 3 deliveries are retrying; the next try is at 14:35.
  - After 8 attempts over 6 hours, deliveries move to the dead-letter queue, kept for 7 days.
  - The dead-letter state the next day: 5 deliveries, with "Replay all".

### Make
- Not connected.
- In 5.3, Marcus Chen connects it at 14:25 as "Make · Acme ops", leaving "Send a transactional email" off.
- The key, shown once: `rc_live_7f0c3d9e41a8b62f05c7e1d4b3a2`.
- Make's first call, "Test connection", arrives at 14:31.

### WordPress sites
| Site | Connected | Plugin | Last heard from | Forms on the site | Post drafts |
|---|---|---|---|---|---|
| blog.acme.io | Marcus Chen, 25 Sep 2026 | 1.2.0 · 1.3.0 available | 6 min ago | Newsletter signup (block, sidebar and About page); Beta apply (shortcode, "Join the beta" page) | Posts in Announcements → a draft for Product Updates, template "Blog post", from Acme Product <product@news.acme.io>, notify Sofia Rossi |
| staging.acme.io | Marcus Chen, 2 Oct 2026 | 1.3.0 | 13 Oct 22:10 · Check now at 14:02 timed out after 10 s | Newsletter signup (block, 1 place). Its signups are rejected: staging.acme.io isn't an allowed domain (as on FormsStates). | Off |

- **Last draft from blog.acme.io:** the post "Scheduled rollouts are here", published 15 Oct 10:04. Its Product Updates draft was created at 10:05, and Sofia Rossi was notified.
- **Feed:** https://blog.acme.io/feed/, not used yet.
- **What 1.3.0 adds:** WPML and Polylang translations as language versions, and the form block's language setting.

### Workspace languages
| Language | Codes | Contacts | Page copy | Added |
|---|---|---|---|---|
| English | en | 41,020: 35,500 English, 5,508 with no language, 12 with languages Acme doesn't offer (pt-BR 7, fr 3, it 2) | Live · default | — |
| Spanish (Español) | es, es-MX, es-ES… | 4,180 | Live · Sofia Rossi, 8 Oct 2026 | Linh Nguyen, 6 Oct 2026 |
| German (Deutsch) | de, de-AT, de-CH… | 2,240 | Live · Sofia Rossi, 8 Oct 2026 | Linh Nguyen, 6 Oct 2026 |
| Vietnamese (Tiếng Việt) | vi | 773 | Not set up: built-in labels ready, Acme's own text not written. Visitors see English. | Linh Nguyen, 13 Oct 2026 |

- 48,213 contacts in all. Default and fallback: English.
- Linh added Vietnamese because 773 contacts already had it, from signup forms and the Brevo import.
- **Weekly Digest list, 31,604 recipients after exclusions:**
  - English 26,890: 23,269 English, 3,613 with no language, 8 with other languages
  - Spanish 2,740
  - German 1,468
  - Vietnamese 506

### Contacts in the examples
| Contact | Language | Set by |
|---|---|---|
| Sam Rivera, sam.rivera@outlook.com | English (en) | Brevo import, 18 Sep 2026 |
| Lucía Fernández, lucia.fernandez@example.com | Spanish (es-MX) | Newsletter signup form, from the browser, 3 Oct 2026 |
| Jonas Weber, jonas.weber@example.de | German (de-AT) | Preference center, 9 Oct 2026 |
| Phạm Minh Anh, minhanh.pham@example.vn | Vietnamese (vi) | Brevo import, 18 Sep 2026 |
| Rafael Souza, rafael.souza@example.com.br | Portuguese (pt-BR), not offered: gets English | API, 12 Oct 2026 |

### Weekly Digest #143 (draft)
- **For** Tue 21 Oct 2026, 09:00 America/New_York; not scheduled yet. Weekly Digest list, 31,604 recipients (the list hasn't changed since #142). Written by Sofia Rossi. English v9, last edited 15 Oct 09:12.
- **Versions:**

  | Language | Status | Subject | Preheader | Translated |
  |---|---|---|---|---|
  | English | Source | Scheduled rollouts, faster exports, and 6 weeks left on the v1 API | Plan a flag change for 2 a.m. and sleep through it. | — |
  | Spanish | Up to date | Despliegues programados, exportaciones más rápidas y 6 semanas para la API v1 | Programa un cambio de flag para las 2 a. m. y sigue durmiendo. | Daniel Park, 15 Oct 11:20, from v9 |
  | German | Outdated · 2 blocks | Geplante Rollouts, schnellere Exporte und noch 6 Wochen für die v1-API | Planen Sie eine Flag-Änderung für 2 Uhr nachts und schlafen Sie weiter. | Sofia Rossi, 14 Oct 16:20, from v6 |
  | Vietnamese | Missing | — | — | 506 readers get English |

- **Blocks**, 9 in all:
  1. Header: logo linked to acme.io (alt "Acme"), and Blog · Docs · Changelog at 12 px
  2. Hero image scheduled-rollouts.png, 552×200, with no alt text
  3. Headline "Scheduled rollouts are here" (Heading 1)
  4. Intro: "Hi {{ first_name | default: "there" }}, you can now schedule flag changes ahead of time, in any timezone. Click here to see how it works."
  5. Two cards, each with an image: "Scheduled rollouts" with "Read the docs", and "Faster CSV exports" ("Exports of up to a million rows now finish in under a minute.") with "See the changelog". Both titles are Heading 3.
  6. "6 weeks left on the v1 API": "The v1 API stops answering on 30 November. Migration guide: https://docs.acme.io/migrate-v2"
  7. Button "Plan your migration": white text on #5EBFB0
  8. Social links: X, LinkedIn, GitHub, each with alt text
  9. Footer: address, Unsubscribe · Email preferences · View in browser, at 11 px
- **Changed in English after German was translated** (v6 → v9, Sofia Rossi, 15 Oct 09:12):
  - Intro: "…you can now schedule flag changes ahead of time." → "…you can now schedule flag changes ahead of time, in any timezone."
  - "Faster CSV exports" card: "Exports now run up to 3× faster." → "Exports of up to a million rows now finish in under a minute."
- **Accessibility:** 6 issues, 2 serious and 4 moderate. Each is in English, Spanish and German ("Haz clic aquí", "Hier klicken").
  1. Serious · the hero image has no alt text.
  2. Serious · "Plan your migration": white on #5EBFB0 is 2.2:1, and needs 4.5:1. Suggested:
     - #0F766E, the color of the other buttons (5.4:1)
     - #348377, the nearest shade that passes AA (4.5:1)
     - #276259, which passes AAA (7.0:1)
  3. Moderate · "Click here" in the intro. Suggested in English: the page title "Scheduled rollouts · Acme Docs". The docs are in English only, so Spanish and German get a field to write the words.
  4. Moderate · the bare URL https://docs.acme.io/migrate-v2. Suggested in English: "Migrating to the v2 API · Acme Docs".
  5. Moderate · both card titles are Heading 3, right after the Heading 1. Fix: Heading 2.
  6. Moderate · footer text at 11 px; the minimum is 12 px.
- **Passing:** `lang` and `dir` set for every version (en, es, de; ltr); `role="presentation"` on 16 layout tables; 6 of 7 images have alt text; 13 of 15 links say where they go.
- **HTML size:** English 66 KB, Spanish 68 KB, German 71 KB, all under Gmail's 102 KB.

### Weekly Digest #142 (sent Tue 14 Oct, 09:00)
- English source; Spanish and German up to date. No Vietnamese version, so 506 Vietnamese readers got English. Accessibility: all clear.
- **Subjects:**
  - English: "Three launches, one deprecation, and the roadmap for Q4"
  - Spanish: "Tres lanzamientos, una retirada y la hoja de ruta para el cuarto trimestre"
  - German: "Drei Launches, eine Abkündigung und die Roadmap für Q4"
- **Results per language,** as of Wed 15 Oct 14:20 (still coming in):

  | Version | Readers | Sent | Delivered | Human opens | Clicks | Unsubscribes | Complaints |
  |---|---|---|---|---|---|---|---|
  | English | English, no language, other languages | 26,890 | 99.2% | 39.8% | 6.1% | 0.21% | 0.03% |
  | Spanish | Spanish | 2,740 | 99.2% | 44.1% | 7.4% | 0.18% | 0.04% |
  | German | German | 1,468 | 99.3% | 46.3% | 6.9% | 0.27% | 0.00% |
  | English, as the fallback | Vietnamese | 506 | 99.2% | 31.5% | 3.8% | 0.40% | 0.00% |

- **Caption:** "Vietnamese readers got English. Their human-open rate is 8.3 points below English readers'."

### Automations and transactional templates
- **Welcome series v4 (draft):** steps 1 to 3 get Spanish and German versions. Step 2's German version is Outdated: its intro changed after translation. Vietnamese readers get English in every step.
- **password-reset v4:** published 15 Oct 12:30 by Sofia Rossi, in English, Spanish and German. At 14:02 it went to lucia.fernandez@example.com: "Spanish, from the contact" · Delivered · 1.2 s · Postmark.

### Page and form copy
| | English | Spanish | German | Vietnamese |
|---|---|---|---|---|
| Form heading | The Acme newsletter | El boletín de Acme | Der Acme-Newsletter | — |
| Button | Subscribe | Suscribirme | Abonnieren | — |
| Consent text | I agree to receive emails from Acme. Unsubscribe any time with one click. See the privacy policy. (v3) | Acepto recibir correos de Acme. Puedo darme de baja cuando quiera con un clic. Consulta la política de privacidad. (v1) | Ich möchte E-Mails von Acme erhalten. Abmelden geht jederzeit mit einem Klick. Siehe Datenschutzerklärung. (v1) | — |
| Status | Live · default | Live | Live | Not set up: visitors see English |

- The Newsletter signup form's language: "Match the page it's on".
- The hosted page at news.acme.io/subscribe follows the browser.

### Accessibility policy and gallery
- **Policy:** Warn, minimum text size 12 px.
- **Block state:** Linh Nguyen switches it to Block at 14:22. The impact preview lists:
  - Weekly Digest #143 (draft): 2 serious issues.
  - Welcome series v4 (draft), step 3 B: 1 serious issue (button contrast). The live v3 keeps sending; publishing v4 needs the fix.
- **Gallery results:**
  - Blank, Weekly digest, Product update, Plain text, Announcement, Product grid: All clear.
  - Acme digest v3 (saved): 2 issues · 1 serious. The serious one: the linked logo has no alt text, so its link has no text. The other: two "Read more" links.
  - Copy of October Product Update: All clear.

### App languages
- **Instance default:** English, ReputeCast standard.
- **Members:**
  - Linh Nguyen: English, ReputeCast standard, as on every board.
  - Marcus Chen: English, region English (United States). He sees "Tue, Oct 14, 2026 · 9:00 AM".
  - In 5.9, Linh tries Tiếng Việt.
- **Invitations:** jo@contractor.dev's invitation is in English, the instance default.
- **Region formats:**

  | Region format | Date | Time | Number | Percent | Cost |
  |---|---|---|---|---|---|
  | ReputeCast standard | Tue 14 Oct 2026 | 09:00 | 48,213 | 99.1% | $2.84 |
  | English (United States) | Tue, Oct 14, 2026 | 9:00 AM | 48,213 | 99.1% | $2.84 |
  | Tiếng Việt (Việt Nam) | Th 3, 14 thg 10, 2026 | 09:00 | 48.213 | 99,1% | 2,84 US$ |
  | Deutsch (Deutschland) | Di., 14. Okt. 2026 | 09:00 | 48.213 | 99,1 % | 2,84 $ |

- **Vietnamese glossary.** These are drafts; a native speaker reviews them before the boards are final.

  | English | Tiếng Việt |
  |---|---|
  | Overview | Tổng quan |
  | Campaigns | Chiến dịch |
  | Automations | Tự động hóa |
  | Audience | Người đăng ký |
  | Forms & pages | Biểu mẫu và trang |
  | Deliverability | Khả năng vào hộp thư |
  | Sending providers | Nhà cung cấp gửi thư |
  | Settings | Cài đặt |
  | New campaign | Chiến dịch mới |
  | Scheduled · Draft · Sent | Đã lên lịch · Bản nháp · Đã gửi |
  | Recipients | Người nhận |
  | Human opens | Lượt mở bởi người đọc |
  | Review & send | Kiểm tra và gửi |
  | Ready with warnings | Sẵn sàng, có cảnh báo |
  | Schedule send to 31,604 recipients | Lên lịch gửi cho 31.604 người nhận |
  | Team · Invite member · Last active | Nhóm · Mời thành viên · Hoạt động gần nhất |

- **Pseudo-locale example:** "Schedule send to 31,604 recipients" becomes "⟦Šçĥéðûļé šéñð ţö 31,604 ŕéçîþîéñţš ·············⟧".

### People
- **Marcus Chen** (Admin) connected Zapier and both WordPress sites, and connects Make in 5.3.
- **Linh Nguyen** (Owner) connected n8n, manages the workspace languages and the accessibility policy, and sets the instance default language.
- **Sofia Rossi** (Editor) writes Weekly Digest #143 and translated it into German. She wrote the Spanish and German page copy, publishes payment-failed and gets the post-draft notifications. She sees integrations but can't connect them.
- **Daniel Park** (Content Writer) translated #143 into Spanish. He can add a language version to an email, but not a workspace language, and can't connect integrations.
- **Aisha Bello** (Viewer) sees integrations, languages and accessibility results, read-only.

---

## 5. New screens

### 5.1 Integrations catalog (Settings › Integrations)
- **Header:** the Settings tabs, with Integrations active. Title "Integrations"; caption "Connect the tools you already use. Every integration is free and uses your own accounts."
- **Attention banner,** when an integration needs attention: it names the problem and links to it. Sample: "WordPress: staging.acme.io hasn't reported in since 13 Oct 22:10." At most three banners; beyond that, one line says "and 2 more".
- **Controls:** search, and pill filters: All · Connected · Automation platforms · Websites · Deliverability · Developer.
- **Groups,** each with a heading and one line:
  - **Automation platforms:** "Start workflows from ReputeCast events, or change contacts and send email from other tools. No code."
  - **Websites:** "Signup forms and post drafts from your WordPress site."
  - **Deliverability:** "Reputation data from outside ReputeCast."
  - **Developer:** "Build your own with the API, signed webhooks and the MCP server."
- **Cards** (section 7):
  - Zapier · Connected · "3 Zaps · last call 2 min ago" · Instant triggers, Actions · Manage
  - Make · Not connected · "6 triggers · 5 actions" · Instant triggers, Actions · Connect
  - n8n · Connected · "3 workflows · last call 6 min ago" · Instant triggers, Actions · Manage
  - WordPress · Needs attention · "2 sites · 1 not heard from · 1 update available" · Signup forms, Post drafts · Manage
  - HostRepute · Connected · "Acme Inc · synced 4 min ago" · Blocklists, Postmaster, SNDS, DMARC · Manage
  - REST API · "OpenAPI 3.1 reference · keys in Keys & MCP" · Open
  - Webhooks · "4 endpoints · 1 failing" · Signed, Retries and replay · Manage
  - MCP server · "On · 2 agents" · Scoped agents · Manage
- **Footer note:** "Integrations use the public API. Anyone can build another one the same way." with a link to the API reference.

States to show:
- the sample
- a new workspace with nothing connected: every card Not connected; HostRepute's card has the invitation it has on Deliverability
- n8n's webhook failing, in the banner and on its card
- search with no results: "No integration called 'salesforce'. Zapier, Make and n8n connect ReputeCast to most other tools."
- read-only (Aisha Bello): Connect and Manage become View, and a line says "Connecting needs integrations:manage. Ask Marcus Chen or Linh Nguyen."
- loading, error
- a layout check, not built in this phase: the catalog once phases 3, 4, 6 and 7 add AI models, Verification, Payments and Commerce, to check that the groups and filters scale

### 5.2 Integration detail (Zapier)
A page under the catalog: Settings › Integrations › Zapier. Make and n8n use the same layout.
- **Header:**
  - logo, name, status badge, and "Connected by Marcus Chen on 24 Sep 2026"
  - actions: Open Zapier, Change what Zapier can do, Rotate key, and a menu with Take over and Disconnect
- **Numbers:** calls in 7 days (169), errors (3), last call (14:18, 2 min ago), managed webhooks (0).
- **Zaps using ReputeCast:** the Zaps seen in the last 30 days, with what they use, calls and errors in 7 days, and last run. A call without a name is listed under "Unnamed Zap".
- **Calls and errors:**
  - the call log (3.1), filtered by All or Errors, and by 24 hours or 7 days
  - each error explains itself and says what to change in Zapier
  - a row opens the request and the response, with secrets masked
- **Webhooks Zapier manages:** event, endpoint (masked), created, deliveries in 24 hours, status, and a link to the delivery log. Empty: "None. Zaps that start with a ReputeCast trigger add their webhook here, and remove it when they're turned off."
- **What Zapier can do:** the triggers and actions of 3.2, with the permission each needs, marked Available or Off. An Off item says why: "Turned off when connecting", or "The key lacks transactional:send, which Marcus Chen doesn't hold."
- **Connection:**
  - the instance URL, with copy
  - the key: name, masked value, created, last used (time, IP, client), expiry
  - permissions as chips, ⚑ flagged, and the list scope
  - "Shown once, on 24 Sep. If it's lost, rotate it."
- **What Zapier receives:** the data of the triggers in use. With none: "Nothing yet. Your Zaps only use actions."
- **Disconnect,** at the bottom.

States to show:
- connected with activity (the sample), with the Errors filter on
- waiting for the first call, right after connecting
- not connected (Make): what it can do, connecting in three steps, and Connect
- key revoked: "Linh Nguyen revoked this key in Keys & MCP at 13:12. 6 calls were rejected since (401 key_revoked). Reconnect creates a new key to paste in Zapier."
- key expired, with the same Reconnect
- owner left: "Marcus Chen was deactivated at 13:40. The key has lost its permissions, and calls fail with 403. Take over to keep Zapier running; the key stays the same."
- webhook deliveries failing (n8n): the managed webhook's status, the retries and the next try
- deliveries in the dead-letter queue (n8n): 5 deliveries, with "Replay all" once n8n is back
- read-only (Aisha Bello): no actions; the activity is visible
- loading, error

### 5.3 Connect dialog
Three steps in one dialog. The board shows Marcus Chen connecting Make.
1. **Choose what Make can do.**
   - Triggers and actions as in 3.2, each with its permission and a switch.
   - "Send a transactional email" starts off, because it needs ⚑ transactional:send.
   - The resulting permissions, updated as switches change.
   - Key name ("Make · Acme ops"), list scope (All lists), expiry (None).
   - "What Make will receive": the data of the triggers you turn on.
   - Create key.
2. **Paste these into Make.**
   - The instance URL https://reputecast.acme.io, with copy. Under it, the public URL check: "Reachable from the internet · checked 4 min ago".
   - The key, shown once, with copy and "I copied the key".
   - Three short steps in Make: add a ReputeCast module, create a connection, paste both. "Open Make" opens Make in a new tab.
3. **Wait for Make.**
   - A live line: "Waiting for Make's first call…", then "Make called at 14:31 · Test connection · 200".
   - Done opens the detail page.
   - Closing early keeps the key; the card then says "Waiting for first call".

States to show:
- steps 1, 2 and 3, for Make
- escalation guard, for a custom role that holds integrations:manage but not transactional:send: that action is locked, with "You can't grant transactional:send because you don't hold it. Ask Linh Nguyen or Marcus Chen." Everything else stays available.
- instance not reachable: "Make can't reach http://10.0.4.12:8080, your public URL. Give the instance a public address (Instance admin › Instance & updates), or use n8n on your own network." Steps 2 and 3 are disabled.
- n8n on a private address: a note that webhooks to private addresses need an instance admin to allow them, naming the instance admins
- no call after 10 minutes: "No call yet. Check the URL for typos, and that the whole key was pasted. You can close this; the key stays valid until you disconnect."
- the Disconnect dialog, for Zapier: what stops (the three Zaps by name), what's removed (the key, its webhooks and their queued deliveries), what stays (contacts, events, messages). The button reads "Disconnect Zapier".

### 5.4 WordPress sites
A page under the catalog: Settings › Integrations › WordPress.
- **Sites:** each site with its status, plugin version, last heard from, forms used on it and drafts created.
- **A site, expanded:**
  - **Health:** plugin, WordPress and PHP versions, last heard from, Check now.
  - **Signup forms** on the site, where they're used, and whether the site is an allowed domain for each.
  - **Post drafts:** the rule (categories, audience, template, sender, who's notified), on or off, and the latest drafts, with links.
  - **Feed:** the URL, "Not used yet".
  - **Key:** Rotate, Take over, Disconnect.
- **Connect a site:** the 5.3 dialog with WordPress's steps:
  1. what the site can do: list forms, create drafts
  2. install the plugin, then paste the URL and the key in WordPress › Settings › ReputeCast
  3. wait for the site to report in
- **Post-draft rule dialog:** categories (from the plugin), audience, template, sender, who's notified.

States to show:
- blog.acme.io, connected, with "Update available: 1.3.0. Update it in WordPress › Plugins. It adds post translations as language versions and the form block's language setting."
- staging.acme.io, not heard from, after a failed Check now: "Timed out after 10 s at 14:02. Last heard from 13 Oct 22:10. If the site is behind a firewall or basic auth, ReputeCast can't reach it, but the plugin can still report in."
- a form rejected on staging.acme.io, with "Allow staging.acme.io" for members with forms:edit
- a new draft created from a post, and the notification Sofia Rossi gets
- no sites yet: "Install the ReputeCast plugin on your WordPress site, then connect it here."
- waiting for a site to report in
- read-only, loading, error

### 5.5 Settings › General: Languages and Accessibility
Two new cards on Settings › General, after Workspace and phase 1's Sidebar sections. The board shows the page scrolled to them; WorkspaceSettings gets the same cards.
- **Languages:**
  - Caption: "The languages your emails and public pages are written in. Your own app language is in Account & security."
  - A table: language (name, its own name, codes), contacts, page copy status, added by, and a row menu (Edit page copy, Remove).
  - English has Default and Fallback badges and its breakdown: "35,500 English · 5,508 with no language · 12 other languages (pt-BR, fr, it) · all get English".
  - Default language and Fallback language, as selects.
  - "How contacts get a language": signup forms (the browser), imports (a mapped column), the API and integrations, the preference center, and your team on the profile. "View contacts with no language" opens Contacts, filtered.
  - Add language (5.8).
- **Accessibility:**
  - "When serious issues remain": Warn (default) or Block, with one line on what each does.
  - What counts as serious, in one line.
  - Minimum text size: 12 px.
  - Who changed it, and when.

States to show:
- the sample
- changing the fallback to Spanish: "5,520 contacts would get Spanish where a Spanish version exists."
- switching the policy to Block: the impact preview, then saved
- Vietnamese page copy not set up, with "Write Vietnamese page copy", which opens Forms & pages
- read-only (Daniel Park): "Changing workspace languages needs workspace:manage. Ask Marcus Chen or Linh Nguyen."
- validation error: "English is the default language. Choose another default before removing it."
- saved, loading

### 5.6 Composer: languages
The composer with Weekly Digest #143, open on German.
- **Language switcher,** in the header after the status: "German · Outdated ▾". It lists every workspace language, with its status and recipients:
  - English · Source · 26,890
  - Spanish · Up to date · 2,740
  - German · Outdated, 2 blocks · 1,468
  - Vietnamese · Missing, gets English · 506

  Then: Compare languages, Add language, "Written in English" (the source; changeable while there's one version) and Translate with AI, disabled ("Not available yet").
- **Subject, preheader and the inbox preview** show the selected language.
- **Outdated banner** over the canvas: "German is outdated. 2 blocks changed in English since Sofia Rossi translated it on 14 Oct. Compare · Mark as up to date". The two blocks carry a "Changed in English" tag.
- **Editing the source:** after an edit, a toast: "Marked outdated in Spanish and German · Keep them up to date". Link and image edits offer "Apply to all languages".
- **Preview** lists contacts with their language ("Lucía Fernández · Spanish"). **Send test** sends one language, or one of each.
- **The status bar** names the language: "German · 71 KB of 102 KB · …".

States to show:
- German, outdated (the sample), with the switcher open
- Spanish, up to date
- Vietnamese, missing: the canvas says "No Vietnamese version. 506 Vietnamese readers get English.", with Add Vietnamese and Translate with AI, disabled
- Vietnamese, draft: "Vietnamese is a draft: 4 of 9 blocks translated. Its readers get English until it's marked translated." Blocks still in English carry a tag. Mark as translated.
- the toast after a source edit
- Daniel Park: he edits Spanish and adds a version; "Add a workspace language" says it needs workspace:manage and who can help
- Aisha Bello: read-only; the switcher and the compare view still work
- the composer's autosave, saved and conflict states, per language

### 5.7 Compare view
Opened from the switcher or the Outdated banner. It replaces the canvas; the header stays.
- **Top:** "Compare English (source) with German ▾", and the progress: "7 of 9 blocks up to date".
- **Rows,** aligned by block, with the subject and preheader first. English on the left, read-only. German on the right, edited in place.
- **Row status:** Up to date · Changed in English, with the English change as a diff (v6 → v9) · Still in English · New in English · Removed from English · Only in German.
- **Row actions:** Mark as up to date · Copy into German (new blocks) · Remove from German (removed blocks) · Show on canvas.
- **Filter:** "Only rows that need attention".
- **Footer:** Mark German as up to date, Done, and Translate with AI, disabled.
- **Tablet:** the two columns stack within each row.

States to show:
- German with its 2 changed blocks (the sample)
- a block new in English, and one removed from English
- everything up to date
- read-only (Aisha Bello)

### 5.8 Add language
- **Workspace dialog** (Settings › General):
  - search by name, own name or code
  - each language with the contacts who already have it ("Contacts with French: 3") and whether built-in page labels exist
  - right-to-left languages listed, disabled: "Right-to-left languages aren't supported yet. Contacts with Arabic get English."
  - after adding: "French added. French readers get French versions where an email has one, and English until then."
- **Email dialog** (composer):
  - the workspace languages without a version, with their recipients
  - start from a copy of the English version (layout and text), or the layout only
  - Translate with AI, disabled
  - creates a Draft
- **Remove a language** (workspace): lists what has a version in it, says its readers get the fallback, and that the versions come back if it's added again. Sample, removing German: "Weekly Digest #143, Welcome series v4 and password-reset have German versions. They're kept but no longer sent, and 2,240 German readers get English."

States to show:
- adding French to the workspace
- picking a right-to-left language
- adding Vietnamese to Weekly Digest #143
- removing German, with the impact
- a contact whose language isn't offered: Rafael Souza's profile, "Portuguese (pt-BR) · not one of your languages · gets English"
- page copy fallback: Forms & pages › Preference center with Vietnamese picked: "Vietnamese page copy isn't set up. Vietnamese visitors see English." The built-in Vietnamese labels are ready; "Copy English" starts the rest.

### 5.9 Account & security › Language & region
- **The new section,** after Profile:
  - App language: Instance default (English), English, Tiếng Việt.
  - Region format, Time, Week starts on.
  - A preview line: "Tue 14 Oct 2026 · 09:00 · 48,213 · 99.1% · $2.84".
  - "Times in Acme Newsletter use the workspace timezone, America/New_York (Settings › General). Your own timezone, in Profile, sets your quiet hours."
  - "Help translate ReputeCast".
- **Saving** shows the app in the new language at once, with a toast in that language and Undo.
- **The instance default** (Instance admin › Instance & updates › Instance): Default language and Default region format, with "Used for the sign-in page, members who haven't chosen, and invitations."

States to show:
- English, ReputeCast standard (the sample)
- Tiếng Việt picked: the preview reads "Th 3, 14 thg 10, 2026 · 09:00 · 48.213 · 99,1% · 2,84 US$"
- saved: the page now in Vietnamese
- the browser suggestion: "Your browser prefers Tiếng Việt. Use ReputeCast in Tiếng Việt?"
- the pseudo-locale option, in development
- the instance default, on InstanceSettings

### 5.10 Stress tests: Campaigns, Review & send, Team
Three dense screens drawn again in Vietnamese and in the pseudo-locale, to settle how text expands.
- **Screens:** Campaigns (the Campaigns board), Review & send for Weekly Digest #142 (SendReadiness), and Team (SettingsTeam).
- **Same data, same widths.** Only the interface changes language. Campaign, list and member names stay as they are.
- **Vietnamese:** real strings (the glossary in section 4), the Tiếng Việt (Việt Nam) region format, and stacked diacritics (ệ, ở, ữ).
- **Pseudo-locale:** every string about 40% longer, accented and in ⟦ ⟧, with the ReputeCast standard formats.
- **What to look for,** noted on each board:
  - cut-off text: a missing ⟧ shows it
  - labels that wrap into icons, badges or numbers
  - buttons that wrap, or push others off
  - table headers on more than two lines, and columns that squeeze numbers
  - the top-bar tabs, the sidebar and the Settings tabs
  - diacritics clipped by the line height, and uppercase overlines
- **Rules to confirm or change** (section 7):
  1. Never cut off: verdicts, status badges, numbers, buttons, permission keys, error messages, anything in a dialog. They wrap, or the layout makes room.
  2. Cut off with an ellipsis, with the full text on hover and focus: user content in table cells (campaign, list and member names, email addresses) and the middle of breadcrumbs.
  3. At most two lines: table headers, stat-tile labels, sidebar labels, card captions.
  4. Buttons don't wrap. A button group that doesn't fit moves to its own row, as the composer header's does.
  5. Badges don't wrap; the cell wraps around them.
  6. Tabs scroll sideways, as today.
  7. Body text keeps a line height of at least 1.4 in Vietnamese. Overlines stay uppercase.
- **Phone:** the three pseudo-locale boards also at 390 px.

### 5.11 Composer: Accessibility panel
The composer with Weekly Digest #143 in English, and the Accessibility tab open in the right panel.
- **Tab:** "Accessibility · 6", next to Style, Settings and Deliverability.
- **Summary:** "6 issues: 2 serious, 4 moderate · in English, Spanish and German". Under the default policy: "They're warnings in Review & send. Your workspace doesn't block on them."
- **Filters:** All · Serious · Moderate, and This language or All languages.
- **Issue rows:** severity badge, the sentence, the languages, the element's path (mjml › mj-body › mj-section · hero), Show on canvas, and the fix in place:
  - alt text: a field, Save, and Decorative image
  - contrast: the ratio, the suggested colors as swatches with their ratios, Use
  - link text: "Use 'Scheduled rollouts · Acme Docs'", or write your own. In Spanish and German, a field: the docs page is in English.
  - headings: "Change both to Heading 2"
  - size: "Set to 12 px"
- **On the canvas:** issues carry tags, like the inline warnings. Show on canvas selects the element and scrolls to it.
- **Fixed:** a fixed issue moves to "Fixed", with Undo.
- **Passing checks,** collapsed: `lang` and `dir` added (en, es, de; ltr), `role="presentation"` on 16 layout tables, 6 of 7 images with alt text, 13 of 15 links that say where they go.
- **Status bar:** "… · 6 accessibility issues (2 serious)".
- **Settings panel:** Text blocks have "Text role": Paragraph, Heading 1, Heading 2, Heading 3.

States to show:
- issues found (the sample)
- fix applied: "Changed 2 headings to Heading 2 · Undo". 5 issues left.
- all clear (Weekly Digest #142): "No issues. lang, dir and layout tables were set for you."
- HTML mode: fixes that edit MJML become "Show in code"
- the policy set to Block: serious issues read "Blocks sending"
- read-only (Aisha Bello): the list without fix controls
- checking, after an edit
- error: "Couldn't check the email. Your edits are saved."

### 5.12 Color picker with contrast
The composer with the "Plan your migration" button selected, the Style panel open, and its Background color picker open.
- **Contrast chip,** under the picker: "2.2:1 · Fails AA", against the button's text color, #FFFFFF.
  - AA and AAA, for normal and for large text, each marked Pass or Fail in words.
  - What it compared: "Button text #FFFFFF on this background".
- **Suggestions,** as swatches with their ratios. Clicking one applies it.
  - #0F766E · "Your other buttons" · 5.4:1
  - #348377 · "Nearest that passes AA" · 4.5:1
  - #276259 · "Passes AAA" · 7.0:1
- **While dragging,** the chip updates. It turns to "Passes AA" at #348377.
- **Picker closed:** the color field keeps a small chip: "2.2:1 · Fails".

States to show:
- failing (the sample)
- passing AA but not AAA
- passing AAA
- large text, where 3:1 is enough
- text over a background image: "Contrast can't be checked over an image. Check it by eye, or set a background color behind the image."

### 5.13 Review & send: Languages and Accessibility
Review & send for Weekly Digest #143, before scheduling.
- **Verdict:** Ready with warnings, with the counts.
- **Languages,** after Content:
  - **Warn** · "German is outdated: 2 blocks changed in English after translation" · Compare · Mark as up to date · Send English to German readers
  - **Pass** · recipients per language: English 26,890 (23,269 English, 3,613 with no language, 8 with other languages), Spanish 2,740, German 1,468, and Vietnamese 506, who get English because there's no Vietnamese version
  - **Pass** · "Vietnamese readers see the preference center and unsubscribe page in English. Vietnamese page copy isn't set up."
- **Accessibility,** after Languages:
  - **Warn** · "The hero image has no alt text · English, Spanish, German" · Write alt text (opens the composer) · Mark as decorative
  - **Warn** · "'Plan your migration' button: text contrast 2.2:1, below 4.5:1" · Use #0F766E
  - **Warn** · "4 moderate issues: 'Click here', a bare URL, a skipped heading level, 11 px footer text" · Fix 2 in one click · Open in composer
  - **Pass** · `lang` and `dir` set for every language
  - **Pass** · `role="presentation"` on 16 layout tables
- **Content:** no alt-text row any more. The size row names each language: "English 66 KB · Spanish 68 KB · German 71 KB, under 102 KB".
- **Confirmation step:** lists the languages ("English, Spanish and German; Vietnamese readers get English") and the warnings to confirm.

States to show:
- warnings (the sample)
- the policy set to Block: the verdict is Blocked, the two serious rows read Block, and a line says "Your workspace blocks sends with serious accessibility issues (Settings › General). Changed by Linh Nguyen at 14:22."
- "Fix 2 in one click": a preview of the heading and text-size changes, then the checks run again. "Click here" and the bare URL stay, to fix in each language.
- Vietnamese as a draft: a Warn row
- everything up to date and no issues (Weekly Digest #142)
- checks running, error

---

## 6. Changes to existing screens

- **Settings › Integrations:**
  - The current page becomes HostRepute's detail page in the hub, under Integrations › HostRepute.
  - The "Other integrations" table goes away; its entries are catalog cards.
  - Google Postmaster Tools and Microsoft SNDS move into HostRepute's card and detail page, as what HostRepute adds.
- **Connect HostRepute, HostRepute states:** unchanged. The catalog card is a new way in.
- **Keys & MCP:**
  - A second table, "Managed by integrations · 4": Zapier · Acme ops, n8n · Acme automations, WordPress · blog.acme.io and WordPress · staging.acme.io. Each has a "Managed by …" badge that opens its integration.
  - Their Rotate and Take over happen in Integrations. Revoke stays here, with a warning that names the workflows that stop.
  - The revoke dialog (Dialogs) carries that warning.
- **Webhooks:**
  - n8n's endpoint has a "Managed by n8n" badge. Its URL, events and secret can't be changed by hand: "To change it, change the workflow in n8n, or disconnect n8n." Its deliveries, retry and replay work as usual.
  - The event picker gets a Forms group with `form.submitted`.
- **Roles & permissions, role editor:** the Integrations group (3.9) in the editor and in the compare matrix.
- **Team › deactivate or remove a member:** lists the integrations they connected, with Take over (the default) or Disconnect.
- **Notifications:** two new kinds:
  - "Integration needs attention", for members with integrations:manage
  - "Post draft created", for the member the rule names
- **Audit log:** the actions of 3.10, and an "Integrations" filter category.
- **Settings › MCP server:** the new read tools and the language parameter in the agents' matrix.
- **Settings › General:** the Languages and Accessibility cards (5.5).
- **Composer, on every composer board:**
  - the language switcher in the header (5.6). On Weekly Digest #142 it reads "English · Source · 3 of 4 languages".
  - the Accessibility tab with its count. The alt-text warning moves there from Deliverability; on ComposerWarnings, the Deliverability count drops by one and Accessibility reads 1.
  - "Text role" in the Settings panel for Text blocks
  - the status bar's accessibility summary. On Weekly Digest #142: "Accessibility: all clear".
  - ComposerEmpty: the gallery shows each template's accessibility result (section 4).
  - ComposerPreview: "Preview as" shows each contact's language; Send test offers one language or one of each.
- **Review & send, approvals:**
  - the Languages and Accessibility groups. On Weekly Digest #142 they pass: "English, Spanish and German up to date; 506 Vietnamese readers get English", `lang` and `dir` set, layout tables marked, no issues.
  - "All 7 images have alt text" moves from Content to Accessibility. The flash sale's alt-text warning moves too (SendReadinessBlocked).
  - the check counts follow the rows, as inventory row 91 already does
  - the confirmation step lists the languages
  - Approval review: a language picker on the preview, and the two new groups in its readiness section
  - Approval states: an invalidation diff names the language ("German: intro changed after approval")
- **Report:**
  - "Results per language" after "Results per sending provider", only for campaigns sent in more than one language. The Report board's October Product Update went out in English only, so it doesn't change.
  - ReportStates shows the section for Weekly Digest #142 (section 4), and the click map's language picker.
- **Contacts:** a Language filter.
- **Contact profile:**
  - the language field as a select, with where it came from: "English (en) · Brevo import, 18 Sep 2026"
  - the activity names the version delivered: "Delivered Weekly Digest #142 · English · via Amazon SES"
  - a language that isn't offered says it gets the fallback
- **Segment builder:** "Language is …" shows language names.
- **Import › Map fields:** a language column is normalized, and unknown values are listed.
- **Forms & pages:**
  - Form builder: a language switcher above the form's copy; Form language (match the page, follow the browser, always one); the copy status per language; the consent text's version per language; the preview in the selected language.
  - Subscribe page, Preference center, Unsubscribe page and Archive tabs: the same switcher, the copy status, and the built-in labels, collapsed.
  - FormsStates: the "Vietnamese not set up" state.
- **Public pages:**
  - every page: the language menu in the footer, "English · Español · Deutsch"
  - preference center: "Email language", listing all four languages
  - archive: "Also in: Español · Deutsch" on issue pages
- **Automations:**
  - Flow editor: the Send email step's panel lists its languages and their statuses; Edit in composer opens the chosen language.
  - Publish dialog: Welcome series v4's diff adds "Spanish and German versions of steps 1 to 3". The checks add languages ("Step 2's German version is outdated") and accessibility ("Step 3 B: 1 serious issue, button contrast"). Under the Warn policy, neither stops publishing.
- **Transactional (phase 1's screens):**
  - template editor (P1 5.3, TransactionalTemplate): the language switcher, and Accessibility in the Checks tab
  - publish dialog (P1 5.4, TransactionalPublish): "Publishes English, Spanish and German together", with any Outdated version listed
  - message log (P1 5.5, TransactionalMessages): a Language column
  - message detail (P1 5.6, TransactionalMessage): "Language: Spanish, from the contact", or "Requested pt-BR, sent English"
  - setup (P1 5.7, TransactionalSetup): `language` in the API docs and code samples
  - the category unsubscribe page (PublicUnsubscribeCategory): the language menu, like every public page
- **Account & security:** the Language & region section (5.9), on desktop, tablet and phone.
- **App shell › user menu:** a "Language: English" item that opens Language & region.
- **Instance admin › Instance & updates:** Default language and Default region format in the Instance card.
- **First sign-in › instance basics:** asks for the default language.
- **Team › invite:** "Invitation language", defaulting to the instance default.
- **Sign-in and invitation pages:** a language menu at the bottom; pages follow the browser when the app has its language.
- **Design system:** the patterns of section 7 on DSExpansion; the new screens in the coverage matrix; the text expansion rules on the layout board.

---

## 7. New design-system patterns

Add these to the "9 Design system" page before the screens, on phase 1's expansion-patterns board (`DSExpansion`). Use the existing tokens and badge colors.

- **Integration catalog card:** the provider catalog's logo tile, name, maintainer, one-line description, capability badges, status badge, fact line and one action. Three cards a row at 1440 px, two at 834, one at 390.
- **"Managed by" badge:** the integration's logo and name in a neutral badge that opens its page. Rows it marks are read-only where the integration owns them.
- **Permission preset list:** triggers and actions with their permission and a switch; ⚑ on sensitive ones; locked rows under the escalation guard, with the reason.
- **Waiting for a first call:** the test-send timeline: waiting, received, nothing after 10 minutes.
- **Language chip with translation status:** the language's name, with its code on hover, and a status:
  - Source: info
  - Up to date: ok
  - Outdated: warn, with the count of blocks
  - Draft: neutral
  - Missing: neutral, with "→ English"

  A compact form for headers and phones: "DE · Outdated".
- **Side-by-side compare:** two columns aligned by block, a status per row, the source's change as an inline diff, and row actions. Rows stack at tablet width.
- **Contrast-ratio chip:** the ratio, rounded down to one decimal; AA and AAA for normal and large text, in words; the colors compared; suggested swatches with their ratios.
- **Issue list that locates an element:** severity badge, sentence, languages, element path, Show on canvas, the fix in place, and a Fixed section with Undo. Shared by the Accessibility panel and Send Readiness.
- **Public language menu:** plain links, each in its language's own name, in the publisher's branding. No JavaScript.
- **Text expansion rules,** from 5.10, on the layout board.

---

## 8. Conflicts to note

1. **A fourth translation status.** The plan names three: Up to date, Outdated, Missing. This brief adds Draft, so a half-translated email is never sent. The owner needs to confirm.
2. **Missing versions pass.** In Send Readiness, a Missing version is a Pass that counts who gets the fallback; only Outdated and Draft versions warn. The plan doesn't say which. The owner needs to confirm.
3. **Alt text moves** from Content to Accessibility, in the composer and in Send Readiness. The check counts change on SendReadiness, SendReadinessBlocked and ApprovalReview; the app already counts the rows it shows (inventory row 91).
4. **Integration keys are narrower than API keys.** The team add-on (3.9) lets a key call anything its permissions allow. An integration key is limited to its integration's operations, and its webhooks:manage to the endpoints it created.
5. **Rotation overlap.** A rotated API key stops at once. A rotated integration key keeps working for 24 hours, so runs don't fail while someone pastes the new one.
6. **Take over.** Since hardening, a key loses its permissions when its creator leaves. Take over gives an integration key a new owner without a new secret, and deactivating a member offers it.
7. **A view permission.** The plan lists only integrations:manage. Every endpoint needs a permission, so the hub's read side uses integrations:view, given to every built-in role and to Content Writer.
8. **Where Block applies.** The plan's workspace setting is for Send Readiness. This brief also applies it to publishing automations and transactional templates.
9. **Event names.** The plan's "campaign sent" trigger is `campaign.completed`. "Form submitted" needs a new event, `form.submitted`, which fires on confirmation, like the automation trigger.
10. **A seventh transactional template.** Phase 1's sample has six; this brief adds payment-failed, published at 11:30 on 15 Oct, after phase 1's samples end.
11. **Two keys called Zapier.** TeamDialogs and RoleEditorStates show "Zapier · forms", a plain key of Sofia Rossi's. It stays a plain key and isn't the integration's key.
12. **blog.acme.io** is also Acme Engineering Blog's sending domain (posts@blog.acme.io). Here the WordPress site connects to Acme Newsletter; a site connects to one workspace.
13. **Headings in existing emails.** The Hero and Article card blocks now output Heading 1 and Heading 2. The demo emails look the same, but their HTML changes.
14. **Dates.** Today the app writes dates in one fixed format (`web/src/lib/format.ts`). "ReputeCast standard" keeps it as the default region format.
15. **New drafts.** Weekly Digest #143 and the Product Updates draft from blog.acme.io are new on 15 Oct. The Campaigns board's counts were drawn before them and stay as they are.
16. **Weekly Digest #142** went out on Tue 14 Oct in this phase's sample. Boards that show it scheduled (Campaigns, SendReadinessStates) keep their earlier moment.
17. **Vietnamese strings** are the first non-English copy on the canvas. They need a native speaker's review before the stress boards are final.

---

## 9. Boards

New boards for the canvas page "P2 Integrations, languages & accessibility". Desktop boards are 1440 px wide. Tablet (834 px) and phone (390 px) boards add `Tablet` or `Phone` to the file name, for example `IntegrationsCatalogPhone.dc.html`.

| File | Title | Implements | Tablet | Phone |
|---|---|---|---|---|
| `IntegrationsCatalog.dc.html` | Integrations · catalog | 5.1 | yes | yes |
| `IntegrationsZapier.dc.html` | Integrations · Zapier, connected with activity | 5.2 | yes | yes |
| `IntegrationsConnect.dc.html` | Integrations · connect Make, escalation guard, unreachable, private address, no call yet, disconnect | 5.3 | — | — |
| `IntegrationsWordPress.dc.html` | Integrations · WordPress sites, update available, not heard from | 5.4 | — | — |
| `IntegrationsStates.dc.html` | Integrations · states of the catalog, the detail page and WordPress sites | 5.1, 5.2 and 5.4 states | — | — |
| `SettingsLanguages.dc.html` | Settings · General · languages and accessibility | 5.5 | yes | yes |
| `ComposerLanguages.dc.html` | Composer · languages, German outdated | 5.6 | yes | yes |
| `ComposerCompare.dc.html` | Composer · compare English and German | 5.7 | — | — |
| `LanguageStates.dc.html` | Languages · states of General, the composer, compare, and Add language | 5.5 to 5.7 states, 5.8 | — | — |
| `LanguageRegion.dc.html` | Account · language & region | 5.9 | — | yes |
| `StressCampaignsVi.dc.html` | Stress test · Campaigns · Vietnamese | 5.10 | — | — |
| `StressCampaignsPseudo.dc.html` | Stress test · Campaigns · pseudo-locale +40% | 5.10 | — | yes |
| `StressReviewVi.dc.html` | Stress test · Review & send · Vietnamese | 5.10 | — | — |
| `StressReviewPseudo.dc.html` | Stress test · Review & send · pseudo-locale +40% | 5.10 | — | yes |
| `StressTeamVi.dc.html` | Stress test · Team · Vietnamese | 5.10 | — | — |
| `StressTeamPseudo.dc.html` | Stress test · Team · pseudo-locale +40% | 5.10 | — | yes |
| `ComposerAccessibility.dc.html` | Composer · accessibility panel, issues found | 5.11 | yes | yes |
| `ComposerContrast.dc.html` | Composer · color picker with contrast | 5.12 | — | — |
| `SendReadinessAccessibility.dc.html` | Review & send · languages and accessibility (Weekly Digest #143) | 5.13 | — | yes |
| `AccessibilityStates.dc.html` | Accessibility · states of the panel, the color picker and Review & send, and the gallery's results | 5.11 to 5.13 states | — | — |

20 desktop boards: 14 new screens and states, and 6 stress renders of existing screens. Plus 5 tablet and 10 phone boards.

Each "States" board holds every state its screens list in section 5 that the screen's own board doesn't show. LanguageRegion shows its states on its own board, below the section.

**Existing boards that section 6 changes:**
- Integrations, keys and webhooks: `SettingsIntegrations`, `SettingsKeys`, `SettingsKeysTablet`, `SettingsKeysPhone`, `Dialogs`, `SettingsAPI`, `SettingsAPIPhone`, `SettingsMCP`, `SettingsAudit`
- Team and roles: `Roles`, `RolesTablet`, `RolesPhone`, `RoleEditor`, `TeamDialogs`, `TeamInvite`
- Settings › General: `WorkspaceSettings`, `WorkspaceSettingsPhone`
- Notifications: `Notifications`, `NotificationsTablet`, `NotificationsPhone`
- Composer: `Composer`, `ComposerTablet`, `ComposerPhone`, `ComposerMobile`, `ComposerCode`, `ComposerWarnings`, `ComposerEmpty`, `ComposerPreview`, `ComposerStates`, `ComposerRequestApproval`
- Review & send and approvals: `SendReadiness`, `SendReadinessTablet`, `SendReadinessPhone`, `SendReadinessBlocked`, `SendReadinessBlockedPhone`, `SendReadinessConfirm`, `SendReadinessStates`, `SendReadinessApproval`, `ApprovalReview`, `ApprovalReviewTablet`, `ApprovalReviewPhone`, `ApprovalStates`
- Reports: `ReportStates`
- Audience and import: `Contacts`, `ContactsTablet`, `ContactsPhone`, `ContactProfile`, `ContactProfilePhone`, `SegmentBuilder`, `MigrationFields`
- Forms and public pages: `FormBuilder`, `FormBuilderTablet`, `FormBuilderPhone`, `FormsStates`, `PublicSubscribe`, `PublicSubscribePhone`, `PublicConfirm`, `PublicPreferences`, `PublicPreferencesPhone`, `PublicUnsubscribe`, `PublicArchive`
- Automations: `AutomationFlow`, `AutomationFlowTablet`, `AutomationFlowPhone`, `AutomationPublish`
- Account, shell, instance and sign-in: `AccountSecurity`, `AccountSecurityTablet`, `AccountSecurityPhone`, `ShellUserMenu`, `InstanceSettings`, `InstanceSetup`, `SignInStates`, `AuthScreens`, `AuthScreensPhone`, `InviteStates`
- Design system: `DSCoverage`, `DSLayout`
- Phase 1's boards: `TransactionalTemplate`, `TransactionalTemplateStates`, `TransactionalPublish`, `TransactionalPublishStates`, `TransactionalMessages`, `TransactionalMessage`, `TransactionalSetup`, `PublicUnsubscribeCategory`, `DSExpansion`
