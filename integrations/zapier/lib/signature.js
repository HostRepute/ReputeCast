'use strict';

const crypto = require('crypto');

// Deliveries are signed as on ReputeCast's Webhooks page:
// X-ReputeCast-Signature is the hex HMAC-SHA256 of "{timestamp}.{raw body}"
// with the webhook's secret, and X-ReputeCast-Timestamp the Unix time.
// Anything older than 5 minutes is refused.

const MAX_AGE_SECONDS = 5 * 60;

/** A header from Zapier's raw request, whose names may be "Http-X-Reputecast-Signature". */
const headerOf = (headers, name) => {
  const want = name.toLowerCase();
  for (const [k, v] of Object.entries(headers || {})) {
    const key = k.toLowerCase().replace(/_/g, '-').replace(/^http-/, '');
    if (key === want) {
      return Array.isArray(v) ? v[0] : String(v);
    }
  }
  return '';
};

/**
 * Verifies a delivery. Returns { ok: true } or { ok: false, why }.
 * now is in seconds.
 */
const verify = ({ secret, body, signature, timestamp, now = Math.floor(Date.now() / 1000) }) => {
  if (!signature || !timestamp) {
    return { ok: false, why: 'the delivery isn’t signed' };
  }
  const ts = Number.parseInt(timestamp, 10);
  if (!Number.isFinite(ts) || Math.abs(now - ts) > MAX_AGE_SECONDS) {
    return { ok: false, why: 'the delivery is more than 5 minutes old' };
  }
  const expected = crypto.createHmac('sha256', secret).update(`${ts}.${body}`).digest('hex');
  const ok = String(signature)
    .split(',')
    .map((s) => s.trim())
    .some((s) => s.length === expected.length && crypto.timingSafeEqual(Buffer.from(s), Buffer.from(expected)));
  return ok ? { ok: true } : { ok: false, why: 'the signature doesn’t match this Zap’s webhook secret' };
};

/** Signs a body, as ReputeCast does (tests and the README's example). */
const sign = (secret, ts, body) => crypto.createHmac('sha256', secret).update(`${ts}.${body}`).digest('hex');

module.exports = { verify, sign, headerOf, MAX_AGE_SECONDS };
