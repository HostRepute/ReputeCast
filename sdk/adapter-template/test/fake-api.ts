import { createHmac } from "node:crypto";

import { REJECT_ADDRESS } from "@reputecast/adapter-sdk/testing";
import type { FakeEnv, FakeReply, FakeRequest, WebhookFixture } from "@reputecast/adapter-sdk/testing";

// A fake of Example ESP's API for the contract suite. ReputeCast's Go
// end-to-end test (internal/providers/remote) has the same fake in Go.

export const API_KEY = "test-example-esp-api-key";
export const WEBHOOK_KEY = "test-example-esp-webhook-signing-key";

export function fakeExampleEsp(env: FakeEnv): (req: FakeRequest) => FakeReply {
  let n = 0;
  return (req) => {
    if (env.mode === "outage") return { status: 503, json: { error: { code: "service_unavailable", message: "Example ESP is down for maintenance." } } };
    if (env.mode === "auth" || req.headers["authorization"] !== `Bearer ${API_KEY}`) {
      return { status: 401, json: { error: { code: "invalid_api_key", message: "The API key is not valid." } } };
    }
    if (req.method === "POST" && req.path === "/v1/messages") {
      const m = req.json<{ to?: { email?: string }[] }>();
      if (m.to?.[0]?.email === REJECT_ADDRESS) {
        return { status: 422, json: { error: { code: "recipient_suppressed", message: "The recipient is on Example ESP's suppression list." } } };
      }
      n++;
      return { status: 202, json: { id: `msg_${String(n).padStart(6, "0")}`, status: "queued" } };
    }
    if (req.method === "GET" && req.path === "/v1/account") {
      return { json: { plan: "Growth", monthly_limit: 50000, sent_this_month: 1234, sandbox: false } };
    }
    if (req.method === "GET" && req.path === "/v1/suppressions") {
      if (!req.query.get("cursor")) {
        return { json: { data: [{ email: "Bounced@Example.com", reason: "bounce", created_at: "2026-10-11T08:00:00Z" }], next_cursor: "page-2" } };
      }
      return { json: { data: [{ email: "complainer@example.com", reason: "complaint", created_at: "2026-10-12T09:30:00Z" }], next_cursor: null } };
    }
    return { status: 404, json: { error: { code: "not_found", message: `No route ${req.method} ${req.path}.` } } };
  };
}

/** Signs a webhook body the way Example ESP does. */
export function signWebhook(body: string, key = WEBHOOK_KEY, timestamp = Math.floor(Date.now() / 1000)): WebhookFixture {
  const sig = createHmac("sha256", key).update(`${timestamp}.${body}`).digest("hex");
  return { headers: { "content-type": "application/json", "x-example-signature": `t=${timestamp},v1=${sig}` }, body };
}
