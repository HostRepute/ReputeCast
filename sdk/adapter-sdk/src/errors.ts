import type { ErrorCode, WireError } from "./protocol.ts";
import { ERROR_STATUS } from "./protocol.ts";

// Errors are recognized by a brand rather than instanceof, so an adapter and
// the server agree even when two copies of the SDK are loaded.
const BRAND = Symbol.for("@reputecast/adapter-sdk/error");
const REJECTED = Symbol.for("@reputecast/adapter-sdk/message-rejected");

export interface ProviderErrorOptions {
  /** The provider's HTTP status. */
  status?: number;
  /** The provider's own error code ("invalid_api_key"). */
  providerCode?: string;
  /** Seconds before retrying (throttling). */
  retryAfter?: number;
  cause?: unknown;
}

/**
 * An operation failed. The code says what ReputeCast does next (see
 * docs/adapters.md): provider_auth asks for new credentials,
 * provider_unavailable fails over, provider_throttled retries later,
 * provider_error is permanent.
 */
export class AdapterError extends Error {
  readonly code: ErrorCode;
  readonly status: number | undefined;
  readonly providerCode: string | undefined;
  readonly retryAfter: number | undefined;
  readonly [BRAND] = true;

  constructor(code: ErrorCode, message: string, opts: ProviderErrorOptions = {}) {
    super(message, opts.cause === undefined ? undefined : { cause: opts.cause });
    this.name = "AdapterError";
    this.code = code;
    this.status = opts.status;
    this.providerCode = opts.providerCode;
    this.retryAfter = opts.retryAfter;
  }
}

/** The provider failed; build one with the static helpers. */
export class ProviderError extends AdapterError {
  constructor(code: Extract<ErrorCode, `provider_${string}`>, message: string, opts: ProviderErrorOptions = {}) {
    super(code, message, opts);
    this.name = "ProviderError";
  }

  /** The provider rejected the credentials (HTTP 401/403). */
  static auth(message: string, opts?: ProviderErrorOptions): ProviderError {
    return new ProviderError("provider_auth", message, opts);
  }

  /** The provider is down or timing out. */
  static unavailable(message: string, opts?: ProviderErrorOptions): ProviderError {
    return new ProviderError("provider_unavailable", message, opts);
  }

  /** The provider is rate limiting. */
  static throttled(message: string, opts?: ProviderErrorOptions): ProviderError {
    return new ProviderError("provider_throttled", message, opts);
  }

  /** The provider refused the request for good. */
  static refused(message: string, opts?: ProviderErrorOptions): ProviderError {
    return new ProviderError("provider_error", message, opts);
  }
}

/** A provider answered an HTTP error (thrown by ctx.http). */
export class HttpError extends ProviderError {
  readonly data: unknown;
  readonly headers: Headers;

  constructor(code: Extract<ErrorCode, `provider_${string}`>, message: string, status: number, data: unknown, headers: Headers, providerCode?: string, retryAfter?: number) {
    super(code, message, { status, ...(providerCode === undefined ? {} : { providerCode }), ...(retryAfter === undefined ? {} : { retryAfter }) });
    this.name = "HttpError";
    this.data = data;
    this.headers = headers;
  }
}

/** The provider's webhook signature is invalid. */
export class BadSignatureError extends AdapterError {
  constructor(message = "The webhook signature doesn't match.") {
    super("bad_signature", message);
    this.name = "BadSignatureError";
  }
}

/** The adapter doesn't implement an operation. */
export class NotSupportedError extends AdapterError {
  constructor(message = "This adapter doesn't support that.") {
    super("not_supported", message);
    this.name = "NotSupportedError";
  }
}

export interface RejectionOptions {
  /** The provider's code for the refusal ("recipient_suppressed"). */
  code?: string;
  /** True when retrying later can work (a full mailbox, throttling). */
  temporary?: boolean;
  throttled?: boolean;
}

/**
 * The provider refused one message: thrown from send(), it becomes a
 * per-message error and the rest of the batch goes on.
 */
export class MessageRejected extends Error {
  readonly code: string;
  readonly temporary: boolean;
  readonly throttled: boolean;
  readonly [REJECTED] = true;

  constructor(message: string, opts: RejectionOptions = {}) {
    super(message);
    this.name = "MessageRejected";
    this.code = opts.code ?? "";
    this.throttled = opts.throttled ?? false;
    this.temporary = (opts.temporary ?? false) || this.throttled;
  }
}

export function isAdapterError(e: unknown): e is AdapterError {
  return typeof e === "object" && e !== null && (e as Record<symbol, unknown>)[BRAND] === true;
}

export function isMessageRejected(e: unknown): e is MessageRejected {
  return typeof e === "object" && e !== null && (e as Record<symbol, unknown>)[REJECTED] === true;
}

/** Trims a text and cuts it at the protocol's limit. */
export function cleanText(s: string, max = 2000): string {
  const t = s.trim();
  return t.length > max ? t.slice(0, max) + "…" : t;
}

/** The wire form of an error; anything unexpected is an internal failure. */
export function toWireError(e: unknown): WireError {
  if (isAdapterError(e)) {
    const out: WireError = { code: e.code, message: cleanText(e.message || e.code) };
    if (e.status !== undefined) out.status = e.status;
    if (e.providerCode) out.providerCode = cleanText(e.providerCode, 200);
    if (e.retryAfter !== undefined && e.retryAfter >= 0) out.retryAfter = e.retryAfter;
    return out;
  }
  if (isMessageRejected(e)) {
    return { code: "provider_error", message: cleanText(e.message), ...(e.code ? { providerCode: e.code } : {}) };
  }
  const msg = e instanceof Error ? e.message : String(e);
  return { code: "internal", message: cleanText("The adapter failed: " + msg) };
}

export function statusOf(code: ErrorCode): number {
  return ERROR_STATUS[code] ?? 500;
}
