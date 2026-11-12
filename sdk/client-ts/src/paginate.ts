import type { FetchOptions } from "openapi-fetch";

import type { ReputeCastClient } from "./client.js";
import { ReputeCastError } from "./errors.js";
import type { paths } from "./schema.js";

interface CursorPage {
  items: unknown[];
  nextCursor?: string;
}

/** The GET operations that page with a cursor: they answer `items` and `nextCursor` and take `cursor`. */
export type PaginatedPath = {
  [P in keyof paths]: paths[P] extends {
    get: {
      parameters: { query?: { cursor?: string } };
      responses: { 200: { content: { "application/json": CursorPage } } };
    };
  }
    ? P
    : never;
}[keyof paths];

type Operation<P extends PaginatedPath> = paths[P]["get"];

/** One item of a paginated operation's pages. */
export type PaginatedItem<P extends PaginatedPath> = Operation<P>["responses"][200]["content"]["application/json"]["items"][number];

type Get = (path: string, init: { params?: { query?: Record<string, unknown> } }) => Promise<{ data?: CursorPage; error?: unknown; response: Response }>;

/**
 * Iterates over every item of a paginated list, fetching the next page
 * with its cursor as the loop reaches it.
 *
 *   for await (const contact of paginate(rc, "/api/v1/workspaces/{workspace}/contacts", {
 *     params: { path: { workspace: "acme-newsletter" }, query: { pageSize: 100 } },
 *   })) { … }
 */
export async function* paginate<P extends PaginatedPath>(
  client: ReputeCastClient,
  path: P,
  init: FetchOptions<Operation<P>>,
): AsyncGenerator<PaginatedItem<P>, void, undefined> {
  const get = client.GET as unknown as Get;
  const options = init as { params?: { query?: Record<string, unknown> } };
  let cursor = options.params?.query?.cursor as string | undefined;
  for (;;) {
    const query = { ...options.params?.query, ...(cursor ? { cursor } : {}) };
    const { data, error, response } = await get(path, { ...options, params: { ...options.params, query } });
    if (!response.ok || !data) throw ReputeCastError.fromResponse(response, error);
    yield* data.items as PaginatedItem<P>[];
    if (!data.nextCursor || data.nextCursor === cursor) return;
    cursor = data.nextCursor;
  }
}
