'use strict';

// The six triggers of brief 3.2 and docs/integrations/hub.md, delivered by
// managed webhooks. A delivery's body is the Webhooks page's envelope,
// { id, type, created_at, workspace, data }; the triggers put data's fields
// at the top, next to the event's id, type and time.

const contactFields = [
  { key: 'email', label: 'Email' },
  { key: 'first_name', label: 'First Name' },
  { key: 'last_name', label: 'Last Name' },
  { key: 'company', label: 'Company' },
  { key: 'country', label: 'Country' },
  { key: 'timezone', label: 'Timezone' },
  { key: 'language', label: 'Language' },
  { key: 'tags[]', label: 'Tags' },
  { key: 'status', label: 'Contact Status' },
  { key: 'contact_id', label: 'Contact ID' },
  { key: 'lists[]id', label: 'Lists ID' },
  { key: 'lists[]name', label: 'Lists Name' },
  { key: 'lists[]status', label: 'Lists Status' },
];

const eventFields = [
  { key: 'id', label: 'Event ID' },
  { key: 'event', label: 'Event' },
  { key: 'created_at', label: 'Event Time', type: 'datetime' },
  { key: 'workspace', label: 'Workspace' },
];

const contact = {
  contact_id: '0192a4c1-7d4e-7c1e-9a40-3f1b2c8d5e01',
  email: 'lucia.fernandez@example.com',
  first_name: 'Lucía',
  last_name: 'Fernández',
  company: '',
  country: 'MX',
  timezone: 'America/Mexico_City',
  language: 'es-MX',
  tags: ['beta-applicant'],
  fields: { plan: 'Team' },
  status: 'active',
  lists: [{ id: '0192a4c1-0000-7000-8000-00000000b3a1', name: 'Weekly Digest', status: 'subscribed' }],
};

const envelope = (type, at) => ({ id: 'evt_sample', event: type, created_at: at, workspace: 'acme-newsletter' });

