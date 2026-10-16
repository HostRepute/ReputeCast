import { createElement, Fragment, type ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import type { CodeLang } from './shared'

/**
 * The code samples of the Templates tab, Setup and the first-send guide: a
 * call that sends a template from an app, its data filled from the app's own
 * values (user.first_name, reset_link) or the variable's sample. They are
 * code, so they stay in English like the API.
 */
type Variable = Schemas['TransactionalVariable']

export interface CodeInput {
  /** The send endpoint: https://…/api/v1/workspaces/acme-newsletter/transactional/messages */
  endpoint: string
  template: string
  variables: Variable[]
  /** Node.js reads the base URL from RC (the Templates tab's short sample). */
  compact?: boolean
  /**
   * cURL sends literal values: these (a test data set), else the variables'
   * samples; with tags and metadata when the guide knows how the app calls it.
   */
  literal?: { data?: Record<string, unknown>; tags?: string[]; metadata?: Record<string, string> }
}

type Style = 'camel' | 'snake' | 'pascal' | 'php'

const cap = (w: string) => w.charAt(0).toUpperCase() + w.slice(1)

function words(name: string): string[] {
  return name.split(/[_\-.]+/).filter(Boolean)
}

function ident(parts: string[], style: Style): string {
  switch (style) {
    case 'camel':
      return parts.map((w, i) => (i === 0 ? w : cap(w))).join('')
    case 'pascal':
      return parts.map((w) => (w === 'id' ? 'ID' : cap(w))).join('')
    case 'php':
      return '$' + parts.map((w, i) => (i === 0 ? w : cap(w))).join('')
  }
  return parts.join('_')
}

const userFields = new Set(['first_name', 'last_name', 'name', 'email', 'company', 'company_name'])

/** What an app passes for a variable: a field of its user, a link it made, or the sample. */
function valueOf(v: Variable, style: Style): { expr?: string; literal?: string | number } {
  const name = v.name
  if (userFields.has(name)) {
    const field = style === 'camel' ? ident(words(name), 'camel') : style === 'pascal' ? ident(words(name), 'pascal') : name
    return { expr: style === 'php' ? `$user->${name}` : `user.${field}` }
  }
  if (name.endsWith('_url')) {
    const base = words(name.slice(0, -4)).concat('link')
    return { expr: ident(base, style) }
  }
  if (v.type === 'number' && v.sample !== '' && !Number.isNaN(Number(v.sample))) return { literal: Number(v.sample) }
  if (v.sample) return { literal: v.sample }
  return { expr: ident(words(name), style) }
}

function idempotency(template: string): string {
  return template === 'password-reset' ? 'pwreset' : template.replace(/-/g, '')
}

function quote(s: string, q: '"' | "'" = '"'): string {
  return q === '"' ? JSON.stringify(s) : `'${s.replace(/\\/g, '\\\\').replace(/'/g, "\\'")}'`
}

function lit(v: string | number, q: '"' | "'" = '"'): string {
  return typeof v === 'number' ? String(v) : quote(v, q)
}

export function codeSample(lang: CodeLang, s: CodeInput): string {
  const vars = s.variables
  const key = idempotency(s.template)
  switch (lang) {
    case 'node': {
      const data = vars.map((v) => {
        const x = valueOf(v, 'camel')
        return `      ${v.name}: ${x.expr ?? lit(x.literal!)},`
      })
      const url = s.compact ? '`${RC}/transactional/messages`' : quote(s.endpoint)
      return [
        `const res = await fetch(${url}, {`,
        `  method: "POST",`,
        `  headers: {`,
        '    Authorization: `Bearer ${process.env.REPUTECAST_KEY}`,',
        `    "Idempotency-Key": \`${key}-\${user.id}-\${event.id}\`,`,
        `    "Content-Type": "application/json",`,
        `  },`,
        `  body: JSON.stringify({`,
        `    template: ${quote(s.template)},`,
        `    to: { email: user.email, name: user.name },`,
        `    data: {`,
        ...data,
        `    },`,
        `  }),`,
        `});`,
        `// 202 Accepted → { id: "0192f1c4-…-2c44", status: "queued" }`,
      ].join('\n')
    }
    case 'python': {
      const data = vars.map((v) => {
        const x = valueOf(v, 'snake')
        return `            ${quote(v.name)}: ${x.expr ?? lit(x.literal!)},`
      })
      return [
        `import os, requests`,
        ``,
        `r = requests.post(`,
        `    ${quote(s.endpoint)},`,
        `    headers={`,
        `        "Authorization": f"Bearer {os.environ['REPUTECAST_KEY']}",`,
        `        "Idempotency-Key": f"${key}-{user.id}-{event.id}",`,
        `    },`,
        `    json={`,
        `        "template": ${quote(s.template)},`,
        `        "to": {"email": user.email, "name": user.name},`,
        `        "data": {`,
        ...data,
        `        },`,
        `    },`,
        `    timeout=10,`,
        `)`,
        `r.raise_for_status()`,
        `# 202 → {"id": "0192f1c4-…", "status": "queued"}`,
      ].join('\n')
    }
    case 'go': {
      const width = Math.max(0, ...vars.map((v) => v.name.length + 2))
      const data = vars.map((v) => {
        const x = valueOf(v, 'pascal')
        const k = `${quote(v.name)}:`.padEnd(width + 1)
        return `\t\t${k} ${x.expr ?? lit(x.literal!)},`
      })
      return [
        `body, _ := json.Marshal(map[string]any{`,
        `\t"template": ${quote(s.template)},`,
        `\t"to":       map[string]string{"email": user.Email, "name": user.Name},`,
        `\t"data": map[string]any{`,
        ...data,
        `\t},`,
        `})`,
        `req, _ := http.NewRequest("POST", ${quote(s.endpoint)}, bytes.NewReader(body))`,
        `req.Header.Set("Authorization", "Bearer "+os.Getenv("REPUTECAST_KEY"))`,
        `req.Header.Set("Idempotency-Key", "${key}-"+user.ID+"-"+event.ID)`,
        `req.Header.Set("Content-Type", "application/json")`,
        `res, err := http.DefaultClient.Do(req)`,
        `// 202 Accepted → {"id": "0192f1c4-…", "status": "queued"}`,
      ].join('\n')
    }
    case 'php': {
      const data = vars.map((v) => {
        const x = valueOf(v, 'php')
        return `            ${quote(v.name, "'")} => ${x.expr ?? lit(x.literal!, "'")},`
      })
      return [
        `$res = (new GuzzleHttp\\Client())->post(${quote(s.endpoint, "'")}, [`,
        `    'headers' => [`,
        `        'Authorization' => 'Bearer ' . getenv('REPUTECAST_KEY'),`,
        `        'Idempotency-Key' => "${key}-{$user->id}-{$event->id}",`,
        `    ],`,
        `    'json' => [`,
        `        'template' => ${quote(s.template, "'")},`,
        `        'to' => ['email' => $user->email, 'name' => $user->name],`,
        `        'data' => [`,
        ...data,
        `        ],`,
        `    ],`,
        `]);`,
        `// 202 Accepted → {"id": "0192f1c4-…", "status": "queued"}`,
      ].join('\n')
    }
  }
  // cURL: a call from a terminal, with literal values (TransactionalGuide).
  const entries = vars.map((v): [string, unknown] => {
    const given = s.literal?.data?.[v.name]
    const raw = given !== undefined ? given : v.type === 'number' && v.sample !== '' && !Number.isNaN(Number(v.sample)) ? Number(v.sample) : v.sample || '…'
    return [v.name, v.sensitive ? masked(raw) : raw]
  })
  const tags = s.literal?.tags?.length ? s.literal.tags : undefined
  const meta = s.literal?.metadata && Object.keys(s.literal.metadata).length ? s.literal.metadata : undefined
  const tail = [
    tags ? `    "tags": [${tags.map((x) => quote(x)).join(', ')}]` : '',
    meta
      ? `    "metadata": { ${Object.entries(meta)
          .map(([k, v]) => `${quote(k)}: ${quote(v)}`)
          .join(', ')} }`
      : '',
  ].filter(Boolean)
  const body = [
    `{`,
    `    "template": ${quote(s.template)},`,
    `    "to": { "email": "sam.rivera@outlook.com", "name": "Sam Rivera" },`,
    entries.length ? `    "data": {` : `    "data": {}${tail.length ? ',' : ''}`,
    ...(entries.length ? entries.map(([k, v], i) => `      ${quote(k)}: ${JSON.stringify(v)}${i < entries.length - 1 ? ',' : ''}`) : []),
    ...(entries.length ? [`    }${tail.length ? ',' : ''}`] : []),
    ...tail.map((l, i) => (i < tail.length - 1 ? `${l},` : l)),
    `  }`,
  ].join('\n')
  return [
    `$ curl ${s.endpoint} \\`,
    `  -H "Authorization: Bearer $REPUTECAST_KEY" \\`,
    `  -H "Idempotency-Key: ${key}-7f3a91" \\`,
    `  -H "Content-Type: application/json" \\`,
    `  -d '${body.replace(/'/g, "'\\''")}'`,
  ].join('\n')
}

/** A sensitive value in a sample: a link keeps its path ("https://app.acme.io/reset/…"), anything else is "…". */
function masked(v: unknown): string {
  if (typeof v === 'string' && /^https?:\/\/[^/]+\//.test(v)) return v.replace(/[^/]*$/, '…')
  return '…'
}

const keywords: Record<CodeLang, string[]> = {
  node: ['const', 'await', 'async', 'let', 'return', 'new'],
  python: ['import', 'from', 'return', 'def', 'True', 'False', 'None'],
  go: ['func', 'return', 'var', 'if', 'nil', 'err'],
  php: ['new', 'return'],
  curl: [],
}

/** Colours a sample the way the boards do: strings green, keywords pink, numbers yellow, comments grey. */
export function highlight(code: string, lang: CodeLang): ReactNode {
  if (lang === 'curl') return highlightCurl(code)
  const kw = keywords[lang].join('|')
  const re = new RegExp(`(\`[^\`]*\`|"(?:[^"\\\\]|\\\\.)*"|'(?:[^'\\\\]|\\\\.)*'|//[^\\n]*|#[^\\n]*|\\b(?:${kw})\\b|\\b\\d+(?:\\.\\d+)?\\b)`, 'g')
  const out: ReactNode[] = []
  let last = 0
  let i = 0
  for (const m of code.matchAll(re)) {
    const tok = m[0]
    const at = m.index ?? 0
    if (at > last) out.push(code.slice(last, at))
    let cls = 'c2'
    if (tok.startsWith('//') || (tok.startsWith('#') && lang === 'python')) cls = 'cm'
    else if (tok.startsWith('#')) {
      out.push(tok)
      last = at + tok.length
      continue
    } else if (/^\d/.test(tok)) cls = 'c3'
    else if (!/^["'`]/.test(tok)) cls = 'c5'
    out.push(createElement('span', { key: i++, className: cls }, tok))
    last = at + tok.length
  }
  if (last < code.length) out.push(code.slice(last))
  return createElement(Fragment, null, ...out)
}

/**
 * cURL as the guide's boards colour it: the prompt grey, header strings and
 * the quoted body green, the body's keys blue, its numbers yellow.
 */
function highlightCurl(code: string): ReactNode {
  const re = /(^\$(?= )|'\{|\}'|"(?:[^"\\]|\\.)*"|(?<=: )-?\d+(?:\.\d+)?\b)/gm
  const out: ReactNode[] = []
  let last = 0
  let i = 0
  for (const m of code.matchAll(re)) {
    const tok = m[0]
    const at = m.index ?? 0
    if (at > last) out.push(code.slice(last, at))
    let cls = 'c2'
    if (tok === '$') cls = 'c4'
    else if (/^-?\d/.test(tok)) cls = 'c3'
    else if (tok.startsWith('"') && /^\s*:/.test(code.slice(at + tok.length))) cls = 'c1'
    out.push(createElement('span', { key: i++, className: cls }, tok))
    last = at + tok.length
  }
  if (last < code.length) out.push(code.slice(last))
  return createElement(Fragment, null, ...out)
}
