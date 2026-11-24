# Licensing

ReputeCast is free and open-source software, and every feature in this
repository is free. The server is licensed under the
[GNU Affero General Public License v3.0 only](LICENSE) (`AGPL-3.0-only`). The
code that runs inside your programs and on your website is
[MIT](LICENSES/MIT.txt), so it adds no obligations to them. Code and content
that are made to be copied are [MIT-0](LICENSES/MIT-0.txt), so you don't even
need to keep a notice.

[`REUSE.toml`](REUSE.toml) says the same per file, in the
[REUSE](https://reuse.software/spec-3.3/) format, and CI checks it with
`reuse lint`. The license texts are in [`LICENSES/`](LICENSES/).

## Which license covers what

| Part | Where | License |
|---|---|---|
| The server, the web app, the CLI, the built-in sending adapters, the migrations, the design and the docs: everything not listed below | the rest of the repository | `AGPL-3.0-only` |
| The client SDKs (TypeScript, Python, Go) and the public OpenAPI documents | `sdk/client-ts`, `sdk/client-python`, `sdk/client-go`, `sdk/openapi` | `MIT` |
| The adapter SDK, for community sending adapters | `sdk/adapter-sdk` | `MIT` |
| The scripts and styles that sites embed: the signup form, the site tracker, the push prompt and its service worker | `internal/forms/embed`, `internal/tracker/static`, `internal/push/static` | `MIT` |
| The integration apps for Zapier, Make and n8n | `integrations/` | `MIT` |
| The WordPress plugin, as WordPress.org asks for a GPL-compatible license | `integrations/wordpress` | `GPL-2.0-or-later` |
| The adapter template and the starter email templates, made to be copied | `sdk/adapter-template`, `internal/templates/starters` | `MIT-0` |
| The ReputeCast name and logos | `brand/` | [`LicenseRef-ReputeCast-Trademarks`](LICENSES/LicenseRef-ReputeCast-Trademarks.txt), see [TRADEMARKS.md](TRADEMARKS.md) |
| The IBM Plex fonts | `internal/countdown/fonts`, `internal/publicpages/static/fonts` | `OFL-1.1`, by IBM |

The rest of `sdk/` (its scripts and test data) is MIT too. A new file takes
the license of the directory it's in. Each published package carries its own
`LICENSE` file, which is what npm, PyPI and the Go module proxy read.

As section 7(e) of the AGPL allows, none of these licenses grants rights to
the ReputeCast name or logos. [TRADEMARKS.md](TRADEMARKS.md) says how you may
use them.

## What this means for you

- **Running ReputeCast for your own organisation:** go ahead. The AGPL asks
  nothing more of you unless you distribute ReputeCast or let others use a
  version you changed.
- **Hosting ReputeCast for others, or selling hosting:** you may. If you
  changed the code, section 13 of the AGPL requires you to offer the users of
  your instance the source of your version: set `REPUTECAST_SOURCE_URL` to it
  and every page links it ([configuration](docs/deploy/configuration.md)).
  The name rules are in [TRADEMARKS.md](TRADEMARKS.md).
- **Building an app with the SDKs or the API, including a closed-source
  one:** you may. The SDKs are MIT, and your app talks to ReputeCast over
  HTTP, so the AGPL doesn't reach it. Keep the SDK's MIT notice.
- **Writing a sending adapter for your provider, including a proprietary
  one:** you may. An adapter is its own process that speaks the
  [adapter protocol](docs/adapters.md#community-adapters). The adapter SDK is
  MIT and the template MIT-0, so your adapter can take any license.
- **Embedding a form, the tracker or the push prompt on your site:** you may.
  The scripts are MIT, and the notice at the top of each one is all they
  need. The form script keeps that notice when ReputeCast serves it.
- **Using a starter template in your newsletters:** you may, with no notice.
- **Forking ReputeCast:** you may, under the AGPL. Give your fork its own name
  and logo ([TRADEMARKS.md](TRADEMARKS.md)).

## Our commitments

1. **Every feature in this repository is free** and stays under an
   OSI-approved license. There's no open core, no paid edition, no license
   key and no seat limit.
2. **ReputeCast Cloud runs the public image, unmodified,** so it has no
   ReputeCast feature that you can't run yourself. Its control plane
   (sign-up, billing, provisioning) is a separate service that operates
   ReputeCast from the outside and doesn't change its code.
3. **Security fixes are released publicly no later than they reach
   ReputeCast Cloud** ([SECURITY.md](SECURITY.md)).
4. **Contributions stay under the license they were made under.**
   Contributors sign off their commits with the
   [Developer Certificate of Origin](CONTRIBUTING.md#developer-certificate-of-origin).
   We use no contributor license agreement and ask for no copyright
   assignment, so nobody can relicense a contribution without its author.

## History

- **Up to v1.8.3** everything except the n8n nodes (MIT) and the WordPress
  plugin (GPL-2.0-or-later) was `AGPL-3.0-only`, including the client SDK
  packages 1.8.1 to 1.8.3. Those releases keep that license.
- **From the first release after v1.8.3** the table above applies. HostRepute
  wrote all of the code that changed license.
