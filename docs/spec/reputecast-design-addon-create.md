# ReputeCast — Add-on spec: create (digests, conditional content, polls and surveys, AI assistant)

Phase 3 of `reputecast-design-expansion-plan.md`. It builds on phase 1 (`reputecast-design-addon-transactional.md`) and on phase 2 as the plan scopes it: the Integrations hub, content languages and accessibility checks.

## 0. How to use this document

Read all of it before you start.

- **The prototype from the main spec, the team add-on and phases 1–2 already exists.**
  - Add the screens in section 5 as a new canvas page, "P3 Create".
  - Apply section 6 to the existing screens.
  - Reuse the existing navigation, components and design system.
  - Add the patterns in section 7 to the "9 Design system" page first.
  - Phase 2's screens are assumed: the Integrations hub (catalog and detail page), the composer's language switcher and its Accessibility panel. If they aren't drawn yet, draw the parts this brief touches in their phase 2 form first, as the plan describes them.
- **Same rules as before:**
  - Empty, loading, error and success states, plus the states listed for each screen.
  - All copy in English.
  - Use the sample data in section 4, together with the main spec's, the team add-on's and phase 1's.
  - Desktop boards at 1440 px. Public pages at 720 px. Tablet (834) boards for 5.2, 5.4 and 5.11; phone (390) boards for 5.4, 5.7 and 5.11.
- **Build order:**
  1. 5.4 Digest overview and issues
  2. 5.1 Type picker and 5.2 Digest setup
  3. 5.3 Composer: digest mode
  4. 5.5 Composer: Visibility and versions
  5. 5.6 Poll block, 5.7 Public answer and survey pages, 5.8 Poll results
  6. 5.11 Composer: AI suggestions
  7. 5.10 AI assistant settings
  8. The rest, in order
  9. Then section 6

  If the work needs several passes, keep going until every screen exists.

---

## 1. Principles

1. **Everything is free.** No limits on digests, issues, conditional blocks, polls, surveys or AI features. The only AI limit is the monthly cap the workspace sets on its own model account.
2. **One editor, one public theme.**
   - Digests, conditional content, poll blocks and AI suggestions all live in the GrapesJS + MJML composer. The Feed items and Poll blocks are custom GrapesJS blocks.
   - The answer page and the survey page use the publisher branding from Forms & pages.
3. **Automatic, never unattended.**
   - A digest sends on its own only after someone with campaigns:send turns it on.
   - Every issue goes through Send Readiness and the approval rules, like a campaign written by hand.
   - A held issue waits for a person. Nothing is "sent anyway".
4. **Honest answers.**
   - Poll answers come from people. The answer page records the answer, not the click, so link scanners can't vote.
   - The report says how many automated visits it ignored, like machine opens.
5. **The workspace's own model, and nothing else.**
   - AI is off until someone connects a model with the workspace's own key.
   - ReputeCast sends the content being edited, never contact data, and only when a person asks.
   - Every suggestion is a diff to accept or reject. Nothing changes on its own.
6. **Stay on ReputeCast's side of the HostRepute line.** The assistant writes and edits; it doesn't diagnose sender reputation. Reputation questions link to HostRepute's assistant when HostRepute is connected.
7. **Same access rules as the team add-on.**
   - Permission keys, the escalation guard, write-only secrets and audit entries.
   - MCP agents propose. They never send, never turn on a digest and never use the workspace's model.

---

## 2. Terms

- **Recurring digest:** a campaign that builds an issue on a schedule from a feed's new items and sends it.
- **Feed:** an RSS 2.0, Atom 1.0 or JSON Feed 1.1 address, or a WordPress site connected in the Integrations hub (phase 2).
- **Item:** one entry in the feed, usually a post.
- **New item:** an item that no sent issue has included yet. Skipped issues don't use up items.
- **Issue:** one email built from a digest. It is its own campaign, with a number, content, recipients, a Send Readiness run, an approval when one is needed, and a report.
- **Preparation:** the moment an issue is built from the feed and checked, 3 hours before its send time by default.
- **Digest owner:** the person who turned the digest on. Issues send on their authority. If they lose campaigns:send, a digest that sends automatically pauses.
- **Sending mode:**
  - **Send automatically:** each issue goes out at its time once its checks and approvals pass.
  - **Create a draft:** each issue becomes a draft campaign; a person sends it through Review & send.
- **Held issue:** an issue whose Send Readiness verdict is Blocked. It waits until someone fixes it or skips it.
- **Late limit:** how long an issue may wait (held, waiting for approval, feed down) before it's skipped. 24 hours by default.
- **Conditional block:** a section or block with a Visibility rule.
- **Version:** one combination of conditional blocks that at least one recipient will get. "3 conditional blocks, 6 versions".
- **Poll block:** a question in an email with two to five answers, a 0–10 scale or an emoji scale. Each answer is a link.
- **Answer page:** the branded page a poll link opens. It records the answer; the click alone records nothing.
- **Automated visit:** a visit to an answer page that ReputeCast attributes to a link scanner, a prefetcher or a proxy. It is never counted as an answer.
- **Survey:** a hosted page with several questions, linked from an email.
- **AI assistant:** the features that send content to the workspace's own model and show the result as a suggestion.
- **Suggestion:** the model's output, shown as a diff to accept, edit or reject.
- **AI activity log:** a record of each AI request: who, when, which feature, on what, the model, tokens and the outcome. It never stores the content.
- **Usage cap:** what the workspace allows its model to cost in a month, estimated from the token counts the provider reports and the prices entered.

---

## 3. Requirements

### 3.1 Recurring and RSS digests
- **Campaign type:** "New campaign" asks for the type first: Regular or Recurring digest. A/B testing stays inside Regular.
- **Feed source:**
  - A feed address: RSS 2.0, Atom 1.0 or JSON Feed 1.1. A page address works too: ReputeCast finds the feed the page links to (blog.acme.io finds blog.acme.io/feed/).
  - Or a WordPress site connected in the Integrations hub. Pick categories and tags. The plugin tells ReputeCast when a post is published, so the preview updates at once.
  - Filters: only items in some categories, and "Leave out items older than 14 days".
  - **Starting point:** "Only items published after the digest is turned on" (default), or "Include the latest N items in the first issue". A feed's back catalogue never floods the first issue by accident.
- **Reading the feed:**
  - Every 30 minutes while the digest is on, and again at preparation.
  - Through the same SSRF guard as importers and webhooks. Up to 3 redirects, each target checked again. A permanent redirect offers "Use the new address".
  - Conditional requests (ETag, Last-Modified), 15-second timeout, 5 MB limit.
  - Items are matched by their ID (`guid`, `id`), then their link, so an edited post isn't sent twice.
- **Schedule:**
  - Daily (every day or weekdays), weekly on a day, or monthly (a day of the month, or the first Monday), at a time in a time zone. The workspace time zone is the default.
  - Or "When N new items arrive", at most once a day, inside a sending window such as 08:00–18:00. Items that arrive at night wait for the window.
  - No new items: the issue is skipped. No email, and the history says why.
  - **Preparation:** 3 hours before the send time by default (1 to 24 hours), so checks and approvals have time. The item-count schedule prepares as soon as the Nth item arrives.
  - **Late limit:** 6, 24 (default) or 48 hours. An issue that can't go out within it is skipped, with the reason.
- **Feed items block** (composer, digest mode):
  - Layouts: List (title and excerpt), Cards (image above the text, one per row) and Grid (two per row, stacked on mobile).
  - Fields to show: image, title (linked), date, author, categories, excerpt (number of words) and a "Read more" button with its text.
  - Maximum items: 1 to 20, default 6. Order: newest or oldest first.
  - More new items than fit: send the newest and leave out the rest (default), or keep the rest for the next issue.
  - Images are copied into the media library and resized to the block's width, so they load from the workspace's domain, not the blog's. Alt text comes from the image's own alt text, else the item's title, marked "from the title" in the checks.
  - Excerpts are plain text. HTML, scripts and styles in the feed are stripped.
  - Item links are tracked like every other link.
- **Feed tokens** for the subject, the preheader and Text blocks: `{{ feed.title }}`, `{{ issue.number }}`, `{{ issue.date }}`, `{{ issue.items.size }}`, `{{ issue.items.first.title }}`, and `issue.items` for loops, each item with `title`, `url`, `date`, `author`, `image`, `excerpt` and `categories`.
  - The subject preview shows the subject with 1, 3 and 6 items, so "…and 0 more" is caught before anyone sees it.
