import type { HttpClient } from "./http.ts";
import type { Logger } from "./log.ts";
import type { Capabilities, NormalizedCapabilities, Terms } from "./manifest.ts";
import { ID_PATTERN, ManifestError, capabilityProblems, normalizeCapabilities } from "./manifest.ts";
import type {
  BounceClass,
  Connection,
  ConnectionValue,
  DnsRecord,
  EventType,
  Message,
  Operation,
  Quota,
  Suppression,
} from "./protocol.ts";
import type { WebhookRequest } from "./webhooks.ts";

/** Everything an operation gets besides its input. */
export interface AdapterContext {
  readonly adapter: string;
  readonly operation: Operation;
  readonly idempotencyKey: string;
  /** The connected provider the call is for. */
  readonly connection: Connection;
  /** Non-secret connection values (the manifest's config fields). */
  readonly settings: Readonly<Record<string, ConnectionValue>>;
  /** Secret connection values (sealed in ReputeCast, sent with each call). */
  readonly credentials: Readonly<Record<string, ConnectionValue>>;
  readonly webhookSecret: string | undefined;
  /** fetch on the adapter's base URL; see HttpClient. */
  readonly http: HttpClient;
  /** Aborts at the deadline. */
  readonly signal: AbortSignal;
  readonly deadline: Date;
  /** Never logs credentials. */
  readonly log: Logger;
}

export interface SendOutcome {
  /** The provider's id for the message (required). */
  providerMessageId: string;
}

/** A per-message refusal returned by sendBatch. */
export interface SendRefusal {
  error: { code?: string; message: string; temporary?: boolean; throttled?: boolean };
}

/** An event normalizeEvent returns; the SDK fills the defaults. */
export interface AdapterEvent {
  type: EventType;
  recipient: string;
  providerMessageId?: string;
  /** ReputeCast's message id when the provider echoes it. */
  messageId?: string;
  /** Default: when the webhook arrived. */
  occurredAt?: Date | string | number;
  /** The provider's event type; default: type. */
  originalType?: string;
  /** The provider's code ("550 5.1.1"); `code` is a shorthand. */
  originalCode?: string;
  code?: string | number;
  diagnostic?: string;
  /** Default for bounces: classifyBounce(code, diagnostic). */
  bounceClass?: BounceClass;
  url?: string;
  userAgent?: string;
  /** The provider's event, kept next to the normalized one (required). */
  raw: unknown;
}

export interface SuppressionPage {
  suppressions: Suppression[];
  nextCursor?: string | null;
}

export interface HttpSettings {
  /** The provider's API base URL (a function to read the connection or the environment). */
  baseUrl: string | ((ctx: Omit<AdapterContext, "http">) => string);
  /** Headers on every request, usually authentication. */
  headers?: (ctx: Omit<AdapterContext, "http">) => Record<string, string>;
  /** Reads the provider's error code from a failed answer (default: error.code, code). */
  errorCode?: (data: unknown, status: number) => string | undefined;
  /** Reads the provider's error message (default: error.message, message). */
  errorMessage?: (data: unknown, status: number) => string | undefined;
  /** The provider's name in error messages. */
  name?: string;
}

