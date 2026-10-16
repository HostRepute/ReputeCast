import { describe, it } from 'node:test'
import assert from 'node:assert/strict'

import { checkRequest, checkResponse, findOperation, integrationOperations, validate } from './openapi-contract.mjs'

const WS = 'https://reputecast.acme.io/api/v1/workspaces/acme-newsletter'

describe('the OpenAPI contract', () => {
  it('finds the operation a request calls', () => {
    assert.equal(findOperation('GET', 'https://reputecast.acme.io/api/v1/integration').operationId, 'get-integration')
    assert.equal(findOperation('POST', `${WS}/contacts/upsert`).operationId, 'upsert-contact')
    const hook = findOperation('DELETE', `${WS}/integration/hooks/0192b7e2-11aa-7c1e-9a40-3f1b2c8d5e99`)
    assert.equal(hook.operationId, 'unsubscribe-integration-hook')
    assert.equal(hook.params.hookId, '0192b7e2-11aa-7c1e-9a40-3f1b2c8d5e99')
    assert.equal(findOperation('PATCH', `${WS}/integration/samples/contact.updated`), null)
  })

  it('accepts a request the server accepts', () => {
    const r = checkRequest({ method: 'POST', url: `${WS}/contacts/tags`, body: { email: 'a@example.com', tags: ['customer'] } })
    assert.deepEqual(r.errors, [])
    assert.equal(r.operationId, 'tag-contact')
  })

  it('refuses fields the server doesn’t take, missing ones and wrong types', () => {
    const r = checkRequest({ method: 'POST', url: `${WS}/contacts/upsert`, body: { mail: 'a@example.com', consent: 'triple', tags: 'x' } })
    assert.ok(r.errors.includes('body.email: required'), r.errors.join('\n'))
    assert.ok(r.errors.includes('body.mail: not a property the server accepts'), r.errors.join('\n'))
    assert.ok(r.errors.some((e) => e.startsWith('body.consent:')), r.errors.join('\n'))
    assert.ok(r.errors.includes('body.tags: expected array, got string'), r.errors.join('\n'))
    assert.ok(checkRequest({ method: 'DELETE', url: `${WS}/integration/hooks/not-a-uuid` }).errors.includes('path.hookId: not a uuid'))
  })

  it('sets aside a field another workstream adds in this release, and says so', () => {
    const body = { template: 'payment-failed', to: { email: 'a@example.com' }, language: 'es' }
    const strict = checkRequest({ method: 'POST', url: `${WS}/transactional/messages`, body })
    const later = checkRequest({ method: 'POST', url: `${WS}/transactional/messages`, body }, { pending: { 'send-transactional-message': ['language'] } })
    if (strict.ok) {
      // The send API has it now: nothing is set aside.
      assert.deepEqual(later.pending, [])
    } else {
      assert.deepEqual(strict.errors, ['body.language: not a property the server accepts'])
      assert.deepEqual(later.errors, [])
      assert.deepEqual(later.pending, ['language'])
    }
  })

  it('checks mocked answers, letting problems carry their own fields', () => {
    assert.ok(checkResponse('tag-contact', 200, { contactId: 'c', email: 'a@example.com', created: false, lists: [], tags: [], confirmationSent: false }).ok)
    assert.ok(!checkResponse('tag-contact', 200, { contactId: 'c' }).ok)
    assert.ok(checkResponse('send-transactional-message', 422, { status: 422, code: 'missing_variables', detail: 'x', missing: ['amount_due'] }).ok)
  })

  it('treats a property set to undefined as absent, as JSON does', () => {
    assert.deepEqual(validate({ type: 'object', properties: { a: { type: 'string' } }, additionalProperties: false }, { a: undefined, b: undefined }), [])
  })

  it('knows what the hub gives integration keys', () => {
    const ops = integrationOperations()
    for (const op of ['get-integration', 'subscribe-integration-hook', 'get-integration-hook', 'list-integration-forms', 'upsert-contact', 'send-transactional-message', 'report-wordpress-site', 'create-wordpress-draft']) {
      assert.ok(ops.includes(op), op)
    }
    assert.ok(!ops.includes('list-campaigns'))
  })
})
