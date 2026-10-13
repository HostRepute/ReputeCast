import { AdapterError, HttpError, ProviderError, cleanText } from "./errors.ts";

export interface HttpResponse<T = any> {
  status: number;
  headers: Headers;
  /** The JSON body when there is one, otherwise the text. */
  data: T;
  text: string;
}

export type QueryValue = string | number | boolean | null | undefined;

export interface RequestOptions {
  query?: Record<string, QueryValue | readonly QueryValue[]>;
  headers?: Record<string, string>;
  /** JSON-encoded unless it is a string or bytes (then set content-type yourself). */
  body?: unknown;
  /** Shorter than the call's deadline. */
  timeoutMs?: number;
}

/**
 * ctx.http: fetch on the adapter's base URL with JSON helpers. A non-2xx
 * answer throws an HttpError whose code ReputeCast acts on: 401/403 →
 * provider_auth, 429 → provider_throttled, 408 and 5xx → provider_unavailable,
 * other 4xx → provider_error. Network errors and timeouts are
 * provider_unavailable. Only hosts the manifest declares can be reached.
 */
export interface HttpClient {
  request<T = any>(method: string, path: string, opts?: RequestOptions): Promise<HttpResponse<T>>;
  get<T = any>(path: string, opts?: RequestOptions): Promise<HttpResponse<T>>;
  post<T = any>(path: string, body?: unknown, opts?: RequestOptions): Promise<HttpResponse<T>>;
  put<T = any>(path: string, body?: unknown, opts?: RequestOptions): Promise<HttpResponse<T>>;
  patch<T = any>(path: string, body?: unknown, opts?: RequestOptions): Promise<HttpResponse<T>>;
  delete<T = any>(path: string, opts?: RequestOptions): Promise<HttpResponse<T>>;
}

export interface HttpClientOptions {
  /** "https://api.example-esp.com"; paths are joined to it. */
  baseUrl: string;
  headers?: Record<string, string>;
  /** Aborts every request (the call's deadline). */
  signal?: AbortSignal;
  /** The network permission check; false refuses the request. */
  allow?: (url: URL) => boolean;
  /** Reads the provider's error code from a failed answer. */
  errorCode?: (data: unknown, status: number) => string | undefined;
  /** Reads the provider's error message from a failed answer. */
  errorMessage?: (data: unknown, status: number) => string | undefined;
  /** The provider's name in messages ("Example ESP"). */
  name?: string;
  /** Largest answer read (16 MiB). */
  maxResponseBytes?: number;
  fetch?: typeof fetch;
}

/** Common error shapes: {error: {code, message}}, {code, message}, {errors: [{code, message}]}. */
function pick(data: unknown, key: "code" | "message"): string | undefined {
  if (typeof data !== "object" || data === null) return undefined;
  const d = data as Record<string, unknown>;
  const fromObj = (o: unknown): string | undefined => {
    if (typeof o !== "object" || o === null) return undefined;
    const v = (o as Record<string, unknown>)[key];
    return typeof v === "string" || typeof v === "number" ? String(v) : undefined;
  };
  if (typeof d["error"] === "string" && key === "message") return d["error"];
  return fromObj(d["error"]) ?? fromObj(d) ?? (Array.isArray(d["errors"]) ? fromObj(d["errors"][0]) : undefined);
}

async function readLimited(res: Response, max: number): Promise<string> {
  if (!res.body) return "";
  const reader = res.body.getReader();
  const chunks: Uint8Array[] = [];
  let size = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    size += value.byteLength;
    if (size > max) {
      await reader.cancel();
      throw ProviderError.refused(`the answer is larger than ${max} bytes`);
    }
    chunks.push(value);
  }
  return Buffer.concat(chunks).toString("utf8");
}

function retryAfter(h: Headers): number | undefined {
  const v = h.get("retry-after");
  if (!v) return undefined;
  const n = Number(v);
  if (Number.isFinite(n)) return Math.max(0, n);
  const t = Date.parse(v);
  return Number.isNaN(t) ? undefined : Math.max(0, Math.round((t - Date.now()) / 1000));
}

