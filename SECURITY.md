# Security policy

## Supported versions

Security fixes go into the latest release. Run the newest version of
ReputeCast to get them; [upgrading](docs/deploy/README.md#upgrades) takes one
command.

## Reporting a vulnerability

Please don't open a public issue, discussion or pull request for a security
problem.

Report it privately through GitHub instead:
[report a vulnerability](https://github.com/hostrepute/reputecast/security/advisories/new)
(the repository's **Security** tab › **Report a vulnerability**). Only the
maintainers can see the report.

Please include:
- the version or commit you tested;
- how to reproduce it, or a proof of concept;
- what an attacker could do with it.

We'll confirm we received the report, keep you updated while we fix it, and
publish a security advisory with the release that fixes it. Tell us if you'd
like to be credited.

Please give us a chance to release a fix before you disclose the problem
publicly, and don't test against instances you don't own.

## How ReputeCast protects data

[docs/architecture.md](docs/architecture.md#security) describes the security
model: password hashing, sessions and API keys, two-factor, encryption of
secrets at rest, the read-only SQL segment role, signed webhooks and the audit
log.
