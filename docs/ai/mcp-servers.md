# MCP servers

The assistant can call tools of remote MCP servers that a manager connected
for the workspace: an issue tracker, a wiki, a knowledge base. Each tool
waits for a manager's review. Tools that only read run once approved; tools
that write ask the person every time, with the exact arguments. Servers
never receive ReputeCast's keys, and what they return is treated as
untrusted data.

**Interoperability with real public MCP servers hasn't been verified yet.**
ReputeCast's tests use the MCP Go SDK's own server handlers and a fake
authorization server on loopback, including the discovery variants below
(a 401's challenge, the root metadata document, OpenID Connect discovery)
and revocation. No public MCP server or public authorization server has
been tried. Read
[what ReputeCast expects from an OAuth server](#what-reputecast-expects-from-an-oauth-server)
before you add one.

This page is about the servers the assistant calls. ReputeCast's own MCP
server, which outside agents call, is in Settings › Keys & MCP.

- [Where and who](#where-and-who)
- [Adding a server](#adding-a-server)
- [Sign-in](#sign-in)
- [Tool review](#tool-review)
- [When a tool changes](#when-a-tool-changes)
- [Who can use a server](#who-can-use-a-server)
- [Using servers in a thread](#using-servers-in-a-thread)
- [Untrusted results](#untrusted-results)
- [What servers receive](#what-servers-receive)
- [Members' own sign-ins](#members-own-sign-ins)

## Where and who

Settings › Integrations › **MCP servers**
(`/w/<workspace>/settings/integrations/mcp-servers`), linked from the
integrations catalog's AI group and from AI models.

- Adding, editing, syncing and removing servers, and reviewing tools, need
  `integrations:manage`.
- People with `integrations:view` see the page read-only: the servers, their
  tools and how each is allowed. With `ai:use` they can connect their own
  account to an OAuth server.
- Using a server's tools in a thread needs `ai:use` and the server allowed
  for the person's role.

## Adding a server

**Add server** opens **Add an MCP server**:

| Field | Notes |
|---|---|
| **Name** | How threads and tool rows name it ("Linear · search_issues"). Its first letters also prefix its tools' names for the model: `mcp__linear__search_issues`. |
| **Server address** | The server's MCP endpoint: Streamable HTTP, or HTTP with SSE. ReputeCast tries Streamable HTTP and falls back to HTTP with SSE when the server doesn't take it. Use the exact address: ReputeCast doesn't follow redirects. |
| **Sign-in** | **OAuth, each member signs in**, **Shared header secret** or **None**. See [sign-in](#sign-in). |
| **Who can use it** | **Everyone with “Use the AI assistant”** or **Owners and Admins**. |

Only remote servers can be added. Local process servers (stdio) aren't
supported.

Every request ReputeCast makes to a server or its authorization server goes
through the SSRF guard, which checks the address at every connection.
Servers on private or loopback addresses need the instance setting
`providers.allow_private_networks`
([configuration](../deploy/configuration.md#settings)); link-local and cloud
metadata addresses are always refused. ReputeCast accepts `http://`
addresses, but then the header secret or token travels unencrypted: use
`https://`.

**Add and sync tools** adds the server. A server with a shared header or
none is synced at once. An OAuth server sends you to its sign-in first,
because a sync lists the tools as you; back on the page, it syncs.

After adding, **Edit** changes the name, who can use it, the header secret's
value (**Replace**), the OAuth client, and **Offer its tools to threads**
(off: threads stop offering its tools; nothing is removed). The address, the
sign-in kind and the header's name can't change: remove the server and add
it again. **Remove server** removes its tools and every member's sign-in to
it (their refresh tokens are [revoked](#revoking-removed-sign-ins) where
the authorization server allows it); threads stop offering its tools, and
a call waiting for confirmation can no longer be allowed, only denied.

## Sign-in

| Kind | Who the server sees | Notes |
|---|---|---|
| **None** | Anonymous requests | For servers that need no credential. |
| **Shared header secret** | The workspace, as one identity | A header name (`Authorization` by default) and value. The value is write-only: "Write-only. ReputeCast never shows it again and only sends it to this server." Everyone allowed to use the server acts with that one credential. |
| **OAuth, each member signs in** | Each member, as themselves | Each member connects their own account. The server's own permissions then apply per person. |

Header secrets, client secrets and tokens are encrypted at rest with the
instance's secret key and never returned by the API.

### The OAuth redirect address

The authorization server sends people back to:

```
<public URL>/api/v1/ai/mcp/oauth/callback
```

for example `https://news.example.com/api/v1/ai/mcp/oauth/callback`. The
public URL is the one saved in Instance admin › Instance & updates, or
`REPUTECAST_PUBLIC_URL`. It's the same address for every server and
workspace.

- When ReputeCast registers itself with the authorization server (dynamic
  client registration), it registers this address. You don't enter
  anything.
- When the server needs a client you registered, open **Use a client
  registered with the server**, enter the **Client ID** and, if it has one,
  the **Client secret**, and register the address above as the client's
  redirect address. After the server is added, **Edit** shows it as
  "Redirect address to register: …".

The MCP authorization specification requires redirect addresses to use
https (or localhost), and authorization servers may refuse others: use an
`https://` public URL for OAuth servers. The people signing in must reach
it from their browser.

If the public URL changes, the redirect address changes with it. Register
the new one for a client you entered. A client ReputeCast registered keeps
the old address: clear its **Client ID** in **Edit**, and the next sign-in
registers a new client. Members' existing sign-ins belong to the old
client; expect them to sign in again.

### What ReputeCast expects from an OAuth server

ReputeCast follows the MCP authorization flow with PKCE (S256), the resource
indicator, and a state bound to the person's session that works once and
for ten minutes. Discovery runs at the first sign-in to the server (and
again after its **Client ID** is cleared):

1. It sends the server an MCP `initialize` request without credentials. If
   the answer is a 401, it reads `resource_metadata` and `scope` from its
   `WWW-Authenticate` Bearer challenge.
2. It reads the protected resource metadata from, in order: the address
   the 401 named; `https://<host>/.well-known/oauth-protected-resource<path>`,
   built from the server address; and the root
   `https://<host>/.well-known/oauth-protected-resource`. A document from
   the first two must name the server address as its `resource`. The root
   document must name the server's origin (`https://<host>`), and that
   origin is then the resource ReputeCast asks tokens for. A document that
   names any other resource is refused.
3. It takes the first authorization server listed and reads its metadata:
   OAuth (RFC 8414) first, then OpenID Connect discovery, with the issuer's
   path inserted after `/.well-known/` and, for OpenID Connect, also
   appended. The metadata must list `S256` in
   `code_challenge_methods_supported`.
4. Without a client you entered, the authorization server needs a
   `registration_endpoint`. Client ID metadata documents aren't supported.
5. It asks for the scope the 401 named, or else every scope in the
   resource's `scopes_supported`.
6. Client authentication is none, or HTTP basic with the client secret.
   Other methods (`client_secret_post`, `private_key_jwt`) aren't supported.
7. Metadata addresses must be https (plain http only on loopback).
8. It keeps the authorization server's `revocation_endpoint`, if it has
   one, to [revoke removed sign-ins](#revoking-removed-sign-ins).

When any of these fails, the sign-in says "…'s sign-in couldn't be set up:
its authorization server didn't answer as expected. Check the address or
enter a registered client."

## Tool review

**Show tools** lists a server's tools. **Sync tools** lists them again from
the server (up to 200 per server). A new tool is **Waiting for review** and
isn't offered to any thread.

**Review** (**View** for people who can't manage integrations) shows the
tool's description, `inputSchema` and annotations as plain text, as the
server sent them (a description is kept up to 4,000 characters; the model
reads its first 1,000). Read them as untrusted: they are the server's own
words, and they are what the model will read. Annotations such as
`readOnlyHint` are the server's hints, not evidence.

| Kind | State | What happens |
|---|---|---|
| **Read-only** · runs after approval | **Approved** | The model may call it; it runs without asking. |
| **Writes** · asks the person every time | **Asks every time** | Each call waits for the person's **Allow once** or **Deny** ([confirmations](assistant.md#confirmations)). |
| **Blocked** · never offered to the model | **Blocked** | Never offered. |

Mark a tool **Writes** when it changes anything outside ReputeCast, even if
it looks harmless. A read-only tool's arguments still leave the instance:
the model writes them, and they can contain what a person attached.

Other states: **Changed · off** ([below](#when-a-tool-changes)) and **Gone
from the server** (the server's last listing didn't include it; it isn't
offered).

## When a tool changes

"Approving pins this description and schema. Any change turns the tool
off." A decision is saved with the hash of the tool's description, input
schema and annotations as you reviewed them. If the tool changed on the
server while you reviewed it, the decision isn't saved and the page shows
the new version: "This tool changed on the server while you reviewed it."

ReputeCast compares a server's tools with what was decided at three
moments:

- **Before every call.** In the call's own session, ReputeCast lists the
  server's tools first. If the tool is no longer listed, or its
  description, schema or annotations differ from the version decided on,
  the call isn't sent. Whatever else the listing changed is recorded too.
- **Every hour**, for servers with **None** or a **Shared header secret**
  that are on and were synced before: the `mcp.servers_sync` job lists
  them as the system, at most 20 servers a run across the instance, the
  least recently updated first. A server that doesn't answer is tried
  again an hour later. OAuth servers are listed as a person, so only the
  check before each call and a manager's **Sync tools** cover them.
- **When a manager clicks Sync tools.**

When one of them finds that an approved or blocked tool changed:

- the tool is **Changed · off**: no thread offers it, and a call waiting for
  confirmation can't run;
- members who manage integrations get a notification ("Tools of … changed");
- threads whose call hit it show **A tool changed and is off**;
- **Review the change** shows **Before** and **After** for the description,
  the input schema and the annotations, with **Block the tool** and
  **Approve the new version**.

A tool the server no longer lists becomes **Gone from the server** and isn't
offered or called; no notification is sent for it. The hourly sync and the
check before a call write an audit entry, as the system, only when the
catalog changed.

## Who can use a server

**Who can use it**:

- **Everyone with “Use the AI assistant”**: everyone with `ai:use`.
- **Owners and Admins**: those two roles.
- Through the API, any list of role keys.

A server limited to roles says "Only … can use it" on its row. A person who
may use a server sees it in their threads' server picker. A server that is
off, or not for their role, isn't offered. Every call checks again that the
person may use the server.

## Using servers in a thread

Members pick a thread's servers in the panel ([picking MCP
servers](assistant.md#picking-mcp-servers)). The thread offers the approved
tools of those servers after ReputeCast's own, within 40 tools a request,
ordered by server name and then tool name; the rest aren't offered. On an
OAuth server, only members who signed in get its tools.

Each call opens its own session with the server, as the thread's owner. In
that session ReputeCast first lists the server's tools
([when a tool changes](#when-a-tool-changes)), then calls the tool; the two
together have 30 seconds. Tool rows say "Server · tool" and how long it
took, or what went wrong: sign-in needed, changed, blocked, unreachable, no
answer in 30 s, an answer over 32 KB, or an error the server reported.

## Untrusted results

What a server returns is data, never instructions:

- Text only. Images, audio and embedded resources become a note such as
  `[image removed]`; HTML becomes its text, without scripts and styles.
- At most 32 KB. A larger answer isn't given to the model ("… sent too
  much").
- Wrapped in `<untrusted_content source="Server · tool">` delimiters that the
  text can't close, and the model is told never to follow instructions
  inside them, call a tool or share data because they ask.
- A tool's description reaches the model prefixed "From the MCP server …,
  outside ReputeCast", at most 1,000 characters.

Delimiters lower the risk; they don't make a model immune to text that tries
to steer it. That is why tools that write ask every time and there is no
"always allow".

## What servers receive

- The arguments of their own tools' calls, as the model wrote them. For a
  writing tool, exactly the arguments the person allowed.
- Their credential: the shared header, or the member's own OAuth token.
- For OAuth servers: an MCP `initialize` request without credentials at
  discovery, and a removed sign-in's refresh token at the authorization
  server's revocation endpoint.

Nothing else: no ReputeCast keys, and nothing from the thread unless the
model puts it in a call's arguments.

ReputeCast refuses what servers ask of it: sampling (a server asking
ReputeCast's model to generate text) and elicitation (a server asking the
person for input). It doesn't list servers' resources or prompts.

Every call is in the audit log as `mcp_tool.called`, with the person, the
thread, the server, the outcome, the duration and a salted digest of the
arguments, never the arguments or the result. Server changes, syncs, tool
decisions, sign-ins and confirmations are audited too.

## Members' own sign-ins

On an OAuth server each member signs in for themselves:

- **Connect your account** on the server's row, or **Sign in to …** in a
  thread's server picker, opens the server's sign-in page. It comes back to
  Settings › Integrations › MCP servers with "Connected to …", "You
  cancelled the sign-in to …" or "… didn't complete the sign-in".
- The row then says **Connected as …**, with **Reconnect** and
  **Disconnect**.
- ReputeCast refreshes the token by itself shortly before it expires. When
  the refresh is refused, or the server answers 401 or 403, the sign-in is
  removed and threads show **Sign in to … again**.
- **Disconnect** deletes the tokens ReputeCast holds.

A member's sign-ins in a workspace are deleted when they leave it, are
removed from it or are deactivated. Deactivating a person in Instance admin
deletes their sign-ins in every workspace. Removing a server deletes every
member's sign-in to it.

### Revoking removed sign-ins

When the authorization server advertised a `revocation_endpoint` at
discovery, ReputeCast also revokes the refresh token of every sign-in it
removes: on **Disconnect**, when the member leaves, is removed or is
deactivated, when the server is removed, and when the server refused the
sign-in. The token is queued, still encrypted, and sent to the endpoint
(RFC 7009) as the client it was issued to:

- On **Disconnect** ReputeCast works through the queue at once (up to 10
  tokens, for up to 10 seconds), so the member's own token normally goes
  then.
- The `mcp.tokens_revoke` job sends what is left every minute, up to 100
  tokens a run.

It is best effort. Each token is tried once: a failed revocation is logged,
without the token, and not tried again, and the sign-in is gone either way.
Access tokens aren't revoked; they lapse when they expire. Without a
`revocation_endpoint`, nothing is revoked: use the server's own settings.
