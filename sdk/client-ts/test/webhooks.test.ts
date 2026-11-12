import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import { verifyWebhook, WebhookVerificationError } from "../src/index.ts";

interface Vector {
  name: string;
  secrets: string[];
  body: string;
  headers: Record<string, string>;
  now: number;
  valid: boolean;
  error?: "missing_header" | "timestamp" | "signature";
}

// Signed by the server's webhooks.Sign (internal/webhooks/vectors_test.go).
const file = JSON.parse(readFileSync(new URL("../../testdata/webhook-vectors.json", import.meta.url), "utf8")) as {
  toleranceSeconds: number;
  vectors: Vector[];
};

describe("verifyWebhook against the server's vectors", () => {
  it("has vectors", () => {
    expect(file.vectors.length).toBeGreaterThan(10);
    expect(file.toleranceSeconds).toBe(300);
  });

  for (const v of file.vectors) {
    it(v.name, async () => {
      const check = verifyWebhook({ secrets: v.secrets, body: v.body, headers: v.headers, now: v.now * 1000 });
      if (v.valid) {
        await expect(check).resolves.toBeUndefined();
      } else {
        await expect(check).rejects.toBeInstanceOf(WebhookVerificationError);
        await expect(check).rejects.toMatchObject({ reason: v.error });
      }
    });
  }

  it("takes a Headers object, bytes and one secret", async () => {
    const v = file.vectors.find((x) => x.valid && x.secrets.length === 1)!;
    await verifyWebhook({ secrets: v.secrets[0]!, body: new TextEncoder().encode(v.body), headers: new Headers(v.headers), now: new Date(v.now * 1000) });
    await verifyWebhook({ secrets: v.secrets, body: new TextEncoder().encode(v.body).buffer, headers: v.headers, now: v.now * 1000 });
  });

  it("applies the tolerance it is given", async () => {
    const v = file.vectors.find((x) => x.name === "expired timestamp")!;
    await verifyWebhook({ secrets: v.secrets, body: v.body, headers: v.headers, now: v.now * 1000, toleranceSeconds: 600 });
  });
});
