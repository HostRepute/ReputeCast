import { afterEach, beforeEach, describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { App, WS, appTester, authData, mockServer, require, setup, teardown } from './helpers.mjs';

const { sign } = require('../lib/signature');

const hookId = '0192b7e2-11aa-7c1e-9a40-3f1b2c8d5e99';
const secret = 'whsec_3f9a1c2b7e4d4c1a9b8e7f6a5d4c3b2a';
const targetUrl = 'https://hooks.zapier.com/hooks/standard/1234567/abcdef/';

const delivery = (type, data, id = 'evt_01JA9ZK7Q2X8M') => ({ id, type, created_at: '2026-10-15T14:02:11Z', workspace: 'acme-newsletter', data });

// A delivery as Zapier hands it to perform: the parsed body, and the raw
// request with Zapier's "Http-" header names.
const raw = (body, { ts = Math.floor(Date.now() / 1000), key = secret } = {}) => {
  const content = JSON.stringify(body);
  return {
    cleanedRequest: body,
    rawRequest: {
      method: 'POST',
      content,
      headers: { 'Http-X-Reputecast-Signature': sign(key, ts, content), 'Http-X-Reputecast-Timestamp': String(ts), 'Http-Content-Type': 'application/json' },
    },
  };
};

describe('instant triggers', () => {
  beforeEach(setup);
  afterEach(teardown);

  it('lists the six triggers of the hub', () => {
    const hooks = Object.values(App.triggers).filter((t) => t.operation.type === 'hook');
    assert.deepEqual(
      hooks.map((t) => [t.key, t.display.label]),
      [
        ['contact_subscribed', 'Contact Subscribed'],
        ['contact_unsubscribed', 'Contact Unsubscribed'],
        ['contact_updated', 'Contact Updated'],
        ['form_submitted', 'Form Submitted'],
        ['campaign_sent', 'Campaign Sent'],
        ['transactional_bounced', 'Transactional Message Bounced'],
      ],
    );
  });

  it('subscribes when the Zap is turned on, naming the Zap', async () => {
    const api = mockServer();
    const created = {
      id: hookId,
      event: 'contact.unsubscribed',
      url: targetUrl,
      secret,
      signatureHeader: 'X-ReputeCast-Signature',
      timestampHeader: 'X-ReputeCast-Timestamp',
      createdAt: '2026-10-15T14:02:11Z',
    };
    api.expect('POST', `${WS}/integration/hooks`, { status: 201, body: created, operation: 'subscribe-integration-hook' });
    const out = await appTester(App.triggers.contact_unsubscribed.operation.performSubscribe, {
      authData,
      targetUrl,
      meta: { zap: { id: 284120001 } },
      inputData: { zap_name: 'Unsubscribes → Acme Cloud CRM' },
    });
    assert.equal(out.secret, secret);
    assert.deepEqual(api.calls[0].body, { event: 'contact.unsubscribed', url: targetUrl });
    assert.equal(api.calls[0].headers['x-reputecast-workflow'], 'Unsubscribes %E2%86%92 Acme Cloud CRM; id=284120001');
    api.done();
  });

  it('unsubscribes when the Zap is turned off, and doesn’t mind a webhook already gone', async () => {
    const api = mockServer();
    api.expect('DELETE', `${WS}/integration/hooks/${hookId}`, { status: 204, operation: 'unsubscribe-integration-hook' });
    api.expect('DELETE', `${WS}/integration/hooks/${hookId}`, { status: 404, body: { title: 'Not Found', status: 404, detail: 'Webhook not found.', code: 'not_found' } });
    const bundle = { authData, subscribeData: { id: hookId, secret } };
    assert.deepEqual(await appTester(App.triggers.contact_unsubscribed.operation.performUnsubscribe, bundle), {});
    assert.deepEqual(await appTester(App.triggers.contact_unsubscribed.operation.performUnsubscribe, bundle), {});
    api.done();
  });

  it('starts the Zap with a signed delivery, its data at the top', async () => {
    const body = delivery('contact.unsubscribed', {
      contact_id: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e01',
      email: 'rob@example.net',
      first_name: 'Rob',
      list_id: '0192a4c1-0000-7000-8000-00000000b3a3',
      list_name: 'Beta Program',
      source: 'preference_center',
    });
    const out = await appTester(App.triggers.contact_unsubscribed.operation.perform, { authData, subscribeData: { id: hookId, secret }, ...raw(body) });
    assert.equal(out.length, 1);
    assert.equal(out[0].id, 'evt_01JA9ZK7Q2X8M');
    assert.equal(out[0].event, 'contact.unsubscribed');
    assert.equal(out[0].email, 'rob@example.net');
    assert.equal(out[0].list_name, 'Beta Program');
  });

  it('ignores a delivery that isn’t signed with this Zap’s secret, or is too old', async () => {
    const body = delivery('contact.unsubscribed', { email: 'rob@example.net' });
    const forged = raw(body, { key: 'whsec_someone_else' });
    assert.deepEqual(await appTester(App.triggers.contact_unsubscribed.operation.perform, { authData, subscribeData: { id: hookId, secret }, ...forged }), []);
    const old = raw(body, { ts: Math.floor(Date.now() / 1000) - 600 });
    assert.deepEqual(await appTester(App.triggers.contact_unsubscribed.operation.perform, { authData, subscribeData: { id: hookId, secret }, ...old }), []);
    const unsigned = { cleanedRequest: body, rawRequest: { content: JSON.stringify(body), headers: {} } };
    assert.deepEqual(await appTester(App.triggers.contact_unsubscribed.operation.perform, { authData, subscribeData: { id: hookId, secret }, ...unsigned }), []);
  });

  it('keeps only the chosen form’s signups', async () => {
    const op = App.triggers.form_submitted.operation;
    assert.equal(op.inputFields[0].dynamic, 'form.id.name');
    const mine = delivery('form.submitted', { form_id: 'frm_b7a412', form_name: 'Beta apply', email: 'priya.n@example.com' });
    const other = delivery('form.submitted', { form_id: 'frm_8c2e1a', form_name: 'Newsletter signup', email: 'lucia@example.com' }, 'evt_2');
    const bundle = { authData, subscribeData: { id: hookId, secret }, inputData: { form: 'frm_b7a412' } };
    assert.equal((await appTester(op.perform, { ...bundle, ...raw(mine) })).length, 1);
    assert.equal((await appTester(op.perform, { ...bundle, ...raw(other) })).length, 0);
    // Without a form chosen, every form's signups start it.
    assert.equal((await appTester(op.perform, { ...bundle, inputData: {}, ...raw(other) })).length, 1);
  });

  it('loads a sample delivery for field mapping', async () => {
    const api = mockServer();
    api.expect('GET', `${WS}/integration/samples/campaign.completed`, {
      body: {
        event: 'campaign.completed',
        payload: delivery('campaign.completed', { campaign_id: '0192a4c1-0000-7000-8000-0000000c0142', name: 'Weekly Digest #142', status: 'sent', sent: 31604 }, 'evt_sample'),
      },
      operation: 'get-integration-trigger-sample',
    });
    const out = await appTester(App.triggers.campaign_sent.operation.performList, { authData });
    assert.equal(out[0].id, 'evt_sample');
    assert.equal(out[0].name, 'Weekly Digest #142');
    assert.equal(out[0].sent, 31604);
    api.done();
  });

  it('has a static sample with an id for every trigger', () => {
    for (const t of Object.values(App.triggers)) {
      assert.ok(t.operation.sample && t.operation.sample.id, `${t.key} has a sample with an id`);
    }
  });
});

describe('dropdowns', () => {
  beforeEach(setup);
  afterEach(teardown);

  it('lists lists, templates and forms', async () => {
    const api = mockServer();
    api.expect('GET', `${WS}/integration/lists`, {
      body: {
        items: [
          { id: '0192a4c1-0000-7000-8000-00000000b3a1', name: 'Weekly Digest', doubleOptIn: false },
          { id: '0192a4c1-0000-7000-8000-00000000b3a3', name: 'Beta Program', doubleOptIn: true },
        ],
      },
      operation: 'list-integration-lists',
    });
    api.expect('GET', `${WS}/integration/templates`, {
      body: { items: [{ key: 'payment-failed', name: 'Payment failed', version: 1, variables: [] }] },
      operation: 'list-integration-templates',
    });
    api.expect('GET', `${WS}/integration/forms`, {
      body: {
        items: [
          { id: 'frm_8c2e1a', name: 'Newsletter signup', status: 'live' },
          { id: 'frm_b7a412', name: 'Beta apply', status: 'paused' },
        ],
      },
      operation: 'list-integration-forms',
    });
    const lists = await appTester(App.triggers.list.operation.perform, { authData });
    assert.deepEqual(
      lists.map((l) => l.name),
      ['Weekly Digest', 'Beta Program (double opt-in)'],
    );
    const templates = await appTester(App.triggers.template.operation.perform, { authData });
    assert.deepEqual(templates, [{ id: 'payment-failed', name: 'Payment failed (payment-failed)', version: 1 }]);
    const forms = await appTester(App.triggers.form.operation.perform, { authData });
    assert.deepEqual(
      forms.map((f) => [f.id, f.name]),
      [
        ['frm_8c2e1a', 'Newsletter signup'],
        ['frm_b7a412', 'Beta apply (paused)'],
      ],
    );
    api.done();
  });
});
