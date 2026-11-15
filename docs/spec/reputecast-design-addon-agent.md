# ReputeCast — Add-on spec: AI agent

The AI agent turns a brief into reviewed drafts. It never sends, schedules, publishes, approves or changes workspace settings. This is the A0 product and design brief for the third competitor-gap batch, dated 13 November 2026; it does not claim that the agent is implemented.

## 0. Status and use

The owner accepted the seven decisions below. A0 delivers this brief and the boards; A1–A5 deliver the product in separate milestones. This brief supersedes the whole-campaign and task-conversation exclusions in `reputecast-design-addon-create.md`, only for the flows described here. Existing one-shot suggestions keep their accept-before-write behavior.

Use the existing Clearwater tokens, shell, composer, segment builder, automation editor, permissions, audit and approval rules. All product copy is English; implementation must localize into the app's five languages. All features remain free and self-hosted, using the workspace's own model account.

## 1. The seven accepted decisions

1. **Scope:** whole-campaign generation and conversation within a named task/run. No general-purpose chat detached from a task.
2. **Data:** under the owner's approved controlled option 2, the model may receive human-reviewed editorial text, selected brand context, projected schema and permitted aggregate counts. No contact-record, recipient-list, secret or export tools; contact-derived labels/values stay in local bindings. Liquid stays unexpanded. Text filters are rejection safeguards, not proof that arbitrary prose is anonymous. People must review what they share and have authority to share it; the UI discloses remaining privacy risk and provider data policies.
3. **Writes:** after a person accepts a plan, the agent may create and revise new drafts belonging to that run. Every change to a pre-existing resource is a proposed diff accepted by a person. No sending, scheduling, publishing, approval, test sends, suppression, routing or settings changes.
4. **Permissions:** `ai:use` plus the relevant resource permissions. An Agent switch in AI settings is controlled by `integrations:manage`. No new permission or role.
5. **Audit:** the initiating user remains the actor. Record `via: ai_agent` and `run_id` in metadata; never introduce another actor type. Human decisions identify the person who made them, with the same run provenance.
6. **Retention:** brief, plan and step summaries for 30 days; model messages and sanitized tool results while the run is open, plus seven days, bounded by the run's 30-day expiry. Only the initiator and workspace integration managers may read a run, still subject to workspace membership and resource scope.
7. **Budget:** default ceilings of 40 tool calls, 300,000 input plus output tokens, and US$2 per run. Integration managers may adjust limits; monthly caps still apply. Reaching any ceiling pauses the run without losing drafts.

## 2. Terms and ownership

- **Run:** a user-owned task in one workspace, with a brief, questions, a versioned plan, steps, conversation, draft bundle, proposals and usage ledger.
- **Plan:** a reviewable list of intended reads, new drafts, changes to existing drafts, dependencies and estimated cost. Build confirms this version, not all future plans.
- **Step:** a bounded model or tool operation, with status, public result summary, usage and idempotency key. The UI displays actions and results, never hidden model reasoning.
- **Run draft:** a new object staged in this run. It cannot enter a live audience, be selected by another send, trigger an automation, be indexed on a public website or change a synced block. Kept objects are ordinary drafts; AI generation is provenance, not an approval.
- **Proposal:** a durable before/after diff against one exact resource revision. Accepting applies that diff; Reject changes no resource. This is not the current MCP `propose_*` notification and never authorizes sending.
- **Bundle:** the dependency-connected set of drafts, such as one segment, an unpublished automation and its three embedded emails. Keep or discard the dependency set coherently; never leave a flow referencing a deleted email.

## 3. Product requirements

### 3.1 Start, clarify and confirm the plan

Entry points: Campaigns and Automations › New › With the agent; the composer's Assist menu › Draft from a brief; the automation editor › Describe the automation; command palette › Ask the agent. All lead to a scoped task. `/w/{workspace}/agent` lists runs and `/w/{workspace}/agent/{run}` opens one. These routes are planned, not present in A0.

