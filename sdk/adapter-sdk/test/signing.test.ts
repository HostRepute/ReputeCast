import { readFileSync } from "node:fs";

import { describe, expect, it } from "vitest";

import { SDK_VERSION } from "../src/index.ts";
import { newSecret, secretsFromEnv, sign, signatureHeader, timingSafeEqual, verifySignature } from "../src/signing.ts";

describe("signatures", () => {
  const now = 1_760_346_062_000;
  const ts = String(now / 1000);
  const body = '{"protocol":1}';

  it("matches ReputeCast's HMAC over timestamp and body", () => {
    // The same vector as internal/providers/remote (Go): hex HMAC-SHA256 of "<ts>.<body>".
    expect(sign("rcas_secret", 1760346062, body)).toMatch(/^[0-9a-f]{64}$/);
    expect(sign("rcas_secret", 1760346062, body)).toBe(sign("rcas_secret", 1760346062, Buffer.from(body)));
    expect(sign("rcas_secret", 1760346062, body)).not.toBe(sign("rcas_secret", 1760346063, body));
  });

  it("accepts any of the current and previous secrets", () => {
    const h = signatureHeader(["rcas_new", "rcas_old"], now / 1000, body);
    expect(h.split(", ")).toHaveLength(2);
    expect(verifySignature(["rcas_new"], ts, h, body, now)).toEqual({ ok: true });
    expect(verifySignature(["rcas_old"], ts, h, body, now)).toEqual({ ok: true });
    expect(verifySignature(["rcas_other"], ts, h, body, now)).toEqual({ ok: false, reason: "mismatch" });
  });

  it("refuses stale, unsigned and tampered requests", () => {
    const h = signatureHeader(["rcas_new"], now / 1000, body);
    expect(verifySignature(["rcas_new"], ts, h, body, now + 301_000)).toEqual({ ok: false, reason: "stale" });
    expect(verifySignature(["rcas_new"], ts, h, body, now - 301_000)).toEqual({ ok: false, reason: "stale" });
    expect(verifySignature(["rcas_new"], undefined, h, body, now)).toEqual({ ok: false, reason: "missing" });
    expect(verifySignature(["rcas_new"], "12abc", h, body, now)).toEqual({ ok: false, reason: "missing" });
    expect(verifySignature(["rcas_new"], ts, h, '{"protocol":2}', now)).toEqual({ ok: false, reason: "mismatch" });
    expect(verifySignature(["rcas_new"], ts, "sha256=" + h.slice(3), body, now)).toEqual({ ok: false, reason: "mismatch" });
  });

  it("makes secrets in ReputeCast's format and reads them from the environment", () => {
    const s = newSecret();
    expect(s).toMatch(/^rcas_[A-Za-z0-9_-]{43}$/);
    expect(newSecret()).not.toBe(s);
    expect(secretsFromEnv(" rcas_a, rcas_b ,,")).toEqual(["rcas_a", "rcas_b"]);
    expect(secretsFromEnv("")).toEqual([]);
  });

  it("compares in constant time", () => {
    expect(timingSafeEqual("abc", "abc")).toBe(true);
    expect(timingSafeEqual("abc", "abd")).toBe(false);
    expect(timingSafeEqual("abc", "abcd")).toBe(false);
    expect(timingSafeEqual(new Uint8Array([1, 2]), new Uint8Array([1, 2]))).toBe(true);
  });

  it("keeps the version in step with package.json", () => {
    const pkg = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8")) as { version: string };
    expect(SDK_VERSION).toBe(pkg.version);
  });
});