export function createHttpClient(o: HttpClientOptions): HttpClient {
  const name = o.name ?? "The provider";
  const max = o.maxResponseBytes ?? 16 << 20;
  const doFetch = o.fetch ?? fetch;

  async function request<T>(method: string, path: string, opts: RequestOptions = {}): Promise<HttpResponse<T>> {
    let url: URL;
    try {
      url = /^https?:\/\//i.test(path) ? new URL(path) : new URL(o.baseUrl.replace(/\/+$/, "") + (path.startsWith("/") ? path : "/" + path));
    } catch {
      throw new AdapterError("internal", `ctx.http can't build a URL from ${JSON.stringify(path)}`);
    }
    for (const [k, v] of Object.entries(opts.query ?? {})) {
      for (const x of Array.isArray(v) ? v : [v]) {
        if (x !== undefined && x !== null) url.searchParams.append(k, String(x));
      }
    }
    if (o.allow && !o.allow(url)) {
      throw new AdapterError("internal", `ctx.http refused ${url.host}: it isn't in the manifest's permissions.network`);
    }
    const headers = new Headers({ accept: "application/json", ...o.headers, ...opts.headers });
    let body: string | Uint8Array | undefined;
    if (opts.body !== undefined) {
      if (typeof opts.body === "string" || opts.body instanceof Uint8Array) {
        body = opts.body;
      } else {
        body = JSON.stringify(opts.body);
        if (!headers.has("content-type")) headers.set("content-type", "application/json");
      }
    }
    const signals = [o.signal, opts.timeoutMs ? AbortSignal.timeout(opts.timeoutMs) : undefined].filter((s): s is AbortSignal => s !== undefined);
    const signal = signals.length === 0 ? undefined : signals.length === 1 ? signals[0] : AbortSignal.any(signals);
    let res: Response;
    try {
      res = await doFetch(url, { method, headers, redirect: "manual", ...(body === undefined ? {} : { body }), ...(signal ? { signal } : {}) });
    } catch (e) {
      if (signal?.aborted) throw ProviderError.unavailable(`${name} timed out on ${method} ${url.pathname}`, { cause: e });
      const cause = (e as { cause?: { code?: string; message?: string } }).cause;
      throw ProviderError.unavailable(`${name} can't be reached: ${cause?.code ?? cause?.message ?? (e as Error).message}`, { cause: e });
    }
    let text: string;
    try {
      text = await readLimited(res, max);
    } catch (e) {
      if (e instanceof AdapterError) throw e;
      if (signal?.aborted) throw ProviderError.unavailable(`${name} timed out on ${method} ${url.pathname}`, { cause: e });
      throw ProviderError.unavailable(`${name}'s answer was cut off`, { cause: e });
    }
    let data: unknown = text;
    if (text !== "" && (res.headers.get("content-type")?.includes("json") || /^\s*[[{]/.test(text))) {
      try {
        data = JSON.parse(text);
      } catch {
        data = text;
      }
    }
    if (res.status >= 200 && res.status < 300) return { status: res.status, headers: res.headers, data: data as T, text };
    const code = o.errorCode?.(data, res.status) ?? pick(data, "code");
    const detail = o.errorMessage?.(data, res.status) ?? pick(data, "message") ?? (typeof data === "string" ? data : "");
    const message = cleanText(`${name} answered ${res.status} on ${method} ${url.pathname}` + (detail ? `: ${detail}` : ""), 500);
    const s = res.status;
    if (s >= 300 && s < 400) {
      throw new HttpError("provider_error", cleanText(`${name} redirected ${method} ${url.pathname} to ${res.headers.get("location") ?? "another URL"}; redirects are not followed`, 500), s, data, res.headers, code);
    }
    if (s === 401 || s === 403) throw new HttpError("provider_auth", message, s, data, res.headers, code);
    if (s === 429) throw new HttpError("provider_throttled", message, s, data, res.headers, code, retryAfter(res.headers));
    if (s === 408 || s >= 500) throw new HttpError("provider_unavailable", message, s, data, res.headers, code, retryAfter(res.headers));
    throw new HttpError("provider_error", message, s, data, res.headers, code);
  }

  return {
    request,
    get: (path, opts) => request("GET", path, opts),
    post: (path, body, opts) => request("POST", path, { ...opts, body }),
    put: (path, body, opts) => request("PUT", path, { ...opts, body }),
    patch: (path, body, opts) => request("PATCH", path, { ...opts, body }),
    delete: (path, opts) => request("DELETE", path, opts),
  };
}

/**
 * Builds the network permission check: hosts from the manifest
 * ("api.example.com", "*.example.com", "host:port", "settings:<key>" for the
 * host of a URL in the connection's settings) plus extra hosts the operator
 * allows (REPUTECAST_ADAPTER_ALLOW_HOSTS, local fakes in tests).
 */
export function hostAllowlist(entries: readonly string[], settings: Record<string, unknown> = {}, extra: readonly string[] = []): (url: URL) => boolean {
  const rules: string[] = [];
  for (const e of [...entries, ...extra]) {
    if (e.startsWith("settings:")) {
      const v = settings[e.slice("settings:".length)];
      if (typeof v === "string" && v !== "") {
        try {
          const u = new URL(/^https?:\/\//i.test(v) ? v : "https://" + v);
          rules.push(u.host.toLowerCase());
        } catch {
          // An unusable setting allows nothing.
        }
      }
      continue;
    }
    rules.push(e.trim().toLowerCase());
  }
  return (url: URL) => {
    const host = url.hostname.toLowerCase().replace(/^\[|\]$/g, "");
    const port = url.port;
    return rules.some((r) => {
      const [rh, rp] = r.startsWith("[") ? [r.slice(1, r.indexOf("]")), r.slice(r.indexOf("]") + 2)] : (r.split(":") as [string, string?]);
      if (rp && rp !== port && !(rp === "443" && port === "" && url.protocol === "https:") && !(rp === "80" && port === "" && url.protocol === "http:")) return false;
      if (rh.startsWith("*.")) return host.endsWith(rh.slice(1)) && host.length > rh.length - 1;
      return host === rh;
    });
  };
}

/** Hosts the operator allows on top of the manifest (REPUTECAST_ADAPTER_ALLOW_HOSTS). */
export function allowHostsFromEnv(value: string | undefined = process.env["REPUTECAST_ADAPTER_ALLOW_HOSTS"]): string[] {
  return (value ?? "")
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s !== "");
}
