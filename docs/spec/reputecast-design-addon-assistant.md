# ReputeCast — Add-on spec: AI assistant

The AI assistant is a chat panel inside the app. It answers questions about ReputeCast, explains the resources a person attaches, and hands work to the existing flows: proposals, agent runs and the screens that own each action. It never sends, schedules, publishes, approves or changes settings. This is the B0 product and design brief, dated 16 November 2026; it does not claim that the assistant is implemented.

## 0. Status and use

The owner accepted decisions D1–D7 below on 16 November 2026. B0 delivers this brief and the boards; B1–B6 deliver the product in separate milestones. Where this brief and `reputecast-design-addon-agent.md` differ, this brief wins for the assistant only. Agent runs, one-shot suggestions and external MCP connections (A5) keep their own rules.

Use the existing Clearwater tokens, shell, permissions, audit, budget and privacy rules. Product copy is English and is localized into the app's five languages. Every feature stays free and self-hosted; models run on the workspace's or the member's own account.

## 1. The seven accepted decisions

1. **D1 · General assistant threads.** A person may open a thread that is not tied to one task. A thread has the same bounds as an agent run: cumulative ceilings, bounded retention and default-deny tools. This amends agent brief §1 decision 1 and §3.8 for the assistant only.
2. **D2 · Third-party MCP tools.** The model may call tools from MCP servers a manager connected. Read-only tools run after a manager approves them. Tools that write run only after the person confirms that exact call. This amends agent brief §3.5 ("models receive neither credentials nor callable third-party tools") for the assistant only. Credentials still never reach the model.
3. **D3 · Several models per workspace.** A workspace may connect several model accounts and pick a default model per feature. The current single connection becomes the first entry.
4. **D4 · Personal keys.** A member may add their own model key when an integration manager allows it. The switch is off by default. Personal usage has its own monthly cap and never counts against the workspace cap.
5. **D5 · Frontend dependencies.** Messages render with `react-markdown` and `remark-gfm`. Code is highlighted with `@lezer/highlight` and the CodeMirror languages already installed. Nothing else is added without its own note.
6. **D6 · Providers.** Hosted providers beyond the current five arrive as presets of the OpenAI-compatible kind. AWS Bedrock is deferred.
7. **D7 · Thread privacy.** Only the person who started a thread can read it. Integration managers see usage, cost and outcomes, never content. This differs from agent runs, which managers may inspect (agent brief §3.6). Threads keep content for 30 days and can be deleted at once.

## 2. Terms

- **Thread:** a person's conversation in one workspace, with its model, ceilings, usage ledger, attached context and expiry.
- **Turn:** one person message and the assistant's answer, including tool calls. A turn is a durable background job; closing the panel does not stop it.
- **Attached context:** a resource the person explicitly attached, such as a campaign, a report or a segment. The current screen is only suggested. Nothing is shared until the person attaches it and confirms sharing.
- **Tool row:** a muted line in the answer that says what was read or called and how long it took.
- **Confirmation card:** the request to run one tool call that writes outside ReputeCast. It shows the server, the tool and the full arguments.
- **Handoff card:** a link into an existing flow, such as Open in composer, Start an agent run or Review the proposal. It navigates only.
- **MCP server:** a remote server a manager connected for the workspace. Each of its tools is approved, blocked or waiting.

## 3. Product requirements

### 3.1 Entry and layout

- Entry points: the Assistant button in the top bar's tools, next to the bell; ⌘J or Ctrl J; ⌘K › Ask the assistant; Ask the assistant buttons on report cards, Send Readiness rows and composer warnings. A contextual button opens the panel with that resource suggested as context.
- Desktop above 1100 px: the panel docks on the right at 400 px and the content column narrows. The person can resize it from 360 to 640 px with a keyboard-operable separator. When the remaining content would be narrower than 720 px, the panel overlays instead. It never covers review decisions.
- Tablet from 768 to 1100 px: the panel is a sheet over the content with a scrim. Focus stays in the sheet; Escape closes it and returns focus to the opener.
- Phone at 767 px and below: the panel is full screen and the tab bar is hidden. The composer stays above the keyboard. Targets are at least 44 px.
- Open in full page moves the thread to `/w/{workspace}/assistant/{thread}`. The thread list has search by title, rename and delete.

