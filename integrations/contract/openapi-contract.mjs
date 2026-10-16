// The contract between the apps in integrations/ and ReputeCast's public API.
//
// The apps' tests capture every request an app makes (and the answers they
// mock) and check them against the OpenAPI document the server generates
// (web/openapi.json): the operation exists, the path parameters fit, and
// the body validates against the operation's schema. Request bodies are
// strict (additionalProperties: false), so a field the server doesn't
// know fails here, not in a user's workflow.
//
// No dependencies: it validates the JSON Schema subset the document uses
// (type, properties, required, additionalProperties, items, enum,
// min/maxLength, min/maxItems, minimum, maximum, pattern, format, $ref).

import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, join } from 'node:path'

const here = dirname(fileURLToPath(import.meta.url))

/** The OpenAPI document the server generates: `go run ./cmd/reputecast openapi`. */
export const openapiPath = join(here, '..', '..', 'web', 'openapi.json')

let cached = null

/** Loads web/openapi.json once. */
export function loadSpec(path = openapiPath) {
  if (!cached || cached.path !== path) {
    cached = { path, spec: JSON.parse(readFileSync(path, 'utf8')) }
  }
  return cached.spec
}

const methods = ['get', 'put', 'post', 'delete', 'patch']

/** Every operation of the document, by operationId. */
export function operations(spec = loadSpec()) {
  const out = new Map()
  for (const [path, item] of Object.entries(spec.paths)) {
    for (const m of methods) {
      const op = item[m]
      if (op?.operationId) out.set(op.operationId, { method: m.toUpperCase(), path, op })
    }
  }
  return out
}

function templateRegex(template) {
  const names = []
  const source = template.replace(/[.*+?^$()|[\]\\]/g, '\\$&').replace(/\\?\{([^}]+)\\?\}/g, (_, name) => {
    names.push(name.replace(/\\/g, ''))
    return '([^/]+)'
  })
  return { re: new RegExp(`^${source}$`), names }
}

/**
 * Finds the operation a request calls: method and URL (absolute, or a path
 * starting with /api/v1). Returns { operationId, method, path, op, params }
 * or null.
 */
export function findOperation(method, url, spec = loadSpec()) {
  const pathname = new URL(url, 'http://reputecast.invalid').pathname
  const m = method.toUpperCase()
  // Literal segments beat parameters: /integration/hooks/{hookId} vs /…/{id}.
  let best = null
  for (const [operationId, o] of operations(spec)) {
    if (o.method !== m) continue
    const { re, names } = templateRegex(o.path)
    const hit = re.exec(pathname)
    if (!hit) continue
    const params = Object.fromEntries(names.map((n, i) => [n, decodeURIComponent(hit[i + 1])]))
    if (!best || names.length < Object.keys(best.params).length) {
      best = { operationId, ...o, params }
    }
  }
  return best
}

function resolve(schema, spec) {
  let s = schema
  for (let i = 0; s && s.$ref && i < 20; i++) {
    const name = s.$ref.replace('#/components/schemas/', '')
    s = spec.components.schemas[name]
  }
  return s
}

const uuidRe = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

function typeOf(v) {
  if (v === null) return 'null'
  if (Array.isArray(v)) return 'array'
  if (typeof v === 'number') return Number.isInteger(v) ? 'integer' : 'number'
  return typeof v
}

function typeMatches(want, v) {
  const got = typeOf(v)
  if (want === 'number') return got === 'number' || got === 'integer'
  return got === want
}

