import { describe, expect, it } from "vitest";

import { WebhookRequest, wireEvent } from "@reputecast/adapter-sdk";
import type { AdapterContext } from "@reputecast/adapter-sdk";
import { testMessage } from "@reputecast/adapter-sdk/testing";

import adapter, { toPayload } from "../src/adapter.ts";
import type { ExampleEvent } from "../src/adapter.ts";
import { WEBHOOK_KEY, signWebhook } from "./fake-api.ts";

const ctx = { credentials: { webhook_signing_key: WEBHOOK_KEY }, log: { debug() {} } } as unknown as AdapterContext;

function event(e: Partial<ExampleEvent>): ExampleEvent {
  return { id: "evt_1", event: "delivered", timestamp: "2026-10-12T13:41:02Z", message_id: "msg_1", recipient: "a@example.com", ...e };
}

describe("Example ESP adapter", () => {
  it("maps a message to the API's body", () => {
    const m = testMessage(0);
    const p = toPayload(m);
    expect(p.to).toEqual([{ email: m.to.address, name: m.to.name }]);
    expect(p.metadata.rc_message).toBe(m.id);
    expect(p.headers["List-Unsubscribe-Post"]).toBe("List-Unsubscribe=One-Click");
    expect(p.tracking).toEqual({ opens: false, clicks: false });
  });

  it.each([
    ["delivered", undefined, "delivered", undefined],
    ["bounced", "hard", "hard_bounce", "mailbox_unknown"],
    ["bounced", "soft", "soft_bounce", "mailbox_unknown"],
    ["complained", undefined, "complaint", undefined],
    ["unsubscribed", undefined, "unsubscribe", undefined],
    ["opened", undefined, "open", undefined],
  ] as const)("normalizes %s %s as %s", (name, bounceType, type, bounceClass) => {
    const raw = event({ event: name, ...(bounceType ? { bounce_type: bounceType, smtp_code: "550 5.1.1" } : {}), metadata: { rc_message: "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40" } });
    const out = adapter.normalizeEvent!(raw, ctx);
    expect(out).not.toBeNull();
    const w = wireEvent(out as never, raw, new Date());
    expect(w.type).toBe(type);
    expect(w.bounceClass).toBe(bounceClass);
    expect(w.messageId).toBe("0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40");
    expect(w.raw).toBe(raw);
  });

  it("skips events it doesn't know", () => {
    expect(adapter.normalizeEvent!(event({ event: "teleported" as never }), ctx)).toBeNull();
  });

  it("verifies Example ESP's webhook signature", async () => {
    const body = JSON.stringify(event({}));
    const ok = signWebhook(body);
    const req = (f: { headers: Record<string, string>; body: string }) =>
      new WebhookRequest("POST", "https://rc.example/hooks/community:example-esp/t", Object.fromEntries(Object.entries(f.headers).map(([k, v]) => [k, [v]])), Buffer.from(f.body));
    expect(await adapter.verifyWebhook!(req(ok), ctx)).toBe(true);
    expect(await adapter.verifyWebhook!(req({ ...ok, body: body + " " }), ctx)).toBe(false);
    expect(await adapter.verifyWebhook!(req(signWebhook(body, "another-key")), ctx)).toBe(false);
    expect(await adapter.verifyWebhook!(req(signWebhook(body, WEBHOOK_KEY, Math.floor(Date.now() / 1000) - 3600)), ctx)).toBe(false);
  });
});
