import { createHmac } from "node:crypto";

import type { WireWebhookRequest } from "./protocol.ts";
import { timingSafeEqual } from "./signing.ts";

/** A provider webhook as ReputeCast received it. */
export class WebhookRequest {
  readonly method: string;
  /** ReputeCast's public URL of the webhook endpoint. */
  readonly url: string;
  /** Lowercase header names, every value. */
  readonly headers: Readonly<Record<string, readonly string[]>>;
  /** The raw body, exactly as the provider sent it (sign this, not a re-encoding). */
  readonly body: Buffer;

  constructor(method: string, url: string, headers: Record<string, readonly string[]>, body: Buffer) {
    this.method = method;
    this.url = url;
    const lower: Record<string, readonly string[]> = {};
    for (const [k, v] of Object.entries(headers)) lower[k.toLowerCase()] = v;
    this.headers = lower;
    this.body = body;
  }

  static fromWire(w: WireWebhookRequest): WebhookRequest {
    return new WebhookRequest(w.method || "POST", w.url, w.headers ?? {}, Buffer.from(w.body ?? "", "base64"));
  }

  /** The first value of a header (case-insensitive). */
  header(name: string): string | undefined {
    return this.headers[name.toLowerCase()]?.[0];
  }

  text(): string {
    return this.body.toString("utf8");
  }

  json<T = unknown>(): T {
    return JSON.parse(this.text()) as T;
  }
}

export interface HmacOptions {
  secret: string | Uint8Array;
  /** What was signed: usually the raw body. */
  payload: string | Uint8Array;
  /** The signature to check (the header value, without its prefix). */
  signature: string | undefined | null;
  algorithm?: "sha256" | "sha1" | "sha512";
  encoding?: "hex" | "base64" | "base64url";
  /** A prefix the signature carries ("sha256="). */
  prefix?: string;
}

/** Checks an HMAC signature in constant time. */
export function verifyHmac(o: HmacOptions): boolean {
  if (!o.signature || (typeof o.secret === "string" ? o.secret === "" : o.secret.length === 0)) return false;
  let sig = o.signature.trim();
  if (o.prefix) {
    if (!sig.startsWith(o.prefix)) return false;
    sig = sig.slice(o.prefix.length);
  }
  const want = createHmac(o.algorithm ?? "sha256", o.secret)
    .update(o.payload)
    .digest(o.encoding ?? "hex");
  const got = o.encoding === "hex" || o.encoding === undefined ? sig.toLowerCase() : sig;
  return timingSafeEqual(want, got);
}

export interface TimestampedHmacOptions {
  secret: string | Uint8Array;
  payload: string | Uint8Array;
  /** Unix seconds the provider sent with the signature. */
  timestamp: string | number | undefined | null;
  /** One or more candidate signatures (providers send several while rotating). */
  signature: string | readonly string[] | undefined | null;
  /** How the timestamp and payload are joined before signing (default "."). */
  separator?: string;
  algorithm?: "sha256" | "sha1" | "sha512";
  encoding?: "hex" | "base64" | "base64url";
  /** Oldest acceptable signature, in seconds (default 300). */
  toleranceSeconds?: number;
  nowMs?: number;
}

/**
 * Checks a signature over "<timestamp><separator><payload>" and refuses old
 * timestamps (replays).
 */
export function verifyTimestampedHmac(o: TimestampedHmacOptions): boolean {
  if (o.timestamp === undefined || o.timestamp === null || o.signature === undefined || o.signature === null) return false;
  const ts = Number(o.timestamp);
  if (!Number.isFinite(ts)) return false;
  const now = (o.nowMs ?? Date.now()) / 1000;
  if (Math.abs(now - ts) > (o.toleranceSeconds ?? 300)) return false;
  const payload = Buffer.concat([Buffer.from(String(o.timestamp) + (o.separator ?? ".")), Buffer.from(o.payload)]);
  const candidates = typeof o.signature === "string" ? [o.signature] : o.signature;
  return candidates.some((signature) =>
    verifyHmac({ secret: o.secret, payload, signature, algorithm: o.algorithm ?? "sha256", encoding: o.encoding ?? "hex" }),
  );
}

/**
 * Parses a signature header of comma-separated key=value pairs, such as
 * "t=1760346062,v1=5b1d…,v1=9c2e…", into { t: ["1760346062"], v1: [...] }.
 */
export function parseSignatureHeader(header: string | undefined | null): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const part of (header ?? "").split(",")) {
    const i = part.indexOf("=");
    if (i <= 0) continue;
    const k = part.slice(0, i).trim();
    const v = part.slice(i + 1).trim();
    (out[k] ??= []).push(v);
  }
  return out;
}

/** Checks HTTP basic auth on a webhook request in constant time. */
export function verifyBasicAuth(req: WebhookRequest, user: string, password: string): boolean {
  const h = req.header("authorization") ?? "";
  if (!h.toLowerCase().startsWith("basic ") || password === "") return false;
  const decoded = Buffer.from(h.slice(6).trim(), "base64").toString("utf8");
  const i = decoded.indexOf(":");
  if (i < 0) return false;
  const okUser = timingSafeEqual(decoded.slice(0, i), user);
  const okPass = timingSafeEqual(decoded.slice(i + 1), password);
  return okUser && okPass;
}

/** The raw events in a webhook body: the items of a JSON array, or the object itself. */
export function splitJson(req: WebhookRequest): unknown[] {
  const body = req.json<unknown>();
  return Array.isArray(body) ? body : [body];
}