/** Validates a value against a schema; returns a list of problems ("body.email: …"). */
export function validate(schema, value, at = 'body', spec = loadSpec()) {
  const s = resolve(schema, spec)
  const errors = []
  if (!s || Object.keys(s).length === 0) return errors
  const types = Array.isArray(s.type) ? s.type : s.type ? [s.type] : []
  if (types.length && !types.some((t) => typeMatches(t, value))) {
    return [`${at}: expected ${types.join(' or ')}, got ${typeOf(value)}`]
  }
  if (s.enum && !s.enum.includes(value)) errors.push(`${at}: ${JSON.stringify(value)} is not one of ${JSON.stringify(s.enum)}`)
  if (typeof value === 'string') {
    const n = [...value].length
    if (s.minLength != null && n < s.minLength) errors.push(`${at}: shorter than ${s.minLength}`)
    if (s.maxLength != null && n > s.maxLength) errors.push(`${at}: longer than ${s.maxLength}`)
    if (s.pattern && !new RegExp(s.pattern, 'u').test(value)) errors.push(`${at}: doesn't match ${s.pattern}`)
    if (s.format === 'uuid' && !uuidRe.test(value)) errors.push(`${at}: not a uuid`)
    if (s.format === 'date-time' && Number.isNaN(Date.parse(value))) errors.push(`${at}: not a date-time`)
    if (s.format === 'email' && !/^[^@\s]+@[^@\s]+$/.test(value)) errors.push(`${at}: not an email address`)
  }
  if (typeof value === 'number') {
    if (s.minimum != null && value < s.minimum) errors.push(`${at}: below ${s.minimum}`)
    if (s.maximum != null && value > s.maximum) errors.push(`${at}: above ${s.maximum}`)
  }
  if (Array.isArray(value)) {
    if (s.minItems != null && value.length < s.minItems) errors.push(`${at}: fewer than ${s.minItems} items`)
    if (s.maxItems != null && value.length > s.maxItems) errors.push(`${at}: more than ${s.maxItems} items`)
    if (s.items) value.forEach((v, i) => errors.push(...validate(s.items, v, `${at}[${i}]`, spec)))
  }
  if (typeOf(value) === 'object') {
    for (const r of s.required ?? []) {
      if (value[r] === undefined) errors.push(`${at}.${r}: required`)
    }
    const props = s.properties ?? {}
    for (const [k, v] of Object.entries(value)) {
      // As JSON sends it: a property set to undefined isn't there.
      if (v === undefined) continue
      if (k in props) {
        errors.push(...validate(props[k], v, `${at}.${k}`, spec))
      } else if (s.additionalProperties === false) {
        errors.push(`${at}.${k}: not a property the server accepts`)
      } else if (s.additionalProperties && typeof s.additionalProperties === 'object') {
        errors.push(...validate(s.additionalProperties, v, `${at}.${k}`, spec))
      }
    }
  }
  return errors
}

/**
 * Checks one request against the document: the operation, its path
 * parameters and its JSON body. `pending` lists body fields another
 * workstream adds to the API in this release ({ operationId: [field] });
 * they're accepted, and reported in `pending` on the result.
 */
export function checkRequest({ method, url, body, headers = {} }, { pending = {}, spec = loadSpec() } = {}) {
  const found = findOperation(method, url, spec)
  if (!found) return { ok: false, errors: [`${method} ${new URL(url, 'http://x').pathname}: no such operation`] }
  const errors = []
  const op = found.op
  for (const p of op.parameters ?? []) {
    if (p.in === 'path') {
      const v = found.params[p.name]
      if (v === undefined || v === '') errors.push(`path.${p.name}: missing`)
      else errors.push(...validate(p.schema ?? {}, v, `path.${p.name}`, spec))
    }
    if (p.in === 'header' && p.required) {
      const has = Object.keys(headers).some((h) => h.toLowerCase() === p.name.toLowerCase())
      if (!has) errors.push(`header.${p.name}: required`)
    }
  }
  const content = op.requestBody?.content?.['application/json']
  let payload = body
  if (typeof payload === 'string' && payload !== '') payload = JSON.parse(payload)
  const used = []
  if (content) {
    if (payload === undefined || payload === null || payload === '') {
      if (op.requestBody.required) errors.push('body: required')
    } else {
      // Fields the document doesn't have yet, but another workstream adds
      // in this release, are set aside and reported.
      const later = new Set(pending[found.operationId] ?? [])
      let strict = payload
      if (typeOf(payload) === 'object') {
        strict = {}
        for (const [k, v] of Object.entries(payload)) {
          const known = k in (resolve(content.schema, spec)?.properties ?? {})
          if (later.has(k) && !known) used.push(k)
          else strict[k] = v
        }
      }
      errors.push(...validate(content.schema, strict, 'body', spec))
    }
  } else if (payload !== undefined && payload !== null && payload !== '') {
    errors.push('body: the operation takes no body')
  }
  return { ok: errors.length === 0, errors, operationId: found.operationId, pending: used }
}

/** Checks a mocked answer against the operation's documented answer. */
export function checkResponse(operationId, status, body, { spec = loadSpec() } = {}) {
  const o = operations(spec).get(operationId)
  if (!o) return { ok: false, errors: [`${operationId}: no such operation`] }
  const r = o.op.responses[String(status)] ?? (status >= 400 ? o.op.responses.default : undefined)
  if (!r) return { ok: false, errors: [`${operationId}: no ${status} answer documented`] }
  const content = r.content?.['application/json'] ?? r.content?.['application/problem+json']
  if (!content) return { ok: body === undefined || body === '' || body === null, errors: [] }
  let schema = content.schema
  if (status >= 400) {
    // Domain packages extend problems with their own fields (the send
    // API's `missing`); the shared ones must still fit.
    schema = { ...resolve(schema, spec), additionalProperties: undefined }
  }
  const errors = validate(schema, body, 'response', spec)
  return { ok: errors.length === 0, errors }
}

/** The operations the apps may call: what the hub gives integration keys. */
export function integrationOperations(spec = loadSpec()) {
  const out = []
  for (const [id, o] of operations(spec)) {
    if (o.op['x-integration'] || o.op['x-access'] === 'key') out.push(id)
  }
  return out.sort()
}
