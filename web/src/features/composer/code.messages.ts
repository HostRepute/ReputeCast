import { defineMessages } from '@/i18n'

/**
 * The code view (ComposerCode): MJML beside the compiled HTML, the compile
 * results and the personalization tokens. {tag} is an MJML tag as written
 * in the code (<mj-body>), kept as it is.
 */
export const mk = defineMessages('composer.code', {
  codeView: 'Code view',
  importMjml: 'Import MJML',
  format: 'Format',
  copyHtml: 'Copy compiled HTML',
  copied: 'Compiled HTML copied',
  copyFailed: "Couldn't copy",
  clipboardBlocked: 'Your browser blocked the clipboard.',
  engineNote: 'mjml 4 · grapesjs-mjml 1.0 · changes here update the canvas and the layer tree',
  applyToCanvas: 'Apply to canvas',
  noteEdited: 'edited the MJML',

  mjmlError: 'MJML error',
  lineError: 'Line {line}: {message}',
  keepsLastGood: 'The canvas keeps the last good version. Fix the code view or revert the last change.',
  goToLine: 'Go to line {line}',
  undoMine: 'Undo my changes',
  needsBody: 'the document needs an {tag}',
  unclosed: 'unclosed {tag}',
  noOpening: '{tag} has no opening tag',

  mjmlPane: 'MJML · source of truth',
  mjmlAside: '{lines} lines · line {line}, col {col}',
  mjmlSource: 'MJML source',
  htmlPane: 'Compiled HTML · read only',
  htmlAside: '{size} · tables and inline CSS for Outlook',
  compiledHtml: 'Compiled HTML',

  panelLabel: 'Compile results and tokens',
  codeTools: 'Code tools',
  tabCompile: 'Compile',
  tabTokens: 'Tokens',
  ok: 'OK',
  error: 'Error',
  warn: 'Warn',
  info: 'Info',
  compiledIn: 'Compiled in {ms} ms · {errors, plural, one {# error} other {# errors}} · {warnings, plural, one {# warning} other {# warnings}}',
  compiling: 'Compiling…',
  markerNote: 'Comments like <mono>rc:footer</mono> mark ReputeCast blocks so they stay editable as one unit on the canvas.',
  trackingNote: "Tracking links and the open pixel are added at send time, per sending provider, so they don't appear here.",
  tokensAvailable: 'Personalization tokens available',
  fallbacksNote: 'Fallbacks after the pipe are used when the field is empty.',
  groupContact: 'Contact fields',
  groupCustom: 'Custom fields',
  groupReferral: 'Referral',
  groupSystem: 'Links and system',
  copiedToken: 'Copied {token}',
})
