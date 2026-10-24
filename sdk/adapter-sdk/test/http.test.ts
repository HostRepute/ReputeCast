import { createServer } from "node:http";
import type { AddressInfo } from "node:net";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { isAdapterError } from "../src/errors.ts";
import { createHttpClient, hostAllowlist } from "../src/http.ts";

let base = "";
const server = createServer((req, res) => {
  const url = new URL(req.url ?? "/", "http://x");
  const reply = (status: number, body: unknown, headers: Record<string, string> = {}) => {
    res.writeHead(status, { "content-type": "application/json", ...headers });
    res.end(JSON.stringify(body));
  };
  switch (url.pathname) {
    case "/ok":
      return reply(200, { id: "msg_1", auth: req.headers["authorization"], q: url.searchParams.getAll("tag") });
    case "/echo": {
      let b = "";
      req.on("data", (c: Buffer) => (b += c.toString()));
      req.on("end", () => reply(200, { body: JSON.parse(b), type: req.headers["content-type"] }));
      return;
    }
    case "/401":
      return reply(401, { error: { code: "invalid_api_key", message: "Bad key" } });
    case "/422":
      return reply(422, { errors: [{ code: "invalid_recipient", message: "No such mailbox" }] });
    case "/429":
      return reply(429, { message: "slow down" }, { "retry-after": "7" });
    case "/503":
      return reply(503, "maintenance");
    case "/302":
      res.writeHead(302, { location: "https://elsewhere.example/" });
      return res.end();
    case "/slow":
      setTimeout(() => reply(200, {}), 2_000);
      return;
    case "/big":
      return reply(200, { pad: "x".repeat(2_000) });
  }
  reply(404, { error: "not found" });
});

beforeAll(async () => {
  await new Promise<void>((r) => server.listen(0, "127.0.0.1", r));
  base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
});
afterAll(() => {
  server.closeAllConnections();
  server.close();
});

async function failure(p: Promise<unknown>) {
  try {
    await p;
  } catch (e) {
    if (!isAdapterError(e)) throw e;
    return e;
  }
  throw new Error("expected a failure");
}

describe("ctx.http", () => {
  const client = (o: Partial<Parameters<typeof createHttpClient>[0]> = {}) =>
    createHttpClient({ baseUrl: base, name: "Fake ESP", headers: { authorization: "Bearer k" }, ...o });

  it("sends JSON and reads JSON", async () => {
    const r = await client().get<{ id: string; auth: string; q: string[] }>("/ok", { query: { tag: ["a", "b"], skip: undefined } });
    expect(r.status).toBe(200);
    expect(r.data).toEqual({ id: "msg_1", auth: "Bearer k", q: ["a", "b"] });
    const e = await client().post<{ body: unknown; type: string }>("/echo", { to: "x" });
    expect(e.data).toEqual({ body: { to: "x" }, type: "application/json" });
  });

  it("joins a base URL that ends in slashes", async () => {
    const r = await client({ baseUrl: base + "///" }).get<{ id: string }>("/ok");
    expect(r.data.id).toBe("msg_1");
  });

  it("maps provider failures to protocol errors", async () => {
    const auth = await failure(client().get("/401"));
    expect([auth.code, auth.status, auth.providerCode]).toEqual(["provider_auth", 401, "invalid_api_key"]);
    expect(auth.message).toBe("Fake ESP answered 401 on GET /401: Bad key");
    const refused = await failure(client().post("/422", {}));
    expect([refused.code, refused.providerCode]).toEqual(["provider_error", "invalid_recipient"]);
    const slow = await failure(client().get("/429"));
    expect([slow.code, slow.retryAfter]).toEqual(["provider_throttled", 7]);
    const down = await failure(client().get("/503"));
    expect(down.code).toBe("provider_unavailable");
    const moved = await failure(client().get("/302"));
    expect(moved.code).toBe("provider_error");
    expect(moved.message).toContain("redirects are not followed");
  });

  it("times out and survives network errors", async () => {
    const t = await failure(client().get("/slow", { timeoutMs: 50 }));
    expect([t.code, t.message]).toEqual(["provider_unavailable", "Fake ESP timed out on GET /slow"]);
    const unreachable = await failure(createHttpClient({ baseUrl: "http://127.0.0.1:1" }).get("/x"));
    expect(unreachable.code).toBe("provider_unavailable");
    expect(unreachable.message).toContain("can't be reached");
    const big = await failure(client({ maxResponseBytes: 100 }).get("/big"));
    expect(big.code).toBe("provider_error");
  });

  it("only reaches the declared hosts", async () => {
    const allow = hostAllowlist(["api.example.com"], {}, []);
    const e = await failure(client({ allow }).get("/ok"));
    expect([e.code, e.message]).toEqual(["internal", `ctx.http refused ${new URL(base).host}: it isn't in the manifest's permissions.network`]);
    expect((await client({ allow: hostAllowlist([], {}, ["127.0.0.1"]) }).get("/ok")).status).toBe(200);
  });
});

describe("hostAllowlist", () => {
  it.each([
    ["api.example.com", "https://api.example.com/v1", true],
    ["api.example.com", "https://api.example.com:8443/v1", true],
    ["api.example.com", "https://evil.example.com/", false],
    ["api.example.com", "https://api.example.com.evil.net/", false],
    ["*.example.com", "https://eu.api.example.com/", true],
    ["*.example.com", "https://example.com/", false],
    ["api.example.com:443", "https://api.example.com/", true],
    ["api.example.com:443", "https://api.example.com:8443/", false],
    ["settings:server_url", "https://postal.acme.io/api", true],
    ["settings:server_url", "https://other.acme.io/api", false],
  ])("%s allows %s: %s", (rule, url, want) => {
    expect(hostAllowlist([rule], { server_url: "https://postal.acme.io" })(new URL(url))).toBe(want);
  });
});
