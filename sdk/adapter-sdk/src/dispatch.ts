import { classifyBounce } from "./bounce.ts";
import type { Adapter, AdapterContext, AdapterEvent, SendOutcome, SendRefusal } from "./define.ts";
import { AdapterError, BadSignatureError, cleanText, isAdapterError, isMessageRejected, toWireError } from "./errors.ts";
import type { MessageRejected } from "./errors.ts";
import type {
  DnsRecord,
  DomainInput,
  HealthOutput,
  Message,
  Operation,
  Quota,
  RegisterInput,
  SendInput,
  SendOutput,
  SendResult,
  Suppression,
  SuppressionsAddInput,
  SuppressionsListInput,
  SuppressionsListOutput,
  WebhookParseInput,
  WebhookParseOutput,
  WebhookVerifyInput,
  WebhookVerifyOutput,
  WireEvent,
} from "./protocol.ts";
import { EVENT_TYPES, BOUNCE_CLASSES, LIMITS, PROTOCOL_VERSION } from "./protocol.ts";
import { SDK_VERSION } from "./version.ts";
import { WebhookRequest, splitJson } from "./webhooks.ts";

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const EMAIL = /^[^\s@]+@[^\s@]+$/;

function invalid(message: string): AdapterError {
  return new AdapterError("invalid_request", message);
}

function bug(message: string): AdapterError {
  return new AdapterError("internal", message);
}

