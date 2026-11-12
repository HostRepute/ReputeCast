/** The headers of a webhook delivery (docs/api/README.md › Outgoing webhooks). */
export const SIGNATURE_HEADER = "X-ReputeCast-Signature";
export const TIMESTAMP_HEADER = "X-ReputeCast-Timestamp";

/** How far a delivery's timestamp may be from the receiver's clock, by default. */
export const DEFAULT_TOLERANCE_SECONDS = 300;

export type WebhookHeaders = Headers | Readonly<Record<string, string | readonly string[] | undefined>>;

export interface VerifyWebhookOptions {
  /** The endpoint's signing secret ("whsec_…"), or several while you rotate your own. */
  secrets: string | readonly string[];
  /** The raw request body, exactly as received: not parsed and serialised again. */
  body: string | Uint8Array | ArrayBuffer;
  /** The request's headers: a Headers object or a plain object such as Node's req.headers. */
  headers: WebhookHeaders;
  toleranceSeconds?: number;
  /** The receiver's clock (tests). */
  now?: Date | number;
}

export type WebhookErrorReason = "missing_header" | "timestamp" | "signature";

/** A delivery that isn't from ReputeCast, was changed on the way or is too old: answer it with 400 or 401. */
export class WebhookVerificationError extends Error {
  override readonly name = "WebhookVerificationError";
  readonly reason: WebhookErrorReason;

  constructor(reason: WebhookErrorReason, message: string) {
    super(message);
    this.reason = reason;
  }
}

const encoder = new TextEncoder();
const HEX_SIGNATURE = /^[0-9a-f]{64}$/;

/**
 * Checks that a webhook delivery was signed by ReputeCast with one of the
 * secrets: the hex HMAC-SHA256 of "<timestamp>.<body>" in
 * X-ReputeCast-Signature (two comma-separated signatures for 24 hours after
 * the endpoint's secret is rotated), with X-ReputeCast-Timestamp within the
 * tolerance of now. Signatures are compared in constant time (Web Crypto's
 * HMAC verify). Throws a WebhookVerificationError when the delivery fails.
 */
export async function verifyWebhook({
  secrets,
  body,
  headers,
  toleranceSeconds = DEFAULT_TOLERANCE_SECONDS,
  now = Date.now(),
}: VerifyWebhookOptions): Promise<void> {
  const timestamp = header(headers, TIMESTAMP_HEADER);
  const signature = header(headers, SIGNATURE_HEADER);
  if (!timestamp || !signature) {
    throw new WebhookVerificationError("missing_header", `The delivery has no ${timestamp ? SIGNATURE_HEADER : TIMESTAMP_HEADER} header.`);
  }
  if (!/^\d+$/.test(timestamp)) throw new WebhookVerificationError("timestamp", `${TIMESTAMP_HEADER} isn't a Unix time in seconds.`);
  const nowMs = typeof now === "number" ? now : now.getTime();
  if (Math.abs(nowMs - Number(timestamp) * 1000) > toleranceSeconds * 1000) {
    throw new WebhookVerificationError("timestamp", `The delivery's timestamp is more than ${toleranceSeconds} seconds from now.`);
  }

  const raw = typeof body === "string" ? encoder.encode(body) : body instanceof Uint8Array ? body : new Uint8Array(body);
  const prefix = encoder.encode(`${timestamp}.`);
  const signed = new Uint8Array(prefix.length + raw.length);
  signed.set(prefix);
  signed.set(raw, prefix.length);

  const candidates = signature
    .split(",")
    .map((s) => s.trim())
    .filter((s) => HEX_SIGNATURE.test(s))
    .map(fromHex);
  for (const secret of typeof secrets === "string" ? [secrets] : secrets) {
    if (!secret) continue;
    const key = await crypto.subtle.importKey("raw", encoder.encode(secret), { name: "HMAC", hash: "SHA-256" }, false, ["verify"]);
    for (const candidate of candidates) {
      if (await crypto.subtle.verify("HMAC", key, candidate, signed)) return;
    }
  }
  throw new WebhookVerificationError("signature", "No signature of the delivery matches the signing secret.");
}

function header(headers: WebhookHeaders, name: string): string | undefined {
  if (headers instanceof Headers) return headers.get(name) ?? undefined;
  const lower = name.toLowerCase();
  for (const [key, value] of Object.entries(headers)) {
    if (key.toLowerCase() !== lower || value === undefined) continue;
    return typeof value === "string" ? value : value.join(",");
  }
  return undefined;
}

function fromHex(hex: string): Uint8Array<ArrayBuffer> {
  const out = new Uint8Array(hex.length / 2);
  for (let i = 0; i < out.length; i++) out[i] = Number.parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  return out;
}
