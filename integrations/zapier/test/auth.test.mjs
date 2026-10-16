import { afterEach, beforeEach, describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { App, BASE, KEY, appTester, authData, mockServer, problem, setup, teardown } from './helpers.mjs';

const me = {
  workspace: 'acme-newsletter',
  workspaceName: 'Acme Newsletter',
  integration: 'zapier',
  key: 'Zapier · Acme ops',
  triggers: ['contact.subscribed'],
  actions: ['upsert_contact', 'add_tag'],
  permissions: ['subscribers:create', 'subscribers:edit', 'webhooks:manage'],
};

const appError = (err) => JSON.parse(err.message);

describe('connecting', () => {
  beforeEach(setup);
  afterEach(teardown);

  it('learns the key’s workspace from the instance URL and the key', async () => {
    const api = mockServer();
    api.expect('GET', '/api/v1/integration', { body: me, operation: 'get-integration' });
    // A URL typed without its scheme, with a trailing slash.
    const out = await appTester(App.authentication.sessionConfig.perform, { authData: { instanceUrl: 'reputecast.acme.io/', apiKey: KEY } });
    assert.deepEqual(out, { workspace: 'acme-newsletter' });
    assert.equal(api.calls[0].headers.authorization, `Bearer ${KEY}`);
    assert.match(api.calls[0].headers['user-agent'], /^ReputeCast-Zapier\/\d+\.\d+\.\d+$/);
    api.done();
  });

  it('drops a pasted API path from the instance URL', async () => {
    const api = mockServer();
    api.expect('GET', '/api/v1/integration', { body: me, operation: 'get-integration' });
    await appTester(App.authentication.sessionConfig.perform, { authData: { instanceUrl: `${BASE}/api/v1/`, apiKey: KEY } });
    api.done();
  });

  it('says why a revoked key is refused, in ReputeCast’s words', async () => {
    const api = mockServer();
    const detail = 'This key was revoked. Reconnect Zapier in Settings › Integrations to get a new key.';
    api.expect('GET', '/api/v1/integration', { status: 401, body: problem(401, 'key_revoked', detail), operation: 'get-integration' });
    await assert.rejects(appTester(App.authentication.sessionConfig.perform, { authData: { instanceUrl: BASE, apiKey: KEY } }), (err) => {
      const e = appError(err);
      return e.message === detail && e.code === 'key_revoked' && e.status === 401;
    });
    api.done();
  });

  it('refuses something that isn’t a key before calling', async () => {
    await assert.rejects(
      appTester(App.authentication.sessionConfig.perform, { authData: { instanceUrl: BASE, apiKey: 'hunter2' } }),
      (err) => appError(err).code === 'invalid_key',
    );
    await assert.rejects(
      appTester(App.authentication.sessionConfig.perform, { authData: { instanceUrl: '', apiKey: KEY } }),
      (err) => appError(err).code === 'invalid_instance_url',
    );
  });

  it('says when the address answers but isn’t ReputeCast', async () => {
    const api = mockServer();
    api.expect('GET', '/api/v1/integration', { body: { hello: 'world' } });
    await assert.rejects(appTester(App.authentication.sessionConfig.perform, { authData: { instanceUrl: BASE, apiKey: KEY } }), (err) =>
      appError(err).message.includes('not as ReputeCast does'),
    );
  });

  it('tests the connection with the same call, and names it after the key', async () => {
    const api = mockServer();
    api.expect('GET', '/api/v1/integration', { body: me, operation: 'get-integration' });
    const out = await appTester(App.authentication.test, { authData });
    assert.equal(out.workspace, 'acme-newsletter');
    const label = App.authentication.connectionLabel({}, { inputData: out });
    assert.equal(label, 'Zapier · Acme ops (Acme Newsletter)');
    api.done();
  });
});
