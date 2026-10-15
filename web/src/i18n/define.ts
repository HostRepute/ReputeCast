/** A message: its ID and its English source text, in ICU MessageFormat. */
export interface MessageDescriptor {
  readonly id: string
  readonly message: string
}

/** Translated ICU text by message ID. */
export type Catalog = Readonly<Record<string, string>>

/** The descriptors `defineMessages` returns, one per key. */
export type Messages<K extends string> = { readonly [P in K]: MessageDescriptor }

const sources = new Map<string, string>()

/** Every English source message defined so far, by ID. */
export function sourceMessages(): ReadonlyMap<string, string> {
  return sources
}

/**
 * Declares a namespace's English messages. Each key becomes the message ID
 * `namespace.key`:
 *
 *   export const m = defineMessages('transactional', {
 *     title: 'Transactional',
 *     sent: '{count, plural, one {# message} other {# messages}} sent',
 *   })
 */
export function defineMessages<K extends string>(namespace: string, messages: Readonly<Record<K, string>>): Messages<K> {
  const out = {} as Record<K, MessageDescriptor>
  for (const key of Object.keys(messages) as K[]) {
    const id = `${namespace}.${key}`
    const message = messages[key]
    const known = sources.get(id)
    if (import.meta.env.DEV && known !== undefined && known !== message) {
      console.warn(`i18n: ${id} is defined twice with different text`)
    }
    sources.set(id, message)
    out[key] = Object.freeze({ id, message })
  }
  return out
}

/**
 * A translation of one namespace, checked against its source: a key that
 * isn't in the source fails to type-check, a missing key falls back to
 * English. Default-export it from the `*.vi.ts` file next to the source:
 *
 *   export default translate(m, { title: 'Giao dịch' })
 */
export function translate<M extends Readonly<Record<string, MessageDescriptor>>>(source: M, messages: { readonly [K in keyof M]?: string }): Catalog {
  const out: Record<string, string> = {}
  for (const key of Object.keys(messages)) {
    const text = messages[key]
    const d = source[key]
    if (text !== undefined && d) out[d.id] = text
  }
  return out
}
