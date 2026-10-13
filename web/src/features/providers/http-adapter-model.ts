import { ApiError, errorMessage, type Schemas } from '@/api/client'

export type WebhookAuth = NonNullable<Schemas['AdapterHTTPMapping']['webhookAuth']>
type Header = Schemas['AdapterHTTPHeader']

/** How the provider's webhooks are authenticated (docs/adapters.md › The generic HTTP adapter). */
export const webhookAuths: { value: WebhookAuth; label: string; header?: string; placeholder?: string }[] = [
  { value: 'hmac', label: 'HMAC signature in a header', header: 'Signature header', placeholder: 'X-Signature' },
  { value: 'standard', label: 'Standard Webhooks (webhook-signature)' },
  { value: 'basic', label: 'Basic auth' },
  { value: 'header', label: 'Secret in a header', header: 'Header with the secret', placeholder: 'X-Webhook-Token' },
  { value: 'none', label: 'None: the webhook URL only' },
]

export type WebhookPaths = Required<Schemas['AdapterHTTPWebhookPaths']>

/** The paths a custom JSONPath map reads, inside one event unless noted. */
export const webhookPathFields: { key: keyof WebhookPaths; label: string; placeholder: string; optional?: boolean }[] = [
  { key: 'type', label: 'Event type', placeholder: '$.event' },
  { key: 'recipient', label: 'Recipient', placeholder: '$.email' },
  { key: 'messageId', label: 'Message id', placeholder: '$.message_id' },
  { key: 'events', label: 'Events in one request', placeholder: '$.events[*]', optional: true },
]

/** "Name: value" lines into request headers; blank lines are skipped. */
export function parseHeaderLines(text: string): { headers: Header[]; error?: string } {
  const headers: Header[] = []
  const lines = text.split('\n')
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]!.trim()
    if (!line) continue
    const at = line.indexOf(':')
    if (at <= 0) return { headers, error: `Line ${i + 1}: write each header as Name: value.` }
    headers.push({ name: line.slice(0, at).trim(), value: line.slice(at + 1).trim() })
  }
  return { headers }
}

/**
 * The first message per field of a problem, by location ("body.headers"); a
 * location inside a list counts for the list ("body.headers[2].name").
 */
export function fieldErrors(e: unknown): Record<string, string> {
  const out: Record<string, string> = {}
  if (e instanceof ApiError) {
    for (const d of e.problem.errors ?? []) {
      const loc = d.location?.replace(/\[\d+\].*$/, '')
      if (loc && d.message && !out[loc]) out[loc] = d.message
    }
  }
  return out
}

/**
 * Splits a problem between the fields a form shows (by location) and one
 * message for the rest, so no problem goes unseen: the problem's detail when
 * it names no field, else the messages of the fields the form doesn't show.
 */
export function formProblems(e: unknown, shown: ReadonlySet<string>): { fields: Record<string, string>; general: string } {
  const fields = fieldErrors(e)
  const located = Object.entries(fields)
  if (located.length === 0) return { fields, general: errorMessage(e) }
  return {
    fields,
    general: located
      .filter(([k]) => !shown.has(k))
      .map(([, m]) => m)
      .join(' '),
  }
}

/** A previewed request as it goes on the wire. */
export function previewText(p: Schemas['AdapterHTTPPreview']): string {
  const head = [`${p.method} ${p.url}`, ...p.headers.map((h) => `${h.name}: ${h.value}`)].join('\n')
  return p.body ? `${head}\n\n${p.body}` : head
}
