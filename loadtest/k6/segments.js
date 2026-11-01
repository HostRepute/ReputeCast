// Segment counts: the builder's live count as people edit conditions.
import { check } from 'k6'
import { api, post } from './common.js'

export const options = {
  scenarios: { count: { executor: 'constant-vus', vus: Number(__ENV.VUS || 4), duration: __ENV.DURATION || '1m' } },
  thresholds: { 'http_req_duration{call:segment}': ['p(95)<2000'], http_req_failed: ['rate<0.01'] },
}

const domains = ['gmail.com', 'outlook.com', 'yahoo.com', 'icloud.com']
const tiers = ['active', 'cooling', 'at_risk']

export default function () {
  const definition = {
    groups: [
      {
        match: 'all',
        conditions: [
          { field: 'email_domain', op: 'is', value: domains[Math.floor(Math.random() * domains.length)] },
          { field: 'tier', op: 'is', value: tiers[Math.floor(Math.random() * tiers.length)] },
        ],
      },
    ],
  }
  check(post(`${api}/segments/preview`, { mode: 'builder', definition }, { call: 'segment' }), { 'count 200': (r) => r.status === 200 })
}
