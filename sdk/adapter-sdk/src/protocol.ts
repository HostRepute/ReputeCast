/**
 * The ReputeCast adapter protocol, version 1: signed JSON over HTTP(S)
 * between ReputeCast and an adapter process. The specification is
 * docs/adapters.md ("Community adapters"); these types mirror it.
 */

export const PROTOCOL_VERSION = 1;

/** Header names (lowercase, as node:http exposes them). */
export const HEADERS = {
  protocol: "x-reputecast-protocol",
  timestamp: "x-reputecast-timestamp",
  signature: "x-reputecast-signature",
  operation: "x-reputecast-operation",
  idempotency: "idempotency-key",
} as const;

/** Every operation, called as POST /v1/<operation>. */
export const OPERATIONS = [
  "health",
  "verify",
  "send",
  "webhooks/verify",
  "webhooks/parse",
  "webhooks/register",
  "domains/add",
  "domains/status",
  "quota",
  "suppressions/list",
  "suppressions/add",
] as const;

export type Operation = (typeof OPERATIONS)[number];

export function isOperation(s: string): s is Operation {
  return (OPERATIONS as readonly string[]).includes(s);
}

/** How long ReputeCast waits for each operation, in milliseconds. */
export const TIMEOUTS: Record<Operation, number> = {
  health: 5_000,
  verify: 20_000,
  send: 30_000,
  "webhooks/verify": 5_000,
  "webhooks/parse": 15_000,
  "webhooks/register": 30_000,
  "domains/add": 30_000,
  "domains/status": 30_000,
  quota: 20_000,
  "suppressions/list": 60_000,
  "suppressions/add": 60_000,
};

export type Stream = "broadcast" | "automation" | "system";

export interface Address {
  name: string;
  address: string;
}

/** One message, already rendered and personalized. */
export interface Message {
  /** ReputeCast's message id; send it back on events (metadata or the X-ReputeCast-Message header). */
  id: string;
  stream: Stream;
  from: Address;
  replyTo?: Address;
  to: Address;
  subject: string;
  html?: string;
  text?: string;
  /** List-Unsubscribe, List-Unsubscribe-Post, Feedback-ID, X-ReputeCast-Message… Send them all. */
  headers: Record<string, string>;
  tags?: Record<string, string>;
}

export type ConnectionValue = string | number | boolean;

/** The connected provider a call is for: only the values the manifest's config declares. */
export interface Connection {
  providerId: string;
  workspaceId: string;
  settings: Record<string, ConnectionValue>;
  credentials: Record<string, ConnectionValue>;
  /** The provider's ReputeCast-generated webhook secret (adapters with webhooks). */
  webhookSecret?: string;
}

/** The body of every request ReputeCast sends. */
export interface Envelope<I = unknown> {
  protocol: 1;
  operation: Operation;
  adapter: string;
  idempotencyKey: string;
  /** RFC 3339: when ReputeCast stops waiting. */
  deadline: string;
  connection?: Connection;
  input: I;
}

export type ErrorCode =
  | "invalid_request"
  | "unauthenticated"
  | "too_large"
  | "bad_signature"
  | "provider_throttled"
  | "provider_auth"
  | "provider_error"
  | "provider_unavailable"
  | "not_supported"
  | "internal";

/** The HTTP status that goes with each error code. */
export const ERROR_STATUS: Record<ErrorCode, number> = {
  invalid_request: 400,
  unauthenticated: 401,
  too_large: 413,
  bad_signature: 422,
  provider_throttled: 429,
  provider_auth: 502,
  provider_error: 502,
  provider_unavailable: 503,
  not_supported: 501,
  internal: 500,
};

export interface WireError {
  code: ErrorCode;
  /** A sentence for ReputeCast's event log; never include credentials. */
  message: string;
  /** What the provider answered (HTTP status). */
  status?: number;
  /** The provider's own error code. */
  providerCode?: string;
  /** Seconds before retrying. */
  retryAfter?: number;
}

/** The body of every answer. */
export interface ResponseEnvelope<R = unknown> {
  protocol: 1;
  idempotencyKey: string;
  result?: R;
  error?: WireError;
}

