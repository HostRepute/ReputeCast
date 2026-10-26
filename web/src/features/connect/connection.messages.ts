import { defineMessages } from '@/i18n'

/**
 * 7.2 step 2, the connection: what the three flows share and the API
 * providers' key and webhooks (SetupError and its happy path). The
 * provider's heading, lead, fields, hints and errors come from the server.
 * {provider} is the provider's short name ("Mailgun", "SES").
 */
export const mn = defineMessages('connect.connection', {
  // ── Shared by the three flows ────────────────────────────────────────
  saveLater: 'Save and finish later',
  continueDomain: 'Continue to domain verification',
  testing: 'Testing…',
  waiting: 'Waiting',

  // ── The key ──────────────────────────────────────────────────────────
  keyCheckFailed: "Couldn't check the {provider} key",
  /** A required field left empty; {field} is its label, in lower case. */
  fieldMissing: 'Enter the {field}.',
  sectionDefault: '1. API key',
  moreSettings: 'More {provider} settings',
  checkAgain: 'Check again',
  connect: 'Connect',
  openKeys: 'Open {provider} API keys',
  /** {call} is the read-only API call: "GET /v3/domains". */
  keysChecked: 'Keys are checked with a read-only call ({call}) and stored encrypted.',
  keysCheckedPlain: 'Keys are checked with a read-only call and stored encrypted.',
  webhookFirst: 'The webhook endpoint has to work first, or use polling.',
  connectFirst: 'Connect {provider} first.',
  storedPlaceholder: 'Stored encrypted · paste a new one to replace it',

  // ── Webhooks (PR-9: the provider proves it can reach ReputeCast) ─────
  endpointAnswered: 'The test webhook arrived',
  webhooksRegistered: 'Webhooks registered at {provider}.',
  retestFailed: "Couldn't ask for another test",
  pollingFailed: "Couldn't switch to polling",
  badgeAfterKey: 'After the key is accepted',
  badgePolling: 'Polling every minute',
  badgeUnreachable: 'Endpoint not reachable from the internet',
  badgeTestWaiting: 'Waiting for a test webhook',
  badgeTestMissing: "Test webhook didn't arrive",
  badgeTestArrived: 'Test webhook arrived',
  badgeFirstEvent: 'Confirmed by the first event',
  badgeRegistered: 'Registered',
  badgeAddManually: 'Add the webhook in {provider}',
  badgeRegistrationFailed: 'Registration failed',
  webhooksTitle: '2. Webhooks',
  /** The address can't be reached from the internet (found before any provider call). */
  unreachableLead: '{provider} has to reach <mono>{url}</mono> to report deliveries, bounces and complaints, and that address can’t be reached from the internet.',
  /** {url} is the webhook address; the provider was asked to call it. */
  testMissingLead:
    'ReputeCast asked {provider} to send a test webhook to <mono>{url}</mono> and nothing arrived in 60 s. {provider} has to reach that address to report deliveries, bounces and complaints.',
  testWaitingLead: 'ReputeCast asked {provider} to send a test webhook to <mono>{url}</mono>. It usually arrives within a few seconds.',
  firstEventLead: "{provider} can't send a test webhook on request, so the test send in step 4 confirms that its events reach <mono>{url}</mono>.",
  pollingLead: "ReputeCast asks {provider}'s events API for new events every minute. Bounce protection during a send lags by up to a minute.",
  callsLead: '{provider} calls ReputeCast to report deliveries, bounces and complaints.',
  useWebhooks: 'Use webhooks instead',
  pollingRow: "Polling {provider}'s events API every minute",
  afterKey: 'after the key is accepted',
  /** The provider's test event, while it's on its way. */
  testRow: 'Test webhook from {provider}',
  /** {time} is when it arrived. */
  testArrived: 'Test webhook from {provider} arrived',
  /** {from} and {to} are the wait's clock times. */
  testMissing: 'Test webhook from {provider}: nothing arrived between {from} and {to}',
  /** The provider refused to send a test; {error} is its answer. */
  testRefused: '{provider} refused to send a test webhook: {error}',
  firstEventRow: 'Events from {provider} at <mono>{url}</mono>',
  afterTestSend: 'after the test send',
  /** Followed by the reason ("reputecast.acme.io resolves to 10.0.4.18, a private address Mailgun can't reach"). */
  likelyCause: 'Likely cause:',
  /** Before connecting: the address can't be reached at all. */
  addressRow: 'Address <mono>{url}</mono>',
  howToFix: 'How to fix',
  setPublicUrl: 'Set a public URL',
  /** {env} is the setting's name, REPUTECAST_PUBLIC_URL. */
  publicUrlPrivate:
    'ReputeCast is running at a private address. Set <mono>{env}</mono> in your docker compose .env to the HTTPS address your reverse proxy exposes, then restart.',
  publicUrlUnreachable:
    "Providers can't reach ReputeCast at its public URL. Set <mono>{env}</mono> in your docker compose .env to the HTTPS address your reverse proxy exposes, then restart.",
  envLabel: 'Setting for the docker compose .env',
  tunnelTitle: 'Or use a tunnel while testing',
  /** {env} is REPUTECAST_PUBLIC_URL. */
  tunnelLine: "For a laptop or staging box, expose port 8080 with Cloudflare Tunnel or ngrok and paste the URL here. {provider}'s webhooks use it instead of <mono>{env}</mono> until you remove it.",
  publicUrl: 'Public URL',
  pollTitle: 'Or poll instead of webhooks',
  pollLine: '{provider} keeps an events API. ReputeCast can poll it every minute. Slower, and bounce protection during a send lags by up to a minute.',
  usePolling: 'Use polling for now',
  retest: 'Ask {provider} for another test',
  recheck: 'Check the address again',
  proxyGuide: 'Reverse-proxy guide',
  registerAgain: 'Try registering again',
  /** Under the footer, next to Continue. */
  continueTest: 'Continue unlocks when a test webhook from {provider} arrives, or when you switch to polling.',
  continueFirstEvent: 'The test send in step 4 confirms that events from {provider} arrive.',

  // ── A webhook added by hand, with its secret ─────────────────────────
  secretFailed: "Couldn't create the webhook secret",
  manualTitle: 'Set up the webhook in {provider}',
  /** {header} is the HTTP header the secret goes in. */
  manualHeader: 'Point it at the URL above and send the secret below in the <mono>{header}</mono> header. ReputeCast refuses deliveries without it.',
  manualSign: "Point it at the URL above and sign the deliveries with the secret below. ReputeCast refuses deliveries it can't verify.",
  copySecret: 'Copy secret',
  secretOnce: "Copy it now: it isn't shown again. Keep it private; anyone with it can post events for this provider.",
  creating: 'Creating…',
  createSecret: 'Create a secret',
  secretReplaces: 'A new secret replaces the one {provider} has now, so paste it there right away. Recorded in the audit log.',

  // ── Registering the webhooks; {events} lists the provider's event names ──
  registered: 'Registered webhooks: {events}',
  addManually: 'Add <mono>{url}</mono> as a webhook in {provider} for {events}',
  copyUrl: 'Copy URL',
  register: 'Register webhooks: {events}',
  waitingAside: 'waiting',
  afterReachable: 'after the endpoint is reachable',
})