function isRecord(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

function short(s: unknown, field: string): string | undefined {
  if (s === undefined || s === null || s === "") return undefined;
  const v = String(s);
  if (v.length > LIMITS.identifier) throw bug(`${field} is longer than ${LIMITS.identifier} characters`);
  return v;
}

/** Runs one operation for an authenticated, well-formed request. */
export async function dispatch(adapter: Adapter, op: Operation, input: unknown, ctx: AdapterContext, version: string): Promise<unknown> {
  switch (op) {
    case "health": {
      const out: HealthOutput = { adapter: adapter.id, version, protocol: PROTOCOL_VERSION, sdk: `@reputecast/adapter-sdk ${SDK_VERSION}`, capabilities: adapter.normalized };
      return out;
    }
    case "verify":
      return quotaOut(adapter.verify ? ((await adapter.verify(ctx)) ?? {}) : {});
    case "send":
      return send(adapter, input as SendInput, ctx);
    case "webhooks/verify":
      return webhookVerify(adapter, input as WebhookVerifyInput, ctx);
    case "webhooks/parse":
      return webhookParse(adapter, input as WebhookParseInput, ctx);
    case "webhooks/register": {
      const url = isRecord(input) ? input["url"] : undefined;
      if (typeof url !== "string" || !/^https?:\/\//.test(url)) throw invalid("webhooks/register needs the webhook url");
      await adapter.registerWebhook!((input as RegisterInput).url, ctx);
      return {};
    }
    case "domains/add":
    case "domains/status": {
      const domain = isRecord(input) ? input["domain"] : undefined;
      if (typeof domain !== "string" || domain === "" || domain.length > 253) throw invalid(`${op} needs a domain`);
      const hook = op === "domains/status" && adapter.domainStatus ? adapter.domainStatus : adapter.setupDomain!;
      return { records: records(await hook.call(adapter, (input as DomainInput).domain, ctx)) };
    }
    case "quota":
      return quotaOut(await adapter.discoverQuota!(ctx));
    case "suppressions/list":
      return suppressionsList(adapter, input as SuppressionsListInput, ctx);
    case "suppressions/add": {
      const emails = isRecord(input) ? input["emails"] : undefined;
      if (!Array.isArray(emails) || emails.length > LIMITS.suppressionsPerPush || emails.some((e) => typeof e !== "string")) {
        throw invalid(`suppressions/add takes at most ${LIMITS.suppressionsPerPush} addresses`);
      }
      await adapter.pushSuppressions!((input as SuppressionsAddInput).emails, ctx);
      return {};
    }
  }
}

function quotaOut(q: unknown): Quota {
  if (!isRecord(q)) throw bug("the quota or verify result must be an object");
  const out: Quota = {};
  if (q["kind"] !== undefined) {
    if (q["kind"] !== "rolling_24h" && q["kind"] !== "monthly_plan") throw bug(`quota kind ${JSON.stringify(q["kind"])} is not rolling_24h or monthly_plan`);
    out.kind = q["kind"];
  }
  for (const k of ["max", "used", "maxRate"] as const) {
    const v = q[k];
    if (v === undefined) continue;
    if (typeof v !== "number" || !Number.isFinite(v) || v < 0) throw bug(`quota ${k} must be a number of at least 0`);
    out[k] = v;
  }
  if (q["sandbox"] !== undefined) out.sandbox = q["sandbox"] === true;
  if (typeof q["account"] === "string" && q["account"] !== "") out.account = cleanText(q["account"]);
  return out;
}

function checkMessages(input: SendInput, batch: number): Message[] {
  const msgs = isRecord(input) ? input["messages"] : undefined;
  if (!Array.isArray(msgs) || msgs.length === 0) throw invalid("send needs messages");
  if (msgs.length > batch) throw invalid(`send got ${msgs.length} messages; the adapter declares batches of ${batch}`);
  for (const m of msgs) {
    if (!isRecord(m) || typeof m["id"] !== "string" || !isRecord(m["to"]) || typeof m["to"]["address"] !== "string" || !isRecord(m["from"])) {
      throw invalid("every message needs an id, from and to");
    }
  }
  return msgs as Message[];
}

function resultOf(messageId: string, r: SendOutcome | SendRefusal | MessageRejected | undefined): SendResult {
  if (isMessageRejected(r)) {
    return { messageId, error: { code: cleanText(r.code, 200), message: cleanText(r.message), temporary: r.temporary, throttled: r.throttled } };
  }
  if (isRecord(r) && isRecord(r["error"])) {
    const e = r["error"];
    const throttled = e["throttled"] === true;
    return {
      messageId,
      error: { code: cleanText(String(e["code"] ?? ""), 200), message: cleanText(String(e["message"] ?? "refused by the provider")), temporary: e["temporary"] === true || throttled, throttled },
    };
  }
  const id = isRecord(r) ? r["providerMessageId"] : undefined;
  if ((typeof id !== "string" && typeof id !== "number") || id === "") {
    // The message may have gone out: refusing it for good avoids a duplicate.
    return { messageId, error: { code: "adapter_bug", message: "The adapter returned no providerMessageId for this message.", temporary: false, throttled: false } };
  }
  return { messageId, providerMessageId: short(id, "providerMessageId")! };
}

/**
 * send: one call per message (sendConcurrency at a time) or the adapter's
 * batch API. A provider-wide failure before any message was accepted fails
 * the call; after one was accepted, the rest become temporary per-message
 * errors so a retry never sends twice.
 */
async function send(adapter: Adapter, input: SendInput, ctx: AdapterContext): Promise<SendOutput> {
  const msgs = checkMessages(input, adapter.normalized.batch);
  if (adapter.sendBatch) {
    const out = await adapter.sendBatch(msgs, ctx);
    if (!Array.isArray(out) || out.length !== msgs.length) throw bug(`sendBatch returned ${Array.isArray(out) ? out.length : "no"} results for ${msgs.length} messages`);
    return { results: msgs.map((m, i) => resultOf(m.id, out[i])) };
  }
  const results: (SendResult | undefined)[] = new Array(msgs.length).fill(undefined);
  let accepted = 0;
  let fatal: unknown;
  let next = 0;
  const worker = async () => {
    for (;;) {
      const i = next++;
      if (i >= msgs.length || fatal !== undefined) return;
      const m = msgs[i]!;
      try {
        const r = resultOf(m.id, await adapter.send!(m, ctx));
        results[i] = r;
        if (!r.error) accepted++;
      } catch (e) {
        if (isMessageRejected(e)) {
          results[i] = { messageId: m.id, error: { code: cleanText(e.code, 200), message: cleanText(e.message), temporary: e.temporary, throttled: e.throttled } };
        } else if (isAdapterError(e) && e.code === "provider_error") {
          results[i] = { messageId: m.id, error: { code: cleanText(e.providerCode ?? (e.status ? String(e.status) : "provider_error"), 200), message: cleanText(e.message), temporary: false, throttled: false } };
        } else if (fatal === undefined) {
          fatal = e;
        }
      }
    }
  };
  await Promise.all(Array.from({ length: Math.min(adapter.sendConcurrency ?? 5, msgs.length) }, worker));
  if (fatal !== undefined) {
    if (accepted === 0) throw fatal;
    const w = toWireError(fatal);
    msgs.forEach((m, i) => {
      results[i] ??= { messageId: m.id, error: { code: w.providerCode ?? w.code, message: w.message, temporary: true, throttled: w.code === "provider_throttled" } };
    });
  }
  return { results: results as SendResult[] };
}

function webhookRequest(input: unknown): WebhookRequest {
  const r = isRecord(input) ? input["request"] : undefined;
  if (!isRecord(r) || typeof r["body"] !== "string" || !isRecord(r["headers"])) throw invalid("the forwarded webhook request is missing");
  return WebhookRequest.fromWire(r as unknown as WebhookVerifyInput["request"]);
}

async function webhookVerify(adapter: Adapter, input: WebhookVerifyInput, ctx: AdapterContext): Promise<WebhookVerifyOutput> {
  const req = webhookRequest(input);
  if ((await adapter.verifyWebhook!(req, ctx)) !== true) throw new BadSignatureError();
  if (adapter.handshake) {
    const body = await adapter.handshake(req, ctx);
    if (body !== undefined) {
      if (Buffer.byteLength(body) > LIMITS.handshake) throw bug("the handshake answer is larger than 16 KiB");
      return { handshake: { body } };
    }
  }
  return {};
}

async function webhookParse(adapter: Adapter, input: WebhookParseInput, ctx: AdapterContext): Promise<WebhookParseOutput> {
  const req = webhookRequest(input);
  const verified = isRecord(input) && input["verified"] === true;
  if (!verified && (await adapter.verifyWebhook!(req, ctx)) !== true) throw new BadSignatureError();
  let raws: unknown[];
  try {
    raws = adapter.splitWebhook ? await adapter.splitWebhook(req, ctx) : splitJson(req);
  } catch (e) {
    if (isAdapterError(e)) throw e;
    throw invalid("The webhook body can't be read: " + (e instanceof Error ? e.message : String(e)));
  }
  const receivedAt = new Date();
  const events: WireEvent[] = [];
  for (const raw of raws) {
    const out = await adapter.normalizeEvent!(raw, ctx);
    for (const e of out === null || out === undefined ? [] : Array.isArray(out) ? out : [out]) {
      events.push(wireEvent(e, raw, receivedAt, ctx));
    }
  }
  return { events };
}

/** Fills an event's defaults and checks it against the protocol. */
export function wireEvent(e: AdapterEvent, raw: unknown, receivedAt: Date, ctx?: Pick<AdapterContext, "log">): WireEvent {
  if (!isRecord(e)) throw bug("normalizeEvent must return an event, a list of events or null");
  if (!EVENT_TYPES.includes(e.type)) throw bug(`normalizeEvent returned the unknown type ${JSON.stringify(e.type)} (use ${EVENT_TYPES.join(", ")})`);
  const recipient = typeof e.recipient === "string" ? e.recipient.trim().toLowerCase() : "";
  if (recipient === "") throw bug("normalizeEvent returned an event without a recipient");
  let at = receivedAt;
  if (e.occurredAt !== undefined && e.occurredAt !== null && e.occurredAt !== "") {
    at = e.occurredAt instanceof Date ? e.occurredAt : new Date(typeof e.occurredAt === "number" && e.occurredAt < 1e12 ? e.occurredAt * 1000 : e.occurredAt);
    if (Number.isNaN(at.getTime())) throw bug(`normalizeEvent returned the time ${JSON.stringify(e.occurredAt)}, which is not a date`);
  }
  const code = e.originalCode ?? (e.code === undefined || e.code === null ? undefined : String(e.code));
  const out: WireEvent = { type: e.type, recipient: short(recipient, "recipient")!, occurredAt: at.toISOString(), originalType: cleanText(e.originalType ?? e.type, 200), raw: e.raw ?? raw };
  const pmid = short(e.providerMessageId, "providerMessageId");
  if (pmid) out.providerMessageId = pmid;
  if (e.messageId) {
    if (UUID.test(e.messageId)) out.messageId = e.messageId.toLowerCase();
    else ctx?.log.debug("dropped a messageId that is not a ReputeCast message id", { messageId: e.messageId });
  }
  if (code) out.originalCode = cleanText(code, 200);
  if (e.diagnostic) out.diagnostic = cleanText(e.diagnostic);
  if (e.bounceClass !== undefined) {
    if (!BOUNCE_CLASSES.includes(e.bounceClass)) throw bug(`normalizeEvent returned the unknown bounceClass ${JSON.stringify(e.bounceClass)}`);
    out.bounceClass = e.bounceClass;
  } else if (e.type === "hard_bounce" || e.type === "soft_bounce") {
    out.bounceClass = classifyBounce(code ?? "", e.diagnostic ?? "");
  }
  const url = short(e.url, "url");
  if (url) out.url = url;
  if (e.userAgent) out.userAgent = cleanText(e.userAgent, 500);
  if (out.raw === undefined) throw bug("normalizeEvent must keep the provider's raw event");
  return out;
}

const PURPOSES = ["dkim", "spf", "return_path", "dmarc", "verification"];
const TYPES = ["CNAME", "TXT", "MX"];
const STATUSES = [undefined, "pending", "verified", "wrong_value", "missing"];

function records(rs: DnsRecord[]): DnsRecord[] {
  if (!Array.isArray(rs)) throw bug("setupDomain and domainStatus must return a list of DNS records");
  return rs.map((r, i) => {
    const type = typeof r?.type === "string" ? (r.type.toUpperCase() as DnsRecord["type"]) : r?.type;
    if (!isRecord(r) || !PURPOSES.includes(r.purpose) || !TYPES.includes(type) || !STATUSES.includes(r.status) || !r.name || !r.value) {
      throw bug(`DNS record ${i} needs a purpose (${PURPOSES.join(", ")}), a type (${TYPES.join(", ")}), a name and a value`);
    }
    const out: DnsRecord = { purpose: r.purpose, type, name: short(r.name, "record name")!, value: String(r.value) };
    if (r.status) out.status = r.status;
    return out;
  });
}

const REASONS = ["hard_bounce", "complaint", "unsubscribe", "manual"];

async function suppressionsList(adapter: Adapter, input: SuppressionsListInput, ctx: AdapterContext): Promise<SuppressionsListOutput> {
  const since = isRecord(input) && typeof input["since"] === "string" ? new Date(input["since"]) : undefined;
  if (since && Number.isNaN(since.getTime())) throw invalid("since is not a time");
  const cursor = isRecord(input) && typeof input["cursor"] === "string" && input["cursor"] !== "" ? input["cursor"] : undefined;
  const page = await adapter.syncSuppressions!({ since, cursor }, ctx);
  if (!isRecord(page) || !Array.isArray(page.suppressions)) throw bug("syncSuppressions must return { suppressions, nextCursor }");
  if (page.suppressions.length > LIMITS.suppressionsPerPage) throw bug(`syncSuppressions returned ${page.suppressions.length} addresses; pages hold at most ${LIMITS.suppressionsPerPage}`);
  const suppressions = page.suppressions.map((s: Suppression, i: number): Suppression => {
    const email = typeof s?.email === "string" ? s.email.trim().toLowerCase() : "";
    if (!EMAIL.test(email) || email.length > LIMITS.identifier) throw bug(`suppression ${i} ${JSON.stringify(s?.email)} is not an email address`);
    if (!REASONS.includes(s.reason)) throw bug(`suppression ${i} has the reason ${JSON.stringify(s.reason)} (use ${REASONS.join(", ")})`);
    const out: Suppression = { email, reason: s.reason };
    if (s.code) out.code = cleanText(String(s.code), 200);
    if (s.createdAt) {
      const t = new Date(s.createdAt);
      if (Number.isNaN(t.getTime())) throw bug(`suppression ${i} createdAt is not a time`);
      out.createdAt = t.toISOString();
    }
    return out;
  });
  const nextCursor = page.nextCursor === undefined || page.nextCursor === null || page.nextCursor === "" ? null : String(page.nextCursor);
  if (nextCursor !== null && nextCursor === cursor) throw bug("syncSuppressions returned the cursor it was given as nextCursor");
  return { suppressions, nextCursor };
}
