import { defineMessages } from '@/i18n'

/**
 * Settings › Integrations › Website tracking (M62: TrackingSites,
 * TrackingSetup, TrackingStates): the sites, their last events, the add,
 * edit and set-up dialog, rotating the identify secret and deleting a site.
 * Code (the snippets, reputecast('consent', true), the HMAC formula) is
 * never translated: it is passed into the messages as values.
 */
export const tm = defineMessages('tracking', {
  // ── The page ──────────────────────────────────────────────────────────
  title: 'Website tracking',
  documentTitle: 'Website tracking · Integrations',
  crumbSettings: 'Settings',
  crumbIntegrations: 'Integrations',
  /** ago: 2 min ago */
  statusLastEvent: 'Connected · last event {ago}',
  statusConnected: 'Connected',
  statusNone: 'Not connected',
  /** the header under the title; name: the member who added the first site; date: 8 Oct 2026 */
  facts: 'Built in · a script on your own sites · {count, plural, one {# site} other {# sites}} · added by {name} on {date}',
  /** date: 8 Oct 2026 */
  factsNoName: 'Built in · a script on your own sites · {count, plural, one {# site} other {# sites}} · added on {date}',
  factsNone: 'Built in · a script on your own sites',
  lead: "Follow up on what known contacts look at on your own sites and stores. ReputeCast's own pages, such as your website, landing pages and signup forms, are never tracked.",
  capPages: 'Page views',
  capProducts: 'Product views',
  capCart: 'Add to cart',
  capCustom: 'Custom events',
  capConsent: 'Consent',
  capIdentify: 'Signed identify',
  addSite: 'Add site',
  hubLine: 'Follow up on what known contacts look at on your own sites and stores.',
  hubNoSites: 'No sites yet',

  // ── Sites ─────────────────────────────────────────────────────────────
  /** count: 2 */
  sitesTitle: 'Sites · {count}',
  sitesSub: 'Each site is one origin. Events from any other origin are refused.',
  sitesTable: 'Sites',
  colSite: 'Site',
  colStore: 'Store',
  colConsent: 'Consent',
  colEvents: 'Events today',
  colLast: 'Last event',
  colActions: 'Actions',
  notLinked: 'Not linked',
  waits: 'Waits for consent',
  noWait: "Doesn't wait",
  noEvents: 'No events yet',
  /** a site on narrow screens; store: Acme Merch or none; platform: Shopify; last: 2 min ago or none */
  narrowLine: '{store, select, none {Store: not linked} other {Store: {store}, {platform}}} · {last, select, none {no events yet} other {{events, plural, one {# event today} other {# events today}} · last event {last}}}',
  setUp: 'Set up',
  /** name: a site, Acme Merch */
  setUpLabel: 'Set up {name}',
  edit: 'Edit',
  /** name: a site, Acme Merch */
  editLabel: 'Edit {name}',
  /** name: a site, Acme Merch */
  moreLabel: 'More actions for {name}: Rotate identify secret, Delete',
  rotate: 'Rotate identify secret',
  delete: 'Delete site…',

  // ── Last events ───────────────────────────────────────────────────────
  /** name: a site, Acme Merch */
  eventsTitle: 'Last events · {name}',
  eventsSub: 'The last 20, newest first. Never an address.',
  /** the table of a site’s last events; name: Acme Merch */
  eventsTable: 'Last events on {name}',
  live: 'Live',
  colEvent: 'Event',
  colWhat: 'Page or product',
  colWho: 'Who',
  colWhen: 'When',
  kindPage: 'Viewed a page',
  kindProduct: 'Viewed a product',
  kindCart: 'Added to cart',
  known: 'Known contact',
  visitor: 'Visitor',
  notNamed: 'Not named: no store linked',
  notFound: 'A product the store doesn’t know',
  /** an event on phones: a page or product, then its time */
  eventLine: '{what} · {time}',
  waitingFirst: 'Waiting for the first event…',
  /** when: 14:31 or 9 Oct */
  waitingBy: '{origin} · added {when} by {name}',
  /** when: 14:31 or 9 Oct */
  waitingAdded: '{origin} · added {when}',
  /** when: 2 hours ago */
  waitingNothing: '{origin} · added {when} · nothing has arrived',
  /** origin: https://blog.acme.io */
  waitingHelp: 'Add the snippet to every page of {origin}. Events show here a few seconds after the first visit.',
  consentTitle: "This site waits for visitors' consent",
  /** call: reputecast('consent', true), as code */
  consentBody: 'Nothing arrives until your consent banner calls {call}. If the snippet is on the site and nothing shows, check that your banner makes that call when a visitor says yes.',
  /** origin: https://www.shop.acme.io */
  refusedTitle: 'Events from {origin} were refused',
  /** time: 14:20 */
  refusedBody: "In the last 24 hours, {count, plural, one {# event} other {# events}} from {origin} were refused, the last at {time}, because it isn't this site's origin. Nothing from them was stored. Add it as a site if it's yours.",
  addAsSite: 'Add it as a site',
  noStoreTitle: 'No store linked',
  /** origin: https://docs.acme.io */
  noStoreBody: "Product events from {origin} don't name products until it's linked to a store. They still count for the site, but store flows and recommendations can't use them.",
  linkStore: 'Link a store',
  eventsError: "Couldn't load the last events.",

  // ── How people are recognised ─────────────────────────────────────────
  recognisedTitle: 'How people are recognised',
  recognisedSub: 'Never from an email address a page states',
  clickTitle: 'A click from an email',
  /** rc is a URL parameter, not translated */
  clickBody: 'Links to your sites carry the click token <mono>rc</mono>, as links to a store do.',
  signupTitle: 'A signup on the same page',
  signupBody: 'A ReputeCast form hands the tracker a signed token that works for 10 minutes.',
  identifyTitle: "A signed identify call from the site's server",
  identifyBody: "Your server signs the address with the site's identify secret. Set up shows how.",
  recognisedFoot: "Only an existing contact who hasn't been erased is linked. Their last 30 days on the site then join their profile. Everyone else stays a visitor.",

  // ── What's kept ───────────────────────────────────────────────────────
  keptTitle: "What's kept",
  keptSub: 'And what never is',
  keptEvents: 'Events',
  keptEventsValue: "90 days, or the workspace's event-detail retention if it's shorter",
  keptVisitors: 'Visitors never linked',
  keptVisitorsValue: 'Deleted 30 days after they were last seen',
  keptInterest: 'Product interest',
  keptInterestValue: '180 days, for store flows and recommendations',
  keptErase: 'Erasing a contact',
  keptEraseValue: 'Deletes all of it. Their data export has a Site activity section.',
  stored: 'Stored',
  storedPaths: 'Page paths, with only their utm_* parameters',
  storedReferrer: 'The referring domain',
  storedProducts: 'Product views and add-to-carts',
  storedCustom: 'Custom events, up to 2 KB each',
  neverStored: 'Never stored',
  neverIp: 'IP addresses',
  neverAgents: 'User agents',
  neverQueries: 'Full query strings',
  neverForms: 'Form contents',
  gpc: "Global Privacy Control is always honoured: a browser that sends it is never tracked, whatever a site's consent setting.",

  // ── States ────────────────────────────────────────────────────────────
  loading: 'Loading the sites',
  emptyTitle: 'No sites yet',
  emptyBody: 'Add your shop or blog, and automations can follow up on what known contacts look at there.',
  errorTitle: "Couldn't load your sites",
  errorBody: "Your sites keep sending events: the script and the address it sends to don't depend on this page. Try again in a minute.",
  retry: 'Retry',
  /** name: the member; role: Viewer */
  readOnlyTitle: 'Read-only · {name}, {role}',
  /** helpers: Marcus Chen or Linh Nguyen */
  readOnlyBody: "You can see the sites, their last events and what's kept. Adding, editing and deleting sites and rotating identify secrets need integrations:manage. Ask {helpers}.",
  readOnlyAlone: "You can see the sites, their last events and what's kept. Adding, editing and deleting sites and rotating identify secrets need integrations:manage.",

  // ── The site dialog ───────────────────────────────────────────────────
  addTitle: 'Add a site',
  eyebrowPath: 'Settings › Integrations › Website tracking',
  steps: 'Steps',
  stepSite: 'The site',
  stepSnippet: 'Add the snippet',
  stepFirst: 'First event',
  name: 'Name',
  nameHint: 'Only shown in ReputeCast.',
  origin: 'Origin',
  originHint: 'The scheme and host as the browser shows it, e.g. https://shop.acme.io. Events from any other origin are refused.',
  originFixed: "The origin can't change. Add another site for another origin.",
  /** origin: https://shop.acme.io */
  originPath: "That's an address, not an origin. Use only the scheme and host: {origin}.",
  originBad: 'Enter the scheme and host, e.g. https://shop.acme.io.',
  nameMissing: 'Name the site, in 80 characters or fewer.',
  store: 'Store',
  /** a store in the Store select: Acme Merch · Shopify */
  storeOption: '{name} · {platform}',
  storeHint: 'Link a store so product events name its products.',
  wait: "Wait for visitors' consent",
  /** call: reputecast('consent', true), as code */
  waitSub: 'Nothing is stored until your consent banner calls {call}.',
  waitNote: 'Browsers that send Global Privacy Control are never tracked, whatever this switch says. Turned off, the site takes responsibility for its own legal basis.',
  cancel: 'Cancel',
  save: 'Save',
  audited: 'Recorded in the audit log.',
  /** name: a site, Acme Merch */
  editTitle: 'Edit {name}',
  /** date: 8 Oct 2026 */
  eyebrowAddedBy: '{name} · added by {by} on {date}',
  /** when: 14:36 or 9 Oct 2026 */
  eyebrowAdded: '{name} · added {when}',
  /** when: 9 Oct 2026 */
  eyebrowAddedWhenBy: '{name} · added {when} by {by}',
  /** origin: https://shop.acme.io */
  snippetTitle: 'Add the snippet to {origin}',
  snippetTabs: 'Snippet',
  tabScript: 'Script',
  tabPixel: 'Shopify custom pixel',
  tabWordPress: 'WordPress',
  /** head: </head>, as code */
  scriptHelp: 'Add this to every page of {origin}, before the closing {head}.',
  html: 'HTML',
  javascript: 'JavaScript',
  copy: 'Copy',
  copied: 'Copied',
  copiedToast: 'Copied to the clipboard',
  copyFailed: "Couldn't copy. Select the text and copy it yourself.",
  copyScript: 'Copy the script',
  copyPixel: 'Copy the Shopify custom pixel',
  /** product: reputecast('product', {id}); cart: reputecast('cart', {id, quantity}); both code */
  storeCalls: "On a store without the pixel, call {product} on product pages and {cart} when something goes in the cart, with the store's product id.",
  pixelHelpWait:
    "In Shopify, open Settings › Customer events, add a custom pixel named “ReputeCast” and paste this. Set its Customer privacy permission to Required, so Shopify runs it only after the visitor's consent, then Save and Connect.",
  pixelHelp: 'In Shopify, open Settings › Customer events, add a custom pixel named “ReputeCast” and paste this, then Save and Connect.',
  wpTitle: 'In WordPress',
  wp1: 'Open Settings › ReputeCast.',
  wp2: "Turn on Site tracking in the ReputeCast plugin's settings and paste the site id.",
  wp3: 'Paste the identify secret too, so customers who sign in are recognised.',
  wp4: 'Save. The plugin adds the script to every page and sends WooCommerce product views and add-to-carts.',
  siteId: 'Site id',
  copySiteId: 'Copy the site id',
  secretOnce: 'Identify secret · shown once',
  secret: 'Identify secret',
  copySecret: 'Copy the identify secret',
  shownOnce: 'Shown once',
  secretBody: "It belongs on the site's server, never in a page. ReputeCast never shows it again; if it's lost, rotate it.",
  /** formula and call are code */
  signHow: 'Sign each address on your server: {formula}, then call {call}.',
  secretCopied: 'I copied the identify secret',
  continue: 'Continue',
  /** name: a site, Docs */
  secretExisting: "It was shown once, when {name} was added. Rotate it from the site's menu to get a new one; the old one then stops working.",
  close: 'Close',
  checkEvents: 'Check that events arrive',
  /** name: a site, Acme Merch */
  eyebrowListening: '{name} · listening',
  /** time: 14:38 */
  eyebrowFirst: '{name} · first event at {time}',
  /** origin: https://shop.acme.io */
  listeningFor: '{origin} · listening for {minutes, plural, one {# min} other {# min}}',
  /** origin: https://shop.acme.io */
  listenHelpWait: 'Open {origin} with the snippet in place. This site waits for consent, so accept your banner first. You can close this: the site starts sending as soon as the snippet is on its pages.',
  /** origin: https://shop.acme.io */
  listenHelp: 'Open {origin} with the snippet in place. You can close this: the site starts sending as soon as the snippet is on its pages.',
  /** store: Acme Merch */
  arrivingStore: 'Events are arriving from {origin}. Product names come from {store}.',
  /** origin: https://shop.acme.io */
  arriving: 'Events are arriving from {origin}.',
  visitorsNote: 'Visitors become known contacts after a click from an email, a signup on the page or a signed identify call.',
  done: 'Done',
  /** a toast; name: a site */
  added: '{name} added',
  /** a toast; name: a site */
  saved: '{name} saved',

  // ── Rotate and delete ─────────────────────────────────────────────────
  rotateTitle: 'Rotate the identify secret?',
  /** Acme Merch · https://shop.acme.io */
  rotateEyebrow: '{name} · {origin}',
  rotateBody: 'The current secret stops working at once. Identify calls your server signs with it link nobody until it uses the new one.',
  stopsWorking: 'Stops working',
  stopsWorkingValue: 'Identify calls signed with the current secret',
  keepsWorking: 'Keeps working',
  keepsWorkingValue: 'Clicks from emails, signups on the page, and every event the site sends',
  rotateButton: 'Rotate secret',
  /** time: 14:42 */
  rotatedEyebrow: 'Rotated {time} by {name}',
  /** name: a site */
  rotatedTitle: 'New identify secret for {name}',
  rotatedBody: "Put it on the site's server in place of the old one. The old one already stopped working.",
  /** name: a site */
  deleteTitle: 'Delete {name}?',
  /** https://shop.acme.io · linked to Acme Merch (Shopify) · 1,240 events today */
  deleteLineStore: '{origin} · linked to {store} ({platform}) · {events, plural, one {# event today} other {# events today}}',
  /** https://docs.acme.io · 312 events today */
  deleteLine: '{origin} · {events, plural, one {# event today} other {# events today}}',
  deleteScript: 'The script',
  /** origin: https://shop.acme.io */
  deleteScriptValue: 'Stops being accepted at once. Events from {origin} are refused.',
  deleteEvents: 'Events',
  deleteEventsValue: 'Already collected events stay until they expire, at most 90 days.',
  deleteFlows: 'Flows',
  /** flows: Browse abandonment and Back in stock */
  deleteFlowsValue: '{flows} get no new people from this site. People already in them carry on.',
  deleteFlowsNone: 'Store flows get no new people from this site. People already in them carry on.',
  deleteNote: "Remove the snippet from the site's pages too. Until then, they keep sending events that are refused.",
  deleteButton: 'Delete site',
  /** a toast; name: a site */
  deleted: '{name} deleted',
})