/** 7.2 step 2 for Amazon SES (SetupSES): AWS access, region, provisioning, sandbox. */
export const mses = defineMessages('connect.connection.ses', {
  accessFailed: "Couldn't check the AWS access",
  regionFailed: "Couldn't move to that region",
  eventsFailed: "Couldn't change how SES events arrive",
  /** Under the step's lead: how long setup takes, honestly. */
  setupTime: 'Most setups take 10–15 minutes; a new AWS account can wait up to 24 hours for production access.',
  grantTitle: '1. How to grant access',
  grantLegend: 'How to grant access',
  /** Where ReputeCast runs decides the access it recommends. */
  hostOffAws: "ReputeCast isn't running on AWS, so CloudFormation would still need keys. IAM access keys are recommended here.",
  /** {kind} is EC2, ECS, EKS or Lambda. */
  hostOnAws: "ReputeCast runs on AWS ({kind}), so the stack's role needs no long-lived keys. CloudFormation is recommended here.",
  hostCheckedOff: 'Checked at {time}: no EC2 or ECS instance metadata answered on this server.',
  hostCheckedOn: 'Checked at {time}: {kind, select, ec2 {the EC2 instance metadata answered} ecs {the ECS task metadata is set} eks {the EKS service account role is set} other {the Lambda runtime is set}} on this server.',
  recommendedHere: 'Recommended here',
  stackTitle: 'CloudFormation stack',
  stackLine: 'Best when ReputeCast runs on AWS (EC2, ECS): the stack creates an IAM role that ReputeCast assumes, so there are no long-lived keys.',
  /** The role path off AWS (PR-5): assuming a role needs credentials first. */
  stackNeedsKeys: "Off AWS, ReputeCast can only assume the stack's role with AWS credentials of its own on this server (environment variables or a shared credentials file).",
  stackNoKeys: "This server has no AWS credentials to assume the role with, so use IAM access keys, or run ReputeCast on AWS.",
  launchStack: 'Launch stack in AWS',
  roleArnLabel: "Role ARN, from the stack's RoleArn output",
  /** {id} is the external id, in the mono face. */
  roleArnHint: "Paste the stack's RoleArn output. External id <mono>{id}</mono> is filled in for you.",
  roleArnHintNoId: "Paste the stack's RoleArn output. External id is filled in for you.",
  keysTitle: 'IAM access keys',
  keysLine: 'Create an IAM user with the policy below and paste its access key. Keys are encrypted at rest and never shown again.',
  replaceKeys: 'Replace keys',
  /** The saved secret's field, read-only. */
  savedSecret: 'Saved, never shown',
  /** AWS's own name for the field, as its console shows it. */
  keyIdLabel: 'Access key id',
  /** AWS's own name for the field, as its console shows it. */
  secretLabel: 'Secret access key',
  /** AWS's own name for the field, as its console shows it. */
  secretPlaceholder: 'Secret access key',
  policyToggle: 'Show the IAM policy (least privilege)',
  policyLabel: 'IAM policy',
  connected: 'Connected',
  /** How the access was checked, then when ({time}) and the AWS account. {key} is the masked access key ("AKIA…7QXP"). */
  connectedLine:
    '{access, select, role {Role <mono>{arn}</mono> assumed} other {{hasKey, select, yes {Access key <mono>{key}</mono>{hasUser, select, yes { for IAM user <mono>{user}</mono>} other {}} verified} other {IAM access keys checked}}}}{hasTime, select, yes { at {time}} other {}}{hasAccount, select, yes { · account {account}} other {}}',
  checkAccess: 'Check access',
  checkAccessLine: 'ReputeCast checks the access with read-only calls first (GetAccount), then stores it encrypted.',
  regionTitle: '2. Region',
  regionLabel: 'SES region',
  regionReadHint: 'ReputeCast read your account in each region. Pick the one where you already have production access, or the one closest to your recipients.',
  regionPickHint: 'Pick the region where you already have production access, or the one closest to your recipients. ReputeCast reads your account in each region once connected.',

  // ── How events come back (PR-4) ──────────────────────────────────────
  eventsTitle: '3. How SES events reach ReputeCast',
  eventsLine: 'Deliveries, bounces and complaints come from SES through SNS. Pick how they get here.',
  eventsLegend: 'How SES events reach ReputeCast',
  eventsWebhook: 'Webhook',
  /** {url} is the webhook address, in the mono face. */
  eventsWebhookLine: 'SNS calls <mono>{url}</mono> · needs a public HTTPS URL',
  eventsQueue: 'Queue',
  eventsQueueLine: 'SNS → an SQS queue ReputeCast polls every 20 s · works behind NAT and firewalls · about $0.40 per million events',
  eventsChangeNote: 'Changing this sets up the topic again and checks events again.',

  provisioningTitle: '4. Provisioning and event check · {done} of {total} done',
  provisioningLine: 'Created for you; each step retries on its own',
  retryFailed: "Couldn't retry provisioning",
  awsFirst: 'Connect your AWS account first.',
  fixPolicy: 'Fix policy',
  running: 'running',
  waiting: 'waiting',
  /** A provisioning row that waits for a verified sending domain. */
  waitingDomain: 'after domain verification',
  /** Under the footer: why Continue is or isn't available. {time} is when the event arrived. */
  continueVerified: 'Continue stays disabled until a real delivery event from SES reaches ReputeCast. It arrived at {time}, so events are verified.',
  continueWaiting: 'Continue stays disabled until a real delivery event from SES reaches ReputeCast.',
  continueDeferred: 'SES only sends from a verified address, so the event check goes out once your domain is verified. You can continue now; setup finishes after the event arrives.',
  eventsFirst: 'Waiting for a delivery event from SES.',

  // ── The SES sandbox ──────────────────────────────────────────────────
  /** {region} is an AWS region code ("us-east-1"). */
  sandboxBadge: 'Sandbox detected in {region}',
  /** {max24h} emails a day and {rate} a second are the region's sandbox quota. */
  sandboxLine:
    '{selected, select, yes {{region} is in the SES sandbox:} other {If you switch to {region}, that region is in the SES sandbox:}} {quota, select, yes {{max24h} emails per day, {rate} per second, and only to verified addresses} other {a small daily quota, and only to verified addresses}}. ReputeCast can file the production-access request for you.',
  website: 'Website',
  mailType: 'Mail type',
  mailMarketing: 'Marketing (newsletter)',
  mailTransactional: 'Transactional',
  useCase: 'Use case description (prefilled from your workspace)',
  dailyVolume: 'Expected daily volume',
  contactLanguage: 'Contact language',
  /** The language AWS writes to you in. */
  langEnglish: 'English',
  /** The language AWS writes to you in. */
  langJapanese: 'Japanese',
  requested: 'Requested {time}',
  requesting: 'Requesting…',
  requestAccess: 'Request production access for {region}',
  awsAnswersSelected: 'AWS usually answers within 24 hours. Meanwhile you can send to verified addresses.',
  awsAnswersOther: 'AWS usually answers within 24 hours. You can keep going with {region} meanwhile.',
  requestedToast: 'Production access requested for {region}',
  requestedDetail: 'AWS usually answers within 24 hours.',
  requestFailed: "Couldn't file the production-access request",
})

