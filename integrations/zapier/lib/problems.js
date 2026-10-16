'use strict';

// ReputeCast answers errors as RFC 9457 problems (application/problem+json):
// { type, title, status, detail, code, errors[], missing[], meta }. The
// detail is a sentence a person can act on; this module turns a problem
// into the message a Zap run shows, and says how Zapier should treat it.

const parse = (content) => {
  if (!content) {
    return {};
  }
  if (typeof content === 'object') {
    return content;
  }
  try {
    const v = JSON.parse(content);
    return v && typeof v === 'object' ? v : {};
  } catch {
    return {};
  }
};

const sentence = (s) => {
  const t = String(s || '').trim();
  if (!t) {
    return '';
  }
  return /[.!?…]$/.test(t) ? t : `${t}.`;
};

// Huma's own validation answers say "validation failed" and put the
// details in errors[]: "body.email: expected required property email to be
// present". ReputeCast's own checks put the same sentence in both.
const fieldErrors = (problem) => {
  const errors = Array.isArray(problem.errors) ? problem.errors : [];
  const detail = String(problem.detail || '');
  return errors
    .map((e) => {
      const msg = String((e && e.message) || '').trim();
      if (!msg || detail.includes(msg)) {
        return '';
      }
      const where = String((e && e.location) || '').replace(/^body\./, '');
      return where && where !== 'body' ? `${where}: ${msg}` : msg;
    })
    .filter(Boolean);
};

/**
 * The message for a problem answer: the detail, then field errors and the
 * missing template variables when the detail doesn't name them.
 */
const messageOf = (status, content) => {
  const p = parse(content);
  const parts = [];
  const detail = sentence(p.detail) || sentence(p.title) || `ReputeCast answered HTTP ${status}.`;
  parts.push(detail);
  const fields = fieldErrors(p);
  if (fields.length) {
    parts.push(sentence(fields.join('; ')));
  }
  const missing = Array.isArray(p.missing) ? p.missing : [];
  const unnamed = missing.filter((m) => !detail.includes(m));
  if (unnamed.length) {
    parts.push(`Missing variables: ${unnamed.join(', ')}.`);
  }
  return parts.join(' ');
};

// Answers that skip one record without breaking the workflow: an action
// never adds someone back to a list they left or adds a suppressed address
// (409), and transactional email isn't sent to a suppressed recipient (422).
const skips = new Set(['409 unsubscribed', '409 suppressed', '422 recipient_suppressed']);

/**
 * How an answer should end a step:
 * - ok: carry on
 * - skipped: consent was kept (409 unsubscribed or suppressed), or the
 *   transactional recipient is suppressed: that record is skipped, the
 *   workflow isn't broken
 * - reauth: the key is revoked, expired or unknown (401)
 * - workspace: the workspace's address changed (404 workspace_not_found);
 *   learn it again from the key
 * - throttled: too many calls (429)
 * - error: anything else, with the problem's sentence
 */
const outcomeOf = (status, content) => {
  if (status < 400) {
    return { kind: 'ok' };
  }
  const p = parse(content);
  const code = String(p.code || '');
  const message = messageOf(status, p);
  if (skips.has(`${status} ${code}`)) {
    return { kind: 'skipped', code, message, meta: p.meta || {} };
  }
  if (status === 401) {
    return { kind: 'reauth', code: code || 'unauthenticated', message };
  }
  if (status === 404 && code === 'workspace_not_found') {
    return { kind: 'workspace', code, message };
  }
  if (status === 429) {
    return { kind: 'throttled', code: code || 'rate_limited', message };
  }
  return { kind: 'error', code: code || `http_${status}`, message };
};

module.exports = { messageOf, outcomeOf, parse };
