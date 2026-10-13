// Installed by the test setup before any app module loads: the API client
// captures `fetch` when it is created, so tests swap the implementation
// behind this wrapper (see mockApi in ./api.ts) instead of the global.
const nativeFetch: typeof fetch = globalThis.fetch.bind(globalThis)
let current: typeof fetch | undefined

export function setTestFetch(f: typeof fetch | undefined) {
  current = f
}

globalThis.fetch = ((input: RequestInfo | URL, init?: RequestInit) => (current ?? nativeFetch)(input, init)) as typeof fetch
