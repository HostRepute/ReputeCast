#!/usr/bin/env node
// Checks the Make app's definition before it goes to Make: every component
// app.json declares has its sections, the sections hold only directives
// Make knows, every {{ }} parses and uses known functions and declared
// parameters, rpc:// and webhook references exist, the groups list every
// module once, samples fit interfaces, and each communication section calls
// an operation of web/openapi.json the hub gives integration keys.
//
//   node scripts/validate.mjs        (exits 1 and lists the problems)

import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

import { FUNCTIONS, references } from './iml.mjs'
import { findOperation, integrationOperations } from '../../contract/openapi-contract.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')

export const EVENTS = ['contact.subscribed', 'contact.unsubscribed', 'contact.updated', 'form.submitted', 'campaign.completed', 'message.bounced']

// Directives Make knows in each kind of section.
const API_KEYS = new Set(['url', 'method', 'headers', 'qs', 'body', 'type', 'response', 'log', 'condition', 'temp', 'pagination', 'repeat', 'aws', 'ca'])
const RESPONSE_KEYS = new Set(['output', 'iterate', 'data', 'metadata', 'error', 'valid', 'uid', 'wrapper', 'limit', 'temp', 'type', 'trigger'])
const WEBHOOK_API_KEYS = new Set(['output', 'iterate', 'condition', 'respond', 'verification', 'uid', 'temp'])
const BASE_KEYS = new Set(['baseUrl', 'headers', 'qs', 'body', 'response', 'log', 'oauth'])
const PARAM_TYPES = new Set([
  'any', 'array', 'boolean', 'buffer', 'cert', 'collection', 'color', 'date', 'email', 'filename', 'file', 'filter', 'folder',
  'hidden', 'integer', 'number', 'password', 'path', 'pkey', 'port', 'select', 'text', 'time', 'timestamp', 'timezone',
  'uinteger', 'url', 'uuid',
])
const MODULE_TYPES = { 4: 'action', 9: 'search', 10: 'instant trigger' }

function read(problems, path) {
  if (!existsSync(path)) {
    problems.push(`${rel(path)}: missing`)
    return undefined
  }
  try {
    return JSON.parse(readFileSync(path, 'utf8'))
  } catch (err) {
    problems.push(`${rel(path)}: not JSON (${err.message})`)
    return undefined
  }
}

const rel = (p) => p.replace(`${root}/`, '')

/** Every parameter name a list of parameter definitions declares, nested ones included. */
function paramNames(params, rpcs, problems, where) {
  const names = new Set()
  const visit = (list, prefix) => {
    if (!Array.isArray(list)) return
    for (const p of list) {
      if (!p || typeof p !== 'object') continue
      if (!p.name) problems.push(`${where}: a parameter without a name`)
      if (!PARAM_TYPES.has(p.type)) problems.push(`${where}: ${p.name} has type ${p.type}, which Make doesn't have`)
      if (!p.label && p.type !== 'hidden') problems.push(`${where}: ${p.name} has no label`)
      const name = prefix ? `${prefix}.${p.name}` : p.name
      names.add(name)
      const rpc = (v) => {
        if (typeof v === 'string' && v.startsWith('rpc://') && !rpcs.has(v.slice(6))) problems.push(`${where}: ${p.name} uses ${v}, which doesn't exist`)
      }
      if (p.type === 'select') {
        if (Array.isArray(p.options)) {
          for (const o of p.options) if (!('value' in o) || !o.label) problems.push(`${where}: an option of ${p.name} needs a label and a value`)
        } else if (typeof p.options === 'string') {
          rpc(p.options)
        } else if (p.options && typeof p.options === 'object') {
          rpc(p.options.store)
          if (Array.isArray(p.options.nested)) visit(p.options.nested, prefix)
          else rpc(p.options.nested)
        } else {
          problems.push(`${where}: select ${p.name} has no options`)
        }
      }
      if (p.type === 'collection') {
        if (Array.isArray(p.spec)) visit(p.spec, name)
        else if (typeof p.spec === 'string') rpc(p.spec)
        else problems.push(`${where}: collection ${p.name} has no spec`)
      }
      if (p.type === 'array' && p.spec && typeof p.spec === 'object' && p.spec.type === 'collection' && Array.isArray(p.spec.spec)) {
        visit(p.spec.spec, `${name}[]`)
      }
      if (p.validate?.pattern) {
        try {
          new RegExp(p.validate.pattern) // eslint-disable-line no-new
        } catch {
          problems.push(`${where}: ${p.name}'s pattern isn't a regular expression`)
        }
      }
    }
  }
  visit(params, '')
  return names
}

