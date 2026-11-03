/**
 * The answers src/early.ts asked for while the app's scripts loaded. Each is
 * used once, by the first fetch of that query; later fetches ask the API.
 * An answer is the parsed body of a 200, or undefined: the caller then asks
 * through the API client.
 */
export interface EarlyAnswers {
  me?: Promise<unknown>
  workspace?: { slug: string; me: Promise<unknown> }
}

function answers(): EarlyAnswers | undefined {
  return typeof window === 'undefined' ? undefined : (window as { __rcEarly?: EarlyAnswers }).__rcEarly
}

/** The early /me answer, once. */
export function takeEarlyMe(): Promise<unknown> | undefined {
  const a = answers()
  const me = a?.me
  if (a) a.me = undefined
  return me
}

/** The early answer for the workspace `slug`, once. */
export function takeEarlyWorkspace(slug: string): Promise<unknown> | undefined {
  const a = answers()
  const ws = a?.workspace
  if (!a || !ws || ws.slug !== slug) return undefined
  a.workspace = undefined
  return ws.me
}
