# Using the assistant

The assistant answers questions about ReputeCast and about the campaigns and
reports you attach. It reads what you share, explains it and hands work to
the screen that owns it: a proposal you review in the composer, an agent
task you start yourself, a link. Its own tools never send, schedule,
publish, approve or change settings.

To use it you need the **Use the AI assistant** permission (`ai:use`;
Owners, Admins and Editors have it), and a manager has to switch the
assistant on and connect a model that passed its tool check
([AI models](models.md)). API keys and MCP agents can't use the assistant.

- [Opening it](#opening-it)
- [Threads](#threads)
- [Attaching context](#attaching-context)
- [Picking a model](#picking-a-model)
- [Picking MCP servers](#picking-mcp-servers)
- [Costs and limits](#costs-and-limits)
- [Stop](#stop)
- [Confirmations](#confirmations)
- [What it can and can't do](#what-it-can-and-cant-do)
- [Reputation questions go to HostRepute](#reputation-questions-go-to-hostrepute)
- [Privacy and retention](#privacy-and-retention)
- [State cards](#state-cards)

## Opening it

- The **Assistant** button in the top bar, next to the bell.
- ⌘J on a Mac, Ctrl J elsewhere. It opens and closes the panel, also while
  you type.
- ⌘K (Ctrl K): type your question and pick **Ask the assistant: …**. The
  question moves into the assistant's composer, unsent.
- **Ask the assistant** on a report's insights, a Send Readiness row and a
  composer warning. The panel opens with that report or draft suggested as
  context. Nothing is shared until you attach it.

The panel's code loads in the background once the page is idle, so it opens
at once. That happens only for people who can use the assistant, while it's
on, and not on a connection set to save data or on 2G.

Where the panel goes depends on the window's width:

| Width | Panel |
|---|---|
| Above 1100 px | Docked on the right at 400 px; the content narrows. Drag the edge, or focus **Resize the assistant** and use the arrow keys, for 360 to 640 px. When the content would be narrower than 720 px, the panel goes over it instead. On screens with review decisions (Review & send, approvals, proposals, an agent run's review) it docks at its narrowest instead, so it never covers the decision. |
| 768 to 1100 px | A sheet over the content. Escape closes it and gives focus back to what opened it. |
| 767 px and below | The whole screen. |

In the composer the panel goes over the content on a desktop, never docked.
The browser remembers the panel's width and the thread it shows.

**Open in full page** moves the thread to `/w/<workspace>/assistant/<thread>`,
with the thread list beside it on a desktop.

## Threads

A thread is one conversation, with its own model, limits and attached
context. **Threads** in the panel's header lists yours, the most recently
active first. **New** starts another. The first message names a thread;
**Rename** and **Delete** are in each thread's actions menu. **Search
threads** looks through the titles of your 500 most recently active threads.

An empty thread offers three suggestions for the screen you are on (on a
report: **Summarize the results**, **Compare with the last issue**, **Suggest
subjects for the next issue**). A suggestion goes into the composer; nothing
is sent, and nothing costs anything, until you send it.

Enter sends, Shift Enter adds a line, and nothing is sent while an input
method is still composing a word. A message can be up to 20,000 characters.

Each message starts a turn: **Queued…**, then **Answering…**, and then done,
stopped or failed. A turn runs on the server. Closing the panel, reloading
the page or losing the connection doesn't stop it; reopen the thread and the
answer is where it got to. A thread answers one message at a time.

## Attaching context

The assistant reads only what you attach. On a report, or on a draft
campaign's edit or review screen, the panel shows **Suggested: …** with
**Attach**. Attaching opens **Review before sharing**: it names the model's
provider, and **Share reviewed content** attaches it.

- A **report** shares its fixed totals as they were when you attached it.
  Counts are rounded down to tens and small numbers are withheld. No
  recipients, no links and no diagnosis of delivery.
- A **draft campaign** shares its subject, preheader and visible text. Links,
  images and Liquid tokens reach the model as placeholders, so their
  destinations stay on the instance and `{{ subscriber.first_name }}` stays
  as written.
- The resource's title goes with every request of the thread.

A thread holds up to 10 attached resources. Remove one with its chip's
**×**. If you can no longer read a resource, its chip says **No longer
available to you** and the assistant doesn't read it. Every read checks your
access again.

What the thread read stays tied to where it came from: each tool result to
its campaign, report or MCP server, and each answer to everything its
request carried, including the titles of what was attached. When you can
no longer read one of them (your role changed, the campaign was deleted,
the server is no longer allowed for you):

- the tool's result isn't sent to the model again, and its row shows that
  it is no longer available;
- an answer that used it, even only its title, is withheld: the thread
  shows it without its text, and the model reads only that an earlier
  answer was left out;
- a summary of earlier messages made from it, or from such an answer, is
  left out too;
- an answer still being written stops showing: within about 5 seconds its
  next pieces arrive without text, the panel hides what it had shown and
  says the answer is hidden, and after a reload it shows no text.

What you already read or copied, and what the model provider already
received, can't be taken back.

Answers from before anything was read or attached stay. Answers saved
before ReputeCast kept their sources count everything the thread had read
before them; answers cut short by **Stop** or by a failure also count what
is attached.

Only reports and email campaigns can be attached for now. Your messages go
to the model exactly as you write them, so don't paste personal data into
them.

## Picking a model

**Model: …** in the header opens **Choose a model**. Models are grouped as
**Workspace** and **Personal** (your own key, when managers allow personal
keys). Each shows what it can do (**Tools**, **Images**, **Streaming**), its
price per million tokens, and whose key it uses. A model the assistant can't
use says why: its account is disconnected, its key was rejected, it failed
or hasn't run the tool check, or personal keys are off.

A new thread uses your own pick for the assistant (Settings › Integrations ›
AI models › **Your model for each feature**), or else the workspace's.
Changing a thread's model applies to its next answers; earlier answers keep
the model that wrote them.

## Picking MCP servers

When managers have connected MCP servers you may use, the header has a
server picker: **No MCP servers** until you pick one, then **1 MCP
server** and so on. Its menu lists the servers:

- **Connected as you**, **Shared for the workspace** or **No sign-in
  needed**: their approved tools are available to the thread once you pick
  them.
- **Sign in to use its tools**: the server signs in each member. **Sign in
  to …** takes you to its sign-in page and back to Settings › Integrations ›
  MCP servers.

The thread may call the approved tools of the servers you pick. Tools that
only read run without asking; tools that write ask you every time
([confirmations](#confirmations)). A request offers at most 40 tools,
ReputeCast's own first; if the servers you picked have more approved tools,
the rest aren't offered. See [MCP servers](mcp-servers.md).

## Costs and limits

Every answer is one or more requests to the model, paid by the model's
account. The header shows the thread's cost against its limit, such as
**$0.04 of $2.00**; it turns to a warning at 80%. Hover it for the tool
calls used.

What costs a request:

- Each answer. An answer that calls tools takes several requests, at most 8
  for one message.
- **Subject ideas** and **Rewrite**, which make their own request with their
  feature's model.
- When a thread gets long (more than 60% of the model's context window),
  earlier messages are summarized in a request of its own. The thread shows
  **Earlier messages are summarized**.
- **Retry**, which sends a new request.

The fixed suggestions, the HostRepute card and attaching cost nothing.

| Limit | Default | What happens |
|---|---|---|
| Per thread | 40 tool calls, 300,000 tokens, US$2 | **This thread reached its limit**. Start a new thread. Managers change the limits for threads started afterwards. |
| The workspace's monthly cap | Set by a manager | **Monthly AI cap reached**. Workspace models pause until the month ends or the cap is raised. |
| Your personal cap | US$10 a month | Only for your own keys, which never count against the workspace's cap. |
| Answers at once | 1 per thread, 2 per person, 10 per workspace | A message waits as **Queued…** until another answer ends. |

Every request is reserved at its largest possible cost before it is sent,
and settled with the token counts the provider reports. When a provider
doesn't report them, the request counts at its maximum ("cost unknown for 1
request, counted at its maximum") until a manager checks the provider's
records.

## Stop

While the assistant answers, **Send** becomes **Stop**. After Stop, no
further model request or tool call starts for that message, and a call
waiting for your confirmation never runs. What was written so far stays. A
request that was already answering may still be charged by the provider;
the thread says **Stopped. An in-flight request may still be charged.**

## Confirmations

A tool that writes outside ReputeCast (an MCP server's tool a manager marked
**Writes**) never runs on its own. The answer stops at **Waiting for you ·
Server · tool**, with **Go to the request**, and shows a card:

- the server and the tool, and **Writes outside ReputeCast**;
- **Asked by** the model, with the tool's name for the model;
- the exact arguments that will be sent, as JSON, with **Copy**;
- **Deny** and **Allow once**.

**Allow once** sends that call, with those arguments, once. **Deny** tells
the model the call was refused, and nothing is sent. Either way the answer
goes on. While a call waits, the thread takes no new message: decide, or
Stop.

You don't have to keep the panel open:

- You get a notification, **The assistant is waiting for you**: "Server ·
  tool waits for you to allow or deny it. Nothing is sent until you do." It
  names the server and the tool, never the arguments. It is in-app; email
  for it is off unless you turn it on in your notification preferences
  ("AI assistant waiting for your confirmation"). Opening it opens the
  panel on that thread, or the thread's full page on screens without the
  panel.
- The **Assistant** button shows a dot while the notification is unread,
  and pressing it then opens the panel on that thread. So the dot is still
  there after a reload, and it also shows when the call started waiting
  after you closed the panel. If you turned the notification off, the dot
  only shows when you close the panel while the call waits.
- The notification goes away once you allow or deny the call, stop the
  answer, or delete the thread, or when the call expires.

A call nobody decides on expires after 24 hours. Nothing is sent: the card
keeps the arguments and says "This request expired after 24 hours without
a decision. Nothing was sent to …", and the answer ends with **The request
expired**. **Retry** sends your message again as a new message, with its
own cost.

If a manager blocked or changed the tool after the model asked, the card
says so and **Allow once** can't run it; **Deny** lets the assistant go on.
If the call was already closed (you stopped the answer, or decided
otherwise in another tab), the card says **This request was already
closed, so your decision wasn't sent.**

There is no "always allow". A thread can hold what you attached from the
workspace and what other tools brought in from outside at the same time.
Text from outside can steer a model, so a writing call could carry workspace
content somewhere it shouldn't go. Checking each call's exact arguments is
the safeguard.

## What it can and can't do

Its tools, as the thread's tool rows name them:

| Row | What it does | Needs |
|---|---|---|
| **Searched help** | Searches ReputeCast's built-in help articles | — |
| **Described the screen** | Says what a screen is for | — |
| **Suggested a link** | A card that opens a screen you may use. Nothing changes from there. | — |
| **Read attached** | Reads an attached campaign or report | Something attached |
| **Read report** | Reads an attached report's fixed totals | View reports, a report attached |
| **Proposed an edit** | A proposal for an attached draft; **Review the proposal** opens it in the composer, and nothing changes until you accept it | Edit campaigns, Draft from a brief on |
| **Drafted an agent task** | A brief for the AI agent; **Start an agent run** opens it, and you review it and confirm sharing before a run starts | Agent runs on |
| **Subject ideas**, **Rewrite** | The composer's suggestions for an attached draft | Edit campaigns, the feature on |
| **Server · tool** | An approved MCP tool | The server picked for the thread |

The model sees only the tools you may use now, and every call is checked
again as it runs.

It can't:

- send, schedule, publish or approve anything, or change settings: none of
  its own tools does. Anything else goes through MCP tools a manager
  approved, and a tool that writes runs only when you allow that call;
- read contacts, recipient lists or exports: there is no tool for them;
- read a campaign or report you didn't attach;
- load images: an image in an answer is a link that names its domain.

Answers are Markdown: lists, tables and code with **Copy**. Links show their
domain and open in a new tab without a referrer.

## Reputation questions go to HostRepute

A message that asks to diagnose sender reputation (a blocklisting, a domain's
or IP's reputation, spam traps, Postmaster Tools or SNDS, why mail lands in
spam) gets the **Ask HostRepute about reputation** card at once, without a
model request and at no cost. It offers **Open HostRepute** and **Open
Deliverability**. Nothing is sent to HostRepute.

Questions about setting things up, such as DKIM, SPF or DMARC, are the
assistant's to answer.

## Privacy and retention

- **Only you can read your threads.** Nobody else in the workspace can open
  them, managers included.
- People who can view integrations see each member's assistant usage and
  cost in the AI activity log, never content. The audit log records that you
  started, renamed or deleted a thread, what you attached (its kind and ID)
  and the calls you allowed or denied, never titles, messages or arguments.
- Thread content is encrypted at rest with the instance's secret key.
  Someone who holds both the database and that key can read it.
- Content is deleted 30 days after the thread's last activity. The thread
  then shows **This thread expired**; its usage stays, without content.
- **Delete** removes a thread's messages, attached resources and events at
  once. Its usage stays in the AI activity log, without content.
- When you leave a workspace or are removed from it, your threads there are
  deleted. Deactivation keeps them.
- What you send reaches the model's provider, and an MCP tool's arguments
  reach that server. Their own terms decide how long they keep it.
  Deleting a thread doesn't undo what a tool you allowed did elsewhere.
- The notification for a waiting call names only the server and the tool.
  It is deleted once the call is decided, stopped or expired, or the thread
  is deleted or expires.

## State cards

Each card says what happened and what to do. When the fix is a manager's,
its button names who can do it.

### When the assistant can't answer at all

| Card | Badge | When | What to do |
|---|---|---|---|
| **No model connected** | Not configured | The workspace has no model account. | A manager adds one: **Connect a model**. |
| **No model can answer here** | Not configured | No model can be used now, most often because none passed its tool check. | **Switch model**, or a manager runs the tool check. |
| **Your role can't use AI** | No access | Your role doesn't have `ai:use`. | Ask someone who manages the team. |
| **The assistant is off** | Off | A manager switched it off. Nothing new can be sent. Threads are kept, and until it's on again they can only be stopped or deleted, through the API: the panel doesn't open them meanwhile. | A manager switches it on: **Open AI settings**. |
| **This thread expired** | Expired | Its content was deleted 30 days after its last message. | **New thread**. |
| **Your access changed during this answer** | Stopped | Your role lost `ai:use` while it answered. What was written stays. | Ask someone who manages the team. |

### When an answer stops

| Card | Badge | When | What to do |
|---|---|---|---|
| **This thread reached its limit** | Paused | The next request wouldn't fit the thread's tool calls, tokens or cost. | **New thread**. Nothing is lost. |
| **Monthly AI cap reached** | Paused | The workspace used its monthly cap. | **Use … (personal)** if you have a personal key, or a manager raises the cap. |
| **Your personal cap is reached** | Paused | Your key used its monthly cap. | **Switch model** to a workspace model, or wait for next month. |
| **… rejected the key** | Key rejected | The provider refused the key. Its models stop; others still work. | For your own key, **Replace key** takes you to **Your personal key**; for a workspace key, a manager replaces it. Or **Switch model**. |
| **… is limiting requests** | Rate limited | The provider's rate limit for this key. | Wait, then **Retry**. |
| **… is unavailable** | Failed | The provider failed, didn't answer in time, or answered too much. The answer so far is kept. | **Retry** or **Switch model**. |
| **The model declined** | Refused | The model refused the request. | **Edit message**, or ask about something you attach. |
| **This thread's model was removed** | Failed | The thread's model was removed, alone or with its account. | **Switch model**, then send again. |
| **The model's account was disconnected** | Disconnected | The account of the thread's model has no connection. Nothing was sent. | **Open AI models** (yours to reconnect for your own key, a manager's otherwise), or **Switch model**. |
| **… hasn't passed its tool check** | Not checked | The thread's model hasn't passed the tool check, for example because its key was replaced. Nothing was sent. | **Run the tool check** (for your own model, or as a manager), then **Retry**; or **Switch model**. |
| **Personal keys are off** | Off | The thread uses your own key, and managers turned personal keys off. Nothing was sent. | **Switch model** to a workspace model, or ask a manager to turn personal keys on. |
| **The model isn't set up** | Not configured | There's no key for your own model, or no model account is connected. Nothing was sent. | **Add your key**, or **Switch model**; for the workspace, a manager uses **Connect a model**. |
| **The request expired** | Expired | A call waited 24 hours for your decision. Nothing was sent. | **Retry** sends your message again. |
| **The answer was interrupted** | Failed | The worker answering it stopped (a restart or deploy), or no worker took it within 30 minutes. Its cost stays unknown until a manager reconciles it. | **Retry**. If it keeps happening, tell whoever runs the instance. |
| **The answer didn't finish** | Failed | Any other reason, named in brackets. | **Retry**. |
| **The assistant can't read contacts** | Denied | The model tried to read contacts, recipients or exports. | **Ask about a segment** asks about segment sizes instead. |
| **Reconnecting** | Reconnecting | The connection to the server dropped. The answer goes on on the server. | Wait, or **Reconnect now**. |

Every **Retry** sends a new request with its own cost.

The cards about the thread's model say whose fix it is: yours when the
model is on your own key, a manager's when it's the workspace's. When a
message is refused before it starts, they show the server's own
explanation, which names the same.

Notes in the conversation, without a card: **Stopped. An in-flight request
may still be charged.**, **The answer stopped before it ended. What arrived
is kept.** and **The answer reached its length limit.**

### MCP servers

| Card | Badge | When | What to do |
|---|---|---|---|
| **Sign in to … again** | Sign-in needed | Your sign-in to the server expired or was never made. The assistant answers without that server. | **Sign in to …**. |
| **A tool changed and is off** | Needs review | The server changed a tool after it was approved. | A manager reviews it: **Review the change**. |
| **… is blocked** | Blocked | A manager blocked the tool. | Nothing; the assistant answers without it. |
| **… is unreachable** | Failed | The server didn't answer. | Ask again later, or ask a manager to check the server. |
| **… took too long** | Failed | The tool didn't answer within 30 seconds. | Ask again later. |
| **… sent too much** | Failed | The answer was over 32 KB, so the model didn't get it. | Ask a narrower question. |

A tool row can also say **You denied …. Nothing was sent to ….** or **… may
have run, but its result was lost. Check … before asking again.** The second
means the call was sent and the worker was lost before its result was saved;
ReputeCast never sends it again.
