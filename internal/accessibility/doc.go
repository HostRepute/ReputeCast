// Package accessibility checks emails for the accessibility failures a
// machine can find (design expansion, M19): images without alt text, links
// and buttons with no text, text contrast below WCAG 2.2 AA against the
// nearest background, link text that says nothing, skipped heading levels
// and text below the workspace's minimum size.
//
// The checks run on the compiled HTML of every language version, the same
// implementation for the composer (as you edit), Send Readiness, publishing
// an automation or a transactional template, and the template gallery. An
// issue found in several versions is listed once with its languages, and
// points at its MJML element so the composer can show it and fix it.
//
// What a machine can add safely is added when HTML is stored (Finalize):
// lang and dir from the version's language, and role="presentation" on
// layout tables MJML missed. The workspace policy decides whether serious
// issues only warn (the default) or block sends and publishing.
package accessibility
