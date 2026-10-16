'use strict';

const { apiUrl, listOf } = require('../lib/client');
const { emailField, zapNameField } = require('../lib/fields');

// Add Tag to Contact: one or more tags on an existing contact.

const perform = async (z, bundle) => {
  const d = bundle.inputData;
  const response = await z.request({
    url: apiUrl(bundle, '/workspaces/{workspace}/contacts/tags'),
    method: 'POST',
    body: { email: String(d.email || '').trim(), tags: listOf(d.tags) },
  });
  return { id: response.data.contactId, ...response.data };
};

module.exports = {
  key: 'add_tag',
  noun: 'Tag',
  display: {
    label: 'Add Tag to Contact',
    description: 'Adds one or more tags to an existing contact.',
  },
  operation: {
    inputFields: [
      emailField('The contact to tag.'),
      { key: 'tags', label: 'Tags', type: 'string', list: true, required: true, helpText: "Tags to add, such as `customer`. Tags they already have stay." },
      zapNameField,
    ],
    perform,
    sample: {
      id: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e02',
      contactId: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e02',
      email: 'dmitri@fabrikam.example',
      created: false,
      lists: [],
      tags: ['customer'],
      confirmationSent: false,
    },
    outputFields: [
      { key: 'id', label: 'Contact ID' },
      { key: 'email', label: 'Email' },
      { key: 'tags[]', label: 'Tags' },
    ],
  },
};