/** Checks the IML of a section: it parses, uses known functions and only the paths given. */
function checkIml(problems, where, section, allowed) {
  let refs
  try {
    refs = references(section)
  } catch (err) {
    problems.push(`${where}: ${err.message}`)
    return
  }
  for (const fn of refs.fns) {
    if (!(fn in FUNCTIONS)) problems.push(`${where}: uses ${fn}(), which the app's IML doesn't use elsewhere: add it to scripts/iml.mjs if Make has it`)
  }
  for (const p of refs.paths) {
    const [scope, name] = p.split('.')
    const rule = allowed[scope]
    if (rule === undefined) {
      problems.push(`${where}: uses ${p}, but ${scope} isn't available here`)
    } else if (rule instanceof Set && name !== undefined && !rule.has(name)) {
      problems.push(`${where}: uses ${p}, which isn't declared`)
    }
  }
}

/** The operation a section calls, from its method and URL template. */
function checkOperation(problems, where, api, ops) {
  if (!api || typeof api.url !== 'string') {
    problems.push(`${where}: no url`)
    return undefined
  }
  const sample = `https://reputecast.example/api/v1${api.url
    .replace('{{connection.workspace}}', 'acme-newsletter')
    .replace(/\{\{[^}]+\}\}/g, '0192a4c1-0000-7000-8000-00000000b3a3')}`
  const found = findOperation(api.method ?? 'GET', sample)
  if (!found) {
    problems.push(`${where}: ${api.method ?? 'GET'} ${api.url} isn't an operation of web/openapi.json`)
    return undefined
  }
  if (!ops.has(found.operationId)) problems.push(`${where}: ${found.operationId} isn't an operation the hub gives integration keys`)
  return found.operationId
}

function checkKeys(problems, where, section, keys) {
  if (!section || typeof section !== 'object' || Array.isArray(section)) return
  for (const k of Object.keys(section)) if (!keys.has(k)) problems.push(`${where}: ${k} isn't a directive Make knows here`)
  if (section.response) for (const k of Object.keys(section.response)) if (!RESPONSE_KEYS.has(k)) problems.push(`${where}: response.${k} isn't a directive Make knows`)
}

/** Keys of a sample that the interface doesn't describe. */
function sampleFits(problems, where, iface, sample, prefix = '') {
  if (!sample || typeof sample !== 'object' || Array.isArray(sample)) return
  const byName = new Map((Array.isArray(iface) ? iface : []).map((f) => [f.name, f]))
  for (const [k, v] of Object.entries(sample)) {
    const f = byName.get(k)
    if (!f) {
      problems.push(`${where}: the sample has ${prefix}${k}, which the interface doesn't describe`)
      continue
    }
    if (f.type === 'collection' && Array.isArray(f.spec)) sampleFits(problems, where, f.spec, v, `${prefix}${k}.`)
  }
}

