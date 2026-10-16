'use strict';

const { apiUrl } = require('../lib/client');

// Hidden triggers that fill the dropdowns: lists (Create or Update Contact,
// Add Contact to List), published transactional templates (Send
// Transactional Email) and signup forms (Form Submitted's filter). Each
// works only when the key has the step that uses it turned on.

const list = {
  key: 'list',
  noun: 'List',
  display: { label: 'List', description: 'Lists, for the dropdowns.', hidden: true },
  operation: {
    perform: async (z, bundle) => {
      const response = await z.request({ url: apiUrl(bundle, '/workspaces/{workspace}/integration/lists') });
      return response.data.items.map((l) => ({
        id: l.id,
        name: l.doubleOptIn ? `${l.name} (double opt-in)` : l.name,
        double_opt_in: l.doubleOptIn,
      }));
    },
    sample: { id: '0192a4c1-0000-7000-8000-00000000b3a1', name: 'Weekly Digest', double_opt_in: false },
    outputFields: [
      { key: 'id', label: 'List ID' },
      { key: 'name', label: 'Name' },
      { key: 'double_opt_in', label: 'Double Opt-In', type: 'boolean' },
    ],
  },
};

const template = {
  key: 'template',
  noun: 'Template',
  display: { label: 'Transactional Template', description: 'Published transactional templates, for the dropdowns.', hidden: true },
  operation: {
    perform: async (z, bundle) => {
      const response = await z.request({ url: apiUrl(bundle, '/workspaces/{workspace}/integration/templates') });
      return response.data.items.map((t) => ({ id: t.key, name: `${t.name} (${t.key})`, version: t.version }));
    },
    sample: { id: 'payment-failed', name: 'Payment failed (payment-failed)', version: 1 },
    outputFields: [
      { key: 'id', label: 'Template ID' },
      { key: 'name', label: 'Name' },
      { key: 'version', label: 'Live Version', type: 'integer' },
    ],
  },
};

const form = {
  key: 'form',
  noun: 'Form',
  display: { label: 'Signup Form', description: 'Signup forms, for the Form Submitted filter.', hidden: true },
  operation: {
    perform: async (z, bundle) => {
      const response = await z.request({ url: apiUrl(bundle, '/workspaces/{workspace}/integration/forms') });
      return response.data.items.map((f) => ({ id: f.id, name: f.status === 'live' ? f.name : `${f.name} (${f.status})`, status: f.status }));
    },
    sample: { id: 'frm_8c2e1a', name: 'Newsletter signup', status: 'live' },
    outputFields: [
      { key: 'id', label: 'Form ID' },
      { key: 'name', label: 'Name' },
      { key: 'status', label: 'Status' },
    ],
  },
};

module.exports = [list, template, form];
