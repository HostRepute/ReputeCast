'use strict';

const { USER_AGENT, apiUrl, instanceBase } = require('./lib/client');
const { outcomeOf } = require('./lib/problems');

// The connection holds the instance URL and the key ReputeCast showed once
// when Zapier was connected (Settings › Integrations › Zapier). The key
// knows its workspace: connecting asks GET /api/v1/integration which one,
// and keeps the answer as a hidden field the calls' paths use.

const checkFields = (z, bundle) => {
  const base = instanceBase(bundle.authData.instanceUrl);
  if (!base) {
    throw new z.errors.Error('Paste the instance URL from ReputeCast, for example https://reputecast.acme.io.', 'invalid_instance_url', 400);
  }
  if (!URL.canParse(base)) {
    throw new z.errors.Error(`${bundle.authData.instanceUrl} isn't a web address. Paste the instance URL from ReputeCast, for example https://reputecast.acme.io.`, 'invalid_instance_url', 400);
  }
  const key = String(bundle.authData.apiKey || '').trim();
  if (!/^rc_[a-z]+_[A-Za-z0-9]{16,}$/.test(key)) {
    throw new z.errors.Error("That isn't a ReputeCast key. Paste the key ReputeCast showed when you connected Zapier; it starts with rc_live_.", 'invalid_key', 400);
  }
  return { base, key };
};

// The session call goes through fetch rather than z.request: Zapier turns
// any 401 from z.request into "refresh the connection", which would hide
// ReputeCast's reason (revoked, expired or unknown key) from the person
// connecting. Zapier still logs the call.
const getWorkspace = async (z, bundle) => {
  const { base, key } = checkFields(z, bundle);
  let response;
  try {
    response = await fetch(`${base}/api/v1/integration`, {
      headers: { Authorization: `Bearer ${key}`, Accept: 'application/json', 'User-Agent': USER_AGENT },
      redirect: 'error',
    });
  } catch (err) {
    throw new z.errors.Error(`Zapier couldn't reach ${base}: ${err.cause ? err.cause.message || err.cause.code : err.message}. Check the instance URL; ReputeCast must be reachable from the internet.`, 'unreachable', 502);
  }
  const text = await response.text();
  const out = outcomeOf(response.status, text);
  if (out.kind !== 'ok') {
    throw new z.errors.Error(out.message, out.code, response.status);
  }
  let me;
  try {
    me = JSON.parse(text);
  } catch {
    throw new z.errors.Error(`${base} answered, but not as ReputeCast does. Check the instance URL.`, 'not_reputecast', 502);
  }
  if (!me || !me.workspace) {
    throw new z.errors.Error(`${base} answered, but not as ReputeCast does. Check the instance URL.`, 'not_reputecast', 502);
  }
  if (me.integration && me.integration !== 'zapier') {
    z.console.log(`This key belongs to ${me.integration}, not Zapier; its calls are listed under ${me.integration} in ReputeCast.`);
  }
  return { workspace: me.workspace };
};

// Test connection: the call the connect dialog in ReputeCast waits for
// ("Zapier called at 14:31 · Test connection · 200").
const test = async (z, bundle) => {
  const response = await z.request({ url: apiUrl(bundle, '/integration') });
  return response.data;
};

module.exports = {
  type: 'session',
  fields: [
    {
      key: 'instanceUrl',
      label: 'Instance URL',
      type: 'string',
      required: true,
      isNoSecret: true,
      helpText:
        'Your ReputeCast address, as shown when you connect Zapier in ReputeCast (**Settings › Integrations › Zapier › Connect**), for example `https://reputecast.acme.io`.',
    },
    {
      key: 'apiKey',
      label: 'Key',
      type: 'password',
      required: true,
      helpText:
        "The key ReputeCast showed once when you connected Zapier. It starts with `rc_live_` and works only for the triggers and actions you turned on. If it's lost, rotate it in ReputeCast.",
    },
    {
      key: 'workspace',
      type: 'string',
      required: false,
      computed: true,
    },
  ],
  sessionConfig: { perform: getWorkspace },
  test,
  // "Zapier · Acme ops (Acme Newsletter)"
  connectionLabel: (z, bundle) => {
    const me = bundle.inputData || {};
    return me.workspaceName ? `${me.key} (${me.workspaceName})` : me.key;
  },
};
