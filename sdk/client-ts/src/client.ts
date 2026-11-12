import createFetchClient, { type Client, type Middleware } from "openapi-fetch";

import { type Fetch, withRetries } from "./retry.js";
import type { paths } from "./schema.js";
import { USER_AGENT } from "./version.js";

export interface ClientOptions {
  /** The instance's URL, e.g. "https://news.example.com" (the paths start with /api/v1). */
  baseUrl: string;
  /** An API key ("rc_live_…") or an MCP agent token ("rc_agent_…"). */
  apiKey: string;
  /** The fetch to send requests with; globalThis.fetch by default. */
  fetch?: Fetch;
  /**
   * How many times a request is sent again after a 429, 502, 503 or 504:
   * GET, HEAD, PUT and DELETE requests, and POSTs with an Idempotency-Key.
   * 2 by default; 0 turns retries off.
   */
  maxRetries?: number;
}

/** A client for every operation an API key may call, typed from the API's OpenAPI document. */
export type ReputeCastClient = Client<paths>;

/**
 * Creates a client for a ReputeCast instance.
 *
 *   const rc = createClient({ baseUrl: "https://news.example.com", apiKey: process.env.REPUTECAST_KEY! });
 *   const lists = unwrap(await rc.GET("/api/v1/workspaces/{workspace}/lists", { params: { path: { workspace: "acme-newsletter" } } }));
 */
export function createClient({ baseUrl, apiKey, fetch, maxRetries = 2 }: ClientOptions): ReputeCastClient {
  if (!baseUrl) throw new TypeError("createClient: baseUrl is the instance's URL, e.g. https://news.example.com.");
  if (!apiKey) throw new TypeError("createClient: apiKey is empty; create a key in Settings › Keys & MCP.");
  if (!Number.isInteger(maxRetries) || maxRetries < 0) throw new TypeError("createClient: maxRetries is a whole number, 0 or more.");
  const send = fetch ?? ((request: Request) => globalThis.fetch(request));
  const client = createFetchClient<paths>({ baseUrl: baseUrl.replace(/\/+$/, ""), fetch: withRetries(send, { maxRetries }) });
  client.use(authenticate(apiKey));
  return client;
}

function authenticate(apiKey: string): Middleware {
  const authorization = `Bearer ${apiKey}`;
  return {
    onRequest({ request }) {
      request.headers.set("Authorization", authorization);
      request.headers.set("User-Agent", USER_AGENT);
    },
  };
}
