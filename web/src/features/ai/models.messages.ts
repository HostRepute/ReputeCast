import { defineMessages } from '@/i18n'

/**
 * Settings › Integrations › AI models (AIModels board, assistant brief B4):
 * the model accounts and their models, the model for each feature, the
 * assistant's switch and limits, personal keys, and the person's own key.
 */
export const mm = defineMessages('ai.models', {
  // ── Model accounts ──────────────────────────────────────────────────
  accountsTitle: 'Model accounts',
  accountsCaption: 'Keys are write-only. They are never shown again and never sent to another address.',
  addAccount: 'Add account',
  thAccount: 'Account',
  thModels: 'Models',
  thStatus: 'Status',
  thActions: 'Actions',
  // date: the day the key was set ("Nov 2"); name: who set it.
  keySetBy: 'Workspace key set {date} by {name}',
  keySet: 'Workspace key set {date}',
  // host: the server's address ("ollama.acme.internal").
  noKeyHost: 'No key · {host}',
  noKeyPrivate: 'No key · {host} (private network allowed by the instance)',
  keyRemoved: 'Key removed · add one to use its models again',
  demoAccount: 'Demo model: fixed sample answers, in development only',
  statusNoKey: 'No key',
  statusToolsFailed: 'Tools failed check',
  manage: 'Manage',
  // account: the account's name ("Anthropic", "Office Ollama").
  manageLabel: 'Manage {account}',
  // spent, cap: dollars ("$13.50", "$25.00").
  totalUsd: 'Workspace total {spent} of the {cap} monthly cap. Personal keys don’t count here.',
  // spent, cap: token counts ("1.6M").
  totalTokens: 'Workspace total {spent} of the {cap}-token monthly cap. Personal keys don’t count here.',
  totalNoCap: 'Workspace total {spent} this month, with no monthly cap. Personal keys don’t count here.',

  // ── Adding an account ───────────────────────────────────────────────
  addAccountTitle: 'Add a model account',
  addPersonalTitle: 'Add your personal key',
  connect: 'Connect',
  addPersonalButton: 'Test and add',
  testFirstReason: 'Test the connection first.',
  accountTestHint: 'The test sends a fixed prompt with no workspace content and counts toward the monthly cap.',
  personalTestHint: 'Adding tests the key with a fixed prompt, in your personal cap. Nothing from the workspace is sent.',
  // provider: the account's name; model: its first model.
  accountAddedToast: '{provider} added',
  accountAddedDetail: '{model} · pick it for a feature below',
  personalAddedToast: 'Your {provider} key was added',
  personalAddedDetail: '{model} · only your requests use it',

  // ── One account (Manage) ────────────────────────────────────────────
  accountEyebrow: 'Model account',
  modelsHeading: 'Models',
  // input, output: dollars per million tokens.
  modelPrices: '{input} input · {output} output per million tokens',
  modelNoPrice: 'No price set',
  // date: when the check ran.
  toolsPassed: 'Tool check passed {date}',
  toolsFailed: 'Failed the tool check {date}',
  toolsUntested: 'Tool check not run',
  // features: the features' names, as a list.
  usedBy: 'Used by {features}',
  usedByNone: 'No feature uses it',
  runToolCheck: 'Run tool check',
  // model: the model's name.
  runToolCheckLabel: 'Run tool check for {model}',
  toolsCost: 'The tool check sends a fixed request with no workspace content and counts toward the monthly cap. The assistant and the agent use only a model that passed it.',
  toolsCostPersonal: 'The tool check sends a fixed request with no workspace content, in your personal cap. The assistant uses your model only once it passed.',
  toolsNeedKey: 'This account’s key doesn’t work. Replace it first.',
  toolsPassedToast: '{model} passed the tool check',
  toolsFailedToast: '{model} failed the tool check',
  toolsFailedDetail: 'It can’t be the assistant’s or the agent’s model. One-shot suggestions still work.',
  editPrices: 'Prices',
  editPricesLabel: 'Prices of {model}',
  removeModel: 'Remove',
  removeModelLabel: 'Remove {model}',
  onlyModel: 'It’s this account’s only model. Add another first, or remove the account.',
  removeModelTitle: 'Remove {model}?',
  removeModelBody: 'Features that use it need another model first. The AI activity log keeps its requests.',
  modelRemovedToast: '{model} removed',
  addModel: 'Add model',
  addModelTitle: 'Add a model',
  // provider: the account's name.
  addModelLead: 'Another model on this {provider} key. Prices follow the model unless you change them.',
  modelAddedToast: '{model} added',
  removeAccount: 'Remove account',
  // account: the account's name.
  removeAccountTitle: 'Remove {account}?',
  removeAccountBody: 'Its key and its {count, plural, one {model} other {# models}} are deleted. Features that use them need another account’s model first. The AI activity log keeps their requests.',
  accountRemovedToast: '{account} removed',
  removeBadge: 'Remove',

  // ── The model for each feature ──────────────────────────────────────
  defaultsTitle: 'Default model for each feature',
  thFeature: 'Feature',
  thModel: 'Model',
  featureAssistant: 'Assistant',
  featureAgent: 'Agent runs',
  chooseModel: 'Choose a model',
  // model: the model's name; why it can't be picked here.
  whyTools: '{model} · failed the tool check',
  whyUntested: '{model} · tool check not run',
  whyImages: '{model} · doesn’t read images',
  whyKey: '{model} · key rejected',
  whyNoKey: '{model} · no key',
  defaultsFoot: 'A model that failed the tool check can’t be the assistant’s or agent’s default. Alt text needs a model that reads images.',
  // feature: the feature's name; model: the model's name.
  modelPickedToast: '{feature} now uses {model}',
  pickRefused: '{feature} keeps its model.',

  // ── The assistant ───────────────────────────────────────────────────
  assistantTitle: 'Assistant',
  assistantToggle: 'Enable the assistant',
  limitPerThread: 'Limit per thread',
  // tokens: a token count ("300,000"); cost: dollars ("$2.00").
  limitValue: '{calls, plural, one {# tool call} other {# tool calls}} · {tokens} tokens · {cost}',
  atOnce: 'At once',
  atOnceValue: '1 per thread · {person, number} per person · {workspace, number} per workspace',
  contentKept: 'Content kept',
  contentKeptValue: '30 days after the last message',
  leavesTitle: 'What leaves the instance',
  leavesBody: 'Only what a person attaches and confirms, ReputeCast’s help articles and the answers of approved MCP tools. MCP servers receive only the arguments of their own tools’ calls. The model account’s own terms decide how the provider stores it.',
  // The link from the note to Settings › Integrations › MCP servers.
  mcpLink: 'MCP servers and their approved tools',
  limitsFoot: 'Limits apply to threads started after a change. Answers already written stay readable when the assistant is off.',
  assistantOnToast: 'The assistant is on',
  assistantOffToast: 'The assistant is off',
  assistantOffDetail: 'Threads keep their answers; nobody can send a new message until it’s on again.',
  assistantChanged: 'Someone changed these settings meanwhile. The page shows their version now.',
  editLimits: 'Edit limits',
  limitsTitle: 'Assistant limits',
  limitsLead: 'Each thread’s ceiling, and the answers written at the same time. Threads started afterwards use them.',
  perPerson: 'Answers at once per person',
  perWorkspace: 'Answers at once in the workspace',
  perThreadFixed: 'Each thread writes one answer at a time.',
  limitsSavedToast: 'Assistant limits saved',

  // ── Personal keys ───────────────────────────────────────────────────
  personalTitle: 'Personal keys',
  personalCaption: 'Members who can use the AI assistant may add their own key',
  personalToggle: 'Allow personal keys',
  personalCap: 'Personal monthly cap',
  // cap: dollars.
  personalCapValue: '{cap} per person',
  personalCapValueBy: '{cap} per person · set by {name}',
  managersSee: 'Managers see',
  managersSeeValue: 'Usage and cost, never content',
  personalOnToast: 'Personal keys are allowed',
  personalOffToast: 'Personal keys are off',
  personalOffDetail: 'People’s keys are kept; their requests are refused until you allow them again.',
  capDialogLead: 'Each person’s budget for their own keys. Their requests stop at it until the month resets.',
  capPerPerson: 'Cap per person',
  editCap: 'Edit cap',
  peopleTitle: 'People with personal keys',
  peopleCaption: 'This month’s usage. Their keys and what they ask are never shown.',
  thPerson: 'Person',
  thKeys: 'Keys',
  thThisMonth: 'This month',
  // cost: dollars.
  personUsage: '{cost} · {requests, plural, one {# request} other {# requests}}',
  noKeyNow: 'No key now',
  // name: the person; provider: the key's provider.
  removeMemberLabel: 'Remove {name}’s {provider} key',
  removeMemberTitle: 'Remove {name}’s {provider} key?',
  removeMemberBody: 'For offboarding. The key and its models are deleted, and {name}’s picks of these models go with them. The AI activity log keeps their requests.',
  memberKeyRemovedToast: '{name}’s {provider} key was removed',
  nobodyPersonal: 'Nobody has added a personal key yet.',

  // ── Your personal key ───────────────────────────────────────────────
  yourTitle: 'Your personal key',
  onlyYou: 'Only you use it',
  offHere: 'Off for this workspace',
  offNote: 'Personal keys are off in this workspace. Your keys are kept, and their requests are refused until a manager allows personal keys again.',
  // cap: dollars a month.
  yourNone: 'Use a key of your own for your requests, within your personal cap of {cap} a month.',
  addPersonal: 'Add a personal key',
  // date: the day the key was set.
  yourKeySet: 'Key set {date}',
  yourUsage: 'This month',
  // spent, cap: dollars.
  yourUsageValue: '{spent} of {cap} this month',
  yourCapReached: 'Your personal cap is reached. Requests on your keys stop until {date}.',
  removeKey: 'Remove key',
  // provider: the key's provider.
  removeKeyLabel: 'Remove key for {provider}',
  removeYourTitle: 'Remove your {provider} key?',
  removeYourBody: 'The key and its models are deleted. Features you picked them for need the workspace’s model first. The AI activity log keeps your requests.',
  yourKeyRemovedToast: 'Your {provider} key was removed',
  yourPicksTitle: 'Your model for each feature',
  yourPicksCaption: 'For your own requests. Everyone else keeps the workspace’s models.',
  // model: the workspace's model for the feature.
  workspaceModel: 'The workspace’s model ({model})',
  workspaceModelNone: 'The workspace’s model',
  // feature: the feature's name; model: your model's name.
  pickedOwnToast: 'Your {feature} requests use {model}',
  pickedWorkspaceToast: 'Your {feature} requests use the workspace’s model',
})