### 3.2 Panel anatomy

From top to bottom: the header (thread title, model picker, thread cost, full page, history and close), the context chips, the messages and the composer.

- The **model picker** groups models by connection. Each model shows its capabilities (tools, images, streaming), its price per million tokens and whether its key belongs to the workspace or to the person. A model that can't be used says why.
- The **thread cost** reads like "$0.04 of $2.00" and turns to a warning at 80%.
- The **context chips** show what is attached. A suggested chip says "Suggested: October Product Update" with Attach. Attaching opens the existing sharing review before anything is sent.
- **Answers** render Markdown: paragraphs, lists, GFM tables, code with Copy. Liquid tokens stay literal. Remote images are never loaded; an image becomes a link with its domain. Links show their domain and open in a new tab without a referrer.
- **Tool rows** say what happened: "Read report · September Product Update · 0.3 s", "Searched help · DKIM", "Linear · search_issues · 1.2 s". They expand to the projected arguments and a short result.
- A **confirmation card** blocks only its own call. It offers Deny and Allow once. There is no "always allow" for a tool that writes once the thread holds workspace content and outside content.
- **Handoff cards** open the existing screens. Accepting a proposal uses the existing review components and edit permissions.
- The **composer** grows to 40% of the panel. Enter sends, Shift Enter adds a line, and nothing is sent while an input method is composing. `/` lists commands and the MCP prompts of enabled servers. While the assistant answers, Send becomes Stop.

### 3.3 Turns, stopping and reconnection

Visible turn states are Queued, Answering, Waiting for you (a confirmation), Stopped, Failed and Done. A turn keeps answering when the panel closes or the browser reconnects; reopening shows the same turn where it is. Stop prevents further model and tool calls; an in-flight provider request may still be charged, and the turn says so. A failed turn keeps what was already written and offers Retry, which is a new request with its own cost.

The empty thread greets the person by the screen they are on and offers three fixed suggestions for that screen, with no model call. On a report: Summarize the results, Compare with the last issue, Suggest subjects for the next issue.

### 3.4 Budget

Threads use the agent's accounting: a conservative reservation before each request, cumulative ceilings per thread, the workspace monthly cap, and "Cost unknown" when a provider gives no trustworthy usage. Defaults per thread are 40 tool calls, 300,000 tokens and US$2, the agent's defaults. Managers may change them. A personal key counts against the person's own monthly cap. Concurrency defaults: one answering turn per thread, two per person and ten per workspace, separate from agent runs.

### 3.5 Tools and privacy

The assistant shares the agent's tool registry and projections. The model only sees tools the person may use, and every call is authorized again when it runs.

| Tool | What it does | Boundary |
|---|---|---|
| Search help | Searches the instance's own documentation | No workspace data |
| Describe this screen | Says what the current screen is for and what it offers | No workspace data |
| Read attached | Reads an attached resource | The agent's projections; Liquid unexpanded; segment literals stay on the server |
| Read report snapshot | Reads a completed campaign's fixed totals | The agent's report rules (§3.5 of the agent brief) |
| Suggest a link | Returns a link card to a screen | The app only shows routes it knows |
| Propose an edit | Creates a durable proposal against an exact draft revision | Accept and Reject with the existing review and edit permission |
| Draft an agent task | Fills in an agent brief | The person starts the run and confirms sharing |
| Subject ideas, Rewrite | The existing one-shot suggestions | Their feature switches |

Denied: the agent brief's denied categories, unchanged. Questions about reputation, blocklists or placement get the HostRepute boundary card.

Brief text, attached content and voice samples pass `internal/editorial` and need `sharingReviewed: true`, as for agent runs. Outside content from MCP tools is untrusted data: it is size-capped, stripped of images and HTML, delimited with its source and never treated as instructions.

### 3.6 MCP servers