The form has the brief, goal, language, source template or brand kit, and explicit resource context. Show the model, what leaves the instance and the budget before the first model request. Require a fresh **Share reviewed content** confirmation before submitting a brief, voice-learning sample, conversation instruction or newly selected source. Cancellation sends nothing. Creating a plan itself consumes usage, even if Build is never pressed. Empty or oversized input is rejected locally; no secret fields, uploads of contact lists or contact picker.

The agent asks at most three questions at a time. It must resolve missing consent/audience semantics, an unknown field or an ambiguous incentive rather than inventing a mapping. An unsupported condition is explained with an available alternative. A material change to audience, resource set, dependencies or limits creates a new plan version for confirmation. Refining copy inside the accepted scope does not need a new plan.

The plan names each new draft and each existing resource it proposes to change, the permissions required, the validators, dependencies, aggregate audience size with source and freshness, and a cost range. Counts are advisory snapshots, never a send commitment. A failed count says unavailable, not zero. Actions are **Build drafts**, **Edit brief**, and **Cancel**. No writes to product resources occur before Build.

### 3.2 Create, revise and review drafts

- Campaigns use the existing composer blocks, MJML compilation, Liquid handling, brand styles and subject checks. Stored HTML and MJML must agree. AI may suggest three subjects, but never chooses a sending time or audience outside the plan.
- Automations use only registered triggers/steps and the existing graph validator. Validate the unpublished graph and its email content. Publish checks and Send Readiness still run later, in the human workflow; passing a draft check is not permission to publish.
- Segments use builder conditions, never user-written SQL. Stage the conditions in the run until Keep, because a normal segment can affect running automations. Count with the same scoped, read-only evaluation path, without returning members.
- Posts remain unpublished and invisible to public archives, feeds and indexing. Existing published content, active segments and live flows can be read in sanitized form but are not in-place mutation targets. Offer a draft copy or an unpublished next version instead.
- Existing drafts get field-level diffs. Accept requires the resource's normal edit permission, checks the base revision and reruns validators. A changed base produces **Changed since this proposal**; refresh the diff for another human decision, never last-write-wins. Existing approvals become stale under the existing rules.
- Existing blocks, templates, brands, pools and products are references, not permission to modify their definitions. Keep coupon tokens unexpanded; read only pool metadata and count, never codes, reserve operations or store write APIs.
- Validation permits at most two model repair attempts per failed output within the same budgets. Persist only valid runnable draft artifacts. After the second failure, retain successful drafts and show the failing field/node and **Edit manually**; do not silently remove an audience condition or claim success.

Review has two clearly separated sections: **New drafts** with Keep/Discard, and **Changes to existing drafts** with Accept/Reject. A review summary counts undecided bundles/proposals. Keep makes a bundle ordinary drafts atomically; it does not enroll contacts or publish a flow. Run ownership ends on Keep: later agent edits are proposals, even in the same run.

Discard deletes only unkept staging objects and dependent staging objects named in the confirmation. If the user removes an email required by a bundle, show the impact and require a revised flow before Keep. Already-kept or externally edited resources are never deleted by Discard all, Cancel, retry or retention. Rejecting a proposal does not delete its target. Decisions are idempotent and validated server-side.

Conversation remains attached to the run and to the selected draft when applicable: “Make email two shorter”. Show a new reviewable revision and retain the user's previous decision; do not overwrite a kept draft. Handoff actions are **Open in composer**, **Open in flow editor**, **Open in Review & send** or **Open approval review**, according to permissions. They navigate only; sending, publishing and requesting/issuing approval remain explicit actions on the existing screens.

### 3.3 Runtime, cancellation and resumption

Visible states are Draft brief, Planning, Needs answers, Ready to build, Queued, Building, Paused, Ready to review, Completed, Cancelled and Expired. Failed steps may coexist with successful drafts; “Completed” means all review decisions are resolved, not that anything was sent.

Use durable River work and persisted checkpoints. Save model responses and tool-call IDs before executing writes. A repeated step uses its recorded result; create/update plus step result must commit atomically or use a domain idempotency record. Never promise exactly-once calls to an external model. Reconcile uncertain requests and reserve usage until reconciled; do not hide their possible cost or automatically charge another full request.

One active model execution per user and two per workspace is the initial operational default, not a paid tier. Queue extra runs with a position and Cancel; enforce across processes. Waiting for a person consumes no execution slot. SSE may report progress, but reconnect fetches the persisted state and last event cursor rather than restarting a step.

