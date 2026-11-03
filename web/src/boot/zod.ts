/**
 * The app's CSP has no 'unsafe-eval', so zod can't compile its parsers. It
 * finds that out by trying (new Function), and the browser reported the try
 * as a policy violation on every load. zod reads this setting when it makes
 * each object schema, and the routes make theirs as they load, so it is set
 * here, in a chunk of its own that main.tsx imports first (vite.config.ts):
 * before the chunk zod is in runs.
 */
const g = globalThis as { __zod_globalConfig?: { jitless?: boolean } }
g.__zod_globalConfig = { ...g.__zod_globalConfig, jitless: true }

export {}
