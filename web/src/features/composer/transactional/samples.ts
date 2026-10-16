import type { Data } from './types'

/**
 * The API tab's code samples: the exact request that sends this template,
 * in five languages, with its declared variables filled in from the
 * preview's data. They are code, so they stay in English like the API.
 */
export type Lang = 'curl' | 'node' | 'python' | 'go' | 'php'

export const LANGS: { key: Lang; label: string }[] = [
  { key: 'curl', label: 'cURL' },
  { key: 'node', label: 'Node.js' },
  { key: 'python', label: 'Python' },
  { key: 'go', label: 'Go' },
  { key: 'php', label: 'PHP' },
]

export interface SampleInput {
  /** The server's origin, such as https://reputecast.acme.io. */
  origin: string
  workspace: string
  /** The template's ID (password-reset). */
  template: string
  to: { email: string; name: string }
  data: Data
  /** The language version to send (internal/languages); omitted for the source. */
  language?: string
}

export function messagesUrl(origin: string, workspace: string): string {
  return `${origin.replace(/\/+$/, '')}/api/v1/workspaces/${encodeURIComponent(workspace)}/transactional/messages`
}

/** An Idempotency-Key that reads as an example. */
function idempotencyKey(template: string): string {
  return `${template.slice(0, 40)}-7f3a91`
}

function payload(s: SampleInput): Data {
  const body: Data = { template: s.template, to: { email: s.to.email, name: s.to.name }, data: s.data }
  if (s.language) body.language = s.language
  return body
}

function indent(text: string, by: string): string {
  return text
    .split('\n')
    .map((l, i) => (i === 0 ? l : by + l))
    .join('\n')
}

const isMap = (v: unknown): v is Record<string, unknown> => !!v && typeof v === 'object' && !Array.isArray(v)

/** A Python literal: dicts, lists, True/False/None. */
function py(v: unknown, pad = ''): string {
  const next = `${pad}    `
  if (v === null || v === undefined) return 'None'
  if (typeof v === 'boolean') return v ? 'True' : 'False'
  if (typeof v === 'number') return String(v)
  if (typeof v === 'string') return JSON.stringify(v)
  if (Array.isArray(v)) return v.length ? `[\n${v.map((x) => next + py(x, next)).join(',\n')},\n${pad}]` : '[]'
  if (isMap(v)) {
    const keys = Object.keys(v)
    return keys.length ? `{\n${keys.map((k) => `${next}${JSON.stringify(k)}: ${py(v[k], next)}`).join(',\n')},\n${pad}}` : '{}'
  }
  return JSON.stringify(String(v))
}

/** A Go literal: map[string]any, []any. */
function go(v: unknown, pad = ''): string {
  const next = `${pad}\t`
  if (v === null || v === undefined) return 'nil'
  if (typeof v === 'boolean' || typeof v === 'number') return String(v)
  if (typeof v === 'string') return JSON.stringify(v)
  if (Array.isArray(v)) return v.length ? `[]any{\n${v.map((x) => `${next}${go(x, next)},`).join('\n')}\n${pad}}` : '[]any{}'
  if (isMap(v)) {
    const keys = Object.keys(v)
    return keys.length ? `map[string]any{\n${keys.map((k) => `${next}${JSON.stringify(k)}: ${go(v[k], next)},`).join('\n')}\n${pad}}` : 'map[string]any{}'
  }
  return JSON.stringify(String(v))
}

function phpString(s: string): string {
  return `'${s.replace(/\\/g, '\\\\').replace(/'/g, "\\'")}'`
}

/** A PHP literal: short arrays, and (object) [] for an empty map so it encodes as {}. */
function php(v: unknown, pad = ''): string {
  const next = `${pad}    `
  if (v === null || v === undefined) return 'null'
  if (typeof v === 'boolean') return v ? 'true' : 'false'
  if (typeof v === 'number') return String(v)
  if (typeof v === 'string') return phpString(v)
  if (Array.isArray(v)) return v.length ? `[\n${v.map((x) => next + php(x, next)).join(',\n')},\n${pad}]` : '[]'
  if (isMap(v)) {
    const keys = Object.keys(v)
    return keys.length ? `[\n${keys.map((k) => `${next}${phpString(k)} => ${php(v[k], next)}`).join(',\n')},\n${pad}]` : '(object) []'
  }
  return phpString(String(v))
}