Stop asks for cancellation, prevents subsequent steps and fences late writes with the run's state/version. An in-flight provider call may still be billed. Cancel preserves successful unkept drafts for review and presents a separate Discard action. Worker restarts resume from checkpoints. Revoked membership, permissions, resource scope or Agent switch pause before the next read/write, including before committing a response. Re-authorize every continuation and decision, not just run creation.

### 3.4 Budget is enforced, not just displayed

Track every planning, repair, retry and continuation request in the same run ledger and the workspace monthly usage. Tool counts include attempted/denied calls; token totals include the resent conversation/context. Bind model and pricing revision to each request. Unknown price or missing trustworthy token usage blocks cost-bounded continuation until a manager configures pricing or the request is reconciled; show “Cost unknown”, never US$0.

Before a model request, atomically reserve a conservative maximum for its input and bounded output against both run and monthly caps. Set output limits to fit. Release unused reservations after accounting; provider invoices can differ from the configured estimate, which the UI states. Bound responses, context size and wall-clock duration as well as steps.

At a ceiling show used/limit for calls, tokens and estimated dollars, completed drafts and the exact next action. **Continue** never resets counters or silently buys another US$2: a manager first raises the cumulative run ceiling, or the runner revises the task to fit remaining budget. Raising a run ceiling cannot bypass a monthly cap. A non-manager sees **Ask an integration manager**. The manager's confirmation shows old/new limits and monthly headroom; both the change and continuation are audited. A paid-model failure keeps reservations until its billing uncertainty is settled.

### 3.5 Privacy and tool boundaries

Use an explicit agent tool registry with argument and response schemas. Reuse domain services, validation, authorization and audit from MCP where appropriate, but do not expose the MCP transport or every MCP tool/result wholesale. The current `internal/mcpserver` wrapper is token-bound; `ai:use` must remain people-only. The in-app agent acts for a user, not a manufactured MCP principal.

Allowed categories: sanitized resource/schema discovery; aggregate counts/reports; unrendered template and draft content; brand voice; validators; creating/revising this run's drafts; creating proposals. Default-deny every unclassified tool. Read-looking tools also need classification: results, previews, free-text errors, report links, webhook URLs and signed URLs can disclose personal data or perform work.

Denied categories: contact lookup/list/export, per-person events/replies/orders, rendered recipient previews, transactional messages, code exports, arbitrary SQL, arbitrary HTTP/browser/shell, sending/test sending, publish/schedule/approve, subscription or suppression changes, settings, routing, providers and deliverability tools. Models receive neither credentials nor callable third-party tools. Do not fetch arbitrary URLs proposed in content or by the model.

Projection happens on the server before model serialization. Return named allowlisted fields, not a redacted arbitrary object. Resource IDs are run-scoped handles where practical; tenant and resource scope are checked on resolution. Aggregate reads exclude per-recipient dimensions and small identifiable slices. Do not expose arbitrary filtering as an oracle for an individual's membership.

Implementation policy for reports: people may select one completed email campaign at least 72 hours old, requiring current `reports:view`, `campaigns:view` and campaign scope. Only fixed UTC-day send dates and campaign-wide totals reach the model. Suppress sends below 100; release a metric only when it and its complement against sent are each at least 20, then round down to tens. Withheld is not zero. Never include names, content, links, rates, dimensions or filters. One snapshot per campaign is shared across runs and external grants without refreshing; after 30 days its source counts are purged and a tombstone prevents reconstruction. A copy selected into an open run follows that run's existing retention: closure plus seven days, bounded by 30 days from run creation. This does not shorten the run or discard recovery drafts early. These controls prevent refresh-based differencing within one campaign, not identification across overlapping campaigns. Disclose that remaining inference risk before sharing; controlled option 2 does not claim mathematical anonymity.

Free text can contain personal data too. User briefs, selected content, resource names, voice samples and validator messages pass the same privacy boundary: remove recipient tracking/signed links and refuse detected personal records, credentials or contact data rather than forward them. User-supplied schema literals used for segmentation stay in a server-side binding; model sees the field/operator and a safe symbolic value. Never extract value examples from contacts. Under controlled option 2, reviewed editorial prose may pass these filters, which can miss personal information. Confirmation never bypasses a rejection; ask the person to edit locally when a detector rejects content.

