# Changelog

## 1.1.1

- Security: refuse trigger deliveries when n8n has no raw request body or
  the workflow has no signing secret. Version 1.1.0 accepted these without
  verifying a signature. Reproduced with an unsigned multipart request in
  n8n 2.48.7; the fixed packed node returns 401 without starting a workflow.
- Security: reject malformed signature candidates before comparing bytes.
  A non-ASCII signature could throw and start an error workflow, or run
  downstream nodes with an empty item when Continue On Fail was enabled.
  A valid signature still works beside malformed candidates in either order.
- A workflow whose signing secret is missing after the upgrade refuses every
  delivery; deactivate and reactivate it to create a new managed webhook.

## 1.1.0

- ReputeCast Trigger: Contact Subscribed, Unsubscribed and Updated, Form
  Submitted (any form, or one), Campaign Sent and Transactional Message
  Bounced, through a managed webhook whose deliveries are signature-checked.
- ReputeCast: create or update a contact, add to a list, add a tag, record an
  event, send a transactional email with its variables mapped field by field.
- The ReputeCast API credential learns the key's workspace once.
- Consent kept as skipped items; ReputeCast's own words for every error.
- The workflow's name and ID in `X-ReputeCast-Workflow`.
