import { afterEach, describe, expect, it } from "vitest";

import { defineAdapter } from "../src/define.ts";
import type { AdapterDefinition } from "../src/define.ts";
import { MessageRejected, ProviderError } from "../src/errors.ts";
import { ManifestError } from "../src/manifest.ts";
import type { Connection, SendOutput, WebhookParseOutput, WebhookVerifyOutput } from "../src/protocol.ts";
import { serve } from "../src/server.ts";
import type { AdapterServer } from "../src/server.ts";
import { newSecret } from "../src/signing.ts";
import { ProtocolClient, testMessage } from "../src/testing/index.ts";
import { manifest } from "./fixtures.ts";

const servers: AdapterServer[] = [];
afterEach(async () => {
  await Promise.all(servers.splice(0).map((s) => s.close()));
});

const conn: Connection = { providerId: "p-1", workspaceId: "w-1", settings: {}, credentials: { api_key: "k" } };

async function start(def: Partial<AdapterDefinition>, opts: { manifest?: unknown } = {}) {
  const adapter = defineAdapter({ id: "fake-esp", capabilities: { batch: 10 }, send: async (m) => ({ providerMessageId: "pm-" + m.id }), ...def });
  const secret = newSecret();
  const s = await serve(adapter, { port: 0, host: "127.0.0.1", secret, logger: false, ...opts });
  servers.push(s);
  return { client: new ProtocolClient({ url: s.url, secret, adapter: "fake-esp" }), secret, url: s.url };
}

