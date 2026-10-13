import { createHmac } from "node:crypto";

import { describe, expect, it } from "vitest";

import { classifyBounce, splitSmtpReply } from "../src/bounce.ts";
import { WebhookRequest, parseSignatureHeader, splitJson, verifyBasicAuth, verifyHmac, verifyTimestampedHmac } from "../src/webhooks.ts";

function request(headers: Record<string, string>, body: string): WebhookRequest {
  const h: Record<string, string[]> = {};
  for (const [k, v] of Object.entries(headers)) h[k] = [v];
  return new WebhookRequest("POST", "https://rc.example/hooks/community:x/t", h, Buffer.from(body));
}

describe("webhook helpers", () => {
  const body = '{"event":"delivered"}';

  it("checks HMAC signatures", () => {
    const hex = createHmac("sha256", "key").update(body).digest("hex");
    const b64 = createHmac("sha256", "key").update(body).digest("base64");
    expect(verifyHmac({ secret: "key", payload: body, signature: hex })).toBe(true);
    expect(verifyHmac({ secret: "key", payload: body, signature: hex.toUpperCase() })).toBe(true);
    expect(verifyHmac({ secret: "key", payload: body, signature: "sha256=" + hex, prefix: "sha256=" })).toBe(true);
    expect(verifyHmac({ secret: "key", payload: body, signature: b64, encoding: "base64" })).toBe(true);
    expect(verifyHmac({ secret: "other", payload: body, signature: hex })).toBe(false);
    expect(verifyHmac({ secret: "", payload: body, signature: hex })).toBe(false);
    expect(verifyHmac({ secret: "key", payload: body + " ", signature: hex })).toBe(false);
    expect(verifyHmac({ secret: "key", payload: body, signature: undefined })).toBe(false);
  });

  it("checks timestamped signatures and refuses replays", () => {
    const now = 1_760_346_062_000;
    const ts = now / 1000;
    const sig = createHmac("sha256", "key").update(`${ts}.${body}`).digest("hex");
    const parsed = parseSignatureHeader(`t=${ts},v1=deadbeef,v1=${sig}`);
    expect(parsed["t"]).toEqual([String(ts)]);
    expect(verifyTimestampedHmac({ secret: "key", payload: body, timestamp: parsed["t"]?.[0], signature: parsed["v1"] ?? [], nowMs: now })).toBe(true);
    expect(verifyTimestampedHmac({ secret: "key", payload: body, timestamp: ts, signature: sig, nowMs: now + 301_000 })).toBe(false);
    expect(verifyTimestampedHmac({ secret: "key", payload: body, timestamp: ts + 1, signature: sig, nowMs: now })).toBe(false);
    expect(verifyTimestampedHmac({ secret: "key", payload: body, timestamp: undefined, signature: sig, nowMs: now })).toBe(false);
  });

  it("checks basic auth", () => {
    const auth = "Basic " + Buffer.from("reputecast:s3cret").toString("base64");
    expect(verifyBasicAuth(request({ Authorization: auth }, body), "reputecast", "s3cret")).toBe(true);
    expect(verifyBasicAuth(request({ authorization: auth }, body), "reputecast", "other")).toBe(false);
    expect(verifyBasicAuth(request({}, body), "reputecast", "s3cret")).toBe(false);
    expect(verifyBasicAuth(request({ authorization: auth }, body), "reputecast", "")).toBe(false);
  });

  it("reads forwarded requests", () => {
    const r = WebhookRequest.fromWire({ method: "POST", url: "https://rc.example/hooks/x/t", headers: { "X-Sig": ["a"] }, body: Buffer.from("[1,2]").toString("base64") });
    expect(r.header("x-sig")).toBe("a");
    expect(r.text()).toBe("[1,2]");
    expect(splitJson(r)).toEqual([1, 2]);
    expect(splitJson(request({}, body))).toEqual([{ event: "delivered" }]);
  });
});

describe("bounces", () => {
  // The same cases as internal/providers/bounce.
  it.each([
    ["550 5.1.1", "user unknown", "mailbox_unknown"],
    ["452 4.2.2", "", "mailbox_full"],
    ["550 5.7.1", "Message rejected due to spam content", "spam_block"],
    ["550 5.7.26", "DMARC policy", "policy_block"],
    ["421 4.7.0", "IP rate limited due to reputation", "reputation"],
    ["550 5.1.2", "", "domain_invalid"],
    ["451 4.3.0", "try later", "temporary"],
    ["", "something odd", "other"],
  ])("classifies %s %s as %s", (status, diag, want) => {
    expect(classifyBounce(status, diag)).toBe(want);
  });

  it("splits SMTP replies", () => {
    expect(splitSmtpReply("smtp; 550 5.1.1 <a@b>: User unknown")).toEqual({ code: "550 5.1.1", rest: "<a@b>: User unknown" });
    expect(splitSmtpReply("550-5.7.1 Blocked")).toEqual({ code: "550 5.7.1", rest: "Blocked" });
    expect(splitSmtpReply("Mailbox full")).toEqual({ code: "", rest: "Mailbox full" });
  });
});
