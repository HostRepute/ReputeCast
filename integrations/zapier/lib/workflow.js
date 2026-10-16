'use strict';

// X-ReputeCast-Workflow names the Zap behind each call, so ReputeCast can
// list "Zaps using ReputeCast" with their calls and errors
// (docs/integrations/hub.md › Workflow names).
//
// Zapier gives apps the Zap's ID but not its name, so every step has an
// optional "Zap name" field; without it ReputeCast lists the calls as
// Unnamed Zap.

const MAX_NAME = 200;
const MAX_ID = 120;

// Header values are ASCII. Characters outside printable ASCII, and the
// four the header uses itself, are percent-encoded (UTF-8); the rest stay
// as typed so the header reads naturally.
const encodeName = (name) =>
  Array.from(name)
    .map((ch) => {
      const c = ch.codePointAt(0);
      if (c >= 0x20 && c <= 0x7e && !'%;="'.includes(ch)) {
        return ch;
      }
      try {
        return encodeURIComponent(ch);
      } catch {
        return ''; // a lone surrogate: not text
      }
    })
    .join('');

const cleanName = (name) =>
  Array.from(String(name ?? '').replace(/\s+/g, ' ').trim())
    .slice(0, MAX_NAME)
    .join('');

const cleanId = (id) =>
  String(id ?? '')
    .replace(/[^A-Za-z0-9._:-]/g, '')
    .slice(0, MAX_ID);

/**
 * The header value: "<name>; id=<id>", either part left out when it's
 * missing. Returns '' when there is neither.
 */
const workflowHeader = ({ name, id } = {}) => {
  const parts = [];
  const n = cleanName(name);
  if (n) {
    parts.push(encodeName(n));
  }
  const i = cleanId(id);
  if (i) {
    parts.push(`id=${i}`);
  }
  return parts.join('; ');
};

/** The Zap behind a call: its ID from Zapier, its name from the step. */
const zapOf = (bundle) => ({
  id: bundle && bundle.meta && bundle.meta.zap ? bundle.meta.zap.id : undefined,
  name: bundle && bundle.inputData ? bundle.inputData.zap_name : undefined,
});

module.exports = { workflowHeader, encodeName, zapOf };
