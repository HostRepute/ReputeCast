// Package digests runs recurring and RSS digests (design expansion, phase
// 3, M20): a campaign that builds an issue on a schedule from a feed's new
// items and sends it.
//
// A digest reads a feed (RSS 2.0, Atom 1.0, JSON Feed 1.1, or a WordPress
// site of the integrations hub) every 30 minutes while it is on and again
// at preparation, through the SSRF guard (package feed). Its template is a
// campaign the composer edits in digest mode, with the Feed items block and
// feed tokens. Every issue is a campaign too: it is built at preparation,
// runs Send Readiness and the approval rules, and goes out through the
// sending engine on its owner's authority, so there is no second sending
// pipeline. A held issue waits for a person; nothing is sent anyway.
package digests