/** What defineAdapter takes. Hooks may be async. */
export interface AdapterDefinition<Raw = any> {
  /** The manifest's id. */
  id: string;
  capabilities: Capabilities;
  terms?: Terms;
  /** The adapter's version (health); default: the manifest's. */
  version?: string;
  http?: HttpSettings;
  /** Sends one message. Throw MessageRejected for a per-message refusal. */
  send?(msg: Message, ctx: AdapterContext): SendOutcome | Promise<SendOutcome>;
  /** Sends a whole batch through a batch API; one result per message, in order. */
  sendBatch?(msgs: Message[], ctx: AdapterContext): Array<SendOutcome | SendRefusal> | Promise<Array<SendOutcome | SendRefusal>>;
  /** How many send() calls run at once for a batch (5). */
  sendConcurrency?: number;
  /** Checks the credentials; may return what it learned about the account. */
  verify?(ctx: AdapterContext): Quota | void | Promise<Quota | void>;
  /** Checks a webhook's signature (required with webhooks). */
  verifyWebhook?(req: WebhookRequest, ctx: AdapterContext): boolean | Promise<boolean>;
  /** Answers a subscription handshake: return the body to send back, or undefined for events. */
  handshake?(req: WebhookRequest, ctx: AdapterContext): string | undefined | Promise<string | undefined>;
  /** Splits a webhook into raw events (default: a JSON array's items, or the object). */
  splitWebhook?(req: WebhookRequest, ctx: AdapterContext): Raw[] | Promise<Raw[]>;
  /** Maps one raw event (required with webhooks); null skips it. */
  normalizeEvent?(raw: Raw, ctx: AdapterContext): AdapterEvent | AdapterEvent[] | null | undefined;
  /** Registers ReputeCast's webhook URL at the provider (registerWebhooks). */
  registerWebhook?(url: string, ctx: AdapterContext): void | Promise<void>;
  /** Sets up a sending domain and returns its DNS records (domainSetup). */
  setupDomain?(domain: string, ctx: AdapterContext): DnsRecord[] | Promise<DnsRecord[]>;
  /** The domain's records and their status (default: setupDomain). */
  domainStatus?(domain: string, ctx: AdapterContext): DnsRecord[] | Promise<DnsRecord[]>;
  /** The account's allowance and usage (quotaDiscovery). */
  discoverQuota?(ctx: AdapterContext): Quota | Promise<Quota>;
  /** One page of the provider's suppression list (suppressionSync pull or two-way). */
  syncSuppressions?(req: { since: Date | undefined; cursor: string | undefined }, ctx: AdapterContext): SuppressionPage | Promise<SuppressionPage>;
  /** Adds addresses to the provider's suppression list (suppressionSync push or two-way). */
  pushSuppressions?(emails: string[], ctx: AdapterContext): void | Promise<void>;
}

/** An adapter checked by defineAdapter. */
export type Adapter<Raw = any> = Readonly<AdapterDefinition<Raw>> & {
  readonly capabilities: Capabilities;
  readonly normalized: NormalizedCapabilities;
};

const ADAPTER = Symbol.for("@reputecast/adapter-sdk/adapter");

export function isAdapter(v: unknown): v is Adapter {
  return typeof v === "object" && v !== null && (v as Record<symbol, unknown>)[ADAPTER] === true;
}

/**
 * Declares an adapter. It is checked when the process starts: the id, the
 * capabilities, and that every declared capability has its hook.
 */
export function defineAdapter<Raw = any>(def: AdapterDefinition<Raw>): Adapter<Raw> {
  const problems: string[] = [];
  if (typeof def.id !== "string" || !ID_PATTERN.test(def.id)) problems.push("id must be 2 to 40 characters: lowercase letters, digits and dashes, starting with a letter");
  problems.push(...capabilityProblems(def.capabilities));
  const caps = normalizeCapabilities(def.capabilities ?? {});
  const need = (ok: unknown, what: string) => {
    if (typeof ok !== "function") problems.push(what);
  };
  if (typeof def.send !== "function" && typeof def.sendBatch !== "function") problems.push("an adapter needs send() or sendBatch()");
  if (caps.webhooks) {
    need(def.verifyWebhook, "capabilities.webhooks needs verifyWebhook()");
    need(def.normalizeEvent, "capabilities.webhooks needs normalizeEvent()");
  }
  if (caps.registerWebhooks) need(def.registerWebhook, "capabilities.registerWebhooks needs registerWebhook()");
  if (caps.domainSetup) need(def.setupDomain, "capabilities.domainSetup needs setupDomain()");
  if (caps.quotaDiscovery) need(def.discoverQuota, "capabilities.quotaDiscovery needs discoverQuota()");
  if (caps.suppressionSync === "pull" || caps.suppressionSync === "two-way") need(def.syncSuppressions, `capabilities.suppressionSync "${caps.suppressionSync}" needs syncSuppressions()`);
  if (caps.suppressionSync === "push" || caps.suppressionSync === "two-way") need(def.pushSuppressions, `capabilities.suppressionSync "${caps.suppressionSync}" needs pushSuppressions()`);
  if (def.sendConcurrency !== undefined && (!Number.isInteger(def.sendConcurrency) || def.sendConcurrency < 1 || def.sendConcurrency > 100)) {
    problems.push("sendConcurrency must be a whole number from 1 to 100");
  }
  if (def.terms && !["allowed", "forbidden", "your_policy"].includes(def.terms.bulkMarketing)) problems.push("terms.bulkMarketing must be allowed, forbidden or your_policy");
  if (problems.length > 0) throw new ManifestError(problems.map((p) => `${def.id ?? "adapter"}: ${p}`));
  return Object.freeze({ ...def, normalized: caps, [ADAPTER]: true });
}