export function apiSample(lang: Lang, s: SampleInput): string {
  const url = messagesUrl(s.origin, s.workspace)
  const key = idempotencyKey(s.template)
  const body = payload(s)
  switch (lang) {
    case 'curl': {
      const json = indent(JSON.stringify(body, null, 2), '  ').replace(/'/g, "'\\''")
      return [
        `curl ${url} \\`,
        `  -H "Authorization: Bearer $REPUTECAST_KEY" \\`,
        `  -H "Idempotency-Key: ${key}" \\`,
        `  -H "Content-Type: application/json" \\`,
        `  -d '${json}'`,
      ].join('\n')
    }
    case 'node':
      return [
        `const res = await fetch(${JSON.stringify(url)}, {`,
        `  method: "POST",`,
        `  headers: {`,
        '    Authorization: `Bearer ${process.env.REPUTECAST_KEY}`,',
        `    "Idempotency-Key": ${JSON.stringify(key)},`,
        `    "Content-Type": "application/json",`,
        `  },`,
        `  body: JSON.stringify(${indent(JSON.stringify(body, null, 2), '  ')}),`,
        `});`,
        `const message = await res.json();`,
      ].join('\n')
    case 'python':
      return [
        `import os`,
        `import requests`,
        ``,
        `res = requests.post(`,
        `    ${JSON.stringify(url)},`,
        `    headers={`,
        `        "Authorization": f"Bearer {os.environ['REPUTECAST_KEY']}",`,
        `        "Idempotency-Key": ${JSON.stringify(key)},`,
        `    },`,
        `    json=${py(body, '    ')},`,
        `)`,
        `res.raise_for_status()`,
        `message = res.json()`,
      ].join('\n')
    case 'go':
      return [
        `body, _ := json.Marshal(${go(body)})`,
        `req, _ := http.NewRequest("POST", ${JSON.stringify(url)}, bytes.NewReader(body))`,
        `req.Header.Set("Authorization", "Bearer "+os.Getenv("REPUTECAST_KEY"))`,
        `req.Header.Set("Idempotency-Key", ${JSON.stringify(key)})`,
        `req.Header.Set("Content-Type", "application/json")`,
        `res, err := http.DefaultClient.Do(req)`,
      ].join('\n')
    case 'php':
      return [
        `$ch = curl_init(${phpString(url)});`,
        `curl_setopt_array($ch, [`,
        `    CURLOPT_POST => true,`,
        `    CURLOPT_RETURNTRANSFER => true,`,
        `    CURLOPT_HTTPHEADER => [`,
        `        'Authorization: Bearer ' . getenv('REPUTECAST_KEY'),`,
        `        ${phpString(`Idempotency-Key: ${key}`)},`,
        `        'Content-Type: application/json',`,
        `    ],`,
        `    CURLOPT_POSTFIELDS => json_encode(${php(body, '    ')}),`,
        `]);`,
        `$message = json_decode(curl_exec($ch), true);`,
      ].join('\n')
  }
}

/** The answer to a call that worked. */
export function acceptedAnswer(): string {
  return JSON.stringify({ id: '0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44', status: 'queued' }, null, 2)
}

function joinAnd(items: string[]): string {
  return items.length < 2 ? (items[0] ?? '') : `${items.slice(0, -1).join(', ')} and ${items[items.length - 1]}`
}

/** The problem a call without the required variables gets (the API's own wording). */
export function missingAnswer(template: string, version: number, missing: string[]): string {
  return JSON.stringify(
    {
      title: 'Missing variables',
      status: 422,
      code: 'missing_variables',
      detail: `${template} v${version} needs ${joinAnd(missing)}. Add ${missing.length === 1 ? 'it' : 'them'} to data and call again.`,
      missing,
    },
    null,
    2,
  )
}

/** The problem a call to a template that was never published gets. */
export function unpublishedAnswer(template: string): string {
  return JSON.stringify(
    {
      title: 'Unprocessable Entity',
      status: 422,
      code: 'template_not_published',
      detail: `${template} has a draft but no published version, so there is nothing to send. Publish it first, then call again with the same idempotency key.`,
    },
    null,
    2,
  )
}
