# The AI assistant

The assistant is a chat panel inside ReputeCast. It answers questions about
ReputeCast, explains the campaigns and reports a person attaches, and hands
work to the screens that own it. Its own tools never send, schedule,
publish, approve or change settings; tools of MCP servers that write run
only when the person allows each call. Models run on the workspace's own
model accounts, or on a member's personal key when managers allow it.

The assistant is off in every workspace until a manager switches it on.
Everything here is free and part of the open-source edition.

| Page | For | What it covers |
|---|---|---|
| [Using the assistant](assistant.md) | Members | Opening it, threads, attaching context, models and MCP servers in a thread, costs, Stop, confirmations, what it can't do, privacy and every state card |
| [AI models](models.md) | Managers | Model accounts and providers, keys, the tool check, each feature's model, the assistant's switch and limits, personal keys, what leaves the instance |
| [MCP servers](mcp-servers.md) | Managers and members | Remote servers, sign-in, the OAuth redirect address, tool review and pinning, the checks for changed tools, who can use a server, untrusted results, members' own sign-ins and their revocation |
| [Configuration › The AI assistant](../deploy/configuration.md#the-ai-assistant) | Operators | The `ai_assistant` queue and `--role`, the event stream's database connection, private networks, the public URL for OAuth, timeouts, metrics |
| [Reverse proxy › Event streams](../deploy/reverse-proxy.md#event-streams) | Operators | Server-sent events behind a proxy: buffering, timeouts, HTTP/2 |
| [Performance › The AI assistant](../deploy/performance.md#the-ai-assistant) | Operators | 200 streams under load, the server's time to the first token, and the panel in the browser |

The product brief is
[`reputecast-design-addon-assistant.md`](../spec/reputecast-design-addon-assistant.md).
Where these pages and the brief differ, these pages describe what the code
does.