- **Audience, sender, route and auto-pause thresholds:** as for a campaign. The audience is counted again for each issue.
- **Sending mode:** Send automatically, or Create a draft.
  - In draft mode each issue becomes a draft campaign. The owner and the people with campaigns:send are told. Nothing goes out until someone uses Review & send. A draft not sent within the late limit is marked "Not sent".
- **Issues:**
  - Each issue is a campaign named "{digest} #{number}". It appears in Campaigns and has its own report, compared with the previous issue.
  - Numbers count sent issues only. Skipped issues have no number.
  - "Show issues in the public archive" is on by default, with the digest as its own filter.
- **Checks and approvals on every issue:**
  - Send Readiness runs at preparation and again when the send starts, as for any scheduled campaign.
  - **Blocked:** the issue is held. The owner is told, and so are the approvers when a request is open. One-click fixes work as usual. Then "Release issue" (needs campaigns:send) or "Skip this issue".
  - **Ready with warnings:** the issue goes out, unless the workspace's approval rule "Send Readiness is Ready with warnings" is on.
  - **Approval rules** are checked for each issue. When one matches, the request is created at preparation in the owner's name, with a "Recurring digest" badge. It shows the items in this issue and what changed since the last approved issue: template, audience, route and schedule.
  - Not approved by the send time: the issue waits. Not approved within the late limit: skipped, "Not approved in time".
  - The setup screen says up front which rules every issue will match.
- **Turning it on:**
  - Needs campaigns:send. A draft-mode digest only needs campaigns:edit, because it never sends by itself.
  - A confirmation states the first send time, the estimated recipients, the checks and approvals, and that the digest sends under this person's authority.
  - If the owner loses campaigns:send, is deactivated or leaves the workspace, the digest pauses. People with campaigns:send are told. Whoever turns it back on becomes the owner.
- **Pausing:** needs campaigns:pause. No new issues are prepared. Resuming needs campaigns:send. An issue that is already sending pauses like any send.
- **Editing a digest that is on:** template changes apply from the next preparation. A prepared issue keeps its content, with "Rebuild with the current template".
- **Feed failures:**
  - Unreachable or invalid at preparation: retried every 15 minutes until the send time, then skipped, "Feed unreachable".
  - Three issues in a row skipped because of the feed: the digest pauses and tells its owner.
- **Issue statuses:** Collecting, Prepared, Waiting for approval, Held, Sending, Sent, Skipped (with the reason: no new items, feed unreachable, not approved in time, skipped by a person) and Not sent (draft mode).

