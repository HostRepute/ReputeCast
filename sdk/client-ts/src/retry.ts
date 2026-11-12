/** The answers a retry may fix: rate limited, or the server briefly unavailable. */
export const RETRY_STATUSES: ReadonlySet<number> = new Set([429, 502, 503, 504]);

/** The longest wait between two attempts, whatever Retry-After asks. */
export const MAX_RETRY_DELAY_MS = 30_000;

const BASE_DELAY_MS = 500;
const IDEMPOTENT_METHODS: Record<string, true> = { GET: true, HEAD: true, PUT: true, DELETE: true };

/**
 * Whether a request may be sent again: its method is idempotent, or it is
 * a POST with an Idempotency-Key (the server answers a repeat with the
 * first result instead of acting twice).
 */
export function isRetryable(request: Request): boolean {
  return IDEMPOTENT_METHODS[request.method] === true || (request.method === "POST" && request.headers.has("Idempotency-Key"));
}

/**
 * How long to wait before retry number `attempt` (from 0): the answer's
 * Retry-After (seconds or an HTTP date) when it has one, else exponential
 * backoff from 0.5 s with jitter; never more than 30 s.
 */
export function retryDelay(attempt: number, retryAfter: string | null, now: number = Date.now(), random: () => number = Math.random): number {
  if (retryAfter) {
    const value = retryAfter.trim();
    let ms: number | undefined;
    if (/^\d+$/.test(value)) {
      ms = Number(value) * 1000;
    } else {
      const at = Date.parse(value);
      if (!Number.isNaN(at)) ms = Math.max(0, at - now);
    }
    if (ms !== undefined) return Math.min(ms, MAX_RETRY_DELAY_MS);
  }
  const ceiling = Math.min(BASE_DELAY_MS * 2 ** attempt, MAX_RETRY_DELAY_MS);
  return Math.round(ceiling / 2 + (random() * ceiling) / 2);
}

export type Fetch = (input: Request) => Promise<Response>;

export interface RetryOptions {
  maxRetries: number;
  /** Waits ms milliseconds (tests replace it). */
  sleep?: (ms: number, signal: AbortSignal) => Promise<void>;
}

/** Wraps fetch to retry 429, 502, 503 and 504 answers to retryable requests. */
export function withRetries(fetch: Fetch, { maxRetries, sleep = wait }: RetryOptions): Fetch {
  return async (request) => {
    if (maxRetries <= 0 || !isRetryable(request)) return fetch(request);
    for (let attempt = 0; ; attempt++) {
      const last = attempt >= maxRetries;
      // Each attempt sends a copy, so the body can be sent again.
      const response = await fetch(last ? request : request.clone());
      if (last || !RETRY_STATUSES.has(response.status)) return response;
      const delay = retryDelay(attempt, response.headers.get("Retry-After"));
      await response.body?.cancel();
      await sleep(delay, request.signal);
    }
  };
}

// The executor form: Promise.withResolvers needs Node 22, the SDK runs on 20.
function wait(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal.aborted) {
      reject(signal.reason);
      return;
    }
    const onAbort = () => {
      clearTimeout(timer);
      reject(signal.reason);
    };
    const timer = setTimeout(() => {
      signal.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    signal.addEventListener("abort", onAbort, { once: true });
  });
}