The server requires `sharingReviewed: true` for learning voice, one-shot drafts, run creation, `answer`/`revise`/`replan`/`propose`/`refresh`, and external source selection. This is the authenticated person's attestation, not automatic classification or consent from a person mentioned in the text. Build and Continue reuse previously selected context within the confirmed task; Keep/Accept do not grant more model access. Record `reviewed-editorial-v1` with the action before disclosure, including failed or cancelled provider calls, without copying the text into audit. The person's provider/account controls storage, retention and training; ReputeCast promises neither zero retention nor no training across all providers or external clients.

Treat external content and tool results as untrusted data, separated from instructions. Enforce prohibitions in tool registration, authz, typed arguments, output projection and domain write guards, not by a prompt alone. Model-produced HTML is sanitized and links validated; it cannot run script in the app. Test an injected draft asking to send, expose contacts or change settings and verify all three remain impossible.

### 3.6 Access, audit, retention and notifications

| Action | Required access |
|---|---|
| Plan, build, revise or continue own run | Current workspace membership, `ai:use`, resource read/edit permissions and resource scope for every step |
| Read own run | Current workspace membership and resource scope; losing `ai:use` removes model actions, not access to permitted recovery drafts |
| Inspect another member's run | `integrations:manage` in that workspace, with resource scope still enforced; does not impersonate or continue their run |
| Keep, Discard, Accept or Reject | Run access plus each affected resource's normal edit permission; integration management alone grants no content-write right |
| Configure Agent, pricing, caps or per-run ceilings | `integrations:manage` |
| Learn voice; save a voice revision | `ai:use` and read permission on selected campaigns; saving also requires the existing brand-kit edit permission (`templates:edit`) |
| Send, publish or approve after handoff | Existing screens and permissions; never available to an agent tool |

Use actual domain keys: `campaigns:view/edit`, `segments:edit` plus the audience read gate, `automations:view/edit`, and the website's existing post edit gate. Implementation maps these from the owning services rather than inventing `posts:edit`. API keys and external MCP agents still cannot hold `ai:use` or start paid in-app runs.

Audit run start, Build, pause/cancel/continue, budget changes, draft mutations, Keep/Discard and Accept/Reject. Keep ordinary domain events with provenance instead of double-logging a mutation. Audit entries and the 90-day AI activity log contain IDs, usage and outcomes, not prompts, raw output, diffs or secrets. Version history carries accepted content and user attribution under its normal retention.

Run expiry is 30 days after creation, including abandoned/open runs. Display the expiry in the run. Purge brief/plan/step content at expiry; purge model messages and sanitized tool results seven days after closure, or at expiry if earlier. Delete unkept staging drafts and expire unapplied proposals; keep ordinary kept resources. A proposal whose source content was purged cannot be accepted or reconstructed from the audit log. Backups follow the instance's backup retention, disclosed separately. No transcript content in application logs, analytics or notification email.

Notify only the initiator, according to preferences, when a run becomes Ready to review or needs their action. Link to the run without content in the email; deduplicate by run and status revision. Settings alerts for a rejected key or monthly cap continue to go to integration managers. The run list shows owner, status, model, updated time, usage and expiry, filtered by the above access rules; no public sharing.

### 3.7 Brand voice and prompt-to-workflow

Brand kit gains **Voice**: tone, form of address, sentence length, preferred words, avoided words and examples. A person may edit manually without AI. **Learn from sent campaigns** requires selecting at most ten readable campaigns; send their unrendered text only, with tokens intact and the privacy checks above. Present the learned result as a diff; Save makes a new voice revision. Learning never updates brand settings itself.

Add an independently controlled `brand_voice` feature. Rewrite, subject ideas, draft-from-brief and agent use the selected saved voice revision and show it in provenance. Changing the saved voice does not silently rewrite existing drafts. Voice without examples remains useful; no broad scrape of campaign history.

