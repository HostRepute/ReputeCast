import { randomUUID } from "node:crypto";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";

import type { Adapter } from "../define.ts";
import { capabilityDifferences, normalizeCapabilities, operationsFor, validateManifest } from "../manifest.ts";
import type { Manifest } from "../manifest.ts";
import type {
  Connection,
  ConnectionValue,
  EventType,
  HealthOutput,
  Message,
  Operation,
  Quota,
  ResponseEnvelope,
  SendOutput,
  SendResult,
  SuppressionsListOutput,
  WebhookParseOutput,
  WebhookVerifyOutput,
  WireEvent,
  WireWebhookRequest,
} from "../protocol.ts";
import { OPERATIONS } from "../protocol.ts";
import { serve } from "../server.ts";
import { newSecret } from "../signing.ts";
import { ProtocolClient } from "./client.ts";

export { ProtocolClient } from "./client.ts";
export type { CallOptions, CallResult } from "./client.ts";

/** A recipient the fake API must refuse permanently (the same as the Go suite's). */
export const REJECT_ADDRESS = "rejected@contract.test";

/** The failure the fake API simulates: "" works, "outage" fails every call with 5xx, "auth" rejects the credentials. */
export type Mode = "" | "outage" | "auth";

/** A request the fake provider API received. */
export interface FakeRequest {
  method: string;
  path: string;
  query: URLSearchParams;
  /** Lowercase names. */
  headers: Record<string, string>;
  body: string;
  json<T = any>(): T;
}

export interface FakeReply {
  status?: number;
  json?: unknown;
  body?: string;
  headers?: Record<string, string>;
}

/** Shared between the runner and the fake API. */
export interface FakeEnv {
  /** The fake API's base URL. */
  readonly url: string;
  mode: Mode;
  readonly requests: FakeRequest[];
}

/** A recorded webhook, built per run (signatures with fresh timestamps). */
export interface WebhookFixture {
  method?: string;
  headers: Record<string, string>;
  body: string;
}

/** An event a recorded webhook must produce (compared like the Go suite does). */
export interface ExpectedEvent {
  type: EventType;
  recipient: string;
  providerMessageId?: string;
  messageId?: string;
  originalType?: string;
  originalCode?: string;
  bounceClass?: string;
}

export interface WebhookCase {
  name: string;
  request(env: FakeEnv): WebhookFixture;
  want: ExpectedEvent[];
}

/** What the runner needs to exercise an adapter (reputecast.contract.ts). */
export interface Contract {
  adapter: Adapter;
  /** manifest.json. */
  manifest: unknown;
  /**
   * A fake of the provider's API: refuses REJECT_ADDRESS permanently, and in
   * env.mode "outage" fails every call with a 5xx, in "auth" with a 401.
   */
  api(env: FakeEnv): (req: FakeRequest) => FakeReply | Promise<FakeReply>;
  /** The connection values a person would enter. */
  connection(env: FakeEnv): { settings?: Record<string, ConnectionValue>; credentials?: Record<string, ConnectionValue>; webhookSecret?: string };
  /** Environment variables for the adapter process (point it at env.url). */
  environment?(env: FakeEnv): Record<string, string>;
  /** Extra checks that a send reached the provider as it should. */
  delivered?(env: FakeEnv, msgs: Message[], results: SendResult[]): void | Promise<void>;
  webhooks?: WebhookCase[];
  /** Breaks a valid webhook; the default flips a byte of the body. */
  tamper?(fixture: WebhookFixture): WebhookFixture;
}

export function defineContract(c: Contract): Contract {
  return c;
}

/** A test message with the headers the sending pipeline sets (the Go suite's Message(i)). */
export function testMessage(i: number): Message {
  const id = randomUUID();
  return {
    id,
    stream: "broadcast",
    from: { name: "Acme Newsletter", address: "news@news.acme.io" },
    to: { name: "Contract Test", address: `reader${String.fromCharCode(97 + i)}@contract.test` },
    subject: `Contract test ${id}`,
    html: "<p>Hello from the contract suite.</p>",
    text: "Hello from the contract suite.",
    headers: {
      "List-Unsubscribe": "<https://news.acme.io/p/acme/unsubscribe/abc>, <mailto:unsubscribe@news.acme.io>",
      "List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
      "Feedback-ID": "contract:acme:reputecast",
      "X-ReputeCast-Message": id,
    },
    tags: { campaign: "contract" },
  };
}

