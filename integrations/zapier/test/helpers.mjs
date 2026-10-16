// Test helpers: the app under Zapier's app tester, and a mock ReputeCast
// whose every call and answer is checked against web/openapi.json.

import { createRequire } from 'node:module';
import assert from 'node:assert/strict';
import { checkRequest, checkResponse } from '../../contract/openapi-contract.mjs';

// zapier-platform-core reads this when it loads: under test it doesn't log
// each run's memory use.
process.env.NODE_ENV = 'test';

export const require = createRequire(import.meta.url);
export const zapier = require('zapier-platform-core');
export const App = require('../index.js');
export const nock = require('nock');
export const appTester = zapier.createAppTester(App);

export const BASE = 'https://reputecast.acme.io';
export const KEY = `rc_live_5d1e${'0'.repeat(56)}7b20`;
export const authData = { instanceUrl: BASE, apiKey: KEY, workspace: 'acme-newsletter' };
export const WS = '/api/v1/workspaces/acme-newsletter';

// Fields another workstream adds to the API in this release (X05 adds
// `language` to the transactional send API); the contract accepts them and
// says so.
export const pending = { 'send-transactional-message': ['language'] };

/**
 * A mock ReputeCast. expect() queues one answer; done() checks that every
 * queued call happened and that requests and answers fit the OpenAPI
 * document.
 */
export function mockServer() {
  const calls = [];
  const problems = [];
  const scope = nock(BASE);
  const expect = (method, path, { status = 200, body, operation, headers = {} } = {}) => {
    if (operation) {
      const r = checkResponse(operation, status, body);
      if (!r.ok) {
        problems.push(`mocked ${status} answer of ${operation}: ${r.errors.join('; ')}`);
      }
    }
    scope.intercept(path, method).reply(function reply(uri, reqBody) {
      calls.push({ method, url: BASE + uri, body: reqBody === '' ? undefined : reqBody, headers: this.req.headers });
      const type = status >= 400 ? 'application/problem+json' : 'application/json';
      return [status, body === undefined ? '' : JSON.stringify(body), { 'Content-Type': type, ...headers }];
    });
  };
  const done = () => {
    scope.done();
    for (const c of calls) {
      const r = checkRequest(c, { pending });
      if (!r.ok) {
        problems.push(`${c.method} ${c.url}: ${r.errors.join('; ')}`);
      }
    }
    assert.deepEqual(problems, []);
  };
  return { expect, calls, done };
}

/** A problem answer, as ReputeCast writes them. */
export const problem = (status, code, detail, extra = {}) => ({
  title: { 401: 'Unauthorized', 403: 'Forbidden', 404: 'Not Found', 409: 'Conflict', 422: 'Unprocessable Entity', 500: 'Internal Server Error' }[status] || 'Error',
  status,
  detail,
  code,
  ...extra,
});

export const setup = () => {
  nock.disableNetConnect();
};

export const teardown = () => {
  nock.cleanAll();
  nock.enableNetConnect();
};
