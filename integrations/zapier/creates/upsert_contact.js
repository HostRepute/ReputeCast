'use strict';

const { apiUrl, compact, listOf } = require('../lib/client');
const { emailField, optInField, zapNameField } = require('../lib/fields');

// Create or Update Contact: matched by email. Sets names, custom fields,
// language and tags, and adds to lists with single or double opt-in.
// Creating needs subscribers:create; updating, subscribers:edit.

const FIELD_PREFIX = 'field__';

const fieldInput = (f) => {
  const out = { key: `${FIELD_PREFIX}${f.key}`, label: f.label || f.key, required: false, type: 'string' };
  switch (f.type) {
    case 'number':
      out.type = 'number';
      break;
    case 'boolean':
      out.type = 'boolean';
      break;
    case 'date':
      out.helpText = 'A date like 2026-10-30.';
      break;
    case 'select':
      out.helpText = "One of the field's options in ReputeCast (Audience › Fields).";
      break;
    default:
  }
  return out;
};

// The workspace's custom fields (Audience › Fields), each its own input.
const customFields = async (z, bundle) => {
  const response = await z.request({ url: apiUrl(bundle, '/workspaces/{workspace}/integration/fields') });
  return response.data.items.map(fieldInput);
};

const perform = async (z, bundle) => {
  const d = bundle.inputData;
  const fields = {};
  for (const [k, v] of Object.entries(d)) {
    if (k.startsWith(FIELD_PREFIX)) {
      fields[k.slice(FIELD_PREFIX.length)] = v;
    }
  }
  const body = compact({
    email: d.email,
    firstName: d.first_name,
    lastName: d.last_name,
    company: d.company,
    language: d.language,
    fields,
    tags: listOf(d.tags),
    lists: listOf(d.lists),
    consent: d.consent,
  });
  const response = await z.request({ url: apiUrl(bundle, '/workspaces/{workspace}/contacts/upsert'), method: 'POST', body });
  return { id: response.data.contactId, ...response.data };
};

module.exports = {
  key: 'upsert_contact',
  noun: 'Contact',
  display: {
    label: 'Create or Update Contact',
    description: 'Creates a contact, or updates the one with the same email, and adds them to lists with single or double opt-in.',
  },
  operation: {
    inputFields: [
      emailField('The contact is matched by this address: an existing contact is updated, otherwise one is created.'),
      { key: 'first_name', label: 'First Name', type: 'string', required: false },
      { key: 'last_name', label: 'Last Name', type: 'string', required: false },
      { key: 'company', label: 'Company', type: 'string', required: false },
      {
        key: 'language',
        label: 'Language',
        type: 'string',
        required: false,
        helpText: 'A language code such as `es` or `es-MX`. Names like Spanish and codes like es_MX work too.',
      },
      { key: 'tags', label: 'Tags', type: 'string', list: true, required: false, helpText: "Added to the contact's tags. Tags they already have stay." },
      { key: 'lists', label: 'Lists', dynamic: 'list.id.name', list: true, required: false, helpText: 'Lists to add the contact to.' },
      optInField,
      customFields,
      zapNameField,
    ],
    perform,
    sample: {
      id: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e01',
      contactId: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e01',
      email: 'priya.n@example.com',
      created: true,
      lists: [{ id: '0192a4c1-0000-7000-8000-00000000b3a3', name: 'Beta Program', status: 'pending' }],
      tags: ['beta-applicant'],
      confirmationSent: true,
    },
    outputFields: [
      { key: 'id', label: 'Contact ID' },
      { key: 'email', label: 'Email' },
      { key: 'created', label: 'Created', type: 'boolean' },
      { key: 'lists[]name', label: 'Lists Name' },
      { key: 'lists[]status', label: 'Lists Status' },
      { key: 'tags[]', label: 'Tags' },
      { key: 'confirmationSent', label: 'Confirmation Email Sent', type: 'boolean' },
    ],
  },
};
