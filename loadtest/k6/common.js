// Shared settings of the load test scripts (review 2026-11 OP-9).
//
//   BASE_URL   the instance, e.g. http://localhost:8080
//   COOKIE     a session cookie (rc_session=…) of the workspace's owner;
//              loadtest/session.sh prints one for the seeded instance
//   WORKSPACE  the workspace slug (load-test)
//   TOKEN      an API key, sent as a bearer token in place of the cookie
//              (send.js: loadtest/sendkey.sh prints a send-only one)
import http from 'k6/http'

export const base = (__ENV.BASE_URL || 'http://localhost:8080').replace(/\/$/, '')
export const workspace = __ENV.WORKSPACE || 'load-test'
export const api = `${base}/api/v1/workspaces/${workspace}`

const headers = __ENV.TOKEN
  ? { Authorization: `Bearer ${__ENV.TOKEN}`, 'Content-Type': 'application/json' }
  : { Cookie: __ENV.COOKIE || '', 'Content-Type': 'application/json' }

export function get(url, tags) {
  return http.get(url, { headers, tags })
}

export function post(url, body, tags) {
  return http.post(url, JSON.stringify(body), { headers, tags })
}
