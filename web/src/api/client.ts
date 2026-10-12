import createFetchClient, { type Middleware } from 'openapi-fetch'
import createClient from 'openapi-react-query'
import type { components, paths } from './schema'

export type Schemas = components['schemas']
export type Problem = Schemas['Problem']

/** Error thrown for non-2xx responses; carries the RFC 9457 problem body. */
export class ApiError extends Error {
  readonly status: number
  readonly problem: Problem
  constructor(status: number, problem: Problem) {
    super(problem.detail || problem.title || `HTTP ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.problem = problem
  }
  get code(): string | undefined {
    return this.problem.code
  }
}

const listeners = new Set<(e: ApiError) => void>()

/** Subscribe to authentication problems (401, password change, 2FA setup). */
export function onAuthProblem(fn: (e: ApiError) => void): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

const authCodes = new Set(['unauthenticated', 'session_expired', 'password_change_required', 'two_factor_setup_required'])

const throwOnError: Middleware = {
  async onResponse({ response }) {
    if (response.ok) return undefined
    let problem: Problem
    try {
      problem = (await response.clone().json()) as Problem
    } catch {
      problem = { status: response.status, title: response.statusText, detail: `The server answered ${response.status}.` } as Problem
    }
    const err = new ApiError(response.status, problem)
    if (response.status === 401 || (problem.code && authCodes.has(problem.code))) {
      listeners.forEach((l) => l(err))
    }
    throw err
  },
}

/** Typed fetch client for the ReputeCast API (same origin, cookie session). */
export const http = createFetchClient<paths>({ baseUrl: '', credentials: 'same-origin', headers: { Accept: 'application/json' } })
http.use(throwOnError)

/** TanStack Query bindings: $api.useQuery('get', '/api/v1/meta'). */
export const $api = createClient(http)

/** Human message for an unknown error. */
export function errorMessage(e: unknown): string {
  if (e instanceof ApiError) return e.message
  if (e instanceof TypeError) return "Can't reach the server. Check your connection; nothing was lost."
  if (e instanceof Error) return e.message
  return 'Something went wrong.'
}