Campaign and automation generation may use the current structured-output adapters before durable tool calling ships. In these one-shot flows, Accept is what creates or updates a draft. The full run flow may create staging drafts after Build. Both share validators and review components; neither adds send authority.

### 3.8 HostRepute and exclusions

For reputation diagnosis, blocklists, placement or delivery causality, explain the boundary and link to HostRepute's assistant when connected, otherwise Deliverability. No run content is forwarded to HostRepute automatically.

Out of A0: runtime code, migrations, API routes, a production Agent UI, real model calls, new dependencies, OAuth implementation or remote deployment. Out of the first agent release: scheduled autonomous runs, general chat, arbitrary tools, SQL generation, contact-level personalization, image generation and autonomous external actions.

ChatGPT/Claude OAuth remains A5: separate consent and external MCP principals, never `ai:use` or workspace-model spend. Its authorization-server design, capability grants, expiry/revocation, client-registration security and real-client tests require their own brief/boards before implementation. A0 does not settle a library or claim an existing notification-only proposal is a durable approval object.

## 4. Example shared by the boards

All counts, costs and outcomes here are illustrative design fixtures, not new seeded data or measured model results. Workspace: Acme Newsletter; initiator Linh Nguyen (Owner); model Claude Sonnet 5.5; saved voice Acme, revision 3. Marcus Chen is an Admin; Sofia Rossi an Editor; Daniel Park a Content Writer; Aisha Bello a Viewer. Schema `plan` is a custom contact field, not an Earn subscription tier.

Brief: “Re-engage Team-plan readers who have not opened in 60 days. Draft three emails over ten days and use an existing discount pool.” Clarification: only consented Weekly Digest readers, no recorded human clicks in 60 days either, merchandising offer for Acme Merch (not a discount on a SaaS subscription). Open tracking is imperfect; the plan states that limitation.

The model receives symbolic predicates and a permitted aggregate, not contact values. Sample count: 214 matching of 48,213 contacts. Re-check consent, suppression, audience membership and pool capacity when the human eventually publishes/sends. Existing pool Welcome 15% (`welcome15`), Shopify Acme Merch, 2,540 available codes; no codes are generated, reserved or exposed. Use `{{ coupons.welcome15.code }}` and `{{ coupons.welcome15.url }}` unchanged.

Bundle: staged segment “Team · quiet 60 days”, unpublished automation “Team re-engagement”, three embedded email drafts on days 0, 5 and 10. The flow's initial future trigger is entry into the kept segment; keeping does not enroll its existing 214 members. Enrollment/backfill is a later human decision in the publish flow. If the plan intended backfill, disclose that handoff explicitly rather than presenting the count as guaranteed sends.

A proposed fourth bonus post is optional and can be Discarded independently. A separate proposal shortens the subject of existing draft “Weekly Digest #143”; it does not belong to the automation bundle. One validation failure removes no content: the second email's missing coupon URL is restored on repair, then checked again. At review, sample usage is 18 of 40 calls, 84,200 of 300,000 tokens and US$0.42 of US$2.00; these are illustrative, not a quote at real provider prices.

## 5. Screens, states and interactions

| Board | Content and behavior |
|---|---|
| `AgentRun` | Task header, brief/clarification, accepted plan preview, cost and privacy disclosure, Build; step log and repair; scoped conversation; run list. No fabricated chain of thought. |
| `AgentReview` | Dependency bundle with email previews and checks, Keep/Discard; optional post; existing-draft subject diff, Accept/Reject; revision conflict; remaining decisions; human-editor handoff. |
| `AgentStates` | Selectable empty/loading/queue states; no model; Tools unavailable; Agent off; run budget; monthly cap; provider error/rate limit/key rejected; refusal; validation exhausted; revoked permission/scope; cancellation; offline reconnect; concurrent revision; expiry; reputation boundary; ready notification. |
| `AgentSettings` | Agent switch, capability badge, three per-run limits, monthly cap retained, member vs manager, content/access/retention disclosure. |
| `AgentVoice` | Current voice, manual editing, selected training campaigns, proposed learned revision and Save/Reject. |
| `AgentEntry` | Entry-point additions to Campaigns, composer, automation editor and command palette, with provenance and consent copy. |

