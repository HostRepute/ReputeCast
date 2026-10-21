# ReputeCast for WordPress

The "ReputeCast" WordPress plugin (1.4.0): ReputeCast's signup forms as a
block and a shortcode, published posts sent to the site's post-draft rule, the
site's report-in and the status endpoint ReputeCast's "Check now" calls. With
WPML or Polylang, a post's translations become the draft's language versions,
and the block has a language setting. With WooCommerce, the checkout gets the
marketing consent checkbox, orders keep the email's click token, and
abandoned checkouts are reported to the store ReputeCast connects.

It uses only ReputeCast's public API, with the site's own key (one key per
site, made in ReputeCast). The API it calls is described in
[docs/integrations/hub.md › WordPress sites](../../docs/integrations/hub.md#wordpress-sites).

- PHP 8.1 or later, WordPress 6.4 or later. No build step: the block's editor
  script is plain JavaScript on WordPress's own packages.
- Licensed GPL-2.0-or-later, as the WordPress plugin directory asks.
  ReputeCast itself is AGPL-3.0.

```
integrations/wordpress/
  reputecast/          the plugin, as it ships (the zip holds this folder)
    reputecast.php     plugin header and bootstrap
    includes/          the classes: options, API client, forms, posts, report-in, status, settings page, WooCommerce
    blocks/signup-form block.json, the editor script, its styles and render.php
    languages/         reputecast.pot
    readme.txt         the plugin directory's page
    uninstall.php
  tests/               PHPUnit + Brain Monkey, checked against web/openapi.json
  scripts/             lint.php (parses every file) and build.php (the zip)
```

## Install

From the WordPress plugin directory, once it's published: Plugins › Add New,
search for "ReputeCast", Install, Activate.

From this repository: `composer build` makes `dist/reputecast-1.4.0.zip`.
Upload it in Plugins › Add New › Upload Plugin, then Activate.

## Configure

1. In ReputeCast, open Settings › Integrations › WordPress and choose
   **Connect a site**. Pick what the site may do (list forms, create drafts)
   and copy the key: it's shown once.
2. In WordPress, open **Settings › ReputeCast**. Paste the ReputeCast address
   (the one you sign in at, for example `https://news.acme.io`) and the key,
   then press **Connect**. The plugin checks the key (`GET /integration`),
   reports in and fetches the forms. ReputeCast lists the site right away.
3. Add a form: insert the **ReputeCast signup form** block and pick a form, or
   paste a shortcode like `[reputecast_form id="frm_8c2e1a"]` (Settings ›
   ReputeCast lists each form's).
4. Post drafts: set the site's post-draft rule in ReputeCast (categories,
   audience, template, sender, who's notified). Every post published from then
   on is sent; the rule decides.

What the plugin does, in short:

| | |
|---|---|
| Forms | An empty container (`<div data-reputecast-form="frm_…">`) and the form's public script, as ReputeCast's embed code does. A popup form loads its script only. The plugin keeps the forms list, so forms keep working after Disconnect. A page view never calls ReputeCast. |
| Allowed domains | The block warns when the site isn't one of the form's allowed domains, when a form is a draft or paused, and when a form is gone. |
| Language | The container gets `lang`: the block's setting, else the page's language from WPML or Polylang. Without either, the page's own `lang` applies. Forms that match the page read it. |
| Posts | Publishing a post (not an update, not a page, not a password-protected post) schedules a background job, so publishing never waits for ReputeCast. The job sends the post (`POST …/wordpress/drafts`) with its published translations, keeps the answer (Settings › ReputeCast lists the last five) and reports in. |
| Translations | The post in the site's default language leads, with its published translations; a translation published later sends the group again, and joins the draft. |
| Report-in | On Connect, every 12 hours (WP-Cron, `twicedaily`) and with every post: plugin, WordPress and PHP versions, the feed URL, the categories in the default language, and where forms are used (blocks, shortcodes and widgets in published posts, pages, templates and widget areas). |
| Check now | `GET /wp-json/reputecast/v1/status` answers 200 with the versions only to calls signed with the site's key (HMAC-SHA256 of `{timestamp}.status` keyed with the key's SHA-256, within 5 minutes, any of the comma-separated signatures), and 401 otherwise. |
| The key | Kept in its own option, not autoloaded, encrypted (libsodium secretbox) with a key derived from the site's salts in `wp-config.php`: a copy of the database alone doesn't reveal it. It's never shown again; Settings shows `rc_live_5d1e…7b20`. |
| WooCommerce consent | With WooCommerce active, an unchecked checkbox under the checkout's email (classic checkout: `woocommerce_after_checkout_billing_form`; block checkout: an additional `contact` field), with exactly "Email me about new products and offers from the store. Unsubscribe any time." The order keeps `_reputecast_marketing` (`yes`/`no`) and, when ticked, `_reputecast_consent_text` and `_reputecast_consent_at`. The filter `reputecast_woocommerce_consent` turns it off. |
| Click token | A visit from an email link (`?rc=…`) keeps the token in a first-party cookie (`reputecast_click`, 30 days, HttpOnly); the order keeps it as `_reputecast_click`, which ReputeCast reads through the store's REST API key to credit the order. |
| Abandoned checkouts | Checkouts with a cart are kept (the newest 500, a week at most) under a token in the shopper's WooCommerce session; every 15 minutes (WP-Cron) the changed ones go to `POST …/woocommerce/checkouts` with the email typed, the consent, the cart, totals and a signed link (`?reputecast_cart=…&k=…`) that restores the cart. An order marks its checkout completed (`_reputecast_checkout`). The filter `reputecast_woocommerce_checkouts` turns it off. |
| Privacy | The plugin talks only to the instance it's connected to. Without WooCommerce, it sends nothing about visitors or users; with it, only the checkouts above (never addresses, phone numbers or payment details). |

Disconnect (Settings › ReputeCast) forgets the key and stops the jobs; forms
keep loading. Disconnect the site in ReputeCast too to revoke the key.
Deactivating stops the jobs and keeps the connection; deleting the plugin
removes its options, jobs, post meta and the checkouts not reported yet
(`uninstall.php`); orders keep their consent and click token.

## Test

```sh
cd integrations/wordpress
composer install
composer lint      # parses every file with this PHP, then phpcs (WordPress Coding Standards)
composer test      # PHPUnit + Brain Monkey
composer build     # dist/reputecast-1.3.0.zip, after checking the version numbers agree
```

CI runs them on PHP 8.1 and 8.4. Locally, with Herd:

```sh
php81 vendor/bin/phpunit
php84 vendor/bin/phpunit
```

The tests run without WordPress: Brain Monkey stubs its functions, options,
HTTP and WP-Cron live in memory. Every request the plugin makes, and every
answer the tests mock, is checked against the OpenAPI document the server
generates (`web/openapi.json`, see `tests/Contract.php`). After changing the
API, regenerate it (`go run ./cmd/reputecast openapi > web/openapi.json`) and
run the tests again. The status signature has the same test vector as
`internal/integrations` (`TestStatusSignature`). The WPML and Polylang tests
run in separate processes.

### By hand, against `make dev`

With ReputeCast running locally (`make dev`), a throwaway WordPress:

```sh
wp core download --path=/tmp/rc-wp
wp config create --path=/tmp/rc-wp --dbname=rc_wp --dbuser=root --dbhost=127.0.0.1:3306
wp db create --path=/tmp/rc-wp
wp core install --path=/tmp/rc-wp --url=http://localhost:8399 --title="Acme Blog" \
  --admin_user=admin --admin_email=admin@example.test --skip-email   # prints a generated password
wp rewrite structure '/%postname%/' --path=/tmp/rc-wp
ln -s "$PWD/reputecast" /tmp/rc-wp/wp-content/plugins/reputecast
wp plugin activate reputecast --path=/tmp/rc-wp
PHP_CLI_SERVER_WORKERS=4 php -S localhost:8399 -t /tmp/rc-wp
```

Then connect a WordPress site in ReputeCast with the address `localhost:8399`,
paste the key in Settings › ReputeCast with the address
`http://localhost:8080`, add the block to a page, and publish a post. "Check
now" reaches a site on your own machine only when the instance allows private
networks (`webhooks.allow_private_networks`).

Release 1.3.0 was checked this way on WordPress 7.1.3 with PHP 8.1, with and
without Polylang 3.8: Connect (and its errors), the forms list and the
allowed-domain warning in the block, the language setting, both forms
rendering on a page (and after Disconnect), a published post becoming a draft,
its Spanish translation joining it as an up-to-date language version, and a
signed "Check now".

## Publish

The plugin directory slug is `reputecast`: ReputeCast's daily update check
reads the newest version from
`api.wordpress.org/plugins/info/1.2/?action=plugin_information&request[slug]=reputecast`.

The first time:

1. Sign in to wordpress.org with the HostRepute account (`hostrepute`, the
   readme's Contributors line) and submit `dist/reputecast-1.3.0.zip` at
   <https://wordpress.org/plugins/developers/add/>. The review reads the
   readme's External services section: keep it accurate.
2. Once approved, check out the SVN repository and add the release:

   ```sh
   svn co https://plugins.svn.wordpress.org/reputecast reputecast-svn
   rsync -a --delete integrations/wordpress/reputecast/ reputecast-svn/trunk/
   svn cp reputecast-svn/trunk reputecast-svn/tags/1.3.0
   svn add --force reputecast-svn/trunk reputecast-svn/tags
   svn ci reputecast-svn -m "ReputeCast 1.3.0"
   ```

3. Add the directory's images to `reputecast-svn/assets/` (not to trunk):
   `icon-128x128.png`, `icon-256x256.png`, `banner-772x250.png`,
   `banner-1544x500.png`, and screenshots (`screenshot-1.png`, …) matching a
   `== Screenshots ==` section in readme.txt.

Each release:

1. Bump the version in `reputecast/reputecast.php` (header and
   `REPUTECAST_VERSION`), `readme.txt` (Stable tag, Changelog, Upgrade Notice),
   `blocks/signup-form/block.json` and `index.asset.php`, and
   `KnownPluginVersion` in `internal/integrations/wordpress.go` (the version
   "Update available" compares with when the update check is off).
   `composer test` fails until they agree.
2. Regenerate `languages/reputecast.pot`:
   `wp i18n make-pot reputecast reputecast/languages/reputecast.pot --domain=reputecast`.
3. Update "Tested up to" after testing on the newest WordPress.
4. `composer lint && composer test && composer build`, then commit trunk and a
   new tag to SVN as above.
