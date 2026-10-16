'use strict';

const { apiUrl } = require('../lib/client');
const { EVENTS, flatten } = require('../lib/events');
const { outcomeOf } = require('../lib/problems');
const { verify, headerOf } = require('../lib/signature');
const { zapNameField } = require('../lib/fields');

// Instant triggers through REST hooks: turning a Zap on subscribes (ReputeCast
// creates a managed webhook, shown read-only on its Webhooks page and on
// Settings › Integrations › Zapier), turning it off unsubscribes. When the
// Zap is deleted without unsubscribing, Zapier answers 410 Gone and
// ReputeCast removes the webhook itself.

const subscribe = (ev) => async (z, bundle) => {
  const response = await z.request({
    url: apiUrl(bundle, '/workspaces/{workspace}/integration/hooks'),
    method: 'POST',
    body: { event: ev.event, url: bundle.targetUrl },
  });
  // Kept by Zapier as bundle.subscribeData: the webhook's id, and its
  // signing secret, which ReputeCast shows only in this answer.
  return response.data;
};

const unsubscribe = async (z, bundle) => {
  const id = bundle.subscribeData && bundle.subscribeData.id;
  if (!id) {
    return {};
  }
  const response = await z.request({
    url: apiUrl(bundle, `/workspaces/{workspace}/integration/hooks/${encodeURIComponent(id)}`),
    method: 'DELETE',
    skipThrowForStatus: true,
  });
  // Already gone: removed after a 410 Gone, or when Zapier was disconnected.
  if (response.status === 404 || response.status < 400) {
    return {};
  }
  const out = outcomeOf(response.status, response.content);
  throw new z.errors.Error(out.message, out.code, response.status);
};

const perform = (ev) => async (z, bundle) => {
  const raw = bundle.rawRequest;
  const secret = bundle.subscribeData && bundle.subscribeData.secret;
  // Every delivery is signed with this Zap's webhook secret; one that isn't
  // didn't come from ReputeCast and starts nothing.
  if (secret && raw && typeof raw.content === 'string') {
    const check = verify({
      secret,
      body: raw.content,
      signature: headerOf(raw.headers, 'X-ReputeCast-Signature'),
      timestamp: headerOf(raw.headers, 'X-ReputeCast-Timestamp'),
    });
    if (!check.ok) {
      z.console.log(`Ignored a delivery: ${check.why}.`);
      return [];
    }
  }
  const env = bundle.cleanedRequest || {};
  if (env.type && env.type !== ev.event) {
    return [];
  }
  const item = flatten(env);
  if (ev.form && bundle.inputData.form && item.form_id !== bundle.inputData.form) {
    return [];
  }
  return [item];
};

// While setting up the Zap: a realistic delivery to map fields from,
// before anything happens in the workspace.
const sample = (ev) => async (z, bundle) => {
  const response = await z.request({ url: apiUrl(bundle, `/workspaces/{workspace}/integration/samples/${ev.event}`) });
  return [flatten(response.data.payload)];
};

const formField = {
  key: 'form',
  label: 'Form',
  dynamic: 'form.id.name',
  required: false,
  helpText: "Only this form's signups start the Zap. Leave it empty for every form.",
};

const trigger = (ev) => ({
  key: ev.key,
  noun: ev.noun,
  display: { label: ev.label, description: ev.description },
  operation: {
    type: 'hook',
    inputFields: [...(ev.form ? [formField] : []), zapNameField],
    performSubscribe: subscribe(ev),
    performUnsubscribe: unsubscribe,
    perform: perform(ev),
    performList: sample(ev),
    sample: ev.sample,
    outputFields: ev.outputFields,
  },
});

module.exports = EVENTS.map(trigger);