export interface ScenarioResult {
  name: string;
  status: "pass" | "fail" | "skip";
  /** Why it failed or was skipped. */
  detail?: string;
  ms: number;
}

export interface ContractReport {
  passed: number;
  failed: number;
  skipped: number;
  results: ScenarioResult[];
}

class Failure extends Error {}
class Skip extends Error {}

function fail(msg: string): never {
  throw new Failure(msg);
}

function skip(msg: string): never {
  throw new Skip(msg);
}

function show(v: unknown): string {
  const s = JSON.stringify(v);
  return s === undefined ? String(v) : s.length > 600 ? s.slice(0, 600) + "…" : s;
}

async function fakeApi(c: Contract): Promise<{ env: FakeEnv; close(): Promise<void> }> {
  const requests: FakeRequest[] = [];
  const env = { url: "", mode: "" as Mode, requests };
  let handler: ((req: FakeRequest) => FakeReply | Promise<FakeReply>) | undefined;
  const server = createServer((req, res) => {
    const chunks: Buffer[] = [];
    req.on("data", (b: Buffer) => chunks.push(b));
    req.on("end", () => {
      const url = new URL(req.url ?? "/", "http://fake");
      const headers: Record<string, string> = {};
      for (const [k, v] of Object.entries(req.headers)) headers[k] = Array.isArray(v) ? v.join(", ") : (v ?? "");
      const body = Buffer.concat(chunks).toString("utf8");
      const fr: FakeRequest = { method: req.method ?? "GET", path: url.pathname, query: url.searchParams, headers, body, json: () => JSON.parse(body) };
      requests.push(fr);
      Promise.resolve()
        .then(() => (handler ? handler(fr) : { status: 500 }))
        .then((r) => {
          const out = r.json !== undefined ? JSON.stringify(r.json) : (r.body ?? "");
          res.writeHead(r.status ?? 200, { "content-type": r.json !== undefined ? "application/json" : "text/plain", ...r.headers });
          res.end(out);
        })
        .catch((e: unknown) => {
          res.writeHead(500, { "content-type": "text/plain" });
          res.end(String(e));
        });
    });
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  env.url = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
  handler = c.api(env);
  return {
    env,
    close: () =>
      new Promise<void>((resolve) => {
        server.close(() => resolve());
        server.closeAllConnections();
      }),
  };
}

function flipByte(f: WebhookFixture): WebhookFixture {
  const b = Buffer.from(f.body);
  if (b.length > 0) {
    const i = Math.floor(b.length / 2);
    b[i] = (b[i] as number) ^ 0x01;
    if (b[i] === 0x22 || b[i] === 0x5c) b[i] = (b[i] as number) ^ 0x03;
  }
  return { ...f, body: b.toString("utf8") };
}

function wire(id: string, f: WebhookFixture, drop: string[] = []): WireWebhookRequest {
  const headers: Record<string, string[]> = {};
  for (const [k, v] of Object.entries(f.headers)) {
    if (!drop.includes(k.toLowerCase())) headers[k.toLowerCase()] = [v];
  }
  return { method: f.method ?? "POST", url: `https://reputecast.example/hooks/community:${id}/contract-token-0123456789`, headers, body: Buffer.from(f.body).toString("base64") };
}

function compareEvents(got: WireEvent[], want: ExpectedEvent[]): void {
  if (got.length !== want.length) fail(`got ${got.length} events, want ${want.length}: ${show(got)}`);
  want.forEach((w, i) => {
    const g = got[i] as WireEvent;
    const same = (a: unknown, b: unknown) => (a ?? "") === (b ?? "");
    if (
      !same(g.type, w.type) ||
      !same(g.recipient, w.recipient) ||
      !same(g.providerMessageId, w.providerMessageId) ||
      !same(g.originalType, w.originalType) ||
      !same(g.originalCode, w.originalCode) ||
      !same(g.bounceClass, w.bounceClass) ||
      !same(g.messageId, w.messageId)
    ) {
      const { raw: _raw, ...rest } = g;
      fail(`event ${i}:\n  got  ${show(rest)}\n  want ${show(w)}`);
    }
    if (!g.occurredAt || Number.isNaN(Date.parse(g.occurredAt))) fail(`event ${i} has no time`);
    if (g.raw === undefined || g.raw === null) fail(`event ${i} does not keep the raw payload`);
  });
}

function expectOK<R>(op: string, r: { status: number; body: ResponseEnvelope<R>; signed: boolean }): R {
  if (!r.signed) fail(`the ${op} answer is not signed with the adapter's secret`);
  if (r.status !== 200 || r.body.error || r.body.result === undefined) fail(`${op} answered ${r.status}: ${show(r.body.error ?? r.body)}`);
  return r.body.result as R;
}

function expectError(op: string, r: { status: number; body: ResponseEnvelope; signed: boolean }, code: string): void {
  if (r.body.error?.code !== code) fail(`${op}: want error ${code}, got ${r.status} ${show(r.body.error ?? r.body.result)}`);
}

/**
 * Runs the contract suite against an adapter over the protocol: the
 * scenarios of the Go suite (internal/providers/contracttest) plus the
 * protocol's own (signatures, idempotency, declared operations).
 */
export async function runContract(c: Contract, o: { log?: (line: string) => void } = {}): Promise<ContractReport> {
  const log = o.log ?? (() => {});
  const results: ScenarioResult[] = [];
  const api = await fakeApi(c);
  const env = api.env;
  const saved: Record<string, string | undefined> = {};
  const vars = c.environment?.(env) ?? {};
  for (const [k, v] of Object.entries(vars)) {
    saved[k] = process.env[k];
    process.env[k] = v;
  }
  const secret = newSecret();
  let server: Awaited<ReturnType<typeof serve>> | undefined;
  const record = (r: ScenarioResult) => {
    results.push(r);
    const mark = r.status === "pass" ? "✓" : r.status === "fail" ? "✗" : "-";
    log(`${mark} ${r.name}${r.status === "pass" ? ` (${r.ms} ms)` : r.detail ? `: ${r.detail}` : ""}`);
  };
  try {
    let manifest: Manifest | undefined;
    const started = Date.now();
    try {
      manifest = validateManifest(c.manifest);
      if (manifest.id !== c.adapter.id) fail(`manifest.json has the id ${manifest.id}, the code ${c.adapter.id}`);
      const diff = capabilityDifferences(c.adapter.capabilities, manifest.capabilities);
      if (diff.length > 0) fail("the code and manifest.json declare different capabilities: " + diff.join("; "));
      if (c.adapter.terms && c.adapter.terms.bulkMarketing !== manifest.terms.bulkMarketing) fail("the code and manifest.json declare different terms");
      server = await serve(c.adapter, { port: 0, host: "127.0.0.1", secret, manifest: c.manifest, allowHosts: ["127.0.0.1", "localhost"], logger: false });
      record({ name: "Registration", status: "pass", ms: Date.now() - started });
    } catch (e) {
      record({ name: "Registration", status: "fail", detail: e instanceof Error ? e.message : String(e), ms: Date.now() - started });
      return summarize(results);
    }
    const client = new ProtocolClient({ url: server.url, secret, adapter: c.adapter.id });
    const caps = normalizeCapabilities(c.adapter.capabilities);
    const declared = new Set<Operation>(operationsFor(c.adapter.capabilities));
    const connection = (): Connection => {
      const v = c.connection(env);
      const out: Connection = { providerId: randomUUID(), workspaceId: randomUUID(), settings: v.settings ?? {}, credentials: v.credentials ?? {} };
      if (v.webhookSecret !== undefined) out.webhookSecret = v.webhookSecret;
      return out;
    };
    const scenario = async (name: string, fn: () => Promise<void>) => {
      env.mode = "";
      env.requests.length = 0;
      const t = Date.now();
      try {
        await fn();
        record({ name, status: "pass", ms: Date.now() - t });
      } catch (e) {
        if (e instanceof Skip) record({ name, status: "skip", detail: e.message, ms: Date.now() - t });
        else record({ name, status: "fail", detail: e instanceof Error ? e.message : String(e), ms: Date.now() - t });
      }
    };

    await scenario("Health", async () => {
      const h = expectOK<HealthOutput>("health", await client.call("health", {}));
      if (h.adapter !== c.adapter.id || h.protocol !== 1) fail(`health answered ${show(h)}`);
      if (manifest && h.version !== manifest.version) fail(`health reports version ${h.version}, manifest.json ${manifest.version}`);
    });

    await scenario("Authentication", async () => {
      const conn = connection();
      expectError("an unsigned request", await client.call("verify", {}, { connection: conn, unsigned: true }), "unauthenticated");
      expectError("a request signed with another secret", await client.call("verify", {}, { connection: conn, secret: newSecret() }), "unauthenticated");
      expectError("a request signed 10 minutes ago", await client.call("verify", {}, { connection: conn, timestampOffsetSeconds: -600 }), "unauthenticated");
      expectError("a body for another operation", await client.call("verify", {}, { connection: conn, operation: "send" }), "invalid_request");
      expectError("a request for another adapter", await client.call("verify", {}, { connection: conn, adapter: "another-adapter" }), "invalid_request");
      const undeclared = OPERATIONS.find((op) => !declared.has(op));
      if (undeclared) expectError(`${undeclared} (not declared)`, await client.call(undeclared, {}, { connection: conn }), "not_supported");
    });

    await scenario("Send", async () => {
      const n = Math.min(3, Math.max(caps.batch, 1));
      const msgs = Array.from({ length: n }, (_, i) => testMessage(i));
      const out = expectOK<SendOutput>("send", await client.call("send", { messages: msgs }, { connection: connection() }));
      if (out.results?.length !== n) fail(`send returned ${out.results?.length} results for ${n} messages`);
      out.results.forEach((r, i) => {
        if (r.messageId !== msgs[i]!.id) fail(`result ${i} is for ${r.messageId}, want ${msgs[i]!.id} (results must keep the input order)`);
        if (r.error) fail(`message ${i} failed: ${show(r.error)}`);
        if (!r.providerMessageId) fail(`message ${i} has no provider message id`);
      });
      await c.delivered?.(env, msgs, out.results);
    });

    await scenario("SendRejected", async () => {
      const m = testMessage(0);
      m.to = { name: "", address: REJECT_ADDRESS };
      const r = await client.call<SendOutput>("send", { messages: [m] }, { connection: connection() });
      if (r.body.error) fail(`a rejected recipient must not fail the whole call: ${show(r.body.error)}`);
      const res = expectOK<SendOutput>("send", r).results?.[0];
      if (!res?.error) fail(`want a per-message error, got ${show(res)}`);
      if (res.error.temporary) fail(`a permanent rejection was reported as temporary: ${show(res.error)}`);
      if (!res.error.code && !res.error.message) fail("the rejection must keep the provider's code or message");
    });

    await scenario("SendOutage", async () => {
      env.mode = "outage";
      const r = await client.call<SendOutput>("send", { messages: [testMessage(0)] }, { connection: connection() });
      if (!r.signed) fail("the answer is not signed");
      const res = r.body.result?.results?.[0];
      if (r.body.error?.code === "provider_unavailable" || (res?.error && res.error.temporary)) return;
      fail(`an outage must fail the call with provider_unavailable or the message temporarily, got ${r.status} ${show(r.body.error ?? r.body.result)}`);
    });

    await scenario("SendAuth", async () => {
      env.mode = "auth";
      const r = await client.call("send", { messages: [testMessage(0)] }, { connection: connection() });
      if (!r.signed) fail("the answer is not signed");
      expectError("send with rejected credentials", r, "provider_auth");
    });

    await scenario("SendIdempotent", async () => {
      const msgs = Array.from({ length: Math.min(2, caps.batch) }, (_, i) => testMessage(i));
      const conn = connection();
      const key = "send-contract-" + randomUUID().replaceAll("-", "");
      const first = expectOK<SendOutput>("send", await client.call("send", { messages: msgs }, { connection: conn, key }));
      const calls = env.requests.length;
      const again = expectOK<SendOutput>("send", await client.call("send", { messages: msgs }, { connection: conn, key }));
      if (JSON.stringify(first) !== JSON.stringify(again)) fail(`a retry with the same idempotency key must get the same answer:\n  first ${show(first)}\n  retry ${show(again)}`);
      if (env.requests.length !== calls) fail(`a retry with the same idempotency key reached the provider again (${env.requests.length - calls} more requests): messages would be sent twice`);
    });

    await scenario("Verify", async () => {
      const q = expectOK<Quota>("verify", await client.call("verify", {}, { connection: connection() }));
      if (typeof q !== "object" || q === null) fail(`verify must answer an object, got ${show(q)}`);
      if (c.adapter.verify) {
        env.mode = "auth";
        expectError("verify with rejected credentials", await client.call("verify", {}, { connection: connection() }), "provider_auth");
      }
    });

    await scenario("Quota", async () => {
      if (!caps.quotaDiscovery) skip("quotaDiscovery isn't declared");
      const q = expectOK<Quota>("quota", await client.call("quota", {}, { connection: connection() }));
      if (q.kind !== "rolling_24h" && q.kind !== "monthly_plan") fail(`quota kind ${show(q.kind)} is not rolling_24h or monthly_plan`);
      if (typeof q.max !== "number" || typeof q.used !== "number") fail(`quota needs max and used: ${show(q)}`);
      if (caps.quotaDiscovery === "rate" && !(typeof q.maxRate === "number" && q.maxRate > 0)) fail(`quotaDiscovery "rate" needs a maxRate: ${show(q)}`);
      env.mode = "outage";
      expectError("quota during an outage", await client.call("quota", {}, { connection: connection() }), "provider_unavailable");
    });

    await scenario("SuppressionsPull", async () => {
      if (caps.suppressionSync !== "pull" && caps.suppressionSync !== "two-way") skip("suppressionSync doesn't pull");
      const conn = connection();
      let cursor: string | null = null;
      let total = 0;
      for (let page = 0; page < 10; page++) {
        const out: SuppressionsListOutput = expectOK<SuppressionsListOutput>("suppressions/list", await client.call("suppressions/list", { since: null, cursor }, { connection: conn }));
        if (!Array.isArray(out.suppressions)) fail(`suppressions/list must answer { suppressions, nextCursor }: ${show(out)}`);
        total += out.suppressions.length;
        if (!out.nextCursor) break;
        cursor = out.nextCursor;
      }
      if (total === 0) fail("the fake API's suppression list should hold at least one address to check the mapping");
    });

    for (const w of c.webhooks ?? []) {
      await scenario(`Webhook/${w.name}`, async () => {
        if (!caps.webhooks) fail("webhook cases need capabilities.webhooks");
        const conn = connection();
        const f = w.request(env);
        const v = expectOK<WebhookVerifyOutput>("webhooks/verify", await client.call("webhooks/verify", { request: wire(c.adapter.id, f) }, { connection: conn }));
        if (v.handshake) fail("an event was answered as a handshake");
        const parsed = expectOK<WebhookParseOutput>("webhooks/parse", await client.call("webhooks/parse", { request: wire(c.adapter.id, f), verified: false }, { connection: conn }));
        compareEvents(parsed.events ?? [], w.want);
        // The normalization job parses the stored request later, without the
        // headers ReputeCast never stores: a verified flag must be enough.
        const again = expectOK<WebhookParseOutput>(
          "webhooks/parse (verified)",
          await client.call("webhooks/parse", { request: wire(c.adapter.id, f, ["authorization", "cookie"]), verified: true }, { connection: conn }),
        );
        compareEvents(again.events ?? [], w.want);
        const bad = (c.tamper ?? flipByte)({ ...f, headers: { ...f.headers } });
        expectError("webhooks/verify of a tampered request", await client.call("webhooks/verify", { request: wire(c.adapter.id, bad) }, { connection: conn }), "bad_signature");
        expectError("webhooks/parse of a tampered request", await client.call("webhooks/parse", { request: wire(c.adapter.id, bad), verified: false }, { connection: conn }), "bad_signature");
      });
    }
    if (caps.webhooks && (c.webhooks ?? []).length === 0) {
      record({ name: "Webhooks", status: "fail", detail: "capabilities.webhooks needs recorded webhooks in the contract (webhooks: [...])", ms: 0 });
    }
  } finally {
    await server?.close();
    await api.close();
    for (const [k, v] of Object.entries(saved)) {
      if (v === undefined) delete process.env[k];
      else process.env[k] = v;
    }
  }
  return summarize(results);
}

function summarize(results: ScenarioResult[]): ContractReport {
  return {
    passed: results.filter((r) => r.status === "pass").length,
    failed: results.filter((r) => r.status === "fail").length,
    skipped: results.filter((r) => r.status === "skip").length,
    results,
  };
}
