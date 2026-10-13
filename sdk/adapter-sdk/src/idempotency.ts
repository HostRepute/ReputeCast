/** A finished answer: HTTP status and the response envelope (unsigned). */
export interface Answer {
  status: number;
  body: string;
}

interface Entry {
  at: number;
  answer?: Answer;
  pending?: Promise<Answer>;
  size: number;
}

/**
 * Remembers successful answers per idempotency key (15 minutes by default)
 * and runs a key only once at a time: a retried batch gets the answer of the
 * first attempt instead of being sent twice. Errors are not remembered.
 */
export class IdempotencyCache {
  private readonly entries = new Map<string, Entry>();
  private bytes = 0;
  private readonly ttlMs: number;
  private readonly maxBytes: number;
  private readonly now: () => number;

  constructor(opts: { ttlMs?: number; maxBytes?: number; now?: () => number } = {}) {
    this.ttlMs = opts.ttlMs ?? 15 * 60_000;
    this.maxBytes = opts.maxBytes ?? 64 << 20;
    this.now = opts.now ?? Date.now;
  }

  /** Runs fn once per key; concurrent and later callers (within the TTL) share its answer. */
  async run(key: string, fn: () => Promise<Answer>): Promise<Answer> {
    this.evict();
    const hit = this.entries.get(key);
    if (hit?.answer) return hit.answer;
    if (hit?.pending) return hit.pending;
    const pending = fn();
    const entry: Entry = { at: this.now(), pending, size: 0 };
    this.entries.set(key, entry);
    try {
      const answer = await pending;
      if (answer.status === 200) {
        entry.answer = answer;
        entry.size = answer.body.length + key.length;
        entry.at = this.now();
        delete entry.pending;
        this.bytes += entry.size;
        this.evict();
      } else {
        this.entries.delete(key);
      }
      return answer;
    } catch (e) {
      this.entries.delete(key);
      throw e;
    }
  }

  get size(): number {
    return this.entries.size;
  }

  private evict(): void {
    const oldest = this.now() - this.ttlMs;
    for (const [k, e] of this.entries) {
      // Map iteration follows insertion order: stop at the first live entry
      // once under the byte budget.
      if (e.pending) continue;
      if (e.at >= oldest && this.bytes <= this.maxBytes) break;
      this.entries.delete(k);
      this.bytes -= e.size;
    }
  }
}
