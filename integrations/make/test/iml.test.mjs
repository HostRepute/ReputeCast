import { describe, it } from 'node:test'
import assert from 'node:assert/strict'

import { evaluate, parse, references } from '../scripts/iml.mjs'

describe('the IML evaluator', () => {
  const scope = {
    parameters: { email: 'priya.n@example.com', list: 'l1', tags: ['a'], properties: [{ name: 'plan', value: 'Team' }] },
    connection: { workspace: 'acme-newsletter' },
    scenario: { id: 4211, name: 'Beta → CRM' },
    statusCode: 409,
    body: { code: 'unsubscribed', items: [{ key: 'a', variables: [1] }, { key: 'b', variables: [2, 3] }] },
  }

  it('joins text and keeps a lone expression’s own value', () => {
    assert.equal(evaluate('/workspaces/{{connection.workspace}}/lists/{{parameters.list}}', scope), '/workspaces/acme-newsletter/lists/l1')
    assert.deepEqual(evaluate('{{parameters.tags}}', scope), ['a'])
    assert.equal(evaluate('{{statusCode = 409}}', scope), true)
    assert.equal(evaluate('{{statusCode}} answered', scope), '409 answered')
  })

  it('runs the functions the app uses', () => {
    assert.deepEqual(evaluate("{{omit(parameters, 'list', 'tags', 'properties')}}", scope), { email: 'priya.n@example.com' })
    assert.deepEqual(evaluate("{{toCollection(parameters.properties, 'name', 'value')}}", scope), { plan: 'Team' })
    assert.deepEqual(evaluate("{{first(map(body.items, 'variables', 'key', 'b'))}}", scope), [2, 3])
    assert.equal(evaluate("{{if(scenario.id, encodeURL(scenario.name) + '; id=' + scenario.id, '')}}", scope), 'Beta%20%E2%86%92%20CRM; id=4211')
    assert.equal(evaluate("{{ifempty(body.detail, 'none')}}", scope), 'none')
    assert.equal(evaluate("{{if(statusCode >= 500, 'ConnectionError', if(statusCode = 409, 'DataError', 'RuntimeError'))}}", scope), 'DataError')
    assert.equal(evaluate("{{length(parameters.tags) > 0 && !body.missing}}", scope), true)
  })

  it('leaves out keys whose value is undefined, as Make does in a request', () => {
    assert.deepEqual(evaluate({ a: '{{parameters.nothing}}', b: '{{if(false, 1)}}', c: 'x' }, scope), { c: 'x' })
  })

  it('lists the paths and functions a section uses', () => {
    const r = references({ url: '/x/{{connection.workspace}}', body: "{{omit(parameters, 'list')}}" })
    assert.deepEqual([...r.paths].sort(), ['connection.workspace', 'parameters'])
    assert.deepEqual([...r.fns], ['omit'])
  })

  it('refuses what it can’t parse', () => {
    assert.throws(() => parse("if(a, 'b'"), SyntaxError)
    assert.throws(() => parse("'open"), SyntaxError)
    assert.throws(() => evaluate('{{nope(1)}}', {}), /unknown IML function/)
  })
})