The three core boards each have 1440, 834 and 390 variants. Extension boards reflow at the same widths. A0 appends linked extension frames to `AISettings`, `BrandKit`, `AIComposer`, `AutomationFlow` and `Campaigns` instead of redesigning their existing states. They are design-only, not routes that a production QA sweep should visit.

On tablet the sidebar becomes a rail; on narrow screens the same content stacks and the shell uses bottom navigation. Nothing critical depends on hover. Controls are at least 44 px on phones, focused controls remain visible, status uses text as well as color, dynamic results are announced, and diff changes carry Added/Removed labels. Review decisions do not disappear behind a floating composer. Match the app's light/dark tokens; phone boards demonstrate narrow Chromium layouts, not real-device testing.

## 6. Delivery and acceptance gates

| Milestone | Deliverable | Required evidence before it is called implemented |
|---|---|---|
| A0 (this change) | Brief, compatibility amendments, boards, canvas/inventory/coverage/QA mapping | All seven decisions traceable; responsive rendered boards; exercised Keep/Discard, Accept/Reject and failure states; no production feature claims |
| A1 | Versioned brand voice | Selected campaigns only; privacy boundary; manual/learned revision acceptance; current permission and audit checks |
| A2 | Campaign and automation from a brief | MJML/Liquid/graph/subject validation; failed output never applied; token/link preservation; one-shot acceptance |
| A3 | Typed tool adapters and durable runtime | Five provider contracts; denied tools; scoped projections; crash/retry/idempotency; cancellation fencing; permission revocation; simultaneous-budget reservations; retention |
| A4 | Production run/review UX | Real Postgres + recorded/fake model E2E; dependency-safe Keep/Discard; stale revision rejected; no send/publish/enrollment; accessibility, five locales, responsive light/dark |
| A5 | OAuth external MCP connector | Separate security brief; consent/scopes/revocation; durable proposals; supported-client interoperability; no in-app model billing |

A1 and A3 can proceed independently after A0. A2 consumes saved voice; A4 needs A2 and A3. A5 is separate from the in-app runtime. Assign repository milestone/migration numbers only when implementation starts; do not reserve M70 or overwrite existing roadmap work.

Regression fixtures must distinguish wrong from right: two users racing the last monthly dollar; worker death after a write but before acknowledgment; permission lost while a model responds; a kept email edited by a person before Discard all; an active segment with a running flow; contact data hidden in a title/error/link; unknown provider usage; a plan changed after Build; a proposal accepted twice or against a new revision. Observe resource counts/revisions, audit and model request payloads, not just HTTP success.

## 7. Conflicts resolved and implementation decisions left open

- Whole-campaign generation and task conversation now have this explicit exception; general chat and all sending prohibitions remain.
- New staging drafts may be written after Build; a pre-existing resource still requires human acceptance. Staging is essential for segments because “draft” is not currently a domain state everywhere.
- Run content has bounded storage, distinct from the existing content-free 90-day activity log. Integration management is not unrestricted access to every referenced resource.
- Continue preserves cumulative accounting; the earlier sketch of “Continue (≈ $x)” cannot bypass the accepted ceilings.
- Domain reuse is required; in-memory MCP transport is an option to evaluate, not a product requirement. Sanitized agent tools are not the entire MCP catalog.
- Physical staging tables, retention job implementation, API schemas, exact safe aggregate thresholds and provider tool-capability probes are A3 decisions. They must satisfy these contracts and real domain behavior; A0 introduces none of them.

## 8. Sources

Behavior: `reputecast-design-addon-create.md` §§1, 3.4–3.9; `reputecast-design-addon-team-access.md`; `internal/ai`, `internal/ai/llm/llm.go`, `internal/mcpserver`, `internal/authz/permissions.go`, `internal/audit`, `internal/automations/graph.go` and `publish.go`. Visuals: `design/design-system/README.md`, `tokens.json`, `AISettings`, `AIComposer`, `AISegmentReport`, `BrandKit`, `Campaigns`, `AutomationFlow`. Working patterns: `docs/dev/CONVENTIONS.md`, `docs/DESIGN_INVENTORY.md`, `web/scripts/qa-map.ts`.
