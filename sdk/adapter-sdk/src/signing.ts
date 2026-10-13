import { createHmac, randomBytes, timingSafeEqual as nodeTimingSafeEqual } from "node:crypto";

/** How far a signed timestamp may be from this process's clock, in seconds. */
export const MAX_SKEW_SECONDS = 300;

/** A fresh signing secret in ReputeCast's format ("rcas_" + 32 random bytes). */
export function newSecret(): string {
  return "rcas_" + randomBytes(32).toString("base64url");
}

/** Lowercase hex HMAC-SHA256 of "<timestamp>.<body>". */
export function sign(secret: string, timestamp: number, body: Uint8Array | string): string {
  const mac = createHmac("sha256", secret);
  mac.update(String(timestamp));
  mac.update(".");
  mac.update(body);
  return mac.digest("hex");
}

/** The X-ReputeCast-Signature value: one "v1=" entry per secret. */
export function signatureHeader(secrets: readonly string[], timestamp: number, body: Uint8Array | string): string {
  return secrets
    .filter((s) => s !== "")
    .map((s) => "v1=" + sign(s, timestamp, body))
    .join(", ");
}

export type SignatureCheck = { ok: true } | { ok: false; reason: "missing" | "stale" | "mismatch" };

/**
 * Checks a timestamp and signature header against any of the secrets
 * (several during a rotation).
 */
export function verifySignature(
  secrets: readonly string[],
  timestamp: string | null | undefined,
  header: string | null | undefined,
  body: Uint8Array | string,
  nowMs: number = Date.now(),
): SignatureCheck {
  if (!timestamp || !header || !/^\d{1,12}$/.test(timestamp)) return { ok: false, reason: "missing" };
  const ts = Number(timestamp);
  if (Math.abs(nowMs / 1000 - ts) > MAX_SKEW_SECONDS) return { ok: false, reason: "stale" };
  for (const part of header.split(",")) {
    const entry = part.trim();
    if (!entry.startsWith("v1=")) continue;
    const got = entry.slice(3);
    for (const s of secrets) {
      if (s !== "" && timingSafeEqual(sign(s, ts, body), got)) return { ok: true };
    }
  }
  return { ok: false, reason: "mismatch" };
}

/** Constant-time comparison of two strings or byte arrays. */
export function timingSafeEqual(a: string | Uint8Array, b: string | Uint8Array): boolean {
  const x = typeof a === "string" ? Buffer.from(a) : Buffer.from(a.buffer, a.byteOffset, a.byteLength);
  const y = typeof b === "string" ? Buffer.from(b) : Buffer.from(b.buffer, b.byteOffset, b.byteLength);
  if (x.length !== y.length) {
    // Compare anyway so the time doesn't depend on where they differ.
    nodeTimingSafeEqual(x, x);
    return false;
  }
  return nodeTimingSafeEqual(x, y);
}

/** Reads the secrets from REPUTECAST_ADAPTER_SECRET (comma-separated during a rotation). */
export function secretsFromEnv(value: string | undefined = process.env["REPUTECAST_ADAPTER_SECRET"]): string[] {
  return (value ?? "")
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s !== "");
}
