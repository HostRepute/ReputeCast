'use strict';

const { version } = require('../package.json');
const { outcomeOf } = require('./problems');
const { workflowHeader, zapOf } = require('./workflow');

/** How ReputeCast's call log names this app: "Zapier, ReputeCast app 1.2.0". */
const USER_AGENT = `ReputeCast-Zapier/${version}`;

/**
 * The instance's address as typed in the connection: a scheme is added when
 * missing, and a pasted API path or trailing slash is dropped.
 */
const instanceBase = (raw) => {
  let url = String(raw || '').trim();
  if (!url) {
    return '';
  }
  if (!/^https?:\/\//i.test(url)) {
    url = `https://${url}`;
  }
  return url.replace(/\/+$/, '').replace(/\/api\/v1$/i, '').replace(/\/+$/, '');
};

/** An API URL: path starts with "/", and {workspace} is the key's workspace. */
const apiUrl = (bundle, path) => {
  const ws = encodeURIComponent((bundle.authData && bundle.authData.workspace) || '');
  return `${instanceBase(bundle.authData && bundle.authData.instanceUrl)}/api/v1${path.replace('{workspace}', ws)}`;
};

const isInstanceCall = (url, bundle) => {
  const base = instanceBase(bundle.authData && bundle.authData.instanceUrl);
  return Boolean(base) && String(url || '').startsWith(`${base}/`);
};

// Sets a header, replacing any other spelling of its name (z.request has
// already merged in its own "user-agent: Zapier").
const setHeader = (headers, name, value) => {
  for (const k of Object.keys(headers)) {
    if (k.toLowerCase() === name.toLowerCase()) {
      delete headers[k];
    }
  }
  headers[name] = value;
};

/** beforeRequest: the key, the app's name, and the Zap behind the call. */
const addHeaders = (request, z, bundle) => {
  if (!isInstanceCall(request.url, bundle)) {
    return request;
  }
  request.headers = request.headers || {};
  setHeader(request.headers, 'authorization', `Bearer ${bundle.authData.apiKey}`);
  setHeader(request.headers, 'accept', 'application/json');
  // Still contains "Zapier", which Zapier's logging looks for.
  setHeader(request.headers, 'user-agent', USER_AGENT);
  const wf = workflowHeader(zapOf(bundle));
  if (wf) {
    setHeader(request.headers, 'x-reputecast-workflow', wf);
  }
  return request;
};

/**
 * afterResponse: turns ReputeCast's problems into what Zapier shows.
 * Consent kept (409) halts the run for that record without counting as an
 * error; a renamed workspace refreshes the connection; the rest shows the
 * problem's sentence. (Zapier turns 401 into a refresh of the connection,
 * and 429 into a retry, before this runs.)
 */
const handleErrors = (response, z, bundle) => {
  if (response.status < 400 || response.skipThrowForStatus || !isInstanceCall(response.request && response.request.url, bundle)) {
    return response;
  }
  const out = outcomeOf(response.status, response.content);
  switch (out.kind) {
    case 'skipped':
      throw new z.errors.HaltedError(out.message);
    case 'workspace':
    case 'reauth':
      throw new z.errors.RefreshAuthError(out.message);
    case 'throttled':
      throw new z.errors.ThrottledError(out.message);
    default:
      throw new z.errors.Error(out.message, out.code, response.status);
  }
};

/** Drops empty values: Zapier sends '' for fields left blank. */
const compact = (obj) => {
  const out = {};
  for (const [k, v] of Object.entries(obj)) {
    if (v === undefined || v === null || v === '') {
      continue;
    }
    if (Array.isArray(v)) {
      const items = v.map((x) => (typeof x === 'string' ? x.trim() : x)).filter((x) => x !== '' && x !== null && x !== undefined);
      if (items.length) {
        out[k] = items;
      }
      continue;
    }
    if (typeof v === 'object') {
      const inner = compact(v);
      if (Object.keys(inner).length) {
        out[k] = inner;
      }
      continue;
    }
    out[k] = typeof v === 'string' ? v.trim() : v;
  }
  return out;
};

/** A comma-separated or line-separated list typed in one field. */
const listOf = (v) => {
  if (Array.isArray(v)) {
    return v.flatMap(listOf);
  }
  return String(v ?? '')
    .split(/[,\n]/)
    .map((s) => s.trim())
    .filter(Boolean);
};

module.exports = { USER_AGENT, instanceBase, apiUrl, addHeaders, handleErrors, compact, listOf };
