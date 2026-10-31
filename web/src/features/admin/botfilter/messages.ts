import { defineMessages } from '@/i18n'

/**
 * Instance admin › Bot filter (InstanceBotFilter, InstanceBotFilterDialogs;
 * review AN-2): the networks and user agents whose opens and clicks aren't
 * people's, with their add and remove dialogs. Addresses, user agent text
 * and product names are arguments, never translated.
 */
export const m = defineMessages('admin.botfilter', {
  documentTitle: 'Bot filter',
  introTitle: 'Only people’s opens and clicks count',
  intro:
    'Opens and clicks from these networks and user agents are a machine’s, or privacy-protected for a privacy proxy. Reports, segments, automations and scores in every workspace leave them out. Built-in entries come with ReputeCast releases; add the ones your recipients’ mail goes through.',
  nothingChanged: 'Nothing was changed.',
  builtinSummary: 'they come with ReputeCast releases',
  remove: 'Remove',
  // entry: a network such as 203.0.113.0/24, or user agent text; never translated.
  removeLabel: 'Remove {entry}',
  // An entry without a name (a script's or a link preview's).
  noName: '–',

  // ── Networks ─────────────────────────────────────────────────────────
  networksTitle: 'Networks',
  networksTitleCount: 'Networks · {count}',
  networksCaption: 'Privacy proxies and mail scanners, by IP address or range',
  networksLoading: 'Loading the networks',
  networksLoadFailed: "Couldn't load the networks",
  addNetwork: 'Add network',
  networksTable: 'Networks admins added',
  networksBuiltinTable: 'Built-in networks',
  networksNoneAdded: 'No network added yet: the built-in ones below apply.',
  colNetwork: 'Network',
  colRunBy: 'Who runs it',
  colKind: 'Kind',
  colAdded: 'Added',
  colActions: 'Actions',
  netPrivacy: 'Privacy proxy',
  netScanner: 'Mail scanner',
  networksBuiltin: 'Built-in · {count, plural, one {# network} other {# networks}}',

  // ── User agents ──────────────────────────────────────────────────────
  agentsTitle: 'User agents',
  agentsTitleCount: 'User agents · {count}',
  agentsCaption: 'Text in the user agent of scanners, scripts and link previews, and of people’s devices the rules take for machines',
  agentsLoading: 'Loading the user agents',
  agentsLoadFailed: "Couldn't load the user agents",
  addAgent: 'Add user agent',
  agentsTable: 'User agents admins added',
  agentsBuiltinTable: 'Built-in user agents',
  agentsNoneAdded: 'No user agent added yet: the built-in ones below apply.',
  colPattern: 'Text in the user agent',
  colWho: 'Who or what',
  kindScanner: 'Scanner',
  kindScript: 'Script',
  kindHeadless: 'Headless browser',
  kindPreview: 'Link preview',
  kindPerson: 'Person’s device',
  agentsBuiltin: 'Built-in · {count, plural, one {# user agent} other {# user agents}}',
  agentsBuiltinNote: 'Security scanners, scripts, headless browsers and link previews, and {people} as people’s',
  agentsBuiltinNoPeople: 'Security scanners, scripts, headless browsers and link previews',

  // ── How the filter decides ───────────────────────────────────────────
  rulesTitle: 'How the filter decides',
  rulesCaption: 'The same rules in every workspace',
  rulesOpens: 'Opens',
  rulesOpensBody:
    'A machine’s: HEAD requests, no user agent, the user agents above, the scanner networks when mail arrives, and any open within 2 seconds of arrival. Privacy-protected: Apple Mail Privacy Protection and the privacy proxies above. Gmail’s and Yahoo’s image proxies are people after the first 30 seconds.',
  rulesClicks: 'Clicks',
  rulesClicksBody:
    'A machine’s: the user agents above, the scanner networks within 2 minutes of arrival, clicks within 10 seconds of arrival, 3 or more links within 5 seconds, and the hidden link no person sees. A click waits a minute before it counts, so the clicks around it can decide.',
  rulesWords: 'Rules',
  rulesWordsBody: 'A user agent with a word ending in “bot”, or with “crawler”, “spider”, “slurp” or “preview”, is a machine’s unless the word is a person’s device on the list.',
  rulesChanges: 'Changes',
  rulesChangesBody: 'Apply to new opens and clicks within 5 minutes; the ones counted already stay as they are. Every change is in the instance <a>audit log</a>.',

  // ── Add a network ────────────────────────────────────────────────────
  dialogEyebrow: 'Dialog · instance admin',
  addNetworkTitle: 'Add a network',
  addNetworkDescription: 'Opens and clicks from it count as a machine’s, or as privacy-protected, in every workspace within 5 minutes.',
  cidrLabel: 'Address or range',
  cidrHint: 'An address or a network, such as 203.0.113.0/24 or 2001:db8::/48: no wider than /8 for IPv4 or /16 for IPv6.',
  runByLabel: 'Who runs it',
  runByPlaceholder: 'Our mail gateway',
  kindLegend: 'Kind',
  netScannerHint: 'Its opens and clicks when mail arrives are a machine’s. Later ones from it can be people behind the gateway, and count.',
  netPrivacyHint: 'It fetches images when mail arrives, whether or not anyone reads it, so its opens are privacy-protected.',
  noteLabel: 'Note',
  notePlaceholder: 'What it is, where you saw it',
  cidrFirst: 'Enter the address or range first',
  nameFirst: 'Say who runs it first',
  networkAdded: '{cidr} added to the bot filter',
  networkRemoved: '{cidr} removed from the bot filter',

  // ── Add a user agent ─────────────────────────────────────────────────
  addAgentTitle: 'Add a user agent',
  addAgentDescription:
    'Text in the user agent of a machine your recipients’ mail meets, or of a person’s device the rules take for one. It applies in every workspace within 5 minutes.',
  itIsLegend: 'It is',
  aMachine: 'A machine',
  aMachineHint: 'A scanner, link checker or script: its opens and clicks are left out wherever the text is in the user agent.',
  aPerson: 'A person’s device',
  aPersonHint: 'A device or app with a word in its user agent that the rules take for a machine’s, such as one ending in “bot”: its opens and clicks count as people’s.',
  patternLabel: 'Text in the user agent',
  patternHint: 'Not case-sensitive. At least 3 characters; a person’s device is one word, letters and digits.',
  whoLabel: 'Who or what it is',
  whoPlaceholder: 'Our mail gateway',
  patternFirst: 'Enter at least 3 characters of the user agent first',
  whoFirst: 'Say who or what it is first',
  agentAdded: '“{pattern}” added to the bot filter',
  agentRemoved: '“{pattern}” removed from the bot filter',

  // ── Remove ───────────────────────────────────────────────────────────
  removeBadge: 'Remove from the bot filter',
  removeNetworkTitle: 'Remove {cidr}?',
  removeNetworkDescription:
    'Opens and clicks from it count as people’s again in every workspace within 5 minutes. The ones counted already stay as they are, and the change is in the instance audit log.',
  removeAgentTitle: 'Remove “{pattern}”?',
  removeAgentDescription:
    'Opens and clicks with it in their user agent count as people’s again in every workspace within 5 minutes. The ones counted already stay as they are, and the change is in the instance audit log.',
  removePersonDescription:
    'Opens and clicks from devices with this word in their user agent count as a machine’s again in every workspace within 5 minutes. The ones counted already stay as they are, and the change is in the instance audit log.',
  addedBy: 'Added by',
  addedByLine: '{hasName, select, yes {{name} on {date}} other {{date}}} · {what}',
})