- Integration managers add remote servers (Streamable HTTP, or HTTP with SSE). Local process servers (stdio) are not supported.
- Authentication is none, a shared header secret, or OAuth per member. OAuth follows the MCP authorization specification of 25 November 2025 with PKCE, dynamic or pre-registered clients and the resource indicator. Tokens and secrets are sealed and write-only.
- Every request goes through the SSRF guard. Private addresses need the instance setting.
- Syncing lists the tools. A new tool waits for a manager, who reads its description and schema and marks it read-only, writes or blocked. Annotations from the server are hints, not evidence.
- When a tool's description, schema or annotations change after approval, the tool is turned off until a manager approves it again, and managers are told.
- A member chooses which enabled servers a thread may use. A request carries at most 40 tools; ReputeCast's own tools come first.
- Sampling requests from servers are refused. Elicitation is out of the first release.

### 3.7 Access, audit, retention and notifications

| Action | Required access |
|---|---|
| Use the assistant, read and delete own threads | Membership, `ai:use`, the assistant switch on, and each resource's read permission and scope |
| Add, change or remove workspace models, prices, caps and the personal-key switch | `integrations:manage` |
| See assistant usage and cost per person (no content) | `integrations:view` |
| Add a personal key | `ai:use` and the personal-key switch on |
| Add MCP servers, approve or block tools | `integrations:manage` |
| Connect one's own account to an OAuth MCP server | `ai:use` and the server allowed for the person |
| Accept or reject a proposal from a thread | The resource's normal edit permission |

No permission key, role or actor type is added. API keys and inbound MCP agents can't use the assistant. Audit records model and server changes, tool approvals, OAuth connections, thread deletion, confirmations and proposal decisions, with the initiating person as actor and `via: ai_assistant`. Audit and the 90-day AI activity log hold IDs, usage and outcomes, never content, arguments or secrets.

Thread content is kept 30 days from the last activity and sealed at rest; deleting a thread removes it at once. Stream events are kept 24 hours after their turn ends. No content goes into application logs, metrics, traces or notification email. The assistant sends no notifications of its own; a confirmation waiting in a closed panel shows a dot on the Assistant button.

## 4. Example shared by the boards

All numbers are illustrative fixtures, not seeded data or measured results. Workspace Acme Newsletter; Linh Nguyen (Owner) asks; Sofia Rossi is an Editor without `integrations:manage`. Models: Claude Sonnet 5.5 (workspace key, Anthropic), GPT-5 mini (workspace key, OpenAI), Llama 3.3 70B on the office Ollama server (no key), and Linh's personal Gemini 2.5 Flash. MCP servers: Linear (OAuth per member, `search_issues` read-only, `create_issue` writes), Notion (OAuth per member, two tools read-only), and an internal wiki whose `search` tool changed after approval.

Linh opens the October Product Update report, attaches it and the September Product Update, and asks why opens fell. The assistant reads both campaigns' fixed snapshots and their unrendered content. It answers that opens fell from about 41% to 38% of delivered while clicks held, and points to October's longer subject. It searches Linear for "tracking pixel" and finds one open issue. It then asks to create a Linear issue titled "Compare subject length before the November update", which waits for Linh's confirmation. The thread has used 6 of 40 tool calls, 21,400 of 300,000 tokens and $0.04 of $2.00.

## 5. Screens, states and interactions

| Board | Content and behavior |
|---|---|
| `AssistantPanel`, `AssistantPanelTablet`, `AssistantPanelPhone` | The report with the docked panel, sheet or full screen; suggested and attached context; the streamed answer with tool rows; the Linear confirmation; Stop; the thread list. |
| `AssistantStates`, `AssistantStatesTablet`, `AssistantStatesPhone` | Empty thread with suggestions; no model; no `ai:use`; assistant off; thread ceiling; monthly cap; key rejected; rate limited; provider down; refusal; reconnecting; tool denied; MCP sign-in needed; tool changed; thread expired; permission lost during a turn; reputation boundary. |
| `AssistantEntry` | The top-bar button with its waiting dot, ⌘J, ⌘K › Ask the assistant and the contextual buttons on a report card, a Send Readiness row and a composer warning. |
| `AIModels` | Settings › Integrations › AI models with several connections, per-feature defaults, prices, capability checks, the personal-key switch, and Linh's personal key. |
| `MCPServers` | Settings › Integrations › MCP servers: add a server, OAuth sign-in, tool approval with description and schema, a changed tool, member access and removal. |

