// Send throughput: transactional messages accepted by the send API, at a
// rising rate. Run it against a workspace with a published template
// (TEMPLATE, password-reset in the demo's acme-newsletter) and test
// recipients; the worker drains the queue through the workspace's route.
// Send with an API key whose rate is above the test's (TOKEN, from
// loadtest/sendkey.sh): a session shares 50 per second, and its 429s would
// measure that limit rather than the server.
import { check } from 'k6'
import { base, post } from './common.js'

const ws = __ENV.SEND_WORKSPACE || 'acme-newsletter'
const template = __ENV.TEMPLATE || 'password-reset'

export const options = {
  scenarios: {
    ramp: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      preAllocatedVUs: 50,
      stages: [
        { target: 50, duration: '30s' },
        { target: Number(__ENV.RATE || 200), duration: '1m' },
        { target: Number(__ENV.RATE || 200), duration: '1m' },
      ],
    },
  },
  thresholds: { 'http_req_duration{call:send}': ['p(95)<300'], http_req_failed: ['rate<0.01'] },
}

export default function () {
  const n = Math.floor(Math.random() * 1e9)
  const r = post(
    `${base}/api/v1/workspaces/${ws}/transactional/messages`,
    { template, to: { email: `load+${n}@example.com` }, data: { first_name: 'Sam', reset_url: 'https://example.com/reset', expires_in: '30 minutes' } },
    { call: 'send' },
  )
  check(r, { 'accepted 202': (x) => x.status === 202 })
}
