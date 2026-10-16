'use strict';

const { apiUrl, compact } = require('../lib/client');
const { zapNameField } = require('../lib/fields');
const { zapOf } = require('../lib/workflow');

// Send Transactional Email: a published template, a recipient, the data
// and an optional language, through the Transactional route (phase 1's
// send API). It needs transactional:send, a sensitive permission, which
// starts off when connecting.

const VAR_PREFIX = 'data__';

// A template variable's value as the template expects it.
const typed = (type, v) => {
  if (v === undefined || v === null || v === '') {
    return undefined;
  }
  switch (type) {
    case 'number': {
      const n = typeof v === 'number' ? v : Number(String(v).trim());
      return Number.isFinite(n) ? n : v;
    }
    case 'boolean': {
      const s = String(v).trim().toLowerCase();
      if (['true', 'yes', '1'].includes(s)) {
        return true;
      }
      if (['false', 'no', '0'].includes(s)) {
        return false;
      }
      return v;
    }
    case 'list':
      if (Array.isArray(v)) {
        return v;
      }
      try {
        const parsed = JSON.parse(v);
        if (Array.isArray(parsed)) {
          return parsed;
        }
      } catch {
        // not JSON: a comma-separated list
      }
      return String(v)
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
    case 'object':
      if (typeof v === 'object') {
        return v;
      }
      try {
        const parsed = JSON.parse(v);
        return parsed && typeof parsed === 'object' ? parsed : v;
      } catch {
        return v;
      }
    default:
      return typeof v === 'string' ? v : String(v);
  }
};

const variableHelp = (v) => {
  const parts = [];
  if (v.description) {
    parts.push(v.description.replace(/\.?$/, '.'));
  }
  if (v.type === 'date') {
    parts.push('A date like 2026-10-30.');
  } else if (v.type === 'url') {
    parts.push('A web address.');
  } else if (v.type === 'list') {
    parts.push('A list: comma-separated values, or a JSON array.');
  } else if (v.type === 'object') {
    parts.push('A JSON object, like `{"plan": "Team"}`.');
  }
  if (v.sensitive) {
    parts.push("Sensitive: ReputeCast masks it in its call log and doesn't store it.");
  }
  return parts.join(' ') || undefined;
};

const templatesOf = async (z, bundle) => {
  const response = await z.request({ url: apiUrl(bundle, '/workspaces/{workspace}/integration/templates') });
  return response.data.items;
};

// The chosen template's variables, each its own input, required as the
// template says.
const variableFields = async (z, bundle) => {
  if (!bundle.inputData.template) {
    return [];
  }
  const t = (await templatesOf(z, bundle)).find((x) => x.key === bundle.inputData.template);
  if (!t) {
    return [];
  }
  return t.variables.map((v) => {
    const f = {
      key: `${VAR_PREFIX}${v.name}`,
      label: v.name,
      required: v.required,
      type: v.type === 'number' ? 'number' : v.type === 'boolean' ? 'boolean' : 'string',
      // The variable's type comes back with the run (bundle.meta.inputFields),
      // so sending needs no second look at the template.
      meta: { variableType: v.type },
    };
    const help = variableHelp(v);
    if (help) {
      f.helpText = help;
    }
    if (v.type === 'list') {
      f.list = true;
    }
    return f;
  });
};

const perform = async (z, bundle) => {
  const d = bundle.inputData;
  const meta = (bundle.meta && bundle.meta.inputFields) || {};
  const data = {};
  if (d.data && typeof d.data === 'object') {
    Object.assign(data, d.data);
  }
  for (const [k, v] of Object.entries(d)) {
    if (k.startsWith(VAR_PREFIX)) {
      const value = typed(meta[k] && meta[k].variableType, v);
      if (value !== undefined) {
        data[k.slice(VAR_PREFIX.length)] = value;
      }
    }
  }
  const body = { template: d.template, to: compact({ email: d.to_email, name: d.to_name }) };
  if (Object.keys(data).length) {
    body.data = data;
  }
  if (d.language) {
    body.language = String(d.language).trim();
  }
  const headers = {};
  if (d.idempotency_key) {
    // zap-284113907-evt_1QKx…: a replay of the same Zap run sends nothing new.
    const zap = zapOf(bundle).id;
    headers['Idempotency-Key'] = `zap-${zap ? `${zap}-` : ''}${String(d.idempotency_key).trim()}`.slice(0, 255);
  }
  const response = await z.request({ url: apiUrl(bundle, '/workspaces/{workspace}/transactional/messages'), method: 'POST', body, headers });
  return response.data;
};

module.exports = {
  key: 'send_transactional',
  noun: 'Email',
  display: {
    label: 'Send Transactional Email',
    description: 'Sends a published transactional template to one recipient, with its data.',
  },
  operation: {
    inputFields: [
      {
        key: 'template',
        label: 'Template',
        dynamic: 'template.id.name',
        required: true,
        altersDynamicFields: true,
        helpText: 'A published transactional template. Its live version is sent.',
      },
      { key: 'to_email', label: 'Recipient Email', type: 'string', required: true },
      { key: 'to_name', label: 'Recipient Name', type: 'string', required: false },
      variableFields,
      {
        key: 'data',
        label: 'More Data',
        dict: true,
        required: false,
        helpText: "Other variables the template uses, by name. The template's own variables are listed above.",
      },
      {
        key: 'language',
        label: 'Language',
        type: 'string',
        required: false,
        helpText: "A language code such as `es`, to send that language's version. Without it, the contact's language is used, then the workspace's fallback.",
      },
      {
        key: 'idempotency_key',
        label: 'Idempotency Key',
        type: 'string',
        required: false,
        helpText:
          "A value unique to this email, such as the triggering event's ID. If the step runs again with the same value within 24 hours, ReputeCast sends nothing new.",
      },
      zapNameField,
    ],
    perform,
    sample: { id: '0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44', status: 'queued' },
    outputFields: [
      { key: 'id', label: 'Message ID' },
      { key: 'status', label: 'Status' },
      { key: 'send_at', label: 'Send At', type: 'datetime' },
      { key: 'test', label: 'Test Mode', type: 'boolean' },
    ],
  },
};
