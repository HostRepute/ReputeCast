import { ApiError, type Schemas } from '@/api/client'

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

/** The first message per field of a problem, by location ("body.headers"). */
export function fieldErrors(e: unknown): Record<string, string> {
  const out: Record<string, string> = {}
  if (e instanceof ApiError) {
    for (const d of e.problem.errors ?? []) {
      if (d.location && d.message && !out[d.location]) out[d.location] = d.message
    }
  }
  return out
}

/** A previewed request as it goes on the wire. */
export function previewText(p: Schemas['AdapterHTTPPreview']): string {
  const head = [`${p.method} ${p.url}`, ...p.headers.map((h) => `${h.name}: ${h.value}`)].join('\n')
  return p.body ? `${head}\n\n${p.body}` : head
}