describe("serve", () => {
  it("answers health, signed", async () => {
    const { client } = await start({}, {});
    const r = await client.call("health", {});
    expect(r.status).toBe(200);
    expect(r.signed).toBe(true);
    expect(r.body.result).toMatchObject({ adapter: "fake-esp", protocol: 1, version: "0.0.0" });
    expect(r.body.idempotencyKey).toMatch(/^[0-9a-f-]{36}$/);
  });

  it("refuses requests it can't authenticate or read", async () => {
    const { client } = await start({});
    const cases = [
      [await client.call("verify", {}, { connection: conn, unsigned: true }), 401, "unauthenticated"],
      [await client.call("verify", {}, { connection: conn, secret: newSecret() }), 401, "unauthenticated"],
      [await client.call("verify", {}, { connection: conn, timestampOffsetSeconds: 400 }), 401, "unauthenticated"],
      [await client.call("verify", {}, { connection: conn, operation: "send" }), 400, "invalid_request"],
      [await client.call("verify", {}, { connection: conn, adapter: "other" }), 400, "invalid_request"],
      [await client.call("verify", {}), 400, "invalid_request"],
      [await client.call("quota", {}, { connection: conn }), 501, "not_supported"],
      [await client.call("verify", {}, { connection: conn, path: "/v2/verify" }), 501, "not_supported"],
    ] as const;
    for (const [r, status, code] of cases) {
      expect([r.status, r.body.error?.code]).toEqual([status, code]);
      expect(r.signed).toBe(true);
    }
  });

  it("refuses to start without a secret or with a manifest that disagrees", async () => {
    const adapter = defineAdapter({ id: "fake-esp", capabilities: { batch: 10 }, send: async () => ({ providerMessageId: "x" }) });
    const saved = process.env["REPUTECAST_ADAPTER_SECRET"];
    delete process.env["REPUTECAST_ADAPTER_SECRET"];
    await expect(serve(adapter, { port: 0, logger: false })).rejects.toThrow(/REPUTECAST_ADAPTER_SECRET/);
    if (saved !== undefined) process.env["REPUTECAST_ADAPTER_SECRET"] = saved;
    await expect(serve(adapter, { port: 0, secret: "rcas_x", manifest: manifest(), logger: false })).rejects.toThrow(ManifestError);
    await expect(serve(adapter, { port: 0, secret: "rcas_x", manifest: manifest(), logger: false })).rejects.toThrow(/batch: 10 in the code, 50 in manifest.json/);
  });

  it("sends batches message by message and keeps the order", async () => {
    const { client } = await start({
      sendConcurrency: 3,
      send: async (m) => {
        await new Promise((r) => setTimeout(r, Math.random() * 10));
        if (m.to.address === "rejected@contract.test") throw new MessageRejected("suppressed", { code: "recipient_suppressed" });
        if (m.to.address === "invalid@contract.test") throw ProviderError.refused("invalid recipient", { status: 422, providerCode: "invalid_recipient" });
        return { providerMessageId: "pm-" + m.id };
      },
    });
    const msgs = Array.from({ length: 6 }, (_, i) => testMessage(i));
    msgs[2]!.to.address = "rejected@contract.test";
    msgs[4]!.to.address = "invalid@contract.test";
    const r = await client.call<SendOutput>("send", { messages: msgs }, { connection: conn });
    const results = r.body.result!.results;
    expect(results.map((x) => x.messageId)).toEqual(msgs.map((m) => m.id));
    expect(results[0]).toEqual({ messageId: msgs[0]!.id, providerMessageId: "pm-" + msgs[0]!.id });
    expect(results[2]!.error).toEqual({ code: "recipient_suppressed", message: "suppressed", temporary: false, throttled: false });
    expect(results[4]!.error).toMatchObject({ code: "invalid_recipient", temporary: false });
    const tooMany = await client.call("send", { messages: Array.from({ length: 11 }, (_, i) => testMessage(i)) }, { connection: conn });
    expect(tooMany.body.error?.code).toBe("invalid_request");
  });

  it("fails the call only while no message was accepted", async () => {
    let n = 0;
    const { client } = await start({
      sendConcurrency: 1,
      send: async (m) => {
        n++;
        if (n >= 2) throw ProviderError.unavailable("Fake ESP answered 503.", { status: 503, providerCode: "down" });
        return { providerMessageId: "pm-" + m.id };
      },
    });
    const msgs = [testMessage(0), testMessage(1), testMessage(2)];
    const r = await client.call<SendOutput>("send", { messages: msgs }, { connection: conn });
    const res = r.body.result!.results;
    expect(res[0]!.providerMessageId).toBe("pm-" + msgs[0]!.id);
    expect(res[1]!.error).toMatchObject({ code: "down", temporary: true });
    expect(res[2]!.error).toMatchObject({ temporary: true });
    const all = await client.call("send", { messages: [testMessage(3)] }, { connection: conn });
    expect([all.status, all.body.error?.code, all.body.error?.providerCode, all.body.error?.status]).toEqual([503, "provider_unavailable", "down", 503]);
  });

  it("answers a retried key once, without sending again", async () => {
    let calls = 0;
    const { client } = await start({
      send: async (m) => {
        calls++;
        await new Promise((r) => setTimeout(r, 30));
        return { providerMessageId: "pm-" + m.id };
      },
    });
    const msgs = [testMessage(0)];
    const [a, b] = await Promise.all([
      client.call<SendOutput>("send", { messages: msgs }, { connection: conn, key: "send-k1" }),
      client.call<SendOutput>("send", { messages: msgs }, { connection: conn, key: "send-k1" }),
    ]);
    const c = await client.call<SendOutput>("send", { messages: msgs }, { connection: conn, key: "send-k1" });
    expect(calls).toBe(1);
    expect(b.body).toEqual(a.body);
    expect(c.body).toEqual(a.body);
    expect(c.signed).toBe(true);
  });

  it("doesn't remember failures", async () => {
    let calls = 0;
    const { client } = await start({
      send: async (m) => {
        calls++;
        if (calls === 1) throw ProviderError.unavailable("down");
        return { providerMessageId: "pm-" + m.id };
      },
    });
    const msgs = [testMessage(0)];
    expect((await client.call("send", { messages: msgs }, { connection: conn, key: "send-k2" })).status).toBe(503);
    expect((await client.call("send", { messages: msgs }, { connection: conn, key: "send-k2" })).status).toBe(200);
  });

  it("stops at the deadline", async () => {
    const { client } = await start({
      verify: async (ctx) =>
        new Promise((_, reject) => {
          ctx.signal.addEventListener("abort", () => reject(new Error("aborted")));
        }),
    });
    const started = Date.now();
    const r = await client.call("verify", {}, { connection: conn, deadlineMs: 200 });
    expect(Date.now() - started).toBeLessThan(5_000);
    expect(r.body.error?.code).toBe("provider_unavailable");
    expect(r.body.error?.message).toContain("deadline");
  });

  it("verifies, answers handshakes and parses webhooks", async () => {
    const { client } = await start({
      capabilities: { webhooks: true },
      verifyWebhook: (req) => req.header("x-sig") === "good",
      handshake: (req) => (req.json<{ challenge?: string }>().challenge ? JSON.stringify({ challenge: req.json<{ challenge: string }>().challenge }) : undefined),
      normalizeEvent: (raw: { e: string; to: string; code?: string; mid?: string }) =>
        raw.e === "skip" ? null : { type: raw.e === "bounce" ? "hard_bounce" : "delivered", recipient: raw.to, code: raw.code, messageId: raw.mid, raw },
    });
    const req = (body: string, sig = "good") => ({ method: "POST", url: "https://rc.example/hooks/x/t", headers: { "x-sig": [sig] }, body: Buffer.from(body).toString("base64") });
    const body = JSON.stringify([
      { e: "bounce", to: "John@Example.com", code: "550 5.1.1", mid: "0199D5A8-2B6E-7C41-9A0E-5B8F1C2D3E40" },
      { e: "skip", to: "x@example.com" },
      { e: "delivered", to: "a@example.com", mid: "not-a-uuid" },
    ]);
    const v = await client.call<WebhookVerifyOutput>("webhooks/verify", { request: req(body) }, { connection: conn });
    expect(v.body.result).toEqual({});
    const hs = await client.call<WebhookVerifyOutput>("webhooks/verify", { request: req('{"challenge":"c1"}') }, { connection: conn });
    expect(hs.body.result).toEqual({ handshake: { body: '{"challenge":"c1"}' } });
    const bad = await client.call("webhooks/verify", { request: req(body, "bad") }, { connection: conn });
    expect([bad.status, bad.body.error?.code]).toEqual([422, "bad_signature"]);
    const p = await client.call<WebhookParseOutput>("webhooks/parse", { request: req(body), verified: false }, { connection: conn });
    const evs = p.body.result!.events;
    expect(evs).toHaveLength(2);
    expect(evs[0]).toMatchObject({ type: "hard_bounce", recipient: "john@example.com", originalCode: "550 5.1.1", originalType: "hard_bounce", bounceClass: "mailbox_unknown", messageId: "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40" });
    expect(evs[1]!.messageId).toBeUndefined();
    expect(evs[1]!.raw).toEqual({ e: "delivered", to: "a@example.com", mid: "not-a-uuid" });
    // Already verified at receipt: no signature needed.
    const verified = await client.call<WebhookParseOutput>("webhooks/parse", { request: req(body, "gone"), verified: true }, { connection: conn });
    expect(verified.body.result!.events).toHaveLength(2);
    const unverified = await client.call("webhooks/parse", { request: req(body, "gone"), verified: false }, { connection: conn });
    expect(unverified.body.error?.code).toBe("bad_signature");
    const garbage = await client.call("webhooks/parse", { request: req("not json"), verified: true }, { connection: conn });
    expect(garbage.body.error?.code).toBe("invalid_request");
  });

  it("reports adapter bugs as internal errors", async () => {
    const { client } = await start({
      capabilities: { webhooks: true, quotaDiscovery: true },
      verifyWebhook: () => true,
      normalizeEvent: () => ({ type: "arrived" as never, recipient: "a@example.com", raw: {} }),
      discoverQuota: () => ({ kind: "weekly" as never }),
    });
    const parse = await client.call("webhooks/parse", { request: { method: "POST", url: "u", headers: {}, body: Buffer.from("{}").toString("base64") }, verified: true }, { connection: conn });
    expect([parse.status, parse.body.error?.code]).toEqual([500, "internal"]);
    expect(parse.body.error?.message).toContain('unknown type "arrived"');
    const q = await client.call("quota", {}, { connection: conn });
    expect(q.body.error?.message).toContain("rolling_24h or monthly_plan");
  });

  it("pages suppressions and pushes them", async () => {
    const pushed: string[][] = [];
    const { client } = await start({
      capabilities: { suppressionSync: "two-way" },
      syncSuppressions: ({ cursor }) =>
        cursor ? { suppressions: [{ email: "B@Example.com", reason: "complaint" }] } : { suppressions: [{ email: "a@example.com", reason: "hard_bounce", createdAt: "2026-10-11T08:00:00Z" }], nextCursor: "p2" },
      pushSuppressions: (emails) => {
        pushed.push(emails);
      },
    });
    const first = await client.call<{ suppressions: unknown[]; nextCursor: string | null }>("suppressions/list", { since: null, cursor: null }, { connection: conn });
    expect(first.body.result).toEqual({ suppressions: [{ email: "a@example.com", reason: "hard_bounce", createdAt: "2026-10-11T08:00:00.000Z" }], nextCursor: "p2" });
    const second = await client.call<{ suppressions: unknown[]; nextCursor: string | null }>("suppressions/list", { since: null, cursor: "p2" }, { connection: conn });
    expect(second.body.result).toEqual({ suppressions: [{ email: "b@example.com", reason: "complaint" }], nextCursor: null });
    await client.call("suppressions/add", { emails: ["x@example.com"] }, { connection: conn });
    expect(pushed).toEqual([["x@example.com"]]);
    const tooMany = await client.call("suppressions/add", { emails: Array.from({ length: 501 }, (_, i) => `a${i}@example.com`) }, { connection: conn });
    expect(tooMany.body.error?.code).toBe("invalid_request");
  });

  it("answers liveness probes without authentication", async () => {
    const { url } = await start({});
    const r = await fetch(url + "/healthz");
    expect(r.status).toBe(200);
    expect(await r.json()).toEqual({ ok: true });
  });

  it("refuses bodies over the limit", async () => {
    const adapter = defineAdapter({ id: "fake-esp", capabilities: {}, send: async () => ({ providerMessageId: "x" }) });
    const s = await serve(adapter, { port: 0, host: "127.0.0.1", secret: newSecret(), logger: false, maxBodyBytes: 1024 });
    servers.push(s);
    const r = await fetch(s.url + "/v1/send", { method: "POST", body: "x".repeat(4096) });
    expect(r.status).toBe(413);
    expect(((await r.json()) as { error: { code: string } }).error.code).toBe("too_large");
  });
});
