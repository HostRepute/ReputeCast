import { defineMessages } from '@/i18n'

/** The Setup tab's table of the API's errors: when each one answers and what to do. */
export const me = defineMessages('transactional.errors', {
  template_not_found_when: "The ID doesn't exist in this workspace",
  template_not_found_do: 'Check the ID on Templates',
  template_not_published_when: 'The template has a draft but no live version',
  template_not_published_do: 'Publish it first',
  missing_variables_when: "A required variable isn't in {data}",
  missing_variables_do: 'The answer lists which ones',
  invalid_recipient_when: "The address isn't a valid email address",
  invalid_recipient_do: 'Validate it in your app',
  recipient_suppressed_when: 'The address bounced, complained or is a spam trap',
  recipient_suppressed_do: 'The answer gives the reason and date',
  sender_not_verified_when: "{from} isn't a verified Transactional sender",
  sender_not_verified_do: 'Verify it in Settings › Senders',
  attachments_too_large_when: "Over the route's limit",
  attachments_too_large_do: '{size} on {provider}',
  rate_limited_when: "Over the key's rate",
  rate_limited_do: 'Wait for {header}',
  no_healthy_route_when: 'Every provider in the route is down',
  no_healthy_route_do: 'Retry with the same idempotency key',
  idempotency_key_reused_when: 'The same key came with a different request within 24 hours',
  idempotency_key_reused_do: 'Use a new key for a new message',
  template_not_allowed_when: "The key can't send this template or category",
  template_not_allowed_do: "Use a key that can, or widen the key's scope",
})
