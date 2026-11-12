import { describe, expect, it } from "vitest";

import { createClient, isRetryable, MAX_RETRY_DELAY_MS, retryDelay, USER_AGENT } from "../src/index.ts";
import { withRetries } from "../src/retry.ts";

const base = "https://news.example.com";

function answers(...statuses: (number | [number, Record<string, string>])[]) {
  const seen: { method: string; body: string; headers: Headers }[] = [];
  const fetch = async (request: Request) => {
    seen.push({ method: request.method, body: await request.text(), headers: request.headers });
    const next = statuses[Math.min(seen.length - 1, statuses.length - 1)]!;
    const [status, headers] = typeof next === "number" ? [next, {}] : next;
    return new Response(status === 204 ? null : JSON.stringify({ status }), { status, headers: { "Content-Type": "application/json", ...headers } });
  };
  return { fetch, seen };
}

function sleeps() {
  const delays: number[] = [];
  return { delays, sleep: async (ms: number) => void delays.push(ms) };
}

describe("retries", () => {
  it("retries a GET after 503 with backoff and returns the answer that worked", async () => {
    const { fetch, seen } = answers(503, 502, 200);
    const { delays, sleep } = sleeps();
    const res = await withRetries(fetch, { maxRetries: 2, sleep })(new Request(`${base}/api/v1/x`));
    expect(res.status).toBe(200);
    expect(seen).toHaveLength(3);
    expect(delays[0]).toBeGreaterThanOrEqual(250);
    expect(delays[0]).toBeLessThanOrEqual(500);
    expect(delays[1]).toBeGreaterThanOrEqual(500);
    expect(delays[1]).toBeLessThanOrEqual(1000);
  });

  it("honours Retry-After in seconds and as a date, capped at 30 s", async () => {
    expect(retryDelay(0, "2")).toBe(2000);
    expect(retryDelay(0, "120")).toBe(MAX_RETRY_DELAY_MS);
    const now = Date.parse("2026-11-11T10:00:00Z");
    expect(retryDelay(0, "Wed, 11 Nov 2026 10:00:05 GMT", now)).toBe(5000);
    expect(retryDelay(0, "Wed, 11 Nov 2026 09:00:00 GMT", now)).toBe(0);
    expect(retryDelay(0, "soon", now, () => 0)).toBe(250);
    expect(retryDelay(20, null, now, () => 1)).toBe(MAX_RETRY_DELAY_MS);

    const { fetch } = answers([429, { "Retry-After": "3" }], 200);
    const { delays, sleep } = sleeps();
    const res = await withRetries(fetch, { maxRetries: 2, sleep })(new Request(`${base}/api/v1/x`, { method: "DELETE" }));
    expect(res.status).toBe(200);
    expect(delays).toEqual([3000]);
  });

  it("retries a POST only with an Idempotency-Key, sending the same body again", async () => {
    const plain = answers(503, 200);
    const { sleep } = sleeps();
    const once = await withRetries(plain.fetch, { maxRetries: 2, sleep })(new Request(`${base}/api/v1/x`, { method: "POST", body: '{"a":1}' }));
    expect(once.status).toBe(503);
    expect(plain.seen).toHaveLength(1);

    const keyed = answers(504, 200);
    const req = new Request(`${base}/api/v1/x`, { method: "POST", body: '{"a":1}', headers: { "Idempotency-Key": "k-1" } });
    expect(isRetryable(req)).toBe(true);
    const twice = await withRetries(keyed.fetch, { maxRetries: 2, sleep })(req);
    expect(twice.status).toBe(200);
    expect(keyed.seen.map((s) => s.body)).toEqual(['{"a":1}', '{"a":1}']);
  });

  it("returns the last answer when retries run out, and never retries other errors", async () => {
    const busy = answers(429);
    const { sleep } = sleeps();
    expect((await withRetries(busy.fetch, { maxRetries: 2, sleep })(new Request(`${base}/x`))).status).toBe(429);
    expect(busy.seen).toHaveLength(3);

    const broken = answers(500, 200);
    expect((await withRetries(broken.fetch, { maxRetries: 2, sleep })(new Request(`${base}/x`))).status).toBe(500);
    expect(broken.seen).toHaveLength(1);

    const off = answers(503, 200);
    expect((await withRetries(off.fetch, { maxRetries: 0, sleep })(new Request(`${base}/x`))).status).toBe(503);
    expect(off.seen).toHaveLength(1);
  });

  it("doesn't wait for a retry once the request is aborted", async () => {
    const controller = new AbortController();
    const { fetch, seen } = answers([503, { "Retry-After": "30" }], 200);
    const aborting = async (request: Request) => {
      const res = await fetch(request);
      controller.abort(new Error("gave up"));
      return res;
    };
    const pending = withRetries(aborting, { maxRetries: 2 })(new Request(`${base}/x`, { signal: controller.signal }));
    await expect(pending).rejects.toThrow("gave up");
    expect(seen).toHaveLength(1);
  });
});

describe("createClient", () => {
  it("sends the key and the SDK's User-Agent", async () => {
    const { fetch, seen } = answers(200);
    const rc = createClient({ baseUrl: `${base}/`, apiKey: "rc_live_test", fetch });
    await rc.GET("/api/v1/workspaces/{workspace}/lists", { params: { path: { workspace: "acme-newsletter" } } });
    expect(seen[0]!.headers.get("Authorization")).toBe("Bearer rc_live_test");
    expect(seen[0]!.headers.get("User-Agent")).toBe(USER_AGENT);
    expect(USER_AGENT).toMatch(/^ReputeCast-SDK-TS\/\S+$/);
  });

  it("refuses options that can't work", () => {
    expect(() => createClient({ baseUrl: "", apiKey: "k" })).toThrow(TypeError);
    expect(() => createClient({ baseUrl: base, apiKey: "" })).toThrow(TypeError);
    expect(() => createClient({ baseUrl: base, apiKey: "k", maxRetries: -1 })).toThrow(TypeError);
  });
});
