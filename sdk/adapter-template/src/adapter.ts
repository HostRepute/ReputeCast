import { defineAdapter, parseSignatureHeader, verifyTimestampedHmac } from "@reputecast/adapter-sdk";
import type { AdapterContext, AdapterEvent, EventType, Message, Quota, Suppression } from "@reputecast/adapter-sdk";

/**
 * Example ESP is a fictional provider with a typical API: Bearer
 * authentication, one message per POST /v1/messages, signed webhooks,
 * a suppression list and account usage. Replace these mappings with your
 * provider's.
 */
const API_URL = "https://api.example-esp.com";

/** A webhook event as Example ESP posts it (one object, or an array of them). */
export interface ExampleEvent {
  id: string;
  event: "delivered" | "deferred" | "bounced" | "complained" | "unsubscribed" | "opened" | "clicked";
  timestamp: string;
  message_id: string;
  recipient: string;
  smtp_code?: string;
  description?: string;
  bounce_type?: "hard" | "soft";
  url?: string;
  user_agent?: string;
  metadata?: Record<string, string>;
}

/** Example ESP's event names and ReputeCast's normalized types. */
const map: Record<Exclude<ExampleEvent["event"], "bounced">, EventType> = {
  delivered: "delivered",
  deferred: "deferred",
  complained: "complaint",
  unsubscribed: "unsubscribe",
  opened: "open",
  clicked: "click",
};

/** The request body of POST /v1/messages. */
export function toPayload(msg: Message) {
  return {
    from: { email: msg.from.address, name: msg.from.name },
    to: [{ email: msg.to.address, name: msg.to.name }],
    ...(msg.replyTo ? { reply_to: { email: msg.replyTo.address, name: msg.replyTo.name } } : {}),
    subject: msg.subject,
    html: msg.html,
    text: msg.text,
    // Every header ReputeCast sets, List-Unsubscribe and List-Unsubscribe-Post included.
    headers: msg.headers,
    // ReputeCast's message id comes back on every event.
    metadata: { ...msg.tags, rc_message: msg.id },
    // ReputeCast tracks opens and clicks itself: no double tracking.
    tracking: { opens: false, clicks: false },
  };
}

interface Account {
  plan: string;
  monthly_limit: number;
  sent_this_month: number;
  sandbox: boolean;
}

async function account(ctx: AdapterContext): Promise<Quota> {
  const r = await ctx.http.get<Account>("/v1/account");
  const a = r.data;
  return {
    kind: "monthly_plan",
    max: a.monthly_limit,
    used: a.sent_this_month,
    sandbox: a.sandbox,
    account: `${a.plan} · ${a.monthly_limit.toLocaleString("en-US")} per month${a.sandbox ? " · sandbox" : ""}`,
  };
}

const reasons: Record<string, Suppression["reason"]> = { bounce: "hard_bounce", complaint: "complaint", unsubscribe: "unsubscribe" };

export default defineAdapter<ExampleEvent>({
  id: "example-esp",
  capabilities: { webhooks: true, batch: 100, suppressionSync: "pull", quotaDiscovery: true },
  terms: { bulkMarketing: "allowed" },
  http: {
    name: "Example ESP",
    // The operator can point the process at a regional endpoint or, in tests, at a fake.
    baseUrl: () => process.env["EXAMPLE_ESP_API_URL"] ?? API_URL,
    headers: (ctx) => ({ authorization: `Bearer ${String(ctx.credentials["api_key"] ?? "")}` }),
  },

  // Called for every message of a batch. A 4xx from ctx.http (an invalid or
  // suppressed recipient) becomes a per-message refusal; 401/403, 429 and
  // 5xx fail the call as provider_auth, provider_throttled and
  // provider_unavailable.
  async send(msg, ctx) {
    const r = await ctx.http.post<{ id: string }>("/v1/messages", toPayload(msg));
    return { providerMessageId: r.data.id };
  },

  async verify(ctx) {
    return account(ctx);
  },

  discoverQuota: account,

  // X-Example-Signature: t=<unix seconds>,v1=<hex HMAC-SHA256 of "<t>.<body>">
  verifyWebhook(req, ctx) {
    const sig = parseSignatureHeader(req.header("x-example-signature"));
    return verifyTimestampedHmac({
      secret: String(ctx.credentials["webhook_signing_key"] ?? ""),
      timestamp: sig["t"]?.[0],
      signature: sig["v1"] ?? [],
      payload: req.body,
    });
  },

  normalizeEvent(raw): AdapterEvent | null {
    const type = raw.event === "bounced" ? (raw.bounce_type === "hard" ? "hard_bounce" : "soft_bounce") : map[raw.event];
    if (!type) return null;
    return {
      type,
      code: raw.smtp_code,
      raw,
      recipient: raw.recipient,
      providerMessageId: raw.message_id,
      messageId: raw.metadata?.["rc_message"],
      occurredAt: raw.timestamp,
      originalType: raw.bounce_type ? `${raw.event} · ${raw.bounce_type}` : raw.event,
      ...(raw.description ? { diagnostic: raw.description } : {}),
      ...(raw.url ? { url: raw.url } : {}),
      ...(raw.user_agent ? { userAgent: raw.user_agent } : {}),
    };
  },

  async syncSuppressions({ since, cursor }, ctx) {
    const r = await ctx.http.get<{ data: { email: string; reason: string; created_at: string }[]; next_cursor: string | null }>("/v1/suppressions", {
      query: { since: since?.toISOString(), cursor, limit: 1000 },
    });
    return {
      suppressions: r.data.data.map((s) => ({ email: s.email, reason: reasons[s.reason] ?? "manual", code: s.reason, createdAt: s.created_at })),
      nextCursor: r.data.next_cursor,
    };
  },
});
