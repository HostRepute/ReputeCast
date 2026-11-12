import type { components } from "./schema.js";

/** The RFC 9457 problem document the API answers errors with. */
export type Problem = components["schemas"]["Problem"];

/** One field's error in a problem's `errors`. */
export type ProblemDetail = components["schemas"]["ErrorDetail"];

/**
 * An error answer of the API (an RFC 9457 problem): its HTTP status, the
 * stable `code` (permission_denied, validation_failed, not_found…), the
 * sentence that explains it, the fields at fault and, for a missing
 * permission, the members who can help.
 */
export class ReputeCastError extends Error {
  override readonly name = "ReputeCastError";
  readonly status: number;
  /** Stable machine-readable code, e.g. "permission_denied"; "" when the answer had none. */
  readonly code: string;
  readonly title: string;
  readonly detail: string;
  readonly type: string;
  /** The fields at fault, for validation errors. */
  readonly errors: ProblemDetail[];
  /** The missing permission key, for permission errors. */
  readonly permission: string;
  /** Members who hold the missing permission. */
  readonly helpers: string[];
  /** Structured details for some codes (a sign-in lock, an error reference). */
  readonly meta: Record<string, unknown>;
  /** The HTTP response, when the error came from one. */
  readonly response: Response | undefined;

  constructor(status: number, problem: Partial<Problem> = {}, response?: Response) {
    super(problem.detail || problem.title || `The API answered HTTP ${status}.`);
    this.status = status;
    this.code = problem.code ?? "";
    this.title = problem.title ?? "";
    this.detail = problem.detail ?? "";
    this.type = problem.type ?? "";
    this.errors = problem.errors ?? [];
    this.permission = problem.permission ?? "";
    this.helpers = problem.helpers ?? [];
    this.meta = problem.meta ?? {};
    this.response = response;
  }

  /** The error for an answer whose body (`error` from the client) is a problem, other JSON or text. */
  static fromResponse(response: Response, body: unknown): ReputeCastError {
    if (body && typeof body === "object" && !Array.isArray(body)) {
      return new ReputeCastError(response.status, body as Partial<Problem>, response);
    }
    const text = typeof body === "string" ? body.trim() : "";
    return new ReputeCastError(response.status, { title: response.statusText, ...(text ? { detail: text.slice(0, 500) } : {}) }, response);
  }
}

interface ClientResult {
  data?: unknown;
  error?: unknown;
  response: Response;
}

/** What `unwrap` returns for a result: its data, or undefined for an answer without a body. */
export type Unwrapped<R extends ClientResult> = [Exclude<R["data"], undefined>] extends [never] ? undefined : Exclude<R["data"], undefined>;

/**
 * Returns the data of a client call, or throws a ReputeCastError when the
 * API answered with an error.
 *
 *   const lists = unwrap(await rc.GET("/api/v1/workspaces/{workspace}/lists", { params: { path: { workspace } } }));
 */
export function unwrap<R extends ClientResult>(result: R): Unwrapped<R> {
  if (!result.response.ok) throw ReputeCastError.fromResponse(result.response, result.error);
  return result.data as Unwrapped<R>;
}
