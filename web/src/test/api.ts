import { setTestFetch } from './fetch'

type Handler = (request: Request) => Response | Promise<Response>

/** A JSON response, like the API's (problems included). */
export function json(body: unknown, status = 200, headers: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': status >= 400 ? 'application/problem+json' : 'application/json', ...headers } })
}

/**
 * Answers fetches to the API, keyed by "METHOD /path". Requests go through
 * the typed client and all its middleware, so non-2xx answers throw ApiError
 * exactly as in the app. Unmocked calls fail the test. Returns a function
 * that removes the mock.
 */
export function mockApi(routes: Record<string, Handler>): () => void {
  setTestFetch(async (input, init) => {
    const request = input instanceof Request ? input : new Request(input, init)
    const key = `${request.method} ${new URL(request.url).pathname}`
    const handler = routes[key]
    if (!handler) throw new Error(`Unmocked API call: ${key}`)
    return handler(request)
  })
  return () => setTestFetch(undefined)
}
