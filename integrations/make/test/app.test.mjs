import { describe, it } from 'node:test'
import assert from 'node:assert/strict'
import { cpSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

import { evaluate, output, request } from '../scripts/iml.mjs'
import { validateApp } from '../scripts/validate.mjs'
import { checkRequest, checkResponse } from '../../contract/openapi-contract.mjs'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
const read = (p) => JSON.parse(readFileSync(join(root, p), 'utf8'))
const app = read('app.json')
const base = read('src/base.imljson')

// Fields another workstream adds to the API in this release (X05 adds
// `language` to the transactional send API).
const pending = { 'send-transactional-message': ['language'] }

const connection = { instanceUrl: 'https://reputecast.acme.io', apiKey: `rc_live_7f0c3d9e41a8b62f05c7e1d4b3a2${'0'.repeat(36)}`, workspace: 'acme-newsletter' }
const scenario = { id: 4211, name: 'Typeform → ReputeCast' }
const beta = '0192a4c1-0000-7000-8000-00000000b3a3'

const send = (kind, name, parameters, extra = {}) => {
  const api = read(`src/${kind}/${name}/api.imljson`)
  const req = request(base, api, { connection, parameters, scenario, ...extra })
  const check = checkRequest({ method: req.method, url: req.url, body: req.body, headers: req.headers }, { pending })
  assert.deepEqual(check.errors, [], `${kind}/${name}: ${check.errors.join('; ')}`)
  return { req, api, operationId: check.operationId }
}

describe('the app definition', () => {
  it('validates', () => {
    assert.deepEqual(validateApp(root), [])
  })

  it('names what each module does, as the hub does', () => {
    const labels = Object.values(app.modules).map((m) => m.label)
    assert.deepEqual(labels, [
      'Watch Subscribed Contacts',
      'Watch Unsubscribed Contacts',
      'Watch Updated Contacts',
      'Watch Form Submissions',
      'Watch Sent Campaigns',
      'Watch Transactional Bounces',
      'Create or Update a Contact',
      'Add a Contact to a List',
      'Add a Tag to a Contact',
      'Record an Event',
      'Send a Transactional Email',
    ])
  })

  it('catches mistakes before they reach Make', () => {
    const dir = mkdtempSync(join(tmpdir(), 'reputecast-make-'))
    try {
      cpSync(root, dir, { recursive: true, filter: (p) => !p.includes('node_modules') && !p.includes(`${root}/dist`) })
      // validate.mjs reads the contract next to the real app.
      const bad = JSON.parse(readFileSync(join(dir, 'src/modules/addContactToList/expect.imljson'), 'utf8'))
      bad[0].options = 'rpc://listListz'
      writeFileSync(join(dir, 'src/modules/addContactToList/expect.imljson'), JSON.stringify(bad))
      const attach = JSON.parse(readFileSync(join(dir, 'src/webhooks/contactUpdated/attach.imljson'), 'utf8'))
      attach.body.event = 'contact.deleted'
      writeFileSync(join(dir, 'src/webhooks/contactUpdated/attach.imljson'), JSON.stringify(attach))
      const api = JSON.parse(readFileSync(join(dir, 'src/modules/addTagToContact/api.imljson'), 'utf8'))
      api.url = '/workspaces/{{connection.workspace}}/contacts/{{parameters.contactId}}/tags'
      api.body = '{{parameterz}}'
      writeFileSync(join(dir, 'src/modules/addTagToContact/api.imljson'), JSON.stringify(api))
      const problems = validateApp(dir)
      assert.ok(problems.some((p) => p.includes('rpc://listListz')), problems.join('\n'))
      assert.ok(problems.some((p) => p.includes('subscribes contact.deleted')), problems.join('\n'))
      assert.ok(problems.some((p) => p.includes("isn't an operation of web/openapi.json")), problems.join('\n'))
      assert.ok(problems.some((p) => p.includes('parameterz')), problems.join('\n'))
    } finally {
      rmSync(dir, { recursive: true, force: true })
    }
  })
})

describe('the connection', () => {
  const api = read('src/connections/reputecast/api.imljson')

  it('tests the key with GET /api/v1/integration and keeps its workspace', () => {
    const parameters = { instanceUrl: connection.instanceUrl, apiKey: connection.apiKey }
    const req = evaluate({ url: api.url, method: api.method, headers: api.headers }, { parameters })
    assert.equal(req.url, 'https://reputecast.acme.io/api/v1/integration')
    assert.equal(req.headers.Authorization, `Bearer ${connection.apiKey}`)
    assert.equal(req.headers['User-Agent'], `ReputeCast-Make/${app.release}`)
    assert.ok(checkRequest({ method: req.method, url: req.url }).ok)
    const me = { workspace: 'acme-newsletter', workspaceName: 'Acme Newsletter', integration: 'make', key: 'Make · Acme ops', triggers: [], actions: [], permissions: [] }
    assert.ok(checkResponse('get-integration', 200, me).ok)
    assert.deepEqual(evaluate(api.response.data, { body: me }), { workspace: 'acme-newsletter', workspaceName: 'Acme Newsletter' })
    assert.equal(evaluate(api.response.metadata.value, { body: me }), 'Make · Acme ops (Acme Newsletter)')
  })

  it('refuses a URL with a trailing slash and something that isn’t a key', () => {
    const [url, key] = read('src/connections/reputecast/parameters.imljson')
    const ok = (p, v) => new RegExp(p.validate.pattern).test(v)
    assert.ok(ok(url, 'https://reputecast.acme.io'))
    assert.ok(ok(url, 'http://10.0.4.12:8080/reputecast'))
    assert.ok(!ok(url, 'https://reputecast.acme.io/'))
    assert.ok(ok(key, connection.apiKey))
    assert.ok(!ok(key, 'hunter2'))
  })
})

describe('actions', () => {
  it('creates or updates a contact with the mapped fields, naming the scenario', () => {
    const { req } = send('modules', 'createOrUpdateContact', {
      email: 'priya.n@example.com',
      firstName: 'Priya',
      language: 'es_MX',
      tags: ['beta-applicant'],
      lists: [beta],
      consent: 'double',
      fields: { plan: 'trial' },
    })
    assert.equal(req.url, 'https://reputecast.acme.io/api/v1/workspaces/acme-newsletter/contacts/upsert')
    assert.equal(req.headers['X-ReputeCast-Workflow'], 'Typeform%20%E2%86%92%20ReputeCast; id=4211')
    assert.equal(req.headers.Authorization, `Bearer ${connection.apiKey}`)
    assert.deepEqual(req.body.fields, { plan: 'trial' })
  })

  it('leaves the workflow header out where Make gives no scenario', () => {
    const api = read('src/modules/addTagToContact/api.imljson')
    const req = request(base, api, { connection, parameters: { email: 'a@example.com', tags: ['x'] } })
    assert.equal(req.headers['X-ReputeCast-Workflow'], undefined)
  })

  it('adds to a list, the list in the path and not in the body', () => {
    const { req } = send('modules', 'addContactToList', { list: beta, email: 'rob@example.net', consent: 'single' })
    assert.equal(req.url, `https://reputecast.acme.io/api/v1/workspaces/acme-newsletter/lists/${beta}/contacts`)
    assert.deepEqual(req.body, { email: 'rob@example.net', consent: 'single' })
  })

  it('tags a contact', () => {
    const { req } = send('modules', 'addTagToContact', { email: 'dmitri@fabrikam.example', tags: ['customer'] })
    assert.deepEqual(req.body, { email: 'dmitri@fabrikam.example', tags: ['customer'] })
  })

  it('records an event, its properties as an object, or none', () => {
    const { req } = send('modules', 'recordEvent', { event: 'workspace.created', email: 'priya.n@example.com', properties: [{ name: 'plan', value: 'Team' }] })
    assert.deepEqual(req.body, { event: 'workspace.created', email: 'priya.n@example.com', properties: { plan: 'Team' } })
    const bare = send('modules', 'recordEvent', { event: 'teammate.invited', email: 'priya.n@example.com' })
    assert.deepEqual(bare.req.body, { event: 'teammate.invited', email: 'priya.n@example.com' })
  })

  it('sends a transactional email with its variables, a language and an idempotency key', () => {
    const { req } = send('modules', 'sendTransactionalEmail', {
      template: 'payment-failed',
      to: { email: 'finance@initech.example', name: 'Priya' },
      data: { amount_due: '$49.00', update_url: 'https://billing.acme.io/u/8812', seats: 12 },
      language: 'es',
      idempotencyKey: 'evt_1QKx',
    })
    assert.deepEqual(req.body, {
      template: 'payment-failed',
      to: { email: 'finance@initech.example', name: 'Priya' },
      data: { amount_due: '$49.00', update_url: 'https://billing.acme.io/u/8812', seats: 12 },
      language: 'es',
    })
    assert.equal(req.headers['Idempotency-Key'], 'make-4211-evt_1QKx')
    const plain = send('modules', 'sendTransactionalEmail', { template: 'payment-failed', to: { email: 'a@example.com' } })
    assert.equal(plain.req.headers['Idempotency-Key'], undefined)
  })

  it('outputs the API’s answer as the interface describes it', () => {
    for (const [name, op] of [
      ['createOrUpdateContact', 'upsert-contact'],
      ['addContactToList', 'add-contact-to-list'],
      ['addTagToContact', 'tag-contact'],
      ['recordEvent', 'send-automation-event'],
      ['sendTransactionalEmail', 'send-transactional-message'],
    ]) {
      const sample = read(`src/modules/${name}/samples.imljson`)
      const status = op === 'send-automation-event' || op === 'send-transactional-message' ? 202 : 200
      const r = checkResponse(op, status, sample)
      assert.deepEqual(r.errors, [], `${name}'s sample is a real ${op} answer`)
      const api = read(`src/modules/${name}/api.imljson`)
      assert.deepEqual(output(api, { body: sample }), sample)
    }
  })
})

describe('errors', () => {
  const kind = (statusCode, body) => evaluate(base.response.error.type, { statusCode, body })
  const message = (statusCode, body) => evaluate(base.response.error.message, { statusCode, body })

  it('ends a bundle in the way the problem calls for', () => {
    assert.equal(kind(401, { code: 'key_revoked' }), 'InvalidAccessTokenError')
    assert.equal(kind(403, { code: 'integration_operation_denied' }), 'InvalidConfigurationError')
    assert.equal(kind(404, { code: 'workspace_not_found' }), 'InvalidConfigurationError')
    assert.equal(kind(404, { code: 'not_found' }), 'RuntimeError')
    assert.equal(kind(409, { code: 'unsubscribed' }), 'DataError')
    assert.equal(kind(422, { code: 'missing_variables' }), 'DataError')
    assert.equal(kind(429, {}), 'RateLimitError')
    assert.equal(kind(503, {}), 'ConnectionError')
  })

  it('says it in ReputeCast’s words', () => {
    const detail = "rob@example.net left Beta Program on 2 Oct. ReputeCast doesn't add people back to a list they left."
    assert.equal(message(409, { code: 'unsubscribed', detail }), detail)
    assert.equal(message(500, {}), 'ReputeCast answered HTTP 500.')
    assert.equal(message(400, { title: 'Bad Request' }), 'Bad Request')
  })
})

describe('dropdowns and dynamic fields', () => {
  const answer = (op, body) => {
    const r = checkResponse(op, 200, body)
    assert.deepEqual(r.errors, [], `the mocked ${op} answer`)
    return body
  }

  it('lists lists, templates and forms', () => {
    const lists = answer('list-integration-lists', { items: [{ id: beta, name: 'Beta Program', doubleOptIn: true }, { id: 'x', name: 'Weekly Digest', doubleOptIn: false }] })
    const { api } = send('rpcs', 'listLists', {})
    assert.deepEqual(output(api, { body: lists }), [
      { label: 'Beta Program (double opt-in)', value: beta },
      { label: 'Weekly Digest', value: 'x' },
    ])
    const templates = answer('list-integration-templates', { items: [{ key: 'payment-failed', name: 'Payment failed', version: 1, variables: [] }] })
    assert.deepEqual(output(send('rpcs', 'listTemplates', {}).api, { body: templates }), [{ label: 'Payment failed (payment-failed)', value: 'payment-failed' }])
    const forms = answer('list-integration-forms', { items: [{ id: 'frm_8c2e1a', name: 'Newsletter signup', status: 'live' }, { id: 'frm_b7a412', name: 'Beta apply', status: 'paused' }] })
    assert.deepEqual(output(send('rpcs', 'listForms', {}).api, { body: forms }), [
      { label: 'Newsletter signup', value: 'frm_8c2e1a' },
      { label: 'Beta apply (paused)', value: 'frm_b7a412' },
    ])
  })

  it('turns custom fields and a template’s variables into parameters', () => {
    const fields = answer('list-integration-fields', { items: [{ key: 'plan', label: 'Plan', type: 'select' }, { key: 'seats', label: 'Seats', type: 'number' }, { key: 'renews_on', label: '', type: 'date' }] })
    assert.deepEqual(output(send('rpcs', 'listFields', {}).api, { body: fields }), [
      { name: 'plan', label: 'Plan', type: 'text', help: 'One of the field options in ReputeCast.' },
      { name: 'seats', label: 'Seats', type: 'number', help: '' },
      { name: 'renews_on', label: 'renews_on', type: 'text', help: 'A date like 2026-10-30.' },
    ])
    const templates = answer('list-integration-templates', {
      items: [
        { key: 'welcome', name: 'Welcome', version: 2, variables: [{ name: 'first_name', type: 'text', required: false, sensitive: false }] },
        {
          key: 'payment-failed',
          name: 'Payment failed',
          version: 1,
          variables: [
            { name: 'amount_due', type: 'text', required: true, sensitive: false, description: 'The amount, with its currency.' },
            { name: 'update_url', type: 'url', required: true, sensitive: true },
            { name: 'items', type: 'list', required: false, sensitive: false },
          ],
        },
      ],
    })
    const out = output(send('rpcs', 'templateVariables', { template: 'payment-failed' }).api, { body: templates, parameters: { template: 'payment-failed' } })
    assert.deepEqual(
      out.map((p) => [p.name, p.type, p.required]),
      [
        ['amount_due', 'text', true],
        ['update_url', 'url', true],
        ['items', 'any', false],
      ],
    )
    assert.equal(out[0].help, 'The amount, with its currency.')
    assert.match(out[1].help, /Sensitive/)
  })
})

describe('instant triggers', () => {
  it('attach the event’s managed webhook and detach it', () => {
    for (const [name, hook] of Object.entries(app.webhooks)) {
      const webhook = { url: 'https://hook.eu2.make.com/abc123def456', externalHookId: '0192b7e2-11aa-7c1e-9a40-3f1b2c8d5e99' }
      const attach = read(`src/webhooks/${name}/attach.imljson`)
      const req = request(base, attach, { connection, webhook, scenario })
      const check = checkRequest({ method: req.method, url: req.url, body: req.body })
      assert.deepEqual(check.errors, [], name)
      assert.deepEqual(req.body, { event: hook.event, url: webhook.url })
      const created = { id: webhook.externalHookId, event: hook.event, url: webhook.url, secret: 'whsec_x', signatureHeader: 'X-ReputeCast-Signature', timestampHeader: 'X-ReputeCast-Timestamp', createdAt: '2026-10-15T14:25:00Z' }
      assert.ok(checkResponse('subscribe-integration-hook', 201, created).ok)
      assert.deepEqual(evaluate(attach.response.data, { body: created }), { externalHookId: webhook.externalHookId })
      const detach = read(`src/webhooks/${name}/detach.imljson`)
      const d = request(base, detach, { connection, webhook })
      assert.equal(checkRequest({ method: d.method, url: d.url }).operationId, 'unsubscribe-integration-hook')
    }
  })

  it('pass the delivery on, and Watch Form Submissions keeps only the chosen form', () => {
    const api = read('src/webhooks/formSubmitted/api.imljson')
    const body = { id: 'evt_1', type: 'form.submitted', created_at: '2026-10-15T14:02:11Z', workspace: 'acme-newsletter', data: { form_id: 'frm_b7a412' } }
    assert.equal(evaluate(api.condition, { body, parameters: { form: 'frm_b7a412' } }), true)
    assert.equal(evaluate(api.condition, { body, parameters: { form: 'frm_8c2e1a' } }), false)
    assert.equal(evaluate(api.condition, { body, parameters: {} }), true)
    assert.deepEqual(evaluate(api.output, { body }), body)
  })

  it('show samples shaped like the webhooks’ deliveries', () => {
    for (const [name, m] of Object.entries(app.modules)) {
      if (m.typeId !== 10) continue
      const sample = read(`src/modules/${name}/samples.imljson`)
      assert.equal(sample.type, app.webhooks[m.webhook].event, name)
      assert.ok(sample.id && sample.created_at && sample.workspace && sample.data, name)
    }
  })
})
