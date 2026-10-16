'use strict';

const { apiUrl, compact } = require('../lib/client');
const { emailField, optInField, zapNameField } = require('../lib/fields');

// Add Contact to List: an existing contact, with single or double opt-in.
// Never adds someone back to a list they left (the run halts for that
// contact, with the date they left).

const perform = async (z, bundle) => {
  const d = bundle.inputData;
  const response = await z.request({
    url: apiUrl(bundle, `/workspaces/{workspace}/lists/${encodeURIComponent(d.list)}/contacts`),
    method: 'POST',
    body: compact({ email: d.email, consent: d.consent }),
  });
  return { id: response.data.contactId, ...response.data };
};

module.exports = {
  key: 'add_to_list',
  noun: 'Contact',
  display: {
    label: 'Add Contact to List',
    description: 'Adds an existing contact to a list, with single or double opt-in.',
  },
  operation: {
    inputFields: [
      { key: 'list', label: 'List', dynamic: 'list.id.name', required: true },
      emailField('The contact to add. To create them first, use Create or Update Contact.'),
      optInField,
      zapNameField,
    ],
    perform,
    sample: {
      id: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e01',
      contactId: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e01',
      email: 'priya.n@example.com',
      created: false,
      lists: [{ id: '0192a4c1-0000-7000-8000-00000000b3a2', name: 'Product Updates', status: 'subscribed' }],
      tags: ['beta-applicant'],
      confirmationSent: false,
    },
    outputFields: [
      { key: 'id', label: 'Contact ID' },
      { key: 'email', label: 'Email' },
      { key: 'lists[]name', label: 'Lists Name' },
      { key: 'lists[]status', label: 'Lists Status' },
      { key: 'confirmationSent', label: 'Confirmation Email Sent', type: 'boolean' },
    ],
  },
};
