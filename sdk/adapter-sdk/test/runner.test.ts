import { createHmac } from "node:crypto";

import { describe, expect, it } from "vitest";

import { defineAdapter } from "../src/define.ts";
import type { AdapterDefinition } from "../src/define.ts";
import { verifyHmac } from "../src/webhooks.ts";
import { REJECT_ADDRESS, defineContract, runContract } from "../src/testing/index.ts";
import type { Contract, FakeEnv, FakeReply, FakeRequest } from "../src/testing/index.ts";

// A tiny provider ("Tiny ESP") with a fake API, to check the runner itself.

const KEY = "tiny-key";

function tinyApi(env: FakeEnv) {
  let n = 0;
  const seen = new Set<string>();
  return (req: FakeRequest): FakeReply => {
    if (env.mode === "outage") return { status: 502, json: { message: "bad gateway" } };
    if (env.mode === "auth" || req.headers["x-key"] !== KEY) return { status: 403, json: { message: "forbidden" } };
    if (req.path === "/send") {
      const b = req.json<{ to: string; id: string }>();
      if (b.to === REJECT_ADDRESS) return { status: 400, json: { code: "bad_recipient", message: "no" } };
      seen.add(b.id);
      return { json: { id: `t${++n}` } };
    }
    if (req.path === "/me") return { json: {} };
    return { status: 404 };
  };
}

const manifest = {
  protocol: 1,
  kind: "remote",
  id: "tiny-esp",
  name: "Tiny ESP",
  version: "0.0.1",
  maintainer: "tests",
  capabilities: { batch: 5, webhooks: true },
  terms: { bulkMarketing: "your_policy" },
  pricing: "free",
  config: [{ key: "api_key", label: "API key", secret: true, required: true }],
};

function tiny(overrides: Partial<AdapterDefinition> = {}) {
  return defineAdapter({
    id: "tiny-esp",
    capabilities: { batch: 5, webhooks: true },
    http: { baseUrl: () => process.env["TINY_URL"] ?? "https://tiny.invalid", headers: (ctx) => ({ "x-key": String(ctx.credentials["api_key"]) }) },
    async send(msg, ctx) {
      const r = await ctx.http.post<{ id: string }>("/send", { to: msg.to.address, id: msg.id });
      return { providerMessageId: r.data.id };
    },
    async verify(ctx) {
      await ctx.http.get("/me");
    },
    verifyWebhook: (req) => verifyHmac({ secret: "whk", payload: req.body, signature: req.header("x-tiny-signature") }),
    normalizeEvent: (raw: { type: string; to: string; id: string }) => ({ type: raw.type === "bounce" ? "hard_bounce" : "delivered", recipient: raw.to, providerMessageId: raw.id, code: "550 5.1.1", raw }),
    ...overrides,
  });
}

function contract(adapter = tiny(), extra: Partial<Contract> = {}): Contract {
  const body = JSON.stringify({ type: "bounce", to: "a@example.com", id: "t1" });
  return defineContract({
    adapter,
    manifest,
    api: tinyApi,
    connection: () => ({ credentials: { api_key: KEY } }),
    environment: (env) => ({ TINY_URL: env.url }),
    webhooks: [
      {
        name: "Bounce",
        request: () => ({ headers: { "x-tiny-signature": createHmac("sha256", "whk").update(body).digest("hex") }, body }),
        want: [{ type: "hard_bounce", recipient: "a@example.com", providerMessageId: "t1", originalType: "hard_bounce", originalCode: "550 5.1.1", bounceClass: "mailbox_unknown" }],
      },
    ],
    ...extra,
  });
}

describe("runContract", () => {
  it("passes a correct adapter", async () => {
    const lines: string[] = [];
    const report = await runContract(contract(), { log: (l) => lines.push(l) });
    expect(report.failed, lines.join("\n")).toBe(0);
    expect(report.results.map((r) => r.name)).toEqual([
      "Registration",
      "Health",
      "Authentication",
      "Send",
      "SendRejected",
      "SendOutage",
      "SendAuth",
      "SendIdempotent",
      "Verify",
      "Quota",
      "SuppressionsPull",
      "Webhook/Bounce",
    ]);
    expect(report.skipped).toBe(2);
    expect(process.env["TINY_URL"]).toBeUndefined();
  });

  it("finds what a broken adapter gets wrong", async () => {
    const broken = tiny({
      // Swallows every failure: rejections look accepted, outages too.
      async send(msg) {
        return { providerMessageId: "always-" + msg.id };
      },
      // Accepts every webhook.
      verifyWebhook: () => true,
    });
    const report = await runContract(contract(broken));
    const failed = Object.fromEntries(report.results.filter((r) => r.status === "fail").map((r) => [r.name, r.detail ?? ""]));
    expect(Object.keys(failed).sort()).toEqual(["SendAuth", "SendOutage", "SendRejected", "Webhook/Bounce"]);
    expect(failed["SendRejected"]).toContain("want a per-message error");
    expect(failed["Webhook/Bounce"]).toContain("tampered");
  });

  it("refuses a manifest that disagrees with the code", async () => {
    const report = await runContract(contract(tiny(), { manifest: { ...manifest, capabilities: { batch: 10, webhooks: true } } }));
    expect(report.results).toHaveLength(1);
    expect(report.results[0]).toMatchObject({ name: "Registration", status: "fail" });
    expect(report.results[0]?.detail).toContain("batch: 5 in the code, 10 in manifest.json");
  });
});