const EVENTS = [
  {
    key: 'contact_subscribed',
    event: 'contact.subscribed',
    noun: 'Contact',
    label: 'Contact Subscribed',
    description: 'Triggers when a contact subscribes to a list: through a form, an import, the API or the app.',
    outputFields: [
      ...contactFields,
      { key: 'list_id', label: 'List ID' },
      { key: 'list_name', label: 'List Name' },
      { key: 'source', label: 'Source' },
      ...eventFields,
    ],
    sample: {
      ...contact,
      list_id: '0192a4c1-0000-7000-8000-00000000b3a1',
      list_name: 'Weekly Digest',
      source: 'form',
      ...envelope('contact.subscribed', '2026-10-15T14:02:11Z'),
    },
  },
  {
    key: 'contact_unsubscribed',
    event: 'contact.unsubscribed',
    noun: 'Contact',
    label: 'Contact Unsubscribed',
    description: 'Triggers when a contact unsubscribes from a list.',
    outputFields: [
      ...contactFields,
      { key: 'list_id', label: 'List ID' },
      { key: 'list_name', label: 'List Name' },
      { key: 'source', label: 'Source' },
      ...eventFields,
    ],
    sample: {
      ...contact,
      lists: [{ id: '0192a4c1-0000-7000-8000-00000000b3a1', name: 'Weekly Digest', status: 'unsubscribed' }],
      list_id: '0192a4c1-0000-7000-8000-00000000b3a1',
      list_name: 'Weekly Digest',
      source: 'preference_center',
      ...envelope('contact.unsubscribed', '2026-10-15T14:05:40Z'),
    },
  },
  {
    key: 'contact_updated',
    event: 'contact.updated',
    noun: 'Contact',
    label: 'Contact Updated',
    description: "Triggers when a contact's names, fields, language, tags or lists change.",
    outputFields: [...contactFields, { key: 'changed[]', label: 'Changed' }, ...eventFields],
    sample: { ...contact, changed: ['language', 'tags'], ...envelope('contact.updated', '2026-10-15T13:48:02Z') },
  },
  {
    key: 'form_submitted',
    event: 'form.submitted',
    noun: 'Submission',
    label: 'Form Submitted',
    description: 'Triggers when a signup through a form is confirmed: at once for single opt-in, after the confirmation click for double opt-in.',
    form: true,
    outputFields: [
      { key: 'form_id', label: 'Form ID' },
      { key: 'form_name', label: 'Form Name' },
      { key: 'email', label: 'Email' },
      { key: 'contact_id', label: 'Contact ID' },
      { key: 'answers__first_name', label: 'Answers First Name' },
      { key: 'lists[]', label: 'Lists' },
      { key: 'consent_text', label: 'Consent Text' },
      { key: 'language', label: 'Language' },
      { key: 'double_opt_in', label: 'Double Opt-In', type: 'boolean' },
      { key: 'submitted_at', label: 'Submitted At', type: 'datetime' },
      { key: 'confirmed_at', label: 'Confirmed At', type: 'datetime' },
      ...eventFields,
    ],
    sample: {
      form_id: 'frm_8c2e1a',
      form_name: 'Newsletter signup',
      contact_id: contact.contact_id,
      email: contact.email,
      answers: { email: contact.email, first_name: 'Lucía' },
      lists: ['Weekly Digest'],
      consent_text: 'I agree to receive emails from Acme. Unsubscribe any time with one click.',
      language: 'es',
      double_opt_in: true,
      submitted_at: '2026-10-03T14:02:11Z',
      confirmed_at: '2026-10-03T14:05:40Z',
      ...envelope('form.submitted', '2026-10-03T14:05:40Z'),
    },
  },
  {
    key: 'campaign_sent',
    event: 'campaign.completed',
    noun: 'Campaign',
    label: 'Campaign Sent',
    description: 'Triggers when a campaign finishes sending, with its totals.',
    outputFields: [
      { key: 'campaign_id', label: 'Campaign ID' },
      { key: 'name', label: 'Campaign Name' },
      { key: 'subject', label: 'Subject' },
      { key: 'status', label: 'Status' },
      { key: 'sent', label: 'Sent', type: 'integer' },
      { key: 'delivered', label: 'Delivered', type: 'integer' },
      { key: 'bounced', label: 'Bounced', type: 'integer' },
      { key: 'complained', label: 'Complained', type: 'integer' },
      { key: 'started_at', label: 'Started At', type: 'datetime' },
      { key: 'completed_at', label: 'Completed At', type: 'datetime' },
      ...eventFields,
    ],
    sample: {
      campaign_id: '0192a4c1-0000-7000-8000-0000000c0142',
      name: 'Weekly Digest #142',
      subject: 'Three launches, one deprecation, and the roadmap for Q4',
      status: 'sent',
      sent: 31604,
      delivered: 31352,
      bounced: 189,
      complained: 4,
      started_at: '2026-10-14T13:00:00Z',
      completed_at: '2026-10-14T13:41:00Z',
      ...envelope('campaign.completed', '2026-10-14T13:41:00Z'),
    },
  },
  {
    key: 'transactional_bounced',
    event: 'message.bounced',
    noun: 'Message',
    label: 'Transactional Message Bounced',
    description: 'Triggers when a transactional email bounces.',
    outputFields: [
      { key: 'message_id', label: 'Message ID' },
      { key: 'template', label: 'Template' },
      { key: 'template_version', label: 'Template Version', type: 'integer' },
      { key: 'category', label: 'Category' },
      { key: 'to', label: 'Recipient' },
      { key: 'status', label: 'Status' },
      { key: 'reason', label: 'Bounce Reason' },
      { key: 'provider', label: 'Sending Provider' },
      { key: 'tags[]', label: 'Tags' },
      { key: 'at', label: 'Bounced At', type: 'datetime' },
      ...eventFields,
    ],
    sample: {
      message_id: '0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44',
      template: 'password-reset',
      template_version: 3,
      category: 'Account',
      to: 'sam.rivera@outlook.com',
      status: 'bounced',
      reason: '550 5.1.1 The email account that you tried to reach does not exist',
      provider: 'postmark',
      tags: ['auth'],
      metadata: { user_id: 'u_8812' },
      at: '2026-10-15T10:42:00Z',
      ...envelope('message.bounced', '2026-10-15T10:42:00Z'),
    },
  },
];

/** A delivery as a trigger result: data's fields, then the event's own. */
const flatten = (env) => {
  const data = env && env.data && typeof env.data === 'object' && !Array.isArray(env.data) ? env.data : {};
  return { ...data, id: env.id, event: env.type, created_at: env.created_at, workspace: env.workspace };
};

module.exports = { EVENTS, flatten };
