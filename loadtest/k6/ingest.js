// Webhook ingest: provider events posted to a provider's webhook URL.
// ReputeCast verifies the signature, stores the raw event and answers at
// once; normalizing happens in a job (watch reputecast_ingest_lag_seconds).
//
//   HOOK_URL   the provider's webhook URL from Sending providers
//   BODY_FILE  a captured, validly signed payload of that provider
//   HEADERS    its headers as JSON, e.g. {"Authorization": "Basic …"}
import http from 'k6/http'
import { check } from 'k6'

const body = open(__ENV.BODY_FILE)
const headers = Object.assign({ 'Content-Type': 'application/json' }, JSON.parse(__ENV.HEADERS || '{}'))

export const options = {
  scenarios: { webhooks: { executor: 'constant-arrival-rate', rate: Number(__ENV.RATE || 300), timeUnit: '1s', duration: __ENV.DURATION || '1m', preAllocatedVUs: 50 } },
  thresholds: { 'http_req_duration{call:ingest}': ['p(95)<100'], http_req_failed: ['rate<0.01'] },
}

export default function () {
  check(http.post(__ENV.HOOK_URL, body, { headers, tags: { call: 'ingest' } }), { 'stored 200': (r) => r.status === 200 })
}