/** Validates the app; returns its problems (empty when it's ready for Make). */
export function validateApp(dir = root) {
  const problems = []
  const src = join(dir, 'src')
  const app = read(problems, join(dir, 'app.json'))
  const pkg = read(problems, join(dir, 'package.json'))
  if (!app) return problems
  for (const k of ['name', 'label', 'description', 'version', 'release', 'theme', 'language', 'audience']) {
    if (app[k] === undefined || app[k] === '') problems.push(`app.json: ${k} is missing`)
  }
  if (!Number.isInteger(app.version) || app.version < 1) problems.push('app.json: version is Make\'s app version, a whole number')
  if (pkg && pkg.version !== app.release) problems.push(`app.json: release ${app.release} isn't package.json's version ${pkg.version}`)
  const ops = new Set(integrationOperations())
  const rpcs = new Set(Object.keys(app.rpcs ?? {}))
  const connections = new Set(Object.keys(app.connections ?? {}))

  // Base: the key, the app's name in the call log, the workflow header.
  const base = read(problems, join(src, 'base.imljson'))
  if (base) {
    checkKeys(problems, 'src/base.imljson', base, BASE_KEYS)
    const h = base.headers ?? {}
    if (!/^Bearer \{\{connection\.apiKey\}\}$/.test(h.Authorization ?? '')) problems.push('src/base.imljson: Authorization must be "Bearer {{connection.apiKey}}"')
    if (h['User-Agent'] !== `ReputeCast-Make/${app.release}`) problems.push(`src/base.imljson: User-Agent must be ReputeCast-Make/${app.release}`)
    if (!h['X-ReputeCast-Workflow']) problems.push('src/base.imljson: X-ReputeCast-Workflow is missing')
    if (!(base.log?.sanitize ?? []).includes('request.headers.authorization')) problems.push('src/base.imljson: the key must be sanitized from logs')
    if (base.baseUrl !== '{{connection.instanceUrl}}/api/v1') problems.push('src/base.imljson: baseUrl must be {{connection.instanceUrl}}/api/v1')
    checkIml(problems, 'src/base.imljson', base, { connection: true, scenario: true, body: true, statusCode: true })
  }

  // Connections: the key's test call keeps the workspace it belongs to.
  const connectionData = new Set()
  for (const [name, c] of Object.entries(app.connections ?? {})) {
    const where = `src/connections/${name}`
    if (c.type !== 'basic') problems.push(`${where}: type ${c.type}; an API key connection is basic`)
    const params = read(problems, join(src, 'connections', name, 'parameters.imljson'))
    const api = read(problems, join(src, 'connections', name, 'api.imljson'))
    const declared = paramNames(params, rpcs, problems, `${where}/parameters.imljson`)
    if (api) {
      checkKeys(problems, `${where}/api.imljson`, api, API_KEYS)
      checkIml(problems, `${where}/api.imljson`, api, { parameters: declared, body: true, statusCode: true })
      if (!/\{\{parameters\.instanceUrl\}\}\/api\/v1\/integration$/.test(api.url ?? '')) problems.push(`${where}/api.imljson: must call {{parameters.instanceUrl}}/api/v1/integration`)
      for (const k of Object.keys(api.response?.data ?? {})) connectionData.add(k)
      if (!connectionData.has('workspace')) problems.push(`${where}/api.imljson: response.data must keep the workspace`)
      if (!(api.log?.sanitize ?? []).includes('request.headers.authorization')) problems.push(`${where}/api.imljson: the key must be sanitized from logs`)
    }
    for (const p of declared) connectionData.add(p)
  }
  const connScope = connectionData

  // RPCs: dropdowns and dynamic fields, each used somewhere.
  const usedRpcs = new Set()
  const noteRpcs = (v) => {
    const s = JSON.stringify(v ?? null)
    for (const m of s.matchAll(/rpc:\/\/([A-Za-z0-9_]+)/g)) usedRpcs.add(m[1])
  }
  for (const [name, r] of Object.entries(app.rpcs ?? {})) {
    const where = `src/rpcs/${name}`
    if (!connections.has(r.connection)) problems.push(`${where}: connection ${r.connection} doesn't exist`)
    const api = read(problems, join(src, 'rpcs', name, 'api.imljson'))
    read(problems, join(src, 'rpcs', name, 'parameters.imljson'))
    if (api) {
      checkKeys(problems, `${where}/api.imljson`, api, API_KEYS)
      checkIml(problems, `${where}/api.imljson`, api, { connection: connScope, parameters: true, body: true, item: true, statusCode: true })
      checkOperation(problems, `${where}/api.imljson`, api, ops)
      if (!api.response?.iterate) problems.push(`${where}/api.imljson: an RPC iterates over the items it lists`)
    }
  }

  // Webhooks: attach registers the event's managed webhook, detach removes it.
  const usedWebhooks = new Set()
  for (const [name, w] of Object.entries(app.webhooks ?? {})) {
    const where = `src/webhooks/${name}`
    if (w.type !== 'web') problems.push(`${where}: type ${w.type}; a dedicated webhook is web`)
    if (!connections.has(w.connection)) problems.push(`${where}: connection ${w.connection} doesn't exist`)
    if (!EVENTS.includes(w.event)) problems.push(`${where}: ${w.event} isn't one of the hub's trigger events`)
    const params = read(problems, join(src, 'webhooks', name, 'parameters.imljson'))
    const declared = paramNames(params, rpcs, problems, `${where}/parameters.imljson`)
    noteRpcs(params)
    const api = read(problems, join(src, 'webhooks', name, 'api.imljson'))
    if (api) {
      checkKeys(problems, `${where}/api.imljson`, api, WEBHOOK_API_KEYS)
      checkIml(problems, `${where}/api.imljson`, api, { parameters: declared, body: true, headers: true, query: true, method: true })
    }
    const attach = read(problems, join(src, 'webhooks', name, 'attach.imljson'))
    if (attach) {
      checkKeys(problems, `${where}/attach.imljson`, attach, API_KEYS)
      checkIml(problems, `${where}/attach.imljson`, attach, { connection: connScope, parameters: declared, webhook: new Set(['url']), body: true, scenario: true })
      if (checkOperation(problems, `${where}/attach.imljson`, attach, ops) !== 'subscribe-integration-hook') problems.push(`${where}/attach.imljson: must subscribe the hook`)
      if (attach.body?.event !== w.event) problems.push(`${where}/attach.imljson: subscribes ${attach.body?.event}, not ${w.event}`)
      if (attach.body?.url !== '{{webhook.url}}') problems.push(`${where}/attach.imljson: the url must be {{webhook.url}}`)
      if (attach.response?.data?.externalHookId !== '{{body.id}}') problems.push(`${where}/attach.imljson: must keep the hook's id as externalHookId`)
    }
    const detach = read(problems, join(src, 'webhooks', name, 'detach.imljson'))
    if (detach) {
      checkKeys(problems, `${where}/detach.imljson`, detach, API_KEYS)
      checkIml(problems, `${where}/detach.imljson`, detach, { connection: connScope, webhook: new Set(['externalHookId']) })
      if (checkOperation(problems, `${where}/detach.imljson`, detach, ops) !== 'unsubscribe-integration-hook') problems.push(`${where}/detach.imljson: must unsubscribe the hook`)
    }
  }

  // Modules.
  const groups = read(problems, join(src, 'groups.imljson')) ?? []
  const grouped = groups.flatMap((g) => g.modules ?? [])
  for (const [name, m] of Object.entries(app.modules ?? {})) {
    const where = `src/modules/${name}`
    if (!MODULE_TYPES[m.typeId]) problems.push(`${where}: typeId ${m.typeId} isn't a type the app uses`)
    if (!m.label || !m.description) problems.push(`${where}: label and description are needed`)
    if (!connections.has(m.connection)) problems.push(`${where}: connection ${m.connection} doesn't exist`)
    const iface = read(problems, join(src, 'modules', name, 'interface.imljson'))
    const sample = read(problems, join(src, 'modules', name, 'samples.imljson'))
    read(problems, join(src, 'modules', name, 'parameters.imljson'))
    paramNames(iface, rpcs, problems, `${where}/interface.imljson`)
    sampleFits(problems, `${where}/samples.imljson`, iface, sample)
    if (grouped.filter((g) => g === name).length !== 1) problems.push(`${where}: must be in exactly one group of src/groups.imljson`)
    if (m.typeId === 10) {
      if (!app.webhooks?.[m.webhook]) problems.push(`${where}: webhook ${m.webhook} doesn't exist`)
      else usedWebhooks.add(m.webhook)
      continue
    }
    const expect = read(problems, join(src, 'modules', name, 'expect.imljson'))
    const declared = paramNames(expect, rpcs, problems, `${where}/expect.imljson`)
    noteRpcs(expect)
    const api = read(problems, join(src, 'modules', name, 'api.imljson'))
    if (api) {
      checkKeys(problems, `${where}/api.imljson`, api, API_KEYS)
      checkIml(problems, `${where}/api.imljson`, api, { connection: connScope, parameters: declared, body: true, scenario: true, statusCode: true })
      checkOperation(problems, `${where}/api.imljson`, api, ops)
    }
  }
  for (const g of grouped) if (!app.modules?.[g]) problems.push(`src/groups.imljson: ${g} isn't a module`)
  for (const w of Object.keys(app.webhooks ?? {})) if (!usedWebhooks.has(w)) problems.push(`src/webhooks/${w}: no instant trigger uses it`)
  for (const r of rpcs) if (!usedRpcs.has(r)) problems.push(`src/rpcs/${r}: nothing uses it`)
  // Every component on disk is declared in app.json.
  for (const [kind, map] of [['modules', app.modules], ['rpcs', app.rpcs], ['webhooks', app.webhooks], ['connections', app.connections]]) {
    const dir = join(src, kind)
    if (!existsSync(dir)) continue
    for (const d of readdirSync(dir)) if (!map?.[d]) problems.push(`src/${kind}/${d}: not declared in app.json`)
  }
  const triggers = Object.values(app.webhooks ?? {}).map((w) => w.event).sort()
  if (JSON.stringify(triggers) !== JSON.stringify([...EVENTS].sort())) problems.push(`app.json: the webhooks cover ${triggers.join(', ')}, not the hub's six events`)
  if (!existsSync(join(src, 'readme.md'))) problems.push('src/readme.md: missing')
  return problems
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const problems = validateApp()
  if (problems.length) {
    console.error(`The Make app has ${problems.length} problem${problems.length === 1 ? '' : 's'}:`)
    for (const p of problems) console.error(`- ${p}`)
    process.exit(1)
  }
  console.log('The Make app is valid: every section, reference and API call checks out.')
}
