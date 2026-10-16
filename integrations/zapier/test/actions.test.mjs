import { afterEach, beforeEach, describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { App, WS, appTester, authData, mockServer, problem, setup, teardown } from './helpers.mjs';

const contactId = '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e01';
const betaProgram = '0192a4c1-0000-7000-8000-00000000b3a3';
const meta = { zap: { id: 284113907 } };
const appError = (err) => JSON.parse(err.message);

const upserted = {
  contactId,
  email: 'priya.n@example.com',
  created: true,
  lists: [{ id: betaProgram, name: 'Beta Program', status: 'pending' }],
  tags: ['beta-applicant'],
  confirmationSent: true,
};

describe('Create or Update Contact', () => {
  beforeEach(setup);
  afterEach(teardown);

  it('sends what was mapped, and names the Zap', async () => {
    const api = mockServer();
    api.expect('POST', `${WS}/contacts/upsert`, { body: upserted, operation: 'upsert-contact' });
    const out = await appTester(App.creates.upsert_contact.operation.perform, {
      authData,
      meta,
      inputData: {
        email: ' priya.n@example.com ',
        first_name: 'Priya',
        last_name: '',
        language: 'es_MX',
        tags: ['beta-applicant', ''],
        lists: [betaProgram],
        consent: 'double',
        field__plan: 'trial',
        field__seats: 3,
        field__empty: '',
        zap_name: 'Beta application (Typeform) → ReputeCast contact',
      },
    });
    assert.equal(out.id, contactId);
    assert.equal(out.confirmationSent, true);
    const call = api.calls[0];
    assert.deepEqual(call.body, {
      email: 'priya.n@example.com',
      firstName: 'Priya',
      language: 'es_MX',
      fields: { plan: 'trial', seats: 3 },
      tags: ['beta-applicant'],
      lists: [betaProgram],
      consent: 'double',
    });
    assert.equal(call.headers['x-reputecast-workflow'], 'Beta application (Typeform) %E2%86%92 ReputeCast contact; id=284113907');
    // ReputeCast's call log names the app from it: "Zapier, ReputeCast app 1.2.0".
    assert.match(call.headers['user-agent'], /^ReputeCast-Zapier\/\d+\.\d+\.\d+$/);
    assert.equal(call.headers.authorization, `Bearer ${authData.apiKey}`);
    api.done();
  });

  it('halts the run for a contact who left the list, without breaking the Zap', async () => {
    const api = mockServer();
    const detail = "rob@example.net left Beta Program on 2 Oct. ReputeCast doesn't add people back to a list they left.";
    api.expect('POST', `${WS}/contacts/upsert`, {
      status: 409,
      body: problem(409, 'unsubscribed', detail, { meta: { list: 'Beta Program', listId: betaProgram, leftAt: '2026-10-02T13:41:00Z' } }),
      operation: 'upsert-contact',
    });
    await assert.rejects(
      appTester(App.creates.upsert_contact.operation.perform, { authData, meta, inputData: { email: 'rob@example.net', lists: [betaProgram] } }),
      (err) => err.name === 'HaltedError' && err.message.startsWith(detail),
    );
    api.done();
  });

  it('halts for a suppressed address too', async () => {
    const api = mockServer();
    const detail = 'olga@example.org is on the suppression list since 12 Sep (hard bounce). ReputeCast doesn’t add suppressed addresses.';
    api.expect('POST', `${WS}/contacts/upsert`, {
      status: 409,
      body: problem(409, 'suppressed', detail, { meta: { reason: 'hard_bounce', since: '2026-09-12T08:00:00Z' } }),
      operation: 'upsert-contact',
    });
    await assert.rejects(appTester(App.creates.upsert_contact.operation.perform, { authData, inputData: { email: 'olga@example.org' } }), {
      name: 'HaltedError',
    });
    api.done();
  });

  it('turns the workspace’s custom fields into inputs', async () => {
    const api = mockServer();
    api.expect('GET', `${WS}/integration/fields`, {
      body: {
        items: [
          { key: 'plan', label: 'Plan', type: 'select' },
          { key: 'seats', label: 'Seats', type: 'number' },
          { key: 'renews_on', label: 'Renews on', type: 'date' },
          { key: 'vip', label: 'VIP', type: 'boolean' },
        ],
      },
      operation: 'list-integration-fields',
    });
    const fieldsFn = App.creates.upsert_contact.operation.inputFields.find((f) => typeof f === 'function');
    const fields = await appTester(fieldsFn, { authData });
    assert.deepEqual(
      fields.map((f) => [f.key, f.label, f.type]),
      [
        ['field__plan', 'Plan', 'string'],
        ['field__seats', 'Seats', 'number'],
        ['field__renews_on', 'Renews on', 'string'],
        ['field__vip', 'VIP', 'boolean'],
      ],
    );
    api.done();
  });

  it('explains a field the server refuses', async () => {
    const api = mockServer();
    const detail = 'seats is a number field; "many" isn\'t a number.';
    api.expect('POST', `${WS}/contacts/upsert`, {
      status: 422,
      body: problem(422, 'validation_failed', detail, { errors: [{ location: 'body.fields.seats', message: detail }] }),
      operation: 'upsert-contact',
    });
    await assert.rejects(appTester(App.creates.upsert_contact.operation.perform, { authData, inputData: { email: 'a@example.com', field__seats: 'many' } }), (err) => {
      const e = appError(err);
      return e.message === detail && e.code === 'validation_failed' && e.status === 422;
    });
    api.done();
  });
});

describe('Add Contact to List, Add Tag to Contact, Record Event', () => {
  beforeEach(setup);
  afterEach(teardown);

  it('adds to the chosen list', async () => {
    const api = mockServer();
    api.expect('POST', `${WS}/lists/${betaProgram}/contacts`, { body: { ...upserted, created: false }, operation: 'add-contact-to-list' });
    const out = await appTester(App.creates.add_to_list.operation.perform, { authData, inputData: { list: betaProgram, email: 'priya.n@example.com', consent: 'single' } });
    assert.equal(out.id, contactId);
    assert.deepEqual(api.calls[0].body, { email: 'priya.n@example.com', consent: 'single' });
    api.done();
  });

  it('says what to do about a contact who isn’t there yet', async () => {
    const api = mockServer();
    const detail = 'No contact with new@example.com in this workspace. Use “Create or update a contact” to add them.';
    api.expect('POST', `${WS}/lists/${betaProgram}/contacts`, { status: 404, body: problem(404, 'not_found', detail), operation: 'add-contact-to-list' });
    await assert.rejects(appTester(App.creates.add_to_list.operation.perform, { authData, inputData: { list: betaProgram, email: 'new@example.com' } }), (err) =>
      appError(err).message === detail,
    );
    api.done();
  });

  it('tags a contact', async () => {
    const api = mockServer();
    api.expect('POST', `${WS}/contacts/tags`, { body: { ...upserted, created: false, lists: [], tags: ['customer'] }, operation: 'tag-contact' });
    await appTester(App.creates.add_tag.operation.perform, { authData, inputData: { email: 'dmitri@fabrikam.example', tags: ['customer'] } });
    assert.deepEqual(api.calls[0].body, { email: 'dmitri@fabrikam.example', tags: ['customer'] });
    api.done();
  });

  it('records an event with its properties', async () => {
    const api = mockServer();
    api.expect('POST', `${WS}/automations/events`, { status: 202, body: { accepted: true, contactId, flows: 1 }, operation: 'send-automation-event' });
    const out = await appTester(App.creates.record_event.operation.perform, {
      authData,
      inputData: { event: 'workspace.created', email: 'priya.n@example.com', properties: { plan: 'Team' } },
    });
    assert.deepEqual(api.calls[0].body, { event: 'workspace.created', email: 'priya.n@example.com', properties: { plan: 'Team' } });
    assert.equal(out.flows, 1);
    api.done();
  });

  it('points to Change what Zapier can do when an action is off', async () => {
    const api = mockServer();
    const detail = 'This key belongs to Zapier and works only for what Zapier can do. Change what Zapier can do in Settings › Integrations.';
    api.expect('POST', `${WS}/automations/events`, { status: 403, body: problem(403, 'integration_operation_denied', detail), operation: 'send-automation-event' });
    await assert.rejects(appTester(App.creates.record_event.operation.perform, { authData, inputData: { event: 'x', email: 'a@example.com' } }), (err) => {
      const e = appError(err);
      return e.message === detail && e.code === 'integration_operation_denied';
    });
    api.done();
  });

  it('asks Zapier to learn the workspace again when its address changed', async () => {
    const api = mockServer();
    api.expect('POST', `${WS}/contacts/tags`, { status: 404, body: problem(404, 'workspace_not_found', 'Workspace not found.'), operation: 'tag-contact' });
    await assert.rejects(appTester(App.creates.add_tag.operation.perform, { authData, inputData: { email: 'a@example.com', tags: ['x'] } }), {
      name: 'RefreshAuthError',
    });
    api.done();
  });
});

const paymentFailed = {
  key: 'payment-failed',
  name: 'Payment failed',
  version: 1,
  variables: [
    { name: 'first_name', type: 'text', required: false, sensitive: false },
    { name: 'amount_due', type: 'text', required: true, sensitive: false, description: 'The amount, with its currency' },
    { name: 'update_url', type: 'url', required: true, sensitive: true },
    { name: 'seats', type: 'number', required: false, sensitive: false },
    { name: 'items', type: 'list', required: false, sensitive: false },
  ],
};

describe('Send Transactional Email', () => {
  beforeEach(setup);
  afterEach(teardown);

  it('offers the template’s variables, required as the template says', async () => {
    const api = mockServer();
    api.expect('GET', `${WS}/integration/templates`, { body: { items: [paymentFailed] }, operation: 'list-integration-templates' });
    const fn = App.creates.send_transactional.operation.inputFields.find((f) => typeof f === 'function');
    const fields = await appTester(fn, { authData, inputData: { template: 'payment-failed' } });
    assert.deepEqual(
      fields.map((f) => [f.key, f.required, f.type, f.meta.variableType]),
      [
        ['data__first_name', false, 'string', 'text'],
        ['data__amount_due', true, 'string', 'text'],
        ['data__update_url', true, 'string', 'url'],
        ['data__seats', false, 'number', 'number'],
        ['data__items', false, 'string', 'list'],
      ],
    );
    assert.match(fields[2].helpText, /Sensitive/);
    assert.equal(fields[4].list, true);
    api.done();
  });

  it('sends the template with typed data and an idempotency key tied to the Zap', async () => {
    const api = mockServer();
    api.expect('POST', `${WS}/transactional/messages`, {
      status: 202,
      body: { id: '0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44', status: 'queued' },
      operation: 'send-transactional-message',
    });
    const out = await appTester(App.creates.send_transactional.operation.perform, {
      authData,
      meta: {
        ...meta,
        inputFields: { data__seats: { variableType: 'number' }, data__items: { variableType: 'list' }, data__update_url: { variableType: 'url' } },
      },
      inputData: {
        template: 'payment-failed',
        to_email: 'finance@initech.example',
        to_name: 'Priya',
        data__first_name: 'Priya',
        data__amount_due: '$49.00',
        data__update_url: 'https://billing.acme.io/u/8812',
        data__seats: '12',
        data__items: ['Team plan', 'Extra seats'],
        data: { invoice: 'INV-2209' },
        idempotency_key: 'evt_1QKxFj2eZvKYlo2C',
        zap_name: 'Stripe payment failed → payment-failed email',
      },
    });
    assert.equal(out.status, 'queued');
    const call = api.calls[0];
    assert.deepEqual(call.body, {
      template: 'payment-failed',
      to: { email: 'finance@initech.example', name: 'Priya' },
      data: { invoice: 'INV-2209', first_name: 'Priya', amount_due: '$49.00', update_url: 'https://billing.acme.io/u/8812', seats: 12, items: ['Team plan', 'Extra seats'] },
    });
    assert.equal(call.headers['idempotency-key'], 'zap-284113907-evt_1QKxFj2eZvKYlo2C');
    assert.equal(call.headers['x-reputecast-workflow'], 'Stripe payment failed %E2%86%92 payment-failed email; id=284113907');
    api.done();
  });

  it('sends a language version when one is chosen', async () => {
    const api = mockServer();
    api.expect('POST', `${WS}/transactional/messages`, {
      status: 202,
      body: { id: '0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c45', status: 'queued' },
      operation: 'send-transactional-message',
    });
    await appTester(App.creates.send_transactional.operation.perform, {
      authData,
      inputData: { template: 'payment-failed', to_email: 'lucia@example.com', language: 'es', data__amount_due: '49 €', data__update_url: 'https://x.example' },
    });
    assert.equal(api.calls[0].body.language, 'es');
    assert.equal(api.calls[0].headers['idempotency-key'], undefined);
    api.done();
  });

  it('names the variables a run is missing', async () => {
    const api = mockServer();
    api.expect('POST', `${WS}/transactional/messages`, {
      status: 422,
      body: problem(422, 'missing_variables', 'payment-failed v1 needs amount_due.', { type: 'https://reputecast.acme.io/problems/missing_variables', missing: ['amount_due'] }),
      operation: 'send-transactional-message',
    });
    await assert.rejects(
      appTester(App.creates.send_transactional.operation.perform, { authData, inputData: { template: 'payment-failed', to_email: 'finance@initech.example' } }),
      (err) => {
        const e = appError(err);
        return e.message === 'payment-failed v1 needs amount_due.' && e.code === 'missing_variables' && e.status === 422;
      },
    );
    api.done();
  });

  it('halts for a suppressed recipient', async () => {
    const api = mockServer();
    api.expect('POST', `${WS}/transactional/messages`, {
      status: 422,
      body: problem(422, 'recipient_suppressed', 'sam.rivera@outlook.com is suppressed (hard bounce) since 9 Oct 2026.'),
      operation: 'send-transactional-message',
    });
    await assert.rejects(
      appTester(App.creates.send_transactional.operation.perform, { authData, inputData: { template: 'password-reset', to_email: 'sam.rivera@outlook.com' } }),
      { name: 'HaltedError' },
    );
    api.done();
  });
});
