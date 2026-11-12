import { describe, expect, it } from "vitest";

import { createClient, paginate, ReputeCastError, unwrap } from "../src/index.ts";

const base = "https://news.example.com";
const workspace = { path: { workspace: "acme-newsletter" } };

function serve(status: number, body: string | null, contentType = "application/problem+json") {
  return async () => new Response(body, { status, headers: body === null ? {} : { "Content-Type": contentType } });
}

describe("unwrap", () => {
  it("returns the data of a success", async () => {
    const rc = createClient({ baseUrl: base, apiKey: "k", fetch: serve(200, '{"items":[{"id":"l1","name":"Weekly Digest"}]}', "application/json") });
    const lists = unwrap(await rc.GET("/api/v1/workspaces/{workspace}/lists", { params: workspace }));
    expect(lists.items[0]?.name).toBe("Weekly Digest");
  });

  it("throws the problem as a ReputeCastError", async () => {
    const problem = {
      type: "about:blank",
      status: 403,
      title: "Forbidden",
      detail: "You need members:view. Linh Tran can give it to you.",
      code: "permission_denied",
      permission: "members:view",
      helpers: ["Linh Tran"],
      errors: [{ location: "path.workspace", message: "No." }],
      meta: { reference: "err_7KQ2M4XA" },
    };
    const rc = createClient({ baseUrl: base, apiKey: "k", fetch: serve(403, JSON.stringify(problem)) });
    const err = await rc.GET("/api/v1/workspaces/{workspace}/lists", { params: workspace }).then(unwrap, (e: unknown) => e).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ReputeCastError);
    const e = err as ReputeCastError;
    expect(e.status).toBe(403);
    expect(e.code).toBe("permission_denied");
    expect(e.message).toBe(problem.detail);
    expect(e.detail).toBe(problem.detail);
    expect(e.title).toBe("Forbidden");
    expect(e.permission).toBe("members:view");
    expect(e.helpers).toEqual(["Linh Tran"]);
    expect(e.errors).toEqual(problem.errors);
    expect(e.meta).toEqual({ reference: "err_7KQ2M4XA" });
    expect(e.response?.status).toBe(403);
  });

  it("explains an answer that isn't a problem", async () => {
    const rc = createClient({ baseUrl: base, apiKey: "k", maxRetries: 0, fetch: serve(502, "Bad gateway from the proxy", "text/plain") });
    const e = (await rc.GET("/api/v1/workspaces/{workspace}/lists", { params: workspace }).then(unwrap).catch((x: unknown) => x)) as ReputeCastError;
    expect(e).toBeInstanceOf(ReputeCastError);
    expect(e.status).toBe(502);
    expect(e.code).toBe("");
    expect(e.detail).toBe("Bad gateway from the proxy");
  });

  it("returns undefined for an answer without a body", async () => {
    const rc = createClient({ baseUrl: base, apiKey: "k", fetch: serve(204, null) });
    const res = await rc.DELETE("/api/v1/workspaces/{workspace}/lists/{id}", { params: { path: { workspace: "acme-newsletter", id: "0193" } } });
    expect(unwrap(res)).toBeUndefined();
  });
});

describe("paginate", () => {
  it("follows nextCursor until the last page", async () => {
    const asked: URL[] = [];
    const pages: Record<string, { items: { id: string }[]; nextCursor?: string }> = {
      "": { items: [{ id: "c1" }, { id: "c2" }], nextCursor: "n1" },
      n1: { items: [{ id: "c3" }], nextCursor: "n2" },
      n2: { items: [] },
    };
    const fetch = async (request: Request) => {
      const url = new URL(request.url);
      asked.push(url);
      const page = pages[url.searchParams.get("cursor") ?? ""];
      return new Response(JSON.stringify({ total: 3, page: 1, pageSize: 2, ...page }), { headers: { "Content-Type": "application/json" } });
    };
    const rc = createClient({ baseUrl: base, apiKey: "k", fetch });
    const ids: string[] = [];
    for await (const contact of paginate(rc, "/api/v1/workspaces/{workspace}/contacts", { params: { ...workspace, query: { pageSize: 2 } } })) {
      ids.push(contact.id);
    }
    expect(ids).toEqual(["c1", "c2", "c3"]);
    expect(asked.map((u) => u.searchParams.get("cursor"))).toEqual([null, "n1", "n2"]);
    expect(asked.every((u) => u.searchParams.get("pageSize") === "2")).toBe(true);
  });

  it("throws a ReputeCastError when a page fails", async () => {
    const rc = createClient({ baseUrl: base, apiKey: "k", fetch: serve(403, '{"status":403,"code":"permission_denied","detail":"No."}') });
    const iterate = async () => {
      for await (const _ of paginate(rc, "/api/v1/workspaces/{workspace}/contacts", { params: workspace })) void _;
    };
    await expect(iterate()).rejects.toMatchObject({ name: "ReputeCastError", status: 403, code: "permission_denied" });
  });
});
