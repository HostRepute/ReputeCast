// End to end against a running ReputeCast, skipped unless REPUTECAST_URL
// and REPUTECAST_KEY name one (README › Test). The key needs every trigger
// and action on; the instance must be allowed to call this machine for
// the trigger test (webhooks.allow_private_networks).
//
//   REPUTECAST_URL=http://localhost:8080 REPUTECAST_KEY=rc_live_… npm test

import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';
import { App, appTester, require } from './helpers.mjs';

const { flatten } = require('../lib/events');

const url = process.env.REPUTECAST_URL;
const key = process.env.REPUTECAST_KEY;
const skip = !url || !key ? 'set REPUTECAST_URL and REPUTECAST_KEY to run against a live instance' : false;
const meta = { zap: { id: 900000001 } };
const zapName = 'Live test → ReputeCast';

/** A local endpoint that keeps what ReputeCast delivers to it. */
const receiver = async () => {
  const deliveries = [];
  const server = http.createServer((req, res) => {
    let body = '';
    req.on('data', (c) => {
      body += c;
    });
    req.on('end', () => {
      res.end('ok');
      deliveries.push({ headers: req.headers, body });
    });
  });
  await new Promise((resolve) => {
    server.listen(0, '127.0.0.1', resolve);
  });
  const waitFor = async (match, ms = 60000) => {
    const deadline = Date.now() + ms;
    for (;;) {
      const hit = deliveries.find(match);
      if (hit || Date.now() > deadline) {
        return hit;
      }
      await new Promise((resolve) => {
        setTimeout(resolve, 500);
      });
    }
  };
  return { url: `http://127.0.0.1:${server.address().port}/hooks/live`, waitFor, close: () => server.close() };
};

describe('a live instance', { skip }, () => {
  let authData;
  const run = (fn, bundle = {}) => appTester(fn, { authData, meta, ...bundle, inputData: { zap_name: zapName, ...(bundle.inputData || {}) } });

  it('connects with the URL and the key', async () => {
    const session = await appTester(App.authentication.sessionConfig.perform, { authData: { instanceUrl: url, apiKey: key } });
    authData = { instanceUrl: url, apiKey: key, ...session };
    const me = await appTester(App.authentication.test, { authData });
    assert.equal(me.workspace, session.workspace);
    assert.ok(me.actions.length > 0, 'the key has actions on');
  });

  it('fills the dropdowns', async () => {
    const lists = await run(App.triggers.list.operation.perform);
    assert.ok(lists.length > 0);
    const forms = await run(App.triggers.form.operation.perform);
    assert.ok(forms.every((f) => /^frm_/.test(f.id)));
    await run(App.triggers.template.operation.perform);
  });

  it('creates, tags a contact and records an event for them', async () => {
    const email = `zapier-live+${Date.now()}@example.com`;
    const [list] = await run(App.triggers.list.operation.perform);
    const out = await run(App.creates.upsert_contact.operation.perform, {
      inputData: { email, first_name: 'Live', language: 'Spanish', tags: ['live-test'], lists: [list.id], consent: 'single' },
    });
    assert.equal(out.created, true);
    assert.deepEqual(out.tags, ['live-test']);
    const tagged = await run(App.creates.add_tag.operation.perform, { inputData: { email, tags: ['customer'] } });
    assert.deepEqual(tagged.tags, ['customer', 'live-test']);
    const event = await run(App.creates.record_event.operation.perform, { inputData: { event: 'live.test', email, properties: { plan: 'Team' } } });
    assert.equal(event.accepted, true);
  });

  it('sends a published transactional template', async () => {
    const templates = await run(App.triggers.template.operation.perform);
    if (!templates.length) {
      return;
    }
    const fn = App.creates.send_transactional.operation.inputFields.find((f) => typeof f === 'function');
    const fields = await run(fn, { inputData: { template: templates[0].id } });
    const inputData = { template: templates[0].id, to_email: 'zapier-live@example.com', idempotency_key: `live-${Date.now()}` };
    const inputFields = {};
    for (const f of fields) {
      inputData[f.key] = f.type === 'number' ? '3' : f.type === 'boolean' ? 'true' : f.list ? ['one'] : f.key.endsWith('url') ? 'https://example.com/x' : 'Live';
      inputFields[f.key] = f.meta;
    }
    const out = await run(App.creates.send_transactional.operation.perform, { meta: { ...meta, inputFields }, inputData });
    assert.ok(['queued', 'scheduled', 'not_delivered'].includes(out.status), JSON.stringify(out));
  });

  it('delivers a signed event to a Zap and stops when it is turned off', async () => {
    const hooks = await receiver();
    const trigger = App.triggers.contact_subscribed.operation;
    const subscribeData = await run(trigger.performSubscribe, { targetUrl: hooks.url });
    try {
      assert.match(subscribeData.secret, /^whsec_/);
      const [list] = await run(App.triggers.list.operation.perform);
      const email = `zapier-hook+${Date.now()}@example.com`;
      await run(App.creates.upsert_contact.operation.perform, { inputData: { email, lists: [list.id], consent: 'single' } });
      // Earlier steps' contacts may arrive first: wait for this one's.
      const delivery = await hooks.waitFor((d) => JSON.parse(d.body).data?.email === email);
      assert.ok(delivery, 'the contact.subscribed delivery arrived');
      // As Zapier hands a delivery to perform: "Http-" header names.
      const headers = Object.fromEntries(Object.entries(delivery.headers).map(([k, v]) => [`Http-${k}`, v]));
      const items = await run(trigger.perform, {
        subscribeData,
        cleanedRequest: JSON.parse(delivery.body),
        rawRequest: { method: 'POST', content: delivery.body, headers },
      });
      assert.equal(items.length, 1);
      assert.equal(items[0].email, email);
      assert.deepEqual(items[0], flatten(JSON.parse(delivery.body)));
    } finally {
      await run(trigger.performUnsubscribe, { subscribeData });
      hooks.close();
    }
  });
});