export interface SendInput {
  messages: Message[];
}

export interface SendResult {
  messageId: string;
  providerMessageId?: string;
  error?: { code: string; message: string; temporary: boolean; throttled: boolean };
}

export interface SendOutput {
  results: SendResult[];
}

/** A provider webhook ReputeCast received and forwards. */
export interface WireWebhookRequest {
  method: string;
  /** ReputeCast's public URL of the webhook endpoint. */
  url: string;
  /** Lowercase header names. */
  headers: Record<string, string[]>;
  /** The raw body, base64. */
  body: string;
}

export interface WebhookVerifyInput {
  request: WireWebhookRequest;
}

export interface WebhookVerifyOutput {
  handshake?: { body: string };
}

export interface WebhookParseInput {
  request: WireWebhookRequest;
  /** True when ReputeCast already verified this request at receipt: don't verify again. */
  verified: boolean;
}

export type EventType =
  | "delivered"
  | "deferred"
  | "hard_bounce"
  | "soft_bounce"
  | "complaint"
  | "rejected"
  | "open"
  | "click"
  | "unsubscribe";

export const EVENT_TYPES: readonly EventType[] = [
  "delivered",
  "deferred",
  "hard_bounce",
  "soft_bounce",
  "complaint",
  "rejected",
  "open",
  "click",
  "unsubscribe",
];

export type BounceClass =
  | "mailbox_unknown"
  | "mailbox_full"
  | "domain_invalid"
  | "policy_block"
  | "spam_block"
  | "reputation"
  | "temporary"
  | "other";

export const BOUNCE_CLASSES: readonly BounceClass[] = [
  "mailbox_unknown",
  "mailbox_full",
  "domain_invalid",
  "policy_block",
  "spam_block",
  "reputation",
  "temporary",
  "other",
];

/** A normalized event as it travels. */
export interface WireEvent {
  type: EventType;
  recipient: string;
  providerMessageId?: string;
  messageId?: string;
  occurredAt: string;
  originalType?: string;
  originalCode?: string;
  diagnostic?: string;
  bounceClass?: BounceClass;
  url?: string;
  userAgent?: string;
  raw: unknown;
}

export interface WebhookParseOutput {
  events: WireEvent[];
}

export interface RegisterInput {
  url: string;
}

export interface DomainInput {
  domain: string;
}

export interface DnsRecord {
  purpose: "dkim" | "spf" | "return_path" | "dmarc" | "verification";
  type: "CNAME" | "TXT" | "MX";
  name: string;
  value: string;
  status?: "pending" | "verified" | "wrong_value" | "missing";
}

export interface DomainOutput {
  records: DnsRecord[];
}

export interface Quota {
  kind?: "rolling_24h" | "monthly_plan";
  /** 0 when unknown or unlimited. */
  max?: number;
  used?: number;
  /** Messages per second; 0 when unknown. */
  maxRate?: number;
  sandbox?: boolean;
  /** The account as people read it: "Growth plan · 50,000 per month". */
  account?: string;
}

export interface SuppressionsListInput {
  since: string | null;
  cursor: string | null;
}

export interface Suppression {
  email: string;
  reason: "hard_bounce" | "complaint" | "unsubscribe" | "manual";
  code?: string;
  createdAt?: string;
}

export interface SuppressionsListOutput {
  suppressions: Suppression[];
  nextCursor: string | null;
}

export interface SuppressionsAddInput {
  emails: string[];
}

export interface HealthOutput {
  adapter: string;
  version: string;
  protocol: 1;
  sdk?: string;
  capabilities?: unknown;
}

export const LIMITS = {
  /** Most suppressions per suppressions/list page. */
  suppressionsPerPage: 1000,
  /** Most addresses per suppressions/add call. */
  suppressionsPerPush: 500,
  /** Identifiers longer than this make an answer invalid. */
  identifier: 512,
  /** Texts are cut at this length. */
  text: 2000,
  /** The largest handshake answer. */
  handshake: 16 << 10,
} as const;
