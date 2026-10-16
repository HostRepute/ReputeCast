'use strict';

const { apiUrl, compact } = require('../lib/client');
const { emailField, zapNameField } = require('../lib/fields');

// Record Event: an API event for a contact, with properties. Automations
// that start with that event (or have it as their goal) run for them.

const perform = async (z, bundle) => {
  const d = bundle.inputData;
  const body = compact({ event: d.event, email: d.email });
  if (d.properties && typeof d.properties === 'object' && Object.keys(d.properties).length) {
    body.properties = d.properties;
  }
  const response = await z.request({ url: apiUrl(bundle, '/workspaces/{workspace}/automations/events'), method: 'POST', body });
  return { id: response.data.contactId, event: body.event, ...response.data };
};

module.exports = {
  key: 'record_event',
  noun: 'Event',
  display: {
    label: 'Record Event',
    description: 'Records an event for a contact, which starts the automations that listen for it.',
  },
  operation: {
    inputFields: [
      {
        key: 'event',
        label: 'Event',
        type: 'string',
        required: true,
        helpText: "The event's name, such as `workspace.created`. Automations that start with this API event, or have it as their goal, run for the contact.",
      },
      emailField('The contact the event is about. They must be in ReputeCast already: add them with Create or Update Contact first.'),
      { key: 'properties', label: 'Properties', dict: true, required: false, helpText: 'Details of the event, such as `plan: Team`. Automations can use them in conditions and emails.' },
      zapNameField,
    ],
    perform,
    sample: { id: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e03', event: 'workspace.created', accepted: true, contactId: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e03', flows: 1 },
    outputFields: [
      { key: 'id', label: 'Contact ID' },
      { key: 'event', label: 'Event' },
      { key: 'accepted', label: 'Accepted', type: 'boolean' },
      { key: 'flows', label: 'Automations Started', type: 'integer' },
    ],
  },
};
