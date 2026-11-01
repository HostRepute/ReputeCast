// The contacts list: its first page, paging on with cursors, and search.
// Targets from the review (AU-1, AU-2): pages under 200 ms, search under
// 150 ms at the 95th percentile.
import { check } from 'k6'
import { api, get } from './common.js'

export const options = {
  scenarios: {
    browse: { executor: 'constant-vus', vus: Number(__ENV.VUS || 10), duration: __ENV.DURATION || '1m' },
  },
  thresholds: {
    'http_req_duration{call:page}': ['p(95)<200'],
    'http_req_duration{call:search}': ['p(95)<150'],
    http_req_failed: ['rate<0.01'],
  },
}

export default function () {
  let cursor = ''
  for (let i = 0; i < 5; i++) {
    const r = get(`${api}/contacts?pageSize=25${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`, { call: 'page' })
    check(r, { 'page 200': (x) => x.status === 200 })
    cursor = r.json('nextCursor') || ''
    if (!cursor) break
  }
  const n = Math.floor(Math.random() * 999999) + 1
  check(get(`${api}/contacts?pageSize=25&q=load${n}`, { call: 'search' }), { 'search 200': (x) => x.status === 200 })
  check(get(`${api}/contacts?pageSize=25&q=${encodeURIComponent(`load${n}@gmail.com`)}`, { call: 'search' }), { 'exact 200': (x) => x.status === 200 })
}