The panel and states boards have 1440, 834 and 390 variants; the other boards reflow at the same widths. All boards have light and dark themes and work without the proprietary `support.js`. Board selector, theme and role buttons are prototype controls, not product chrome. They are design-only, not routes that the production QA sweep visits.

Nothing critical depends on hover. Focus is always visible. Streaming text is not read token by token: a separate polite live region says when an answer starts, when it ends with its first sentence, and when a confirmation waits. Status uses words and icons, not color alone.

## 6. Delivery and acceptance gates

| Milestone | Deliverable | Evidence before it is called implemented |
|---|---|---|
| B0 (this change) | Brief, agent-brief amendments, boards, canvas, inventory and QA map | D1–D7 traceable to boards; boards render at 1440, 834 and 390 in light and dark; confirmation, Stop and state switching exercised |
| B1 | Model registry, per-feature defaults, presets, capability checks, personal keys | Existing connection, usage and caps preserved by the migration; key sentinel absent from responses, logs, audit and events |
| B2 | Streaming in `internal/ai/llm` | Recorded SSE fixtures per provider: truncated stream, missing usage, tool arguments split across chunks, refusal mid-stream, idle timeout, size limit |
| B3 | Threads, turns on River, events and SSE, budgets, tools, retention, audit | Real Postgres: worker death mid-stream, Stop fencing, resume without gaps or duplicates, two turns racing the last dollar, permission lost mid-turn, split web and worker roles |
| B4 | The panel, states and settings screens | Fake-model browser runs; axe; no overflow at 320, 390, 834 and 1440; coarse pointer targets; input method composition; reduced motion; bundle budget; five locales |
| B5 | MCP servers, OAuth, tool approval and confirmation | Fake servers: benign, injection, changed tool, huge output, slow, expired sign-in; SSRF; two real public servers recorded with client, version and date; until then, interoperability is unverified |
| B6 | Hardening and release | Injection fixtures through every path; 200 concurrent streams with p95 delta under 100 ms; docs; roadmap |

B1 and B2 can proceed in parallel. B3 needs both. B4 can start against mocked responses once B3's API is fixed. B5 needs B3, and its screens need B4. Assign migration numbers when implementation starts.

## 7. Conflicts resolved

- General chat (agent brief §1 decision 1, §3.8 and `reputecast-design-addon-create.md`'s exclusion) is allowed for assistant threads with run-like bounds. Agent runs keep their task scope.
- Callable third-party tools (agent brief §3.5) are allowed for the assistant through approved MCP tools and confirmations. Agent runs still have no third-party tools, and models still never receive credentials.
- One model per workspace (`internal/ai/settings.go`) becomes several connections with per-feature defaults.
- Manager inspection of runs (agent brief §1 decision 6, §3.6) does not extend to assistant threads.

## 8. Sources

Behavior: `reputecast-design-addon-agent.md`, `reputecast-design-addon-agent-oauth.md`, `internal/ai`, `internal/ai/llm`, `internal/editorial/privacy.go`, `internal/mcpserver`, `internal/mcpconnect`, `internal/netguard`, `internal/server/middleware.go` and `internal/authz/permissions.go`. Visuals: `design/design-system/README.md`, `agent-a0.css`, `AISettings`, `AgentRun`, `AgentStates`, `MCPConnect`. Patterns: `docs/dev/CONVENTIONS.md`, `docs/DESIGN_INVENTORY.md` and `web/scripts/qa-map.ts`. External: [MCP authorization, 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization) and [MCP security best practices](https://modelcontextprotocol.io/specification/2025-11-25/basic/security_best_practices).
