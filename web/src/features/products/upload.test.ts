import { describe, expect, it } from 'vitest'
import { ApiError, type Problem } from '@/api/client'
import { UploadStopped, uploadFile, type UploadProgressState, type UploadTransport } from './upload'

const problem = (status: number, code: string, meta?: Record<string, unknown>): ApiError =>
  new ApiError(status, { status, title: code, detail: code, code, meta, type: 'about:blank' } as Problem)

/** A server that keeps the bytes it was sent, like the products API. */
function fakeServer(opts: { chunkBytes?: number; fail?: (call: number, offset: number) => unknown; finish?: () => unknown } = {}) {
  const calls: { offset: number; length: number }[] = []
  let have = 0
  let size = 0
  let canceled = false
  const api: UploadTransport<{ name: string; size: number }> & { calls: typeof calls; canceled: () => boolean; setHave: (n: number) => void } = {
    calls,
    canceled: () => canceled,
    setHave: (n) => {
      have = n
    },
    async start(input) {
      size = input.size
      return { id: 'u1', received: 0, size, chunkBytes: opts.chunkBytes ?? 4 }
    },
    async chunk(id, offset, body): Promise<UploadProgressState> {
      calls.push({ offset, length: body.size })
      const failure = opts.fail?.(calls.length, offset)
      if (failure) throw failure
      if (offset !== have) throw problem(409, 'upload_offset', { received: have })
      have += body.size
      return { id, received: have, size, chunkBytes: opts.chunkBytes ?? 4 }
    },
    async finish() {
      const failure = opts.finish?.()
      if (failure) throw failure
      return { name: 'guide.pdf', size: have }
    },
    async cancel() {
      canceled = true
    },
  }
  return api
}

const file = (bytes: number) => Object.assign(new Blob([new Uint8Array(bytes)]), { name: 'guide.pdf' })
const noWait = async () => {}

describe('Chunked uploads of product files', () => {
  it('sends the file in chunks at the bytes received so far, then finishes', async () => {
    const api = fakeServer({ chunkBytes: 4 })
    const progress: number[] = []
    const out = await uploadFile(api, file(10), { onProgress: (r) => progress.push(r), wait: noWait })
    expect(api.calls).toEqual([
      { offset: 0, length: 4 },
      { offset: 4, length: 4 },
      { offset: 8, length: 2 },
    ])
    expect(progress).toEqual([0, 4, 8, 10])
    expect(out).toEqual({ name: 'guide.pdf', size: 10 })
  })

  it('reports progress within a chunk', async () => {
    const api = fakeServer({ chunkBytes: 8 })
    const chunk = api.chunk.bind(api)
    api.chunk = async (id, offset, body, onSent) => {
      onSent?.(body.size / 2)
      return chunk(id, offset, body)
    }
    const progress: number[] = []
    await uploadFile(api, file(12), { onProgress: (r) => progress.push(r), wait: noWait })
    expect(progress).toEqual([0, 4, 8, 10, 12])
  })

  it('continues from where the server stands after a 409 upload_offset', async () => {
    // The second chunk's answer is lost: the server has 8 bytes, the client thinks 4.
    const api = fakeServer({ chunkBytes: 4 })
    let lost = false
    const chunk = api.chunk.bind(api)
    api.chunk = async (id, offset, body) => {
      const r = await chunk(id, offset, body)
      if (offset === 4 && !lost) {
        lost = true
        throw new TypeError('Failed to fetch')
      }
      return r
    }
    const out = await uploadFile(api, file(10), { wait: noWait })
    // Sent again at 4 (refused: the server has 8), then from 8.
    expect(api.calls.map((c) => c.offset)).toEqual([0, 4, 4, 8])
    expect(out.size).toBe(10)
  })

  it('tries a chunk again after a network error or a 5xx', async () => {
    const waits: number[] = []
    const api = fakeServer({ chunkBytes: 4, fail: (call) => (call === 2 ? new TypeError('Failed to fetch') : call === 3 ? problem(503, 'unavailable') : undefined) })
    const out = await uploadFile(api, file(8), { wait: async (ms) => void waits.push(ms) })
    expect(api.calls.map((c) => c.offset)).toEqual([0, 4, 4, 4])
    expect(waits).toEqual([1000, 2000])
    expect(out.size).toBe(8)
  })

  it('stops after three failed attempts and says how far it got', async () => {
    const api = fakeServer({ chunkBytes: 4, fail: (_, offset) => (offset === 4 ? new TypeError('Failed to fetch') : undefined) })
    const err = await uploadFile(api, file(10), { wait: noWait }).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(UploadStopped)
    expect((err as UploadStopped).received).toBe(4)
    expect((err as UploadStopped).uploadId).toBe('u1')
    expect(api.calls.filter((c) => c.offset === 4)).toHaveLength(3)
  })

  it('doesn’t retry what the server refused', async () => {
    const tooBig = problem(422, 'invalid')
    const api = fakeServer({ chunkBytes: 4, fail: (call) => (call === 1 ? tooBig : undefined) })
    const err = await uploadFile(api, file(10), { wait: noWait }).catch((e: unknown) => e)
    expect(err).toBeInstanceOf(UploadStopped)
    expect((err as UploadStopped).cause).toBe(tooBig)
    expect(api.calls).toHaveLength(1)
  })

  it('passes a type that doesn’t match the name on to the caller', async () => {
    const mismatch = problem(422, 'file_mismatch', { claimed: 'pdf', actual: 'zip' })
    const api = fakeServer({ chunkBytes: 4, finish: () => mismatch })
    await expect(uploadFile(api, file(6), { wait: noWait })).rejects.toBe(mismatch)
  })

  it('sends the rest when finishing finds bytes missing', async () => {
    const api = fakeServer({ chunkBytes: 8 })
    let first = true
    api.finish = async () => {
      if (first) {
        // The server kept only the first half of the only chunk.
        first = false
        api.setHave(4)
        throw problem(409, 'upload_incomplete', { received: 4 })
      }
      return { name: 'guide.pdf', size: 8 }
    }
    const out = await uploadFile(api, file(8), { wait: noWait })
    expect(api.calls).toEqual([
      { offset: 0, length: 8 },
      { offset: 4, length: 4 },
    ])
    expect(out.size).toBe(8)
  })

  it('stops when cancelled', async () => {
    const controller = new AbortController()
    const api = fakeServer({ chunkBytes: 4 })
    const err = await uploadFile(api, file(12), {
      signal: controller.signal,
      wait: noWait,
      onProgress: (r) => {
        if (r === 4) controller.abort()
      },
    }).catch((e: unknown) => e)
    expect((err as Error).name).toBe('AbortError')
    expect(api.calls).toHaveLength(1)
  })
})
