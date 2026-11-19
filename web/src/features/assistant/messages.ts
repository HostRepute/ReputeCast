import { defineMessages } from '@/i18n'

/**
 * The AI assistant's panel, thread list and full page (assistant brief
 * §3.1–§3.5, the AssistantPanel, AssistantStates and AssistantEntry
 * boards). Model, provider, workspace, campaign and people's names are
 * arguments, never translated; so are tool names (search_issues) and error
 * codes.
 */
export const m = defineMessages('assistant', {
  // ── Header ──────────────────────────────────────────────────────────
  title: 'Assistant',
  threadsTitle: 'Threads',
  untitled: 'Untitled thread',
  history: 'Threads',
  fullPage: 'Open in full page',
  close: 'Close the assistant',
  back: 'Back to the thread',
  newThread: 'New',
  newThreadAction: 'New thread',
  resize: 'Resize the assistant',
  // The full-page route's browser tab: "Why opens fell · Assistant".
  documentTitle: '{title} · Assistant',

  // ── Model picker ────────────────────────────────────────────────────
  // model: the model's name, as the button shows it.
  modelButton: 'Model: {model}',
  modelMenu: 'Choose a model',
  groupWorkspace: 'Workspace',
  groupPersonal: 'Personal',
  keyWorkspace: 'workspace key',
  keyPersonal: 'your personal key',
  capTools: 'Tools',
  capImages: 'Images',
  capStreaming: 'Streaming',
  // input, output: prices per million tokens, formatted ("$2", "$0.25").
  price: '{input} in · {output} out per 1M tokens',
  priceUnknown: 'No price set',
  reasonDisconnected: 'Its account is disconnected. A manager reconnects it in Settings › Integrations › AI models.',
  reasonKeyRejected: '{provider} rejected its key. It works again once the key is replaced.',
  reasonToolsFailed: 'Failed the tool check. It still works for one-shot suggestions.',
  reasonToolsUntested: 'Its tool check hasn’t run. A manager runs it in Settings › Integrations › AI models.',
  reasonPersonalOff: 'Personal keys are off in this workspace.',
  modelChanged: 'Model changed to {model}. Earlier answers keep the model that wrote them.',
  noModelPicked: 'Choose a model',
  // used, limit: amounts of money ("$0.04 of $2.00").
  cost: '{used} of {limit}',
  // calls, limit: tool calls of the thread.
  costTitle: '{calls} of {limit} tool calls · this thread',
  // count: requests whose usage the provider never reported; held: what
  // they count at, their reserved maximum (included in used).
  costUnknown:
    '{used} of {limit} · {count, plural, one {cost unknown for # request, counted at its maximum ({held})} other {cost unknown for # requests, counted at their maximum ({held})}}',

  // ── Attached context ────────────────────────────────────────────────
  contextLabel: 'Attached context',
  // title: a campaign's or report's name.
  suggested: 'Suggested: {title}',
  draftTitle: '{title} · draft',
  attach: 'Attach',
  remove: 'Remove {title}',
  unavailable: 'No longer available to you',
  attached: 'Attached {title}.',
  removed: 'Removed {title} from this thread.',
  attachFailed: 'Couldn’t attach {title}',
  removeFailed: 'Couldn’t remove {title}',

  // ── Conversation ────────────────────────────────────────────────────
  log: 'Conversation',
  you: 'You',
  answer: 'The assistant',
  queued: 'Queued…',
  answering: 'Answering…',
  stoppedNote: 'Stopped. An in-flight request may still be charged.',
  partialNote: 'The answer stopped before it ended. What arrived is kept.',
  lengthNote: 'The answer reached its length limit.',
  summaryMarker: 'Earlier messages are summarized',
  copy: 'Copy',
  copied: 'Copied',
  // lang: the code block's language as the answer names it (liquid, sql).
  codeLabel: 'Code ({lang})',
  codeLabelPlain: 'Code',
  tableLabel: 'Table in the answer',
  // domain: where the image is; alt: its description.
  image: 'Image: {alt} ({domain})',
  imageNoAlt: 'Image ({domain})',
  newTab: '(opens in a new tab)',
  // name: a tool the assistant called; unknown ones show their own name.
  tool: '{name, select, read_report {Read report} read_attached {Read attached} search_help {Searched help} describe_screen {Described the screen} suggest_link {Suggested a link} propose_edit {Proposed an edit} draft_agent_task {Drafted an agent task} subject_ideas {Subject ideas} rewrite {Rewrite} other {Called a tool}}',
  toolFailed: '{code, select, not_available {not available to you} no_longer_available {no longer available to you} forbidden {not allowed} tool_limit {over the thread’s tool-call limit} not_found {not found} too_large {result too large} invalid_arguments {invalid arguments} report_unavailable {report not available} other {failed}}',
  // s: seconds, one decimal ("0.3").
  seconds: '{s} s',
  // ms: milliseconds.
  milliseconds: '{ms} ms',
  cardLink: 'Opens the screen. Nothing changes from here.',
  cardProposal: 'Nothing changes until you accept it.',
  cardAgent: 'You review the brief and confirm sharing before a run starts.',
  openCard: 'Open',
  reviewProposal: 'Review the proposal',
  startAgent: 'Start an agent run',
  hostReputeTitle: 'Ask HostRepute about reputation',
  hostReputeBadge: 'Outside the assistant',
  openHostRepute: 'Open HostRepute',
  openDeliverability: 'Open Deliverability',

  // ── An empty thread ─────────────────────────────────────────────────
  // name: the campaign or report the screen shows.
  emptyAbout: 'Ask about {name}',
  emptyTitle: 'Ask the assistant',
  emptyBody: 'The assistant reads what you attach, explains it and hands work to the right screen. It never sends, schedules or publishes.',
  emptyNote: 'Suggestions are fixed for this screen and cost nothing until you send one.',
  suggestReport1: 'Summarize the results',
  suggestReport2: 'Compare with the last issue',
  suggestReport3: 'Suggest subjects for the next issue',
  suggestCampaign1: 'Check this draft before I send it',
  suggestCampaign2: 'Suggest a shorter subject',
  suggestCampaign3: 'What will readers see on a phone?',
  suggestCampaigns1: 'Which campaign did best this month?',
  suggestCampaigns2: 'How do I schedule a campaign?',
  suggestCampaigns3: 'How does A/B testing work?',
  suggestOverview1: 'What needs my attention?',
  suggestOverview2: 'How do I grow my audience?',
  suggestOverview3: 'Explain the numbers on this screen',
  suggestAudience1: 'How do segments work?',
  suggestAudience2: 'How do I import contacts?',
  suggestAudience3: 'Why do contacts become inactive?',
  suggestAutomations1: 'How do I start a welcome series?',
  suggestAutomations2: 'What can start an automation?',
  suggestAutomations3: 'How do delays work?',
  suggestDeliverability1: 'How do I set up DKIM?',
  suggestDeliverability2: 'What does DMARC do?',
  suggestDeliverability3: 'Why do placement tests matter?',
  suggestSettings1: 'How do I invite my team?',
  suggestSettings2: 'How do API keys work?',
  suggestSettings3: 'Where do I connect a model?',
  suggestDefault1: 'What can you do on this screen?',
  suggestDefault2: 'How do I send my first campaign?',
  suggestDefault3: 'Where do I find the reports?',

  // ── States (AssistantStates) ────────────────────────────────────────
  noModelTitle: 'No model connected',
  noModelBadge: 'Not configured',
  noModelBody: 'Connect a model account in Settings › Integrations › AI models to use the assistant.',
  noModelAction: 'Connect a model',
  noUsableTitle: 'No model can answer here',
  noUsableBody: 'The assistant needs a model that passed its tool check. Settings › Integrations › AI models shows each model’s check.',
  noAccessTitle: 'Your role can’t use AI',
  noAccessBadge: 'No access',
  // workspace: the workspace's name.
  noAccessBody: 'Your role in {workspace} can’t use AI features, which the assistant needs.',
  offTitle: 'The assistant is off',
  offBadge: 'Off',
  offBody: 'An integration manager switched the assistant off for {workspace}. One-shot suggestions in the composer still work.',
  offAction: 'Open AI settings',
  threadCapTitle: 'This thread reached its limit',
  threadCapBadge: 'Paused',
  // calls…costLimit: the thread's usage and ceilings, formatted.
  threadCapBody: '{calls} of {callLimit} tool calls · {tokens} of {tokenLimit} tokens · {cost} of {costLimit}. Nothing here is lost. Start a new thread to go on.',
  monthlyTitle: 'Monthly AI cap reached',
  monthlyBadge: 'Paused',
  monthlyBody: '{workspace} used its monthly AI cap. Workspace models pause until the month ends or a manager raises the cap.',
  monthlyPersonal: 'Your personal key has its own cap: you can go on with {model}.',
  // model: a personal model's name.
  usePersonal: 'Use {model} (personal)',
  personalCapTitle: 'Your personal cap is reached',
  personalCapBody: 'Your personal key used its monthly cap. Pick a workspace model, or wait for next month.',
  keyTitle: '{provider} rejected the key',
  keyBadge: 'Key rejected',
  // own: yes when the thread's model is on the person's own key.
  keyBody: '{own, select, yes {{provider} rejected your own key, so nothing was sent. Replace it in AI models, or switch model.} other {{provider} rejected the key, so its models stop until someone who can manage integrations replaces it. Other models still work.}}',
  replaceKey: 'Replace key',
  switchModel: 'Switch model',
  rateTitle: '{provider} is limiting requests',
  rateBadge: 'Rate limited',
  rateBody: 'Too many requests for this key. Wait a moment, then retry. Retry sends a new request with its own cost.',
  providerTitle: '{provider} is unavailable',
  providerBadge: 'Failed',
  providerBody: '{provider} didn’t answer. The answer so far is kept. Retry sends a new request with its own cost.',
  refusalTitle: 'The model declined',
  refusalBadge: 'Refused',
  refusalBody: '{model} declined this request. Rephrase it, or ask about a resource you attach.',
  editMessage: 'Edit message',
  reconnectTitle: 'Reconnecting',
  reconnectBadge: 'Reconnecting',
  // seconds: since the connection dropped; attempt, max: reconnection attempts.
  reconnectBody: 'The connection dropped {seconds} s ago. The answer continues on the server and appears when the connection is back. Attempt {attempt} of {max}.',
  reconnectStopped: 'The connection dropped {seconds} s ago and didn’t come back. The answer continues on the server.',
  reconnectNow: 'Reconnect now',
  contactsTitle: 'The assistant can’t read contacts',
  contactsBadge: 'Denied',
  contactsBody: 'Contact records, recipient lists and exports are never available to the assistant. Ask about a segment’s size or a campaign’s totals instead.',
  contactsAction: 'Ask about a segment',
  contactsQuestion: 'How many contacts are in each of my segments?',
  expiredTitle: 'This thread expired',
  expiredBadge: 'Expired',
  expiredBody: 'Threads keep content for 30 days after the last message. Its usage stays in the AI activity log, without content.',
  lostTitle: 'Your access changed during this answer',
  lostBadge: 'Stopped',
  lostBody: 'Your role can no longer use AI features. The answer stopped before its next step; what was written stays readable.',
  modelGoneTitle: 'This thread’s model was removed',
  modelGoneBody: 'Pick another model for the thread, then send your message again.',
  disconnectedTitle: 'The model’s account was disconnected',
  disconnectedBadge: 'Disconnected',
  disconnectedBody: '{own, select, yes {Your own {provider} account was disconnected, so nothing was sent. Connect it again in AI models, or switch model.} other {The {provider} account of this model was disconnected, so nothing was sent. Someone who can manage integrations can connect it again.}}',
  // model: the model's name.
  toolsTitle: '{model} hasn’t passed its tool check',
  toolsBadge: 'Not checked',
  toolsBody: '{own, select, yes {The assistant needs your own model to pass its tool check, so nothing was sent. Run it, or switch model.} other {The assistant needs a model that passed its tool check (it runs again after a key is replaced), so nothing was sent. Someone who can manage integrations can run it.}}',
  runCheck: 'Run the tool check',
  checkPassed: '{model} passed its tool check',
  checkFailed: '{model} didn’t pass its tool check',
  personalOffTitle: 'Personal keys are off',
  personalOffBody: 'Personal keys were turned off in {workspace}, so this thread can’t use your own model and nothing was sent. Switch to a workspace model, or ask someone who can manage integrations to turn them on.',
  notConfiguredTitle: 'The model isn’t set up',
  notConfiguredBody: '{own, select, yes {There’s no key for your own {provider} model, so nothing was sent. Add it in AI models, or switch model.} other {No model account is connected, so nothing was sent. Someone who can manage integrations can connect one.}}',
  openModels: 'Open AI models',
  addKey: 'Add your key',
  confirmExpiredTitle: 'The request expired',
  confirmExpiredBody: 'The tool call waited 24 hours for your decision, so nothing was sent and the answer stopped. Retry sends your message again as a new message, with its own cost.',
  interruptedTitle: 'The answer was interrupted',
  interruptedBody: 'The server answering it stopped. What was written is kept; its cost stays unknown until a manager reconciles it. Retry sends a new request with its own cost.',
  failedTitle: 'The answer didn’t finish',
  failedBadge: 'Failed',
  // reason: the turn's reason code (provider_error, refused…).
  failedBody: 'It stopped ({reason}). What was written is kept. Retry sends a new request with its own cost.',
  retry: 'Retry',
  loadFailed: 'The thread didn’t load',
  threadsFailed: 'Threads didn’t load',
  sendFailed: 'Your message wasn’t sent',
  stopFailed: 'The answer wasn’t stopped',

  // ── Composer ────────────────────────────────────────────────────────
  composerLabel: 'Message the assistant',
  placeholder: 'Ask the assistant…',
  placeholderReport: 'Ask about this report…',
  placeholderCampaign: 'Ask about this campaign…',
  hint: 'Enter to send · Shift Enter for a new line',
  send: 'Send',
  stop: 'Stop',

  // ── Announcements (a polite live region) ────────────────────────────
  announceStart: 'The assistant is answering.',
  // sentence: the answer's first sentence.
  announceDone: 'Answer finished: {sentence}',
  announceStopped: 'Stopped.',
  // title: the state's title ("Anthropic is unavailable").
  announceFailed: 'The answer stopped: {title}',

  // ── Thread list ─────────────────────────────────────────────────────
  search: 'Search threads',
  searchPlaceholder: 'Search thread titles',
  // when: how long ago ("3 days ago"); model: the model's name.
  threadMeta: '{when} · {model}',
  threadMetaNoModel: '{when}',
  noThreads: 'No threads yet. Ask a question to start one.',
  noMatch: 'No thread titles match “{q}”.',
  privacy: 'Only you can read your threads. Managers see usage and cost, not content. Content is deleted 30 days after the last message.',
  threadActions: 'Actions for {title}',
  rename: 'Rename',
  renameTitle: 'Rename thread',
  titleLabel: 'Title',
  delete: 'Delete',
  deleteBadge: 'Delete thread',
  deleteTitle: 'Delete “{title}”?',
  deleteBody: 'Its messages and attached resources are deleted at once. Its usage stays in the AI activity log, without content.',
  deleted: 'Thread deleted',
  renamed: 'Thread renamed',
})
