# AI models

Settings › Integrations › **AI models** connects the model accounts a
workspace's AI features use, picks a model for each feature, and holds the
assistant's switch, its limits and personal keys. Changing anything here
needs `integrations:manage`, and only people can do it (API keys and MCP
agents can't). People with `integrations:view` see the page read-only.

- [Model accounts and providers](#model-accounts-and-providers)
- [Keys are write-only](#keys-are-write-only)
- [The tool check](#the-tool-check)
- [Default model for each feature](#default-model-for-each-feature)
- [The assistant: switch and limits](#the-assistant-switch-and-limits)
- [Monthly cap and reservations](#monthly-cap-and-reservations)
- [Personal keys](#personal-keys)
- [What leaves the instance](#what-leaves-the-instance)
- [What managers see](#what-managers-see)

## Model accounts and providers

**Model accounts** lists the workspace's accounts. Each account has one key
(or none, for a server on your own network) and one or more models.
**Add account** opens **Add a model account**:

| Provider | What it needs |
|---|---|
| Anthropic | An API key. |
| OpenAI | An API key. |
| Google Gemini | An API key. |
| Azure OpenAI | The endpoint, the deployment, an API key and the API version (`2024-10-21` when left empty). |
| OpenAI-compatible | A **Service**: **Your own server (Ollama, vLLM, LM Studio)** with the server's address ending in `/v1` and an optional key, or one of the hosted presets below. |

The hosted presets are OpenAI-compatible services with their address filled
in. They need their API key. Their models come from the service's own model
list, and ReputeCast has no prices for them: enter the prices from the
service's pricing page.

| Preset | Address |
|---|---|
| OpenRouter | `https://openrouter.ai/api/v1` |
| Mistral | `https://api.mistral.ai/v1` |
| Groq | `https://api.groq.com/openai/v1` |
| DeepSeek | `https://api.deepseek.com/v1` |
| xAI | `https://api.x.ai/v1` |
| Together AI | `https://api.together.xyz/v1` |
| Fireworks AI | `https://api.fireworks.ai/inference/v1` |

AWS Bedrock isn't supported.

Adding an account runs **Test connection** first: a fixed prompt with no
workspace content, which counts toward the monthly cap. **Connect** adds the
account once the test passed. Its model becomes the default of every feature
that has none yet, so the workspace's first model is every feature's
default, **Assistant** and **Agent runs** included. Those two still need its
[tool check](#the-tool-check) to pass before they use it. A later account's
model is used only where you pick it.

**Manage** on an account shows its models, each with its prices, its tool
check and the features that use it. From there: **Add model** (another model
on the same key), **Prices**, **Run tool check**, **Remove** (a model no
feature uses, and not the account's only one), **Replace** or **Add a key**,
**Test connection** and **Remove account**. Removing an account deletes its
key and models; features that use them need another model first. The AI
activity log keeps their requests.

Prices are per million input and output tokens. ReputeCast's price list (1
October 2026) fills them in for the main Anthropic, OpenAI and Gemini
models; change them when your contract differs. A model with no price set
can't take requests ("Cost unknown. An integration manager must configure
both model prices before another request can be sent."), so enter prices
for presets and your own servers too, `0` if they cost nothing. A model
priced at `0` is free as far as the dollar limits go: use a token cap or
the thread's token limit to bound it.

A server on a private address (an Ollama box on the office network) needs
`providers.allow_private_networks` on the instance
([configuration](../deploy/configuration.md#settings)). The account then
reads **No key · host (private network allowed by the instance)**.

## Keys are write-only

- A key is never shown after you save it. The API only says whether one is set,
  when and by whom ("Workspace key set Nov 2 by Linh Nguyen"); the audit log
  records that a key was added or replaced, never the key.
- Keys are encrypted at rest with the instance's secret key, bound to their
  account. Back that key up with the database
  ([backups](../deploy/README.md#backups)).
- A saved key is only sent to the address it was saved with. A new server
  address needs the key pasted again.
- **Replace** tests the new key before saving it.
- When the provider rejects a key, the account says so, its models stop, and
  the members who manage integrations are told (for a personal key, its
  owner). Other accounts keep working.

## The tool check

The assistant and the agent call tools, so they only use a model that passed
the tool check. **Run tool check** sends a fixed request that asks the model
to call one harmless tool exactly once. It has no workspace content and
counts toward the monthly cap (for a personal key, its owner's cap).

| Shown | Meaning |
|---|---|
| **Tool check passed** (date) | The assistant and the agent can use it. |
| **Failed the tool check** (date) | They can't. One-shot suggestions still work. |
| **Tool check not run** | They can't until it passes. |

The result belongs to the model with its key, address and connection.
Replacing the key or reconnecting the account resets it to **Tool check not
run**: run it again, or threads on that model stop with **This thread's
model was removed**. Prices, names and feature switches don't reset it.

## Default model for each feature

**Default model for each feature** has one row per feature: **Assistant**,
**Agent runs**, Subject ideas, Rewrite, shorten and tone, Translate a
language version, Alt text, Accessibility fixes, Describe the segment,
Report summaries, Brand voice learning and Draft from a brief.

A model can be a feature's default when its account is connected and its key
works. Alt text also needs a model that reads images; **Assistant** and
**Agent runs** need a passed tool check. The menu says why a model can't be
picked ("failed the tool check", "doesn't read images", "key rejected"). A
personal key's model is never a workspace default.

Changing a default affects new requests. An assistant thread keeps the model
it started with until its owner picks another; an agent run keeps the model
it began with. A one-shot request prepared with the previous model isn't
sent.

## The assistant: switch and limits

The **Assistant** card:

| Row | Default | Notes |
|---|---|---|
| **Enable the assistant** | Off | Off: nobody can send a new message, and the panel shows **The assistant is off**. Threads are kept, still expire on schedule, and can't be opened until it's on again. |
| **Limit per thread** | 40 tool calls · 300,000 tokens · $2.00 | Each thread's cumulative ceiling. Up to 200 tool calls, 2,000,000 tokens and US$100. |
| **At once** | 1 per thread · 2 per person · 10 per workspace | Answers written at the same time, apart from agent runs. **Answers at once per person**: 1 to 10. **Answers at once in the workspace**: 1 to 100. |
| **Content kept** | 30 days after the last message | Fixed. |

**Edit limits** changes the limits. They apply to threads started afterwards:
a thread keeps the limits it started with.

The card's **What leaves the instance** note links to
[MCP servers](mcp-servers.md) and their approved tools.

The assistant also needs a model for **Assistant** in the defaults (or a
member's own pick) that passed the tool check, and members need
`ai:use`. Members who lack something see a state card that names it
([state cards](assistant.md#state-cards)).

## Monthly cap and reservations

The workspace's **Monthly cap** (on the usage card) is in dollars (US$1 to
US$100,000), in tokens, or none. Every model request of the workspace counts
toward it, the assistant's included; personal keys don't. Members who manage
integrations are told at 80% and at the cap. At the cap, workspace models
stop until the month resets or the cap is raised; threads show **Monthly AI
cap reached**.

Each request is reserved at its largest possible cost before it is sent, and
settled with the token counts the provider reports. A request the remaining
budget can't cover isn't sent. When a provider reports no usage (a stopped
answer, a lost connection), the request stays reserved at its maximum and
shows as cost unknown. **Reserved requests** lists them; **Reconcile usage**
records the token counts you checked in the provider's records, from ten
minutes after the request started. It never repeats a request. A personal
key's owner can reconcile their own holds through the API
(`reconcile-ai-personal-budget`); the page doesn't list them yet.

## Personal keys

A member with `ai:use` may add their own model account when managers allow
it. Only that member's requests use it.

| Row | Default | Notes |
|---|---|---|
| **Allow personal keys** | Off | Off: personal keys are kept, and their requests are refused until they are allowed again. |
| **Personal monthly cap** | $10.00 per person | **Edit cap**: US$1 to US$100,000. Each person's requests on their own keys stop at it until the month resets. Personal use never counts against the workspace's cap. |
| **Managers see** | Usage and cost, never content | What the card tells members. |

A member adds a key under **Your personal key** › **Add a personal key**
(**Test and add** tests it with a fixed prompt, in their personal cap), runs
the tool check on it, and picks it per feature under **Your model for each
feature** ("For your own requests. Everyone else keeps the workspace's
models."). The same providers and presets are available.

**People with personal keys** lists, for managers, each member's personal
accounts by provider and model and this month's cost and requests. It never
shows a key or what they asked.

Offboarding:

- **Remove …'s … key** deletes a member's key and its models, and their
  picks of those models. The AI activity log keeps their requests.
- When a member leaves the workspace, is removed from it or is deactivated,
  their personal keys there are deleted. Deactivating a person in Instance
  admin deletes their personal keys in every workspace.

## What leaves the instance

To the thread's model provider, with each request:

- ReputeCast's fixed instruction and the tool definitions, including the
  approved MCP tools' names, descriptions (at most 1,000 characters each)
  and schemas;
- the person's messages, the model's earlier answers and tool results, or a
  summary of earlier messages;
- the key of the screen the thread was opened on (such as
  `campaign.report`), never what the screen shows;
- the kind and title of each attached resource;
- what the tools return: ReputeCast's help articles and screen descriptions,
  an attached report's fixed totals (rounded down to tens, small numbers
  withheld), an attached draft's subject, preheader and visible text (links,
  images and Liquid tokens as placeholders), and approved MCP tools'
  results (text only, at most 32 KB).

**Subject ideas** and **Rewrite** send what those features send, to their own
feature's model. The connection test and the tool check send fixed text with
no workspace content.

To an MCP server: the arguments of its own tools' calls, and its credential
(the shared header, or the member's own OAuth token). See
[MCP servers](mcp-servers.md#what-servers-receive).

Never: contacts, recipient lists or exports (the assistant has no tool for
them), anything a person didn't attach, other members' threads, or
ReputeCast's keys. The model account's own terms decide how the provider
stores what it receives.

## What managers see

- **Usage and cost.** People with `integrations:view` see the workspace's
  usage by feature and by member, and the AI activity log (90 days): each
  request's member, feature, tokens, cost and outcome, never its content.
- **No thread content.** Nobody but its owner can read a thread, managers
  included.
- **The audit log** records changes to accounts, models, prices, defaults,
  the assistant's settings and personal keys, and, for threads, that one was
  started, renamed or deleted, what was attached (kind and ID) and which
  calls were allowed or denied. Never titles, messages, arguments or keys.
