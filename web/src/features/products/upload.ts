import { ApiError } from '@/api/client'

/**
 * A product file's upload in chunks (start-product-upload, then one PUT per
 * chunk at the bytes received so far, then finish-product-upload, which
 * checks the type from the content). A chunk that fails on the network or
 * with a 5xx is sent again; a 409 upload_offset says where the server
 * stands and the upload continues from there.
 */

/** Where an upload stands (the API's UploadState). */
export interface UploadProgressState {
  id: string
  received: number
  size: number
  chunkBytes: number
}

/** The calls an upload makes; the screen binds them to the API, tests to fakes. */
export interface UploadTransport<F> {
  start(input: { name: string; size: number; replaces?: string }): Promise<UploadProgressState>
  /** Sends a chunk; `onSent` reports the chunk's bytes sent so far, for progress within a chunk. */
  chunk(id: string, offset: number, body: Blob, onSent?: (bytes: number) => void): Promise<UploadProgressState>
  finish(id: string): Promise<F>
  cancel(id: string): Promise<void>
}

export interface UploadOptions {
  /** A file the new one replaces once it finished uploading. */
  replaces?: string
  signal?: AbortSignal
  /** The upload's id, as soon as the server gave it. */
  onStart?: (id: string) => void
  /** Bytes the server has, after each chunk. */
  onProgress?: (received: number, size: number) => void
  /** Called once every byte arrived, while the server checks the content. */
  onChecking?: () => void
  /** Attempts per chunk before the upload stops (default 3). */
  attempts?: number
  /** Waits between attempts: 1 s, then 2 s (tests pass their own). */
  wait?: (ms: number) => Promise<void>
}

/** The chunk size when the server doesn't say (4 MB, the API's limit). */
export const CHUNK_BYTES = 4 << 20

/** An upload that stopped: the bytes the server had, and why. */
export class UploadStopped extends Error {
  readonly received: number
  readonly uploadId: string
  constructor(uploadId: string, received: number, cause: unknown) {
    super(cause instanceof Error ? cause.message : 'The upload stopped.', { cause })
    this.name = 'UploadStopped'
    this.uploadId = uploadId
    this.received = received
  }
}

/** A failure worth another attempt: the network, a timeout, a 5xx. */
export function retryable(e: unknown): boolean {
  if (e instanceof ApiError) return e.status >= 500 || e.status === 408 || e.status === 429
  return e instanceof TypeError
}

export function isAbort(e: unknown): boolean {
  return (e instanceof DOMException || e instanceof Error) && e.name === 'AbortError'
}

/** The offset a 409 upload_offset (or upload_incomplete) reports. */
export function offsetOf(e: unknown): number | undefined {
  if (!(e instanceof ApiError) || e.status !== 409) return undefined
  if (e.code !== 'upload_offset' && e.code !== 'upload_incomplete') return undefined
  const r = e.problem.meta?.received
  return typeof r === 'number' && r >= 0 ? r : undefined
}

function aborted(signal?: AbortSignal) {
  if (signal?.aborted) throw new DOMException('The upload was cancelled.', 'AbortError')
}

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

/**
 * Measures an upload's speed: give it the bytes received so far, it
 * answers bytes per second since the first report (undefined for the first
 * half second, when the number would only be noise).
 */
export function speedMeter(clock: () => number = Date.now): (received: number) => number | undefined {
  const begun = clock()
  let first: number | undefined
  return (received) => {
    if (first === undefined) first = received
    const secs = (clock() - begun) / 1000
    return secs > 0.5 && received > first ? (received - first) / secs : undefined
  }
}

/**
 * Uploads a file and returns what finish-product-upload answered. Throws
 * UploadStopped when a chunk keeps failing, the abort error when cancelled,
 * and the API's error when the server refuses the file (too big, or a type
 * that doesn't match its name).
 */
export async function uploadFile<F>(api: UploadTransport<F>, file: Blob & { name: string }, opts: UploadOptions = {}): Promise<F> {
  const wait = opts.wait ?? sleep
  const attempts = Math.max(1, opts.attempts ?? 3)
  aborted(opts.signal)
  const state = await api.start({ name: file.name, size: file.size, replaces: opts.replaces })
  const id = state.id
  opts.onStart?.(id)
  const chunk = state.chunkBytes > 0 ? state.chunkBytes : CHUNK_BYTES
  let received = Math.min(state.received, file.size)
  // Failures in a row (network, 5xx) and offset corrections in a row.
  let failures = 0
  let moves = 0
  opts.onProgress?.(received, file.size)
  const again = async (e: unknown) => {
    if (isAbort(e) || opts.signal?.aborted) throw e
    const at = offsetOf(e)
    if (at !== undefined && moves < 5) {
      received = Math.min(at, file.size)
      moves++
      opts.onProgress?.(received, file.size)
      return
    }
    if (retryable(e) && failures < attempts - 1) {
      failures++
      await wait(1000 * 2 ** (failures - 1))
      aborted(opts.signal)
      return
    }
    throw new UploadStopped(id, received, e)
  }
  for (;;) {
    while (received < file.size) {
      aborted(opts.signal)
      const at = received
      try {
        const next = await api.chunk(id, at, file.slice(at, Math.min(file.size, at + chunk)), (sent) => opts.onProgress?.(Math.min(file.size, at + sent), file.size))
        received = Math.min(next.received, file.size)
        failures = 0
        moves = 0
        opts.onProgress?.(received, file.size)
      } catch (e) {
        await again(e)
      }
    }
    aborted(opts.signal)
    opts.onChecking?.()
    try {
      return await api.finish(id)
    } catch (e) {
      // Not every byte arrived after all: send the rest. Other refusals
      // (a type that doesn't match its name) are the caller's to show.
      if (offsetOf(e) !== undefined || retryable(e)) {
        await again(e)
        continue
      }
      throw e
    }
  }
}
