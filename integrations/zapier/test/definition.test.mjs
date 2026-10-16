import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { App, require, zapier } from './helpers.mjs';
import { integrationOperations } from '../../contract/openapi-contract.mjs';

const { workflowHeader } = require('../lib/workflow');
const { messageOf, outcomeOf } = require('../lib/problems');
const { instanceBase, compact, listOf } = require('../lib/client');

describe('the app definition', () => {
  it('passes Zapier’s schema validation', async () => {
    const handler = zapier.createAppHandler(App);
    const out = await handler({ command: 'validate' });
    assert.deepEqual(out.results, []);
  });

  it('offers the hub’s five actions', () => {
    assert.deepEqual(Object.keys(App.creates), ['upsert_contact', 'add_to_list', 'add_tag', 'record_event', 'send_transactional']);
  });

  it('calls only operations the hub gives integration keys', () => {
    const allowed = new Set(integrationOperations());
    for (const op of [
      'get-integration',
      'subscribe-integration-hook',
      'unsubscribe-integration-hook',
      'get-integration-trigger-sample',
      'list-integration-lists',
      'list-integration-fields',
      'list-integration-templates',
      'list-integration-forms',
    ]) {
      assert.ok(allowed.has(op), `${op} is an integration operation`);
    }
  });
});

describe('the workflow header', () => {
  it('keeps plain names readable and percent-encodes the rest', () => {
    assert.equal(workflowHeader({ name: 'HubSpot deal won', id: 284113907 }), 'HubSpot deal won; id=284113907');
    assert.equal(workflowHeader({ name: 'Beta application (Typeform) → ReputeCast contact', id: '284113907' }), 'Beta application (Typeform) %E2%86%92 ReputeCast contact; id=284113907');
    assert.equal(workflowHeader({ name: 'Weekly; 100% = "done"' }), 'Weekly%3B 100%25 %3D %22done%22');
    assert.equal(workflowHeader({ name: '  Café\n  opens ' }), 'Caf%C3%A9 opens');
    assert.equal(workflowHeader({ id: 284113907 }), 'id=284113907');
    assert.equal(workflowHeader({ id: 'wf 2;x' }), 'id=wf2x');
    assert.equal(workflowHeader({}), '');
    assert.equal(workflowHeader({ name: 'x'.repeat(300) }).length, 200);
  });
});

describe('problems', () => {
  it('reads RFC 9457 problems as sentences', () => {
    assert.equal(messageOf(422, { detail: 'payment-failed v1 needs amount_due.', missing: ['amount_due'] }), 'payment-failed v1 needs amount_due.');
    assert.equal(messageOf(422, { detail: 'Some variables are missing', missing: ['amount_due', 'update_url'] }), 'Some variables are missing. Missing variables: amount_due, update_url.');
    // Huma's own validation answers put the details in errors[].
    assert.equal(
      messageOf(422, { detail: 'validation failed', errors: [{ location: 'body.email', message: 'expected length >= 3' }] }),
      'validation failed. email: expected length >= 3.',
    );
    // ReputeCast's checks say it once, in the detail.
    assert.equal(
      messageOf(422, { detail: 'seats is a number field.', errors: [{ location: 'body.fields.seats', message: 'seats is a number field.' }] }),
      'seats is a number field.',
    );
    assert.equal(messageOf(500, ''), 'ReputeCast answered HTTP 500.');
    assert.equal(messageOf(503, '{"title":"Service Unavailable"}'), 'Service Unavailable.');
  });

  it('skips a record when consent is kept, without failing the workflow', () => {
    assert.equal(outcomeOf(409, { code: 'unsubscribed', detail: 'x left y.' }).kind, 'skipped');
    assert.equal(outcomeOf(409, { code: 'suppressed', detail: 'x is suppressed.' }).kind, 'skipped');
    assert.equal(outcomeOf(422, { code: 'recipient_suppressed', detail: 'x is suppressed.' }).kind, 'skipped');
    assert.equal(outcomeOf(409, { code: 'conflict', detail: 'Already.' }).kind, 'error');
    assert.equal(outcomeOf(401, { code: 'key_expired', detail: 'Expired.' }).kind, 'reauth');
    assert.equal(outcomeOf(404, { code: 'workspace_not_found', detail: 'Workspace not found.' }).kind, 'workspace');
    assert.equal(outcomeOf(429, {}).kind, 'throttled');
    assert.equal(outcomeOf(200, {}).kind, 'ok');
  });
});

describe('helpers', () => {
  it('cleans the instance URL', () => {
    assert.equal(instanceBase('reputecast.acme.io'), 'https://reputecast.acme.io');
    assert.equal(instanceBase(' https://reputecast.acme.io/api/v1/ '), 'https://reputecast.acme.io');
    assert.equal(instanceBase('http://10.0.4.12:8080/'), 'http://10.0.4.12:8080');
    assert.equal(instanceBase('https://acme.io/reputecast/'), 'https://acme.io/reputecast');
  });

  it('drops what was left blank', () => {
    assert.deepEqual(compact({ a: '', b: ' x ', c: [], d: ['', 'y'], e: { f: '' }, g: false, h: 0 }), { b: 'x', d: ['y'], g: false, h: 0 });
    assert.deepEqual(listOf('a, b\nc'), ['a', 'b', 'c']);
    assert.deepEqual(listOf(['a,b', 'c']), ['a', 'b', 'c']);
  });
});
