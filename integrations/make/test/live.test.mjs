// The app's sections run against a live ReputeCast through the IML
// evaluator, skipped unless REPUTECAST_URL and REPUTECAST_KEY name one
// (README › Test). The key needs every trigger and action on; for the
// trigger test the instance must be allowed to call this machine
// (webhooks.allow_private_networks).

import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import http from 'node:http'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

import { evaluate, output, request } from '../scripts/iml.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const read = (p) => JSON.parse(readFileSync(join(root, p), 'utf8'))
const base = read('src/base.imljson')
const url = process.env.REPUTECAST_URL
const key = process.env.REPUTECAST_KEY
const skip = !url || !key ? 'set REPUTECAST_URL and REPUTECAST_KEY to run against a live instance' : false
const scenario = { id: 900000002, name: 'Live test → ReputeCast' }

const call = async (req) => {
  const res = await fetch(req.url, {
    method: req.method,
    headers: { 'Content-Type': 'application/json', ...req.headers },
    body: req.body === undefined ? undefined : JSON.stringify(req.body),
  })
  const text = await res.text()
  return { statusCode: res.status, body: text ? JSON.parse(text) : undefined }
}

describe('a live instance', { skip }, () => {
  const connection = { instanceUrl: url, apiKey: key }

  const run = async (kind, name, parameters, extra = {}) => {
    const api = read(`src/${kind}/${name}/api.imljson`)
    const res = await call(request(base, api, { connection, parameters, scenario, ...extra }))
    assert.ok(res.statusCode < 400, `${name}: ${res.statusCode} ${JSON.stringify(res.body)} (${evaluate(base.response.error.message, res)})`)
    return output(api, { body: res.body, parameters, connection })
  }

  it('connects: the test call keeps the workspace', async () => {
    const api = read('src/connections/reputecast/api.imljson')
    const req = evaluate({ url: api.url, method: api.method, headers: api.headers }, { parameters: connection })
    const res = await call(req)
    assert.equal(res.statusCode, 200)
    Object.assign(connection, evaluate(api.response.data, { body: res.body }))
    assert.ok(connection.workspace)
  })

  it('fills the dropdowns and dynamic fields', async () => {
    const lists = await run('rpcs', 'listLists', {})
    assert.ok(lists.length > 0 && lists.every((o) => o.label && o.value))
    await run('rpcs', 'listForms', {})
    await run('rpcs', 'listFields', {})
    const templates = await run('rpcs', 'listTemplates', {})
    if (templates.length) {
      const vars = await run('rpcs', 'templateVariables', { template: templates[0].value })
      assert.ok(vars.every((v) => v.name && v.type))
    }
  })

  it('runs every action', async () => {
    const [list] = await run('rpcs', 'listLists', {})
    const email = `make-live+${Date.now()}@example.com`
    const created = await run('modules', 'createOrUpdateContact', { email, firstName: 'Live', language: 'Spanish', tags: ['live-test'], lists: [list.value], consent: 'single' })
    assert.equal(created.created, true)
    const tagged = await run('modules', 'addTagToContact', { email, tags: ['customer'] })
    assert.deepEqual(tagged.tags, ['customer', 'live-test'])
    const event = await run('modules', 'recordEvent', { event: 'live.test', email, properties: [{ name: 'plan', value: 'Team' }] })
    assert.equal(event.accepted, true)
    const templates = await run('rpcs', 'listTemplates', {})
    if (templates.length) {
      const template = templates[0].value
      const vars = await run('rpcs', 'templateVariables', { template })
      const data = Object.fromEntries(vars.map((v) => [v.name, v.type === 'number' ? 3 : v.type === 'boolean' ? true : v.type === 'url' ? 'https://example.com/x' : v.type === 'any' ? ['one'] : 'Live']))
      const sent = await run('modules', 'sendTransactionalEmail', { template, to: { email: 'make-live@example.com' }, data, idempotencyKey: `live-${Date.now()}` })
      assert.ok(['queued', 'scheduled', 'not_delivered'].includes(sent.status))
    }
  })

  it('attaches a webhook, passes a delivery on, and detaches it', async () => {
    const deliveries = []
    const server = http.createServer((req, res) => {
      let body = ''
      req.on('data', (c) => {
        body += c
      })
      req.on('end', () => {
        res.end('ok')
        deliveries.push(JSON.parse(body))
      })
    })
    await new Promise((resolve) => {
      server.listen(0, '127.0.0.1', resolve)
    })
    const webhook = { url: `http://127.0.0.1:${server.address().port}/make/live` }
    const attach = read('src/webhooks/contactSubscribed/attach.imljson')
    const res = await call(request(base, attach, { connection, webhook, scenario }))
    assert.equal(res.statusCode, 201)
    Object.assign(webhook, evaluate(attach.response.data, { body: res.body }))
    try {
      const [list] = await run('rpcs', 'listLists', {})
      const email = `make-hook+${Date.now()}@example.com`
      await run('modules', 'createOrUpdateContact', { email, lists: [list.value], consent: 'single' })
      const deadline = Date.now() + 60000
      let delivery
      while (!delivery && Date.now() < deadline) {
        delivery = deliveries.find((d) => d.data?.email === email)
        if (!delivery) {
          await new Promise((resolve) => {
            setTimeout(resolve, 500)
          })
        }
      }
      assert.ok(delivery, 'the contact.subscribed delivery arrived')
      const api = read('src/webhooks/contactSubscribed/api.imljson')
      assert.equal(evaluate(api.output, { body: delivery }).data.email, email)
    } finally {
      const detach = read('src/webhooks/contactSubscribed/detach.imljson')
      const d = await call(request(base, detach, { connection, webhook }))
      assert.equal(d.statusCode, 204)
      server.close()
    }
  })
})