/** 7.2 step 2 for an SMTP server (SetupSMTP): outgoing server and bounce mailbox. */
export const msmtp = defineMessages('connect.connection.smtp', {
  testFailed: "Couldn't test the SMTP server",
  serverTitle: '1. Outgoing server',
  serverLine: "Also fine for a Postal server's SMTP endpoint",
  host: 'Host',
  port: 'Port',
  encryption: 'Encryption',
  /** {port} is the port number it uses. */
  implicitTls: 'Implicit TLS · {port}',
  noTls: 'no TLS',
  username: 'Username',
  password: 'Password',
  passwordPlaceholder: 'Stored encrypted, never shown again',
  maxRate: "Max rate you're allowed",
  /** An example of what to type; the server reads the number. */
  maxRatePlaceholder: '5 per second',
  maxRateHint: "SMTP servers don't publish limits; ReputeCast throttles to this and backs off on 4xx replies.",
  dailyCap: 'Daily cap',
  enterHost: 'Enter the host first.',
  testConnection: 'Test connection',
  /** {summary} is the server's description of what it found. */
  connectedAt: 'Connected at {time} · {summary}',
  couldntConnect: "Couldn't connect",
  canTitle: "What SMTP can and can't do here",
  canBounces: 'Bounces and complaints via the bounce mailbox, usually within a few minutes of the event',
  canSuppression: 'Shared suppression list, auto-pause thresholds, per-mailbox-provider throttling',
  canDomain: "Domain setup: ReputeCast tells you the DNS records; it can't add them to the server for you",
  canDelivered: 'Delivered events: inferred from the SMTP 250 reply, not from the mailbox provider',
  canQuota: 'Quota and rate discovery, batch sending, sandbox mode',
  testMailboxFirst: 'Test the bounce mailbox first.',
  testServerFirst: 'Test the SMTP server first.',

  // ── The bounce mailbox ───────────────────────────────────────────────
  mailboxFailed: "Couldn't test the bounce mailbox",
  mailboxTitle: '2. Bounce mailbox',
  requiredForSmtp: 'Required for SMTP',
  mailboxLine:
    'Mailbox providers send delivery failures (DSNs) back to the return-path address. ReputeCast polls that mailbox, parses each DSN and ARF report, and turns it into the same normalized bounce or complaint event other providers deliver by webhook.',
  returnPath: 'Return-path address',
  protocol: 'Protocol',
  imapRecommended: 'IMAP (recommended)',
  mailServer: 'Mail server',
  storedPlaceholder: 'Stored encrypted',
  folder: 'Folder',
  pollEvery: 'Poll every',
  pollMinutes: '{n, plural, one {# minute} other {# minutes}}',
  afterProcessing: 'After processing',
  /** {folder} is the mailbox folder's name, "Processed". */
  moveProcessed: 'Move to "{folder}"',
  leave: 'Leave in place',
  outgoingFirst: 'Test the outgoing server first.',
  testMailbox: 'Test mailbox',
  /** {summary} is the server's description of the mailbox. */
  mailboxConnected: 'Connected · {summary}',
  mailboxUnreadable: "Couldn't read the mailbox",
})
