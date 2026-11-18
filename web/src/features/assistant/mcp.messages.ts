import { defineMessages } from '@/i18n'

/**
 * The assistant panel's MCP parts (assistant brief §4, the AssistantPanel
 * and AssistantStates boards): the thread's server picker, the
 * confirmation card for a writing tool, its tool rows and states. Server,
 * tool and model names are arguments, never translated; so are error codes.
 */
export const mx = defineMessages('assistant.mcp', {
  // ── The thread's servers ────────────────────────────────────────────
  picker: '{count, plural, =0 {No MCP servers} one {# MCP server} other {# MCP servers}}',
  menuLabel: 'MCP servers for this thread',
  menuGroup: 'MCP servers',
  menuNote: 'The thread may call the tools of the servers you pick. Tools that write ask you every time.',
  connected: 'Connected as you',
  shared: 'Shared for the workspace',
  noSignIn: 'No sign-in needed',
  needsSignIn: 'Sign in to use its tools',
  signIn: 'Sign in to {server}',
  manage: 'Manage MCP servers',
  on: '{server} is on for this thread',
  off: '{server} is off for this thread',
  failed: 'The thread’s MCP servers weren’t changed',
  signInFailed: 'Couldn’t start the sign-in to {server}',

  // ── Confirmation ────────────────────────────────────────────────────
  badge: 'Writes outside ReputeCast',
  // server, tool: "Linear · create_issue".
  title: '{server} · {tool}',
  check: 'Check exactly what will be sent to {server}.',
  // model: the thread's model; name: the tool's name for the model (mcp__linear__create_issue).
  askedBy: 'Asked by {model} · {name}',
  argsLabel: 'Arguments for {server} · {tool}',
  note: 'This tool writes outside ReputeCast, so it asks every time. Nothing is sent until you allow it.',
  deny: 'Deny',
  allow: 'Allow once',
  closed: 'This request was already closed, so your decision wasn’t sent. The thread shows what happened.',
  confirmChanged: '{server} · {tool} changed after the model asked, so it can’t run. Deny it to let the assistant go on.',
  confirmBlocked: 'A manager blocked {server} · {tool} after the model asked, so it can’t run. Deny it to let the assistant go on.',
  decideFailed: 'Your decision wasn’t sent',
  announce: '{server} · {tool} waits for you to allow or deny it.',
  announceAllowed: 'Allowed once. Calling {server} · {tool}.',
  announceDenied: 'Denied. Nothing was sent to {server}.',
  waiting: 'Waiting for you · {server} · {tool}',
  goTo: 'Go to the request',

  // ── Tool rows ───────────────────────────────────────────────────────
  // label: "Linear · create_issue".
  rowDenied: 'You denied {label}. Nothing was sent to {server}.',
  rowUnknown: '{label} may have run, but its result was lost. Check {server} before asking again.',
  rowProblem: '{label} · {code, select, mcp_sign_in {sign-in needed} mcp_tool_changed {changed, off until reviewed} mcp_tool_blocked {blocked} mcp_unreachable {server unreachable} mcp_timeout {no answer in 30 s} mcp_too_large {answer over 32 KB} tool_error {the server reported an error} not_available {not available to you} forbidden {not allowed} tool_limit {over the thread’s tool-call limit} other {failed}}',

  // ── States ──────────────────────────────────────────────────────────
  signInTitle: 'Sign in to {server} again',
  signInBadge: 'Sign-in needed',
  signInBody: 'Your {server} connection expired or was never made. The assistant answers without {server} until you sign in again.',
  changedTitle: 'A tool changed and is off',
  changedBadge: 'Needs review',
  changedBody: '{server} changed the description of {tool} after it was approved. It stays off until a manager reviews the change.',
  reviewChange: 'Review the change',
  blockedTitle: '{server} · {tool} is blocked',
  blockedBadge: 'Blocked',
  blockedBody: 'A manager blocked this tool, so the assistant answers without it.',
  unreachableTitle: '{server} is unreachable',
  unreachableBody: '{server} didn’t answer. The assistant went on without it; ask again later, or ask a manager to check the server.',
  timeoutTitle: '{server} took too long',
  timeoutBody: '{tool} didn’t answer within 30 seconds, so the assistant went on without it.',
  tooLargeTitle: '{server} sent too much',
  tooLargeBody: '{tool}’s answer was over 32 KB, so the model didn’t get it. Ask a narrower question.',
  failedBadge: 'Failed',
})
