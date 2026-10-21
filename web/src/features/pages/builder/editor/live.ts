/**
 * Live blocks (products, forms, countdowns, events…) are drawn by the
 * server, as readers see them (render-page-blocks). The canvas asks for
 * them in batches and keeps what came back by the block's HTML, so undo,
 * redo and re-renders draw at once; a changed attribute is a new key.
 */

/** Draws blocks on the server: one HTML string per block, in order ('' when it shows nothing). */
export type RenderBlocks = (blocks: string[]) => Promise<string[]>

const MAX_BATCH = 100

export class LiveRenderer {
  private readonly render: RenderBlocks
  /** What came back, by key: the HTML, or null when it couldn't be drawn. */
  private readonly cache = new Map<string, string | null>()
  private readonly waiting = new Map<string, ((html: string | null) => void)[]>()
  private queue: string[] = []
  private timer: ReturnType<typeof setTimeout> | undefined
  private disposed = false

  constructor(render: RenderBlocks) {
    this.render = render
  }

  /** The block's drawing when it's known: HTML, null for a failure, undefined while unknown. */
  peek(key: string): string | null | undefined {
    return this.cache.get(key)
  }

  /** Draws a block, in the next batch. */
  request(key: string): Promise<string | null> {
    const known = this.cache.get(key)
    if (known !== undefined) return Promise.resolve(known)
    return new Promise((resolve) => {
      const list = this.waiting.get(key)
      if (list) {
        list.push(resolve)
        return
      }
      this.waiting.set(key, [resolve])
      this.queue.push(key)
      if (this.timer === undefined) this.timer = setTimeout(() => this.flush(), 40)
    })
  }

  private flush() {
    this.timer = undefined
    const batch = this.queue.splice(0, MAX_BATCH)
    if (this.queue.length) this.timer = setTimeout(() => this.flush(), 0)
    if (!batch.length) return
    this.render(batch).then(
      (out) => batch.forEach((key, i) => this.settle(key, out[i] ?? '')),
      () => batch.forEach((key) => this.settle(key, null, false)),
    )
  }

  private settle(key: string, html: string | null, keep = true) {
    if (this.disposed) return
    // A failure isn't kept: the next paint asks again.
    if (keep) this.cache.set(key, html)
    const list = this.waiting.get(key) ?? []
    this.waiting.delete(key)
    for (const fn of list) fn(html)
  }

  /** Forgets every drawing, e.g. after a form was resumed or a product changed. */
  clear() {
    this.cache.clear()
  }

  dispose() {
    this.disposed = true
    if (this.timer !== undefined) clearTimeout(this.timer)
    this.waiting.clear()
    this.queue = []
  }
}