### 3.2 Conditional content
- **Visibility:** a section in the inspector's Settings tab for any section or block, in campaigns, digests and automation emails.
  - "Show to everyone" (default), "Only contacts who match" or "Everyone except contacts who match".
  - Conditions: in a segment, on a list, has a tag, a field (with the segment builder's operators), language (phase 2), engagement tier. Match ALL or ANY.
  - Score (phase 4) and paid tier (phase 6) are added later. Leave room for them in the condition picker.
  - **Not offered:**
    - On the footer block. Every version keeps the unsubscribe link and the address, so the picker says "The footer is always shown".
    - In transactional templates, whose recipients may not be contacts. Their editor keeps Liquid `{% if %}` on variables.
- **On the canvas:**
  - A visibility badge on each conditional block's label, with the condition in words: "Beta Program list".
  - An eye-off icon on the block's row in Layers.
  - A toggle above the canvas: "Show all blocks" or "Show as a contact sees it".
- **Preview as:** the existing contact picker (ComposerPreview) shows that contact's version.
  - "Show why" collapses each hidden block to a thin line: "Hidden: Beta corner · Sam Rivera isn't on Beta Program".
  - The preview names the version and how many recipients get it.
- **Versions panel:** "3 conditional blocks · 6 versions".
  - Versions are the combinations that occur in the current audience, not every possible one.
  - Each version: the blocks it shows, recipients, size, and "Preview this version".
  - Counted with the audience's live count, inside the same 10-second limit, with "Counting…" while it runs.
- **Rendering:** per recipient when the send runs. Segment, list, tag, field and tier are read once, when the send's audience is built, so a contact gets one consistent version.
- **Send Readiness, "Conditional content" group:**
  - Every version is rendered and checked.
  - A version with nothing between the header and the footer: Block, with its recipients.
  - A version over Gmail's clipping size: same severity as the existing size check, naming the version.
  - A condition that refers to a deleted segment, field, list or tag: Block, with the fix.
  - Links are checked across the union of all versions.
  - More than 50 versions: the 50 with the most recipients are checked, plus the largest and the smallest email. The check says so.
- **Deleting a segment, list, field or tag** that a draft's Visibility rule uses warns first and names the email and the block.

### 3.3 Polls and surveys
- **Poll block:** a ReputeCast block in campaigns, digests and automation emails.
  - Kinds: Choice (two to five answers, text with an optional emoji), 0–10 scale (with the NPS wording as a preset), and emoji scale (three or five faces).
  - Layout: stacked buttons or one row. The 0–10 row wraps to two rows on mobile, 6 and 5 cells.
  - **Email-safe:** each answer is a plain link drawn as an MJML button. It works in every client, in dark mode, and in the plain-text version as a list of links.
  - Settings:
    - the question, and the answers
    - save the answer to a contact field (number for scales, text for choices) or as a tag per answer, such as `wants:postgres`
    - close on a date, after N days, or by hand
    - after answering: show the thank-you message, or continue to a survey (the answer fills its first question)
    - answers from people who aren't contacts: count them separately (default) or don't accept them
    - "Show results after answering": off by default
  - The poll and its links are created when the email is saved, so a test send already works. Test-send answers are marked Test and never counted.
- **Answer page:** on the publisher's domain, with the publisher's branding.
  - The link carries a signed token for the recipient, from the same family as preference-center links. Opening the page records nothing.
  - **With JavaScript:** the page shows the chosen answer as selected, then records it once the page is visible and a moment has passed, with a POST that carries the token. "Thanks, Sam. Your answer is saved."
  - **Without JavaScript:** the chosen answer is preselected with a "Confirm my answer" button, a form POST.
  - **Automated visits:** most link scanners don't run scripts, so they never record anything. Those that do are caught by the tracking module's classifier (agent, network, timing, and every answer of one email opened within a second) and recorded as automated visits.
  - **Change the answer:** the other answers stay on the page. Picking another replaces it: "Changed from 7 to 9". The latest answer counts; the history is kept.
  - **Comment:** an optional box, "Anything to add?", up to 500 characters, saved with the answer.
  - **Forwarded email:** a forwarded link still answers as the original recipient. The latest answer counts. The help text says so plainly; every email poll has this trade-off.
  - **Not a contact:** links in the web archive and "View in browser" carry no token. Those answers are counted apart as "Not a contact", or refused when the poll doesn't accept them.
  - **Closed poll:** the page says when it closed and records nothing. It shows the results when "Show results after answering" is on.
- **What an answer does:**
  - It is saved with the time, the campaign, the answer and the comment.
  - It sets the field or adds the tag. A changed answer updates the field, or swaps the tag.
  - It starts the automation trigger "Answered a poll or survey", which filters by poll and answer. A changed answer starts it again only when the trigger says so.
- **Report, Poll section:**
  - results by answer, as counts and shares; for a 0–10 scale, the NPS (promoters minus detractors) and the 11 bars
  - by segment: engagement tier, list or language
  - over time: the first 72 hours, like engagement
  - comments, with the contact's name for people with subscribers:view
  - automated visits ignored, answers from non-contacts, answers changed
  - Export: "Results (CSV)", aggregate only, audited like other report exports; "Answers with contacts (CSV)", which needs subscribers:export ⚑ and is audited
- **Survey:** a hosted page with several questions.
  - Created in Forms & pages › Forms: "New form…" asks "Signup form or Survey".
  - Questions, up to 20 on one page: single choice, multiple choice, 0–10 scale, emoji scale, short text, long text. Each can be required, have a description, and save to a field or tags.
  - Settings: open until a date or until closed; one response per contact, editable until it closes; accept non-contacts (anonymous) or not; the thank-you text.
  - Linked from an email in two ways: the link picker offers "Survey: …", which adds the contact's token, or a Poll block that continues to the survey.
  - The page needs no script: the answers are recorded when the person presses Submit, which scanners don't do.
  - "Answered a poll or survey" also fires for surveys.
- **Survey results:** per-question charts, completion (started and completed), responses with the contact for people with subscribers:view, and the same two exports.
- **Privacy:** answers and comments are contact data. They are erased with the contact (subscribers:delete) and included in the contact's data export.

### 3.4 AI assistant, with the workspace's own model
- **Settings › Integrations › AI assistant** (the hub's "AI models" group):
  - **Catalog:** Anthropic, OpenAI, Google (Gemini), Azure OpenAI, and OpenAI-compatible servers such as Ollama, vLLM or LM Studio.
  - Capability badges: text, image input, structured output, runs on your own network (a server on a private address), data region (Azure).
  - **Connection:** provider; base URL (Azure endpoint and deployment, or the server's address); API key (write-only, optional for local servers); model, picked from the provider's list or typed. One connection per workspace.
  - **Test connection** sends a fixed prompt with no workspace content and shows the answer and the time it took.
  - A server on a private address works when the instance allows private addresses, the same setting SMTP servers and community adapters use.
  - **Monthly usage cap:** an amount in US dollars, estimated from the token counts the provider reports and the prices per million tokens. Prices are pre-filled for known models from a list dated with ReputeCast's release, and editable. A notice at 80%. A local model can use a token cap instead, or none.
  - **Features:** a switch for each one, all on when connected.
  - **What is sent:** a plain statement per feature (below), and the line "Never sent: contacts, addresses, names, field values, recipient lists, keys or credentials."
  - **Usage:** this month by feature and by member.
  - **AI activity log:** 90 days, without content.
  - **Disconnect:** removes the key. Every feature shows the not-configured state; the log stays.
- **What each feature sends:**
  - **Subject ideas:** the subject, the preheader, the email's text without images, its language, and the subject rules ReputeCast checks.
  - **Rewrite, shorten, change the tone:** the selected text and an optional instruction. Tones: friendlier, more formal, more direct.
  - **Translate a language variant (phase 2):** the source variant's text, block by block, its subject and preheader, and the target language.
  - **Alt text:** the image, resized to 1,024 px, and the text around it.
  - **Fix an accessibility issue (phase 2):** link text that says nothing, or missing alt text, with the text around it. Heading levels and text size are fixed without AI, as in phase 2.
  - **Describe the segment:** the description typed, and the workspace's condition schema: field names and types, list, segment and tag names, tiers. No values from any contact.
  - **Report summary:** the report's aggregate numbers. No addresses.
  - Liquid tokens are sent as written, such as `{{ first_name | default: "there" }}`, never filled in. A preview "as a contact" is never sent.
- **Suggestions:**
  - Always a diff card: Accept, Edit then accept, Reject, Try again. Nothing is applied on its own.
  - Checks run on the suggestion: the subject heuristics for subjects; for translations, every Liquid token and link of the source must still be there. A token or link the model changed is restored from the source, and the card says so.
  - An accepted suggestion goes into the version history: "Subject from an AI suggestion, accepted by Daniel Park".
  - A translated variant stays marked "Translated by AI · not reviewed" until someone marks it reviewed or edits it.
- **Segments:** "Describe the segment" drafts conditions with the builder's own conditions only. The draft appears as a diff on the builder, with the live count; the model's note says what it couldn't express. The SQL editor is never drafted.
- **Reports:** "Summarize results" writes a plain-language summary from the report's numbers. It is a suggestion card: "Keep" saves it on the report, labelled "Summary by AI, kept by Linh Nguyen"; "Discard" drops it. It never appears on read-only report links.
- **Limits of the assistant:**
  - It can't send, schedule, publish, approve, or change audiences, routes or settings.
  - It doesn't answer questions about sender reputation, blocklists or why mail lands in spam. A rewrite instruction that asks one gets: "Sender reputation isn't something this assistant looks at", with "Ask HostRepute's assistant" when HostRepute is connected, or a link to Deliverability when it isn't.
  - Report summaries describe the numbers. They don't diagnose causes.
- **Errors are shown plainly and change nothing:** the provider's error or rate limit, a refusal in the model's own words, the cap reached.

### 3.5 Permissions
One new permission, in a new group "AI":
- use the AI assistant (ai:use)

It isn't sensitive: what leaves the instance, and the monthly cap, are set by whoever holds integrations:manage ⚑ (phase 2).

Everything else uses existing keys:

| Action | Permission |
|---|---|
| Create or edit a digest, its template and its settings | campaigns:edit |
| Turn on a digest in "Send automatically" mode, resume it, release a held issue | campaigns:send ⚑ |
| Turn on a digest in "Create a draft" mode | campaigns:edit |
| Pause a digest, skip an issue | campaigns:pause |
| Approve an issue | campaigns:approve ⚑ |
| Visibility rules and Poll blocks | the email's own edit permission: campaigns:edit, templates:edit or automations:edit |
| Close or reopen a poll | campaigns:edit |
| Create, edit, publish or close a survey | forms:edit |
| See poll and survey results and comments | reports:view |
| See who answered | subscribers:view |
| Export answers with contacts | subscribers:export ⚑ |
| See the AI assistant's settings, usage and activity log | integrations:view (phase 2) |
| Use AI features | ai:use |
| Connect, change or disconnect the AI model, set the cap and the features | integrations:manage ⚑ |

Who gets ai:use:
- **Built-in roles:** Owner, Admin and Editor. Not Viewer.
- **Content Writer**, the sample's custom role: yes.
- **API keys and MCP agents:** can't hold it. The key dialog explains: "AI features are for people. Agents bring their own model."

**MCP agents:** can list digests and issues and read poll and survey results. They can't turn a digest on, release or skip an issue, or use the AI assistant.

### 3.6 Audit
- **Audited:**
  - a digest created, turned on, paused, resumed, deleted; its settings changed (feed, schedule, audience, mode, late limit), with before and after
  - a digest paused because its owner lost campaigns:send, and its new owner
  - an issue released, skipped or rebuilt by a person
  - a poll closed or reopened by a person
  - a survey created, published, closed or deleted, and its settings changed
  - poll or survey answers exported, either kind
  - an answer deleted
  - the AI connection created, changed or removed; its key replaced; the model, the cap or a feature switch changed
- **Issue sends** are audited as campaign sends are today (campaign.started, campaign.completed), with the actor "System · Engineering blog roundup, turned on by Sofia Rossi".
- **Not in the audit log:**
  - Each AI request. The AI activity log is its record, for 90 days, without content. See section 8.
  - Content edits, including Visibility rules, Poll blocks and accepted suggestions. They are in the version history, like every content edit today.

### 3.7 Notifications
New kinds, in the notification center and by email, following each member's preferences:
- **Digest issue held** (Sending): the owner, plus the approvers when a request is open.
- **Digest paused** (Sending): the owner, and everyone with campaigns:send when it paused because the owner lost the permission.
- **Digest feed failing** (Sending): the owner, at the first failed preparation, then when the issue is skipped.
- **Digest draft ready** (Sending): draft mode only, the owner and members with campaigns:send.
- **AI usage at 80% and cap reached:** members with integrations:manage.
- **AI key rejected by the provider:** members with integrations:manage.

Waiting-for-approval issues use the existing "Approval requested" kind.

### 3.8 HostRepute
- A reputation question in the assistant links to HostRepute's AI Reputation Assistant when HostRepute is connected. When it isn't, the link goes to Deliverability, which keeps today's invitation. Nothing more.
- Nothing from this phase is sent to HostRepute: no content, prompts, feed items, poll answers or survey responses.

### 3.9 Out of scope for this phase
- **Digests:** several feeds in one digest, full-text items, feeds that need a sign-in, choosing items per recipient (use Visibility on blocks instead).
- **Conditional content:** score (phase 4), paid tier (phase 6), conditions in transactional templates.
- **Polls and surveys:** quizzes with right answers, image answers, multi-page surveys, branching logic, file uploads.
- **AI:** reputation diagnosis, sending or scheduling, writing a whole campaign from a prompt, image generation, a chat window, contact data in prompts.

---

## 4. Sample data

### Today
- **Wednesday 15 Oct 2026**, 10:30 America/New_York. Workspace "Acme Newsletter", instance reputecast.acme.io.
- **Content languages** (phase 2): English 41,020 · Spanish 4,180 · German 2,240 · Vietnamese 773 contacts.

### The blog and its feed
- **blog.acme.io:** Acme's engineering blog, on WordPress. Connected in the Integrations hub (phase 2) by Marcus Chen on 25 Sep 2026. Its Announcements posts already become Product Updates drafts through phase 2's post-draft rule.
- **Feed:** blog.acme.io/feed/, reported by the plugin. RSS 2.0, 10 items. Categories: Engineering, Infrastructure, Announcements, Company. Last read 10:05, when the plugin reported a new post.

### Recurring digest: "Engineering blog roundup"
- **History:** Daniel Park built it on Thu 25 Sep, in draft mode, as soon as the blog was connected. Sofia Rossi switched it to "Send automatically" and turned it on that afternoon. Owner: Sofia Rossi.
- **Feed:** the WordPress site blog.acme.io, categories Engineering and Infrastructure. Announcements are left out, because they already reach Product Updates as drafts. Starting point: items published after it was turned on.
- **Schedule:** weekly, Fridays at 08:00 America/New_York. Prepared at 05:00. Late limit 24 hours.
- **Audience:** the Product Updates list, 14,110 contacts. 13,944 recipients after excluding 148 suppressed, 11 role addresses and 7 duplicates.
- **From:** Acme Product <product@news.acme.io>. **Route:** Broadcasts, SES 90% / Postal 10%.
- **Approvals:** every issue matches "audience ≥ 10,000 recipients", and, until Postal's warm-up ends on Fri 24 Oct, "route includes a Warming-up provider". Marcus Chen or Linh Nguyen approves.
- **Template** (the "Blog digest" starter):
  - header with the logo
  - intro: `Hi {{ first_name | default: "there" }}, here's what's new on the Acme engineering blog this week.`
  - Feed items block: Cards; image, title, date, author, 40-word excerpt and a "Read the post" button; up to 6, newest first
  - footer
- **Subject:** `{{ issue.items.first.title }}{% if issue.items.size > 1 %} and {{ issue.items.size | minus: 1 }} more{% endif %} from the Acme blog`
- **Preheader:** `New on blog.acme.io · issue {{ issue.number }}`
- **Public archive:** on.

### Issues
| Issue | Send time | Status | Posts | Recipients | Human opens | Clicks | Approved by |
|---|---|---|---|---|---|---|---|
| #3 | Fri 17 Oct, 08:00 | Collecting · 2 new posts so far | 2 | about 13,944 | – | – | – |
| #2 | Fri 10 Oct, 08:00 | Sent | 3 | 13,902 | 38.9% | 7.4% | Marcus Chen, 06:41 |
| #1 | Fri 3 Oct, 08:00 | Sent | 2 | 13,871 | 40.3% | 8.1% | Linh Nguyen, 07:55 |
| – | Fri 26 Sep, 08:00 | Skipped · no new posts since it was turned on | 0 | – | – | – | – |

### Feed items (newest 6 of 10)
| Title | Author | Published | Category | In the digest |
|---|---|---|---|---|
| Scheduled rollouts are here | Joel Weber | Wed 15 Oct, 10:04 | Announcements | Left out by the category filter |
| A year of on-call in numbers | Joel Weber | Wed 15 Oct, 07:30 | Engineering | New · goes in #3 |
| How we test database migrations | Mei Tanaka | Tue 14 Oct | Infrastructure | New · goes in #3 |
| Moving our ingest pipeline to Rust | Ana Lima | Wed 8 Oct | Engineering | Sent in #2 |
| Feature flags at 2,000 teams: what broke first | Joel Weber | Tue 7 Oct | Engineering | Sent in #2 |
| Postgres partitioning without downtime | Mei Tanaka | Mon 6 Oct | Infrastructure | Sent in #2 |

- **Older items:** "Our 2027 internship program" (Company, Thu 2 Oct) was left out by the filter. "Why we moved three services back to Postgres" (Wed 1 Oct) and "Load-testing with production traffic" (Tue 30 Sep) went in #1.

- Issue #3's subject, as it stands: "A year of on-call in numbers and 1 more from the Acme blog".
- The subject preview with 1 item: "A year of on-call in numbers from the Acme blog".

### Digest states
- **Feed unreachable** (Fri 17 Oct, 05:31): "blog.acme.io didn't answer: connection timed out after 15 s. Tried at 05:00, 05:15 and 05:30. Next try 05:45. If it still fails at 08:00, issue #3 is skipped."
- **Feed invalid:** "blog.acme.io/feed/ isn't a feed. Line 1 starts with `<!DOCTYPE html>`: the site may be showing a maintenance page."
- **Held** (Fri 17 Oct, 05:02): Blocked, "Routing: Postal is Down and Broadcasts failover is set to pause and ask". Fixes: "Send 100% via SES" (adds $0.14 and about 2 minutes) and "Skip this issue".
- **Waiting for approval** (Fri 17 Oct, 05:00): "Prepared at 05:00. Marcus Chen or Linh Nguyen can approve. It goes out at 08:00 once approved, and is skipped if it isn't approved by Sat 18 Oct, 08:00."
- **Paused by a person:** Sofia Rossi, Wed 15 Oct 10:20: "The blog moves to a new theme next week; images may break."
- **Paused automatically:** "Sofia Rossi's role changed from Editor to Content Writer on 15 Oct, so she no longer has campaigns:send. The digest paused. Anyone with campaigns:send can turn it back on and becomes its owner."
- **Draft mode, as Daniel Park:** "Sending automatically needs campaigns:send. Ask Sofia Rossi, Marcus Chen or Linh Nguyen. You can turn it on in draft mode: each issue becomes a draft you send through Review & send."

### Conditional content: Weekly Digest #143
- **Draft** by Sofia Rossi, for Tue 21 Oct 2026 at 09:00 America/New_York. Audience: the Weekly Digest list, 31,604 recipients.
- **Subject:** "This week: on-call numbers and safer migrations".
- **Conditional blocks:**
  1. "We've missed you" note at the top: engagement tier is At risk.
  2. "Beta corner" section: on the Beta Program list.
  3. "Start a free trial" banner: field `plan` is empty.
- **Versions** (6 of the 8 possible; At risk contacts on Beta Program don't occur):

| Version | Shows | Recipients | Size |
|---|---|---|---|
| 1 | none of the three | 9,812 | 61 KB |
| 2 | free trial | 16,940 | 66 KB |
| 3 | missed you | 1,166 | 63 KB |
| 4 | missed you, free trial | 2,284 | 68 KB |
| 5 | Beta corner | 812 | 97 KB |
| 6 | Beta corner, free trial | 590 | 104 KB |

- **Totals:** 31,604 recipients; At risk 3,450; on Beta Program 1,402.
- **Preview as Sam Rivera** (sam.rivera@outlook.com: Active, `plan` team, not on Beta Program): version 1, "9,812 recipients get this version".
- **Warning:** version 6 is 104 KB, so Gmail clips it for 590 recipients.
- **Broken condition:** a fourth block, "Annual billing offer", uses the segment "Customers with mrr over $50", which Marcus Chen deleted on Tue 14 Oct.
- **Empty version:** if the whole content section is set to "Only contacts on Beta Program", 30,202 recipients get nothing between the header and the footer. Previewing as Sam Rivera shows exactly that.

### Poll: topics, in Weekly Digest #143
- **Question:** "What should we write about next?"
- **Answers:** Postgres internals · Feature flag patterns · On-call stories · Rust in production.
- **Saved as tags:** `wants:postgres`, `wants:flags`, `wants:oncall`, `wants:rust`.
- **Closes** 7 days after sending, Tue 28 Oct. Results shown after answering.

### Poll: NPS, in Weekly Digest #141 (sent Tue 7 Oct: 31,520 sent, 31,299 delivered)
- **Question:** "How likely are you to recommend the Weekly Digest to a colleague?" 0–10, saved to the field `nps_score`. Closes Tue 21 Oct.
- **Results** on Wed 15 Oct, 10:00:
  - 2,412 answers, 7.7% of delivered. 61% came in the first 6 hours.
  - **NPS 26:** promoters 1,086 (45.0%), passives 868 (36.0%), detractors 458 (19.0%).
  - By score: 0: 41 · 1: 12 · 2: 18 · 3: 27 · 4: 39 · 5: 128 · 6: 193 · 7: 352 · 8: 516 · 9: 489 · 10: 597.
  - By tier: Active 1,932 answers, NPS 31 · Cooling 341, NPS 12 · At risk 125, NPS −8 · Sunset 14, too few to compare.
  - By language: English 2,108, NPS 27 · Spanish 201, NPS 19 · German 82, NPS 22 · Vietnamese 21, too few to compare.
  - 318 comments. 74 people changed their answer.
  - **1,377 automated visits ignored:** 1,129 from security gateways that open links before the reader does, 248 from prefetching proxies.
  - 9 answers from people who aren't contacts (the web archive), counted apart.
- **Comments:**
  - 10 · Priya Natarajan: "The Postgres pieces are the best thing in my inbox."
  - 9 · Omar Haddad: "Short enough to read on the train. Keep the links section."
  - 6 · Diego Alvarez: "Good, but every week is a lot. Monthly would suit me."
  - 3 · Not a contact: "Too much about feature flags lately."

### Survey: "Beta feedback · October 2026"
- **Created** by Sofia Rossi on Mon 13 Oct. Page: news.acme.io/s/beta-feedback-oct-2026. Closes Fri 31 Oct, 23:59 America/New_York. One response per contact, editable until it closes. Non-contacts not accepted.
- **Linked from** the campaign "Beta Program: October check-in", sent Mon 13 Oct at 10:00 to 2,094 Beta Program recipients. The email opens with a Poll block of question 1 that continues to the survey.
- **Questions and results** on Wed 15 Oct, 10:00 (688 started, 571 completed, 83.0%):
  1. How often do you use feature flags? Single choice. Daily 158 · Weekly 241 · Monthly 97 · Not yet 75.
  2. Which beta features have you tried? Multiple choice. Feature flags 498 · Audit log exports 287 · Usage alerts 164 · API v2 121.
  3. How likely are you to recommend Acme to a colleague? 0–10. NPS 41: promoters 290, passives 225, detractors 56.
  4. What's the one thing we should fix before general release? Long text, optional. 412 answers, such as "Flag rules by region are confusing in the UI."
  5. Can we contact you about your answers? Yes 342 (59.9%) · No 229. Saved to the field `research_ok`.
  6. Your role. Engineer 361 · Engineering manager 118 · Product 64 · Other 28. Saved to the field `role`.
- 117 people answered question 1 in the email and stopped there.

### AI assistant
- **Connection:** Anthropic, model Claude Sonnet 5.5 (`claude-sonnet-5-5`). Connected by Marcus Chen on Mon 6 Oct 2026. Key: write-only, "Added 6 Oct by Marcus Chen · Replace".
- **Prices:** $2.00 per million input tokens, $10.00 per million output tokens, pre-filled from ReputeCast's price list of 1 Oct 2026.
- **Cap:** $25.00 a month, set by Linh Nguyen. Used this month: $7.84 (1.62M input and 460K output tokens). Notice at $20.00.
- **Features:** all on.
- **This month by feature:** subject ideas $0.91 · rewrite and tone $1.64 · translate $3.86 · alt text $0.42 · accessibility fixes $0.19 · segment drafts $0.24 · report summaries $0.58.
- **By member:** Sofia Rossi $4.97 · Daniel Park $2.05 · Linh Nguyen $0.44 · Marcus Chen $0.38.
- **Activity log:**

| Time | Member | Feature | On | Tokens | Outcome |
|---|---|---|---|---|---|
| Wed 15 Oct 10:31 | Daniel Park | Subject ideas | Weekly Digest #143 | 1,240 | Accepted 1 of 5 |
| Wed 15 Oct 10:12 | Sofia Rossi | Translate to Spanish | Weekly Digest #143 | 9,880 | Accepted with edits |
| Wed 15 Oct 09:58 | Sofia Rossi | Alt text | Weekly Digest #143, 2 images | 2,410 | Accepted |
| Wed 15 Oct 09:40 | Marcus Chen | Describe the segment | new segment | 1,980 | Accepted |
| Tue 14 Oct 16:02 | Linh Nguyen | Report summary | October Product Update | 3,640 | Kept |
| Tue 14 Oct 11:20 | Daniel Park | Rewrite with an instruction | Weekly Digest #143 subject | 610 | Refused by the model |

- **Subject ideas** for Weekly Digest #143, each with the subject checks:
  1. "A year of on-call, in numbers" · 29 characters · Pass
  2. "How we test every database migration" · 36 · Pass
  3. "On-call data, safer migrations and what to read next" · 52 · Pass
  4. "You won't believe our on-call numbers" · 37 · Warn: "you won't believe" trips a spam heuristic
  5. "Weekly Digest #143: on-call, migrations, your vote" · 50 · Pass
- **Shorten** (the intro of #143):
  - Before (57 words): "Hi {{ first_name | default: "there" }}, this week we're sharing a lot of numbers from our first full year of on-call, and also a look at the way our database team tests every single migration before it ever reaches production, plus a quick poll at the end so you can tell us what you would like us to write about next."
  - Suggestion (29 words): "Hi {{ first_name | default: "there" }}, this week: a year of on-call in numbers, how our database team tests every migration before production, and a one-click poll on what we should write next."
- **Translate:** the Spanish variant of #143 was Missing. 14 blocks, the subject ("Un año de guardias, en cifras") and the preheader were translated; every token and link was kept. It shows "Translated by AI · not reviewed" until Sofia marks it reviewed. 2,744 recipients get Spanish.
- **Alt text:** oncall-heatmap.png, 600 × 240. Suggestion: "Heat map of on-call pages by hour and weekday in 2025. Most pages came between 09:00 and 11:00 on weekdays."
- **Accessibility fix:** the link text "click here" in the poll's intro. Suggestion: "See all the 2025 on-call charts".
- **Describe the segment** (Marcus Chen): "Team-plan customers who clicked something in the last 30 days but haven't opened anything in two weeks".
  - Draft: Custom field `plan` equals team; Clicked any email in the last 30 days; Last open not in the last 14 days. 214 contacts.
  - The model's note: "There's no condition for a specific link, so I used 'Clicked any email'."
- **Report summary** (October Product Update, kept by Linh Nguyen): "October Product Update reached 47,548 inboxes, 99.1% of the 47,980 sent. 41.2% opened it as people; another 18.6% were machine opens, which aren't counted as people. 6.3% clicked, and the feature-flags link drew 38% of the clicks. Subject B, 'What shipped in October, and what we're retiring', won the test with 42.6% human opens against 39.8%. Outlook had the highest bounce rate (1.0%). 112 people unsubscribed, most often for 'too many emails'."
- **Refusal:** Daniel asked "Make the subject look like a reply to them, start with Re:". The model answered: "I can't write a subject that pretends to be a reply to an email the reader never sent." Nothing changed.
- **Not configured:** the workspace "Acme Engineering Blog" has no model connected.

### People
- **Sofia Rossi** (Editor) owns the digest, builds Weekly Digest #143 and created the survey.
- **Daniel Park** (Content Writer) created the digest in draft mode and writes with the assistant. He can't turn on automatic sending or build surveys (no forms:edit).
- **Marcus Chen** (Admin) connected the blog and the model, and approves issues.
- **Linh Nguyen** (Owner) set the AI cap and approves issues.
- **Aisha Bello** (Viewer) sees digests, results and kept summaries. She has no ai:use.

---

## 5. New screens

No new sidebar items. Digests live in Campaigns, polls in the composer and the report, surveys in Forms & pages › Forms, and the AI assistant's settings in Settings › Integrations.

### 5.1 New campaign: type picker
A dialog from "New campaign" on Campaigns and on the Overview.
- Two option cards:
  - **Regular:** "One email, sent once. A/B tests live here."
  - **Recurring digest:** "Built from a feed on a schedule. Each issue is checked and, when your rules say so, approved before it goes out."
- The digest card names what it needs: "a feed address or a connected WordPress site". When blog.acme.io is connected, it offers it directly.
- Continue: Regular opens the composer on the template gallery, as today. Recurring digest opens 5.2.

States to show:
- default, with blog.acme.io offered
- no WordPress site connected: the card asks for a feed address
- Daniel Park: both cards available; the digest card adds "You can build it and run it in draft mode"

### 5.2 Digest setup
A settings page: breadcrumb Campaigns › Engineering blog roundup › Settings. Sections on the left, live previews on the right.
- **Name.**
- **Feed:**
  - source: a feed address, or a connected WordPress site
  - "Find feed" from a page address
  - categories, "Leave out items older than 14 days", and the starting point
  - a status line: "Last read 10:05 · RSS 2.0 · 10 items · reported by the WordPress plugin"
- **Schedule:**
  - frequency, day, time and time zone, or "When N new items arrive" with the sending window
  - preparation lead time and late limit
  - a sentence that sums it up: "Every Friday at 08:00 America/New_York. Prepared at 05:00. Skipped when there's nothing new."
- **Audience and sender:** list or segment with the live count, From, and the route as a weighted bar.
- **Sending mode:** two option cards, Send automatically and Create a draft.
- **Approvals:** what every issue will match: "With your approval policy, every issue needs approval: the audience is over 10,000 recipients, and the route includes a Warming-up provider until Fri 24 Oct."
- **Public archive:** a switch.
- **Right column:**
  - **Feed preview:** the newest items with a badge each: New, Sent in #2, Left out by a filter.
  - **Next issue:** date, posts so far, recipients, and the subject with 1, 3 and 6 items.
- **Actions:** Save, "Design the email" (5.3), and Turn on.
- **Turn-on confirmation** (shown here as Sofia turns it back on after a pause): "Turn on Engineering blog roundup? The next issue goes out Fri 17 Oct at 08:00 America/New_York to about 13,944 recipients, if there are new posts. Each issue runs Send Readiness and your approval rules. It sends under your name: if you lose campaigns:send, it pauses."

States to show:
- the sample, as Sofia Rossi
- finding the feed from blog.acme.io
- feed unreachable on the first read, with the exact error
- feed invalid: an HTML page instead of a feed
- WordPress site unreachable, or its plugin outdated (phase 2's states)
- on, with no new items yet: "Nothing new since you turned it on. The first issue goes out after the next post."
- the turn-on confirmation
- Daniel Park: "Send automatically" disabled with the reason and who can help; draft mode available
- read-only (Aisha Bello)
- validation error: no category left after filtering
- saved, loading, error

### 5.3 Composer: digest mode
The campaign composer, with the digest's template.
- **Header:**
  - a "Recurring digest" chip and the digest's name
  - "Preview with": Next issue (#3, 2 posts), Last issue (#2), 1 item, 6 items
  - Send test, which sends the next issue as it would be built now
  - no "Review & send": issues are checked when they're prepared; "Check next issue" runs Send Readiness on demand
- **Blocks:** Feed items in the ReputeCast group, in digest mode only.
- **Feed items block selected:**
  - layout as a segmented control: List, Cards, Grid
  - fields to show, excerpt length, button text
  - maximum items, order, and what happens to extra items
  - fallback image for items without one
- **Canvas:** renders the chosen preview data. Each item carries a small label: "Feed item 1 of 2".
- **Subject and preheader:** feed tokens in the token menu, and the subject previewed with 1, 3 and 6 items.
- **Checks:** size measured with the maximum number of items ("With 6 items: 88 KB"); alt text taken from the title is flagged.

States to show:
- Cards with issue #3's 2 posts (sample)
- List layout, and Grid on mobile
- preview with 6 items
- the subject without the `{% if %}`, caught by the preview: "…and 0 more"
- an item without an image: the fallback image, and alt text from the title
- editing a prepared issue: "You're editing issue #3, prepared at 05:00. Changes apply to this issue only." with "Rebuild with the current template"
- read-only

### 5.4 Digest overview and issues
Opened from the digest's row in Campaigns.
- **Header:**
  - name and status badge: On · Send automatically, On · Draft mode, Paused or Off
  - owner, schedule and audience in one line
  - actions: Pause, Settings, Design, Preview next issue, Check next issue (Send Readiness on the issue as it would be built now), Send test of the next issue
- **Next issue card:** date and time, posts so far, estimated recipients, the subject as it stands, and "Needs approval" with the matched rules.
- **Feed card:** source, last read, items waiting.
- **Issues table:**
  - issue, send time, status badge with its reason
  - posts, recipients, human opens and clicks, each compared with the previous issue
  - approved by
  - actions: Report, Open, Release, Skip
- **Trend:** human opens of the last sent issues, one bar each.

States to show:
- the sample: on, collecting for #3
- waiting for approval (Fri 17 Oct, 05:00)
- held: Blocked by routing, with "Send 100% via SES" and "Skip this issue"
- sending #3, linking to the live monitor
- feed unreachable, retrying
- skipped issues: no new posts, feed unreachable, not approved in time
- paused by Sofia Rossi, with her note
- paused because the owner lost campaigns:send
- paused after three feed failures in a row
- draft mode: "Draft ready · not sent yet"
- just turned on, no issues yet
- read-only (Aisha Bello)
- loading, error

### 5.5 Composer: Visibility and versions
Weekly Digest #143, with "Beta corner" selected.
- **Inspector, Settings tab, Visibility section:**
  - Everyone, Only contacts who match, or Everyone except contacts who match
  - condition rows named like the segment builder's (field, operator, value), with ALL or ANY
  - "1,402 of 31,604 recipients see this block"
- **Canvas:** a badge on each conditional block's label; the toggle "Show all blocks" or "Show as a contact sees it".
- **Layers:** an eye-off icon on conditional rows.
- **Versions:** "3 conditional blocks · 6 versions" in the composer's footer bar opens the versions table: blocks shown, recipients, size, "Preview this version".
- **Preview as:** the contact picker, the version's name and recipients, and "Show why".

States to show:
- the sample
- the condition picker open: segment, list, tag, field, language, engagement tier
- the footer selected: "The footer is always shown"
- previewing as Sam Rivera: version 1
- version 6 over Gmail's clipping size
- a broken condition: the deleted segment, with "Pick another segment", "Remove the condition" and "Hide this block for everyone"
- an empty version, and previewing as Sam Rivera who sees nothing between the header and the footer
- counting versions
- more than 50 versions: which ones are checked
- read-only

### 5.6 Composer: Poll block
The topics poll in Weekly Digest #143.
- **Blocks:** Poll in the ReputeCast group.
- **Inspector, Settings tab:**
  - kind: Choice, 0–10 scale, Emoji scale
  - the question; the answers, two to five, with add, remove and reorder
  - layout
  - save to: a tag per answer, or a field
  - close: on a date, after N days, or by hand
  - after answering: thank-you message, or continue to a survey
  - non-contacts: count apart, or don't accept
  - show results after answering
  - "Preview the answer page"
- **Canvas:** the poll as buttons; the mobile preview; the plain-text preview as a list of links.

States to show:
- the sample, a Choice poll
- a 0–10 scale wrapping to two rows on mobile
- an emoji scale
- "Continue to a survey" chosen, with the survey picker
- check: "The poll closes Mon 20 Oct, before this campaign sends on Tue 21 Oct" (Warn)
- check: the publisher domain isn't verified, so answer links use reputecast.acme.io (Warn)
- after a test send: "Test answers aren't counted"
- read-only

### 5.7 Public answer page and survey page
Public pages at 720 px and on phones, in the publisher's branding (Acme logo, Acme indigo), never the teal.
- **Answer page:**
  - the question, with the chosen answer selected
  - "Thanks, Sam. Your answer is saved." in a status region
  - the other answers, to change it
  - "Anything to add?" with Send
  - the results, when the poll shows them
  - the usual footer: Privacy · Email preferences · Sent with ReputeCast
- **Survey page:**
  - title, description and the questions on one page, with Submit
  - question 1 filled in from the email's poll
  - after submitting: "Your answers are saved. You can change them until Fri 31 Oct."

States to show (answer page):
- saved, with JavaScript (NPS poll, Sam Rivera answered 9)
- without JavaScript: the answer preselected and "Confirm my answer"
- answer changed: "Changed from 7 to 9"
- comment sent
- results shown (the topics poll)
- closed: "This poll closed on Tue 21 Oct."
- not a contact (opened from the archive): "Thanks. Your answer is counted, but it isn't linked to you."
- non-contacts not accepted: "This poll is for subscribers", with Subscribe
- a broken or altered link: "This link doesn't work anymore"
- couldn't save, with Try again
- phone

States to show (survey page):
- open, with question 1 filled in
- a required question left empty: the form-level error summary and the message under the question
- submitted
- changing an earlier response
- closed
- non-contacts not accepted
- phone

### 5.8 Report: poll results
A Poll section in the campaign report, after the click map. The NPS poll of Weekly Digest #141.
- **Header:** the question; status ("Open · closes Tue 21 Oct" or "Closed"); answers and their share of delivered; "1,377 automated visits ignored", with an explanation on hover or focus.
- **0–10 scale:** the NPS as a stat tile, a bar of promoters, passives and detractors, and the 11 bars.
- **Choice and emoji polls:** one bar per answer with count and share.
- **Breakdown:** by engagement tier, list or language, as a table. Fewer than 30 answers in a row: "Too few to compare".
- **Over time:** cumulative answers over the first 72 hours.
- **Comments:** score, comment, contact (or "Not a contact") and time. Filter by promoters, passives and detractors; search.
- **Footer line:** answers changed, answers from non-contacts.
- **Actions:** Close poll now, Export results, Export answers with contacts.

States to show:
- the NPS sample
- automated visits filtered: the explanation open, "1,129 from security gateways that open links before the reader does, 248 from prefetching proxies. None of them counts as an answer."
- a Choice poll's results
- closed
- results still coming in, in the first hours
- no answers yet
- Daniel Park: "Exporting answers with contacts needs subscribers:export. Ask Marcus Chen or Linh Nguyen."
- loading, error

### 5.9 Survey builder and results
In Forms & pages › Forms, which now lists signup forms and surveys with a Type column. "New form…" asks "Signup form or Survey".
- **Questions tab:**
  - left: the questions, reorderable, and "Add question" with the six types
  - the selected question: type, text, description, required, answers, and save to a field or tags
  - right: the live page preview, desktop and mobile
  - top: status (Draft, Open · closes Fri 31 Oct, Closed) and actions: Publish, Close, Copy link
- **Settings tab:** closing date, one response per contact, non-contacts, thank-you text.
- **Results tab:**
  - started and completed, with the completion rate
  - a chart per question; long-text answers as a list
  - responses: contact, completed at, answers (contacts shown with subscribers:view)
  - the two exports

States to show:
- a new survey with no questions: an empty state that adds the first question
- the sample, editing question 3
- validation: "A choice question needs at least two answers"
- publishing
- open, with the results
- closed
- Daniel Park, who lacks forms:edit: read-only, with the reason and who can help
- loading, error

### 5.10 Settings › Integrations › AI assistant
The phase 2 integration detail page, laid out like Zapier's (`IntegrationsZapier.dc.html`), for the "AI models" group.
- **Not connected:** the five providers as catalog cards with capability badges, and "Off until you connect a model. Nothing is sent anywhere until then."
- **Connect dialog:**
  - provider, base URL where needed, API key (write-only), model
  - Test connection
  - monthly cap and prices
  - feature switches
  - what is sent, as the outbound-data notice
- **Connected:**
  - status: "Connected · Anthropic · Claude Sonnet 5.5", the last test, and the key as a masked value with Replace
  - usage meter: "$7.84 of $25.00 this month · resets 1 Nov"
  - usage by feature and by member
  - feature switches
  - what is sent
  - the AI activity log
  - Disconnect, which lists what stops working

States to show:
- not connected, as Linh Nguyen
- the connect dialog, Anthropic filled in
- an OpenAI-compatible server on a private address: allowed, and refused ("ollama.acme.internal resolves to 10.0.4.12, a private address. An instance admin can allow private addresses.")
- test passed, and test failed (key rejected)
- a model without image input: "Alt text needs a model that reads images. It stays off."
- usage at 80%, and the cap reached
- the key rejected after connecting: status Error, with Replace
- read-only for members without integrations:manage: status, features, cap, usage and what is sent; no key, no switches
- disconnect confirmation
- loading, error

### 5.11 Composer: AI suggestions
Weekly Digest #143, as Sofia Rossi and Daniel Park.
- **Entry points:**
  - an "Assist" menu in the composer header: Subject ideas, Rewrite selection, Translate this variant, Alt text for images, Fix accessibility issues
  - the rich-text toolbar: Shorten, Friendlier, More formal, More direct, With an instruction…
  - "Ideas" beside the subject
  - "Suggest alt text" in an image's Settings
  - "Suggest a fix" on an Accessibility panel issue (phase 2)
  - "Translate with AI" wherever phase 2 shows it disabled: the language menu, the compare view and the Add language dialog
- **Suggestion card** (section 7): next to the element, or in the right panel for longer work.
- **Subject ideas:** five cards, each with its checks and "Use this".
- **Rewrite:** an inline diff in the text block, with Accept, Edit, Reject and Try again.
- **Translate:** phase 2's side-by-side compare, a diff per block, "Accept block" and "Accept all", and the token-and-link check. Afterwards the language chip shows "AI · not reviewed".
- **Card footer:** "Sent to Anthropic (Claude Sonnet 5.5): this paragraph, 412 characters. No contact data. About $0.01."

States to show:
- subject ideas (sample), with one Warn
- Shorten (sample diff)
- translating to Spanish, with all tokens kept
- alt text
- accessibility fix for "click here"
- working: a skeleton with "Writing…" and Cancel
- a reputation question in an instruction: with HostRepute connected, and without
- refusal, in the model's words
- provider error: "Anthropic returned an error (529, overloaded) twice. Nothing was changed."
- rate limited: "Try again in 20 s", counting down
- cap reached: "This month's AI budget ($25.00) is used up. AI features return on 1 Nov, or Linh Nguyen or Marcus Chen can raise the cap."
- not configured, as an invitation and never a paywall or upgrade prompt: for Linh Nguyen ("Connect a model") and for Daniel Park ("Marcus Chen or Linh Nguyen can connect one")
- a feature turned off: "Translation with AI is turned off for this workspace"
- tablet, and phone (the card stacks under the canvas)

### 5.12 AI in segments and reports
- **Segment builder:**
  - "Describe the segment" above the conditions, with Draft
  - the draft as a diff on the builder: added conditions marked, removed ones struck through, the live count updated
  - the model's note
  - Accept, Edit, Reject
  - SQL mode is never drafted
- **Report:**
  - "Summarize results" in the report header
  - the suggestion card with the summary: Keep, Discard, Try again
  - a kept summary at the top of the report, labelled "Summary by AI, kept by Linh Nguyen", never on read-only links
  - "Questions about sender reputation? Ask HostRepute's assistant" when HostRepute is connected

States to show:
- Marcus Chen's segment draft (sample)
- a draft that replaces existing conditions
- nothing could be drafted: "There's no condition for 'opened on a phone'. Nothing was drafted."
- the October Product Update summary, then kept
- Aisha Bello viewing a kept summary: no Summarize button
- not configured, loading, error

---

## 6. Changes to existing screens

- **Campaigns:**
  - "New campaign" opens the type picker (5.1).
  - A "Recurring · 1" tab lists digests: name, status, schedule, next issue, the last issue's human opens, owner.
  - Each issue appears as a campaign with a "Digest" badge that links to its digest. A prepared issue appears under Scheduled.
- **Overview:** the "Next scheduled campaign" card can be an issue: "Engineering blog roundup #3 · Fri 17 Oct, 08:00 · 2 posts so far · needs approval". Recent campaigns include sent issues.
- **Composer:**
  - Blocks: Poll in the ReputeCast group, and Feed items in digest mode.
  - Settings tab: the Visibility section. Canvas: visibility badges. Footer bar: "3 conditional blocks · 6 versions".
  - Header: the "Assist" menu. Rich-text toolbar: the rewrite actions.
- **Template gallery (ComposerEmpty):** a "Blog digest" starter, shown in digest mode only.
- **Composer previews:** "Preview as" names the version and its recipients, with "Show why". Poll blocks preview with their answer links.
- **Review & send:**
  - A "Conditional content" group (3.2).
  - Content checks for polls: the poll closes before the send; the answer page's domain isn't verified.
  - For an issue: "Issue #3 of Engineering blog roundup · prepared 05:00", the matched approval rules, and "Release issue" in place of Schedule when it's held.
- **Send approvals (inbox and review):** a "Recurring digest" badge. The review shows the issue's posts and what changed since the last approved issue: template, audience, route and schedule. The button reads "Approve issue #3".
- **Settings › Approvals:** under the rules, "Rules apply to each issue of a recurring digest."
- **Campaign report:**
  - the Poll section (5.8)
  - "Summarize results" and a kept summary (5.12)
  - for an issue: each number compared with the previous issue, and a link to the digest
  - the click map shows a poll as one region, "Poll · 2,412 answers", from the answer page's records rather than clicks
- **Segment builder:** "Describe the segment" (5.12). A segment used by a Visibility rule says where ("Used by 1 conditional block in Weekly Digest #143"), and deleting it warns first.
- **Automation flow:**
  - Trigger "Answered a poll or survey": pick the poll or survey, filter by answer, and "Start again when the answer changes".
  - Email steps get Visibility, Poll blocks and Assist, as in the composer.
- **Forms & pages › Forms (FormBuilder):** the tab "Signup forms · 3" becomes "Forms · 4" and lists signup forms and surveys with a Type column. "New form…" asks which kind.
- **Forms empty state (FormsStates):** "Signup forms and surveys you create show up here."
- **Public archive:** issues of digests with "Show in the archive" on, with the digest as a filter. Archived emails with a poll record "Not a contact" answers, or show "This poll is closed".
- **Contact profile:**
  - The activity timeline lists poll answers and survey responses: "Answered 9 to 'How likely are you to recommend the Weekly Digest…' in Weekly Digest #141 · with a comment".
  - A field set by a poll shows where it came from: "`nps_score` 9 · from a poll, 7 Oct".
- **Settings › Integrations (phase 2 hub):** the "AI models" group shows the connected Anthropic card with its usage meter, and the other providers.
- **Roles & permissions, role editor:** a new "AI" group with ai:use. Built-in roles as in 3.5. Adding it to Content Writer shows the impact preview: "1 member gains ai:use".
- **API keys & MCP agents:** ai:use isn't offered. The permission list says why: "AI features are for people."
- **Settings › MCP server:** read-only tools `list_digests`, `get_digest_issues` and `get_poll_results`. The agents matrix states that agents can't turn on digests or release issues.
- **Audit log:** the actions in 3.6, and the filter categories Digests, Polls & surveys, and AI.
- **Notifications:** the kinds in 3.7, in the list and in the preferences.
- **Phase 2 screens:** "Translate with AI" becomes active in the language menu, the compare view and the Add language dialog; Accessibility panel issues get "Suggest a fix" with AI for link text and alt text.

---

## 7. New design-system patterns

Add these to the "9 Design system" page before the screens. Use the existing tokens, badges and components.

- **Visibility badge:** on a conditional block's label, an eye-off icon and the condition in words ("Beta Program list"), in the `info` badge. A broken condition uses `bad`. The same icon marks the row in Layers.
- **Email-safe poll buttons:**
  - MJML buttons in the publisher's accent, at least 44 px tall on mobile.
  - The 0–10 row has 11 equal cells and wraps to 6 and 5 on mobile. Ends labelled "Not likely" and "Very likely".
  - Emoji answers carry words for screen readers and in plain text ("1 of 5, very unhappy").
  - The answer page shows the selected answer with a check icon and the word "Your answer".
- **AI suggestion diff card:**
  - header: the feature and the model ("Shorten · Claude Sonnet 5.5")
  - body: an inline diff for short text, side by side for blocks, using the existing diff component
  - a checks row: subject heuristics, or "All 4 tokens and 3 links kept"
  - actions: Accept, Edit, Reject, Try again
  - footer: what was sent and the estimated cost
  - working state: skeleton lines with Cancel. No spinner alone.
- **Issue status badges:** Collecting (neutral), Prepared (info), Waiting for approval (warn), Held (bad), Sending (info, pulse), Sent (ok), Skipped (neutral, with the reason), Not sent (neutral).
- **Outbound-data notice:** an `info` note that names the service and lists "Sent" and "Never sent". Phase 4 reuses it for verification services.
- **Feed item row:** thumbnail, title, date and author, and a badge: New, Sent in #2, Left out.
- **NPS bar:** promoters, passives and detractors as one stacked bar in three steps of one chart hue, labelled with words and counts. Never status colours.

---

## 8. Conflicts to note

1. **Audit log vs AI activity log.** CLAUDE.md says every state-changing action writes an audit entry. An AI request changes nothing until a person accepts it, and accepted changes land in the version history. This brief keeps each request in the AI activity log (90 days, no content) and audits the configuration (3.6), as phase 1 did for the message log. The owner needs to confirm this.
2. **Automatic sending and approvals.** The plan runs the approval rules on every issue. Under Acme's policy (audience ≥ 10,000) every issue of this digest waits for an approver, so "Send automatically" means "send once approved". An option to approve a digest once for its template, audience and route isn't in this brief; the owner may want it.
3. **Who sends an issue.** Nobody presses Send. Issues go out on the owner's authority, recorded as "System · {digest}, turned on by {owner}", and pause when the owner loses campaigns:send, like API keys lose their permissions when their creator leaves. The rule that MCP agents never send still holds.
4. **The Forms tab.** The FormBuilder board names the tab "Signup forms · 3"; the plan's navigation (section 4) calls it "Forms". Surveys are why it changes.
5. **Poll links aren't tracked clicks.** The main spec's click map counts link clicks. Poll answers come from the answer page, so the map shows a poll as one region with its answer count.
6. **What AI report summaries send.** The plan's rule is "never contact data". A report summary sends a campaign's aggregate numbers, which aren't contact data but do leave the instance. The settings say so and the feature can be switched off. The HostRepute data boundary doesn't change.
7. **Content edits.** Visibility rules, Poll blocks and accepted suggestions are content edits, kept in the version history like every content edit today, not in the audit log.
8. **Composer blocks.** The main spec (7.4) lists the composer's blocks. This brief adds Feed items and Poll as ReputeCast blocks.
9. **The blog's feed address.** This brief uses blog.acme.io/feed/. Phase 2's sample (`reputecast-design-addon-connect-reach.md`) has the plugin reporting https://blog.acme.io/feed/, WordPress's default. The two need to agree before either is drawn.

---

## 9. Boards

New boards, on the canvas page "P3 Create":

| File | Canvas title | Implements | Tablet and phone |
|---|---|---|---|
| `DigestSetup.dc.html` | Digest · type picker and setup | 5.1, 5.2 | `DigestSetupTablet.dc.html` |
| `DigestComposer.dc.html` | Composer · digest mode and Feed items block | 5.3 | — |
| `DigestOverview.dc.html` | Digest · overview and issues | 5.4 | `DigestOverviewTablet.dc.html`, `DigestOverviewPhone.dc.html` |
| `DigestStates.dc.html` | Digest · waiting for approval, held, skipped, paused, draft mode, feed errors | 5.2–5.4 states | — |
| `ComposerVisibility.dc.html` | Composer · Visibility, versions and preview as | 5.5 | — |
| `VisibilityStates.dc.html` | Visibility · broken condition, empty version, over size, Send Readiness group | 5.5 states, section 6 (Review & send) | — |
| `ComposerPoll.dc.html` | Composer · Poll block | 5.6 | — |
| `PublicPoll.dc.html` | Public · poll answer page and survey page (720) | 5.7 | `PublicPollPhone.dc.html` |
| `PollResults.dc.html` | Report · poll results | 5.8 | — |
| `SurveyBuilder.dc.html` | Forms · survey builder and results | 5.9 | — |
| `AISettings.dc.html` | Integrations · AI assistant | 5.10 | — |
| `AIComposer.dc.html` | Composer · AI suggestions | 5.11 | `AIComposerTablet.dc.html`, `AIComposerPhone.dc.html` |
| `AISegmentReport.dc.html` | Segments and report · AI drafts and summaries | 5.12 | — |
| `AIStates.dc.html` | AI · not configured, errors, refusal, cap, reputation questions | 5.10–5.12 states | — |

14 desktop boards and 6 tablet and phone boards.

Existing boards that section 6 changes:
- `Campaigns.dc.html`, `CampaignsTablet.dc.html`, `CampaignsPhone.dc.html`
- `Main.dc.html`
- `Composer.dc.html`, `ComposerEmpty.dc.html`, `ComposerPreview.dc.html`
- `SendReadiness.dc.html`, `SendReadinessStates.dc.html`
- `ApprovalInbox.dc.html`, `ApprovalReview.dc.html`, `ApprovalPolicy.dc.html`
- `Report.dc.html`
- `SegmentBuilder.dc.html`
- `AutomationFlow.dc.html`
- `FormBuilder.dc.html`, `FormsStates.dc.html`
- `PublicArchive.dc.html`
- `ContactProfile.dc.html`
- Phase 2's `IntegrationsCatalog.dc.html` (the AI models group)
- `Roles.dc.html`, `RoleEditor.dc.html`
- `SettingsKeys.dc.html`, `SettingsKeysStates.dc.html`, `SettingsMCP.dc.html`
- `SettingsAudit.dc.html`
- `Notifications.dc.html`
- `DSPatterns.dc.html`, `DSComponents.dc.html` (section 7), `DSCoverage.dc.html` (the new screens and states)
- Phase 2's `ComposerLanguages.dc.html`, `ComposerCompare.dc.html` and `ComposerAccessibility.dc.html` (Translate with AI and Suggest a fix become active)
